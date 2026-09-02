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

func TestAddExperiment_Nil(t *testing.T) {
	a := NewAssigner()
	a.AddExperiment(nil) // should be a no-op
	assert.Equal(t, 0, len(a.Assign("u")))
}

func TestAddExperiment_ReplaceExisting(t *testing.T) {
	a := NewAssigner()
	a.AddExperiment(&Experiment{
		Name: "color",
		Variants: []Variant{
			{Name: "red", Weight: 1},
		},
	})
	a.AddExperiment(&Experiment{
		Name: "color",
		Variants: []Variant{
			{Name: "blue", Weight: 1},
		},
	})
	v, err := a.AssignOne("color", "u-1")
	require.NoError(t, err)
	assert.Equal(t, "blue", v)
}

func TestAssign_EmptyVariants(t *testing.T) {
	a := NewAssigner()
	a.AddExperiment(&Experiment{Name: "empty", Variants: nil})
	res := a.Assign("u-1")
	assert.Equal(t, "", res["empty"])

	v, err := a.AssignOne("empty", "u-1")
	require.NoError(t, err)
	assert.Equal(t, "", v)
}

func TestAssign_SingleVariant(t *testing.T) {
	a := NewAssigner()
	a.AddExperiment(&Experiment{
		Name: "single",
		Variants: []Variant{
			{Name: "only", Weight: 5},
		},
	})
	for i := 0; i < 50; i++ {
		v, err := a.AssignOne("single", "u-"+itoa(i))
		require.NoError(t, err)
		assert.Equal(t, "only", v)
	}
}

func TestAssign_NegativeWeightSkipped(t *testing.T) {
	a := NewAssigner()
	a.AddExperiment(&Experiment{
		Name: "neg",
		Variants: []Variant{
			{Name: "a", Weight: -1},
			{Name: "b", Weight: 2},
		},
	})
	for i := 0; i < 100; i++ {
		v, err := a.AssignOne("neg", "u-"+itoa(i))
		require.NoError(t, err)
		assert.Equal(t, "b", v)
	}
}

func TestPickVariant_FallbackPath(t *testing.T) {
	// pct of 1.0 makes target == total, so no variant satisfies target < cum,
	// exercising the last-valid-variant fallback.
	exp := &Experiment{
		Name: "fb",
		Variants: []Variant{
			{Name: "a", Weight: 1},
			{Name: "b", Weight: 1},
		},
	}
	assert.Equal(t, "b", pickVariant(exp, 1.0))
}

func TestPickVariant_AllNegativeWeight(t *testing.T) {
	exp := &Experiment{
		Name: "allneg",
		Variants: []Variant{
			{Name: "a", Weight: -1},
			{Name: "b", Weight: -2},
		},
	}
	assert.Equal(t, "", pickVariant(exp, 0.5))
}

func TestPickVariant_FallbackAllNegative(t *testing.T) {
	// total == 0 short-circuits before the fallback loop, returning "".
	exp := &Experiment{
		Name: "fbneg",
		Variants: []Variant{
			{Name: "a", Weight: -1},
		},
	}
	assert.Equal(t, "", pickVariant(exp, 1.0))
}

func TestAssign_EmptyAssigner(t *testing.T) {
	a := NewAssigner()
	res := a.Assign("u-1")
	assert.NotNil(t, res)
	assert.Empty(t, res)
}

func TestAssign_ConcurrentSafety(t *testing.T) {
	a := newTestAssigner()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 1000; i++ {
			_ = a.Assign("u-" + itoa(i))
		}
	}()
	for i := 0; i < 1000; i++ {
		_, err := a.AssignOne("color", "u-"+itoa(i))
		require.NoError(t, err)
	}
	<-done
}
