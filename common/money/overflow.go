// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package money

import "math"

// ErrOverflow is returned when an int64 arithmetic operation would overflow.
var ErrOverflow = errOverflow{}

type errOverflow struct{}

func (errOverflow) Error() string { return "money: int64 overflow" }

func addInt64(a, b int64) (int64, error) {
	if b > 0 && a > math.MaxInt64-b {
		return 0, ErrOverflow
	}
	if b < 0 && a < math.MinInt64-b {
		return 0, ErrOverflow
	}
	return a + b, nil
}

func subInt64(a, b int64) (int64, error) {
	// a - b 不能先算 -b：当 b == MinInt64 时 -b 会溢出。
	if b > 0 && a < math.MinInt64+b {
		return 0, ErrOverflow
	}
	if b < 0 && a > math.MaxInt64+b {
		return 0, ErrOverflow
	}
	return a - b, nil
}

func negInt64(a int64) (int64, error) {
	if a == math.MinInt64 {
		return 0, ErrOverflow
	}
	return -a, nil
}

func mulInt64(a, b int64) (int64, error) {
	if a == 0 || b == 0 {
		return 0, nil
	}
	if a == -1 && b == math.MinInt64 {
		return 0, ErrOverflow
	}
	if b == -1 && a == math.MinInt64 {
		return 0, ErrOverflow
	}
	result := a * b
	if result/a != b {
		return 0, ErrOverflow
	}
	return result, nil
}

// pow10 returns 10^n. n must be in [0, 18].
func pow10(n int) (int64, error) {
	if n < 0 || n > 18 {
		return 0, ErrOverflow
	}
	r := int64(1)
	for i := 0; i < n; i++ {
		r *= 10
	}
	return r, nil
}
