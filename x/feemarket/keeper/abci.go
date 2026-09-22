// x/feemarket/keeper/abci.go
package keeper

import (
	"context"

	"github.com/CH4rnel/ChaosChain/pkg/feemarket"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EndBlock updates the fee market state based on the block's gas consumption.
func (k Keeper) EndBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// 1. Gather block data
	gasUsed := float64(sdkCtx.GasMeter().GasConsumed())

	// 2. Fetch current state and params from KVStore
	params, err := k.GetParams(ctx)
	if err != nil {
		sdkCtx.Logger().Error("failed to get feemarket params", "error", err)
		return err
	}

	currentState, err := k.GetState(ctx)
	if err != nil {
		sdkCtx.Logger().Error("failed to get feemarket state", "error", err)
		return err
	}

	// 3. Delegate to pure domain logic
	nextState, err := feemarket.Next(currentState, gasUsed, params)
	if err != nil {
		sdkCtx.Logger().Error("failed to calculate next fee market state", "error", err)
		return err
	}

	// 4. Persist new state
	if err := k.SetState(ctx, nextState); err != nil {
		sdkCtx.Logger().Error("failed to set feemarket state", "error", err)
		return err
	}

	sdkCtx.Logger().Info("fee market state updated",
		"new_base_fee", nextState.BaseFee,
		"new_acc", nextState.Acc,
		"gas_used", gasUsed,
	)

	return nil
}
