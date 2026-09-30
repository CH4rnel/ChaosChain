package penalty

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/appmodule"
	domain "github.com/CH4rnel/ChaosChain/pkg/penalty"
	"github.com/CH4rnel/ChaosChain/x/penalty/keeper"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
)

const ModuleName = "penalty"

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
}

func defaultGenesisState() GenesisState {
	return GenesisState{Params: domain.DefaultParams()}
}

func mustMarshalGenesis(genesis GenesisState) json.RawMessage {
	bz, err := json.Marshal(genesis)
	if err != nil {
		panic(fmt.Errorf("encode penalty genesis: %w", err))
	}
	return bz
}

func decodeAndValidateGenesis(bz json.RawMessage) (GenesisState, error) {
	var payload struct {
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(bz, &payload); err != nil {
		return GenesisState{}, fmt.Errorf("decode penalty genesis: %w", err)
	}
	var rootFields map[string]json.RawMessage
	if err := json.Unmarshal(bz, &rootFields); err != nil {
		return GenesisState{}, fmt.Errorf("decode penalty genesis: %w", err)
	}
	for name := range rootFields {
		if name != "params" {
			return GenesisState{}, fmt.Errorf("unknown penalty genesis field: %s", name)
		}
	}
	if len(payload.Params) == 0 {
		return GenesisState{}, fmt.Errorf("missing penalty genesis params")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload.Params, &fields); err != nil {
		return GenesisState{}, fmt.Errorf("decode penalty genesis params: %w", err)
	}
	if fields == nil {
		return GenesisState{}, fmt.Errorf("missing penalty genesis params")
	}
	for name := range fields {
		if name != "base_slash" && name != "kappa" {
			return GenesisState{}, fmt.Errorf("unknown penalty genesis parameter: %s", name)
		}
	}
	var params struct {
		BaseSlash *float64 `json:"base_slash"`
		Kappa     *float64 `json:"kappa"`
	}
	if err := json.Unmarshal(payload.Params, &params); err != nil {
		return GenesisState{}, fmt.Errorf("decode penalty genesis params: %w", err)
	}
	if params.BaseSlash == nil {
		return GenesisState{}, fmt.Errorf("missing penalty genesis parameter: base_slash")
	}
	if params.Kappa == nil {
		return GenesisState{}, fmt.Errorf("missing penalty genesis parameter: kappa")
	}
	genesis := GenesisState{Params: domain.Params{BaseSlash: *params.BaseSlash, Kappa: *params.Kappa}}
	if err := domain.ValidateParams(genesis.Params); err != nil {
		return GenesisState{}, fmt.Errorf("validate penalty genesis: %w", err)
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
		panic(fmt.Errorf("initialize penalty parameters: %w", err))
	}
}

func (m AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	params, err := m.keeper.GetParams(ctx)
	if err != nil {
		panic(fmt.Errorf("export penalty parameters: %w", err))
	}
	return mustMarshalGenesis(GenesisState{Params: params})
}

func (m AppModule) ConsensusVersion() uint64 { return 1 }
