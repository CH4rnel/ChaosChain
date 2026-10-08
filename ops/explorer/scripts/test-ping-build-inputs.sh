#!/usr/bin/env bash
set -euo pipefail

explorer_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
profile="$(cd "${explorer_root}/../.." && pwd)/config/devnet.json"
rendered="$(mktemp /tmp/chaoschain-ping-config.XXXXXX)"
trap 'rm -f "$rendered"' EXIT

PUBLIC_URL=https://testnet-explorer.example.org
CHAIN_ID="$(jq -er '.chain_id' "$profile")"
export PUBLIC_URL CHAIN_ID
# Keep the variable names literal so envsubst expands only the chain template fields.
# shellcheck disable=SC2016
substitutions='${PUBLIC_URL} ${CHAIN_ID}'
envsubst "$substitutions" <"${explorer_root}/chains/chaoschain.json.template" >"$rendered"

jq -e \
	--arg chain_id "$CHAIN_ID" \
	--arg base_denom "$(jq -er '.token.base_denom' "$profile")" \
	--arg display_denom "$(jq -er '.token.display_denom' "$profile")" \
	--arg decimals "$(jq -er '.token.decimals | tostring' "$profile")" '
  .chain_name == "chaoschain-testnet" and
  .chain_id == $chain_id and
  .network_type == "testnet" and
  .sdk_version == "0.54.0" and
  .addr_prefix == "chaos" and
  .api == [{"address":"https://testnet-explorer.example.org/api","provider":"ChaosChain"}] and
  .rpc == [{"address":"https://testnet-explorer.example.org/rpc","provider":"ChaosChain"}] and
  .assets == [{"base":$base_denom,"symbol":$display_denom,"exponent":$decimals,"coingecko_id":"","logo":""}] and
  (.features | index("staking") != null) and
  (.features | index("governance") != null) and
  (.features | index("ibc") == null)
' "$rendered" >/dev/null

rg -q '^ARG PING_REF$' "${explorer_root}/Dockerfile"
rg -q 'git fetch --depth 1 origin "\$\{PING_REF\}"' "${explorer_root}/Dockerfile"
if rg -q 'go (get|install)|go\.mod' "$explorer_root"; then
	echo "Explorer build inputs entered the Go dependency graph" >&2
	exit 1
fi

echo "Ping build input tests passed"
