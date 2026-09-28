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
