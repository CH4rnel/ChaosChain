# Validator host installation

This installer prepares one fresh Linux AMD64 host from a release archive and a public bootstrap bundle. It verifies both SHA256SUMS files, checks the bootstrap manifest against genesis, initializes a new local validator identity, and installs a hardened systemd service. It refuses to replace an existing validator home. Never copy a validator home or private validator key between hosts.

The host needs Bash, jq, tar, coreutils, systemd, and a ChaosChain release archive. Run the installer as root with the release archive, the matching release SHA256SUMS file, the extracted public bootstrap directory, and a unique moniker:

```sh
sudo ops/validator/scripts/install-validator.sh \
  chaoschaind-vX-linux-amd64.tar.gz \
  SHA256SUMS \
  /opt/chaoschain/public-bootstrap \
  validator-one
```

The installer creates the chaoschain system account, initializes `/var/lib/chaoschain` on that host, validates genesis, and enables `chaoschaind.service`. The service runs without Linux capabilities, with a read-only system filesystem and access only to its node home. The private validator key is mode 0600. CometBFT RPC remains on loopback and Prometheus metrics remain disabled by the daemon defaults; expose only the configured P2P port in the host firewall.

Before using this on a real validator, review the release commit and genesis with the network operator, confirm the advertised persistent peers are reachable, and verify the VPS firewall allows inbound P2P while keeping RPC, REST, SSH, and metrics restricted. This installer does not provision a VPS, create a funded account, submit a validator transaction, or prove independent multi-host consensus. Confirm service health and catch-up separately with `scripts/check-validator-health.sh`.

The isolated fixture test runs without root, creates no system account, and uses stubbed systemd commands:

```sh
make test-validator-install
```
