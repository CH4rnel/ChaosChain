package feemarket

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/appmodule"
	sdkmath "cosmossdk.io/math"
	domain "github.com/CH4rnel/ChaosChain/pkg/feemarket"
	"github.com/CH4rnel/ChaosChain/x/feemarket/keeper"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
)

const ModuleName = "feemarket"

var (
	_ module.AppModuleBasic = AppModule{}
	_ module.AppModule      = AppModule{}
	_ appmodule.AppModule   = AppModule{}
)

type AppModule struct {
	cdc    codec.Codec
	keeper keeper.Keeper
}

type GenesisState struct {
	Params domain.Params `json:"params"`
	State  domain.State  `json:"state"`
}

func defaultGenesisState() GenesisState {
	return GenesisState{
		Params: domain.DefaultParams(),
		State:  domain.State{BaseFee: sdkmath.LegacyNewDec(10), Acc: sdkmath.LegacyZeroDec()},
	}
}

type genesisDecimal struct {
	value sdkmath.LegacyDec
}

func (d *genesisDecimal) UnmarshalJSON(bz []byte) error {
	var text string
	if len(bz) > 0 && bz[0] == '"' {
		if err := json.Unmarshal(bz, &text); err != nil {
			return err
		}
	} else {
		text = string(bz)
	}
	value, err := sdkmath.LegacyNewDecFromStr(text)
	if err != nil {
		return err
	}
	d.value = value
	return nil
}

func mustMarshalGenesis(genesis GenesisState) json.RawMessage {
	bz, err := json.Marshal(genesis)
	if err != nil {
		panic(fmt.Errorf("encode fee market genesis: %w", err))
	}
	return bz
}

func rejectUnknownGenesisFields(bz json.RawMessage, objectName, scope string, allowed ...string) error {
	var objects map[string]json.RawMessage
	if err := json.Unmarshal(bz, &objects); err != nil {
		return fmt.Errorf("decode fee market genesis: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(objects[objectName], &fields); err != nil {
		return fmt.Errorf("decode fee market genesis %s: %w", scope, err)
	}
	known := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		known[name] = struct{}{}
	}
	for name := range fields {
		if _, ok := known[name]; !ok {
			return fmt.Errorf("unknown fee market genesis %s: %s", scope, name)
		}
	}
	return nil
}

func decodeAndValidateGenesis(bz json.RawMessage) (GenesisState, error) {
	var payload struct {
		Params *struct {
			Kp              *genesisDecimal `json:"kp"`
			Ki              *genesisDecimal `json:"ki"`
			AntiWindupLimit *genesisDecimal `json:"anti_windup_limit"`
			GasTarget       *genesisDecimal `json:"gas_target"`
		} `json:"params"`
		State *struct {
			BaseFee *genesisDecimal `json:"base_fee"`
			Acc     *genesisDecimal `json:"acc"`
		} `json:"state"`
	}
	if err := json.Unmarshal(bz, &payload); err != nil {
		return GenesisState{}, fmt.Errorf("decode fee market genesis: %w", err)
	}
	if payload.Params == nil {
		return GenesisState{}, fmt.Errorf("missing fee market genesis params")
	}
	if err := rejectUnknownGenesisFields(bz, "params", "parameter", "kp", "ki", "anti_windup_limit", "gas_target"); err != nil {
		return GenesisState{}, err
	}
	params := payload.Params
	if params.Kp == nil {
		return GenesisState{}, fmt.Errorf("missing fee market genesis parameter: kp")
	}
	if params.Ki == nil {
		return GenesisState{}, fmt.Errorf("missing fee market genesis parameter: ki")
	}
	if params.AntiWindupLimit == nil {
		return GenesisState{}, fmt.Errorf("missing fee market genesis parameter: anti_windup_limit")
	}
	if params.GasTarget == nil {
		return GenesisState{}, fmt.Errorf("missing fee market genesis parameter: gas_target")
	}
	if payload.State == nil {
		return GenesisState{}, fmt.Errorf("missing fee market genesis state")
	}
	if err := rejectUnknownGenesisFields(bz, "state", "state field", "base_fee", "acc"); err != nil {
		return GenesisState{}, err
	}
	if payload.State.BaseFee == nil {
		return GenesisState{}, fmt.Errorf("missing fee market genesis state field: base_fee")
	}
	if payload.State.Acc == nil {
		return GenesisState{}, fmt.Errorf("missing fee market genesis state field: acc")
	}
	genesis := GenesisState{
		Params: domain.Params{
			Kp:              params.Kp.value,
			Ki:              params.Ki.value,
			AntiWindupLimit: params.AntiWindupLimit.value,
			GasTarget:       params.GasTarget.value,
		},
		State: domain.State{BaseFee: payload.State.BaseFee.value, Acc: payload.State.Acc.value},
	}
	if err := domain.ValidateParams(genesis.Params); err != nil {
		return GenesisState{}, fmt.Errorf("validate fee market genesis parameters: %w", err)
	}
	if err := domain.ValidateStateWithParams(genesis.State, genesis.Params); err != nil {
		return GenesisState{}, fmt.Errorf("validate fee market genesis state: %w", err)
	}
	return genesis, nil
}

func (AppModule) Name() string        { return ModuleName }
func (AppModule) IsOnePerModuleType() {}
func (AppModule) IsAppModule()        {}

func (AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return mustMarshalGenesis(defaultGenesisState())
}

func (AppModule) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	_, err := decodeAndValidateGenesis(bz)
	return err
}

func (AppModule) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino)                           {}
func (AppModule) RegisterInterfaces(registry codectypes.InterfaceRegistry)                  {}
func (AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {}
func (AppModule) GetTxCmd() *cobra.Command                                                  { return nil }
func (AppModule) GetQueryCmd() *cobra.Command                                               { return nil }

func NewAppModule(cdc codec.Codec, keeper keeper.Keeper) AppModule {
	return AppModule{cdc: cdc, keeper: keeper}
}

func (m AppModule) RegisterInvariants(sdk.InvariantRegistry) {}
func (m AppModule) RegisterServices(cfg module.Configurator) {}

func (m AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	genesis, err := decodeAndValidateGenesis(data)
	if err != nil {
		panic(err)
	}
	if err := m.keeper.SetParams(ctx, genesis.Params); err != nil {
		panic(fmt.Errorf("initialize fee market parameters: %w", err))
	}
	if err := m.keeper.SetState(ctx, genesis.State); err != nil {
		panic(fmt.Errorf("initialize fee market state: %w", err))
	}
}

func (m AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	params, err := m.keeper.GetParams(ctx)
	if err != nil {
		panic(fmt.Errorf("export fee market parameters: %w", err))
	}
	state, err := m.keeper.GetState(ctx)
	if err != nil {
		panic(fmt.Errorf("export fee market state: %w", err))
	}
	return mustMarshalGenesis(GenesisState{Params: params, State: state})
}

func (m AppModule) ConsensusVersion() uint64 { return 1 }

func (m AppModule) EndBlock(ctx context.Context) error { return m.keeper.EndBlock(ctx) }
