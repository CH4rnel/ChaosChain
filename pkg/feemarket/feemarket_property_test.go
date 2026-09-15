package feemarket

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

// realisticInputs implements the quick.Generator interface
type realisticInputs struct {
	BaseFee    float64
	GasUsed    float64
	GasTarget  float64
	Kp         float64
	Ki         float64
	AntiWindup float64
}

func (r realisticInputs) Generate(rand *rand.Rand, size int) reflect.Value {
	return reflect.ValueOf(realisticInputs{
		BaseFee:    rand.Float64() * 1000,               // 0 - 1000
		GasUsed:    rand.Float64() * 20000000,           // 0 - 20M gas
		GasTarget:  10000000 + rand.Float64()*10000000,  // 10M - 20M gas
		Kp:         rand.Float64() * 2.0,                // 0 - 2.0
		Ki:         rand.Float64() * 0.5,                // 0 - 0.5
		AntiWindup: 1.0 + rand.Float64()*9.0,            // 1.0 - 10.0
	})
}

func TestProperty_BaseFeeAlwaysPositive(t *testing.T) {
	f := func(inputs realisticInputs) bool {
		prevState := State{BaseFee: inputs.BaseFee, Acc: 0.0}
		params := Params{Kp: inputs.Kp, Ki: inputs.Ki, AntiWindupLimit: inputs.AntiWindup}
		
		nextState, err := Next(prevState, inputs.GasUsed, inputs.GasTarget, params)
		if err != nil {
			return true // Skip invalid inputs
		}
		
		return nextState.BaseFee > 0
	}
	
	if err := quick.Check(f, &quick.Config{MaxCount: 10000}); err != nil {
		t.Errorf("Invariant violated: %v", err)
	}
}

// inputsWithAcc implements the quick.Generator interface
type inputsWithAcc struct {
	realisticInputs
	PrevAcc float64
}

func (r inputsWithAcc) Generate(rand *rand.Rand, size int) reflect.Value {
	return reflect.ValueOf(inputsWithAcc{
		realisticInputs: realisticInputs{
			BaseFee:    rand.Float64() * 1000,
			GasUsed:    rand.Float64() * 20000000,
			GasTarget:  10000000 + rand.Float64()*10000000,
			Kp:         rand.Float64() * 2.0,
			Ki:         rand.Float64() * 0.5,
			AntiWindup: 1.0 + rand.Float64()*9.0,
		},
		PrevAcc: (rand.Float64() - 0.5) * 20.0, // -10 to +10
	})
}

func TestProperty_AccumulatorBounded(t *testing.T) {
	f := func(inputs inputsWithAcc) bool {
		prevState := State{BaseFee: inputs.BaseFee, Acc: inputs.PrevAcc}
		params := Params{Kp: inputs.Kp, Ki: inputs.Ki, AntiWindupLimit: inputs.AntiWindup}
		
		nextState, err := Next(prevState, inputs.GasUsed, inputs.GasTarget, params)
		if err != nil {
			return true
		}
		
		return nextState.Acc >= -inputs.AntiWindup && nextState.Acc <= inputs.AntiWindup
	}
	
	if err := quick.Check(f, &quick.Config{MaxCount: 10000}); err != nil {
		t.Errorf("Invariant violated: %v", err)
	}
}

func TestProperty_MonotonicityOnOverload(t *testing.T) {
	f := func(inputs realisticInputs) bool {
		if inputs.GasUsed <= inputs.GasTarget {
			return true
		}
		
		prevState := State{BaseFee: inputs.BaseFee, Acc: 0.0}
		params := Params{Kp: inputs.Kp, Ki: inputs.Ki, AntiWindupLimit: inputs.AntiWindup}
		
		nextState, err := Next(prevState, inputs.GasUsed, inputs.GasTarget, params)
		if err != nil {
			return true
		}
		
		return nextState.BaseFee > prevState.BaseFee
	}
	
	if err := quick.Check(f, &quick.Config{MaxCount: 10000}); err != nil {
		t.Errorf("Invariant violated: %v", err)
	}
}

func TestProperty_MonotonicityOnUnderload(t *testing.T) {
	f := func(inputs realisticInputs) bool {
		if inputs.GasUsed >= inputs.GasTarget || inputs.GasUsed < 0 {
			return true
		}
		
		prevState := State{BaseFee: inputs.BaseFee, Acc: 0.0}
		params := Params{Kp: inputs.Kp, Ki: inputs.Ki, AntiWindupLimit: inputs.AntiWindup}
		
		nextState, err := Next(prevState, inputs.GasUsed, inputs.GasTarget, params)
		if err != nil {
			return true
		}
		
		return nextState.BaseFee < prevState.BaseFee
	}
	
	if err := quick.Check(f, &quick.Config{MaxCount: 10000}); err != nil {
		t.Errorf("Invariant violated: %v", err)
	}
}

// inputsWithAccForDeterminism implements the quick.Generator interface
type inputsWithAccForDeterminism struct {
	realisticInputs
	Acc float64
}

func (r inputsWithAccForDeterminism) Generate(rand *rand.Rand, size int) reflect.Value {
	return reflect.ValueOf(inputsWithAccForDeterminism{
		realisticInputs: realisticInputs{
			BaseFee:    rand.Float64() * 1000,
			GasUsed:    rand.Float64() * 20000000,
			GasTarget:  10000000 + rand.Float64()*10000000,
			Kp:         rand.Float64() * 2.0,
			Ki:         rand.Float64() * 0.5,
			AntiWindup: 1.0 + rand.Float64()*9.0,
		},
		Acc: (rand.Float64() - 0.5) * 20.0,
	})
}

func TestProperty_Determinism(t *testing.T) {
	f := func(inputs inputsWithAccForDeterminism) bool {
		prevState := State{BaseFee: inputs.BaseFee, Acc: inputs.Acc}
		params := Params{Kp: inputs.Kp, Ki: inputs.Ki, AntiWindupLimit: inputs.AntiWindup}
		
		result1, err1 := Next(prevState, inputs.GasUsed, inputs.GasTarget, params)
		result2, err2 := Next(prevState, inputs.GasUsed, inputs.GasTarget, params)
		
		if err1 != nil || err2 != nil {
			return err1 == err2
		}
		
		return result1.BaseFee == result2.BaseFee && result1.Acc == result2.Acc
	}
	
	if err := quick.Check(f, &quick.Config{MaxCount: 10000}); err != nil {
		t.Errorf("Invariant violated: %v", err)
	}
}