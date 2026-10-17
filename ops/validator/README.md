# Validator host installation

This installer prepares one fresh Linux AMD64 host from a release archive and a public bootstrap bundle. It verifies both SHA256SUMS files, checks the bootstrap manifest against genesis, initializes a new local validator identity, and installs a hardened systemd service. It refuses to replace an existing validator home. Never copy a validator home or private validator key between hosts.

## Published validator releases

The CI workflow publishes a durable GitHub release when a `vMAJOR.MINOR.PATCH` tag points to a commit already present in `master`. The archive and `SHA256SUMS` are attached only after the release archive receives a GitHub artifact attestation from this repository's CI workflow. Pull requests and ordinary branch builds do not publish GitHub releases. Do not reuse a version tag.

Create and push a version tag only after reviewing the commit and confirming that CI is green:

```sh
git tag -a v1.2.3 -m "chaoschain v1.2.3"
git push origin v1.2.3
```

Wait for the tag's `ChaosChain CI` run to finish successfully, then download and verify the exact release assets before installation:

```sh
release_dir="$(mktemp -d)"
gh release download v1.2.3 \
  --repo CH4rnel/ChaosChain \
  --pattern 'chaoschain-v1.2.3-linux-amd64.tar.gz' \
  --pattern SHA256SUMS \
  --dir "$release_dir"
(
  cd "$release_dir"
  sha256sum --check --strict SHA256SUMS
  gh attestation verify chaoschain-v1.2.3-linux-amd64.tar.gz \
    --repo CH4rnel/ChaosChain \
    --signer-workflow CH4rnel/ChaosChain/.github/workflows/ci.yml
)
```

The validator installer repeats checksum and provenance verification before extracting the daemon or creating a node home. A checksum detects changed bytes; the attestation identifies the trusted build workflow. Neither proves that the genesis belongs to the intended network, so provide the independently obtained genesis fingerprint as documented below.

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

For block-progress, peer, missed-block, and scrape alerts, see the [validator monitoring response notes](../../monitoring/README.md). The Prometheus rules are tested in CI, but this repository does not configure Alertmanager notifications.

The isolated fixture test runs without root, creates no system account, and uses stubbed systemd commands:

```sh
make test-validator-install
make test-verify-release-provenance
make test-public-peer-preflight
```
