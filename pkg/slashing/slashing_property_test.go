package slashing

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

// realisticSlashingInputs implements the quick.Generator interface for generation. 
// economically realistic slashing parameters.
type realisticSlashingInputs struct {
	BaseSlash   float64
	Kappa       float64
	FaultyShare float64
}

func (r realisticSlashingInputs) Generate(rand *rand.Rand, size int) reflect.Value {
	return reflect.ValueOf(realisticSlashingInputs{
		BaseSlash:   rand.Float64() * 0.1,  // 0 - 10% base slashing
		Kappa:       rand.Float64() * 10.0, // 0 - 10.0 correlation coefficient
		FaultyShare: rand.Float64(),        // 0.0 - 1.0 failure rate
	})
}

// TestProperty_SlashFractionBounded verifies a critical invariant:
// The slashing fraction is strictly limited to the range [0.0, 1.0].
func TestProperty_SlashFractionBounded(t *testing.T) {
	f := func(inputs realisticSlashingInputs) bool {
		params := Params{BaseSlash: inputs.BaseSlash, Kappa: inputs.Kappa}
		result, err := CalculateSlashFraction(inputs.FaultyShare, params)
		if err != nil {
			return true // Skip invalid input data.
		}
		
		return result <= 1.0 && result >= 0.0
	}
	
	if err := quick.Check(f, &quick.Config{MaxCount: 10000}); err != nil {
		t.Errorf("Invariant violated: %v", err)
	}
}

// TestProperty_SlashAtLeastBase checks the invariant:
// The slashing share is always >= the base slashing.
func TestProperty_SlashAtLeastBase(t *testing.T) {
	f := func(inputs realisticSlashingInputs) bool {
		params := Params{BaseSlash: inputs.BaseSlash, Kappa: inputs.Kappa}
		result, err := CalculateSlashFraction(inputs.FaultyShare, params)
		if err != nil {
			return true
		}
		
		return result >= inputs.BaseSlash
	}
	
	if err := quick.Check(f, &quick.Config{MaxCount: 10000}); err != nil {
		t.Errorf("Invariant violated: %v", err)
	}
}

// twoShareInputs implements quick.Generator for tests requiring two failure rate values.
type twoShareInputs struct {
	realisticSlashingInputs
	FaultyShare2 float64
}

func (r twoShareInputs) Generate(rand *rand.Rand, size int) reflect.Value {
	return reflect.ValueOf(twoShareInputs{
		realisticSlashingInputs: realisticSlashingInputs{
			BaseSlash:   rand.Float64() * 0.1,
			Kappa:       rand.Float64() * 10.0,
			FaultyShare: rand.Float64(),
		},
		FaultyShare2: rand.Float64(),
	})
}

// TestProperty_Monotonicity checks the invariant:
// As the proportion of failures increases, the proportion of slashing increases monotonically.
func TestProperty_Monotonicity(t *testing.T) {
	f := func(inputs twoShareInputs) bool {
		if inputs.FaultyShare >= inputs.FaultyShare2 {
			return true
		}
		
		params := Params{BaseSlash: inputs.BaseSlash, Kappa: inputs.Kappa}
		result1, err1 := CalculateSlashFraction(inputs.FaultyShare, params)
		result2, err2 := CalculateSlashFraction(inputs.FaultyShare2, params)
		
		if err1 != nil || err2 != nil {
			return true
		}
		
		return result2 >= result1
	}
	
	if err := quick.Check(f, &quick.Config{MaxCount: 10000}); err != nil {
		t.Errorf("Invariant violated: %v", err)
	}
}

// TestProperty_QuadraticGrowth verifies a key economic invariant:
// Doubling the failure rate more than doubles the additional penalty (quadratic growth).
func TestProperty_QuadraticGrowth(t *testing.T) {
	f := func(inputs realisticSlashingInputs) bool {
		// We limit FaultyShare to 0.25 so that 2 * FaultyShare is <= 0.5.
		if inputs.FaultyShare <= 0 || inputs.FaultyShare > 0.25 || inputs.Kappa <= 0 {
			return true
		}
		
		params := Params{BaseSlash: inputs.BaseSlash, Kappa: inputs.Kappa}
		
		// CRITICAL FIX: Calculate the "raw" result BEFORE applying clamp.
		// If the doubled failure rate results in a value >= 1.0, it triggers. 
		// protective limit (maximum 100% slashing). In this case, we skip 
		// iteration, since clamp intentionally suppresses further growth for the sake of safety.
		rawResult2 := inputs.BaseSlash + inputs.Kappa*(2*inputs.FaultyShare)*(2*inputs.FaultyShare)
		if rawResult2 >= 1.0 {
			return true 
		}
		
		result1, err1 := CalculateSlashFraction(inputs.FaultyShare, params)
		if err1 != nil {
			return true
		}
		
		result2, err2 := CalculateSlashFraction(2*inputs.FaultyShare, params)
		if err2 != nil {
			return true
		}
		
		additionalPenalty1 := result1 - inputs.BaseSlash
		additionalPenalty2 := result2 - inputs.BaseSlash
		
		// INVARIANT: If the protective clamp did not trigger, faulty_share is doubled. 
		// it should more than double the additional penalty (due to the square).
		return additionalPenalty2 > 2*additionalPenalty1
	}
	
	if err := quick.Check(f, &quick.Config{MaxCount: 10000}); err != nil {
		t.Errorf("Invariant violated: %v", err)
	}
}

// baseInputs implements quick.Generator for tests without FaultyShare..
type baseInputs struct {
	BaseSlash float64
	Kappa     float64
}

func (r baseInputs) Generate(rand *rand.Rand, size int) reflect.Value {
	return reflect.ValueOf(baseInputs{
		BaseSlash: rand.Float64() * 0.1,
		Kappa:     rand.Float64() * 10.0,
	})
}

// TestProperty_ZeroFaultyShare checks a boundary condition:
// If the failure rate is 0, there is no additional penalty.
func TestProperty_ZeroFaultyShare(t *testing.T) {
	f := func(inputs baseInputs) bool {
		params := Params(inputs) // fix: direct type casting (The crazy S1016 Golang rule)
		result, err := CalculateSlashFraction(0.0, params)
		if err != nil {
			return true
		}
		
		return result == inputs.BaseSlash
	}
	
	if err := quick.Check(f, &quick.Config{MaxCount: 10000}); err != nil {
		t.Errorf("Invariant violated: %v", err)
	}
}