// x/feemarket/module.go
package feemarket

import (
	"context"

	"cosmossdk.io/core/appmodule"
	"github.com/CH4rnel/ChaosChain/x/feemarket/keeper"
	"github.com/cosmos/cosmos-sdk/codec"
)

var (
	_ appmodule.AppModule     = AppModule{}
	_ appmodule.HasEndBlocker = AppModule{}
)

// AppModule implements the appmodule.AppModule interface for the feemarket module.
type AppModule struct {
	cdc    codec.Codec
	keeper keeper.Keeper
}

// NewAppModule creates a new AppModule object.
func NewAppModule(cdc codec.Codec, keeper keeper.Keeper) AppModule {
	return AppModule{
		cdc:    cdc,
		keeper: keeper,
	}
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModule) IsAppModule() {}

// Name returns the module's name.
func (AppModule) Name() string { return "feemarket" }

// EndBlock executes the ABCI EndBlock logic for the feemarket module.
func (am AppModule) EndBlock(ctx context.Context) error {
	return am.keeper.EndBlock(ctx)
}