#!/usr/bin/env bash
set -euo pipefail

readonly govulncheck_version="v1.8.0"
readonly allowed_vulnerability="GO-2026-5932"
report="$(mktemp)"
trap 'rm -f "$report"' EXIT

# Reviewed 2026-10-02: GO-2026-5932 is an unfixable package-level advisory for
# the deprecated OpenPGP package. The CI traces reviewed for this exception are
# limited to armor.Encode/Decode and armor line-reader/writer helpers used by
# Cosmos SDK key import/export; no OpenPGP packet parser or cryptographic
# operation (such as ReadMessage, packet, elgamal, or s2k) appears in the traces.
# Revisit by 2026-12-31 or earlier if Cosmos SDK removes its armor dependency.
go run "golang.org/x/vuln/cmd/govulncheck@${govulncheck_version}" -json ./... >"$report"

if ! jq -s -e '
  length > 0
  and .[0].config.protocol_version == "v1.0.0"
  and all(.[]; (keys | length) == 1)
  and all(.[]; if has("finding") then
    (.finding.osv | type == "string" and length > 0)
    and (.finding.trace | type == "array" and length > 0)
    else true end)
' "$report" >/dev/null; then
	echo "govulncheck emitted invalid or unsupported JSON output" >&2
	exit 1
fi

reachable="$(jq -r 'select(.finding and (.finding.trace[0].function // "") != "") | .finding.osv' "$report" | sort -u)"
if [[ -z "$reachable" ]]; then
	echo "govulncheck found no reachable vulnerable symbols"
	exit 0
fi

unexpected="$(jq -r --arg allowed "$allowed_vulnerability" '
  select(.finding and (.finding.trace[0].function // "") != "")
  | select(.finding.osv != $allowed or
      (.finding.trace[0].package != "golang.org/x/crypto/openpgp/armor" and
       .finding.trace[0].package != "golang.org/x/crypto/openpgp/errors"))
  | [.finding.osv, .finding.trace[0].package] | @tsv
' "$report" | sort -u)"
if [[ -n "$unexpected" ]]; then
	echo "govulncheck found reachable symbols outside the reviewed exception:" >&2
	printf '%s\n' "$unexpected" >&2
	exit 1
fi

echo "govulncheck found only reviewed reachable symbols for ${allowed_vulnerability}"
