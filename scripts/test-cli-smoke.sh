#!/usr/bin/env bash
set -euo pipefail

readonly chain_id="chaos-cli-smoke"
smoke_dir="$(mktemp -d /tmp/chaoschain-cli-smoke.XXXXXX)"
readonly node_home="${smoke_dir}/node"
readonly binary="${smoke_dir}/chaoschaind"
readonly rpc="tcp://127.0.0.1:26657"
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
"$binary" init validator --chain-id "$chain_id" --home "$node_home" >/dev/null 2>&1
"$binary" keys add validator --keyring-backend test --home "$node_home" >/dev/null 2>&1
"$binary" keys add alice --keyring-backend test --home "$node_home" >/dev/null 2>&1

validator_address="$($binary keys show validator -a --keyring-backend test --home "$node_home" 2>/dev/null)"
alice_address="$($binary keys show alice -a --keyring-backend test --home "$node_home" 2>/dev/null)"
validator_operator="$($binary keys show validator -a --bech val --keyring-backend test --home "$node_home" 2>/dev/null)"

"$binary" genesis add-genesis-account validator 1000000000000stake --keyring-backend test --home "$node_home" >/dev/null 2>&1
"$binary" genesis add-genesis-account alice 1000000000stake --keyring-backend test --home "$node_home" >/dev/null 2>&1
"$binary" genesis gentx validator 100000000stake --chain-id "$chain_id" --keyring-backend test --home "$node_home" >/dev/null 2>&1
"$binary" genesis collect-gentxs --home "$node_home" >/dev/null 2>&1

"$binary" start --home "$node_home" --minimum-gas-prices 0stake >"${smoke_dir}/node.log" 2>&1 &
node_pid=$!
wait_for_rpc

initial_response="$($binary query bank balances "$alice_address" --node "$rpc" --home "$node_home" --output json)"
initial_balance="$(jq -er '.balances[] | select(.denom == "stake") | .amount' <<<"$initial_response")"
send_response="$($binary tx bank send validator "$alice_address" 1000stake --chain-id "$chain_id" --keyring-backend test --gas 200000 --fees 100stake --yes --node "$rpc" --home "$node_home" --output json 2>/dev/null)"
wait_for_tx "$(jq -r '.txhash' <<<"$send_response")"
updated_balance="$($binary query bank balances "$alice_address" --node "$rpc" --home "$node_home" --output json 2>/dev/null | jq -r '.balances[] | select(.denom == "stake") | .amount')"
[[ "$updated_balance" -eq $((initial_balance + 1000)) ]]

delegate_response="$($binary tx staking delegate "$validator_operator" 100stake --from alice --chain-id "$chain_id" --keyring-backend test --gas 200000 --fees 100stake --yes --node "$rpc" --home "$node_home" --output json 2>/dev/null)"
wait_for_tx "$(jq -r '.txhash' <<<"$delegate_response")"
delegation_response="$($binary query staking delegation "$alice_address" "$validator_operator" --node "$rpc" --home "$node_home" --output json 2>&1 | sed '/^WARNING:/d')"
delegated="$(jq -er '.delegation_response.balance.amount' <<<"$delegation_response")"
[[ "$delegated" == "100" ]]

"$binary" query distribution rewards "$alice_address" --node "$rpc" --home "$node_home" --output json >/dev/null 2>&1
"$binary" query gov proposals --node "$rpc" --home "$node_home" --output json >/dev/null 2>&1

echo "local L1 CLI journey passed for validator ${validator_address} and account ${alice_address}"
