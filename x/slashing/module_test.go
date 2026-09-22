package slashing

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

	domain "github.com/CH4rnel/ChaosChain/pkg/slashing"
	"github.com/CH4rnel/ChaosChain/x/slashing/keeper"
)

func TestGenesisRoundTrip(t *testing.T) {
	key := storetypes.NewKVStoreKey(ModuleName)
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_slashing"))
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

func TestKeeperRejectsInvalidPenaltyConfiguration(t *testing.T) {
	key := storetypes.NewKVStoreKey(ModuleName)
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_slashing"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(c, runtime.NewKVStoreService(key))
	params := domain.DefaultParams()
	params.Kappa = -1

	require.ErrorContains(t, k.SetParams(ctx, params), "validate slashing parameters")
}
