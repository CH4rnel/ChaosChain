package feemarket

import (
	"math"
	"testing"
)

func FuzzValidateParams(f *testing.F) {
	f.Add(0.1, 0.01, 10.0, 10_000_000.0)
	f.Add(100.0, 100.0, 1_000_000.0, 1.0)
	f.Add(-1.0, -1.0, -1.0, -1.0)
	f.Add(math.NaN(), 0.1, 1.0, 100.0)

	f.Fuzz(func(t *testing.T, kp, ki, antiWindup, gasTarget float64) {
		values := []float64{kp, ki, antiWindup, gasTarget}
		for _, value := range values {
			if math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value) > 1e18 {
				return
			}
		}
		params := Params{Kp: decFloat(kp), Ki: decFloat(ki), AntiWindupLimit: decFloat(antiWindup), GasTarget: decFloat(gasTarget)}
		if err := ValidateParams(params); err == nil && (params.Kp.IsNegative() || params.Kp.GT(maxKp) || params.Ki.IsNegative() || params.Ki.GT(maxKi) || params.AntiWindupLimit.IsNegative() || params.AntiWindupLimit.GT(maxAntiWindup) || !params.GasTarget.IsPositive() || params.GasTarget.GT(maxGasTarget)) {
			t.Fatalf("accepted out-of-range parameters: %+v", params)
		}
	})
}

func FuzzNextPreservesStateInvariants(f *testing.F) {
	f.Add(10.0, 0.0, 10_000_000.0, 0.1, 0.01, 10.0, 10_000_000.0)
	f.Add(1.0, -10.0, 0.0, 0.0, 0.0, 0.0, 1.0)
	f.Add(1e18, 0.0, 1e15, 100.0, 100.0, 1_000_000.0, 1e15)

	f.Fuzz(func(t *testing.T, baseFee, acc, gasUsed, kp, ki, antiWindup, gasTarget float64) {
		values := []float64{baseFee, acc, gasUsed, kp, ki, antiWindup, gasTarget}
		for _, value := range values {
			if math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value) > 1e18 {
				return
			}
		}
		params := Params{Kp: decFloat(kp), Ki: decFloat(ki), AntiWindupLimit: decFloat(antiWindup), GasTarget: decFloat(gasTarget)}
		previous := State{BaseFee: decFloat(baseFee), Acc: decFloat(acc)}
		next, err := Next(previous, decFloat(gasUsed), params)
		if err == nil {
			if err := ValidateStateWithParams(next, params); err != nil {
				t.Fatalf("Next returned invalid state %+v: %v", next, err)
			}
		}
	})
}
