package feemarket

import (
	"context"
	"encoding/json"

	"cosmossdk.io/core/appmodule"
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

func (AppModule) Name() string        { return ModuleName }
func (AppModule) IsOnePerModuleType() {}
func (AppModule) IsAppModule()        {}

func (AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return json.RawMessage(`{}`)
}

func (AppModule) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
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
}

func (m AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	return json.RawMessage(`{}`)
}

func (m AppModule) ConsensusVersion() uint64 { return 1 }

func (m AppModule) EndBlock(ctx context.Context) error { return m.keeper.EndBlock(ctx) }
