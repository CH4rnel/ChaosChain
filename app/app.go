package app

import (
	"encoding/json"
	"fmt"
	"io"

	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	// Local modules registration (side-effect imports for depinject)
	_ "github.com/CH4rnel/ChaosChain/x/feemarket"
	_ "github.com/CH4rnel/ChaosChain/x/slashing"
)

var (
	_ runtime.AppI            = (*ChaosChainApp)(nil)
	_ servertypes.Application = (*ChaosChainApp)(nil)
)

type ChaosChainApp struct {
	*runtime.App
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry codectypes.InterfaceRegistry

	// Core Keepers
	BankKeeper    banktypes.Keeper
	StakingKeeper stakingtypes.Keeper

	// Module lifecycle management
	ModuleManager *module.Manager
	configurator  module.Configurator
}

func NewChaosChainApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *ChaosChainApp {
	var (
		app        = &ChaosChainApp{}
		appBuilder *runtime.AppBuilder
	)

	// Dependency Injection for core SDK modules and keepers
	if err := depinject.Inject(
		depinject.Configs(
			// In production, app_config.go is loaded here.
			// For the current build, we use the standard SDK configuration.
		),
		&appBuilder,
		&app.appCodec,
		&app.legacyAmino,
		&app.txConfig,
		&app.interfaceRegistry,
		&app.BankKeeper,
		&app.StakingKeeper,
	); err != nil {
		panic(fmt.Errorf("failed to inject dependencies: %w", err))
	}

	app.App = appBuilder.Build(db, traceStore, baseAppOptions...)

	// Initializing ModuleManager for lifecycle management
	app.ModuleManager = module.NewManager(
		bank.NewAppModule(app.appCodec, app.BankKeeper, nil),
		staking.NewAppModule(app.appCodec, app.StakingKeeper, app.BankKeeper, nil),
		// feemarket.NewAppModule(...),
		// slashing.NewAppModule(...),
	)

	// Determining the execution order of BeginBlockers (critical for staking and banking)
	app.ModuleManager.SetOrderBeginBlockers(
		stakingtypes.ModuleName,
		banktypes.ModuleName,
	)

	// Determining the execution order of EndBlockers (validator updates, issuance)
	app.ModuleManager.SetOrderEndBlockers(
		stakingtypes.ModuleName,
		banktypes.ModuleName,
	)

	// Genesis initialization and export sequence (staking must be initialized before bank to ensure correct binding of delegations)
	genesisModuleOrder := []string{
		stakingtypes.ModuleName,
		banktypes.ModuleName,
	}
	app.ModuleManager.SetOrderInitGenesis(genesisModuleOrder...)
	app.ModuleManager.SetOrderExportGenesis(genesisModuleOrder...)

	// Service registration and message routing
	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	app.ModuleManager.RegisterServices(app.configurator)

	// Sealing the application and loading the latest state version
	if err := app.LoadLatestVersion(); err != nil {
		panic(fmt.Errorf("failed to load latest version: %w", err))
	}

	return app
}

func (app *ChaosChainApp) Name() string { return "ChaosChain" }
func (app *ChaosChainApp) LegacyAmino() *codec.LegacyAmino { return app.legacyAmino }
func (app *ChaosChainApp) AppCodec() codec.Codec { return app.appCodec }
func (app *ChaosChainApp) InterfaceRegistry() codectypes.InterfaceRegistry { return app.interfaceRegistry }
func (app *ChaosChainApp) TxConfig() client.TxConfig { return app.txConfig }

// DefaultGenesis restores the default state for all registered modules.
func (app *ChaosChainApp) DefaultGenesis() map[string]json.RawMessage {
	return app.ModuleManager.DefaultGenesis()
}

// InitGenesis initializes the application state from the genesis file.
func (app *ChaosChainApp) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, genesisState map[string]json.RawMessage) {
	app.ModuleManager.InitGenesis(ctx, cdc, genesisState)
}

// ExportGenesis exports the application's current state to the genesis format (for snapshots and migrations).
func (app *ChaosChainApp) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) map[string]json.RawMessage {
	return app.ModuleManager.ExportGenesis(ctx, cdc)
}

func (app *ChaosChainApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

func (app *ChaosChainApp) ModuleManager() *module.Manager { return app.ModuleManager }
func (app *ChaosChainApp) SimulationManager() *module.SimulationManager { return nil }

func (app *ChaosChainApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	app.App.RegisterAPIRoutes(apiSvr, apiConfig)
}

func (app *ChaosChainApp) RegisterTxService(clientCtx client.Context) {
	app.App.RegisterTxService(clientCtx)
}

func (app *ChaosChainApp) RegisterTendermintService(clientCtx client.Context) {
	app.App.RegisterTendermintService(clientCtx)
}