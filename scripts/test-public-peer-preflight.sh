#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_root="$(mktemp -d /tmp/chaoschain-peer-preflight.XXXXXX)"
trap 'rm -rf "$test_root"' EXIT

mkdir -p "${test_root}/bootstrap" "${test_root}/bin"
printf '%s\n' '{"chain_id":"chaos-devnet-1"}' >"${test_root}/bootstrap/genesis.json"
trusted_genesis_sha="$(sha256sum "${test_root}/bootstrap/genesis.json" | cut -d ' ' -f 1)"
jq -n --arg sha "$trusted_genesis_sha" '{chain_id:"chaos-devnet-1",minimum_gas_price:"0.025uchaos",genesis_sha256:$sha,persistent_peers:("a" * 40 + "@alpha.example.test:27656," + "b" * 40 + "@beta.example.test:27666," + "c" * 40 + "@sleeping.example.test:27676")}' >"${test_root}/bootstrap/bootstrap.json"
(cd "${test_root}/bootstrap" && sha256sum bootstrap.json genesis.json >SHA256SUMS)

cat >"${test_root}/bin/getent" <<'GETENT'
#!/usr/bin/env bash
set -euo pipefail
[[ "$1" == ahostsv4 ]]
case "$2" in
  alpha.example.test|beta.example.test|sleeping.example.test) printf '192.0.2.1 STREAM %s\n' "$2" ;;
  *) exit 2 ;;
esac
GETENT
cat >"${test_root}/bin/timeout" <<'TIMEOUT'
#!/usr/bin/env bash
set -euo pipefail
[[ "$1" == 3 && "$2" == bash && "$3" == -c ]]
[[ "$4" == *"/dev/tcp/"* ]]
[[ "$4" != *"/dev/tcp/${UNREACHABLE_PEER:-}/"* ]]
TIMEOUT
chmod +x "${test_root}/bin/getent" "${test_root}/bin/timeout"
export PATH="${test_root}/bin:${PATH}"
export UNREACHABLE_PEER=sleeping.example.test

"${repo_root}/scripts/check-public-peers.sh" "${test_root}/bootstrap" 2 "$trusted_genesis_sha" >"${test_root}/success.log"
grep -Fq '2 of 3 advertised peers reachable' "${test_root}/success.log"
grep -Fq 'unreachable sleeping.example.test:27676' "${test_root}/success.log"
if "${repo_root}/scripts/check-public-peers.sh" "${test_root}/bootstrap" 3 "$trusted_genesis_sha" >/dev/null 2>&1; then
	echo "peer preflight passed below the required reachable peer count" >&2
	exit 1
fi

if output="$("${repo_root}/scripts/check-public-peers.sh" "${test_root}/bootstrap" 1 "$(printf '%064d' 0)" 2>&1)"; then
	echo "peer preflight accepted the wrong trusted genesis fingerprint" >&2
	exit 1
fi
grep -Fq 'genesis fingerprint does not match the trusted value' <<<"$output"

printf '%s\n' 'tampered' >>"${test_root}/bootstrap/genesis.json"
if "${repo_root}/scripts/check-public-peers.sh" "${test_root}/bootstrap" 1 "$trusted_genesis_sha" >/dev/null 2>&1; then
	echo "peer preflight accepted a bootstrap checksum mismatch" >&2
	exit 1
fi

echo "public peer preflight tests passed"
