// pkg/slashing/slashing_test.go
package slashing

import (
	"math"
	"testing"
)

// TestCalculateSlashFraction_Basic verifies base slashing without correlation penalty.
func TestCalculateSlashFraction_Basic(t *testing.T) {
	params := Params{
		BaseSlash: 0.01, // 1% base penalty
		Kappa:     0.5,  // correlation penalty coefficient
	}

	// Minimal fault (e.g., 1% of voting power)
	faultyShare := 0.01
	result, err := CalculateSlashFraction(faultyShare, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected: 0.01 + 0.5 * (0.01)^2 = 0.01 + 0.5 * 0.0001 = 0.01005
	expected := 0.01005
	if math.Abs(result-expected) > 1e-9 {
		t.Errorf("expected %f, got %f", expected, result)
	}
}

// TestCalculateSlashFraction_CorrelationPenalty verifies quadratic growth on mass faults.
func TestCalculateSlashFraction_CorrelationPenalty(t *testing.T) {
	params := Params{
		BaseSlash: 0.05,
		Kappa:     2.0, // High penalty for correlation
	}

	// Mass fault (e.g., 30% of voting power, close to the 1/3 BFT limit)
	faultyShare := 0.30
	result, err := CalculateSlashFraction(faultyShare, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected: 0.05 + 2.0 * (0.30)^2 = 0.05 + 2.0 * 0.09 = 0.05 + 0.18 = 0.23
	expected := 0.23
	if math.Abs(result-expected) > 1e-9 {
		t.Errorf("expected %f, got %f", expected, result)
	}
}

// TestCalculateSlashFraction_Capping verifies that the result never exceeds 1.0 (100%).
func TestCalculateSlashFraction_Capping(t *testing.T) {
	params := Params{
		BaseSlash: 0.5,
		Kappa:     10.0, // Extreme penalty
	}

	// 50% faulty share with extreme kappa would yield 0.5 + 10.0 * 0.25 = 3.0
	result, err := CalculateSlashFraction(0.5, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must be clamped to 1.0
	if result > 1.0 {
		t.Errorf("expected result to be capped at 1.0, got %f", result)
	}
	if math.Abs(result-1.0) > 1e-9 {
		t.Errorf("expected exactly 1.0, got %f", result)
	}
}

// TestCalculateSlashFraction_Errors verifies input validation.
func TestCalculateSlashFraction_Errors(t *testing.T) {
	params := Params{BaseSlash: 0.01, Kappa: 0.5}

	tests := []struct {
		name        string
		faultyShare float64
		params      Params
	}{
		{"negative faulty share", -0.1, params},
		{"faulty share > 1.0", 1.1, params},
		{"negative base slash", 0.1, Params{BaseSlash: -0.01, Kappa: 0.5}},
		{"negative kappa", 0.1, Params{BaseSlash: 0.01, Kappa: -0.5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateSlashFraction(tt.faultyShare, tt.params)
			if err == nil {
				t.Errorf("expected error for %s, got nil", tt.name)
			}
		})
	}
}