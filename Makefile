# Makefile
.PHONY: build test test-cover lint sast vuln sbom clean

GOCMD=go
GOTEST=$(GOCMD) test
GOBUILD=$(GOCMD) build
LINT_CMD=golangci-lint run ./...
SAST_CMD=$(GOCMD) run github.com/securego/gosec/v2/cmd/gosec@latest ./...
VULN_CMD=$(GOCMD) run golang.org/x/vuln/cmd/govulncheck@latest ./...
SBOM_CMD=$(GOCMD) run github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest mod -json -output sbom.json

build:
	$(GOBUILD) -v -o ./bin/chaoschaind ./cmd/chaoschaind

test:
	$(GOTEST) -race -count=1 ./...

test-cover:
	$(GOTEST) -race -count=1 -covermode=atomic -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

lint:
	$(LINT_CMD)

sast:
	$(SAST_CMD)

vuln:
	$(VULN_CMD)

sbom:
	$(SBOM_CMD)

clean:
	rm -rf ./bin coverage.out coverage.html sbom.json