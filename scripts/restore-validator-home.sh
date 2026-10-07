#!/usr/bin/env bash
set -euo pipefail
umask 077

usage() {
	echo "Usage: $0 <archive.tar.gz> <validator-home> --confirm-stopped" >&2
}

if [[ "$#" -ne 3 || "$3" != "--confirm-stopped" ]]; then
	usage
	exit 2
fi

archive_input="$1"
target_input="$2"
if [[ ! -f "$archive_input" ]]; then
	echo "validator archive is not a file: ${archive_input}" >&2
	exit 2
fi

archive_parent="$(cd "$(dirname -- "$archive_input")" && pwd -P)"
archive_name="$(basename -- "$archive_input")"
archive_path="${archive_parent}/${archive_name}"
checksum_path="${archive_path}.sha256"
if [[ ! -f "$checksum_path" ]]; then
	echo "validator archive checksum is missing: ${checksum_path}" >&2
	exit 2
fi

target_parent_input="$(dirname -- "$target_input")"
target_name="$(basename -- "$target_input")"
if [[ ! -d "$target_parent_input" || "$target_name" == "." || "$target_name" == ".." ]]; then
	echo "validator home parent must exist and target must name a new directory" >&2
	exit 2
fi
target_parent="$(cd "$target_parent_input" && pwd -P)"
target_path="${target_parent}/${target_name}"
if [[ -e "$target_path" || -L "$target_path" ]]; then
	echo "refusing to restore into an occupied validator home: ${target_path}" >&2
	exit 1
fi

actual_sha256="$(sha256sum "$archive_path" | cut -d ' ' -f 1)"
expected_line="${actual_sha256}  ${archive_name}"
if [[ "$(<"$checksum_path")" != "$expected_line" ]]; then
	echo "validator archive checksum does not match: ${archive_path}" >&2
	exit 1
fi

restore_tmp="$(mktemp -d "${target_parent}/.chaoschain-restore.XXXXXX")"
cleanup() {
	if [[ -d "$restore_tmp" ]]; then
		rm -rf -- "$restore_tmp"
	fi
}
trap cleanup EXIT

tar -xzf "$archive_path" --no-same-owner -C "$restore_tmp"
mv -T -n -- "$restore_tmp" "$target_path"
if [[ -d "$restore_tmp" ]]; then
	echo "validator home appeared during restore: ${target_path}" >&2
	exit 1
fi

echo "validator home restored at ${target_path}"
