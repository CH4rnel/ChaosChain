#!/usr/bin/env bash
set -euo pipefail
umask 077

usage() {
	echo "Usage: $0 <bootstrap-bundle> <new-node-home> <moniker> [p2p-port] [rpc-port] [api-port]" >&2
}

if [[ "$#" -lt 3 || "$#" -gt 6 ]]; then
	usage
	exit 2
fi

bundle_input="$1"
home_input="$2"
moniker="$3"
p2p_port="${4:-26656}"
rpc_port="${5:-26657}"
api_port="${6:-1317}"
binary="${CHAOSCHAIN_BINARY:-chaoschaind}"

if [[ ! -d "$bundle_input" || -L "$bundle_input" ]]; then
	echo "Bootstrap bundle must be a directory: ${bundle_input}" >&2
	exit 2
fi
bundle="$(cd "$bundle_input" && pwd -P)"
for file in bootstrap.json genesis.json SHA256SUMS; do
	if [[ ! -f "${bundle}/${file}" || -L "${bundle}/${file}" ]]; then
		echo "Bootstrap bundle is missing a regular ${file}" >&2
		exit 2
	fi
done
if ! awk '
	!/^[0-9a-f]{64}  (bootstrap\.json|genesis\.json)$/ { exit 1 }
	{ count++; if ($2 == "bootstrap.json") bootstrap++; if ($2 == "genesis.json") genesis++ }
	END { if (count != 2 || bootstrap != 1 || genesis != 1) exit 1 }
' "${bundle}/SHA256SUMS"; then
	echo "Bootstrap checksum manifest must contain exactly bootstrap.json and genesis.json" >&2
	exit 1
fi
if ! (cd "$bundle" && sha256sum --check --status SHA256SUMS); then
	echo "Bootstrap checksum verification failed" >&2
	exit 1
fi

chain_id="$(jq -er '.chain_id | select(type == "string" and length > 0 and test("^[A-Za-z0-9._-]+$"))' "${bundle}/bootstrap.json")"
minimum_gas_price="$(jq -er '.minimum_gas_price | select(type == "string" and test("^[0-9]+([.][0-9]+)?[A-Za-z][A-Za-z0-9/:._-]*$"))' "${bundle}/bootstrap.json")"
expected_genesis_sha="$(jq -er '.genesis_sha256 | select(type == "string" and test("^[0-9a-f]{64}$"))' "${bundle}/bootstrap.json")"
actual_genesis_sha="$(sha256sum "${bundle}/genesis.json" | cut -d ' ' -f 1)"
genesis_chain_id="$(jq -er '.chain_id | select(type == "string" and length > 0)' "${bundle}/genesis.json")"
if [[ "$actual_genesis_sha" != "$expected_genesis_sha" || "$genesis_chain_id" != "$chain_id" ]]; then
	echo "Bootstrap chain identity does not match its verified genesis" >&2
	exit 1
fi

if ! persistent_peers="$(jq -er '
	.persistent_peers |
	select(type == "string" and length > 0) |
	split(",") |
	select(length > 0 and all(.[]; test("^[0-9a-f]{40}@[A-Za-z0-9][A-Za-z0-9.-]*:[0-9]{1,5}$"))) |
	select(all(.[]; (split(":")[-1] | tonumber) >= 1 and (split(":")[-1] | tonumber) <= 65535)) |
	join(",")
' "${bundle}/bootstrap.json" 2>/dev/null)"; then
	echo "Bootstrap peer list is missing or malformed" >&2
	exit 1
fi

