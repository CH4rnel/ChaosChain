#!/usr/bin/env bash
set -euo pipefail

readonly profile="config/devnet.json"
smoke_dir="$(mktemp -d /tmp/chaoschain-multinode-smoke.XXXXXX)"
readonly network_root="${smoke_dir}/network"
readonly binary="${smoke_dir}/chaoschaind"
pids=()

cleanup() {
	for pid in "${pids[@]}"; do
		kill -TERM "$pid" 2>/dev/null || true
	done
	for pid in "${pids[@]}"; do
		wait "$pid" 2>/dev/null || true
	done
	rm -rf "$smoke_dir"
}
trap cleanup EXIT

start_node() {
	local index="$1"
	"$binary" start --home "${network_root}/node${index}" >"${smoke_dir}/node${index}.log" 2>&1 &
	pids[index]=$!
}

wait_for_height() {
	local port="$1"
	local target="$2"
	local height
	for _ in $(seq 1 90); do
		height="$(curl -fsS --max-time 2 "http://127.0.0.1:${port}/status" 2>/dev/null | jq -r '.result.sync_info.latest_block_height // 0')" || true
		if [[ "${height:-0}" -ge "$target" ]]; then
			return 0
		fi
		sleep 1
	done
	echo "node on RPC port ${port} did not reach height ${target}" >&2
	return 1
}

wait_for_tx() {
	local port="$1"
	local hash="$2"
	local response
	for _ in $(seq 1 30); do
		if response="$($binary query tx "$hash" --node "tcp://127.0.0.1:${port}" --home "${network_root}/node1" --output json 2>/dev/null)"; then
			jq -e '.code == 0' <<<"$response" >/dev/null
			return 0
		fi
		sleep 1
	done
	echo "transaction ${hash} was not observed through RPC port ${port}" >&2
	return 1
}

block_hash() {
	local port="$1"
	local height="$2"
	curl -fsS "http://127.0.0.1:${port}/block?height=${height}" | jq -er '.result.block_id.hash'
}

go build -o "$binary" ./cmd/chaoschaind
CHAOSCHAIN_BINARY="$binary" scripts/init-multinode-devnet.sh "$network_root" >/dev/null

validator_count="$(jq -er '.multinode.validator_count' "$profile")"
chain_id="$(jq -er '.chain_id' "$profile")"
base_denom="$(jq -er '.token.base_denom' "$profile")"
validator_name="$(jq -er '.staking.validator_account' "$profile")"
alice_name="$(jq -er --arg validator "$validator_name" '.accounts[] | select(.name != $validator) | .name' "$profile" | head -n 1)"
gas="$(jq -er '.smoke.gas' "$profile")"
fee="$(jq -er '.smoke.fee' "$profile")"
transfer="$(jq -er '.smoke.transfer' "$profile")"
delegation="$(jq -er '.smoke.delegation' "$profile")"
expected_genesis="$(jq -er '.genesis_sha256' "${network_root}/network.json")"

for index in $(seq 1 "$validator_count"); do
	actual_genesis="$(sha256sum "${network_root}/node${index}/config/genesis.json" | cut -d ' ' -f 1)"
	[[ "$actual_genesis" == "$expected_genesis" ]]
	start_node "$index"
done

for index in $(seq 1 "$validator_count"); do
	rpc_port="$(jq -er --argjson index "$index" '.nodes[] | select(.index == $index) | .rpc_port' "${network_root}/network.json")"
	wait_for_height "$rpc_port" 2
	peers="$(curl -fsS "http://127.0.0.1:${rpc_port}/net_info" | jq -er '.result.n_peers | tonumber')"
	[[ "$peers" -ge 3 ]]
done

reference_hash="$(block_hash "$(jq -er '.nodes[0].rpc_port' "${network_root}/network.json")" 2)"
for index in $(seq 2 "$validator_count"); do
	rpc_port="$(jq -er --argjson index "$index" '.nodes[] | select(.index == $index) | .rpc_port' "${network_root}/network.json")"
	[[ "$(block_hash "$rpc_port" 2)" == "$reference_hash" ]]
done

