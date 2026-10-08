#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 1 ]]; then
	echo "Usage: $0 <release-archive>" >&2
	exit 2
fi

archive="$(realpath "$1")"
if [[ ! -f "$archive" ]]; then
	echo "release archive does not exist: ${archive}" >&2
	exit 1
fi
if ! command -v gh >/dev/null; then
	echo "GitHub CLI with attestation support is required to verify release provenance" >&2
	exit 1
fi

gh attestation verify "$archive" \
	--repo CH4rnel/ChaosChain \
	--signer-workflow CH4rnel/ChaosChain/.github/workflows/ci.yml
