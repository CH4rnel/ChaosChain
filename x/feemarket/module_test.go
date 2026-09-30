package feemarket

import (
	"encoding/json"
	"strings"
	"testing"

	"cosmossdk.io/log/v2"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	"github.com/stretchr/testify/require"

	domain "github.com/CH4rnel/ChaosChain/pkg/feemarket"
	"github.com/CH4rnel/ChaosChain/x/feemarket/keeper"
)

func d(value string) sdkmath.LegacyDec {
	result, err := sdkmath.LegacyNewDecFromStr(strings.ReplaceAll(value, "_", ""))
	if err != nil {
		panic(err)
	}
	return result
}

func TestGenesisRoundTrip(t *testing.T) {
	key := storetypes.NewKVStoreKey(ModuleName)
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_feemarket"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(c, runtime.NewKVStoreService(key))
	module := NewAppModule(c, k)
	want := GenesisState{
		Params: domain.Params{Kp: d("0.2"), Ki: d("0.03"), AntiWindupLimit: d("7"), GasTarget: d("15000000")},
		State:  domain.State{BaseFee: d("25"), Acc: d("0.5")},
	}

	module.InitGenesis(ctx, c, mustMarshalGenesis(want))

	var got GenesisState
	require.NoError(t, json.Unmarshal(module.ExportGenesis(ctx, c), &got))
	require.Equal(t, want, got)
}

func TestDefaultGenesisMatchesModuleContract(t *testing.T) {
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	module := AppModule{}
	genesis := module.DefaultGenesis(c)

	require.Equal(t, ModuleName, module.Name())
	require.Equal(t, uint64(1), module.ConsensusVersion())
	require.NoError(t, module.ValidateGenesis(c, nil, genesis))

	var got GenesisState
	require.NoError(t, json.Unmarshal(genesis, &got))
	require.Equal(t, defaultGenesisState(), got)
}

func TestValidateGenesisRejectsInvalidControllerConfiguration(t *testing.T) {
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	genesis := defaultGenesisState()
	genesis.Params.GasTarget = d("0")

	err := (AppModule{}).ValidateGenesis(c, nil, mustMarshalGenesis(genesis))
	require.ErrorContains(t, err, "gasTarget out of valid range")
}

func TestValidateGenesisRejectsAccumulatorOutsideAntiWindupBounds(t *testing.T) {
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	genesis := defaultGenesisState()
	genesis.Params.AntiWindupLimit = d("2")
	genesis.State.Acc = d("2.1")

	err := (AppModule{}).ValidateGenesis(c, nil, mustMarshalGenesis(genesis))
	require.ErrorContains(t, err, "accumulator exceeds anti-windup limit")
}

func TestValidateGenesisRequiresEveryFeeMarketParameter(t *testing.T) {
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	for _, params := range []string{
		`{"ki":0.01,"anti_windup_limit":10,"gas_target":10000000}`,
		`{"kp":0.1,"anti_windup_limit":10,"gas_target":10000000}`,
		`{"kp":0.1,"ki":0.01,"gas_target":10000000}`,
		`{"kp":0.1,"ki":0.01,"anti_windup_limit":10}`,
		`{"kp":null,"ki":0.01,"anti_windup_limit":10,"gas_target":10000000}`,
	} {
		genesis := json.RawMessage(`{"params":` + params + `,"state":{"base_fee":10}}`)
		err := (AppModule{}).ValidateGenesis(c, nil, genesis)
		require.ErrorContains(t, err, "missing fee market genesis parameter")
	}
}

func TestValidateGenesisAcceptsExplicitZeroControllerParameters(t *testing.T) {
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	genesis := GenesisState{
		Params: domain.Params{GasTarget: d("10000000")},
		State:  domain.State{BaseFee: d("10")},
	}

	require.NoError(t, (AppModule{}).ValidateGenesis(c, nil, mustMarshalGenesis(genesis)))
}

func TestValidateGenesisRequiresEveryFeeMarketStateField(t *testing.T) {
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	for _, state := range []string{
		`{"base_fee":10}`,
		`{"base_fee":10,"acc":null}`,
	} {
		genesis := json.RawMessage(`{"params":{"kp":0.1,"ki":0.01,"anti_windup_limit":10,"gas_target":10000000},"state":` + state + `}`)
		err := (AppModule{}).ValidateGenesis(c, nil, genesis)
		require.ErrorContains(t, err, "missing fee market genesis state field: acc")
	}
}

func TestValidateGenesisAcceptsExplicitZeroAccumulator(t *testing.T) {
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	genesis := GenesisState{
		Params: domain.DefaultParams(),
		State:  domain.State{BaseFee: d("10"), Acc: d("0")},
	}

	require.NoError(t, (AppModule{}).ValidateGenesis(c, nil, mustMarshalGenesis(genesis)))
}

