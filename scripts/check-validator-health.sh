#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "Usage: $0 <rpc-url> <expected-chain-id> [minimum-peers]" >&2
}

if [[ "$#" -lt 2 || "$#" -gt 3 ]]; then
	usage
	exit 2
fi

rpc_url="${1%/}"
expected_chain_id="$2"
minimum_peers="${3:-1}"

if [[ ! "$minimum_peers" =~ ^[0-9]+$ ]]; then
	echo "minimum-peers must be a non-negative integer" >&2
	exit 2
fi

for dependency in curl jq; do
	if ! command -v "$dependency" >/dev/null 2>&1; then
		echo "required command not found: ${dependency}" >&2
		exit 2
	fi
done

status_json="$(curl -fsS --max-time 5 "${rpc_url}/status")" || {
	echo "validator status endpoint is unavailable: ${rpc_url}/status" >&2
	exit 1
}
net_info_json="$(curl -fsS --max-time 5 "${rpc_url}/net_info")" || {
	echo "validator peer endpoint is unavailable: ${rpc_url}/net_info" >&2
	exit 1
}

if ! snapshot="$(jq -er '
	.result.node_info.network as $chain |
	.result.sync_info.latest_block_height as $height |
	.result.sync_info.catching_up as $catching_up |
	select(($chain | type) == "string" and ($chain | length) > 0) |
	select(($height | tostring | test("^[0-9]+$"))) |
	select(($catching_up | type) == "boolean") |
	[$chain, ($height | tostring), ($catching_up | tostring)] | @tsv
' <<<"$status_json" 2>/dev/null)"; then
	echo "validator returned malformed status data" >&2
	exit 1
fi

if ! peer_count="$(jq -er '
	.result.n_peers as $peers |
	select(($peers | tostring | test("^[0-9]+$"))) |
	$peers | tostring
' <<<"$net_info_json" 2>/dev/null)"; then
	echo "validator returned malformed peer data" >&2
	exit 1
fi

IFS=$'\t' read -r chain_id height catching_up <<<"$snapshot"
if [[ "$chain_id" != "$expected_chain_id" ]]; then
	echo "wrong chain: expected ${expected_chain_id}, got ${chain_id}" >&2
	exit 1
fi
if [[ "$catching_up" != "false" ]]; then
	echo "validator is still catching up at height ${height}" >&2
	exit 1
fi
if (( peer_count < minimum_peers )); then
	echo "insufficient peers: expected at least ${minimum_peers}, got ${peer_count}" >&2
	exit 1
fi

echo "healthy chain=${chain_id} height=${height} peers=${peer_count}"
