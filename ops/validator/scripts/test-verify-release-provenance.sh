#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
shared_verifier="${repo_root}/scripts/verify-release-provenance.sh"
test_root="$(mktemp -d /tmp/chaoschain-release-attestation-test.XXXXXX)"
trap 'rm -rf "$test_root"' EXIT

mkdir -p "${test_root}/bin"
touch "${test_root}/chaoschaind.tar.gz"
cat >"${test_root}/bin/gh" <<'GH'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >"${GH_VERIFY_ARGS_FILE}"
exit "${GH_VERIFY_STATUS:-0}"
GH
chmod +x "${test_root}/bin/gh"
export PATH="${test_root}/bin:${PATH}"
export GH_VERIFY_ARGS_FILE="${test_root}/gh-args"

"${repo_root}/ops/validator/scripts/verify-release-provenance.sh" "${test_root}/chaoschaind.tar.gz"
grep -Fq 'attestation verify ' "${GH_VERIFY_ARGS_FILE}"
grep -Fq -- '--repo CH4rnel/ChaosChain' "${GH_VERIFY_ARGS_FILE}"
grep -Fq -- '--signer-workflow CH4rnel/ChaosChain/.github/workflows/ci.yml' "${GH_VERIFY_ARGS_FILE}"
grep -Fq "${test_root}/chaoschaind.tar.gz" "${GH_VERIFY_ARGS_FILE}"

"$shared_verifier" "${test_root}/chaoschaind.tar.gz"
grep -Fq 'attestation verify ' "${GH_VERIFY_ARGS_FILE}"
grep -Fq -- '--repo CH4rnel/ChaosChain' "${GH_VERIFY_ARGS_FILE}"
grep -Fq -- '--signer-workflow CH4rnel/ChaosChain/.github/workflows/ci.yml' "${GH_VERIFY_ARGS_FILE}"
grep -Fq "${test_root}/chaoschaind.tar.gz" "${GH_VERIFY_ARGS_FILE}"

export GH_VERIFY_STATUS=1
if "${repo_root}/ops/validator/scripts/verify-release-provenance.sh" "${test_root}/chaoschaind.tar.gz" >/dev/null 2>&1; then
	echo "release verifier accepted an artifact without trusted provenance" >&2
	exit 1
fi

if "${repo_root}/ops/validator/scripts/verify-release-provenance.sh" "${test_root}/missing.tar.gz" >/dev/null 2>&1; then
	echo "release verifier accepted a missing artifact" >&2
	exit 1
fi
if "$shared_verifier" "${test_root}/missing.tar.gz" >/dev/null 2>&1; then
	echo "shared release verifier accepted a missing artifact" >&2
	exit 1
fi

echo "release provenance verification tests passed"
