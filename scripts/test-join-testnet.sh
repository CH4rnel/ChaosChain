#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_root="$(mktemp -d /tmp/chaoschain-full-node-join-test.XXXXXX)"
trap 'rm -rf -- "$test_root"' EXIT

bundle="${test_root}/bootstrap"
binary="${test_root}/chaoschaind"
home="${test_root}/full-node"
mkdir -p "$bundle"
printf '%s\n' '{"chain_id":"chaos-devnet-1"}' >"${bundle}/genesis.json"
trusted_genesis_sha="$(sha256sum "${bundle}/genesis.json" | cut -d ' ' -f 1)"
jq -n --arg sha "$trusted_genesis_sha" '{
  chain_id: "chaos-devnet-1",
  minimum_gas_price: "0.025uchaos",
  genesis_sha256: $sha,
  persistent_peers: (("a" * 40) + "@seed.example.org:27656," + ("b" * 40) + "@198.51.100.10:27666")
}' >"${bundle}/bootstrap.json"
(cd "$bundle" && sha256sum bootstrap.json genesis.json >SHA256SUMS)

cat >"$binary" <<'DAEMON'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$1" == genesis && "$2" == validate ]]; then
	[[ "${FAIL_GENESIS_VALIDATION:-0}" != 1 ]]
	exit
fi
[[ "$1" == init ]]
shift
moniker="$1"
shift
while (($#)); do
	case "$1" in
		--chain-id) chain_id="$2"; shift 2 ;;
		--home) home="$2"; shift 2 ;;
		*) exit 2 ;;
	esac
done
mkdir -p "${home}/config"
printf 'chain_id=%s\nmoniker=%s\n' "$chain_id" "$moniker" >"${home}/config/init.args"
cat >"${home}/config/app.toml" <<'CONFIG'
minimum-gas-prices = ""
[api]
enable = true
[grpc]
enable = true
CONFIG
cat >"${home}/config/config.toml" <<'CONFIG'
[rpc]
laddr = "tcp://0.0.0.0:26657"
[p2p]
laddr = "tcp://0.0.0.0:26656"
persistent_peers = ""
CONFIG
printf '%s\n' 'local node key' >"${home}/config/node_key.json"
printf '%s\n' 'local validator key' >"${home}/config/priv_validator_key.json"
DAEMON
chmod +x "$binary"

join_script="${repo_root}/scripts/join-testnet.sh"
if [[ ! -x "$join_script" ]]; then
	echo "full-node join script is missing or not executable: ${join_script}" >&2
	exit 1
fi

CHAOSCHAIN_BINARY="$binary" "$join_script" "$bundle" "$home" full-node "$trusted_genesis_sha" >/dev/null
[[ "$(jq -er '.chain_id' "${home}/config/genesis.json")" == chaos-devnet-1 ]]
[[ "$(<"${home}/config/init.args")" == $'chain_id=chaos-devnet-1\nmoniker=full-node' ]]
grep -Eq '^laddr = "tcp://127\.0\.0\.1:26657"$' "${home}/config/config.toml"
grep -Eq '^persistent_peers = "[a-f0-9]{40}@seed\.example\.org:27656,[a-f0-9]{40}@198\.51\.100\.10:27666"$' "${home}/config/config.toml"
grep -Fxq 'enable = false' "${home}/config/app.toml"
awk '/^\[grpc\]/{grpc=1;next} /^\[/{grpc=0} grpc && $0 == "enable = false"{found=1} END{exit !found}' "${home}/config/app.toml"
grep -Fxq 'minimum-gas-prices = "0.025uchaos"' "${home}/config/app.toml"
[[ "$(<"${home}/config/node_key.json")" == 'local node key' ]]
[[ "$(<"${home}/config/priv_validator_key.json")" == 'local validator key' ]]

occupied="$(sha256sum "${home}/config/genesis.json")"
if CHAOSCHAIN_BINARY="$binary" "$join_script" "$bundle" "$home" replacement "$trusted_genesis_sha" >/dev/null 2>&1; then
	echo "full-node join overwrote an existing home" >&2
	exit 1
fi
[[ "$(sha256sum "${home}/config/genesis.json")" == "$occupied" ]]

if CHAOSCHAIN_BINARY="$binary" "$join_script" "$bundle" "${test_root}/missing-fingerprint" missing >/dev/null 2>&1; then
	echo "full-node join accepted a missing trusted fingerprint" >&2
	exit 1
fi
[[ ! -e "${test_root}/missing-fingerprint" ]]

if CHAOSCHAIN_BINARY="$binary" "$join_script" "$bundle" "${test_root}/wrong-fingerprint" wrong "$(printf '%064d' 0)" >/dev/null 2>&1; then
	echo "full-node join accepted the wrong trusted fingerprint" >&2
	exit 1
fi
[[ ! -e "${test_root}/wrong-fingerprint" ]]

bad_bundle="${test_root}/bad-bundle"
cp -a "$bundle" "$bad_bundle"
printf '%s\n' tampered >>"${bad_bundle}/genesis.json"
if CHAOSCHAIN_BINARY="$binary" "$join_script" "$bad_bundle" "${test_root}/tampered-node" tampered "$trusted_genesis_sha" >/dev/null 2>&1; then
	echo "full-node join accepted a corrupted bundle" >&2
	exit 1
fi
[[ ! -e "${test_root}/tampered-node" ]]

bad_peers="${test_root}/bad-peers"
cp -a "$bundle" "$bad_peers"
jq '.persistent_peers = "not-a-peer"' "${bad_peers}/bootstrap.json" >"${bad_peers}/bootstrap.tmp"
mv "${bad_peers}/bootstrap.tmp" "${bad_peers}/bootstrap.json"
(cd "$bad_peers" && sha256sum bootstrap.json genesis.json >SHA256SUMS)
if CHAOSCHAIN_BINARY="$binary" "$join_script" "$bad_peers" "${test_root}/bad-peer-node" bad-peer "$trusted_genesis_sha" >/dev/null 2>&1; then
	echo "full-node join accepted a malformed peer list" >&2
	exit 1
fi
[[ ! -e "${test_root}/bad-peer-node" ]]

if FAIL_GENESIS_VALIDATION=1 CHAOSCHAIN_BINARY="$binary" "$join_script" "$bundle" "${test_root}/invalid-genesis" invalid "$trusted_genesis_sha" >/dev/null 2>&1; then
	echo "full-node join accepted genesis rejected by the daemon" >&2
	exit 1
fi
[[ ! -e "${test_root}/invalid-genesis" ]]

echo "full-node join tests passed"
