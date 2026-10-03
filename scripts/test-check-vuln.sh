#!/usr/bin/env bash
set -euo pipefail

go() {
	printf '%s\n' "$VULN_TEST_STREAM"
	return "${VULN_TEST_SCANNER_STATUS:-0}"
}
export -f go

expect_status() {
	local name="$1"
	local expected="$2"
	local stream="$3"
	local scanner_status="${4:-0}"
	local actual
	local output

	export VULN_TEST_STREAM="$stream"
	export VULN_TEST_SCANNER_STATUS="$scanner_status"
	if output="$(bash scripts/check-vuln.sh 2>&1)"; then
		actual=0
	else
		actual=$?
	fi

	if [[ "$actual" != "$expected" ]]; then
		printf 'case %s: expected exit %s, got %s\n%s\n' "$name" "$expected" "$actual" "$output" >&2
		exit 1
	fi
	printf 'case %s: passed\n' "$name"
}

config='{"config":{"protocol_version":"v1.0.0"}}'
reviewed='{"finding":{"osv":"GO-2026-5932","trace":[{"module":"golang.org/x/crypto","package":"golang.org/x/crypto/openpgp/armor","function":"Encode"}]}}'
module_only='{"finding":{"osv":"GO-2099-0001","trace":[{"module":"example.com/vulnerable"}]}}'
new_symbol='{"finding":{"osv":"GO-2099-0001","trace":[{"module":"example.com/vulnerable","package":"example.com/vulnerable","function":"Read"}]}}'
unreviewed_pgp='{"finding":{"osv":"GO-2026-5932","trace":[{"module":"golang.org/x/crypto","package":"golang.org/x/crypto/openpgp/packet","function":"Read"}]}}'

expect_status "reviewed armor symbol" 0 "$config
$reviewed"
expect_status "unreachable advisory" 0 "$config
$module_only"
expect_status "new reachable advisory" 1 "$config
$reviewed
$new_symbol"
expect_status "unreviewed OpenPGP package" 1 "$config
$unreviewed_pgp"
expect_status "malformed JSON" 1 'not-json'
expect_status "scanner failure" 1 "$config
$reviewed" 1