node1_rpc="$(jq -er '.nodes[0].rpc_port' "${network_root}/network.json")"
node2_rpc="$(jq -er '.nodes[1].rpc_port' "${network_root}/network.json")"
node3_rpc="$(jq -er '.nodes[2].rpc_port' "${network_root}/network.json")"
alice_address="$($binary keys show "$alice_name" -a --keyring-backend test --home "${network_root}/node1" 2>/dev/null)"
validator_operator="$($binary keys show "$validator_name" -a --bech val --keyring-backend test --home "${network_root}/node1" 2>/dev/null)"
initial_balance="$($binary query bank balances "$alice_address" --node "tcp://127.0.0.1:${node2_rpc}" --home "${network_root}/node2" --output json 2>/dev/null | jq -er --arg denom "$base_denom" '.balances[] | select(.denom == $denom) | .amount')"
send_response="$($binary tx bank send "$validator_name" "$alice_address" "${transfer}${base_denom}" --chain-id "$chain_id" --keyring-backend test --gas "$gas" --fees "${fee}${base_denom}" --yes --node "tcp://127.0.0.1:${node1_rpc}" --home "${network_root}/node1" --output json 2>/dev/null)"
wait_for_tx "$node2_rpc" "$(jq -er '.txhash' <<<"$send_response")"
updated_balance="$($binary query bank balances "$alice_address" --node "tcp://127.0.0.1:${node2_rpc}" --home "${network_root}/node2" --output json 2>/dev/null | jq -er --arg denom "$base_denom" '.balances[] | select(.denom == $denom) | .amount')"
[[ "$updated_balance" -eq $((initial_balance + transfer)) ]]

delegate_response="$($binary tx staking delegate "$validator_operator" "${delegation}${base_denom}" --from "$alice_name" --chain-id "$chain_id" --keyring-backend test --gas "$gas" --fees "${fee}${base_denom}" --yes --node "tcp://127.0.0.1:${node1_rpc}" --home "${network_root}/node1" --output json 2>/dev/null)"
wait_for_tx "$node2_rpc" "$(jq -er '.txhash' <<<"$delegate_response")"
delegated="$($binary query staking delegation "$alice_address" "$validator_operator" --node "tcp://127.0.0.1:${node2_rpc}" --home "${network_root}/node2" --output json 2>&1 | sed '/^WARNING:/d' | jq -er '.delegation_response.balance.amount')"
[[ "$delegated" == "$delegation" ]]

unbond_response="$($binary tx staking unbond "$validator_operator" "${delegation}${base_denom}" --from "$alice_name" --chain-id "$chain_id" --keyring-backend test --gas "$gas" --fees "${fee}${base_denom}" --yes --node "tcp://127.0.0.1:${node1_rpc}" --home "${network_root}/node1" --output json 2>/dev/null)"
wait_for_tx "$node3_rpc" "$(jq -er '.txhash' <<<"$unbond_response")"
unbonding="$($binary query staking unbonding-delegation "$alice_address" "$validator_operator" --node "tcp://127.0.0.1:${node3_rpc}" --home "${network_root}/node3" --output json 2>&1 | sed '/^WARNING:/d' | jq -er '[.unbond.entries[].balance | tonumber] | add')"
[[ "$unbonding" -eq "$delegation" ]]

offline_index="$validator_count"
offline_rpc="$(jq -er --argjson index "$offline_index" '.nodes[] | select(.index == $index) | .rpc_port' "${network_root}/network.json")"
kill -TERM "${pids[$offline_index]}"
wait "${pids[$offline_index]}" 2>/dev/null || true
survivor_height="$(curl -fsS "http://127.0.0.1:${node1_rpc}/status" | jq -er '.result.sync_info.latest_block_height | tonumber')"
recovery_height=$((survivor_height + 2))
wait_for_height "$node1_rpc" "$recovery_height"

start_node "$offline_index"
wait_for_height "$offline_rpc" "$recovery_height"
[[ "$(block_hash "$offline_rpc" "$recovery_height")" == "$(block_hash "$node1_rpc" "$recovery_height")" ]]

echo "four-validator devnet preserved consensus transfer state and node recovery at height ${recovery_height}"
