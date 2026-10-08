#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
test_root="$(mktemp -d /tmp/chaoschain-explorer-config-test.XXXXXX)"
trap 'rm -rf "$test_root"' EXIT

new_home() {
	local home="$1"
	mkdir -p "${home}/config"
	cat >"${home}/config/app.toml" <<'EOF'
pruning = "default"
min-retain-blocks = 10

[api]
enable = false
swagger = true
address = "tcp://localhost:1317"
max-open-connections = 1000
enabled-unsafe-cors = true

[grpc]
enable = true
EOF
	cat >"${home}/config/config.toml" <<'EOF'
[rpc]
laddr = "tcp://127.0.0.1:26657"

[tx_index]
indexer = "null"
EOF
}

home="${test_root}/node"
new_home "$home"
"${script_dir}/enable-explorer-api.sh" "$home" 1417 >/dev/null

grep -Eq '^pruning = "nothing"$' "${home}/config/app.toml"
grep -Eq '^min-retain-blocks = 0$' "${home}/config/app.toml"
grep -Eq '^enable = true$' "${home}/config/app.toml"
grep -Eq '^swagger = false$' "${home}/config/app.toml"
grep -Eq '^address = "tcp://127.0.0.1:1417"$' "${home}/config/app.toml"
grep -Eq '^max-open-connections = 500$' "${home}/config/app.toml"
grep -Eq '^enabled-unsafe-cors = false$' "${home}/config/app.toml"
grep -Eq '^indexer = "kv"$' "${home}/config/config.toml"
[[ "$(find "${home}/config" -name 'app.toml.bak.*' | wc -l)" -eq 1 ]]
[[ "$(find "${home}/config" -name 'config.toml.bak.*' | wc -l)" -eq 1 ]]

broken_home="${test_root}/broken"
new_home "$broken_home"
sed -i '/^enabled-unsafe-cors =/d' "${broken_home}/config/app.toml"
app_before="$(sha256sum "${broken_home}/config/app.toml")"
comet_before="$(sha256sum "${broken_home}/config/config.toml")"
if "${script_dir}/enable-explorer-api.sh" "$broken_home" >/dev/null 2>&1; then
	echo "Explorer configuration accepted a missing required key" >&2
	exit 1
fi
[[ "$(sha256sum "${broken_home}/config/app.toml")" == "$app_before" ]]
[[ "$(sha256sum "${broken_home}/config/config.toml")" == "$comet_before" ]]
if find "${broken_home}/config" -name '*.bak.*' | grep -q .; then
	echo "Failed configuration left backups for an unmodified node" >&2
	exit 1
fi

echo "Explorer node configuration tests passed"
