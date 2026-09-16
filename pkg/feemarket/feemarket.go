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
}

// State represents the current fee market state.
type State struct {
	BaseFee float64 `json:"base_fee"`
	Acc     float64 `json:"acc"`
}

// Next calculates the next block's fee market state based on the PI-regulator.
func Next(prev State, gasUsed, gasTarget float64, p Params) (State, error) {
	// 1. Basic validation (sign and zero checks)
	if gasTarget <= 0 {
		return State{}, errors.New("gasTarget must be strictly greater than 0")
	}
	if prev.BaseFee <= 0 {
		return State{}, errors.New("baseFee must be strictly greater than 0")
	}
	if gasUsed < 0 {
		return State{}, errors.New("gasUsed cannot be negative")
	}

	// 2. Overflow protection
	if prev.BaseFee > MaxBaseFee {
		return State{}, errors.New("baseFee exceeds maximum reasonable value")
	}
	if gasUsed > MaxGasUsed {
		return State{}, errors.New("gasUsed exceeds maximum reasonable value")
	}
	if gasTarget > MaxGasTarget {
		return State{}, errors.New("gasTarget exceeds maximum reasonable value")
	}
	if p.Kp > MaxKp || p.Kp < 0 {
		return State{}, errors.New("kp out of valid range [0, MaxKp]") // fix: lowercase letter (The crazy ST1005 Golang rule)
	}
	if p.Ki > MaxKi || p.Ki < 0 {
		return State{}, errors.New("ki out of valid range [0, MaxKi]") // fix: lowercase letter (The crazy ST1005 Golang rule)
	}
	if p.AntiWindupLimit > MaxAntiWindup || p.AntiWindupLimit < 0 {
		return State{}, errors.New("antiWindupLimit out of valid range [0, MaxAntiWindup]")
	}

	// 3. Calculate regulation error
	e := (gasUsed / gasTarget) - 1.0

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