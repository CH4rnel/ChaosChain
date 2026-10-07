# Offline validator-home backups

The backup command archives a validator home only after the operator confirms that `chaoschaind` is stopped. It does not detect every process manager or prove that the database is idle; stop the node through its supervisor and confirm the process has exited before running it.

```sh
scripts/backup-validator-home.sh /var/lib/chaoschaind /srv/chaoschain-backups/node1-2026-10-07.tar.gz --confirm-stopped
```

The destination directory must already exist and must be outside the validator home. The command refuses to replace an existing archive or checksum. Keep both generated files together and store a copy somewhere that does not share the node's disk failure domain.

Restore the archive into a new path after stopping the validator and confirming that its original process has exited:

```sh
scripts/restore-validator-home.sh /srv/chaoschain-backups/node1-2026-10-07.tar.gz /var/lib/chaoschaind-restore --confirm-stopped
```

The restore command checks the adjacent checksum, extracts into a private temporary directory, and then creates the requested home path. That path must not already exist. Restore only for the same validator and network. Never run the original home and its restored copy at the same time: both contain the same validator signing key. Before starting the restored node, check its chain configuration and genesis against the intended network, set ownership for the daemon user, then use the normal process supervisor and health probe to confirm recovery.

The four-validator CI rehearsal now replaces one stopped node home with a restored archive, verifies its validator identity and application state, and checks that all four nodes agree after restart. The checksum detects accidental archive corruption; it does not authenticate the archive or prove that an operator's own backup is recent.
