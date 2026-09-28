package keeper

import (
	"testing"

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
