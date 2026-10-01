package app

import (
	"context"
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
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
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
	authcli "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankmodule "github.com/cosmos/cosmos-sdk/x/bank"
	bankcli "github.com/cosmos/cosmos-sdk/x/bank/client/cli"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensusmodule "github.com/cosmos/cosmos-sdk/x/consensus"
	distributionmodule "github.com/cosmos/cosmos-sdk/x/distribution"
	distributioncli "github.com/cosmos/cosmos-sdk/x/distribution/client/cli"
	genutilmodule "github.com/cosmos/cosmos-sdk/x/genutil"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govmodule "github.com/cosmos/cosmos-sdk/x/gov"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	mintmodule "github.com/cosmos/cosmos-sdk/x/mint"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	slashingmodule "github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingmodule "github.com/cosmos/cosmos-sdk/x/staking"
	stakingcli "github.com/cosmos/cosmos-sdk/x/staking/client/cli"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	upgrademodule "github.com/cosmos/cosmos-sdk/x/upgrade"
	"github.com/spf13/cobra"

	feemarketmodule "github.com/CH4rnel/ChaosChain/x/feemarket"
	feemarketkeeper "github.com/CH4rnel/ChaosChain/x/feemarket/keeper"
	penaltymodule "github.com/CH4rnel/ChaosChain/x/penalty"
	penaltykeeper "github.com/CH4rnel/ChaosChain/x/penalty/keeper"

	_ "cosmossdk.io/api/cosmos/app/runtime/v1alpha1"
	_ "cosmossdk.io/api/cosmos/consensus/module/v1"
	_ "cosmossdk.io/api/cosmos/distribution/module/v1"
	_ "cosmossdk.io/api/cosmos/genutil/module/v1"
	_ "cosmossdk.io/api/cosmos/gov/module/v1"
	_ "cosmossdk.io/api/cosmos/tx/config/v1"
	_ "cosmossdk.io/api/cosmos/upgrade/module/v1"
	_ "github.com/cosmos/cosmos-sdk/x/auth"
	_ "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	_ "github.com/cosmos/cosmos-sdk/x/bank"
	_ "github.com/cosmos/cosmos-sdk/x/consensus"
	_ "github.com/cosmos/cosmos-sdk/x/distribution"
	_ "github.com/cosmos/cosmos-sdk/x/genutil"
	_ "github.com/cosmos/cosmos-sdk/x/gov"
	_ "github.com/cosmos/cosmos-sdk/x/slashing"
	_ "github.com/cosmos/cosmos-sdk/x/staking"
	_ "github.com/cosmos/cosmos-sdk/x/upgrade"
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
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *ChaosChainApp {
	var (
		appBuilder        *runtime.AppBuilder
		appCodec          codec.Codec
		legacyAmino       *codec.LegacyAmino
		txConfig          client.TxConfig
		interfaceRegistry codectypes.InterfaceRegistry
		accountKeeper     authkeeper.AccountKeeper
		bankKeeper        bankkeeper.BaseKeeper
		stakingKeeper     *stakingkeeper.Keeper
		slashingKeeper    slashingkeeper.Keeper
	)

	if err := depinject.Inject(
		depinject.Configs(
			appconfig.LoadYAML(appConfigYAML),
			depinject.Supply(logger),
			depinject.Supply(appOpts),
		),
		&appBuilder,
		&appCodec,
		&legacyAmino,
		&txConfig,
		&interfaceRegistry,
		&accountKeeper,
		&bankKeeper,
		&stakingKeeper,
		&slashingKeeper,
	); err != nil {
		panic(fmt.Errorf("wire application dependencies: %w", err))
	}

	runtimeApp := appBuilder.Build(db, baseAppOptions...)
	feeMarketKey := storetypes.NewKVStoreKey(feemarketmodule.ModuleName)
	penaltyKey := storetypes.NewKVStoreKey(penaltymodule.ModuleName)
	mintKey := storetypes.NewKVStoreKey("mint")
	if err := runtimeApp.RegisterStores(feeMarketKey, penaltyKey, mintKey); err != nil {
		panic(fmt.Errorf("register local module stores: %w", err))
	}

	feeMarketKeeper := feemarketkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(feeMarketKey))
	penaltyKeeper := penaltykeeper.NewKeeper(appCodec, runtime.NewKVStoreService(penaltyKey))
	mintKeeper := mintkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(mintKey),
		stakingKeeper,
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress("gov").String(),
	)
	if err := runtimeApp.RegisterModules(
		mintmodule.NewAppModule(appCodec, mintKeeper, accountKeeper, nil, nil),
		feemarketmodule.NewAppModule(appCodec, feeMarketKeeper),
		penaltymodule.NewAppModule(appCodec, penaltyKeeper),
	); err != nil {
		panic(fmt.Errorf("register local modules: %w", err))
	}

	runtimeApp.ModuleManager.SetOrderBeginBlockers("mint", "distribution", "slashing", "staking", "bank")
	runtimeApp.ModuleManager.SetOrderInitGenesis("auth", "bank", "staking", "mint", "distribution", "slashing", "gov", "upgrade", "genutil", "consensus", feemarketmodule.ModuleName, penaltymodule.ModuleName)
	runtimeApp.ModuleManager.SetOrderExportGenesis("auth", "bank", "staking", "mint", "distribution", "slashing", "gov", "upgrade", "genutil", "consensus", feemarketmodule.ModuleName, penaltymodule.ModuleName)
	runtimeApp.ModuleManager.SetOrderEndBlockers("staking", "bank", "gov", feemarketmodule.ModuleName)

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
		if cmd.Context().Value(server.ServerContextKey) == nil {
			cmd.SetContext(context.WithValue(cmd.Context(), server.ServerContextKey, server.NewDefaultContext()))
		}
		if err := client.SetCmdClientContextHandler(clientCtx, cmd); err != nil {
			return err
		}
		if err := server.InterceptConfigsPreRunHandler(cmd, config.DefaultConfigTemplate, config.DefaultConfig(), cmtcfg.DefaultConfig()); err != nil {
			return err
		}
		serverCtx := server.GetServerContextFromCmd(cmd)
		if !serverCtx.Viper.IsSet(server.FlagPruning) {
			serverCtx.Viper.Set(server.FlagPruning, config.DefaultConfig().Pruning)
		}
		return nil
	}
	rootCmd.AddCommand(clientkeys.Commands())
	rootCmd.AddCommand(genutilcli.InitCmd(basics, DefaultNodeHome))
	rootCmd.AddCommand(genutilcli.Commands(txConfig, basics, DefaultNodeHome))
	rootCmd.AddCommand(server.StartCmd(newServerApp, DefaultNodeHome))
	nativeStartCmd := server.StartCmdWithOptions(newServerApp, DefaultNodeHome, server.StartCmdOptions{
		StartCommandHandler: func(serverCtx *server.Context, _ client.Context, appCreator servertypes.AppCreator, withCometBFT bool, opts server.StartCmdOptions) error {
			if !withCometBFT {
				return fmt.Errorf("direct CometBFT startup requires in-process consensus")
			}
			return runNativeCometNode(serverCtx, appCreator, opts)
		},
	})
	nativeStartCmd.Use = "start-native"
	rootCmd.AddCommand(nativeStartCmd)
	rootCmd.AddCommand(server.StatusCommand())
	rootCmd.AddCommand(newTxCommand())
	rootCmd.AddCommand(newQueryCommand())
	return rootCmd
}

