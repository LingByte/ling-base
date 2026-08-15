// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package mathutil

import (
	"math"
	"testing"
)

// ──────────────────────────────────────────────
// Range helpers
// ──────────────────────────────────────────────

func TestClamp(t *testing.T) {
	tests := []struct {
		value, min, max, want int
	}{
		{5, 0, 10, 5},
		{-1, 0, 10, 0},
		{15, 0, 10, 10},
		{0, 0, 10, 0},
		{10, 0, 10, 10},
	}
	for _, tt := range tests {
		if got := Clamp(tt.value, tt.min, tt.max); got != tt.want {
			t.Fatalf("Clamp(%d, %d, %d) = %d, want %d", tt.value, tt.min, tt.max, got, tt.want)
		}
	}
}

func TestClamp_Float(t *testing.T) {
	if got := Clamp(15.5, 0.0, 10.0); got != 10.0 {
		t.Fatalf("Clamp(15.5, 0, 10) = %f, want 10", got)
	}
}

func TestClamp_PanicOnInvalidRange(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Clamp should panic when min > max")
		}
	}()
	Clamp(5, 10, 0)
}

func TestMin(t *testing.T) {
	if Min(3, 7) != 3 {
		t.Fatal("Min(3, 7) should be 3")
	}
	if Min(7, 3) != 3 {
		t.Fatal("Min(7, 3) should be 3")
	}
}

func TestMax(t *testing.T) {
	if Max(3, 7) != 7 {
		t.Fatal("Max(3, 7) should be 7")
	}
}

func TestMinMax(t *testing.T) {
	min, max := MinMax(3, 7)
	if min != 3 || max != 7 {
		t.Fatalf("MinMax(3, 7) = (%d, %d), want (3, 7)", min, max)
	}
	min, max = MinMax(7, 3)
	if min != 3 || max != 7 {
		t.Fatalf("MinMax(7, 3) = (%d, %d), want (3, 7)", min, max)
	}
}

func TestInRange(t *testing.T) {
	if !InRange(5, 0, 10) {
		t.Fatal("5 should be in [0, 10]")
	}
	if !InRange(0, 0, 10) {
		t.Fatal("0 should be in [0, 10]")
	}
	if !InRange(10, 0, 10) {
		t.Fatal("10 should be in [0, 10]")
	}
	if InRange(-1, 0, 10) {
		t.Fatal("-1 should not be in [0, 10]")
	}
	if InRange(11, 0, 10) {
		t.Fatal("11 should not be in [0, 10]")
	}
}

func TestMinSlice(t *testing.T) {
	if MinSlice([]int{3, 1, 4, 1, 5, 9, 2, 6}) != 1 {
		t.Fatal("MinSlice failed")
	}
}

func TestMinSlice_PanicEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MinSlice should panic on empty slice")
		}
	}()
	MinSlice([]int{})
}

func TestMaxSlice(t *testing.T) {
	if MaxSlice([]int{3, 1, 4, 1, 5, 9, 2, 6}) != 9 {
		t.Fatal("MaxSlice failed")
	}
}

func TestMaxSlice_PanicEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MaxSlice should panic on empty slice")
		}
	}()
	MaxSlice([]int{})
}

// ──────────────────────────────────────────────
// Precision / rounding
// ──────────────────────────────────────────────

func TestRound(t *testing.T) {
	tests := []struct {
		x         float64
		precision int
		want      float64
	}{
		{3.14159, 2, 3.14},
		{3.14159, 4, 3.1416},
		{2.5, 0, 3},
		{2.4, 0, 2},
		{1.2345, 2, 1.23},
		{-2.5, 0, -3},
	}
	for _, tt := range tests {
		if got := Round(tt.x, tt.precision); math.Abs(got-tt.want) > 1e-9 {
			t.Fatalf("Round(%f, %d) = %f, want %f", tt.x, tt.precision, got, tt.want)
		}
	}
}

func TestRound_NaNInf(t *testing.T) {
	if !math.IsNaN(Round(math.NaN(), 2)) {
		t.Fatal("Round(NaN) should be NaN")
	}
	if !math.IsInf(Round(math.Inf(1), 2), 1) {
		t.Fatal("Round(+Inf) should be +Inf")
	}
}

