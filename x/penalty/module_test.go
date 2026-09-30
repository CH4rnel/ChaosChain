package penalty

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

	domain "github.com/CH4rnel/ChaosChain/pkg/penalty"
	"github.com/CH4rnel/ChaosChain/x/penalty/keeper"
)

func TestGenesisRoundTrip(t *testing.T) {
	key := storetypes.NewKVStoreKey(ModuleName)
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_penalty"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(c, runtime.NewKVStoreService(key))
	module := NewAppModule(c, k)
	want := GenesisState{Params: domain.Params{BaseSlash: 0.05, Kappa: 3}}

	module.InitGenesis(ctx, c, mustMarshalGenesis(want))

	var got GenesisState
	require.NoError(t, json.Unmarshal(module.ExportGenesis(ctx, c), &got))
	require.Equal(t, want, got)
}

func TestValidateGenesisRejectsInvalidPenaltyConfiguration(t *testing.T) {
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	genesis := defaultGenesisState()
	genesis.Params.Kappa = -1

	err := (AppModule{}).ValidateGenesis(c, nil, mustMarshalGenesis(genesis))
	require.ErrorContains(t, err, "kappa must be in the range")
}

func TestValidateGenesisRequiresExplicitParameters(t *testing.T) {
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	for _, bz := range [][]byte{[]byte(`{}`), []byte(`{"params":null}`)} {
		err := (AppModule{}).ValidateGenesis(c, nil, bz)
		require.ErrorContains(t, err, "missing penalty genesis params")
	}
}

func TestValidateGenesisRequiresEveryPenaltyParameter(t *testing.T) {
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	for _, params := range []string{
		`{"base_slash":0.01}`,
		`{"kappa":2}`,
		`{"base_slash":0.01,"kappa":null}`,
	} {
		genesis := json.RawMessage(`{"params":` + params + `}`)
		err := (AppModule{}).ValidateGenesis(c, nil, genesis)
		require.ErrorContains(t, err, "missing penalty genesis parameter")
	}
}

func TestInitGenesisAcceptsExplicitZeroParameters(t *testing.T) {
	key := storetypes.NewKVStoreKey(ModuleName)
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_penalty"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(c, runtime.NewKVStoreService(key))
	module := NewAppModule(c, k)
	want := GenesisState{Params: domain.Params{}}
	genesis := mustMarshalGenesis(want)

	require.NoError(t, module.ValidateGenesis(c, nil, genesis))
	module.InitGenesis(ctx, c, genesis)
	got, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, want.Params, got)
}

func TestInitGenesisRejectsInvalidParamsWithoutOverwritingStoredState(t *testing.T) {
	key := storetypes.NewKVStoreKey(ModuleName)
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_penalty"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(c, runtime.NewKVStoreService(key))
	module := NewAppModule(c, k)
	storedParams := domain.DefaultParams()
	genesis := defaultGenesisState()
	genesis.Params.BaseSlash = -0.01
	require.NoError(t, k.SetParams(ctx, storedParams))

	require.Panics(t, func() { module.InitGenesis(ctx, c, mustMarshalGenesis(genesis)) })
	gotParams, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, storedParams, gotParams)
}

func TestKeeperRejectsInvalidPenaltyConfiguration(t *testing.T) {
	key := storetypes.NewKVStoreKey(ModuleName)
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_penalty"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(c, runtime.NewKVStoreService(key))
	params := domain.DefaultParams()
	params.Kappa = -1

	require.ErrorContains(t, k.SetParams(ctx, params), "validate penalty parameters")
}

func TestKeeperRejectsInvalidStoredPenaltyConfiguration(t *testing.T) {
	key := storetypes.NewKVStoreKey(ModuleName)
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_penalty"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(c, runtime.NewKVStoreService(key))
	params := domain.DefaultParams()
	params.Kappa = -1
	require.NoError(t, k.Params.Set(ctx, params))

	_, err := k.GetParams(ctx)
	require.ErrorContains(t, err, "validate stored penalty parameters")
}

func FuzzValidateGenesisNeverPanics(f *testing.F) {
	f.Add([]byte(mustMarshalGenesis(defaultGenesisState())))
	f.Add([]byte("{"))
	f.Add([]byte(`{"params":{}}`))

	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	f.Fuzz(func(t *testing.T, data []byte) {
		_ = (AppModule{}).ValidateGenesis(c, nil, data)
	})
}
