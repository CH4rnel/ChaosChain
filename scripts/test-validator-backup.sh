#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_dir="$(mktemp -d /tmp/chaoschain-validator-backup.XXXXXX)"
trap 'rm -rf "$test_dir"' EXIT

home="${test_dir}/node1"
archive="${test_dir}/backups/node1.tar.gz"
mkdir -p "${home}/config" "${home}/data" "${test_dir}/backups"
printf '%s\n' 'genesis fixture' >"${home}/config/genesis.json"
printf '%s\n' 'state fixture' >"${home}/data/state.db"

if "${repo_root}/scripts/backup-validator-home.sh" "$home" "$archive" >/dev/null 2>&1; then
	echo "backup was allowed without stopped-daemon confirmation" >&2
	exit 1
fi

"${repo_root}/scripts/backup-validator-home.sh" "$home" "$archive" --confirm-stopped
(
	cd "${test_dir}/backups"
	sha256sum -c node1.tar.gz.sha256
)

mkdir "${test_dir}/restored"
tar -xzf "$archive" -C "${test_dir}/restored"
cmp "${home}/config/genesis.json" "${test_dir}/restored/config/genesis.json"
cmp "${home}/data/state.db" "${test_dir}/restored/data/state.db"

if "${repo_root}/scripts/backup-validator-home.sh" "$home" "$archive" --confirm-stopped >/dev/null 2>&1; then
	echo "backup overwrote an existing archive" >&2
	exit 1
fi

if "${repo_root}/scripts/backup-validator-home.sh" "$home" "${home}/nested.tar.gz" --confirm-stopped >/dev/null 2>&1; then
	echo "backup was allowed inside the validator home" >&2
	exit 1
fi

echo "validator backup tests passed"
