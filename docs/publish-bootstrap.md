# Publish a bootstrap bundle from real validator endpoints

The public bundle is a map to the live network, not a copy of the local four-process rehearsal. `network.json` supplies the canonical chain ID, gas price, and genesis fingerprint; a separate endpoint file supplies the node IDs and reachable P2P addresses of the actual hosts. Do not publish node IDs copied from a local devnet or a validator's private key.

After the private validators are installed and running, collect each node ID from that host's loopback RPC endpoint. For the default service configuration:

```sh
curl -fsS http://127.0.0.1:26657/status | jq -er '.result.node_info.id | select(test("^[0-9a-f]{40}$"))'
```

Confirm the matching public DNS name or IPv4 address and inbound P2P port with each host operator. Keep the order stable and create a local file such as `peer-endpoints.json`:

```json
{
  "peers": [
    {"node_id": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "host": "validator-one.example.org", "p2p_port": 26656},
    {"node_id": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "host": "198.51.100.10", "p2p_port": 26656}
  ]
}
```

The example IDs and addresses are placeholders. Each node ID must match the installed host's own RPC status. The exporter rejects empty lists, malformed IDs, hosts or ports, duplicate IDs, duplicate address/port pairs, and extra fields. It does not prove that an endpoint is reachable or that the node is a validator.

Run the export from a trusted network root containing `network.json` and `node1/config/genesis.json`. The destination must not already exist:

```sh
scripts/export-bootstrap.sh /path/to/network-root /path/to/public-bootstrap peer-endpoints.json
(
  cd /path/to/public-bootstrap
  sha256sum --check --strict SHA256SUMS
)
```

Publish only the generated bundle, not the network root or host homes. Give joiners the expected genesis SHA-256 through a separate authenticated channel; the bundle's own checksums cannot authenticate the chain. From an independent host, run `scripts/check-public-peers.sh /path/to/public-bootstrap 2 "$trusted_genesis_sha256"` before asking an outsider to join. Re-export to a new destination and redistribute the bundle whenever a node ID or advertised address changes.

This procedure republishes endpoints after the private hosts exist. It does not provision the first seed host, establish the genesis validator set, or prove four-host consensus. Those steps still need an actual multi-VPS rehearsal and observation across the network.
