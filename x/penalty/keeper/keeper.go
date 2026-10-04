package keeper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	sdkmath "cosmossdk.io/math"
	"github.com/CH4rnel/ChaosChain/pkg/penalty"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	// ParamsKey defines the collections prefix for penalty parameters.
	ParamsKey = collections.NewPrefix(0)
	// PendingFaultsKey defines the collections prefix for queued faults.
	PendingFaultsKey = collections.NewPrefix(1)
)

type StakingKeeper interface {
	Slash(context.Context, sdk.ConsAddress, int64, int64, sdkmath.LegacyDec) (sdkmath.Int, error)
}

type CorrelatedFault struct {
	ConsensusAddress []byte            `json:"consensus_address"`
	InfractionHeight int64             `json:"infraction_height"`
	Power            int64             `json:"power"`
	FaultyShare      sdkmath.LegacyDec `json:"faulty_share"`
}

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
	staking      StakingKeeper

	Schema        collections.Schema
	Params        collections.Item[penalty.Params]
	PendingFaults collections.Item[[]CorrelatedFault]
}

// NewKeeper creates a new slashing Keeper instance.
func NewKeeper(cdc codec.BinaryCodec, storeService storetypes.KVStoreService, staking ...StakingKeeper) Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	var stakingKeeper StakingKeeper
	if len(staking) > 0 {
		stakingKeeper = staking[0]
	}

	k := Keeper{
		cdc:          cdc,
		storeService: storeService,
		staking:      stakingKeeper,
		Params: collections.NewItem(
			sb, ParamsKey, "params", jsonCodec[penalty.Params]{},
		),
		PendingFaults: collections.NewItem(
			sb, PendingFaultsKey, "pending_faults", jsonCodec[[]CorrelatedFault]{},
		),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

// GetParams retrieves penalty parameters.
func (k Keeper) GetParams(ctx context.Context) (penalty.Params, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {

		if errors.Is(err, collections.ErrNotFound) {
			return penalty.DefaultParams(), nil
		}
		return penalty.Params{}, err
	}
	if err := penalty.ValidateParams(params); err != nil {
		return penalty.Params{}, fmt.Errorf("validate stored penalty parameters: %w", err)
	}
	return params, nil
}

// SetParams saves penalty parameters.
func (k Keeper) SetParams(ctx context.Context, params penalty.Params) error {
	if err := penalty.ValidateParams(params); err != nil {
		return fmt.Errorf("validate penalty parameters: %w", err)
	}
	return k.Params.Set(ctx, params)
}

// Slash calculates the penalty fraction based on the faulty share and current params.
func (k Keeper) Slash(ctx context.Context, faultyShare float64) (float64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0.0, err
	}

	return penalty.CalculateSlashFraction(faultyShare, params)
}

func ValidateCorrelatedFault(fault CorrelatedFault) error {
	if err := sdk.VerifyAddressFormat(fault.ConsensusAddress); err != nil {
		return fmt.Errorf("invalid consensus address: %w", err)
	}
	if fault.InfractionHeight < 0 {
		return errors.New("infraction height cannot be negative")
	}
	if fault.Power <= 0 {
		return errors.New("validator power must be positive")
	}
	if fault.FaultyShare.IsNegative() || fault.FaultyShare.GT(sdkmath.LegacyOneDec()) {
		return errors.New("faulty share must be in the range [0, 1]")
	}
	return nil
}

func (k Keeper) QueueCorrelatedFault(ctx context.Context, fault CorrelatedFault) error {
	if err := ValidateCorrelatedFault(fault); err != nil {
		return err
	}
	faults, err := k.PendingFaults.Get(ctx)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return fmt.Errorf("load pending faults: %w", err)
	}
	faults = append(faults, fault)
	return k.PendingFaults.Set(ctx, faults)
}

func (k Keeper) SetPendingFaults(ctx context.Context, faults []CorrelatedFault) error {
	for _, fault := range faults {
		if err := ValidateCorrelatedFault(fault); err != nil {
			return fmt.Errorf("validate pending fault: %w", err)
		}
	}
	return k.PendingFaults.Set(ctx, faults)
}

func (k Keeper) GetPendingFaults(ctx context.Context) ([]CorrelatedFault, error) {
	faults, err := k.PendingFaults.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for _, fault := range faults {
		if err := ValidateCorrelatedFault(fault); err != nil {
			return nil, fmt.Errorf("validate pending fault: %w", err)
		}
	}
	return faults, nil
}

func (k Keeper) ApplyPendingFaults(ctx context.Context) error {
	if k.staking == nil {
		return errors.New("staking keeper is not configured")
	}
	faults, err := k.GetPendingFaults(ctx)
	if err != nil {
		return err
	}
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}
	for index, fault := range faults {
		fraction, err := penalty.CalculateSlashFractionDec(fault.FaultyShare, params)
		if err != nil {
			return fmt.Errorf("calculate correlated slash: %w", err)
		}
		if _, err := k.staking.Slash(ctx, sdk.ConsAddress(fault.ConsensusAddress), fault.InfractionHeight, fault.Power, fraction); err != nil {
			return fmt.Errorf("slash validator: %w", err)
		}
		if err := k.PendingFaults.Set(ctx, faults[index+1:]); err != nil {
			return fmt.Errorf("remove applied fault: %w", err)
		}
	}
	return nil
}
