#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -lt 1 || "$#" -gt 2 ]]; then
	echo "Usage: $0 <node-home> [api-port]" >&2
	exit 2
fi

node_home_input="$1"
api_port="${2:-1317}"
if [[ ! "$api_port" =~ ^[0-9]+$ ]] || ((api_port < 1 || api_port > 65535)); then
	echo "API port must be an integer from 1 through 65535" >&2
	exit 2
fi
if [[ ! -d "$node_home_input" ]]; then
	echo "Node home does not exist: ${node_home_input}" >&2
	exit 2
fi

node_home="$(cd "$node_home_input" && pwd -P)"
app_config="${node_home}/config/app.toml"
comet_config="${node_home}/config/config.toml"
if [[ ! -f "$app_config" || ! -f "$comet_config" ]]; then
	echo "Expected app.toml and config.toml under ${node_home}/config" >&2
	exit 2
fi

if pgrep -af chaoschaind 2>/dev/null | grep -F -- "--home ${node_home}" >/dev/null; then
	echo "Stop the node using ${node_home} before changing its configuration" >&2
	exit 1
fi

app_staging="$(mktemp "${app_config}.staging.XXXXXX")"
comet_staging="$(mktemp "${comet_config}.staging.XXXXXX")"
cleanup() {
	rm -f -- "$app_staging" "$comet_staging"
}
trap cleanup EXIT
cp -p -- "$app_config" "$app_staging"
cp -p -- "$comet_config" "$comet_staging"

set_toml() {
	local file="$1"
	local section="$2"
	local key="$3"
	local value="$4"
	local output
	output="$(mktemp "${file}.rewrite.XXXXXX")"
	if ! awk -v wanted_section="$section" -v wanted_key="$key" -v wanted_value="$value" '
		BEGIN { section = ""; changed = 0 }
		/^[[:space:]]*\[[^]]+\][[:space:]]*$/ {
			section = $0
			gsub(/^[[:space:]]*\[|\][[:space:]]*$/, "", section)
		}
		section == wanted_section && !changed && $0 ~ "^[[:space:]]*" wanted_key "[[:space:]]*=" {
			print wanted_key " = " wanted_value
			changed = 1
			next
		}
		{ print }
		END { if (!changed) exit 3 }
	' "$file" >"$output"; then
		rm -f -- "$output"
		echo "Required TOML key is missing: [${section}] ${key} in ${file}" >&2
		exit 1
	fi
	mv -- "$output" "$file"
}

set_toml "$app_staging" "" pruning '"nothing"'
set_toml "$app_staging" "" min-retain-blocks 0
set_toml "$app_staging" api enable true
set_toml "$app_staging" api swagger false
set_toml "$app_staging" api address "\"tcp://127.0.0.1:${api_port}\""
set_toml "$app_staging" api enabled-unsafe-cors false
set_toml "$app_staging" api max-open-connections 500
set_toml "$comet_staging" tx_index indexer '"kv"'

stamp="$(date -u +%Y%m%dT%H%M%SZ)"
app_backup="${app_config}.bak.${stamp}"
comet_backup="${comet_config}.bak.${stamp}"
if [[ -e "$app_backup" || -e "$comet_backup" ]]; then
	echo "Backup name collision at timestamp ${stamp}" >&2
	exit 1
fi
cp -p -- "$app_config" "$app_backup"
cp -p -- "$comet_config" "$comet_backup"
mv -- "$app_staging" "$app_config"
mv -- "$comet_staging" "$comet_config"

echo "Explorer node configuration updated; restart with chaoschaind start, not start-native"
echo "Backups: ${app_backup} ${comet_backup}"
