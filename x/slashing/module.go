package slashing

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/appmodule"
	domain "github.com/CH4rnel/ChaosChain/pkg/slashing"
	"github.com/CH4rnel/ChaosChain/x/slashing/keeper"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
)

const ModuleName = "slashing"

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
		panic(fmt.Errorf("encode slashing genesis: %w", err))
	}
	return bz
}

func (AppModule) Name() string        { return ModuleName }
func (AppModule) IsOnePerModuleType() {}
func (AppModule) IsAppModule()        {}

func (AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return mustMarshalGenesis(defaultGenesisState())
}

func (AppModule) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var genesis GenesisState
	if err := json.Unmarshal(bz, &genesis); err != nil {
		return fmt.Errorf("decode slashing genesis: %w", err)
	}
	if err := domain.ValidateParams(genesis.Params); err != nil {
		return fmt.Errorf("validate slashing genesis: %w", err)
	}
	return nil
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
	var genesis GenesisState
	if err := json.Unmarshal(data, &genesis); err != nil {
		panic(fmt.Errorf("decode slashing genesis: %w", err))
	}
	if err := m.keeper.SetParams(ctx, genesis.Params); err != nil {
		panic(fmt.Errorf("initialize slashing parameters: %w", err))
	}
}

func (m AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	params, err := m.keeper.GetParams(ctx)
	if err != nil {
		panic(fmt.Errorf("export slashing parameters: %w", err))
	}
	return mustMarshalGenesis(GenesisState{Params: params})
}

func (m AppModule) ConsensusVersion() uint64 { return 1 }