func TestRoundTo(t *testing.T) {
	if RoundTo(17, 5) != 15 {
		t.Fatalf("RoundTo(17, 5) = %f, want 15", RoundTo(17, 5))
	}
	if RoundTo(18, 5) != 20 {
		t.Fatalf("RoundTo(18, 5) = %f, want 20", RoundTo(18, 5))
	}
}

func TestRoundTo_Zero(t *testing.T) {
	if RoundTo(17, 0) != 17 {
		t.Fatal("RoundTo(x, 0) should return x")
	}
}

func TestFloorTo(t *testing.T) {
	if FloorTo(17, 5) != 15 {
		t.Fatalf("FloorTo(17, 5) = %f, want 15", FloorTo(17, 5))
	}
}

func TestCeilTo(t *testing.T) {
	if CeilTo(17, 5) != 20 {
		t.Fatalf("CeilTo(17, 5) = %f, want 20", CeilTo(17, 5))
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate(3.9999, 2); got != 3.99 {
		t.Fatalf("Truncate(3.9999, 2) = %f, want 3.99", got)
	}
	if got := Truncate(-3.9999, 2); got != -3.99 {
		t.Fatalf("Truncate(-3.9999, 2) = %f, want -3.99", got)
	}
}

// ──────────────────────────────────────────────
// Statistics
// ──────────────────────────────────────────────

func TestSum(t *testing.T) {
	if got := Sum([]float64{1, 2, 3, 4, 5}); got != 15 {
		t.Fatalf("Sum = %f, want 15", got)
	}
}

func TestSum_Empty(t *testing.T) {
	if Sum(nil) != 0 {
		t.Fatal("Sum of empty should be 0")
	}
}

func TestSumInt(t *testing.T) {
	if got := SumInt([]int{1, 2, 3}); got != 6 {
		t.Fatalf("SumInt = %d, want 6", got)
	}
}

func TestMean(t *testing.T) {
	if got := Mean([]float64{1, 2, 3, 4, 5}); got != 3 {
		t.Fatalf("Mean = %f, want 3", got)
	}
}

func TestMean_Empty(t *testing.T) {
	if !math.IsNaN(Mean(nil)) {
		t.Fatal("Mean of empty should be NaN")
	}
}

func TestMedian_Odd(t *testing.T) {
	if got := Median([]float64{3, 1, 2}); got != 2 {
		t.Fatalf("Median = %f, want 2", got)
	}
}

func TestMedian_Even(t *testing.T) {
	if got := Median([]float64{1, 2, 3, 4}); got != 2.5 {
		t.Fatalf("Median = %f, want 2.5", got)
	}
}

func TestMedian_Empty(t *testing.T) {
	if !math.IsNaN(Median(nil)) {
		t.Fatal("Median of empty should be NaN")
	}
}

func TestMode(t *testing.T) {
	mode := Mode([]float64{1, 2, 2, 3, 3, 3})
	if len(mode) != 1 || mode[0] != 3 {
		t.Fatalf("Mode = %v, want [3]", mode)
	}
}

func TestMode_Multiple(t *testing.T) {
	mode := Mode([]float64{1, 1, 2, 2})
	if len(mode) != 2 {
		t.Fatalf("Mode = %v, want [1 2]", mode)
	}
}

func TestMode_Empty(t *testing.T) {
	if mode := Mode(nil); mode != nil {
		t.Fatalf("Mode of empty = %v, want nil", mode)
	}
}

func TestVariance(t *testing.T) {
	// Variance of [1, 2, 3, 4, 5] = 2
	got := Variance([]float64{1, 2, 3, 4, 5})
	if math.Abs(got-2) > 1e-9 {
		t.Fatalf("Variance = %f, want 2", got)
	}
}

func TestSampleVariance(t *testing.T) {
	// Sample variance of [1, 2, 3, 4, 5] = 2.5
	got := SampleVariance([]float64{1, 2, 3, 4, 5})
	if math.Abs(got-2.5) > 1e-9 {
		t.Fatalf("SampleVariance = %f, want 2.5", got)
	}
}

func TestSampleVariance_TooSmall(t *testing.T) {
	if !math.IsNaN(SampleVariance([]float64{1})) {
		t.Fatal("SampleVariance of single element should be NaN")
	}
}

func TestStdDev(t *testing.T) {
	got := StdDev([]float64{1, 2, 3, 4, 5})
	if math.Abs(got-math.Sqrt(2)) > 1e-9 {
		t.Fatalf("StdDev = %f, want %f", got, math.Sqrt(2))
	}
}

func TestSampleStdDev(t *testing.T) {
	got := SampleStdDev([]float64{1, 2, 3, 4, 5})
	if math.Abs(got-math.Sqrt(2.5)) > 1e-9 {
		t.Fatalf("SampleStdDev = %f, want %f", got, math.Sqrt(2.5))
	}
}

func TestPercentile(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if got := Percentile(data, 50); got != 5.5 {
		t.Fatalf("Percentile(50) = %f, want 5.5", got)
	}
	if got := Percentile(data, 0); got != 1 {
		t.Fatalf("Percentile(0) = %f, want 1", got)
	}
	if got := Percentile(data, 100); got != 10 {
		t.Fatalf("Percentile(100) = %f, want 10", got)
	}
	if got := Percentile(data, 25); got != 3.25 {
		t.Fatalf("Percentile(25) = %f, want 3.25", got)
	}
}

func TestPercentile_Empty(t *testing.T) {
	if !math.IsNaN(Percentile(nil, 50)) {
		t.Fatal("Percentile of empty should be NaN")
	}
}

func TestPercentile_PanicInvalidP(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Percentile should panic for p < 0 or p > 100")
		}
	}()
	Percentile([]float64{1, 2, 3}, 150)
}

