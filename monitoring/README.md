# Validator monitoring response

Prometheus evaluates `ChaosChainConsensusStalled` when a validator's latest committed block height has not changed for five minutes while its metrics endpoint remains reachable. The rule then waits another two minutes before firing. A missing endpoint is handled by `ChaosChainValidatorUnavailable` instead, and a low peer count or rising missed-block count has its own alert.

One stalled validator can indicate a local node or network path problem. Several validators reporting the same unchanged height points toward a chain-wide consensus stall. Compare at least two validator RPC endpoints and check their `/status` responses twice, about a minute apart, before deciding which case you have. Use `scripts/check-validator-health.sh` to verify the chain ID, catch-up state, and peer count, then inspect `journalctl -u chaoschaind --since "15 minutes ago"` on affected hosts.

Do not restart every validator at once. Preserve the logs and current heights first, confirm the remaining voting power can keep a quorum, and coordinate a single-node recovery with the other operators. If the whole network is intentionally halted, expect the stalled-height alert after metrics return while block height remains fixed. Silence it only for the planned maintenance window in the deployed alerting system, and remove that silence after the network resumes.

This repository defines and tests Prometheus alert rules, but it does not configure an Alertmanager receiver or notification channel. Confirm that the deployment routes firing alerts to an operator before relying on pages or messages; otherwise they are visible only in Prometheus.
