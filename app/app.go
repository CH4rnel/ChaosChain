// app/app.go
package app

import (
	"os"
	"path/filepath"

	"cosmossdk.io/depinject"
	"cosmossdk.io/log/v2" // Updated to v2 for compatibility with Cosmos SDK v0.54+.
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"
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
// In future steps, this will contain the ModuleManager and Keepers.
type ChaosChainApp struct {
	*baseapp.BaseApp
	cdc               codec.Codec
	interfaceRegistry codectypes.InterfaceRegistry
}

// NewRootCmd creates the root command for the ChaosChain daemon.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "chaoschaind",
		Short: "ChaosChain Daemon (Layer 1 Core)",
	}

	// Add basic server commands (status, etc.)
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

// WireApp uses depinject to build the application components declaratively.
// This is the core implementation of ADR-0002: avoiding manual wiring of keepers.
func WireApp() (*ChaosChainApp, error) {
	var (
		appInstance       ChaosChainApp
		cdc               codec.Codec
		interfaceRegistry codectypes.InterfaceRegistry
	)

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

	appInstance.cdc = cdc
	appInstance.interfaceRegistry = interfaceRegistry

	// Initialize BaseApp with an in-memory DB for skeleton validation.
	// In production, this will be replaced by the proper DB from server context.
	db := dbm.NewMemDB()
	appInstance.BaseApp = baseapp.NewBaseApp("chaoschain", log.NewNopLogger(), db, nil)

	return &appInstance, nil
}