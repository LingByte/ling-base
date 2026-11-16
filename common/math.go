// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package common

import (
	"fmt"
	"math"
)

// Clamp returns value limited to [min, max]. Panics if min > max.
func Clamp[T int | int64 | float32 | float64](value, min, max T) T {
	if min > max {
		panic(fmt.Sprintf("common: Clamp min (%v) > max (%v)", min, max))
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// Min returns the smaller of a and b.
func Min[T int | int64 | float32 | float64](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// Max returns the larger of a and b.
func Max[T int | int64 | float32 | float64](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// RoundFloat rounds x to precision decimal places (half away from zero).
func RoundFloat(x float64, precision int) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return x
	}
	pow := math.Pow(10, float64(precision))
	return math.Round(x*pow) / pow
}

// GCD returns the greatest common divisor of a and b (non-negative).
func GCD(a, b int) int {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