func TestQuantile(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5}
	if got := Quantile(data, 0.5); got != 3 {
		t.Fatalf("Quantile(0.5) = %f, want 3", got)
	}
}

// ──────────────────────────────────────────────
// Number theory
// ──────────────────────────────────────────────

func TestGCD(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{12, 18, 6},
		{7, 13, 1},
		{0, 5, 5},
		{0, 0, 0},
		{-12, 18, 6},
	}
	for _, tt := range tests {
		if got := GCD(tt.a, tt.b); got != tt.want {
			t.Fatalf("GCD(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestLCM(t *testing.T) {
	if LCM(4, 6) != 12 {
		t.Fatalf("LCM(4, 6) = %d, want 12", LCM(4, 6))
	}
	if LCM(0, 5) != 0 {
		t.Fatal("LCM(0, 5) should be 0")
	}
}

func TestIsPrime(t *testing.T) {
	primes := []int{2, 3, 5, 7, 11, 13, 17, 19, 23, 29}
	for _, p := range primes {
		if !IsPrime(p) {
			t.Fatalf("%d should be prime", p)
		}
	}
	notPrimes := []int{0, 1, 4, 6, 8, 9, 10, 12, 15, 21}
	for _, n := range notPrimes {
		if IsPrime(n) {
			t.Fatalf("%d should not be prime", n)
		}
	}
}

func TestFactorial(t *testing.T) {
	if Factorial(0) != 1 {
		t.Fatal("0! should be 1")
	}
	if Factorial(5) != 120 {
		t.Fatal("5! should be 120")
	}
}

func TestFactorial_Negative(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Factorial should panic for negative")
		}
	}()
	Factorial(-1)
}

