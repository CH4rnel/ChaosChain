#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
test_root="$(mktemp -d /tmp/chaoschain-host-install-test.XXXXXX)"
trap 'rm -rf "$test_root"' EXIT

mkdir -p "${test_root}/release/chaoschain-test-linux-amd64" "${test_root}/bootstrap" "${test_root}/bin"
cat >"${test_root}/release/chaoschain-test-linux-amd64/chaoschaind" <<'DAEMON'
#!/usr/bin/env bash
set -euo pipefail
case "$1" in
  version) printf '%s\n' '{"version":"test","commit":"abc123"}' ;;
  init)
    home=""
    while (($#)); do
      if [[ "$1" == "--home" ]]; then home="$2"; shift 2; else shift; fi
    done
    mkdir -p "${home}/config"
    printf '%s\n' '[p2p]' 'persistent_peers = ""' '[rpc]' 'laddr = "tcp://127.0.0.1:26657"' '[instrumentation]' 'prometheus = false' >"${home}/config/config.toml"
    printf '%s\n' 'minimum-gas-prices = ""' >"${home}/config/app.toml"
    printf '%s\n' '{"chain_id":"chaos-devnet-1"}' >"${home}/config/genesis.json"
    printf '%s\n' 'local-private-validator-key' >"${home}/config/priv_validator_key.json"
    ;;
  genesis) [[ "$2" == "validate" ]] ;;
  *) exit 2 ;;
esac
DAEMON
chmod +x "${test_root}/release/chaoschain-test-linux-amd64/chaoschaind"
printf '%s\n' '{"version":"test","commit":"abc123"}' >"${test_root}/release/chaoschain-test-linux-amd64/build-info.json"
tar -C "${test_root}/release" -czf "${test_root}/chaoschain-test-linux-amd64.tar.gz" chaoschain-test-linux-amd64
(cd "$test_root" && sha256sum chaoschain-test-linux-amd64.tar.gz >SHA256SUMS)
printf '%s\n' '{"chain_id":"chaos-devnet-1"}' >"${test_root}/bootstrap/genesis.json"
jq -n --arg sha "$(sha256sum "${test_root}/bootstrap/genesis.json" | cut -d ' ' -f 1)" '{chain_id:"chaos-devnet-1",minimum_gas_price:"0.025uchaos",genesis_sha256:$sha,persistent_peers:"peerid@seed.example.org:27656"}' >"${test_root}/bootstrap/bootstrap.json"
(cd "${test_root}/bootstrap" && sha256sum bootstrap.json genesis.json >SHA256SUMS)

for command in useradd chown systemctl gh; do
	cat >"${test_root}/bin/${command}" <<'STUB'
#!/usr/bin/env bash
if [[ "${0##*/}" == gh ]]; then
	exit "${GH_VERIFY_STATUS:-0}"
fi
exit 0
STUB
	chmod +x "${test_root}/bin/${command}"
done

export PATH="${test_root}/bin:${PATH}"
export CHAOSCHAIN_TEST_ROOT="$test_root/target"
"${repo_root}/ops/validator/scripts/install-validator.sh" \
	"${test_root}/chaoschain-test-linux-amd64.tar.gz" \
	"${test_root}/SHA256SUMS" \
	"${test_root}/bootstrap" \
	"test-validator"

[[ -x "${test_root}/target/usr/local/bin/chaoschaind" ]]
[[ -f "${test_root}/target/var/lib/chaoschain/config/priv_validator_key.json" ]]
[[ "$(stat -c '%a' "${test_root}/target/var/lib/chaoschain/config/priv_validator_key.json")" == 600 ]]
if grep -Fq 'local-private-validator-key' "${test_root}/target/etc/systemd/system/chaoschaind.service"; then
	echo "validator private key leaked into the systemd unit" >&2
	exit 1
fi
grep -Fq 'User=chaoschain' "${test_root}/target/etc/systemd/system/chaoschaind.service"
grep -Fq 'ProtectSystem=strict' "${test_root}/target/etc/systemd/system/chaoschaind.service"
grep -Fq 'persistent_peers = "peerid@seed.example.org:27656"' "${test_root}/target/var/lib/chaoschain/config/config.toml"
grep -Fq 'minimum-gas-prices = "0.025uchaos"' "${test_root}/target/var/lib/chaoschain/config/app.toml"
grep -Fq 'laddr = "tcp://127.0.0.1:26657"' "${test_root}/target/var/lib/chaoschain/config/config.toml"
grep -Fq 'prometheus = false' "${test_root}/target/var/lib/chaoschain/config/config.toml"

if "${repo_root}/ops/validator/scripts/install-validator.sh" \
	"${test_root}/chaoschain-test-linux-amd64.tar.gz" \
	"${test_root}/SHA256SUMS" \
	"${test_root}/bootstrap" \
	"test-validator" >/dev/null 2>&1; then
	echo "installer accepted an existing validator home" >&2
	exit 1
fi

if GH_VERIFY_STATUS=1 CHAOSCHAIN_TEST_ROOT="${test_root}/untrusted-target" \
	"${repo_root}/ops/validator/scripts/install-validator.sh" \
	"${test_root}/chaoschain-test-linux-amd64.tar.gz" \
	"${test_root}/SHA256SUMS" \
	"${test_root}/bootstrap" \
	"untrusted-validator" >/dev/null 2>&1; then
	echo "installer accepted a release with rejected provenance" >&2
	exit 1
fi
[[ ! -e "${test_root}/untrusted-target/usr/local/bin/chaoschaind" ]]

echo "validator host installation tests passed"