func newTxCommand() *cobra.Command {
	accountCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	validatorCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())
	txCmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	txCmd.AddCommand(
		authcli.GetSignCommand(),
		authcli.GetSignBatchCommand(),
		authcli.GetMultiSignCommand(),
		authcli.GetMultiSignBatchCmd(),
		authcli.GetBroadcastCommand(),
		authcli.GetEncodeCommand(),
		authcli.GetValidateSignaturesCommand(),
		bankcli.NewTxCmd(accountCodec),
		stakingcli.NewTxCmd(validatorCodec, accountCodec),
		govcli.NewTxCmd(nil),
		distributioncli.NewTxCmd(validatorCodec, accountCodec),
	)
	return txCmd
}

func newQueryCommand() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:     "query",
		Aliases: []string{"q"},
		Short:   "Querying subcommands",
	}
	queryCmd.AddCommand(
		authcli.QueryTxCmd(),
		authcli.QueryTxsByEventsCmd(),
		newBankQueryCommand(),
	)
	return queryCmd
}

func newBankQueryCommand() *cobra.Command {
	bankCmd := &cobra.Command{
		Use:                        "bank",
		Short:                      "Bank query subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	balancesCmd := &cobra.Command{
		Use:   "balances [address]",
		Short: "Query all balances for an account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := banktypes.NewQueryClient(clientCtx)
			response, err := queryClient.AllBalances(cmd.Context(), &banktypes.QueryAllBalancesRequest{Address: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(response)
		},
	}
	clientflags.AddQueryFlagsToCmd(balancesCmd)
	bankCmd.AddCommand(balancesCmd)
	return bankCmd
}

func newServerApp(logger log.Logger, db dbm.DB, opts servertypes.AppOptions) servertypes.Application {
	return NewChaosChainApp(logger, db, nil, true, opts, server.DefaultBaseappOptions(opts)...)
}

func ModuleBasics() module.BasicManager {
	return module.NewBasicManager(
		authmodule.AppModuleBasic{},
		bankmodule.AppModuleBasic{},
		consensusmodule.AppModuleBasic{},
		stakingmodule.AppModuleBasic{},
		mintmodule.AppModuleBasic{},
		distributionmodule.AppModuleBasic{},
		govmodule.NewAppModuleBasic(nil),
		upgrademodule.AppModuleBasic{},
		slashingmodule.AppModuleBasic{},
		genutilmodule.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		feemarketmodule.AppModule{},
		penaltymodule.AppModule{},
	)
}

var _ servertypes.Application = (*ChaosChainApp)(nil)
