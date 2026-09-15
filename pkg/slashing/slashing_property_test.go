package slashing

import (
	"reflect"
	"testing"
	"testing/quick"
)

type realisticSlashingInputs struct {
	BaseSlash   float64
	Kappa       float64
	FaultyShare float64
}

func (r realisticSlashingInputs) Generate(rand *quick.Rand, size int) reflect.Value {
	return reflect.ValueOf(realisticSlashingInputs{
		BaseSlash:   rand.Float64() * 0.1,  // 0 - 10%
		Kappa:       rand.Float64() * 10.0, // 0 - 10.0
		FaultyShare: rand.Float64(),         // 0.0 - 1.0
	})
}

func TestProperty_SlashFractionBounded(t *testing.T) {
	f := func(inputs realisticSlashingInputs) bool {
		params := Params{BaseSlash: inputs.BaseSlash, Kappa: inputs.Kappa}
		result, err := CalculateSlashFraction(inputs.FaultyShare, params)
		if err != nil {
			return true
		}
		
		return result <= 1.0 && result >= 0.0
	}
	
	if err := quick.Check(f, &quick.Config{MaxCount: 10000}); err != nil {
		t.Errorf("Invariant violated: %v", err)
	}
}

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

func TestProperty_Monotonicity(t *testing.T) {
	type twoShareInputs struct {
		realisticSlashingInputs
		FaultyShare2 float64
	}
	
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
	
	generate := func(rand *quick.Rand, size int) reflect.Value {
		return reflect.ValueOf(twoShareInputs{
			realisticSlashingInputs: realisticSlashingInputs{
				BaseSlash:   rand.Float64() * 0.1,
				Kappa:       rand.Float64() * 10.0,
				FaultyShare: rand.Float64(),
			},
			FaultyShare2: rand.Float64(),
		})
	}
	
	if err := quick.Check(f, &quick.Config{MaxCount: 10000, Values: generate}); err != nil {
		t.Errorf("Invariant violated: %v", err)
	}
}

func TestProperty_QuadraticGrowth(t *testing.T) {
	f := func(inputs realisticSlashingInputs) bool {
		if inputs.FaultyShare <= 0 || inputs.FaultyShare > 0.5 || inputs.Kappa <= 0 {
			return true
		}
		
		params := Params{BaseSlash: inputs.BaseSlash, Kappa: inputs.Kappa}
		
		result1, err1 := CalculateSlashFraction(inputs.FaultyShare, params)
		if err1 != nil {
			return true
		}
		additionalPenalty1 := result1 - inputs.BaseSlash
		
		result2, err2 := CalculateSlashFraction(2*inputs.FaultyShare, params)
		if err2 != nil {
			return true
		}
		additionalPenalty2 := result2 - inputs.BaseSlash
		
		return additionalPenalty2 > 2*additionalPenalty1
	}
	
	if err := quick.Check(f, &quick.Config{MaxCount: 10000}); err != nil {
		t.Errorf("Invariant violated: %v", err)
	}
}

func TestProperty_ZeroFaultyShare(t *testing.T) {
	type baseInputs struct {
		BaseSlash float64
		Kappa     float64
	}
	
	f := func(inputs baseInputs) bool {
		params := Params{BaseSlash: inputs.BaseSlash, Kappa: inputs.Kappa}
		result, err := CalculateSlashFraction(0.0, params)
		if err != nil {
			return true
		}
		
		return result == inputs.BaseSlash
	}
	
	generate := func(rand *quick.Rand, size int) reflect.Value {
		return reflect.ValueOf(baseInputs{
			BaseSlash: rand.Float64() * 0.1,
			Kappa:     rand.Float64() * 10.0,
		})
	}
	
	if err := quick.Check(f, &quick.Config{MaxCount: 10000, Values: generate}); err != nil {
		t.Errorf("Invariant violated: %v", err)
	}
}