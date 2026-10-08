#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 4 ]]; then
	echo "Usage: $0 <release-archive> <release-SHA256SUMS> <public-bootstrap-directory> <moniker>" >&2
	exit 2
fi

release_archive="$(realpath "$1")"
release_sums="$(realpath "$2")"
bootstrap_dir="$(realpath "$3")"
moniker="$4"
if [[ ! "$moniker" =~ ^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$ ]]; then
	echo "moniker must contain only letters, numbers, dots, underscores, and hyphens" >&2
	exit 2
fi
for command in jq sha256sum tar install; do
	command -v "$command" >/dev/null || { echo "required command not found: ${command}" >&2; exit 1; }
done
if [[ -z "${CHAOSCHAIN_TEST_ROOT:-}" && "$EUID" -ne 0 ]]; then
	echo "run this installer as root" >&2
	exit 1
fi

target_root="${CHAOSCHAIN_TEST_ROOT:-}"
binary_path="${target_root}/usr/local/bin/chaoschaind"
node_home="${target_root}/var/lib/chaoschain"
service_path="${target_root}/etc/systemd/system/chaoschaind.service"
if [[ -e "$node_home" || -L "$node_home" ]]; then
	echo "refusing to overwrite existing validator home: ${node_home}" >&2
	exit 1
fi
[[ -f "$release_archive" && -f "$release_sums" ]] || { echo "release archive or checksum file is missing" >&2; exit 1; }
[[ -f "${bootstrap_dir}/SHA256SUMS" && -f "${bootstrap_dir}/bootstrap.json" && -f "${bootstrap_dir}/genesis.json" ]] || {
	echo "public bootstrap directory is incomplete" >&2
	exit 1
}
(cd "$(dirname "$release_sums")" && sha256sum --check --strict "$(basename "$release_sums")")
(cd "$bootstrap_dir" && sha256sum --check --strict SHA256SUMS)

chain_id="$(jq -er '.chain_id | select(type == "string" and length > 0)' "${bootstrap_dir}/bootstrap.json")"
minimum_gas_price="$(jq -er '.minimum_gas_price | select(type == "string" and test("^[0-9]+([.][0-9]+)?[A-Za-z][A-Za-z0-9/:._-]*$"))' "${bootstrap_dir}/bootstrap.json")"
persistent_peers="$(jq -er '.persistent_peers | select(type == "string" and length > 0)' "${bootstrap_dir}/bootstrap.json")"
genesis_sha="$(jq -er '.genesis_sha256 | select(type == "string" and test("^[0-9a-f]{64}$"))' "${bootstrap_dir}/bootstrap.json")"
[[ "$(sha256sum "${bootstrap_dir}/genesis.json" | cut -d ' ' -f 1)" == "$genesis_sha" ]] || {
	echo "bootstrap manifest does not match genesis" >&2
	exit 1
}
[[ "$(jq -er '.chain_id' "${bootstrap_dir}/genesis.json")" == "$chain_id" ]] || {
	echo "bootstrap chain ID does not match genesis" >&2
	exit 1
}

staging="$(mktemp -d)"
cleanup() { rm -rf -- "$staging"; }
trap cleanup EXIT
tar -tzf "$release_archive" >"${staging}/archive-files"
if ! awk 'NF && ($0 ~ /^\// || $0 ~ /(^|\/)\.\.($|\/)/) { bad=1 } END { exit bad }' "${staging}/archive-files"; then
	echo "release archive contains an unsafe path" >&2
	exit 1
fi
mapfile -t roots < <(awk -F/ 'NF { print $1 }' "${staging}/archive-files" | sort -u)
[[ "${#roots[@]}" -eq 1 && "${roots[0]}" =~ ^chaoschain-[A-Za-z0-9._-]+-linux-amd64$ ]] || {
	echo "release archive must contain one versioned Linux AMD64 directory" >&2
	exit 1
}
tar -xzf "$release_archive" -C "$staging"
release_dir="${staging}/${roots[0]}"
[[ -x "${release_dir}/chaoschaind" && -f "${release_dir}/build-info.json" ]] || {
	echo "release archive is missing its daemon or build metadata" >&2
	exit 1
}
"${release_dir}/chaoschaind" version --output json >/dev/null

install -D -m 0755 "${release_dir}/chaoschaind" "$binary_path"
mkdir -p "$(dirname "$node_home")" "$(dirname "$service_path")"
if [[ -n "$target_root" ]]; then
	"$binary_path" init "$moniker" --chain-id "$chain_id" --home "$node_home"
else
	if ! id chaoschain >/dev/null 2>&1; then
		useradd --system --user-group --home-dir /var/lib/chaoschain --create-home --shell /usr/sbin/nologin chaoschain
	fi
	mkdir -p "$node_home"
	chown chaoschain:chaoschain "$(dirname "$node_home")"
	runuser -u chaoschain -- "$binary_path" init "$moniker" --chain-id "$chain_id" --home "$node_home"
fi
install -m 0644 "${bootstrap_dir}/genesis.json" "${node_home}/config/genesis.json"
"$binary_path" genesis validate "${node_home}/config/genesis.json" --home "$node_home"
sed -i -E "s|^[[:space:]]*persistent_peers[[:space:]]*=.*|persistent_peers = \"${persistent_peers}\"|" "${node_home}/config/config.toml"
sed -i -E "s|^[[:space:]]*minimum-gas-prices[[:space:]]*=.*|minimum-gas-prices = \"${minimum_gas_price}\"|" "${node_home}/config/app.toml"
install -m 0644 "$(dirname "${BASH_SOURCE[0]}")/../chaoschaind.service" "$service_path"
if [[ -z "$target_root" ]]; then
	chown -R chaoschain:chaoschain "$node_home"
else
	chmod -R go-rwx "${node_home}/config"
fi
systemctl daemon-reload
systemctl enable --now chaoschaind.service

echo "validator installed for chain ${chain_id}; its private validator key stays on this host"
