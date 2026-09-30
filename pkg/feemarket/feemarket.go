// Package feemarket implements a deterministic fixed-point dynamic fee market.
package feemarket

import (
	"errors"

	sdkmath "cosmossdk.io/math"
)

var (
	maxBaseFee    = sdkmath.LegacyNewDec(1_000_000_000_000_000_000)
	maxGasUsed    = sdkmath.LegacyNewDec(1_000_000_000_000_000)
	maxGasTarget  = sdkmath.LegacyNewDec(1_000_000_000_000_000)
	maxKp         = sdkmath.LegacyNewDec(100)
	maxKi         = sdkmath.LegacyNewDec(100)
	maxAntiWindup = sdkmath.LegacyNewDec(1_000_000)
	maxStep       = sdkmath.LegacyNewDecWithPrec(125, 3) // Limit changes to 12.5% per block.
	minBaseFee    = sdkmath.LegacySmallestDec()
	one           = sdkmath.LegacyOneDec()
	zero          = sdkmath.LegacyZeroDec()
)

// Params holds the configuration for the fixed-point PI regulator.
type Params struct {
	Kp              sdkmath.LegacyDec `json:"kp"`
	Ki              sdkmath.LegacyDec `json:"ki"`
	AntiWindupLimit sdkmath.LegacyDec `json:"anti_windup_limit"`
	GasTarget       sdkmath.LegacyDec `json:"gas_target"`
}

func DefaultParams() Params {
	return Params{
		Kp:              sdkmath.LegacyNewDecWithPrec(1, 1),
		Ki:              sdkmath.LegacyNewDecWithPrec(1, 2),
		AntiWindupLimit: sdkmath.LegacyNewDec(10),
		GasTarget:       sdkmath.LegacyNewDec(10_000_000),
	}
}

func decimalOrZero(value sdkmath.LegacyDec) sdkmath.LegacyDec {
	if value.IsNil() {
		return zero
	}
	return value
}

func ValidateParams(p Params) error {
	kp := decimalOrZero(p.Kp)
	ki := decimalOrZero(p.Ki)
	antiWindup := decimalOrZero(p.AntiWindupLimit)
	gasTarget := decimalOrZero(p.GasTarget)
	if gasTarget.IsNil() || !gasTarget.IsPositive() || gasTarget.GT(maxGasTarget) {
		return errors.New("gasTarget out of valid range (0, MaxGasTarget]")
	}
	if kp.IsNegative() || kp.GT(maxKp) {
		return errors.New("kp out of valid range [0, MaxKp]")
	}
	if ki.IsNegative() || ki.GT(maxKi) {
		return errors.New("ki out of valid range [0, MaxKi]")
	}
	if antiWindup.IsNegative() || antiWindup.GT(maxAntiWindup) {
		return errors.New("antiWindupLimit out of valid range [0, MaxAntiWindup]")
	}
	return nil
}

func ValidateState(state State) error {
	if state.BaseFee.IsNil() || !state.BaseFee.IsPositive() || state.BaseFee.GT(maxBaseFee) {
		return errors.New("baseFee out of valid range (0, MaxBaseFee]")
	}
	return nil
}

func ValidateStateWithParams(state State, params Params) error {
	if err := ValidateParams(params); err != nil {
		return err
	}
	if err := ValidateState(state); err != nil {
		return err
	}
	acc := decimalOrZero(state.Acc)
	limit := decimalOrZero(params.AntiWindupLimit)
	if acc.GT(limit) || acc.LT(limit.Neg()) {
		return errors.New("accumulator exceeds anti-windup limit")
	}
	return nil
}

// State represents the current fee market state.
type State struct {
	BaseFee sdkmath.LegacyDec `json:"base_fee"`
	Acc     sdkmath.LegacyDec `json:"acc"`
}

// Next computes the next fee market state with deterministic fixed-point math.
func Next(prev State, gasUsed sdkmath.LegacyDec, p Params) (State, error) {
	if err := ValidateParams(p); err != nil {
		return State{}, err
	}
	if err := ValidateStateWithParams(prev, p); err != nil {
		return State{}, err
	}
	if gasUsed.IsNil() || gasUsed.IsNegative() || gasUsed.GT(maxGasUsed) {
		return State{}, errors.New("gasUsed out of valid range [0, MaxGasUsed]")
	}

	gasTarget := decimalOrZero(p.GasTarget)
	e := gasUsed.Quo(gasTarget).Sub(one)
	acc := decimalOrZero(prev.Acc).Add(e)
	antiWindup := decimalOrZero(p.AntiWindupLimit)
	if acc.GT(antiWindup) {
		acc = antiWindup
	} else if acc.LT(antiWindup.Neg()) {
		acc = antiWindup.Neg()
	}

	step := decimalOrZero(p.Kp).Mul(e).Add(decimalOrZero(p.Ki).Mul(acc))
	if step.GT(maxStep) {
		step = maxStep
	} else if step.LT(maxStep.Neg()) {
		step = maxStep.Neg()
	}
	baseFeeNext := prev.BaseFee.Mul(one.Add(step))
	if baseFeeNext.LT(minBaseFee) {
		baseFeeNext = minBaseFee
	} else if baseFeeNext.GT(maxBaseFee) {
		baseFeeNext = maxBaseFee
	}

	next := State{BaseFee: baseFeeNext, Acc: acc}
	if err := ValidateStateWithParams(next, p); err != nil {
		return State{}, err
	}
	return next, nil
}
