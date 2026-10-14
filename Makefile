# Makefile
.PHONY: build release test test-cover check-network-profile check-monitoring test-export-bootstrap test-validator-health test-validator-backup test-validator-restore test-validator-install test-verify-release-provenance test-public-peer-preflight test-cli-smoke test-multinode-smoke test-release lint sast vuln sbom clean install-tools clean-tools

GOCMD=go
GOTEST=$(GOCMD) test
GOBUILD=$(GOCMD) build
VERSION?=$(shell git describe --tags --always --dirty)
COMMIT?=$(shell git rev-parse HEAD)
BUILD_TIME?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILD_LDFLAGS=-X github.com/CH4rnel/ChaosChain/internal/buildinfo.Version=$(VERSION) -X github.com/CH4rnel/ChaosChain/internal/buildinfo.Commit=$(COMMIT) -X github.com/CH4rnel/ChaosChain/internal/buildinfo.BuildTime=$(BUILD_TIME)

# Explicitly resolve the path to GOPATH to avoid conflicts with /usr/bin.
GOPATH=$(shell go env GOPATH)
GOLANGCI_LINT_BIN=$(GOPATH)/bin/golangci-lint

# Version pinning
GOLANGCI_LINT_VERSION=latest
GOSEC_VERSION=v2.22.3
CYCLONEDX_GOMOD_VERSION=v1.12.0
SBOM_OUTPUT?=sbom.json

build:
	mkdir -p ./bin
	$(GOBUILD) -v -ldflags "$(BUILD_LDFLAGS)" -o ./bin/chaoschaind ./cmd/chaoschaind

release:
	VERSION="$(VERSION)" COMMIT="$(COMMIT)" BUILD_TIME="$(BUILD_TIME)" bash scripts/build-release.sh

test:
	$(GOTEST) -race -count=1 ./...

test-cover:
	$(GOTEST) -race -count=1 -covermode=atomic -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

check-network-profile:
	bash scripts/check-network-profile.sh

check-monitoring:
	docker run --rm --network none --read-only -v "$(CURDIR):/project:ro" -w /project --entrypoint /bin/promtool prom/prometheus@sha256:29c1e0b48ad816750245dea3d65ac3ce57dda30547e67b1e9a5176e2953cc4dc check config monitoring/prometheus-devnet.yml
	docker run --rm --network none --read-only --tmpfs /tmp:rw,noexec,nosuid,size=64m -v "$(CURDIR):/project:ro" -w /project --entrypoint /bin/promtool prom/prometheus@sha256:29c1e0b48ad816750245dea3d65ac3ce57dda30547e67b1e9a5176e2953cc4dc test rules monitoring/alerts-devnet.test.yml

test-export-bootstrap:
	bash scripts/test-export-bootstrap.sh

test-validator-health:
	bash scripts/test-validator-health.sh

test-validator-backup:
	bash scripts/test-validator-backup.sh

test-validator-restore:
	bash scripts/test-validator-restore.sh

test-validator-install:
	bash ops/validator/scripts/test-install-validator.sh

test-verify-release-provenance:
	bash ops/validator/scripts/test-verify-release-provenance.sh

test-public-peer-preflight:
	bash scripts/test-public-peer-preflight.sh

test-cli-smoke: check-network-profile
	bash scripts/test-cli-smoke.sh

test-multinode-smoke: check-network-profile
	bash scripts/test-multinode-smoke.sh

test-release:
	bash scripts/test-release.sh

clean-tools:
	@echo "Removing old golangci-lint binary..."
	@rm -f $(GOLANGCI_LINT_BIN)

install-tools: clean-tools
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

# We invoke it explicitly using the absolute path, rather than via $PATH.
lint: install-tools
	$(GOLANGCI_LINT_BIN) run ./...

sast:
	$(GOCMD) run github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION) ./...

vuln:
	bash scripts/check-vuln.sh

sbom:
	GOTOOLCHAIN=go1.26.9 $(GOCMD) run github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@$(CYCLONEDX_GOMOD_VERSION) mod -json -output $(SBOM_OUTPUT)

clean: clean-tools
	rm -rf ./bin coverage.out coverage.html sbom.json
