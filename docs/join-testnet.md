# Join the testnet as a full node

Obtain the Linux AMD64 release archive, its `SHA256SUMS`, and the public bootstrap directory from the network operator. Verify both sets of files before running anything:

```sh
sha256sum -c SHA256SUMS
cd public-bootstrap
sha256sum -c SHA256SUMS
```

Extract the release and inspect `build-info.json`. The commit and version must match the release announced by the network operator. Set the paths below to the extracted binary, an empty node home, and the verified bootstrap directory.

```sh
CHAOSCHAIND=/opt/chaoschain/chaoschaind
CHAOSCHAIN_HOME=/var/lib/chaoschaind
BOOTSTRAP=/opt/chaoschain/public-bootstrap

CHAIN_ID=$(jq -er '.chain_id' "$BOOTSTRAP/bootstrap.json")
PEERS=$(jq -er '.persistent_peers' "$BOOTSTRAP/bootstrap.json")

"$CHAOSCHAIND" init my-node --chain-id "$CHAIN_ID" --home "$CHAOSCHAIN_HOME"
install -m 0644 "$BOOTSTRAP/genesis.json" "$CHAOSCHAIN_HOME/config/genesis.json"
"$CHAOSCHAIND" genesis validate "$CHAOSCHAIN_HOME/config/genesis.json" --home "$CHAOSCHAIN_HOME"
sed -i -E "s|^persistent_peers = .*|persistent_peers = \"$PEERS\"|" "$CHAOSCHAIN_HOME/config/config.toml"
sed -i -E 's|^minimum-gas-prices = .*|minimum-gas-prices = "0.025uchaos"|' "$CHAOSCHAIN_HOME/config/app.toml"
"$CHAOSCHAIND" start --home "$CHAOSCHAIN_HOME"
```

Run the daemon under a process supervisor for a persistent node. After it starts, compare its chain ID and height with an independent network endpoint and wait until `catching_up` is false. The repository health probe performs these checks when the RPC endpoint is reachable:

```sh
scripts/check-validator-health.sh http://127.0.0.1:26657 "$CHAIN_ID" 1
```

This procedure joins a non-validating full node. Becoming a validator requires a funded account, a staking transaction, secure key custody, and coordination with the live network parameters. Never copy another validator's home or private validator key to bootstrap a new machine.

The network operator creates the public directory from a stopped or running initialized network root with an advertised DNS name or IPv4 address:

```sh
scripts/export-bootstrap.sh /srv/chaoschain/network /srv/chaoschain/public-bootstrap seed.example.org
```
