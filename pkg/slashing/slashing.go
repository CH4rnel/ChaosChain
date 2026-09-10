// Package slashing implements the correlated slashing penalty logic
// as specified in ChaosChain Architecture Blueprint v2.0, Section 3.5.
// It is intentionally decoupled from the Cosmos SDK for pure domain testing.
package slashing

import (
	"errors"
	"math"
)

// Params holds the configuration for the slashing penalty calculation.
type Params struct {
	// BaseSlash is the minimum penalty for any protocol violation (e.g., 0.01 for 1%).
	BaseSlash float64
	// Kappa is the correlation penalty coefficient.
	// A higher value exponentially increases the penalty for mass, coordinated failures.
	Kappa float64
}

// CalculateSlashFraction computes the slashing percentage based on the formula:
// slash_fraction = base_slash + κ × faulty_share²
// The result is strictly clamped to a maximum of 1.0 (100% slash).
func CalculateSlashFraction(faultyShare float64, p Params) (float64, error) {
	// 1. Input validation (Guard clauses)
	if faultyShare < 0.0 || faultyShare > 1.0 {
		return 0.0, errors.New("faultyShare must be in the range [0.0, 1.0]")
	}
	if p.BaseSlash < 0.0 {
		return 0.0, errors.New("BaseSlash cannot be negative")
	}
	if p.Kappa < 0.0 {
		return 0.0, errors.New("Kappa cannot be negative")
	}

	// 2. Calculate correlation penalty
	correlationPenalty := p.Kappa * math.Pow(faultyShare, 2)

	// 3. Calculate total slash fraction
	slashFraction := p.BaseSlash + correlationPenalty

	// 4. Clamp to maximum 1.0 (100%) to prevent logical absurdities
	// (e.g., trying to slash 150% of a validator's stake).
	if slashFraction > 1.0 {
		slashFraction = 1.0
	}

	return slashFraction, nil
}