#!/usr/bin/env bash
set -euo pipefail

readonly version="v0.0.0-test"
commit="$(git rev-parse HEAD)"
readonly commit
readonly build_time="2026-10-06T00:00:00Z"
release_dir="$(mktemp -d /tmp/chaoschain-release-test.XXXXXX)"
readonly release_dir

cleanup() {
	rm -rf "$release_dir"
}
trap cleanup EXIT

VERSION="$version" COMMIT="$commit" BUILD_TIME="$build_time" scripts/build-release.sh "$release_dir" >/dev/null
(
	cd "$release_dir"
	sha256sum --check SHA256SUMS
)
archive="$(find "$release_dir" -maxdepth 1 -type f -name '*.tar.gz' -print -quit)"
tar -xzf "$archive" -C "$release_dir"
bundle="$(find "$release_dir" -mindepth 1 -maxdepth 1 -type d -name 'chaoschain-*' -print -quit)"
reported="$("${bundle}/chaoschaind" version --output json)"
jq -e --arg version "$version" --arg commit "$commit" --arg build_time "$build_time" '
  .version == $version and .commit == $commit and .build_time == $build_time
' <<<"$reported" >/dev/null
jq -e --argjson reported "$reported" '. == $reported' "${bundle}/build-info.json" >/dev/null

echo "release archive checksum and embedded build identity are valid"
