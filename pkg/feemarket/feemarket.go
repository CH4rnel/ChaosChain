// Package feemarket implements a PI-regulator based dynamic fee market
package feemarket

import (
	"errors"
	"math"
)

// Maximum reasonable limits for parameters (protection against float64 overflow)
const (
	MaxBaseFee    = 1e18 // An absurdly large commission (1 quintillion)
	MaxGasUsed    = 1e15 // Unrealistic volume of gas
	MaxGasTarget  = 1e15
	MaxKp         = 100.0 // Extreme proportional coefficient
	MaxKi         = 100.0 // Extreme integral coefficient
	MaxAntiWindup = 1e6   // Extreme battery limit
)

// Params holds the configuration for the PI-regulator.
type Params struct {
	Kp              float64 `json:"kp"`
	Ki              float64 `json:"ki"`
	AntiWindupLimit float64 `json:"anti_windup_limit"`
	GasTarget       float64 `json:"gas_target"`
}

func DefaultParams() Params {
	return Params{Kp: 0.1, Ki: 0.01, AntiWindupLimit: 10, GasTarget: 10_000_000}
}

func ValidateParams(p Params) error {
	values := []struct {
		name  string
		value float64
	}{
		{"kp", p.Kp},
		{"ki", p.Ki},
		{"antiWindupLimit", p.AntiWindupLimit},
		{"gasTarget", p.GasTarget},
	}
	for _, item := range values {
		if math.IsNaN(item.value) || math.IsInf(item.value, 0) {
			return errors.New(item.name + " must be finite")
		}
	}
	if p.GasTarget <= 0 || p.GasTarget > MaxGasTarget {
		return errors.New("gasTarget out of valid range (0, MaxGasTarget]")
	}
	if p.Kp < 0 || p.Kp > MaxKp {
		return errors.New("kp out of valid range [0, MaxKp]")
	}
	if p.Ki < 0 || p.Ki > MaxKi {
		return errors.New("ki out of valid range [0, MaxKi]")
	}
	if p.AntiWindupLimit < 0 || p.AntiWindupLimit > MaxAntiWindup {
		return errors.New("antiWindupLimit out of valid range [0, MaxAntiWindup]")
	}
	return nil
}

func ValidateState(state State) error {
	if math.IsNaN(state.BaseFee) || math.IsInf(state.BaseFee, 0) {
		return errors.New("baseFee must be finite")
	}
	if state.BaseFee <= 0 || state.BaseFee > MaxBaseFee {
		return errors.New("baseFee out of valid range (0, MaxBaseFee]")
	}
	if math.IsNaN(state.Acc) || math.IsInf(state.Acc, 0) {
		return errors.New("accumulator must be finite")
	}
	return nil
}

// State represents the current fee market state.
type State struct {
	BaseFee float64 `json:"base_fee"`
	Acc     float64 `json:"acc"`
}

// Next calculates the next block's fee market state based on the PI-regulator.
func Next(prev State, gasUsed float64, p Params) (State, error) {
	if err := ValidateParams(p); err != nil {
		return State{}, err
	}
	if err := ValidateState(prev); err != nil {
		return State{}, err
	}
	if math.IsNaN(gasUsed) || math.IsInf(gasUsed, 0) {
		return State{}, errors.New("gasUsed must be finite")
	}
	if gasUsed < 0 || gasUsed > MaxGasUsed {
		return State{}, errors.New("gasUsed out of valid range [0, MaxGasUsed]")
	}

	// 3. Calculate regulation error
	e := (gasUsed / p.GasTarget) - 1.0

	// 4. Update accumulated error
	acc := prev.Acc + e

	// 5. Anti-windup clamping
	if acc > p.AntiWindupLimit {
		acc = p.AntiWindupLimit
	} else if acc < -p.AntiWindupLimit {
		acc = -p.AntiWindupLimit
	}

	// 6. Calculate next base fee using exponential PI control
	exponent := p.Kp*e + p.Ki*acc

	// exp() overflow protection
	if exponent > 700 { // exp(700) ≈ 1e+304, close to MaxFloat64
		exponent = 700
	} else if exponent < -700 {
		exponent = -700
	}

	baseFeeNext := prev.BaseFee * math.Exp(exponent)

	// 7. Final safety check: ensure result is finite and positive
	if math.IsInf(baseFeeNext, 0) || math.IsNaN(baseFeeNext) || baseFeeNext <= 0 {
		return State{}, errors.New("calculated baseFee is not finite or positive")
	}

	return State{
		BaseFee: baseFeeNext,
		Acc:     acc,
	}, nil
}
