// Package feemarket implements a PI-regulator based dynamic fee market
// as specified in ChaosChain Architecture Blueprint v2.0, Section 3.4.
// It is intentionally decoupled from the Cosmos SDK for pure domain testing.
package feemarket

import (
	"errors"
	"math"
)

// Params holds the configuration for the PI-regulator.
type Params struct {
	// Kp is the proportional gain coefficient.
	Kp float64
	// Ki is the integral gain coefficient.
	Ki float64
	// AntiWindupLimit prevents the integral term from growing unbounded
	// during sustained overload/underload, ensuring the system can
	// recover and adjust fees downward when load normalizes.
	AntiWindupLimit float64
}

// State represents the current fee market state.
type State struct {
	// BaseFee is the current base fee per gas unit.
	BaseFee float64
	// Acc is the accumulated integral error term.
	Acc float64
}

// Next calculates the next block's fee market state based on the
// PI-regulator formula: baseFee_next = baseFee_prev * exp(Kp*e + Ki*acc).
func Next(prev State, gasUsed, gasTarget float64, p Params) (State, error) {
	// 1. Input validation (Guard clauses for KISS/SOLID)
	if gasTarget <= 0 {
		return State{}, errors.New("gasTarget must be strictly greater than 0")
	}
	if prev.BaseFee <= 0 {
		return State{}, errors.New("baseFee must be strictly greater than 0")
	}
	if gasUsed < 0 {
		return State{}, errors.New("gasUsed cannot be negative")
	}

	// 2. Calculate regulation error
	e := (gasUsed / gasTarget) - 1.0

	// 3. Update accumulated error
	acc := prev.Acc + e

	// 4. Anti-windup clamping (Refactor step)
	// Prevents integral windup, ensuring system responsiveness after saturation.
	if acc > p.AntiWindupLimit {
		acc = p.AntiWindupLimit
	} else if acc < -p.AntiWindupLimit {
		acc = -p.AntiWindupLimit
	}

	// 5. Calculate next base fee using exponential PI control
	baseFeeNext := prev.BaseFee * math.Exp(p.Kp*e+p.Ki*acc)

	return State{
		BaseFee: baseFeeNext,
		Acc:     acc,
	}, nil
}