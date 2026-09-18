package app_test

import (
	"testing"

	"cosmossdk.io/log/v2"
	storetypes "cosmossdk.io/store/types"
	"github.com/CH4rnel/ChaosChain/app"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"
)

// TestAppInitializationAndEndBlock verifies that the App wires up correctly,
// the store is mounted, and the EndBlocker logic actually persists state changes.
func TestAppInitializationAndEndBlock(t *testing.T) {
	// 1. Initialize the App
	chaosApp, err := app.WireApp()
	require.NoError(t, err, "WireApp should not fail")
	require.NotNil(t, chaosApp, "App should be initialized")

	// 2. Create a Context manually with all required arguments for v0.54+
	// sdk.NewContext requires: (MultiStore, Header, isCheckTx, Logger)
	header := cmtproto.Header{Height: 1}
	ctx := sdk.NewContext(chaosApp.CommitMultiStore(), header, false, log.NewNopLogger())

	// 3. Verify Initial State (Defaults)
	initialState, err := chaosApp.FeeMarketKeeper.GetState(ctx)
	require.NoError(t, err)
	require.Equal(t, 10.0, initialState.BaseFee, "Initial base fee should be default (10.0)")

	// 4. Simulate Block Gas Consumption (Overload)
	// Target is 10M. Consume 15M to trigger fee increase.
	gasMeter := storetypes.NewGasMeter(20000000)
	gasMeter.ConsumeGas(15000000, "test block gas")
	ctx = ctx.WithGasMeter(gasMeter)

	// 5. Execute EndBlock
	err = chaosApp.FeeMarketKeeper.EndBlock(ctx)
	require.NoError(t, err, "EndBlock should not error")

	// 6. Verify State Persistence
	newState, err := chaosApp.FeeMarketKeeper.GetState(ctx)
	require.NoError(t, err)

	// Since gas_used (15M) > gas_target (10M), the PI-regulator should increase the fee
	require.Greater(t, newState.BaseFee, initialState.BaseFee,
		"BaseFee should increase after block with gas > target")

	t.Logf("Success! BaseFee changed from %.2f to %.2f", initialState.BaseFee, newState.BaseFee)
}