package keeper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"github.com/CH4rnel/ChaosChain/pkg/slashing"
	"github.com/cosmos/cosmos-sdk/codec"
)

var (
	// ParamsKey defines the collections prefix for the slashing parameters.
	ParamsKey = collections.NewPrefix(0)
)

// jsonCodec implements the collections.ValueCodec interface for JSON serialization.
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
	Params collections.Item[slashing.Params]
}

// NewKeeper creates a new slashing Keeper instance.
func NewKeeper(cdc codec.BinaryCodec, storeService storetypes.KVStoreService) Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		cdc:          cdc,
		storeService: storeService,
		Params: collections.NewItem(
			sb, ParamsKey, "params", jsonCodec[slashing.Params]{},
		),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

// GetParams retrieves the slashing parameters.
func (k Keeper) GetParams(ctx context.Context) (slashing.Params, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {

		if errors.Is(err, collections.ErrNotFound) {
			return slashing.DefaultParams(), nil
		}
		return slashing.Params{}, err
	}
	return params, nil
}

// SetParams saves the slashing parameters.
func (k Keeper) SetParams(ctx context.Context, params slashing.Params) error {
	return k.Params.Set(ctx, params)
}

// Slash calculates the penalty fraction based on the faulty share and current params.
func (k Keeper) Slash(ctx context.Context, faultyShare float64) (float64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0.0, err
	}

	return slashing.CalculateSlashFraction(faultyShare, params)
}