func TestFibonacci(t *testing.T) {
	tests := []struct {
		n, want int
	}{
		{0, 0},
		{1, 1},
		{2, 1},
		{3, 2},
		{10, 55},
	}
	for _, tt := range tests {
		if got := Fibonacci(tt.n); got != tt.want {
			t.Fatalf("Fibonacci(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

func TestFibonacci_Negative(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Fibonacci should panic for negative")
		}
	}()
	Fibonacci(-1)
}

func TestIsPowerOfTwo(t *testing.T) {
	powers := []int{1, 2, 4, 8, 16, 32, 64, 128, 256}
	for _, p := range powers {
		if !IsPowerOfTwo(p) {
			t.Fatalf("%d should be power of two", p)
		}
	}
	notPowers := []int{0, 3, 5, 6, 7, 9, 10, 15, 17}
	for _, n := range notPowers {
		if IsPowerOfTwo(n) {
			t.Fatalf("%d should not be power of two", n)
		}
	}
}

func TestNextPowerOfTwo(t *testing.T) {
	tests := []struct {
		n, want int
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 4},
		{5, 8},
		{9, 16},
		{17, 32},
	}
	for _, tt := range tests {
		if got := NextPowerOfTwo(tt.n); got != tt.want {
			t.Fatalf("NextPowerOfTwo(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

// ──────────────────────────────────────────────
// Mapping / interpolation
// ──────────────────────────────────────────────

func TestMapRange(t *testing.T) {
	if got := MapRange(5, 0, 10, 0, 100); got != 50 {
		t.Fatalf("MapRange(5, 0, 10, 0, 100) = %f, want 50", got)
	}
}

func TestMapRange_ZeroInputRange(t *testing.T) {
	if got := MapRange(5, 5, 5, 0, 100); got != 0 {
		t.Fatalf("MapRange with zero input range = %f, want 0", got)
	}
}

func TestLerp(t *testing.T) {
	if got := Lerp(0, 10, 0.5); got != 5 {
		t.Fatalf("Lerp(0, 10, 0.5) = %f, want 5", got)
	}
	if got := Lerp(0, 10, 0); got != 0 {
		t.Fatalf("Lerp(0, 10, 0) = %f, want 0", got)
	}
	if got := Lerp(0, 10, 1); got != 10 {
		t.Fatalf("Lerp(0, 10, 1) = %f, want 10", got)
	}
}

func TestInvLerp(t *testing.T) {
	if got := InvLerp(0, 10, 5); got != 0.5 {
		t.Fatalf("InvLerp(0, 10, 5) = %f, want 0.5", got)
	}
}

func TestInvLerp_SameAB(t *testing.T) {
	if got := InvLerp(5, 5, 5); got != 0 {
		t.Fatalf("InvLerp with a==b = %f, want 0", got)
	}
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

func TestAbs(t *testing.T) {
	if Abs(-5) != 5 {
		t.Fatal("Abs(-5) should be 5")
	}
	if Abs(5) != 5 {
		t.Fatal("Abs(5) should be 5")
	}
}

func TestSign(t *testing.T) {
	if Sign(5) != 1 {
		t.Fatal("Sign(5) should be 1")
	}
	if Sign(-5) != -1 {
		t.Fatal("Sign(-5) should be -1")
	}
	if Sign(0) != 0 {
		t.Fatal("Sign(0) should be 0")
	}
	if !math.IsNaN(Sign(math.NaN())) {
		t.Fatal("Sign(NaN) should be NaN")
	}
}

func TestIsInteger(t *testing.T) {
	if !IsInteger(5.0) {
		t.Fatal("5.0 should be integer")
	}
	if IsInteger(5.5) {
		t.Fatal("5.5 should not be integer")
	}
	if IsInteger(math.NaN()) {
		t.Fatal("NaN should not be integer")
	}
	if IsInteger(math.Inf(1)) {
		t.Fatal("Inf should not be integer")
	}
}

func TestIsEven(t *testing.T) {
	if !IsEven(4) {
		t.Fatal("4 should be even")
	}
	if IsEven(5) {
		t.Fatal("5 should not be even")
	}
}

func TestIsOdd(t *testing.T) {
	if !IsOdd(5) {
		t.Fatal("5 should be odd")
	}
	if IsOdd(4) {
		t.Fatal("4 should not be odd")
	}
}

func TestSquare(t *testing.T) {
	if Square(3) != 9 {
		t.Fatal("Square(3) should be 9")
	}
}

func TestCube(t *testing.T) {
	if Cube(3) != 27 {
		t.Fatal("Cube(3) should be 27")
	}
}
