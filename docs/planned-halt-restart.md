# Planned halt and restart

This is an operator coordinated stop of an existing testnet. It does not schedule an on-chain halt or control remote hosts. Agree on a UTC maintenance window, the expected chain ID, the binary release, the validator roster, and who will confirm that enough voting power is online after restart. Keep CometBFT RPC private; use a trusted network or SSH tunnels to reach multiple validators from the operator workstation.

Before stopping anything, check at least two validators with different `validator_info.address` values and record a block they both have. The checkpoint height and hash belong in the maintenance record outside the node homes. This example assumes the two RPC URLs are already reachable through private access:

```sh
CHAIN_ID=chaos-devnet-1
RPC_A=http://127.0.0.1:27657
RPC_B=http://127.0.0.1:27667

scripts/check-validator-health.sh "$RPC_A" "$CHAIN_ID" 1
scripts/check-validator-health.sh "$RPC_B" "$CHAIN_ID" 1
ADDRESS_A="$(curl -fsS --max-time 5 "$RPC_A/status" | jq -er '.result.validator_info.address')"
ADDRESS_B="$(curl -fsS --max-time 5 "$RPC_B/status" | jq -er '.result.validator_info.address')"
HEIGHT_A="$(curl -fsS --max-time 5 "$RPC_A/status" | jq -er '.result.sync_info.latest_block_height | tonumber')"
HEIGHT_B="$(curl -fsS --max-time 5 "$RPC_B/status" | jq -er '.result.sync_info.latest_block_height | tonumber')"
CHECKPOINT_HEIGHT=$((HEIGHT_A < HEIGHT_B ? HEIGHT_A : HEIGHT_B))
CHECKPOINT_HASH="$(curl -fsS --max-time 5 "$RPC_A/block?height=$CHECKPOINT_HEIGHT" | jq -er '.result.block_id.hash')"
OTHER_HASH="$(curl -fsS --max-time 5 "$RPC_B/block?height=$CHECKPOINT_HEIGHT" | jq -er '.result.block_id.hash')"
if [[ "$ADDRESS_A" == "$ADDRESS_B" ]]; then
  echo 'rpc endpoints report the same validator identity; do not begin the halt' >&2
elif [[ "$CHECKPOINT_HASH" == "$OTHER_HASH" ]]; then
  printf 'chain=%s height=%s hash=%s\n' "$CHAIN_ID" "$CHECKPOINT_HEIGHT" "$CHECKPOINT_HASH"
else
  echo 'validators disagree at checkpoint; do not begin the halt' >&2
fi
```

Record the binary version and genesis fingerprint on each host, save recent logs, and arrange an alert silence limited to this maintenance window. Stop each `chaoschaind.service` through systemd and confirm every process has exited. Do not copy a running validator home. If a backup is required, run `scripts/backup-validator-home.sh /var/lib/chaoschain /srv/chaoschain-backups/<host>.tar.gz --confirm-stopped` on each stopped host and store each archive separately.

Keep every validator's existing genesis, private key, and `data/priv_validator_state.json` with its own home. Never start two processes with the same signing key. CometBFT uses the last signed state to prevent conflicting signatures; restoring an older copy after that validator has signed later blocks requires a separate, coordinated recovery decision. See the [CometBFT signing specification](https://github.com/cometbft/cometbft/blob/main/spec/consensus/signing.md) and [validator configuration reference](https://github.com/cometbft/cometbft/blob/main/docs/core/configuration.md).

If this halt includes a binary change, verify the release checksum and artifact attestation as described in the [validator release instructions](../ops/validator/README.md) before replacing the binary. Restart the existing service on each host through systemd. Coordinate the return of enough voting power for consensus, then check every validator from private RPC endpoints. The recovery check requires at least two distinct validator identities and verifies that each node reports the expected chain, has peers, is no longer catching up, has passed the recorded height, and returns the same block hash at that height:

```sh
scripts/check-network-recovery.sh "$CHAIN_ID" "$CHECKPOINT_HEIGHT" "$CHECKPOINT_HASH" \
  "$RPC_A" "$RPC_B" "$RPC_C" "$RPC_D"
```

Use the actual validator RPC URLs or private tunnels for `RPC_C` and `RPC_D`. A passing check is a shared checkpoint and forward progress, not a full application audit. Confirm the expected account or staking state through the CLI, review validator logs and signing metrics, and remove the alert silence once the network is moving normally.

If the check fails, preserve the failing responses and logs, identify the affected host or disagreement, and coordinate the next action with the operators. Do not reset state, restore an old signing state, or start a second copy of a validator key as a blind retry. The four-validator CI rehearsal exercises this same recovery check after a cold restart and binary replacement; independent VPS operation still requires a real maintenance drill.
