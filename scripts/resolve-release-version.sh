#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 2 ]]; then
	echo "Usage: $0 <git-ref> <commit-sha>" >&2
	exit 2
fi

git_ref="$1"
commit_sha="$2"
if [[ ! "$commit_sha" =~ ^[0-9a-f]{40}$ ]]; then
	echo "commit SHA must be a 40-character lowercase hexadecimal value" >&2
	exit 2
fi

if [[ "$git_ref" == refs/tags/* ]]; then
	tag="${git_ref#refs/tags/}"
	if [[ ! "$tag" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
		echo "release tags must use the vMAJOR.MINOR.PATCH format" >&2
		exit 1
	fi
	printf '%s\n' "$tag"
	exit 0
fi

printf 'ci-%s\n' "${commit_sha:0:12}"
