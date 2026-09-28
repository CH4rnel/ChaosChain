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
	cmtcfg "github.com/cometbft/cometbft/config"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	clientflags "github.com/cosmos/cosmos-sdk/client/flags"
	clientkeys "github.com/cosmos/cosmos-sdk/client/keys"
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
	authmodule "github.com/cosmos/cosmos-sdk/x/auth"
	bankmodule "github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	genutilmodule "github.com/cosmos/cosmos-sdk/x/genutil"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	slashingmodule "github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingmodule "github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/spf13/cobra"

	feemarketmodule "github.com/CH4rnel/ChaosChain/x/feemarket"
	feemarketkeeper "github.com/CH4rnel/ChaosChain/x/feemarket/keeper"
	penaltymodule "github.com/CH4rnel/ChaosChain/x/penalty"
	penaltykeeper "github.com/CH4rnel/ChaosChain/x/penalty/keeper"

	_ "cosmossdk.io/api/cosmos/app/runtime/v1alpha1"
	_ "cosmossdk.io/api/cosmos/genutil/module/v1"
	_ "cosmossdk.io/api/cosmos/tx/config/v1"
	_ "github.com/cosmos/cosmos-sdk/x/auth"
	_ "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	_ "github.com/cosmos/cosmos-sdk/x/bank"
	_ "github.com/cosmos/cosmos-sdk/x/genutil"
	_ "github.com/cosmos/cosmos-sdk/x/slashing"
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
	SlashingKeeper  slashingkeeper.Keeper
	FeeMarketKeeper feemarketkeeper.Keeper
	PenaltyKeeper   penaltykeeper.Keeper
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
		slashingKeeper    slashingkeeper.Keeper
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
		&slashingKeeper,
	); err != nil {
		panic(fmt.Errorf("wire application dependencies: %w", err))
	}

	runtimeApp := appBuilder.Build(db, baseAppOptions...)
	feeMarketKey := storetypes.NewKVStoreKey(feemarketmodule.ModuleName)
	penaltyKey := storetypes.NewKVStoreKey(penaltymodule.ModuleName)
	if err := runtimeApp.RegisterStores(feeMarketKey, penaltyKey); err != nil {
		panic(fmt.Errorf("register local module stores: %w", err))
	}

	feeMarketKeeper := feemarketkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(feeMarketKey))
	penaltyKeeper := penaltykeeper.NewKeeper(appCodec, runtime.NewKVStoreService(penaltyKey))
	if err := runtimeApp.RegisterModules(
		feemarketmodule.NewAppModule(appCodec, feeMarketKeeper),
		penaltymodule.NewAppModule(appCodec, penaltyKeeper),
	); err != nil {
		panic(fmt.Errorf("register local modules: %w", err))
	}

	runtimeApp.ModuleManager.SetOrderInitGenesis("auth", "bank", "staking", "slashing", "genutil", feemarketmodule.ModuleName, penaltymodule.ModuleName)
	runtimeApp.ModuleManager.SetOrderExportGenesis("auth", "bank", "staking", "slashing", "genutil", feemarketmodule.ModuleName, penaltymodule.ModuleName)
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
		SlashingKeeper:    slashingKeeper,
		FeeMarketKeeper:   feeMarketKeeper,
		PenaltyKeeper:     penaltyKeeper,
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
	rootCmd.PersistentFlags().String(clientflags.FlagHome, DefaultNodeHome, "The application home directory")
	basics := ModuleBasics()
	var (
		appCodec          codec.Codec
		legacyAmino       *codec.LegacyAmino
		txConfig          client.TxConfig
		interfaceRegistry codectypes.InterfaceRegistry
	)
	if err := depinject.Inject(
		depinject.Configs(
			appconfig.LoadYAML(appConfigYAML),
			depinject.Supply(log.NewNopLogger()),
		),
		&appCodec,
		&legacyAmino,
		&txConfig,
		&interfaceRegistry,
	); err != nil {
		panic(fmt.Errorf("wire CLI encoding configuration: %w", err))
	}
	clientCtx := client.Context{}.
		WithCodec(appCodec).
		WithLegacyAmino(legacyAmino).
		WithInterfaceRegistry(interfaceRegistry).
		WithTxConfig(txConfig).
		WithHomeDir(DefaultNodeHome)
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if err := client.SetCmdClientContextHandler(clientCtx, cmd); err != nil {
			return err
		}
		return server.InterceptConfigsPreRunHandler(cmd, "", nil, cmtcfg.DefaultConfig())
	}
	rootCmd.AddCommand(clientkeys.Commands())
	rootCmd.AddCommand(genutilcli.InitCmd(basics, DefaultNodeHome))
	rootCmd.AddCommand(genutilcli.Commands(txConfig, basics, DefaultNodeHome))
	rootCmd.AddCommand(server.StartCmd(
		func(logger log.Logger, db dbm.DB, opts servertypes.AppOptions) servertypes.Application {
			return NewChaosChainApp(logger, db, nil, true, opts)
		},
		DefaultNodeHome,
	))
	rootCmd.AddCommand(server.StatusCommand())
	return rootCmd
}

func ModuleBasics() module.BasicManager {
	return module.NewBasicManager(
		authmodule.AppModuleBasic{},
		bankmodule.AppModuleBasic{},
		stakingmodule.AppModuleBasic{},
		slashingmodule.AppModuleBasic{},
		genutilmodule.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		feemarketmodule.AppModule{},
		penaltymodule.AppModule{},
	)
}

var _ servertypes.Application = (*ChaosChainApp)(nil)
