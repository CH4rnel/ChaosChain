#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 3 ]]; then
	echo "Usage: $0 <network-root> <output-directory> <peer-endpoints.json>" >&2
	exit 2
fi

network_input="$1"
output_input="$2"
endpoints="$3"
if [[ ! -f "$endpoints" ]]; then
	echo "peer endpoints file does not exist: ${endpoints}" >&2
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
minimum_gas_price="$(jq -er '.minimum_gas_price | select(type == "string" and test("^[0-9]+([.][0-9]+)?[A-Za-z][A-Za-z0-9/:._-]*$"))' "$manifest")"
expected_sha="$(jq -er '.genesis_sha256 | select(type == "string" and test("^[0-9a-f]{64}$"))' "$manifest")"
actual_sha="$(sha256sum "$genesis" | cut -d ' ' -f 1)"
genesis_chain_id="$(jq -er '.chain_id | select(type == "string" and length > 0)' "$genesis")"
if [[ "$actual_sha" != "$expected_sha" || "$genesis_chain_id" != "$chain_id" ]]; then
	echo "genesis does not match the network manifest" >&2
	exit 1
fi
if ! jq -e '
  def valid_host:
    type == "string" and length <= 253 and
    (split(".") | length > 0 and all(.[]; length > 0 and length <= 63 and test("^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?$"))) and
    (if test("^[0-9.]+$") then
      (split(".") | length == 4 and all(.[]; test("^[0-9]{1,3}$") and (tonumber <= 255)))
    else true end);
  (.peers | type == "array" and length > 0) and
  (all(.peers[]; (keys | sort) == ["host", "node_id", "p2p_port"] and
    (.node_id | type == "string" and test("^[0-9a-f]{40}$")) and
    (.host | valid_host) and
    (.p2p_port | type == "number" and floor == . and . >= 1 and . <= 65535))) and
  ([.peers[].node_id] | unique | length) == (.peers | length) and
  ([.peers[] | .host + ":" + (.p2p_port | tostring)] | unique | length) == (.peers | length)
' "$endpoints" >/dev/null; then
	echo "peer endpoints file has invalid or duplicate identities, hosts, or ports" >&2
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
jq -n --arg chain "$chain_id" --arg minimum_gas_price "$minimum_gas_price" --arg sha "$actual_sha" --slurpfile endpoints "$endpoints" '{
  chain_id: $chain,
  minimum_gas_price: $minimum_gas_price,
  genesis_sha256: $sha,
  persistent_peers: ([$endpoints[0].peers[] | .node_id + "@" + .host + ":" + (.p2p_port | tostring)] | join(","))
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
