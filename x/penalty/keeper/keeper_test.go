package keeper

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/log/v2"
	sdkmath "cosmossdk.io/math"
	domain "github.com/CH4rnel/ChaosChain/pkg/penalty"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func newTestKeeper(t *testing.T) (sdk.Context, Keeper) {
	t.Helper()

	key := storetypes.NewKVStoreKey("penalty")
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_penalty"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	return ctx, NewKeeper(cdc, runtime.NewKVStoreService(key))
}

type stakingCall struct {
	address  sdk.ConsAddress
	height   int64
	power    int64
	fraction sdkmath.LegacyDec
}

type stakingStub struct {
	calls []stakingCall
	err   error
}

func (s *stakingStub) Slash(_ context.Context, address sdk.ConsAddress, height, power int64, fraction sdkmath.LegacyDec) (sdkmath.Int, error) {
	if s.err != nil {
		return sdkmath.Int{}, s.err
	}
	s.calls = append(s.calls, stakingCall{address: address, height: height, power: power, fraction: fraction})
	return sdkmath.NewInt(1), nil
}

func newTestKeeperWithStaking(t *testing.T, staking StakingKeeper) (sdk.Context, Keeper) {
	t.Helper()
	key := storetypes.NewKVStoreKey("penalty")
	testContext := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_penalty"))
	ctx := testContext.Ctx.WithLogger(log.NewNopLogger())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	return ctx, NewKeeper(cdc, runtime.NewKVStoreService(key), staking)
}

func TestJSONCodecSupportsCollectionsSerializationContracts(t *testing.T) {
	valueCodec := jsonCodec[domain.Params]{}
	want := domain.DefaultParams()

	encoded, err := valueCodec.EncodeJSON(want)
	require.NoError(t, err)
	got, err := valueCodec.DecodeJSON(encoded)
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.NotEmpty(t, valueCodec.Stringify(want))
	require.Equal(t, "json", valueCodec.ValueType())
}

func TestJSONCodecRejectsMalformedJSON(t *testing.T) {
	valueCodec := jsonCodec[domain.Params]{}

	_, err := valueCodec.DecodeJSON([]byte("{"))
	require.Error(t, err)
}

func TestGetParamsReturnsDefaultWhenParamsAreMissing(t *testing.T) {
	ctx, k := newTestKeeper(t)

	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, domain.DefaultParams(), params)
}

func TestKeeperPersistsParamsAndAppliesCorrelatedSlashFormula(t *testing.T) {
	ctx, k := newTestKeeper(t)
	params := domain.Params{BaseSlash: 0.05, Kappa: 2}
	require.NoError(t, k.SetParams(ctx, params))

	gotParams, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, params, gotParams)

	fraction, err := k.Slash(ctx, 0.3)
	require.NoError(t, err)
	require.InDelta(t, 0.23, fraction, 1e-12)
}

func TestSlashRejectsInvalidPersistedPenaltyParams(t *testing.T) {
	ctx, k := newTestKeeper(t)
	params := domain.DefaultParams()
	params.Kappa = -1
	require.NoError(t, k.Params.Set(ctx, params))

	_, err := k.Slash(ctx, 0.3)
	require.ErrorContains(t, err, "validate stored penalty parameters")
}

func TestSetParamsRejectsInvalidConfigurationWithoutPersistingIt(t *testing.T) {
	ctx, k := newTestKeeper(t)
	params := domain.DefaultParams()
	params.BaseSlash = -0.01
	require.ErrorContains(t, k.SetParams(ctx, params), "validate penalty parameters")
	_, err := k.Params.Get(ctx)
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func TestApplyPendingFaultsSlashesStakingAndConsumesQueue(t *testing.T) {
	staking := &stakingStub{}
	ctx, k := newTestKeeperWithStaking(t, staking)
	fault := CorrelatedFault{
		ConsensusAddress: bytes.Repeat([]byte{0x2a}, 20),
		InfractionHeight: 7,
		Power:            11,
		FaultyShare:      sdkmath.LegacyMustNewDecFromStr("0.3"),
	}

	require.NoError(t, k.QueueCorrelatedFault(ctx, fault))
	require.NoError(t, k.ApplyPendingFaults(ctx))
	require.Len(t, staking.calls, 1)
	require.Equal(t, sdk.ConsAddress(fault.ConsensusAddress), staking.calls[0].address)
	require.Equal(t, int64(7), staking.calls[0].height)
	require.Equal(t, int64(11), staking.calls[0].power)
	require.Equal(t, "0.190000000000000000", staking.calls[0].fraction.String())
	pending, err := k.GetPendingFaults(ctx)
	require.NoError(t, err)
	require.Empty(t, pending)
}

func TestApplyPendingFaultsKeepsRejectedSlashQueued(t *testing.T) {
	staking := &stakingStub{err: errors.New("staking refused slash")}
	ctx, k := newTestKeeperWithStaking(t, staking)
	fault := CorrelatedFault{
		ConsensusAddress: bytes.Repeat([]byte{0x2a}, 20),
		InfractionHeight: 7,
		Power:            11,
		FaultyShare:      sdkmath.LegacyMustNewDecFromStr("0.3"),
	}

	require.NoError(t, k.QueueCorrelatedFault(ctx, fault))
	require.ErrorContains(t, k.ApplyPendingFaults(ctx), "staking refused slash")
	pending, err := k.GetPendingFaults(ctx)
	require.NoError(t, err)
	require.Equal(t, []CorrelatedFault{fault}, pending)
}

func TestQueueCorrelatedFaultRejectsInvalidInput(t *testing.T) {
	ctx, k := newTestKeeper(t)
	fault := CorrelatedFault{
		ConsensusAddress: bytes.Repeat([]byte{0x2a}, 20),
		InfractionHeight: 7,
		Power:            0,
		FaultyShare:      sdkmath.LegacyMustNewDecFromStr("0.3"),
	}

	require.ErrorContains(t, k.QueueCorrelatedFault(ctx, fault), "power must be positive")
}
