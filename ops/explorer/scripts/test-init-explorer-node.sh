#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
test_root="$(mktemp -d /tmp/chaoschain-explorer-init-test.XXXXXX)"
trap 'rm -rf "$test_root"' EXIT

bundle="${test_root}/bundle"
binary="${test_root}/chaoschaind"
home="${test_root}/explorer-node"
mkdir -p "$bundle"
printf '%s\n' '{"chain_id":"chaos-devnet-1"}' >"${bundle}/genesis.json"
genesis_sha="$(sha256sum "${bundle}/genesis.json" | cut -d ' ' -f 1)"
jq -n --arg sha "$genesis_sha" '{
  chain_id: "chaos-devnet-1",
  minimum_gas_price: "0.025uchaos",
  genesis_sha256: $sha,
  persistent_peers: (("a" * 40) + "@seed.example.org:27656," + ("b" * 40) + "@198.51.100.10:27656")
}' >"${bundle}/bootstrap.json"
(cd "$bundle" && sha256sum bootstrap.json genesis.json >SHA256SUMS)

cat >"$binary" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$1" == genesis && "$2" == validate ]]; then
	exit 0
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
pruning = "default"
min-retain-blocks = 10
minimum-gas-prices = ""
[api]
enable = false
swagger = true
address = "tcp://localhost:1317"
max-open-connections = 1000
enabled-unsafe-cors = true
[grpc]
enable = false
CONFIG
cat >"${home}/config/config.toml" <<'CONFIG'
proxy_app = "tcp://127.0.0.1:26658"
[rpc]
laddr = "tcp://127.0.0.1:26657"
pprof_laddr = "localhost:6060"
[p2p]
laddr = "tcp://0.0.0.0:26656"
persistent_peers = ""
[tx_index]
indexer = "kv"
CONFIG
printf '%s\n' 'local generated signer' >"${home}/config/priv_validator_key.json"
EOF
chmod +x "$binary"

CHAOSCHAIN_BINARY="$binary" "$repo_root/ops/explorer/scripts/init-explorer-node.sh" \
	"$bundle" "$home" explorer-node 27656 27657 1417 >/dev/null

[[ "$(jq -er '.chain_id' "${home}/config/genesis.json")" == chaos-devnet-1 ]]
[[ "$(<"${home}/config/init.args")" == $'chain_id=chaos-devnet-1\nmoniker=explorer-node' ]]
grep -Eq '^persistent_peers = "[a-f0-9]{40}@seed\.example\.org:27656,[a-f0-9]{40}@198\.51\.100\.10:27656"$' "${home}/config/config.toml"
grep -Eq '^laddr = "tcp://127\.0\.0\.1:27656"$' "${home}/config/config.toml"
grep -Eq '^laddr = "tcp://127\.0\.0\.1:27657"$' "${home}/config/config.toml"
grep -Eq '^enable = true$' "${home}/config/app.toml"
grep -Eq '^address = "tcp://127\.0\.0\.1:1417"$' "${home}/config/app.toml"
grep -Eq '^pruning = "nothing"$' "${home}/config/app.toml"
grep -Eq '^minimum-gas-prices = "0.025uchaos"$' "${home}/config/app.toml"
grep -Eq '^indexer = "kv"$' "${home}/config/config.toml"
[[ "$(<"${home}/config/priv_validator_key.json")" == 'local generated signer' ]]
[[ ! -e "${home}/config/priv_validator_state.json" ]]

occupied="$(sha256sum "${home}/config/genesis.json")"
if CHAOSCHAIN_BINARY="$binary" "$repo_root/ops/explorer/scripts/init-explorer-node.sh" "$bundle" "$home" replacement >/dev/null 2>&1; then
	echo "Explorer initializer overwrote an occupied home" >&2
	exit 1
fi
[[ "$(sha256sum "${home}/config/genesis.json")" == "$occupied" ]]

bad_bundle="${test_root}/bad-bundle"
cp -a "$bundle" "$bad_bundle"
printf '%s\n' 'tampered' >>"${bad_bundle}/genesis.json"
if CHAOSCHAIN_BINARY="$binary" "$repo_root/ops/explorer/scripts/init-explorer-node.sh" "$bad_bundle" "${test_root}/tampered-node" bad-node >/dev/null 2>&1; then
	echo "Explorer initializer accepted a broken bootstrap checksum" >&2
	exit 1
fi
[[ ! -e "${test_root}/tampered-node" ]]

wrong_chain="${test_root}/wrong-chain"
cp -a "$bundle" "$wrong_chain"
jq '.chain_id = "wrong-chain-1"' "${wrong_chain}/bootstrap.json" >"${wrong_chain}/bootstrap.tmp"
mv "${wrong_chain}/bootstrap.tmp" "${wrong_chain}/bootstrap.json"
(cd "$wrong_chain" && sha256sum bootstrap.json genesis.json >SHA256SUMS)
if CHAOSCHAIN_BINARY="$binary" "$repo_root/ops/explorer/scripts/init-explorer-node.sh" "$wrong_chain" "${test_root}/wrong-node" wrong-node >/dev/null 2>&1; then
	echo "Explorer initializer accepted a chain identity mismatch" >&2
	exit 1
fi
[[ ! -e "${test_root}/wrong-node" ]]

bad_peers="${test_root}/bad-peers"
cp -a "$bundle" "$bad_peers"
jq '.persistent_peers = "not-a-peer"' "${bad_peers}/bootstrap.json" >"${bad_peers}/bootstrap.tmp"
mv "${bad_peers}/bootstrap.tmp" "${bad_peers}/bootstrap.json"
(cd "$bad_peers" && sha256sum bootstrap.json genesis.json >SHA256SUMS)
if CHAOSCHAIN_BINARY="$binary" "$repo_root/ops/explorer/scripts/init-explorer-node.sh" "$bad_peers" "${test_root}/bad-peer-node" bad-peer-node >/dev/null 2>&1; then
	echo "Explorer initializer accepted a malformed peer list" >&2
	exit 1
fi
[[ ! -e "${test_root}/bad-peer-node" ]]

echo "Explorer node bootstrap tests passed"
