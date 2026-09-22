package app

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"cosmossdk.io/depinject"
	"cosmossdk.io/depinject/appconfig"
	"cosmossdk.io/log/v2"
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
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/spf13/cobra"

	feemarketmodule "github.com/CH4rnel/ChaosChain/x/feemarket"
	feemarketkeeper "github.com/CH4rnel/ChaosChain/x/feemarket/keeper"
	slashingmodule "github.com/CH4rnel/ChaosChain/x/slashing"
	slashingkeeper "github.com/CH4rnel/ChaosChain/x/slashing/keeper"

	_ "cosmossdk.io/api/cosmos/app/runtime/v1alpha1"
	_ "cosmossdk.io/api/cosmos/tx/config/v1"
	_ "github.com/cosmos/cosmos-sdk/x/auth"
	_ "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	_ "github.com/cosmos/cosmos-sdk/x/bank"
	_ "github.com/cosmos/cosmos-sdk/x/staking"
)

//go:embed app.yaml
var appConfigYAML []byte

var DefaultNodeHome string

func init() {
	sdkConfig := sdk.GetConfig()
	sdkConfig.SetBech32PrefixForAccount("chaos", "chaospub")
	sdkConfig.SetBech32PrefixForValidator("chaosvaloper", "chaosvaloperpub")
	sdkConfig.SetBech32PrefixForConsensusNode("chaosvalcons", "chaosvalconspub")
	sdkConfig.Seal()

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	DefaultNodeHome = filepath.Join(userHomeDir, ".chaoschain")
}

type ChaosChainApp struct {
	*runtime.App
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry codectypes.InterfaceRegistry

	BankKeeper      bankkeeper.BaseKeeper
	StakingKeeper   *stakingkeeper.Keeper
	FeeMarketKeeper feemarketkeeper.Keeper
	SlashingKeeper  slashingkeeper.Keeper
}

func NewChaosChainApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	_ servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *ChaosChainApp {
	var (
		appBuilder        *runtime.AppBuilder
		appCodec          codec.Codec
		legacyAmino       *codec.LegacyAmino
		txConfig          client.TxConfig
		interfaceRegistry codectypes.InterfaceRegistry
		bankKeeper        bankkeeper.BaseKeeper
		stakingKeeper     *stakingkeeper.Keeper
	)

	if err := depinject.Inject(
		depinject.Configs(
			appconfig.LoadYAML(appConfigYAML),
			depinject.Supply(logger),
		),
		&appBuilder,
		&appCodec,
		&legacyAmino,
		&txConfig,
		&interfaceRegistry,
		&bankKeeper,
		&stakingKeeper,
	); err != nil {
		panic(fmt.Errorf("wire application dependencies: %w", err))
	}

	runtimeApp := appBuilder.Build(db, baseAppOptions...)
	feeMarketKey := storetypes.NewKVStoreKey(feemarketmodule.ModuleName)
	slashingKey := storetypes.NewKVStoreKey(slashingmodule.ModuleName)
	if err := runtimeApp.RegisterStores(feeMarketKey, slashingKey); err != nil {
		panic(fmt.Errorf("register local module stores: %w", err))
	}

	feeMarketKeeper := feemarketkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(feeMarketKey))
	slashingKeeper := slashingkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(slashingKey))
	if err := runtimeApp.RegisterModules(
		feemarketmodule.NewAppModule(appCodec, feeMarketKeeper),
		slashingmodule.NewAppModule(appCodec, slashingKeeper),
	); err != nil {
		panic(fmt.Errorf("register local modules: %w", err))
	}

	runtimeApp.ModuleManager.SetOrderInitGenesis("auth", "bank", "staking", feemarketmodule.ModuleName, slashingmodule.ModuleName)
	runtimeApp.ModuleManager.SetOrderExportGenesis("auth", "bank", "staking", feemarketmodule.ModuleName, slashingmodule.ModuleName)
	runtimeApp.ModuleManager.SetOrderEndBlockers("staking", "bank", feemarketmodule.ModuleName)

	if err := runtimeApp.Load(loadLatest); err != nil {
		panic(fmt.Errorf("load application state: %w", err))
	}

	return &ChaosChainApp{
		App:               runtimeApp,
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		txConfig:          txConfig,
		interfaceRegistry: interfaceRegistry,
		BankKeeper:        bankKeeper,
		StakingKeeper:     stakingKeeper,
		FeeMarketKeeper:   feeMarketKeeper,
		SlashingKeeper:    slashingKeeper,
	}
}

func (app *ChaosChainApp) Name() string                    { return "ChaosChain" }
func (app *ChaosChainApp) LegacyAmino() *codec.LegacyAmino { return app.legacyAmino }
func (app *ChaosChainApp) AppCodec() codec.Codec           { return app.appCodec }
func (app *ChaosChainApp) InterfaceRegistry() codectypes.InterfaceRegistry {
	return app.interfaceRegistry
}
func (app *ChaosChainApp) TxConfig() client.TxConfig                    { return app.txConfig }
func (app *ChaosChainApp) LoadHeight(height int64) error                { return app.LoadVersion(height) }
func (app *ChaosChainApp) SimulationManager() *module.SimulationManager { return nil }

func (app *ChaosChainApp) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, state map[string]json.RawMessage) {
	if _, err := app.ModuleManager.InitGenesis(ctx, cdc, state); err != nil {
		panic(fmt.Errorf("initialize genesis: %w", err))
	}
}

func (app *ChaosChainApp) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) map[string]json.RawMessage {
	state, err := app.ModuleManager.ExportGenesis(ctx, cdc)
	if err != nil {
		panic(fmt.Errorf("export genesis: %w", err))
	}
	return state
}

func (app *ChaosChainApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	app.App.RegisterAPIRoutes(apiSvr, apiConfig)
}

func (app *ChaosChainApp) RegisterTxService(clientCtx client.Context) {
	app.App.RegisterTxService(clientCtx)
}

func (app *ChaosChainApp) RegisterTendermintService(clientCtx client.Context) {
	app.App.RegisterTendermintService(clientCtx)
}

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{Use: "chaoschaind", Short: "ChaosChain daemon (Layer 1 core)"}
	rootCmd.AddCommand(server.StatusCommand())
	return rootCmd
}

var _ servertypes.Application = (*ChaosChainApp)(nil)
