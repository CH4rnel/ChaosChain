#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_dir="$(mktemp -d /tmp/chaoschain-bootstrap-test.XXXXXX)"
trap 'rm -rf "$test_dir"' EXIT

network_root="${test_dir}/network"
bundle="${test_dir}/public-bootstrap"
mkdir -p "${network_root}/node1/config"
printf '%s\n' '{"chain_id":"chaos-devnet-1"}' >"${network_root}/node1/config/genesis.json"
printf '%s\n' 'private signing key' >"${network_root}/node1/config/priv_validator_key.json"
genesis_sha="$(sha256sum "${network_root}/node1/config/genesis.json" | cut -d ' ' -f 1)"
jq -n --arg sha "$genesis_sha" '{
  chain_id: "chaos-devnet-1",
  minimum_gas_price: "0.025uchaos",
  genesis_sha256: $sha,
  nodes: [
    {node_id: ("a" * 40), p2p_port: 27656},
    {node_id: ("b" * 40), p2p_port: 27666}
  ]
}' >"${network_root}/network.json"
endpoints="${test_dir}/peer-endpoints.json"
jq -n '{peers: [
  {node_id: ("c" * 40), host: "alpha.example.org", p2p_port: 26656},
  {node_id: ("d" * 40), host: "198.51.100.10", p2p_port: 26656}
]}' >"$endpoints"

"${repo_root}/scripts/export-bootstrap.sh" "$network_root" "$bundle" "$endpoints"
(
	cd "$bundle"
	sha256sum -c SHA256SUMS
)
[[ "$(find "$bundle" -maxdepth 1 -type f -printf '%f\n' | sort)" == "$(printf '%s\n' SHA256SUMS bootstrap.json genesis.json | sort)" ]]
jq -e --arg sha "$genesis_sha" '
  .chain_id == "chaos-devnet-1" and
  .minimum_gas_price == "0.025uchaos" and
  .genesis_sha256 == $sha and
  .persistent_peers == (("c" * 40) + "@alpha.example.org:26656," + ("d" * 40) + "@198.51.100.10:26656") and
  (keys | sort) == ["chain_id", "genesis_sha256", "minimum_gas_price", "persistent_peers"]
' "${bundle}/bootstrap.json" >/dev/null
if rg -q 'private signing key|/tmp/chaoschain-bootstrap-test' "$bundle"; then
	echo "bootstrap bundle leaked private data or builder paths" >&2
	exit 1
fi

if "${repo_root}/scripts/export-bootstrap.sh" "$network_root" "$bundle" "$endpoints" >/dev/null 2>&1; then
	echo "bootstrap exporter overwrote an existing bundle" >&2
	exit 1
fi
for mutation in duplicate_id duplicate_endpoint bad_host bad_ip bad_port extra_key; do
	case "$mutation" in
		duplicate_id) filter='.peers[1].node_id = .peers[0].node_id' ;;
		duplicate_endpoint) filter='.peers[1].host = .peers[0].host | .peers[1].p2p_port = .peers[0].p2p_port' ;;
		bad_host) filter='.peers[1].host = "bad_host"' ;;
		bad_ip) filter='.peers[1].host = "999.51.100.10"' ;;
		bad_port) filter='.peers[1].p2p_port = 0' ;;
		extra_key) filter='.peers[1].private_key = "secret"' ;;
	esac
	jq "$filter" "$endpoints" >"${test_dir}/invalid-${mutation}.json"
	if "${repo_root}/scripts/export-bootstrap.sh" "$network_root" "${test_dir}/invalid-${mutation}-bundle" "${test_dir}/invalid-${mutation}.json" >/dev/null 2>&1; then
		echo "bootstrap exporter accepted ${mutation}" >&2
		exit 1
	fi
done
printf '%s\n' 'tampered' >>"${network_root}/node1/config/genesis.json"
if "${repo_root}/scripts/export-bootstrap.sh" "$network_root" "${test_dir}/tampered-bundle" "$endpoints" >/dev/null 2>&1; then
	echo "bootstrap exporter accepted a genesis hash mismatch" >&2
	exit 1
fi

echo "public bootstrap bundle tests passed"
