// pkg/feemarket/feemarket_test.go
package feemarket

import (
	"math"
	"testing"
)

// TestNext_Stability verifies that the base fee remains stable
// when gas usage exactly matches the target.
func TestNext_Stability(t *testing.T) {
	prev := State{BaseFee: 10.0, Acc: 0.0}
	p := Params{Kp: 0.1, Ki: 0.01, AntiWindupLimit: 10.0, GasTarget: 100.0}

	next, err := Next(prev, 100.0, p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// e = 0, acc remains 0, baseFee should remain 10.0
	if math.Abs(next.BaseFee-10.0) > 1e-9 {
		t.Errorf("expected baseFee to remain 10.0, got %f", next.BaseFee)
	}
	if math.Abs(next.Acc-0.0) > 1e-9 {
		t.Errorf("expected acc to remain 0.0, got %f", next.Acc)
	}
}

// TestNext_GrowthAndDrop verifies that base fee grows on overload
// and drops on underload.
func TestNext_GrowthAndDrop(t *testing.T) {
	p := Params{Kp: 0.1, Ki: 0.01, AntiWindupLimit: 10.0, GasTarget: 100.0}

	// Test Growth (50% overload)
	prevGrowth := State{BaseFee: 10.0, Acc: 0.0}
	nextGrowth, err := Next(prevGrowth, 150.0, p)
	if err != nil {
		t.Fatalf("unexpected error on growth: %v", err)
	}
	if nextGrowth.BaseFee <= 10.0 {
		t.Errorf("expected baseFee to grow, got %f", nextGrowth.BaseFee)
	}

	// Test Drop (50% underload)
	prevDrop := State{BaseFee: 10.0, Acc: 0.0}
	nextDrop, err := Next(prevDrop, 50.0, p)
	if err != nil {
		t.Fatalf("unexpected error on drop: %v", err)
	}
	if nextDrop.BaseFee >= 10.0 {
		t.Errorf("expected baseFee to drop, got %f", nextDrop.BaseFee)
	}
}

// TestNext_Errors verifies that invalid inputs return appropriate errors.
func TestNext_Errors(t *testing.T) {
	p := Params{Kp: 0.1, Ki: 0.01, AntiWindupLimit: 10.0, GasTarget: 100.0}

	tests := []struct {
		name      string
		prev      State
		gasUsed   float64
		gasTarget float64
	}{
		{"gasTarget <= 0", State{BaseFee: 10.0, Acc: 0.0}, 10.0, 0.0},
		{"baseFee <= 0", State{BaseFee: 0.0, Acc: 0.0}, 10.0, 100.0},
		{"gasUsed < 0", State{BaseFee: 10.0, Acc: 0.0}, -10.0, 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := p
			params.GasTarget = tt.gasTarget
			_, err := Next(tt.prev, tt.gasUsed, params)
			if err == nil {
				t.Errorf("expected error for %s, got nil", tt.name)
			}
		})
	}
}

func TestDefaultParamsDefineOperationalGasTarget(t *testing.T) {
	params := DefaultParams()
	if params.GasTarget != 10_000_000 {
		t.Fatalf("expected default gas target 10000000, got %f", params.GasTarget)
	}
}

func TestNextRejectsNonFiniteInputs(t *testing.T) {
	tests := []struct {
		name    string
		state   State
		gasUsed float64
		params  Params
	}{
		{"nan base fee", State{BaseFee: math.NaN()}, 1, DefaultParams()},
		{"infinite accumulator", State{BaseFee: 10, Acc: math.Inf(1)}, 1, DefaultParams()},
		{"nan gas used", State{BaseFee: 10}, math.NaN(), DefaultParams()},
		{"infinite gain", State{BaseFee: 10}, 1, Params{Kp: math.Inf(1), GasTarget: 10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Next(tt.state, tt.gasUsed, tt.params)
			if err == nil {
				t.Fatal("expected non-finite input to be rejected")
			}
		})
	}
}

// TestNext_AntiWindupRegression is a regression test ensuring that
// prolonged overload does not cause unbounded accumulator growth,
// which would prevent the fee from correcting downwards when load drops.
func TestNext_AntiWindupRegression(t *testing.T) {
	p := Params{Kp: 0.1, Ki: 0.05, AntiWindupLimit: 5.0, GasTarget: 100.0}
	prev := State{BaseFee: 10.0, Acc: 0.0}

	// Simulate 50 consecutive blocks with 100% overload (gasUsed = 2 * gasTarget)
	for i := 0; i < 50; i++ {
		var err error
		prev, err = Next(prev, 200.0, p)
		if err != nil {
			t.Fatalf("unexpected error at block %d: %v", i, err)
		}

		if prev.Acc > p.AntiWindupLimit {
			t.Errorf("acc exceeded AntiWindupLimit at block %d: got %f, limit %f", i, prev.Acc, p.AntiWindupLimit)
		}
		if prev.Acc < -p.AntiWindupLimit {
			t.Errorf("acc fell below -AntiWindupLimit at block %d: got %f, limit %f", i, prev.Acc, p.AntiWindupLimit)
		}
	}
}
