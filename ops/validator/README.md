# Validator host installation

This installer prepares one fresh Linux AMD64 host from a release archive and a public bootstrap bundle. It verifies both SHA256SUMS files, checks the bootstrap manifest against genesis, initializes a new local validator identity, and installs a hardened systemd service. It refuses to replace an existing validator home. Never copy a validator home or private validator key between hosts.

The host needs Bash, jq, tar, coreutils, systemd, GitHub CLI with attestation support, and a ChaosChain release archive. The installer verifies the archive checksum and requires GitHub's artifact attestation to identify the ChaosChain repository and its CI workflow as the signer before it extracts or installs anything. Use an archive produced by a push to the official master branch; pull request build artifacts are not signed. Run the installer as root with the release archive, the matching release SHA256SUMS file, the extracted public bootstrap directory, a unique moniker, and the trusted genesis SHA-256 fingerprint:

```sh
trusted_genesis_sha256=replace-this-with-the-64-character-value-from-the-operator
sudo ops/validator/scripts/install-validator.sh \
  chaoschaind-vX-linux-amd64.tar.gz \
  SHA256SUMS \
  /opt/chaoschain/public-bootstrap \
  validator-one \
  "$trusted_genesis_sha256"
```

Get the trusted fingerprint from the network operator through a separate authenticated channel, then compare it with `sha256sum /opt/chaoschain/public-bootstrap/genesis.json`. Do not copy the value from bootstrap.json or SHA256SUMS in that same bundle; those files can agree perfectly while the entire bundle describes the wrong chain. The installer rejects a missing, malformed, or mismatched fingerprint before installing the daemon or creating a validator home.

The installer creates the chaoschain system account, initializes `/var/lib/chaoschain` on that host, validates genesis, and enables `chaoschaind.service`. The service runs without Linux capabilities, with a read-only system filesystem and access only to its node home. The private validator key is mode 0600. CometBFT RPC remains on loopback and Prometheus metrics remain disabled by the daemon defaults; expose only the configured P2P port in the host firewall.

Before installing, check peer reachability from the VPS itself using the same bootstrap bundle and the independently obtained genesis fingerprint. The second argument is the minimum number of advertised peers that must accept a TCP connection:

```sh
scripts/check-public-peers.sh /opt/chaoschain/public-bootstrap 2 "$trusted_genesis_sha256"
```

The check verifies bundle checksums, compares genesis against the independently supplied fingerprint, confirms the manifest/genesis match, resolves each advertised DNS name or IPv4 address, and attempts its P2P port with a short timeout. It tests outbound reachability from this host; it does not prove that inbound firewall rules work, that a remote node is synchronized, or that consensus is healthy. Run the reachability check from each host and use `scripts/check-validator-health.sh` after the node starts.

Before using this on a real validator, review the release commit and genesis with the network operator, confirm the advertised persistent peers are reachable, and verify the VPS firewall allows inbound P2P while keeping RPC, REST, SSH, and metrics restricted. This installer does not provision a VPS, create a funded account, submit a validator transaction, or prove independent multi-host consensus. Confirm service health and catch-up separately with `scripts/check-validator-health.sh`.

The isolated fixture test runs without root, creates no system account, and uses stubbed systemd commands:

```sh
make test-validator-install
make test-verify-release-provenance
make test-public-peer-preflight
```
