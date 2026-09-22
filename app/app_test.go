package app

import (
	"testing"

	"cosmossdk.io/log/v2"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
)

type mockAppOptions struct{}

func (m mockAppOptions) Get(key string) any {
	return nil
}

func TestChaosChainAppInitialization(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewTestLogger(t)

	app := NewChaosChainApp(logger, db, nil, true, mockAppOptions{})

	require.NotNil(t, app, "Application instance must not be nil")
	require.NotNil(t, app.ModuleManager, "ModuleManager must be initialized")
	require.NotNil(t, app.BankKeeper, "BankKeeper must be injected")
	require.NotNil(t, app.StakingKeeper, "StakingKeeper must be injected")
}

func TestChaosChainAppInitGenesisRejectsEmptyValidatorSet(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewTestLogger(t)

	app := NewChaosChainApp(logger, db, nil, true, mockAppOptions{})

	genesisState := app.DefaultGenesis()
	require.NotNil(t, genesisState, "Default genesis must not be nil")
	require.Contains(t, genesisState, "bank", "Genesis must contain bank module state")
	require.Contains(t, genesisState, "staking", "Genesis must contain staking module state")

	ctx := app.BaseApp.NewNextBlockContext(cmtproto.Header{Height: 1})
	var panicValue any
	func() {
		defer func() { panicValue = recover() }()
		app.InitGenesis(ctx, app.AppCodec(), genesisState)
	}()

	require.ErrorContains(t, panicValue.(error), "initialize genesis: validator set is empty")
}

func TestModuleManagerOrderExecution(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewTestLogger(t)

	app := NewChaosChainApp(logger, db, nil, true, mockAppOptions{})
	ctx := app.BaseApp.NewNextBlockContext(cmtproto.Header{Height: 1})

	require.NotPanics(t, func() {
		app.ModuleManager.BeginBlock(ctx)
		app.ModuleManager.EndBlock(ctx)
	}, "ModuleManager lifecycle hooks must not panic")
}
