// File: x/slashing/module.go
package slashing

import (
	"encoding/json"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

var (
	_ module.AppModuleBasic = AppModule{}
	_ appmodule.AppModule   = AppModule{}
)

type AppModule struct {
	cdc    codec.Codec
	keeper *Keeper
}

type Keeper struct {
	store store.KVStoreService
}

func NewKeeper(storeService store.KVStoreService) *Keeper {
	return &Keeper{store: storeService}
}

func (k *Keeper) InitGenesis(ctx sdk.Context, data json.RawMessage) {}
func (k *Keeper) ExportGenesis(ctx sdk.Context) json.RawMessage { return json.RawMessage("{}") }

func NewAppModule(cdc codec.Codec, keeper *Keeper) AppModule {
	return AppModule{cdc: cdc, keeper: keeper}
}

func (AppModule) IsOnePerModuleType() {}
func (AppModule) IsAppModule()        {}

func (m AppModule) RegisterInterfaces(registry codectypes.InterfaceRegistry) {}
func (m AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux interface{}) {}
func (m AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	m.keeper.InitGenesis(ctx, data)
}
func (m AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	return m.keeper.ExportGenesis(ctx)
}
func (m AppModule) ConsensusVersion() uint64 { return 1 }

func init() {
	appmodule.Register(&struct{}{},
		appmodule.Provide(ProvideModule),
	)
}

type ModuleInputs struct {
	depinject.In
	StoreService store.KVStoreService
}

type ModuleOutputs struct {
	depinject.Out
	Keeper    *Keeper
	AppModule appmodule.AppModule
}

func ProvideModule(in ModuleInputs) ModuleOutputs {
	k := NewKeeper(in.StoreService)
	m := NewAppModule(nil, k)
	return ModuleOutputs{Keeper: k, AppModule: m}
}