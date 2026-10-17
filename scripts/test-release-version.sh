#!/usr/bin/env bash
set -euo pipefail

resolver="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/resolve-release-version.sh"
readonly resolver
readonly commit="0123456789abcdef0123456789abcdef01234567"

assert_version() {
	local ref="$1" expected="$2" actual
	actual="$("$resolver" "$ref" "$commit")"
	if [[ "$actual" != "$expected" ]]; then
		echo "unexpected version for ${ref}: expected ${expected}, got ${actual}" >&2
		exit 1
	fi
}

assert_rejected() {
	local ref="$1"
	if "$resolver" "$ref" "$commit" >/dev/null 2>&1; then
		echo "invalid ref was accepted: ${ref}" >&2
		exit 1
	fi
}

assert_version refs/tags/v1.2.3 v1.2.3
assert_version refs/heads/master ci-0123456789ab
assert_version refs/pull/42/merge ci-0123456789ab
assert_rejected refs/tags/v1.2
assert_rejected refs/tags/v01.2.3
assert_rejected refs/tags/v1.2.3-rc.1

if "$resolver" refs/tags/v1.2.3 invalid-sha >/dev/null 2>&1; then
	echo "invalid commit SHA was accepted" >&2
	exit 1
fi

echo "release version resolution tests passed"
