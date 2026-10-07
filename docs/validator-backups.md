# Offline validator-home backups

The backup command archives a validator home only after the operator confirms that `chaoschaind` is stopped. It does not detect every process manager or prove that the database is idle; stop the node through its supervisor and confirm the process has exited before running it.

```sh
scripts/backup-validator-home.sh /var/lib/chaoschaind /srv/chaoschain-backups/node1-2026-10-07.tar.gz --confirm-stopped
```

The destination directory must already exist and must be outside the validator home. The command refuses to replace an existing archive or checksum. Keep both generated files together and store a copy somewhere that does not share the node's disk failure domain.

Check the archive before restoring it:

```sh
cd /srv/chaoschain-backups
sha256sum -c node1-2026-10-07.tar.gz.sha256
mkdir /var/lib/chaoschaind-restore
tar -xzf node1-2026-10-07.tar.gz -C /var/lib/chaoschaind-restore
```

Restore into a new, empty directory for the same validator and network. Do not merge an old home into a running node or copy one validator's state into another validator's identity. Before starting the restored node, check its chain configuration and genesis against the intended network, then use the normal process supervisor and health probe to confirm recovery.

This is an offline file archive, not a hot snapshot, remote backup service, retention policy, or disaster-recovery rehearsal. A checksum detects accidental archive corruption; it does not prove the archive is recent or restorable until an operator tests the restore.
