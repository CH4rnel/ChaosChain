# ChaosChain testnet explorer

This deployment builds Ping Dashboard from the exact commit in `.env` and serves it beside a read-only proxy to a dedicated non-validator node. It does not add Ping, Node, or nginx dependencies to the ChaosChain Go module.

Create a fresh non-validator home from a checksum-verified public bootstrap bundle. The bundle carries the chain ID, minimum gas price, genesis, and public peers. The node initializer checks those values, enables transaction indexing and complete history, and binds P2P, RPC, and REST to loopback:

```sh
ops/explorer/scripts/init-explorer-node.sh /srv/chaoschain/public-bootstrap /var/lib/chaoschain-explorer explorer-node 26656 26657 1317
chaoschaind start --home /var/lib/chaoschain-explorer
```

Use the regular `start` command. `start-native` intentionally rejects REST, gRPC, and gRPC-Web because its in-process host currently exposes CometBFT RPC only.

For an existing stopped full-node home, `scripts/enable-explorer-api.sh` applies the archive and REST settings with timestamped config backups. It does not replace the node's genesis or peers.

Copy `.env.example` to `.env`, replace the public URL, and review the pinned Ping commit before deployment. Then build and start the service:

```sh
cd ops/explorer
docker compose config
docker compose up -d --build
```

The container listens on loopback by default so a TLS terminator can publish it. Keep the node REST and RPC listeners on loopback as well. If another reverse proxy sits in front of this container, configure its trusted client IP handling before changing the listen address or relying on per-IP rate limits.

The public REST path accepts only GET and HEAD requests. The CometBFT path additionally restricts requests to the explicit read-method allowlist in `nginx/explorer.conf.template`; transaction broadcast, peer discovery, unsafe RPC, and arbitrary JSON-RPC POST requests remain unavailable.

Run the local contract checks with:

```sh
ops/explorer/scripts/test-enable-explorer-api.sh
ops/explorer/scripts/test-init-explorer-node.sh
ops/explorer/scripts/test-ping-build-inputs.sh
CHAOSCHAIN_EXPLORER_IMAGE=chaoschain-ping:test ops/explorer/scripts/test-proxy-policy.sh
```

The last check requires Docker and uses temporary loopback ports. It verifies the dashboard, same-origin CORS response, REST write denial, and RPC allowlist against a disposable upstream.
