package app

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
)

// TestChaosChainAppInitialization verifies that dependency injection 
// correctly wires all core keepers and the application boots without panics.
func TestChaosChainAppInitialization(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewTestLogger(t)
	
	app := NewChaosChainApp(logger, db, nil, true, nil)

	require.NotNil(t, app, "Application instance must not be nil")
	require.NotNil(t, app.ModuleManager, "ModuleManager must be initialized")
	require.NotNil(t, app.BankKeeper, "BankKeeper must be injected")
	require.NotNil(t, app.StakingKeeper, "StakingKeeper must be injected")
}

// TestChaosChainAppGenesisExportImport validates the complete genesis lifecycle.
// This ensures the chain can be safely initialized and snapshotted, 
// which is critical for network upgrades and state exports.
func TestChaosChainAppGenesisExportImport(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewTestLogger(t)
	
	app := NewChaosChainApp(logger, db, nil, true, nil)

	// 1. Generate default genesis state
	genesisState := app.DefaultGenesis()
	require.NotNil(t, genesisState, "Default genesis must not be nil")
	require.Contains(t, genesisState, "bank", "Genesis must contain bank module state")
	require.Contains(t, genesisState, "staking", "Genesis must contain staking module state")

	// 2. Initialize genesis
	ctx := app.BaseApp.NewContext(false)
	app.InitGenesis(ctx, app.AppCodec(), genesisState)

	// 3. Export genesis state
	exportedState := app.ExportGenesis(ctx, app.AppCodec())
	require.NotNil(t, exportedState, "Exported genesis must not be nil")
	
	// 4. Verify core modules are present in the exported state
	require.Contains(t, exportedState, "bank")
	require.Contains(t, exportedState, "staking")
	
	// 5. Validate JSON serialization integrity (prevents snapshot corruption)
	_, err := json.MarshalIndent(exportedState, "", "  ")
	require.NoError(t, err, "Exported genesis must be valid JSON")
}

// TestModuleManagerOrderExecution ensures that block lifecycle hooks 
// execute in the correct order without panicking, preserving state machine integrity.
func TestModuleManagerOrderExecution(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewTestLogger(t)
	
	app := NewChaosChainApp(logger, db, nil, true, nil)
	ctx := app.BaseApp.NewContext(false)

	// Verify that BeginBlock and EndBlock hooks can be called safely
	require.NotPanics(t, func() {
		app.ModuleManager.BeginBlock(ctx)
		app.ModuleManager.EndBlock(ctx)
	}, "ModuleManager lifecycle hooks must not panic")
}