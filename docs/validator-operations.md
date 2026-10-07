# Validator health check

The health probe is a read-only snapshot of one validator's CometBFT RPC endpoints. It checks that the node reports the expected chain ID, is not catching up, and has at least the requested number of peers. It prints the latest reported block height when all checks pass.

Run it from a host that can reach the validator's RPC listener:

```sh
scripts/check-validator-health.sh http://127.0.0.1:26657 chaoschain-1 3
```

The final argument is the minimum peer count and defaults to one. Use the chain ID from the network's published genesis configuration. Keep RPC access restricted to trusted hosts; this check does not require or justify exposing the RPC listener to the public internet.

A failed check is a prompt to inspect the node and its connectivity, not an instruction to restart it automatically. Compare the reported height with other validators and inspect the daemon logs before intervening. For a planned stop, stop the validator through its process supervisor, confirm the process exited, and preserve its home directory. Start it again with the same home and network configuration, then rerun the health check and confirm that it has caught up and reconnected.

This snapshot does not measure historical uptime, missed blocks, consensus signing performance, alert delivery, or backup integrity. Those remain separate public-testnet readiness work; a green line here is not a production monitoring system.
