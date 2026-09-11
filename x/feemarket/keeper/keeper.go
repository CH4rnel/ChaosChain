// x/feemarket/keeper/keeper.go
package keeper

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"github.com/CH4rnel/ChaosChain/pkg/feemarket"
	"github.com/cosmos/cosmos-sdk/codec"
)

var (
	// StateKey defines the collections prefix for the fee market state.
	StateKey = collections.NewPrefix(0)
	// ParamsKey defines the collections prefix for the fee market parameters.
	ParamsKey = collections.NewPrefix(1)
)

// jsonCodec implements the collections.ValueCodec interface for JSON serialization of plain Go structs.
type jsonCodec[T any] struct{}

func (c jsonCodec[T]) Encode(value T) ([]byte, error) {
	return json.Marshal(value)
}

func (c jsonCodec[T]) Decode(b []byte) (T, error) {
	var v T
	err := json.Unmarshal(b, &v)
	return v, err
}

func (c jsonCodec[T]) EncodeJSON(value T) ([]byte, error) {
	return json.Marshal(value)
}

func (c jsonCodec[T]) DecodeJSON(b []byte) (T, error) {
	var v T
	err := json.Unmarshal(b, &v)
	return v, err
}

func (c jsonCodec[T]) Stringify(value T) string {
	return fmt.Sprintf("%v", value)
}

func (c jsonCodec[T]) ValueType() string {
	return "json"
}

// Keeper maintains the link to storage and exposes getter/setter methods.
type Keeper struct {
	cdc          codec.BinaryCodec
	storeService storetypes.KVStoreService

	Schema collections.Schema
	State  collections.Item[feemarket.State]
	Params collections.Item[feemarket.Params]
}

// NewKeeper creates a new feemarket Keeper instance.
func NewKeeper(cdc codec.BinaryCodec, storeService storetypes.KVStoreService) Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		cdc:          cdc,
		storeService: storeService,
		State: collections.NewItem(
			sb, StateKey, "state", jsonCodec[feemarket.State]{},
		),
		Params: collections.NewItem(
			sb, ParamsKey, "params", jsonCodec[feemarket.Params]{},
		),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err) // Fatal error initializing storage schema
	}
	k.Schema = schema
	return k
}

// GetState retrieves the fee market state from the store.
func (k Keeper) GetState(ctx context.Context) (feemarket.State, error) {
	state, err := k.State.Get(ctx)
	if err != nil {
		if err == collections.ErrNotFound {
			return feemarket.State{BaseFee: 10.0, Acc: 0.0}, nil
		}
		return feemarket.State{}, err
	}
	return state, nil
}

// SetState saves the fee market state to the store.
func (k Keeper) SetState(ctx context.Context, state feemarket.State) error {
	return k.State.Set(ctx, state)
}

// GetParams retrieves the fee market parameters.
func (k Keeper) GetParams(ctx context.Context) (feemarket.Params, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		if err == collections.ErrNotFound {
			return feemarket.Params{Kp: 0.1, Ki: 0.01, AntiWindupLimit: 10.0}, nil
		}
		return feemarket.Params{}, err
	}
	return params, nil
}

// SetParams saves the fee market parameters.
func (k Keeper) SetParams(ctx context.Context, params feemarket.Params) error {
	return k.Params.Set(ctx, params)
}