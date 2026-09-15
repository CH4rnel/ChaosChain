// Package slashing implements the correlated slashing penalty logic
package slashing

import (
	"errors"
	"math"
)

// Maximum reasonable limits for parameters
const (
	MaxBaseSlash = 1.0   // 100% slashing — the absolute maximum.
	MaxKappa     = 100.0 // Extreme correlation coefficient
)

// Params holds the configuration for the slashing penalty calculation.
type Params struct {
	BaseSlash float64 `json:"base_slash"`
	Kappa     float64 `json:"kappa"`
}

// CalculateSlashFraction computes the slashing percentage.
func CalculateSlashFraction(faultyShare float64, p Params) (float64, error) {
	// 1. Basic validation
	if faultyShare < 0.0 || faultyShare > 1.0 {
		return 0.0, errors.New("faultyShare must be in the range [0.0, 1.0]")
	}
	if p.BaseSlash < 0.0 {
		return 0.0, errors.New("baseSlash cannot be negative")
	}
	if p.Kappa < 0.0 {
		return 0.0, errors.New("kappa cannot be negative")
	}

	// 2. Overflow protection
	if p.BaseSlash > MaxBaseSlash {
		return 0.0, errors.New("baseSlash exceeds maximum (1.0 = 100%)")
	}
	if p.Kappa > MaxKappa {
		return 0.0, errors.New("kappa exceeds maximum reasonable value")
	}

	// 3. Calculate correlation penalty
	correlationPenalty := p.Kappa * faultyShare * faultyShare

	// 4. Calculate total slash fraction
	slashFraction := p.BaseSlash + correlationPenalty

	// 5. Clamp to maximum 1.0 (100%)
	if slashFraction > 1.0 {
		slashFraction = 1.0
	}

	// 6. Final safety check
	if math.IsInf(slashFraction, 0) || math.IsNaN(slashFraction) {
		return 0.0, errors.New("calculated slashFraction is not finite")
	}

	return slashFraction, nil
}