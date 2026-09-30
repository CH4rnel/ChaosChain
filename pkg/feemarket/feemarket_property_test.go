package feemarket

import (
	"math/rand"
	"reflect"
	"strconv"
	"testing"
	"testing/quick"

	sdkmath "cosmossdk.io/math"
)

func decFloat(value float64) sdkmath.LegacyDec {
	result, err := sdkmath.LegacyNewDecFromStr(strconv.FormatFloat(value, 'f', sdkmath.LegacyPrecision, 64))
	if err != nil {
		panic(err)
	}
	return result
}

type transitionInputs struct {
	baseFee, gasUsed, gasTarget, kp, ki, antiWindup, acc float64
}

func (transitionInputs) Generate(random *rand.Rand, _ int) reflect.Value {
	limit := random.Float64() * 10
	return reflect.ValueOf(transitionInputs{
		baseFee:    1 + random.Float64()*1_000,
		gasUsed:    random.Float64() * 20_000_000,
		gasTarget:  10_000_000 + random.Float64()*10_000_000,
		kp:         random.Float64() * 2,
		ki:         random.Float64() * 0.5,
		antiWindup: limit,
		acc:        (random.Float64()*2 - 1) * limit,
	})
}

func TestProperty_FixedPointTransitionsPreserveInvariants(t *testing.T) {
	check := func(input transitionInputs) bool {
		params := Params{Kp: decFloat(input.kp), Ki: decFloat(input.ki), AntiWindupLimit: decFloat(input.antiWindup), GasTarget: decFloat(input.gasTarget)}
		state := State{BaseFee: decFloat(input.baseFee), Acc: decFloat(input.acc)}
		next, err := Next(state, decFloat(input.gasUsed), params)
		return err == nil && ValidateStateWithParams(next, params) == nil
	}
	if err := quick.Check(check, &quick.Config{MaxCount: 10_000}); err != nil {
		t.Fatalf("fixed-point state invariant failed: %v", err)
	}
}

func TestProperty_TransitionIsDeterministic(t *testing.T) {
	params := Params{Kp: dec("0.37"), Ki: dec("0.12"), AntiWindupLimit: dec("8"), GasTarget: dec("10000000")}
	previous := State{BaseFee: dec("42.123456789123456789"), Acc: dec("-1.25")}
	first, err := Next(previous, dec("17892341"), params)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Next(previous, dec("17892341"), params)
	if err != nil {
		t.Fatal(err)
	}
	if !first.BaseFee.Equal(second.BaseFee) || !first.Acc.Equal(second.Acc) {
		t.Fatalf("identical inputs produced different states: %+v vs %+v", first, second)
	}
}