for port in "$p2p_port" "$rpc_port" "$api_port"; do
	if [[ ! "$port" =~ ^[0-9]{1,5}$ ]] || ((10#$port < 1 || 10#$port > 65535)); then
		echo "Node ports must be integers from 1 through 65535" >&2
		exit 2
	fi
done
if [[ "$p2p_port" == "$rpc_port" || "$p2p_port" == "$api_port" || "$rpc_port" == "$api_port" ]]; then
	echo "P2P, RPC, and API ports must be distinct" >&2
	exit 2
fi
if [[ ! "$moniker" =~ ^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$ ]]; then
	echo "Moniker must contain 1 to 64 safe characters" >&2
	exit 2
fi
if ! command -v "$binary" >/dev/null 2>&1 && [[ ! -x "$binary" ]]; then
	echo "ChaosChain binary is unavailable: ${binary}" >&2
	exit 2
fi

home_parent_input="$(dirname -- "$home_input")"
home_name="$(basename -- "$home_input")"
if [[ ! -d "$home_parent_input" || "$home_name" == "." || "$home_name" == ".." ]]; then
	echo "Node home parent must exist and home must name a new directory" >&2
	exit 2
fi
home_parent="$(cd "$home_parent_input" && pwd -P)"
node_home="${home_parent}/${home_name}"
if [[ -e "$node_home" || -L "$node_home" ]]; then
	echo "Refusing to overwrite an existing node home: ${node_home}" >&2
	exit 1
fi

staging="$(mktemp -d "${home_parent}/.chaoschain-explorer-node.XXXXXX")"
cleanup() {
	if [[ -d "$staging" ]]; then
		rm -rf -- "$staging"
	fi
}
trap cleanup EXIT

"$binary" init "$moniker" --chain-id "$chain_id" --home "$staging" >/dev/null
cp -- "${bundle}/genesis.json" "${staging}/config/genesis.json"

set_toml() {
	local file="$1"
	local section="$2"
	local key="$3"
	local value="$4"
	local output
	output="$(mktemp "${file}.rewrite.XXXXXX")"
	if ! awk -v wanted_section="$section" -v wanted_key="$key" -v wanted_value="$value" '
		BEGIN { section = ""; changed = 0 }
		/^[[:space:]]*\[[^]]+\][[:space:]]*$/ {
			section = $0
			gsub(/^[[:space:]]*\[|\][[:space:]]*$/, "", section)
		}
		section == wanted_section && !changed && $0 ~ "^[[:space:]]*" wanted_key "[[:space:]]*=" {
			print wanted_key " = " wanted_value
			changed = 1
			next
		}
		{ print }
		END { if (!changed) exit 3 }
	' "$file" >"$output"; then
		rm -f -- "$output"
		echo "Required TOML key is missing: [${section}] ${key} in ${file}" >&2
		exit 1
	fi
	mv -- "$output" "$file"
}

app_config="${staging}/config/app.toml"
comet_config="${staging}/config/config.toml"
set_toml "$app_config" "" pruning '"nothing"'
set_toml "$app_config" "" min-retain-blocks 0
set_toml "$app_config" "" minimum-gas-prices "\"${minimum_gas_price}\""
set_toml "$app_config" api enable true
set_toml "$app_config" api swagger false
set_toml "$app_config" api address "\"tcp://127.0.0.1:${api_port}\""
set_toml "$app_config" api enabled-unsafe-cors false
set_toml "$app_config" api max-open-connections 500
set_toml "$comet_config" rpc laddr "\"tcp://127.0.0.1:${rpc_port}\""
set_toml "$comet_config" p2p laddr "\"tcp://127.0.0.1:${p2p_port}\""
set_toml "$comet_config" p2p persistent_peers "\"${persistent_peers}\""
set_toml "$comet_config" tx_index indexer '"kv"'

"$binary" genesis validate "${staging}/config/genesis.json" --home "$staging" >/dev/null
mv -T -n -- "$staging" "$node_home"
if [[ -d "$staging" ]]; then
	echo "Node home appeared during initialization: ${node_home}" >&2
	exit 1
fi

echo "Explorer full node initialized at ${node_home} for ${chain_id}"
echo "Start it with: chaoschaind start --home ${node_home}"
