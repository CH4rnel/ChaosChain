package feemarket

import (
	"testing"

	sdkmath "cosmossdk.io/math"
)

func dec(value string) sdkmath.LegacyDec {
	result, err := sdkmath.LegacyNewDecFromStr(value)
	if err != nil {
		panic(err)
	}
	return result
}

func TestNextUsesDeterministicFixedPointRegulation(t *testing.T) {
	params := Params{Kp: dec("0.1"), Ki: dec("0.01"), AntiWindupLimit: dec("10"), GasTarget: dec("100")}
	previous := State{BaseFee: dec("10"), Acc: dec("0")}

	stable, err := Next(previous, dec("100"), params)
	if err != nil {
		t.Fatal(err)
	}
	if !stable.BaseFee.Equal(dec("10")) || !stable.Acc.IsZero() {
		t.Fatalf("at target expected unchanged state, got %+v", stable)
	}

	up, err := Next(previous, dec("150"), params)
	if err != nil {
		t.Fatal(err)
	}
	down, err := Next(previous, dec("50"), params)
	if err != nil {
		t.Fatal(err)
	}
	if !up.BaseFee.GT(previous.BaseFee) || !down.BaseFee.LT(previous.BaseFee) {
		t.Fatalf("expected overload to raise and underload to lower fee, got up=%s down=%s", up.BaseFee, down.BaseFee)
	}
}

func TestNextClampsPerBlockFeeChangeAndAccumulator(t *testing.T) {
	params := Params{Kp: dec("100"), Ki: dec("100"), AntiWindupLimit: dec("2"), GasTarget: dec("1")}
	previous := State{BaseFee: dec("100"), Acc: dec("0")}
	next, err := Next(previous, maxGasUsed, params)
	if err != nil {
		t.Fatal(err)
	}
	if !next.BaseFee.Equal(previous.BaseFee.Mul(one.Add(maxStep))) {
		t.Fatalf("positive fee step was not clamped: got %s", next.BaseFee)
	}
	if next.Acc.GT(params.AntiWindupLimit) {
		t.Fatalf("accumulator exceeded anti-windup limit: %s", next.Acc)
	}

	params.Ki = dec("0")
	params.Kp = dec("100")
	previous.Acc = dec("0")
	next, err = Next(previous, dec("0"), params)
	if err != nil {
		t.Fatal(err)
	}
	if !next.BaseFee.Equal(previous.BaseFee.Mul(one.Sub(maxStep))) {
		t.Fatalf("negative fee step was not clamped: got %s", next.BaseFee)
	}
}

func TestNextSaturatesAtFeeLimitsWithoutFailingTheBlock(t *testing.T) {
	params := Params{Kp: dec("100"), Ki: dec("0"), AntiWindupLimit: dec("0"), GasTarget: dec("1")}
	for _, test := range []struct {
		name string
		fee  sdkmath.LegacyDec
		gas  sdkmath.LegacyDec
		want sdkmath.LegacyDec
	}{
		{name: "maximum fee", fee: maxBaseFee, gas: maxGasUsed, want: maxBaseFee},
		{name: "minimum fee", fee: minBaseFee, gas: dec("0"), want: minBaseFee},
	} {
		t.Run(test.name, func(t *testing.T) {
			next, err := Next(State{BaseFee: test.fee, Acc: dec("0")}, test.gas, params)
			if err != nil {
				t.Fatal(err)
			}
			if !next.BaseFee.Equal(test.want) {
				t.Fatalf("expected bounded fee %s, got %s", test.want, next.BaseFee)
			}
		})
	}
}

func TestNextRejectsInvalidConsensusInputs(t *testing.T) {
	params := DefaultParams()
	for _, test := range []struct {
		name  string
		state State
		gas   sdkmath.LegacyDec
	}{
		{name: "nil base fee", state: State{}, gas: dec("1")},
		{name: "negative gas", state: State{BaseFee: dec("10")}, gas: dec("-1")},
		{name: "excessive gas", state: State{BaseFee: dec("10")}, gas: maxGasUsed.Add(one)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Next(test.state, test.gas, params); err == nil {
				t.Fatal("expected invalid input to be rejected")
			}
		})
	}

	params.GasTarget = dec("0")
	if err := ValidateParams(params); err == nil {
		t.Fatal("expected zero gas target to be rejected")
	}
}

func TestDefaultParamsDefineOperationalGasTarget(t *testing.T) {
	if !DefaultParams().GasTarget.Equal(dec("10000000")) {
		t.Fatal("default gas target must be 10000000")
	}
}
