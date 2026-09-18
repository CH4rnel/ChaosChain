package app

import (
	"os"
	"path/filepath"

	"cosmossdk.io/depinject"
	"cosmossdk.io/log/v2"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/spf13/cobra"

	feemarketkeeper "github.com/CH4rnel/ChaosChain/x/feemarket/keeper"
	slashingkeeper "github.com/CH4rnel/ChaosChain/x/slashing/keeper"
)

// DefaultNodeHome defines the default home directory for the node.
var DefaultNodeHome string

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	DefaultNodeHome = filepath.Join(userHomeDir, ".chaoschain")
}

// ChaosChainApp extends baseapp.BaseApp and holds the application state.
type ChaosChainApp struct {
	*baseapp.BaseApp
	cdc               codec.Codec
	interfaceRegistry codectypes.InterfaceRegistry

	FeeMarketKeeper feemarketkeeper.Keeper
	SlashingKeeper  slashingkeeper.Keeper
	
	keys map[string]*storetypes.KVStoreKey
}

// NewRootCmd creates the root command for the ChaosChain daemon.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "chaoschaind",
		Short: "ChaosChain Daemon (Layer 1 Core)",
	}
	rootCmd.AddCommand(server.StatusCommand())
	return rootCmd
}

// ProvideInterfaceRegistry provides the interface registry via depinject.
func ProvideInterfaceRegistry() codectypes.InterfaceRegistry {
	return codectypes.NewInterfaceRegistry()
}

// ProvideCodec provides the protobuf codec via depinject.
func ProvideCodec(interfaceRegistry codectypes.InterfaceRegistry) codec.Codec {
	return codec.NewProtoCodec(interfaceRegistry)
}

// WireApp builds the application using canonical Cosmos SDK v0.54+ initialization patterns.
func WireApp() (*ChaosChainApp, error) {
	var (
		appInstance       ChaosChainApp
		cdc               codec.Codec
		interfaceRegistry codectypes.InterfaceRegistry
	)

	// 1. Initialization of the database and storage keys
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()
	
	keys := storetypes.NewKVStoreKeys("feemarket", "slashing")

	// 2. Initializing BaseApp
	appInstance.BaseApp = baseapp.NewBaseApp("chaoschain", logger, db, nil)
	
	// 3.Mounting storage in BaseApp
	appInstance.MountKVStores(keys)

	// 4. Setting up handlers before loading the version (avoids "panic: sealed BaseApp")
	appInstance.SetEndBlocker(func(ctx sdk.Context) (sdk.EndBlock, error) {
		_ = appInstance.FeeMarketKeeper.EndBlock(ctx)
		return sdk.EndBlock{}, nil
	})
	
	// 5. Loading the latest version of the repository (seals BaseApp)
	if err := appInstance.LoadLatestVersion(); err != nil {
		return nil, err
	}

	// 6. Injecting basic dependencies via depinject
	err := depinject.Inject(
		depinject.Configs(
			depinject.Provide(
				ProvideInterfaceRegistry,
				ProvideCodec,
			),
		),
		&cdc,
		&interfaceRegistry,
	)
	if err != nil {
		return nil, err
	}

	// 7. Manually creating Keepers and passing them the correct Store Services.
	feeMarketKeeper := feemarketkeeper.NewKeeper(cdc, runtime.NewKVStoreService(keys["feemarket"]))
	slashingKeeper := slashingkeeper.NewKeeper(cdc, runtime.NewKVStoreService(keys["slashing"]))

	// 8.Final assembly of the application structure
	appInstance.cdc = cdc
	appInstance.interfaceRegistry = interfaceRegistry
	appInstance.FeeMarketKeeper = feeMarketKeeper
	appInstance.SlashingKeeper = slashingKeeper
	appInstance.keys = keys

	return &appInstance, nil
}