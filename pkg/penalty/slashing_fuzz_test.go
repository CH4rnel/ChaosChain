package penalty

import (
	"math"
	"testing"
)

func FuzzCalculateSlashFractionPreservesBounds(f *testing.F) {
	f.Add(0.01, 2.0, 0.5)
	f.Add(1.0, 100.0, 1.0)
	f.Add(-1.0, -1.0, -1.0)

	f.Fuzz(func(t *testing.T, baseSlash, kappa, faultyShare float64) {
		fraction, err := CalculateSlashFraction(faultyShare, Params{
			BaseSlash: baseSlash,
			Kappa:     kappa,
		})
		if err == nil && (math.IsNaN(fraction) || math.IsInf(fraction, 0) || fraction < 0 || fraction > 1 || fraction < baseSlash) {
			t.Fatalf("returned invalid slash fraction %v for base=%v kappa=%v share=%v", fraction, baseSlash, kappa, faultyShare)
		}
	})
}
