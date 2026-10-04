#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly repo_root
readonly profile="${CHAOSCHAIN_NETWORK_PROFILE:-${repo_root}/config/devnet.json}"
readonly binary="${CHAOSCHAIN_BINARY:-chaoschaind}"
readonly node_home="${1:?usage: scripts/init-local-devnet.sh NODE_HOME}"

if [[ -e "$node_home" ]]; then
	echo "refusing to overwrite existing node home: ${node_home}" >&2
	exit 1
fi

"${repo_root}/scripts/check-network-profile.sh" "$profile" >/dev/null

chain_id="$(jq -er '.chain_id' "$profile")"
moniker="$(jq -er '.moniker' "$profile")"
base_denom="$(jq -er '.token.base_denom' "$profile")"
display_denom="$(jq -er '.token.display_denom' "$profile")"
decimals="$(jq -er '.token.decimals' "$profile")"
unbonding_time="$(jq -er '.staking.unbonding_time' "$profile")"
validator_account="$(jq -er '.staking.validator_account' "$profile")"
self_delegation="$(jq -er '.staking.self_delegation' "$profile")"
minimum_deposit="$(jq -er '.governance.minimum_deposit' "$profile")"
minimum_gas_price="$(jq -er '.minimum_gas_price' "$profile")"

"$binary" init "$moniker" --chain-id "$chain_id" --home "$node_home" >/dev/null 2>&1

while IFS= read -r account; do
	name="$(jq -er '.name' <<<"$account")"
	balance="$(jq -er '.balance' <<<"$account")"
	"$binary" keys add "$name" --keyring-backend test --home "$node_home" >/dev/null 2>&1
	"$binary" genesis add-genesis-account "$name" "${balance}${base_denom}" --keyring-backend test --home "$node_home" >/dev/null 2>&1
done < <(jq -c '.accounts[]' "$profile")

genesis="${node_home}/config/genesis.json"
updated_genesis="$(mktemp "${node_home}/config/genesis.json.XXXXXX")"
jq \
	--arg base "$base_denom" \
	--arg display "$display_denom" \
	--argjson decimals "$decimals" \
	--arg unbonding "$unbonding_time" \
	--arg minimum_deposit "$minimum_deposit" '
  .app_state.staking.params.bond_denom = $base |
  .app_state.staking.params.unbonding_time = $unbonding |
  .app_state.mint.params.mint_denom = $base |
  .app_state.gov.params.min_deposit[0] = {denom: $base, amount: $minimum_deposit} |
  .app_state.gov.params.expedited_min_deposit[0].denom = $base |
  .app_state.bank.denom_metadata = [{
    description: "The native token of ChaosChain",
    denom_units: [
      {denom: $base, exponent: 0, aliases: []},
      {denom: $display, exponent: $decimals, aliases: []}
    ],
    base: $base,
    display: $display,
    name: $display,
    symbol: $display,
    uri: "",
    uri_hash: ""
  }]
' "$genesis" >"$updated_genesis"
mv "$updated_genesis" "$genesis"

sed -i -E "s|^minimum-gas-prices = .*|minimum-gas-prices = \"${minimum_gas_price}${base_denom}\"|" "${node_home}/config/app.toml"

"$binary" genesis gentx "$validator_account" "${self_delegation}${base_denom}" --chain-id "$chain_id" --keyring-backend test --home "$node_home" >/dev/null 2>&1
"$binary" genesis collect-gentxs --home "$node_home" >/dev/null 2>&1
"$binary" genesis validate "$genesis" --home "$node_home" >/dev/null 2>&1

echo "initialized ${chain_id} at ${node_home}"
