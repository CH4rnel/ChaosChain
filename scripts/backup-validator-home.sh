#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "Usage: $0 <validator-home> <archive.tar.gz> --confirm-stopped" >&2
}

if [[ "$#" -ne 3 || "$3" != "--confirm-stopped" ]]; then
	usage
	exit 2
fi

home_input="$1"
archive_input="$2"
if [[ ! -d "$home_input" ]]; then
	echo "validator home is not a directory: ${home_input}" >&2
	exit 2
fi
home="$(cd "$home_input" && pwd -P)"

archive_parent_input="$(dirname -- "$archive_input")"
archive_name="$(basename -- "$archive_input")"
if [[ "$archive_name" != *.tar.gz || "$archive_name" == ".tar.gz" ]]; then
	echo "archive path must end in .tar.gz" >&2
	exit 2
fi
if [[ ! -d "$archive_parent_input" ]]; then
	echo "archive parent directory does not exist: ${archive_parent_input}" >&2
	exit 2
fi
archive_parent="$(cd "$archive_parent_input" && pwd -P)"
case "${archive_parent}/" in
	"${home}/"*)
		echo "archive destination must be outside the validator home" >&2
		exit 2
		;;
esac

archive_path="${archive_parent}/${archive_name}"
checksum_name="${archive_name}.sha256"
checksum_path="${archive_parent}/${checksum_name}"
if [[ -e "$archive_path" || -L "$archive_path" || -e "$checksum_path" || -L "$checksum_path" ]]; then
	echo "refusing to overwrite an existing archive or checksum" >&2
	exit 1
fi

archive_tmp="$(mktemp "${archive_parent}/.chaoschain-backup.XXXXXX")"
checksum_tmp="$(mktemp "${archive_parent}/.chaoschain-checksum.XXXXXX")"
created_archive=false
complete=false
cleanup() {
	rm -f -- "$archive_tmp" "$checksum_tmp"
	if [[ "$created_archive" == true && "$complete" != true ]]; then
		rm -f -- "$archive_path" "$checksum_path"
	fi
}
trap cleanup EXIT

tar -czf "$archive_tmp" -C "$home" .
archive_sha256="$(sha256sum "$archive_tmp" | cut -d ' ' -f 1)"
printf '%s  %s\n' "$archive_sha256" "$archive_name" >"$checksum_tmp"

if ! ln -- "$archive_tmp" "$archive_path"; then
	echo "could not create archive without overwriting: ${archive_path}" >&2
	exit 1
fi
created_archive=true
if ! ln -- "$checksum_tmp" "$checksum_path"; then
	echo "could not create checksum without overwriting: ${checksum_path}" >&2
	exit 1
fi
complete=true

echo "validator home archived at ${archive_path}"
echo "checksum recorded at ${checksum_path}"
