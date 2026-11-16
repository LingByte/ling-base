// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package money

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinInt64Arithmetic(t *testing.T) {
	m := New(math.MinInt64, "USD")

	_, err := m.Neg()
	assert.ErrorIs(t, err, ErrOverflow)
	_, err = m.Abs()
	assert.ErrorIs(t, err, ErrOverflow)
	_, err = m.Div(-1)
	assert.ErrorIs(t, err, ErrOverflow)
	_, _, err = m.DivMod(-1)
	assert.ErrorIs(t, err, ErrOverflow)

	// 0 - MinInt64 overflows int64.
	_, err = New(0, "USD").Sub(m)
	assert.ErrorIs(t, err, ErrOverflow)

	// -1 - MinInt64 = MaxInt64 (and must not negate MinInt64 first).
	diff, err := New(-1, "USD").Sub(m)
	require.NoError(t, err)
	assert.Equal(t, int64(math.MaxInt64), diff.Amount())
}

func TestConvertRatio(t *testing.T) {
	// $1.00 -> ¥7.20 at 72/10
	out, err := New(100, "USD").ConvertRatio("CNY", 72, 10, RoundHalfUp)
	require.NoError(t, err)
	assert.Equal(t, "CNY", out.Currency())
	assert.Equal(t, int64(720), out.Amount())
}
