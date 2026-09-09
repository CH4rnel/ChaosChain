# Makefile
.PHONY: build test test-cover lint sast vuln sbom clean install-tools clean-tools

GOCMD=go
GOTEST=$(GOCMD) test
GOBUILD=$(GOCMD) build

# Explicitly resolve the path to GOPATH to avoid conflicts with /usr/bin.
GOPATH=$(shell go env GOPATH)
GOLANGCI_LINT_BIN=$(GOPATH)/bin/golangci-lint

# Version pinning
GOLANGCI_LINT_VERSION=latest
GOSEC_VERSION=v2.22.3

build:
	$(GOBUILD) -v -o ./bin/chaoschaind ./cmd/chaoschaind

test:
	$(GOTEST) -race -count=1 ./...

test-cover:
	$(GOTEST) -race -count=1 -covermode=atomic -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

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
	$(GOCMD) run golang.org/x/vuln/cmd/govulncheck@latest ./...

sbom:
	$(GOCMD) run github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest mod -json -output sbom.json

clean: clean-tools
	rm -rf ./bin coverage.out coverage.html sbom.json