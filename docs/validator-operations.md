# Validator health check

The health probe is a read-only snapshot of one validator's CometBFT RPC endpoints. It checks that the node reports the expected chain ID, is not catching up, and has at least the requested number of peers. It prints the latest reported block height when all checks pass.

Run it from a host that can reach the validator's RPC listener:

```sh
scripts/check-validator-health.sh http://127.0.0.1:26657 chaoschain-1 3
```

The final argument is the minimum peer count and defaults to one. Use the chain ID from the network's published genesis configuration. Keep RPC access restricted to trusted hosts; this check does not require or justify exposing the RPC listener to the public internet.

A failed check is a prompt to inspect the node and its connectivity, not an instruction to restart it automatically. Compare the reported height with other validators and inspect the daemon logs before intervening. For a planned stop, stop the validator through its process supervisor, confirm the process exited, and preserve its home directory. Start it again with the same home and network configuration, then rerun the health check and confirm that it has caught up and reconnected.

The four-node profile enables CometBFT metrics on loopback ports 27660, 27670, 27680, and 27690. From the repository root, a Prometheus instance on the same host can scrape them with:

```sh
prometheus --config.file=monitoring/prometheus-devnet.yml --web.listen-address=127.0.0.1:9090
```

The example rules in `monitoring/alerts-devnet.yml` flag unreachable nodes, low peer count, stalled signing, and a rising missed block count. In Prometheus, `100 * avg_over_time(up{job="chaoschain-devnet"}[1h])` shows the percentage of successful scrapes over one hour. That is scrape availability, not a historical guarantee of consensus participation. Check `cometbft_consensus_validator_last_signed_height` alongside the latest block height for signing progress.

CometBFT publishes its missed block metric only after a miss and resets that gauge when the process restarts. The delta rule needs at least two samples, so the first miss and a restart can escape that rule; the signing lag and unreachable-node rules cover other active failures. Prometheus evaluates these rules locally; configure an Alertmanager separately if notifications are needed. Keep metrics endpoints on loopback or an authenticated private network, and replace the static targets when moving beyond this single-host devnet.
