#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly repo_root
readonly output_dir="${1:-${repo_root}/dist}"
readonly version="${VERSION:-$(git -C "$repo_root" describe --tags --always --dirty)}"
readonly commit="${COMMIT:-$(git -C "$repo_root" rev-parse HEAD)}"
readonly build_time="${BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
safe_version="$(tr -cs 'A-Za-z0-9._-' '-' <<<"$version" | sed 's/^-//; s/-$//')"
readonly safe_version
readonly release_name="chaoschain-${safe_version}-linux-amd64"
readonly archive_name="${release_name}.tar.gz"
staging="$(mktemp -d /tmp/chaoschain-release.XXXXXX)"
readonly staging
readonly bundle="${staging}/${release_name}"

cleanup() {
	rm -rf "$staging"
}
trap cleanup EXIT

mkdir -p "$bundle" "$output_dir"
cp "${repo_root}/LICENSE" "${bundle}/LICENSE"
ldflags="-X github.com/CH4rnel/ChaosChain/internal/buildinfo.Version=${version} -X github.com/CH4rnel/ChaosChain/internal/buildinfo.Commit=${commit} -X github.com/CH4rnel/ChaosChain/internal/buildinfo.BuildTime=${build_time}"
(
	cd "$repo_root"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$ldflags" -o "${bundle}/chaoschaind" ./cmd/chaoschaind
)
"${bundle}/chaoschaind" version --output json >"${bundle}/build-info.json"
tar --sort=name --owner=0 --group=0 --numeric-owner --mtime="$build_time" -C "$staging" -czf "${output_dir}/${archive_name}" "$release_name"
(
	cd "$output_dir"
	sha256sum "$archive_name" >SHA256SUMS
)

printf '%s\n' "${output_dir}/${archive_name}"
