#!/usr/bin/env bash
set -euo pipefail

if (( $# < 5 )); then
	echo "Usage: $0 <chain-id> <checkpoint-height> <checkpoint-hash> <rpc-url> <rpc-url> [rpc-url ...]" >&2
	exit 2
fi

expected_chain="$1"
checkpoint_height="$2"
checkpoint_hash="${3^^}"
shift 3

if [[ ! "$expected_chain" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]]; then
	echo "chain ID must be a nonempty identifier" >&2
	exit 2
fi
if [[ ! "$checkpoint_height" =~ ^[1-9][0-9]{0,17}$ ]]; then
	echo "checkpoint height must be a positive decimal integer" >&2
	exit 2
fi
if [[ ! "$checkpoint_hash" =~ ^[0-9A-F]{64}$ ]]; then
	echo "checkpoint hash must be a 64-character hexadecimal value" >&2
	exit 2
fi
for dependency in curl jq; do
	if ! command -v "$dependency" >/dev/null 2>&1; then
		echo "required command not found: ${dependency}" >&2
		exit 2
	fi
done

lowest_height=""
declare -A validator_addresses=()
for rpc_input in "$@"; do
	if [[ ! "$rpc_input" =~ ^https?://[^/[:space:]]+/?$ ]]; then
		echo "invalid RPC URL: ${rpc_input}" >&2
		exit 2
	fi
	rpc_url="${rpc_input%/}"
	status_json="$(curl -fsS --max-time 5 -- "${rpc_url}/status")" || {
		echo "status endpoint is unavailable: ${rpc_url}" >&2
		exit 1
	}
	if ! snapshot="$(jq -er '
		.result.node_info.network as $chain |
		.result.sync_info.latest_block_height as $height |
		.result.sync_info.catching_up as $catching_up |
		.result.validator_info.address as $address |
		select(($chain | type) == "string" and ($chain | length) > 0) |
		select(($height | tostring | test("^(0|[1-9][0-9]{0,17})$"))) |
		select(($catching_up | type) == "boolean") |
		select(($address | type) == "string" and ($address | test("^[0-9a-fA-F]{40}$"))) |
		[$chain, ($height | tostring), ($catching_up | tostring), $address] | @tsv
	' <<<"$status_json" 2>/dev/null)"; then
		echo "malformed status response: ${rpc_url}" >&2
		exit 1
	fi
	IFS=$'\t' read -r chain_id height catching_up validator_address <<<"$snapshot"
	validator_address="${validator_address^^}"
	if [[ "$validator_address" == "0000000000000000000000000000000000000000" || -n "${validator_addresses[$validator_address]:-}" ]]; then
		echo "duplicate or inactive validator identity at ${rpc_url}" >&2
		exit 1
	fi
	validator_addresses[$validator_address]=1
	if [[ "$chain_id" != "$expected_chain" ]]; then
		echo "wrong chain at ${rpc_url}: expected ${expected_chain}, got ${chain_id}" >&2
		exit 1
	fi
	if [[ "$catching_up" != false ]]; then
		echo "node is still catching up at ${rpc_url}" >&2
		exit 1
	fi
	if (( height <= checkpoint_height )); then
		echo "height has not advanced past checkpoint at ${rpc_url}: ${height}" >&2
		exit 1
	fi

	peers_json="$(curl -fsS --max-time 5 -- "${rpc_url}/net_info")" || {
		echo "peer endpoint is unavailable: ${rpc_url}" >&2
		exit 1
	}
	if ! peer_count="$(jq -er '
		.result.n_peers | tostring |
		select(test("^(0|[1-9][0-9]{0,17})$"))
	' <<<"$peers_json" 2>/dev/null)"; then
		echo "malformed peer response: ${rpc_url}" >&2
		exit 1
	fi
	if (( peer_count < 1 )); then
		echo "node has no peers: ${rpc_url}" >&2
		exit 1
	fi

	block_json="$(curl -fsS --max-time 5 -- "${rpc_url}/block?height=${checkpoint_height}")" || {
		echo "checkpoint block is unavailable: ${rpc_url}" >&2
		exit 1
	}
	if ! block_hash="$(jq -er '
		.result.block_id.hash |
		select(type == "string" and test("^[0-9a-fA-F]{64}$"))
	' <<<"$block_json" 2>/dev/null)"; then
		echo "malformed checkpoint block: ${rpc_url}" >&2
		exit 1
	fi
	if [[ "${block_hash^^}" != "$checkpoint_hash" ]]; then
		echo "checkpoint hash mismatch at ${rpc_url}" >&2
		exit 1
	fi
	if [[ -z "$lowest_height" ]] || (( height < lowest_height )); then
		lowest_height="$height"
	fi
done

echo "network recovered chain=${expected_chain} checkpoint=${checkpoint_height} minimum_height=${lowest_height} nodes=$#"
