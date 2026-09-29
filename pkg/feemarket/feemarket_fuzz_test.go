package feemarket

import "testing"

func FuzzValidateParams(f *testing.F) {
	f.Add(0.1, 0.01, 10.0, 10_000_000.0)
	f.Add(100.0, 100.0, 1_000_000.0, 1.0)
	f.Add(-1.0, -1.0, -1.0, -1.0)

	f.Fuzz(func(t *testing.T, kp, ki, antiWindup, gasTarget float64) {
		params := Params{
			Kp:              kp,
			Ki:              ki,
			AntiWindupLimit: antiWindup,
			GasTarget:       gasTarget,
		}
		if err := ValidateParams(params); err == nil {
			if kp < 0 || kp > MaxKp || ki < 0 || ki > MaxKi || antiWindup < 0 || antiWindup > MaxAntiWindup || gasTarget <= 0 || gasTarget > MaxGasTarget {
				t.Fatalf("accepted out-of-range params: %+v", params)
			}
		}
	})
}
