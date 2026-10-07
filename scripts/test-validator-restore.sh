#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_dir="$(mktemp -d /tmp/chaoschain-validator-restore.XXXXXX)"
trap 'rm -rf "$test_dir"' EXIT

source_home="${test_dir}/node1"
restored_home="${test_dir}/restored-node1"
archive="${test_dir}/node1.tar.gz"
mkdir -p "${source_home}/config" "${source_home}/data"
printf '%s\n' 'chain identity' >"${source_home}/config/genesis.json"
printf '%s\n' 'committed state' >"${source_home}/data/state.db"
"${repo_root}/scripts/backup-validator-home.sh" "$source_home" "$archive" --confirm-stopped >/dev/null

if "${repo_root}/scripts/restore-validator-home.sh" "$archive" "$restored_home" >/dev/null 2>&1; then
	echo "restore was allowed without stopped-daemon confirmation" >&2
	exit 1
fi

"${repo_root}/scripts/restore-validator-home.sh" "$archive" "$restored_home" --confirm-stopped
cmp "${source_home}/config/genesis.json" "${restored_home}/config/genesis.json"
cmp "${source_home}/data/state.db" "${restored_home}/data/state.db"

if "${repo_root}/scripts/restore-validator-home.sh" "$archive" "$restored_home" --confirm-stopped >/dev/null 2>&1; then
	echo "restore was allowed into an occupied validator home" >&2
	exit 1
fi

printf '%064d  %s\n' 0 'node1.tar.gz' >"${archive}.sha256"
if "${repo_root}/scripts/restore-validator-home.sh" "$archive" "${test_dir}/tampered" --confirm-stopped >/dev/null 2>&1; then
	echo "restore accepted a mismatched checksum" >&2
	exit 1
fi
if [[ -e "${test_dir}/tampered" ]]; then
	echo "failed restore left a partial validator home" >&2
	exit 1
fi

echo "validator restore tests passed"
