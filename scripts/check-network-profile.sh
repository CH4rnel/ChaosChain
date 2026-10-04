#!/usr/bin/env bash
set -euo pipefail

readonly profile="${1:-config/devnet.json}"

jq -e '
  . as $profile |
  (.chain_id | type == "string" and test("^[a-z][a-z0-9-]{2,49}$")) and
  (.moniker | type == "string" and length > 0) and
  (.token.base_denom | type == "string" and test("^[a-z][a-z0-9]{2,127}$")) and
  (.token.display_denom | type == "string" and test("^[A-Z][A-Z0-9]{1,11}$")) and
  (.token.decimals | type == "number" and . >= 0 and . <= 18 and floor == .) and
  (.staking.unbonding_time | type == "string" and test("^[1-9][0-9]*s$")) and
  (.staking.validator_account | type == "string" and length > 0) and
  (.staking.self_delegation | type == "string" and test("^[1-9][0-9]*$")) and
  (.governance.minimum_deposit | type == "string" and test("^[1-9][0-9]*$")) and
  (.minimum_gas_price | type == "string" and test("^(0|[1-9][0-9]*)([.][0-9]+)?$")) and
  (all(.smoke.gas, .smoke.fee, .smoke.transfer, .smoke.delegation; type == "string" and test("^[1-9][0-9]*$"))) and
  ((.smoke.fee | tonumber) >= ((.smoke.gas | tonumber) * (.minimum_gas_price | tonumber))) and
  (.multinode.validator_count == 4) and
  (.multinode.validator_balance | type == "string" and test("^[1-9][0-9]*$")) and
  (.multinode.self_delegation | type == "string" and test("^[1-9][0-9]*$")) and
  ((.multinode.validator_balance | tonumber) >= (.multinode.self_delegation | tonumber)) and
  (.multinode.port_stride | type == "number" and . >= 8 and floor == .) and
  (all(.multinode.ports[]; type == "number" and . >= 1024 and . <= 65000 and floor == .)) and
  ([.multinode.ports[]] | unique | length) == (.multinode.ports | length) and
  (([.multinode.ports[]] | max) + ((.multinode.validator_count - 1) * .multinode.port_stride) <= 65535) and
  (.accounts | type == "array" and length >= 2) and
  (all(.accounts[]; (.name | type == "string" and length > 0) and (.balance | type == "string" and test("^[1-9][0-9]*$")))) and
  ([.accounts[].name] | unique | length) == (.accounts | length) and
  ([.accounts[] | select(.name == $profile.staking.validator_account)] | length) == 1 and
  ((.accounts[] | select(.name == $profile.staking.validator_account) | .balance | tonumber) >= (.staking.self_delegation | tonumber))
' "$profile" >/dev/null

echo "network profile is valid: ${profile}"
