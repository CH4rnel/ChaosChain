#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 3 ]]; then
	echo "Usage: $0 <network-root> <output-directory> <advertised-host>" >&2
	exit 2
fi

network_input="$1"
output_input="$2"
advertised_host="$3"
if [[ ! "$advertised_host" =~ ^[A-Za-z0-9][A-Za-z0-9.-]*$ ]]; then
	echo "advertised host must be a DNS name or IPv4 address" >&2
	exit 2
fi
if [[ ! -d "$network_input" ]]; then
	echo "network root does not exist: ${network_input}" >&2
	exit 2
fi
network_root="$(cd "$network_input" && pwd -P)"
manifest="${network_root}/network.json"
genesis="${network_root}/node1/config/genesis.json"
if [[ ! -f "$manifest" || ! -f "$genesis" ]]; then
	echo "network manifest or genesis is missing" >&2
	exit 2
fi

chain_id="$(jq -er '.chain_id | select(type == "string" and length > 0)' "$manifest")"
expected_sha="$(jq -er '.genesis_sha256 | select(type == "string" and test("^[0-9a-f]{64}$"))' "$manifest")"
actual_sha="$(sha256sum "$genesis" | cut -d ' ' -f 1)"
genesis_chain_id="$(jq -er '.chain_id | select(type == "string" and length > 0)' "$genesis")"
if [[ "$actual_sha" != "$expected_sha" || "$genesis_chain_id" != "$chain_id" ]]; then
	echo "genesis does not match the network manifest" >&2
	exit 1
fi
if ! jq -e '
  (.nodes | type == "array" and length > 0) and
  (all(.nodes[]; (.node_id | type == "string" and test("^[0-9a-f]{40}$")) and
                 (.p2p_port | type == "number" and floor == . and . >= 1 and . <= 65535))) and
  ([.nodes[].node_id] | unique | length) == (.nodes | length)
' "$manifest" >/dev/null; then
	echo "network manifest has invalid peer identities or ports" >&2
	exit 1
fi

output_parent_input="$(dirname -- "$output_input")"
output_name="$(basename -- "$output_input")"
if [[ ! -d "$output_parent_input" || "$output_name" == "." || "$output_name" == ".." ]]; then
	echo "output parent must exist and output must name a new directory" >&2
	exit 2
fi
output_parent="$(cd "$output_parent_input" && pwd -P)"
output_dir="${output_parent}/${output_name}"
if [[ -e "$output_dir" || -L "$output_dir" ]]; then
	echo "refusing to overwrite an existing bootstrap bundle: ${output_dir}" >&2
	exit 1
fi

staging="$(mktemp -d "${output_parent}/.chaoschain-bootstrap.XXXXXX")"
cleanup() {
	if [[ -d "$staging" ]]; then
		rm -rf -- "$staging"
	fi
}
trap cleanup EXIT

cp "$genesis" "${staging}/genesis.json"
jq -n --arg chain "$chain_id" --arg sha "$actual_sha" --arg host "$advertised_host" --slurpfile network "$manifest" '{
  chain_id: $chain,
  genesis_sha256: $sha,
  persistent_peers: ([$network[0].nodes[] | .node_id + "@" + $host + ":" + (.p2p_port | tostring)] | join(","))
}' >"${staging}/bootstrap.json"
(
	cd "$staging"
	sha256sum bootstrap.json genesis.json >SHA256SUMS
)
mv -T -n -- "$staging" "$output_dir"
if [[ -d "$staging" ]]; then
	echo "bootstrap destination appeared during export: ${output_dir}" >&2
	exit 1
fi

echo "public bootstrap bundle exported to ${output_dir}"
