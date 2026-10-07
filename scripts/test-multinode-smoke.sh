#!/usr/bin/env bash
set -euo pipefail

readonly profile="config/devnet.json"
smoke_dir="$(mktemp -d /tmp/chaoschain-multinode-smoke.XXXXXX)"
readonly network_root="${smoke_dir}/network"
readonly binary="${smoke_dir}/chaoschaind"
readonly replacement_binary="${smoke_dir}/chaoschaind.next"
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

stop_node() {
	local index="$1"
	local pid="${pids[$index]}"
	kill -TERM "$pid"
	wait "$pid" 2>/dev/null || true
	if kill -0 "$pid" 2>/dev/null; then
		echo "node ${index} process ${pid} survived shutdown" >&2
		return 1
	fi
	unset 'pids[index]'
}

stop_network() {
	local count="$1"
	local index
	for index in $(seq 1 "$count"); do
		stop_node "$index"
	done
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
original_binary_sha="$(sha256sum "$binary" | cut -d ' ' -f 1)"
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
	metrics_port="$(jq -er --argjson index "$index" '.nodes[] | select(.index == $index) | .metrics_port' "${network_root}/network.json")"
	wait_for_height "$rpc_port" 2
	scripts/check-validator-health.sh "http://127.0.0.1:${rpc_port}" "$chain_id" 3
	metrics="$(curl -fsS --max-time 5 "http://127.0.0.1:${metrics_port}/metrics")"
	for metric in cometbft_p2p_peers cometbft_consensus_validator_last_signed_height; do
		if ! grep -Eq "^${metric}(\\{| )" <<<"$metrics"; then
			echo "node ${index} did not expose ${metric}" >&2
			exit 1
		fi
	done
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
persisted_balance="$($binary query bank balances "$alice_address" --node "tcp://127.0.0.1:${node2_rpc}" --home "${network_root}/node2" --output json 2>/dev/null | jq -er --arg denom "$base_denom" '.balances[] | select(.denom == $denom) | .amount')"

offline_index="$validator_count"
offline_rpc="$(jq -er --argjson index "$offline_index" '.nodes[] | select(.index == $index) | .rpc_port' "${network_root}/network.json")"
stop_node "$offline_index"
survivor_height="$(curl -fsS "http://127.0.0.1:${node1_rpc}/status" | jq -er '.result.sync_info.latest_block_height | tonumber')"
offline_recovery_height=$((survivor_height + 8))
offline_transfer_count=3
for _ in $(seq 1 "$offline_transfer_count"); do
	backlog_send="$($binary tx bank send "$validator_name" "$alice_address" "${transfer}${base_denom}" --chain-id "$chain_id" --keyring-backend test --gas "$gas" --fees "${fee}${base_denom}" --yes --node "tcp://127.0.0.1:${node1_rpc}" --home "${network_root}/node1" --output json 2>/dev/null)"
	wait_for_tx "$node2_rpc" "$(jq -er '.txhash' <<<"$backlog_send")"
done
wait_for_height "$node1_rpc" "$offline_recovery_height"
backlog_balance="$($binary query bank balances "$alice_address" --node "tcp://127.0.0.1:${node2_rpc}" --home "${network_root}/node2" --output json 2>/dev/null | jq -er --arg denom "$base_denom" '.balances[] | select(.denom == $denom) | .amount')"
[[ "$backlog_balance" -eq $((persisted_balance + offline_transfer_count * transfer)) ]]
persisted_balance="$backlog_balance"

start_node "$offline_index"
wait_for_height "$offline_rpc" "$offline_recovery_height"
[[ "$(block_hash "$offline_rpc" "$offline_recovery_height")" == "$(block_hash "$node1_rpc" "$offline_recovery_height")" ]]
scripts/check-validator-health.sh "http://127.0.0.1:${offline_rpc}" "$chain_id" 3
offline_node_balance="$($binary query bank balances "$alice_address" --node "tcp://127.0.0.1:${offline_rpc}" --home "${network_root}/node${offline_index}" --output json 2>/dev/null | jq -er --arg denom "$base_denom" '.balances[] | select(.denom == $denom) | .amount')"
[[ "$offline_node_balance" -eq "$backlog_balance" ]]

cold_height="$(curl -fsS "http://127.0.0.1:${node1_rpc}/status" | jq -er '.result.sync_info.latest_block_height | tonumber')"
validator_address="$(curl -fsS "http://127.0.0.1:${node1_rpc}/status" | jq -er '.result.validator_info.address')"
validator_key_sha="$(sha256sum "${network_root}/node1/config/priv_validator_key.json" | cut -d ' ' -f 1)"
stop_network "$validator_count"
backup_archive="${smoke_dir}/node1-before-upgrade.tar.gz"
scripts/backup-validator-home.sh "${network_root}/node1" "$backup_archive" --confirm-stopped
mv "${network_root}/node1" "${smoke_dir}/node1-original"
scripts/restore-validator-home.sh "$backup_archive" "${network_root}/node1" --confirm-stopped
[[ "$(sha256sum "${network_root}/node1/config/priv_validator_key.json" | cut -d ' ' -f 1)" == "$validator_key_sha" ]]
go build -ldflags=-buildid=chaoschain-upgrade-rehearsal -o "$replacement_binary" ./cmd/chaoschaind
replacement_binary_sha="$(sha256sum "$replacement_binary" | cut -d ' ' -f 1)"
[[ "$replacement_binary_sha" != "$original_binary_sha" ]]
mv "$replacement_binary" "$binary"
for index in $(seq 1 "$validator_count"); do
	start_node "$index"
done
cold_recovery_height=$((cold_height + 2))
wait_for_height "$node1_rpc" "$cold_recovery_height"
[[ "$(curl -fsS "http://127.0.0.1:${node1_rpc}/status" | jq -er '.result.validator_info.address')" == "$validator_address" ]]
cold_recovery_hash="$(block_hash "$node1_rpc" "$cold_recovery_height")"
for index in $(seq 2 "$validator_count"); do
	rpc_port="$(jq -er --argjson index "$index" '.nodes[] | select(.index == $index) | .rpc_port' "${network_root}/network.json")"
	wait_for_height "$rpc_port" "$cold_recovery_height"
	[[ "$(block_hash "$rpc_port" "$cold_recovery_height")" == "$cold_recovery_hash" ]]
done

recovered_balance="$($binary query bank balances "$alice_address" --node "tcp://127.0.0.1:${node2_rpc}" --home "${network_root}/node2" --output json 2>/dev/null | jq -er --arg denom "$base_denom" '.balances[] | select(.denom == $denom) | .amount')"
[[ "$recovered_balance" == "$persisted_balance" ]]
restored_balance="$($binary query bank balances "$alice_address" --node "tcp://127.0.0.1:${node1_rpc}" --home "${network_root}/node1" --output json 2>/dev/null | jq -er --arg denom "$base_denom" '.balances[] | select(.denom == $denom) | .amount')"
[[ "$restored_balance" == "$persisted_balance" ]]
recovered_unbonding="$($binary query staking unbonding-delegation "$alice_address" "$validator_operator" --node "tcp://127.0.0.1:${node3_rpc}" --home "${network_root}/node3" --output json 2>&1 | sed '/^WARNING:/d' | jq -er '[.unbond.entries[].balance | tonumber] | add')"
[[ "$recovered_unbonding" -eq "$unbonding" ]]

echo "four-validator devnet restored a validator home and preserved consensus and application state across binary replacement at height ${cold_recovery_height}"
