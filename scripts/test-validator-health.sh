#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_dir="$(mktemp -d /tmp/chaoschain-validator-health.XXXXXX)"
trap 'rm -rf "$test_dir"' EXIT

mkdir -p "${test_dir}/bin"
cat >"${test_dir}/bin/curl" <<'CURL'
#!/usr/bin/env bash
set -euo pipefail
url="${*: -1}"
case "$url" in
  */status) cat "$HEALTH_STATUS_FIXTURE" ;;
  */net_info) cat "$HEALTH_NET_INFO_FIXTURE" ;;
  *) exit 2 ;;
esac
CURL
chmod +x "${test_dir}/bin/curl"
export PATH="${test_dir}/bin:${PATH}"

cat >"${test_dir}/status.json" <<'JSON'
{"result":{"node_info":{"network":"chaoschain-1"},"sync_info":{"latest_block_height":"42","catching_up":false}}}
JSON
cat >"${test_dir}/net_info.json" <<'JSON'
{"result":{"n_peers":"3"}}
JSON
export HEALTH_STATUS_FIXTURE="${test_dir}/status.json"
export HEALTH_NET_INFO_FIXTURE="${test_dir}/net_info.json"

output="$("${repo_root}/scripts/check-validator-health.sh" http://validator.example:26657 chaoschain-1 3)"
grep -q 'healthy' <<<"$output"
grep -q 'height=42' <<<"$output"
grep -q 'peers=3' <<<"$output"

if "${repo_root}/scripts/check-validator-health.sh" http://validator.example:26657 wrong-chain 3 >/dev/null 2>&1; then
	 echo "wrong chain ID was accepted" >&2
	 exit 1
fi

cat >"${test_dir}/status.json" <<'JSON'
{"result":{"node_info":{"network":"chaoschain-1"},"sync_info":{"latest_block_height":"42","catching_up":true}}}
JSON
if "${repo_root}/scripts/check-validator-health.sh" http://validator.example:26657 chaoschain-1 3 >/dev/null 2>&1; then
	 echo "catching-up validator was accepted" >&2
	 exit 1
fi

cat >"${test_dir}/status.json" <<'JSON'
not-json
JSON
if "${repo_root}/scripts/check-validator-health.sh" http://validator.example:26657 chaoschain-1 3 >/dev/null 2>&1; then
	 echo "malformed RPC response was accepted" >&2
	 exit 1
fi

echo "validator health probe tests passed"
