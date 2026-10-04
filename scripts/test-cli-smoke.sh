#!/usr/bin/env bash
set -euo pipefail

smoke_dir="$(mktemp -d /tmp/chaoschain-cli-smoke.XXXXXX)"
readonly node_home="${smoke_dir}/node"
readonly binary="${smoke_dir}/chaoschaind"
readonly rpc="tcp://127.0.0.1:26657"
readonly profile="config/devnet.json"
node_pid=""

cleanup() {
	if [[ -n "$node_pid" ]] && kill -0 "$node_pid" 2>/dev/null; then
		kill -TERM "$node_pid"
		wait "$node_pid" || true
	fi
	rm -rf "$smoke_dir"
}
trap cleanup EXIT

wait_for_rpc() {
	local status
	for _ in $(seq 1 60); do
		if status="$(curl -fsS --max-time 2 http://127.0.0.1:26657/status 2>/dev/null)" &&
			[[ "$(jq -r '.result.sync_info.latest_block_height' <<<"$status")" -gt 0 ]]; then
			return 0
		fi
		sleep 1
	done
	echo "CometBFT RPC did not become ready" >&2
	return 1
}

wait_for_tx() {
	local hash="$1"
	local response
	for _ in $(seq 1 30); do
		if response="$($binary query tx "$hash" --node "$rpc" --home "$node_home" --output json 2>/dev/null)"; then
			jq -e '.code == 0' <<<"$response" >/dev/null
			return 0
		fi
		sleep 1
	done
	echo "transaction ${hash} was not committed" >&2
	return 1
}

go build -o "$binary" ./cmd/chaoschaind
CHAOSCHAIN_BINARY="$binary" scripts/init-local-devnet.sh "$node_home" >/dev/null

chain_id="$(jq -er '.chain_id' "$profile")"
base_denom="$(jq -er '.token.base_denom' "$profile")"
validator_name="$(jq -er '.staking.validator_account' "$profile")"
alice_name="$(jq -er --arg validator "$validator_name" '.accounts[] | select(.name != $validator) | .name' "$profile" | head -n 1)"
gas="$(jq -er '.smoke.gas' "$profile")"
fee="$(jq -er '.smoke.fee' "$profile")"
transfer="$(jq -er '.smoke.transfer' "$profile")"
delegation="$(jq -er '.smoke.delegation' "$profile")"

validator_address="$($binary keys show "$validator_name" -a --keyring-backend test --home "$node_home" 2>/dev/null)"
alice_address="$($binary keys show "$alice_name" -a --keyring-backend test --home "$node_home" 2>/dev/null)"
validator_operator="$($binary keys show "$validator_name" -a --bech val --keyring-backend test --home "$node_home" 2>/dev/null)"

jq -e --arg denom "$base_denom" '
  .app_state.staking.params.bond_denom == $denom and
  .app_state.mint.params.mint_denom == $denom and
  all(.app_state.gov.params.min_deposit[]; .denom == $denom) and
  all(.app_state.bank.balances[].coins[]; .denom == $denom)
' "${node_home}/config/genesis.json" >/dev/null

"$binary" start --home "$node_home" >"${smoke_dir}/node.log" 2>&1 &
node_pid=$!
wait_for_rpc

initial_response="$($binary query bank balances "$alice_address" --node "$rpc" --home "$node_home" --output json)"
initial_balance="$(jq -er --arg denom "$base_denom" '.balances[] | select(.denom == $denom) | .amount' <<<"$initial_response")"
send_response="$($binary tx bank send "$validator_name" "$alice_address" "${transfer}${base_denom}" --chain-id "$chain_id" --keyring-backend test --gas "$gas" --fees "${fee}${base_denom}" --yes --node "$rpc" --home "$node_home" --output json 2>/dev/null)"
wait_for_tx "$(jq -r '.txhash' <<<"$send_response")"
updated_balance="$($binary query bank balances "$alice_address" --node "$rpc" --home "$node_home" --output json 2>/dev/null | jq -r --arg denom "$base_denom" '.balances[] | select(.denom == $denom) | .amount')"
[[ "$updated_balance" -eq $((initial_balance + transfer)) ]]

delegate_response="$($binary tx staking delegate "$validator_operator" "${delegation}${base_denom}" --from "$alice_name" --chain-id "$chain_id" --keyring-backend test --gas "$gas" --fees "${fee}${base_denom}" --yes --node "$rpc" --home "$node_home" --output json 2>/dev/null)"
wait_for_tx "$(jq -r '.txhash' <<<"$delegate_response")"
delegation_response="$($binary query staking delegation "$alice_address" "$validator_operator" --node "$rpc" --home "$node_home" --output json 2>&1 | sed '/^WARNING:/d')"
delegated="$(jq -er '.delegation_response.balance.amount' <<<"$delegation_response")"
[[ "$delegated" == "$delegation" ]]

"$binary" query distribution rewards "$alice_address" --node "$rpc" --home "$node_home" --output json >/dev/null 2>&1
"$binary" query gov proposals --node "$rpc" --home "$node_home" --output json >/dev/null 2>&1

echo "local L1 CLI journey passed for validator ${validator_address} and account ${alice_address}"
