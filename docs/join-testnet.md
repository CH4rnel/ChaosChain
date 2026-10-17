# Join the testnet as a full node

Obtain the Linux AMD64 daemon release and the public bootstrap bundle from the network operator. Verify the release checksum and inspect its build information against the release announced by the operator. The bootstrap bundle's own checksums only show that its files agree with that bundle; they do not authenticate which chain it describes.

Get the expected genesis SHA-256 from the operator through a separate authenticated channel. Do not copy it from `bootstrap.json`, `SHA256SUMS`, or another file in the bundle. Compare it with the bundle's genesis before joining:

```sh
cd /opt/chaoschain/public-bootstrap
sha256sum genesis.json
```

Run the join command with the same independently obtained fingerprint. It verifies the exact checksum manifest, the external fingerprint, the manifest/genesis chain identity, and the peer list before creating a fresh node home. The daemon then validates genesis before the new home is installed. Existing homes are never overwritten.

```sh
CHAOSCHAIND=/opt/chaoschain/chaoschaind
BOOTSTRAP=/opt/chaoschain/public-bootstrap
CHAOSCHAIN_HOME="$HOME/.local/share/chaoschain-testnet"
TRUSTED_GENESIS_SHA256=replace-with-the-64-character-value-from-the-operator

CHAOSCHAIN_BINARY="$CHAOSCHAIND" scripts/join-testnet.sh \
  "$BOOTSTRAP" \
  "$CHAOSCHAIN_HOME" \
  my-full-node \
  "$TRUSTED_GENESIS_SHA256"
```

The command initializes a new local node identity and copies no validator key from the bootstrap bundle. It pins RPC to loopback and leaves REST and gRPC disabled. P2P uses the peers from the verified bundle; allow only the required P2P port through the host firewall. The home path's parent must already exist and be writable by the account that will run the daemon:

```sh
"$CHAOSCHAIND" start --home "$CHAOSCHAIN_HOME"
```

After startup, check the expected chain ID, peer count, and catch-up state from a trusted host:

```sh
CHAIN_ID="$(jq -er '.chain_id' "$BOOTSTRAP/bootstrap.json")"
scripts/check-validator-health.sh http://127.0.0.1:26657 "$CHAIN_ID" 1
```

This creates a non-validating full node. Becoming a validator requires a funded account, a staking transaction, secure key custody, and coordination with the live network parameters. Never copy another validator's home or private validator key to bootstrap a new machine.
