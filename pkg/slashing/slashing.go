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

func DefaultParams() Params {
	return Params{BaseSlash: 0.01, Kappa: 2}
}

func ValidateParams(p Params) error {
	if math.IsNaN(p.BaseSlash) || math.IsInf(p.BaseSlash, 0) {
		return errors.New("baseSlash must be finite")
	}
	if math.IsNaN(p.Kappa) || math.IsInf(p.Kappa, 0) {
		return errors.New("kappa must be finite")
	}
	if p.BaseSlash < 0 || p.BaseSlash > MaxBaseSlash {
		return errors.New("baseSlash must be in the range [0.0, 1.0]")
	}
	if p.Kappa < 0 || p.Kappa > MaxKappa {
		return errors.New("kappa must be in the range [0.0, MaxKappa]")
	}
	return nil
}

// CalculateSlashFraction computes the slashing percentage.
func CalculateSlashFraction(faultyShare float64, p Params) (float64, error) {
	if err := ValidateParams(p); err != nil {
		return 0, err
	}
	if math.IsNaN(faultyShare) || math.IsInf(faultyShare, 0) {
		return 0, errors.New("faultyShare must be finite")
	}
	if faultyShare < 0.0 || faultyShare > 1.0 {
		return 0.0, errors.New("faultyShare must be in the range [0.0, 1.0]")
	}

	correlationPenalty := p.Kappa * faultyShare * faultyShare

	slashFraction := p.BaseSlash + correlationPenalty

	if slashFraction > 1.0 {
		slashFraction = 1.0
	}

	if math.IsInf(slashFraction, 0) || math.IsNaN(slashFraction) {
		return 0.0, errors.New("calculated slashFraction is not finite")
	}

	return slashFraction, nil
}
