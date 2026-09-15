package app

import (
	"os"
	"path/filepath"

	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log/v2"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	feemarket "github.com/CH4rnel/ChaosChain/x/feemarket"
	feemarketkeeper "github.com/CH4rnel/ChaosChain/x/feemarket/keeper"
	slashing "github.com/CH4rnel/ChaosChain/x/slashing"
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

	// Module Keepers
	FeeMarketKeeper feemarketkeeper.Keeper
	SlashingKeeper  slashingkeeper.Keeper
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

// ProvideFeeMarketKeeper creates the feemarket keeper via depinject.
func ProvideFeeMarketKeeper(cdc codec.Codec, storeService store.KVStoreService) feemarketkeeper.Keeper {
	return feemarketkeeper.NewKeeper(cdc, storeService)
}

// ProvideFeeMarketModule creates the feemarket app module via depinject.
func ProvideFeeMarketModule(cdc codec.Codec, k feemarketkeeper.Keeper) feemarket.AppModule {
	return feemarket.NewAppModule(cdc, k)
}

// ProvideSlashingKeeper creates the slashing keeper via depinject.
func ProvideSlashingKeeper(cdc codec.Codec, storeService store.KVStoreService) slashingkeeper.Keeper {
	return slashingkeeper.NewKeeper(cdc, storeService)
}

// ProvideSlashingModule creates the slashing app module via depinject.
func ProvideSlashingModule(cdc codec.Codec, k slashingkeeper.Keeper) slashing.AppModule {
	return slashing.NewAppModule(cdc, k)
}

// WireApp uses depinject to build the application components declaratively.
// This is the core implementation of ADR-0002: avoiding manual wiring of keepers.
func WireApp() (*ChaosChainApp, error) {
	var (
		appInstance       ChaosChainApp
		cdc               codec.Codec
		interfaceRegistry codectypes.InterfaceRegistry
		feeMarketKeeper   feemarketkeeper.Keeper
		slashingKeeper    slashingkeeper.Keeper
	)

	err := depinject.Inject(
		depinject.Configs(
			depinject.Provide(
				ProvideInterfaceRegistry,
				ProvideCodec,
				ProvideFeeMarketKeeper,
				ProvideFeeMarketModule,
				ProvideSlashingKeeper,
				ProvideSlashingModule,
			),
		),
		&cdc,
		&interfaceRegistry,
		&feeMarketKeeper,
		&slashingKeeper,
	)

	if err != nil {
		return nil, err
	}

	appInstance.cdc = cdc
	appInstance.interfaceRegistry = interfaceRegistry
	appInstance.FeeMarketKeeper = feeMarketKeeper
	appInstance.SlashingKeeper = slashingKeeper

	db := dbm.NewMemDB()
	appInstance.BaseApp = baseapp.NewBaseApp("chaoschain", log.NewNopLogger(), db, nil)

	// In Cosmos SDK v0.54+, the EndBlocker signature strictly requires returning `(sdk.EndBlock, error)`.
	appInstance.SetEndBlocker(func(ctx sdk.Context) (sdk.EndBlock, error) {
		// We invoke the fee market update logic. Errors are handled within the keeper.
		_ = appInstance.FeeMarketKeeper.EndBlock(ctx)
		
		// Returning an empty EndBlock, as validator updates are still being processed.
		// using standard (staking) modules, rather than our custom ones.
		return sdk.EndBlock{}, nil
	})

	return &appInstance, nil
}