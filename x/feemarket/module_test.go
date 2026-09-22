package feemarket

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/log/v2"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	"github.com/stretchr/testify/require"

	domain "github.com/CH4rnel/ChaosChain/pkg/feemarket"
	"github.com/CH4rnel/ChaosChain/x/feemarket/keeper"
)

func TestGenesisRoundTrip(t *testing.T) {
	key := storetypes.NewKVStoreKey(ModuleName)
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_feemarket"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(c, runtime.NewKVStoreService(key))
	module := NewAppModule(c, k)
	want := GenesisState{
		Params: domain.Params{Kp: 0.2, Ki: 0.03, AntiWindupLimit: 7, GasTarget: 15_000_000},
		State:  domain.State{BaseFee: 25, Acc: 0.5},
	}

	module.InitGenesis(ctx, c, mustMarshalGenesis(want))

	var got GenesisState
	require.NoError(t, json.Unmarshal(module.ExportGenesis(ctx, c), &got))
	require.Equal(t, want, got)
}

func TestValidateGenesisRejectsInvalidControllerConfiguration(t *testing.T) {
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	genesis := defaultGenesisState()
	genesis.Params.GasTarget = 0

	err := (AppModule{}).ValidateGenesis(c, nil, mustMarshalGenesis(genesis))
	require.ErrorContains(t, err, "gasTarget out of valid range")
}

func TestKeeperRejectsInvalidControllerData(t *testing.T) {
	key := storetypes.NewKVStoreKey(ModuleName)
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_feemarket"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(c, runtime.NewKVStoreService(key))

	params := domain.DefaultParams()
	params.GasTarget = 0
	require.ErrorContains(t, k.SetParams(ctx, params), "validate fee market parameters")
	require.ErrorContains(t, k.SetState(ctx, domain.State{}), "validate fee market state")
}

func TestKeeperRejectsInvalidStoredControllerData(t *testing.T) {
	key := storetypes.NewKVStoreKey(ModuleName)
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_feemarket"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(c, runtime.NewKVStoreService(key))
	params := domain.DefaultParams()
	params.GasTarget = 0
	require.NoError(t, k.Params.Set(ctx, params))
	require.NoError(t, k.State.Set(ctx, domain.State{}))

	_, err := k.GetParams(ctx)
	require.ErrorContains(t, err, "validate stored fee market parameters")
	_, err = k.GetState(ctx)
	require.ErrorContains(t, err, "validate stored fee market state")
}

func TestEndBlockPropagatesInvalidControllerState(t *testing.T) {
	key := storetypes.NewKVStoreKey(ModuleName)
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_feemarket"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(c, runtime.NewKVStoreService(key))
	require.NoError(t, k.Params.Set(ctx, domain.DefaultParams()))
	require.NoError(t, k.State.Set(ctx, domain.State{}))

	require.ErrorContains(t, k.EndBlock(ctx), "baseFee out of valid range")
}
