#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 2 || ! "$2" =~ ^[1-9][0-9]*$ ]]; then
	echo "Usage: $0 <verified-bootstrap-directory> <minimum-reachable-peers>" >&2
	exit 2
fi

bootstrap_dir="$(realpath "$1")"
minimum_reachable="$2"
for command in jq sha256sum getent timeout; do
	command -v "$command" >/dev/null || { echo "required command not found: ${command}" >&2; exit 1; }
done
[[ -f "${bootstrap_dir}/SHA256SUMS" && -f "${bootstrap_dir}/bootstrap.json" && -f "${bootstrap_dir}/genesis.json" ]] || {
	echo "public bootstrap directory is incomplete" >&2
	exit 1
}
(cd "$bootstrap_dir" && sha256sum --check --strict SHA256SUMS)

chain_id="$(jq -er '.chain_id | select(type == "string" and length > 0)' "${bootstrap_dir}/bootstrap.json")"
expected_genesis_sha="$(jq -er '.genesis_sha256 | select(type == "string" and test("^[0-9a-f]{64}$"))' "${bootstrap_dir}/bootstrap.json")"
persistent_peers="$(jq -er '.persistent_peers | select(type == "string" and length > 0)' "${bootstrap_dir}/bootstrap.json")"
[[ "$(sha256sum "${bootstrap_dir}/genesis.json" | cut -d ' ' -f 1)" == "$expected_genesis_sha" ]] || {
	echo "bootstrap manifest does not match genesis" >&2
	exit 1
}
[[ "$(jq -er '.chain_id' "${bootstrap_dir}/genesis.json")" == "$chain_id" ]] || {
	echo "bootstrap chain ID does not match genesis" >&2
	exit 1
}

IFS=',' read -r -a peers <<<"$persistent_peers"
if ((minimum_reachable > ${#peers[@]})); then
	echo "minimum reachable peers exceeds the ${#peers[@]} advertised peers" >&2
	exit 2
fi

reachable=0
for peer in "${peers[@]}"; do
	if [[ ! "$peer" =~ ^[0-9a-f]{40}@([A-Za-z0-9][A-Za-z0-9.-]{0,252}):([0-9]{1,5})$ ]]; then
		echo "invalid advertised peer: ${peer}" >&2
		exit 1
	fi
	peer_host="${BASH_REMATCH[1]}"
	peer_port="${BASH_REMATCH[2]}"
	peer_port_number=$((10#$peer_port))
	if ((peer_port_number < 1 || peer_port_number > 65535)); then
		echo "invalid P2P port in advertised peer: ${peer}" >&2
		exit 1
	fi
	if ! getent ahostsv4 "$peer_host" >/dev/null 2>&1; then
		echo "unreachable ${peer_host}:${peer_port} (DNS lookup failed)"
		continue
	fi
	if timeout 3 bash -c "exec 3<>/dev/tcp/${peer_host}/${peer_port}" 2>/dev/null; then
		((reachable += 1))
		printf 'reachable %s:%s\n' "$peer_host" "$peer_port"
	else
		printf 'unreachable %s:%s (TCP connection failed)\n' "$peer_host" "$peer_port"
	fi
done

printf '%s of %s advertised peers reachable; minimum required: %s\n' "$reachable" "${#peers[@]}" "$minimum_reachable"
if ((reachable < minimum_reachable)); then
	echo "public peer reachability is below the required minimum" >&2
	exit 1
fi