func TestValidateGenesisRejectsUnknownParameterAndStateFields(t *testing.T) {
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	for _, testCase := range []struct {
		name string
		json string
		want string
	}{
		{
			name: "parameter",
			json: `{"params":{"kp":0.1,"ki":0.01,"anti_windup_limit":10,"gas_target":10000000,"kp_typo":0.2},"state":{"base_fee":10,"acc":0}}`,
			want: "unknown fee market genesis parameter: kp_typo",
		},
		{
			name: "state",
			json: `{"params":{"kp":0.1,"ki":0.01,"anti_windup_limit":10,"gas_target":10000000},"state":{"base_fee":10,"acc":0,"basefee":11}}`,
			want: "unknown fee market genesis state field: basefee",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := (AppModule{}).ValidateGenesis(c, nil, json.RawMessage(testCase.json))
			require.ErrorContains(t, err, testCase.want)
		})
	}
}

func TestInitGenesisRejectsUnknownFieldsWithoutOverwritingStoredState(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "parameter",
			json: `{"params":{"kp":0.1,"ki":0.01,"anti_windup_limit":10,"gas_target":10000000,"kp_typo":0.2},"state":{"base_fee":20,"acc":1}}`,
		},
		{
			name: "state",
			json: `{"params":{"kp":0.1,"ki":0.01,"anti_windup_limit":10,"gas_target":10000000},"state":{"base_fee":20,"acc":1,"basefee":21}}`,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			key := storetypes.NewKVStoreKey(ModuleName)
			testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_feemarket"))
			ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
			c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
			k := keeper.NewKeeper(c, runtime.NewKVStoreService(key))
			module := NewAppModule(c, k)
			storedParams := domain.DefaultParams()
			storedState := domain.State{BaseFee: d("12"), Acc: d("0.5")}
			require.NoError(t, k.SetParams(ctx, storedParams))
			require.NoError(t, k.SetState(ctx, storedState))

			require.Panics(t, func() { module.InitGenesis(ctx, c, json.RawMessage(testCase.json)) })

			gotParams, err := k.GetParams(ctx)
			require.NoError(t, err)
			require.Equal(t, storedParams, gotParams)
			gotState, err := k.GetState(ctx)
			require.NoError(t, err)
			require.Equal(t, storedState, gotState)
		})
	}
}

func TestInitGenesisValidatesAllControllerDataBeforeWriting(t *testing.T) {
	key := storetypes.NewKVStoreKey(ModuleName)
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_feemarket"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(c, runtime.NewKVStoreService(key))
	module := NewAppModule(c, k)
	genesis := defaultGenesisState()
	genesis.Params.Kp = d("0.5")
	genesis.State.BaseFee = d("0")

	require.Panics(t, func() { module.InitGenesis(ctx, c, mustMarshalGenesis(genesis)) })
	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, domain.DefaultParams(), params)
}

func TestKeeperRejectsInvalidControllerData(t *testing.T) {
	key := storetypes.NewKVStoreKey(ModuleName)
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_feemarket"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(c, runtime.NewKVStoreService(key))

	params := domain.DefaultParams()
	params.GasTarget = d("0")
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
	params.GasTarget = d("0")
	require.NoError(t, k.Params.Set(ctx, params))
	require.NoError(t, k.State.Set(ctx, domain.State{}))

	_, err := k.GetParams(ctx)
	require.ErrorContains(t, err, "validate stored fee market parameters")
	_, err = k.GetState(ctx)
	require.ErrorContains(t, err, "validate stored fee market state")
}

func TestKeeperRejectsStoredAccumulatorOutsideAntiWindupBounds(t *testing.T) {
	key := storetypes.NewKVStoreKey(ModuleName)
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_feemarket"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(c, runtime.NewKVStoreService(key))
	params := domain.DefaultParams()
	params.AntiWindupLimit = d("2")
	require.NoError(t, k.Params.Set(ctx, params))
	require.NoError(t, k.State.Set(ctx, domain.State{BaseFee: d("10"), Acc: d("2.1")}))

	_, err := k.GetState(ctx)
	require.ErrorContains(t, err, "accumulator exceeds anti-windup limit")
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

func TestEndBlockRegulatesAgainstFinalizedBlockGas(t *testing.T) {
	key := storetypes.NewKVStoreKey(ModuleName)
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_feemarket"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger()).WithBlockGasUsed(20_000_000)
	ctx.GasMeter().ConsumeGas(1_000, "transaction gas meter")
	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(c, runtime.NewKVStoreService(key))
	params := domain.DefaultParams()
	params.GasTarget = d("10000000")
	initial := domain.State{BaseFee: d("10")}
	require.NoError(t, k.SetParams(ctx, params))
	require.NoError(t, k.SetState(ctx, initial))

	require.NoError(t, k.EndBlock(ctx))
	got, err := k.GetState(ctx)
	require.NoError(t, err)
	want, err := domain.Next(initial, sdkmath.LegacyNewDec(20_000_000), params)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func FuzzValidateGenesisNeverPanics(f *testing.F) {
	f.Add([]byte(mustMarshalGenesis(defaultGenesisState())))
	f.Add([]byte("{"))
	f.Add([]byte(`{"params":{},"state":{}}`))

	c := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	f.Fuzz(func(t *testing.T, data []byte) {
		_ = (AppModule{}).ValidateGenesis(c, nil, data)
	})
}
