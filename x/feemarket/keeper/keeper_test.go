package keeper

import (
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/log/v2"
	"github.com/CH4rnel/ChaosChain/pkg/feemarket"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func newTestKeeper(t *testing.T) (sdk.Context, Keeper) {
	t.Helper()

	key := storetypes.NewKVStoreKey("feemarket")
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_feemarket"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	return ctx, NewKeeper(cdc, runtime.NewKVStoreService(key))
}

func TestJSONCodecSupportsCollectionsSerializationContracts(t *testing.T) {
	valueCodec := jsonCodec[feemarket.Params]{}
	want := feemarket.DefaultParams()

	encoded, err := valueCodec.EncodeJSON(want)
	require.NoError(t, err)
	got, err := valueCodec.DecodeJSON(encoded)
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.NotEmpty(t, valueCodec.Stringify(want))
	require.Equal(t, "json", valueCodec.ValueType())
}

func TestJSONCodecRejectsMalformedJSON(t *testing.T) {
	valueCodec := jsonCodec[feemarket.Params]{}

	_, err := valueCodec.DecodeJSON([]byte("{"))
	require.Error(t, err)
}

func TestGetParamsReturnsDefaultWhenParamsAreMissing(t *testing.T) {
	ctx, k := newTestKeeper(t)

	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, feemarket.DefaultParams(), params)
}

func TestSetParamsRejectsInvalidConfigurationWithoutPersistingIt(t *testing.T) {
	ctx, k := newTestKeeper(t)
	params := feemarket.DefaultParams()
	params.GasTarget = 0

	require.ErrorContains(t, k.SetParams(ctx, params), "validate fee market parameters")
	_, err := k.Params.Get(ctx)
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func TestGetStateRejectsInvalidStoredParamsWhenStateIsMissing(t *testing.T) {
	ctx, k := newTestKeeper(t)
	params := feemarket.DefaultParams()
	params.GasTarget = 0
	require.NoError(t, k.Params.Set(ctx, params))

	_, err := k.GetState(ctx)
	require.ErrorContains(t, err, "validate stored fee market parameters")
}

func TestGetStateReturnsDefaultWhenStateAndParamsAreMissing(t *testing.T) {
	ctx, k := newTestKeeper(t)

	state, err := k.GetState(ctx)
	require.NoError(t, err)
	require.Equal(t, feemarket.State{BaseFee: 10}, state)
}

func TestKeeperPersistsValidControllerParamsAndState(t *testing.T) {
	ctx, k := newTestKeeper(t)
	params := feemarket.Params{Kp: 0.2, Ki: 0.03, AntiWindupLimit: 7, GasTarget: 15_000_000}
	state := feemarket.State{BaseFee: 25, Acc: 0.5}

	require.NoError(t, k.SetParams(ctx, params))
	require.NoError(t, k.SetState(ctx, state))

	gotParams, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, params, gotParams)
	gotState, err := k.GetState(ctx)
	require.NoError(t, err)
	require.Equal(t, state, gotState)
}

func TestSetStateDoesNotPersistAccumulatorOutsideConfiguredBounds(t *testing.T) {
	ctx, k := newTestKeeper(t)
	params := feemarket.DefaultParams()
	params.AntiWindupLimit = 2
	require.NoError(t, k.SetParams(ctx, params))

	err := k.SetState(ctx, feemarket.State{BaseFee: 10, Acc: 2.1})
	require.ErrorContains(t, err, "accumulator exceeds anti-windup limit")
	_, err = k.State.Get(ctx)
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func TestGetStateRejectsInvalidPersistedState(t *testing.T) {
	ctx, k := newTestKeeper(t)
	require.NoError(t, k.Params.Set(ctx, feemarket.DefaultParams()))
	require.NoError(t, k.State.Set(ctx, feemarket.State{}))

	_, err := k.GetState(ctx)
	require.ErrorContains(t, err, "validate stored fee market state")
}

func TestGetStateRejectsPersistedStateOutsideParameterBounds(t *testing.T) {
	ctx, k := newTestKeeper(t)
	params := feemarket.DefaultParams()
	params.AntiWindupLimit = 2
	require.NoError(t, k.Params.Set(ctx, params))
	require.NoError(t, k.State.Set(ctx, feemarket.State{BaseFee: 10, Acc: 2.1}))

	_, err := k.GetState(ctx)
	require.ErrorContains(t, err, "accumulator exceeds anti-windup limit")
}

func TestSetStateRejectsInvalidStateWithoutPersistingIt(t *testing.T) {
	ctx, k := newTestKeeper(t)

	require.ErrorContains(t, k.SetState(ctx, feemarket.State{}), "validate fee market state")
	_, err := k.State.Get(ctx)
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func TestEndBlockPersistsTransitionUsingFinalizedBlockGas(t *testing.T) {
	ctx, k := newTestKeeper(t)
	ctx = ctx.WithBlockGasUsed(20_000_000)
	params := feemarket.DefaultParams()
	initial := feemarket.State{BaseFee: 10, Acc: 0}
	require.NoError(t, k.SetParams(ctx, params))
	require.NoError(t, k.SetState(ctx, initial))
	want, err := feemarket.Next(initial, float64(ctx.BlockGasUsed()), params)
	require.NoError(t, err)

	require.NoError(t, k.EndBlock(ctx))
	got, err := k.GetState(ctx)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestEndBlockRejectsInvalidInputsWithoutChangingStoredState(t *testing.T) {
	tests := []struct {
		name          string
		gasUsed       uint64
		params        feemarket.Params
		state         feemarket.State
		expectedError string
	}{
		{
			name:          "invalid persisted parameters",
			params:        feemarket.Params{GasTarget: 0},
			state:         feemarket.State{BaseFee: 10},
			expectedError: "validate stored fee market parameters",
		},
		{
			name:          "invalid persisted state",
			params:        feemarket.DefaultParams(),
			state:         feemarket.State{},
			expectedError: "validate stored fee market state",
		},
		{
			name:          "gas above consensus limit",
			gasUsed:       uint64(feemarket.MaxGasUsed) + 1,
			params:        feemarket.DefaultParams(),
			state:         feemarket.State{BaseFee: 10},
			expectedError: "gasUsed out of valid range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, k := newTestKeeper(t)
			ctx = ctx.WithBlockGasUsed(tt.gasUsed)
			require.NoError(t, k.Params.Set(ctx, tt.params))
			require.NoError(t, k.State.Set(ctx, tt.state))

			require.ErrorContains(t, k.EndBlock(ctx), tt.expectedError)
			gotParams, err := k.Params.Get(ctx)
			require.NoError(t, err)
			require.Equal(t, tt.params, gotParams)
			gotState, err := k.State.Get(ctx)
			require.NoError(t, err)
			require.Equal(t, tt.state, gotState)
		})
	}
}
