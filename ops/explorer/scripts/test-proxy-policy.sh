#!/usr/bin/env bash
set -euo pipefail

image="${CHAOSCHAIN_EXPLORER_IMAGE:-chaoschain-ping:test}"
container="chaoschain-explorer-policy-test-${RANDOM}"
test_root="$(mktemp -d /tmp/chaoschain-explorer-proxy-test.XXXXXX)"
upstream_port=$((38000 + RANDOM % 1000))
explorer_port=$((39000 + RANDOM % 1000))
upstream_pid=""

cleanup() {
	docker rm -f "$container" >/dev/null 2>&1 || true
	if [[ -n "$upstream_pid" ]]; then
		kill "$upstream_pid" >/dev/null 2>&1 || true
	fi
	rm -rf -- "$test_root"
}
trap cleanup EXIT

if ! docker image inspect "$image" >/dev/null 2>&1; then
	echo "Build the explorer image before running this test: ${image}" >&2
	exit 2
fi

printf '%s\n' 'upstream' >"${test_root}/index.html"
python3 -m http.server "$upstream_port" --bind 127.0.0.1 --directory "$test_root" >"${test_root}/upstream.log" 2>&1 &
upstream_pid=$!

docker run -d --name "$container" --network host \
	-e NODE_HOST=127.0.0.1 \
	-e NODE_API_PORT="$upstream_port" \
	-e NODE_RPC_PORT="$upstream_port" \
	-e LISTEN_HOST=127.0.0.1 \
	-e LISTEN_PORT="$explorer_port" \
	-e PUBLIC_ORIGIN=https://testnet-explorer.example.org \
	"$image" >/dev/null

base="http://127.0.0.1:${explorer_port}"
for _ in $(seq 1 30); do
	if curl -fsS "${base}/" >/dev/null 2>&1; then
		break
	fi
	sleep 0.2
done
curl -fsS "${base}/" >/dev/null

api_headers="$(curl -sS -D - -o /dev/null -H 'Origin: https://testnet-explorer.example.org' "${base}/api/")"
grep -qi '^Access-Control-Allow-Origin: https://testnet-explorer.example.org' <<<"$api_headers"
[[ "$(curl -sS -o /dev/null -w '%{http_code}' -X POST "${base}/api/cosmos/tx/v1beta1/txs")" == 403 ]]
[[ "$(curl -sS -o /dev/null -w '%{http_code}' "${base}/rpc/net_info")" == 403 ]]
[[ "$(curl -sS -o /dev/null -w '%{http_code}' -X POST "${base}/rpc/status")" == 403 ]]
[[ "$(curl -sS -o /dev/null -w '%{http_code}' "${base}/rpc/status")" != 403 ]]

echo "Explorer proxy policy tests passed"
