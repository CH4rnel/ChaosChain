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
  genesis_sha256: $sha,
  nodes: [
    {node_id: ("a" * 40), p2p_port: 27656},
    {node_id: ("b" * 40), p2p_port: 27666}
  ]
}' >"${network_root}/network.json"

"${repo_root}/scripts/export-bootstrap.sh" "$network_root" "$bundle" seed.example.org
(
	cd "$bundle"
	sha256sum -c SHA256SUMS
)
[[ "$(find "$bundle" -maxdepth 1 -type f -printf '%f\n' | sort)" == "$(printf '%s\n' SHA256SUMS bootstrap.json genesis.json | sort)" ]]
jq -e --arg sha "$genesis_sha" '
  .chain_id == "chaos-devnet-1" and
  .genesis_sha256 == $sha and
  .persistent_peers == (("a" * 40) + "@seed.example.org:27656," + ("b" * 40) + "@seed.example.org:27666") and
  (keys | sort) == ["chain_id", "genesis_sha256", "persistent_peers"]
' "${bundle}/bootstrap.json" >/dev/null
if rg -q 'private signing key|/tmp/chaoschain-bootstrap-test' "$bundle"; then
	echo "bootstrap bundle leaked private data or builder paths" >&2
	exit 1
fi

if "${repo_root}/scripts/export-bootstrap.sh" "$network_root" "$bundle" seed.example.org >/dev/null 2>&1; then
	echo "bootstrap exporter overwrote an existing bundle" >&2
	exit 1
fi
printf '%s\n' 'tampered' >>"${network_root}/node1/config/genesis.json"
if "${repo_root}/scripts/export-bootstrap.sh" "$network_root" "${test_dir}/tampered-bundle" seed.example.org >/dev/null 2>&1; then
	echo "bootstrap exporter accepted a genesis hash mismatch" >&2
	exit 1
fi

echo "public bootstrap bundle tests passed"
