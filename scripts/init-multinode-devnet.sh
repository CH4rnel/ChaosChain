#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly repo_root
readonly profile="${CHAOSCHAIN_NETWORK_PROFILE:-${repo_root}/config/devnet.json}"
readonly binary="${CHAOSCHAIN_BINARY:-chaoschaind}"
readonly network_root="${1:?usage: scripts/init-multinode-devnet.sh NETWORK_ROOT}"

node_id() {
	local home="$1"
	jq -r '.priv_key.value' "${home}/config/node_key.json" | base64 -d | tail -c 32 | sha256sum | cut -c 1-40
}

if [[ -e "$network_root" ]]; then
	echo "refusing to overwrite existing network root: ${network_root}" >&2
	exit 1
fi

"${repo_root}/scripts/check-network-profile.sh" "$profile" >/dev/null

chain_id="$(jq -er '.chain_id' "$profile")"
base_denom="$(jq -er '.token.base_denom' "$profile")"
validator_count="$(jq -er '.multinode.validator_count' "$profile")"
validator_balance="$(jq -er '.multinode.validator_balance' "$profile")"
self_delegation="$(jq -er '.multinode.self_delegation' "$profile")"
port_stride="$(jq -er '.multinode.port_stride' "$profile")"
p2p_base="$(jq -er '.multinode.ports.p2p' "$profile")"
rpc_base="$(jq -er '.multinode.ports.rpc' "$profile")"
grpc_base="$(jq -er '.multinode.ports.grpc' "$profile")"
api_base="$(jq -er '.multinode.ports.api' "$profile")"
pprof_base="$(jq -er '.multinode.ports.pprof' "$profile")"
metrics_base="$(jq -er '.multinode.ports.metrics' "$profile")"
minimum_gas_price="$(jq -er '.minimum_gas_price' "$profile")"
mkdir -p "$network_root"
first_home="${network_root}/node1"
CHAOSCHAIN_BINARY="$binary" CHAOSCHAIN_NETWORK_PROFILE="$profile" "${repo_root}/scripts/init-local-devnet.sh" "$first_home" >/dev/null

node_ids=()
node_ids+=("$(node_id "$first_home")")

for index in $(seq 2 "$validator_count"); do
	home="${network_root}/node${index}"
	name="validator-${index}"
	"$binary" init "chaos-validator-${index}" --chain-id "$chain_id" --home "$home" >/dev/null 2>&1
	"$binary" keys add "$name" --keyring-backend test --home "$home" >/dev/null 2>&1
	address="$($binary keys show "$name" -a --keyring-backend test --home "$home" 2>/dev/null)"
	"$binary" genesis add-genesis-account "$address" "${validator_balance}${base_denom}" --home "$first_home" >/dev/null 2>&1
	node_ids+=("$(node_id "$home")")
done

genesis="${first_home}/config/genesis.json"
empty_gentxs="$(mktemp "${first_home}/config/genesis.json.XXXXXX")"
jq '.app_state.genutil.gen_txs = []' "$genesis" >"$empty_gentxs"
mv "$empty_gentxs" "$genesis"

for index in $(seq 2 "$validator_count"); do
	home="${network_root}/node${index}"
	cp "$genesis" "${home}/config/genesis.json"
	"$binary" genesis gentx "validator-${index}" "${self_delegation}${base_denom}" --chain-id "$chain_id" --keyring-backend test --home "$home" >/dev/null 2>&1
	find "${home}/config/gentx" -type f -name '*.json' -exec cp {} "${first_home}/config/gentx/" \;
done

"$binary" genesis collect-gentxs --home "$first_home" >/dev/null 2>&1
"$binary" genesis validate "$genesis" --home "$first_home" >/dev/null 2>&1

nodes='[]'
for index in $(seq 1 "$validator_count"); do
	offset=$(((index - 1) * port_stride))
	home="${network_root}/node${index}"
	p2p_port=$((p2p_base + offset))
	rpc_port=$((rpc_base + offset))
	grpc_port=$((grpc_base + offset))
	api_port=$((api_base + offset))
	pprof_port=$((pprof_base + offset))
	metrics_port=$((metrics_base + offset))

	if [[ "$index" -gt 1 ]]; then
		cp "$genesis" "${home}/config/genesis.json"
	fi

	peers=""
	for peer_index in $(seq 1 "$validator_count"); do
		if [[ "$peer_index" -eq "$index" ]]; then
			continue
		fi
		peer_offset=$(((peer_index - 1) * port_stride))
		peer="${node_ids[$((peer_index - 1))]}@127.0.0.1:$((p2p_base + peer_offset))"
		peers="${peers:+${peers},}${peer}"
	done

	config="${home}/config/config.toml"
	app_config="${home}/config/app.toml"
	sed -i -E "/^\[rpc\]/,/^\[p2p\]/{s|^laddr = .*|laddr = \"tcp://127.0.0.1:${rpc_port}\"|; s|^pprof_laddr = .*|pprof_laddr = \"localhost:${pprof_port}\"|}" "$config"
	sed -i -E "/^\[p2p\]/,/^\[mempool\]/{s|^laddr = .*|laddr = \"tcp://127.0.0.1:${p2p_port}\"|; s|^persistent_peers = .*|persistent_peers = \"${peers}\"|; s|^allow_duplicate_ip = .*|allow_duplicate_ip = true|}" "$config"
	sed -i -E -e 's|^prometheus = .*|prometheus = true|' -e "s|^prometheus_listen_addr = .*|prometheus_listen_addr = \"127.0.0.1:${metrics_port}\"|" "$config"
	sed -i -E "/^\[api\]/,/^\[grpc\]/{s|^address = .*|address = \"tcp://localhost:${api_port}\"|}" "$app_config"
	sed -i -E "/^\[grpc\]/,/^\[grpc-web\]/{s|^address = .*|address = \"localhost:${grpc_port}\"|}" "$app_config"
	sed -i -E "s|^minimum-gas-prices = .*|minimum-gas-prices = \"${minimum_gas_price}${base_denom}\"|" "$app_config"

	nodes="$(jq -c \
		--argjson index "$index" \
		--arg home "$home" \
		--arg id "${node_ids[$((index - 1))]}" \
		--argjson p2p "$p2p_port" \
		--argjson rpc "$rpc_port" \
		--argjson grpc "$grpc_port" \
		--argjson metrics "$metrics_port" \
		'. + [{index: $index, home: $home, node_id: $id, p2p_port: $p2p, rpc_port: $rpc, grpc_port: $grpc, metrics_port: $metrics}]' <<<"$nodes")"
done

jq -n --arg chain_id "$chain_id" --arg minimum_gas_price "${minimum_gas_price}${base_denom}" --arg genesis_sha256 "$(sha256sum "$genesis" | cut -d ' ' -f 1)" --argjson nodes "$nodes" \
	'{chain_id: $chain_id, minimum_gas_price: $minimum_gas_price, genesis_sha256: $genesis_sha256, nodes: $nodes}' >"${network_root}/network.json"

echo "initialized ${validator_count} validators at ${network_root}"
