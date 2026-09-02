// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package abtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAssigner() *Assigner {
	a := NewAssigner()
	a.AddExperiment(&Experiment{
		Name: "color",
		Variants: []Variant{
			{Name: "red", Weight: 1},
			{Name: "blue", Weight: 1},
		},
	})
	return a
}

func TestAssign_Consistency(t *testing.T) {
	a := newTestAssigner()
	first := a.Assign("user-123")
	require.Contains(t, first, "color")
	second := a.Assign("user-123")
	assert.Equal(t, first, second)

	one1, err := a.AssignOne("color", "user-123")
	require.NoError(t, err)
	one2, err := a.AssignOne("color", "user-123")
	require.NoError(t, err)
	assert.Equal(t, one1, one2)
	assert.Equal(t, first["color"], one1)
}

func TestAssignOne_NotFound(t *testing.T) {
	a := newTestAssigner()
	_, err := a.AssignOne("missing", "user-1")
	assert.ErrorIs(t, err, ErrExperimentNotFound)
}

func TestAssign_Distribution(t *testing.T) {
	a := NewAssigner()
	a.AddExperiment(&Experiment{
		Name: "weight",
		Variants: []Variant{
			{Name: "a", Weight: 3},
			{Name: "b", Weight: 1},
		},
	})

	const n = 20000
	counts := map[string]int{}
	for i := 0; i < n; i++ {
		uid := "u-" + itoa(i)
		v, err := a.AssignOne("weight", uid)
		require.NoError(t, err)
		counts[v]++
	}

	// Expect roughly 75% / 25%.
	aRatio := float64(counts["a"]) / n
	assert.InDelta(t, 0.75, aRatio, 0.05)
	bRatio := float64(counts["b"]) / n
	assert.InDelta(t, 0.25, bRatio, 0.05)
}

func TestAssign_EqualWeights(t *testing.T) {
	a := NewAssigner()
	a.AddExperiment(&Experiment{
		Name: "even",
		Variants: []Variant{
			{Name: "x", Weight: 1},
			{Name: "y", Weight: 1},
			{Name: "z", Weight: 1},
		},
	})

	const n = 30000
	counts := map[string]int{}
	for i := 0; i < n; i++ {
		uid := "u-" + itoa(i)
		v, err := a.AssignOne("even", uid)
		require.NoError(t, err)
		counts[v]++
	}
	// Each ~33%.
	for _, name := range []string{"x", "y", "z"} {
		r := float64(counts[name]) / n
		assert.InDelta(t, 1.0/3.0, r, 0.05)
	}
}

func TestHashPercentage_Range(t *testing.T) {
	for _, u := range []string{"a", "b", "user-1", "user-2", ""} {
		p := HashPercentage(u, "exp")
		assert.GreaterOrEqual(t, p, 0.0)
		assert.Less(t, p, 1.0)
	}
}

func TestHashPercentage_Deterministic(t *testing.T) {
	assert.Equal(t, HashPercentage("u1", "e1"), HashPercentage("u1", "e1"))
	// Different experiment yields different hash (extremely likely).
	assert.NotEqual(t, HashPercentage("u1", "e1"), HashPercentage("u1", "e2"))
}

func TestAssign_ZeroWeight(t *testing.T) {
	a := NewAssigner()
	a.AddExperiment(&Experiment{
		Name: "zero",
		Variants: []Variant{
			{Name: "a", Weight: 0},
			{Name: "b", Weight: 1},
		},
	})
	for i := 0; i < 100; i++ {
		v, err := a.AssignOne("zero", "u-"+itoa(i))
		require.NoError(t, err)
		assert.Equal(t, "b", v)
	}
}

func TestAssign_AllZeroWeight(t *testing.T) {
	a := NewAssigner()
	a.AddExperiment(&Experiment{
		Name: "allzero",
		Variants: []Variant{
			{Name: "a", Weight: 0},
			{Name: "b", Weight: 0},
		},
	})
	v, err := a.AssignOne("allzero", "u-1")
	require.NoError(t, err)
	assert.Equal(t, "", v)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
