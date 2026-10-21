#!/usr/bin/env bash
set -euo pipefail

checker="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/check-network-recovery.sh"
readonly checker
readonly hash="AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
readonly split_hash="BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
readonly node_one="http://node1:26657"
readonly node_two="http://node2:26657"
export hash split_hash

curl() {
	local url="${*: -1}" chain="chaos-devnet-1" height="11" catching_up=false peers=2 block_hash="$hash" address="1111111111111111111111111111111111111111"
	if [[ "$url" == *node2* ]]; then
		address="2222222222222222222222222222222222222222"
		case "${FIXTURE_CASE:-healthy}" in
			wrong_chain) chain="another-chain" ;;
			stale) height="10" ;;
			catching_up) catching_up=true ;;
			zero_peers) peers=0 ;;
			split_hash) block_hash="$split_hash" ;;
			duplicate_identity) address="1111111111111111111111111111111111111111" ;;
			unreachable) return 7 ;;
			malformed) printf '%s\n' '{"result":{}}'; return 0 ;;
		esac
	fi
	case "$url" in
		*/status)
			printf '{"result":{"node_info":{"network":"%s"},"sync_info":{"latest_block_height":"%s","catching_up":%s},"validator_info":{"address":"%s"}}}\n' "$chain" "$height" "$catching_up" "$address"
			;;
		*/net_info)
			printf '{"result":{"n_peers":"%s"}}\n' "$peers"
			;;
		*/block\?height=10)
			printf '{"result":{"block_id":{"hash":"%s"}}}\n' "$block_hash"
			;;
		*) return 7 ;;
	esac
}
export -f curl

result="$("$checker" chaos-devnet-1 10 "$hash" "$node_one" "$node_two")"
[[ "$result" == "network recovered chain=chaos-devnet-1 checkpoint=10 minimum_height=11 nodes=2" ]]

for fixture_case in wrong_chain stale catching_up zero_peers split_hash duplicate_identity unreachable malformed; do
	if FIXTURE_CASE="$fixture_case" "$checker" chaos-devnet-1 10 "$hash" "$node_one" "$node_two" >/dev/null 2>&1; then
		echo "recovery check accepted ${fixture_case}" >&2
		exit 1
	fi
done

if "$checker" chaos-devnet-1 10 invalid "$node_one" "$node_two" >/dev/null 2>&1; then
	echo "recovery check accepted an invalid checkpoint hash" >&2
	exit 1
fi
if "$checker" chaos-devnet-1 10 "$hash" "$node_one" >/dev/null 2>&1; then
	echo "recovery check accepted one RPC endpoint" >&2
	exit 1
fi

echo "network recovery fixture tests passed"
