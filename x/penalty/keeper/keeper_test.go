package keeper

import (
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/log/v2"
	domain "github.com/CH4rnel/ChaosChain/pkg/penalty"
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

	key := storetypes.NewKVStoreKey("penalty")
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_penalty"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	return ctx, NewKeeper(cdc, runtime.NewKVStoreService(key))
}

func TestGetParamsReturnsDefaultWhenParamsAreMissing(t *testing.T) {
	ctx, k := newTestKeeper(t)

	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, domain.DefaultParams(), params)
}

func TestKeeperPersistsParamsAndAppliesCorrelatedSlashFormula(t *testing.T) {
	ctx, k := newTestKeeper(t)
	params := domain.Params{BaseSlash: 0.05, Kappa: 2}
	require.NoError(t, k.SetParams(ctx, params))

	gotParams, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, params, gotParams)

	fraction, err := k.Slash(ctx, 0.3)
	require.NoError(t, err)
	require.InDelta(t, 0.23, fraction, 1e-12)
}

func TestSlashRejectsInvalidPersistedPenaltyParams(t *testing.T) {
	ctx, k := newTestKeeper(t)
	params := domain.DefaultParams()
	params.Kappa = -1
	require.NoError(t, k.Params.Set(ctx, params))

	_, err := k.Slash(ctx, 0.3)
	require.ErrorContains(t, err, "validate stored penalty parameters")
}

func TestSetParamsRejectsInvalidConfigurationWithoutPersistingIt(t *testing.T) {
	ctx, k := newTestKeeper(t)
	params := domain.DefaultParams()
	params.BaseSlash = -0.01
	require.ErrorContains(t, k.SetParams(ctx, params), "validate penalty parameters")
	_, err := k.Params.Get(ctx)
	require.ErrorIs(t, err, collections.ErrNotFound)
}
