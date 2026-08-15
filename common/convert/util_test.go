// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// DeepCopy
// ──────────────────────────────────────────────

type deepInner struct {
	Tags []string
	Meta map[string]int
}

type deepOuter struct {
	Name  string
	Inner deepInner
	List  []deepInner
}

func TestDeepCopy(t *testing.T) {
	t.Run("struct with nested fields deep independence", func(t *testing.T) {
		orig := deepOuter{
			Name: "alpha",
			Inner: deepInner{
				Tags: []string{"a", "b"},
				Meta: map[string]int{"x": 1},
			},
			List: []deepInner{
				{Tags: []string{"c"}, Meta: map[string]int{"y": 2}},
			},
		}
		cp := DeepCopy(orig)

		require.Equal(t, orig, cp)

		// Modify the copy and ensure the original is untouched.
		cp.Name = "beta"
		cp.Inner.Tags[0] = "zzz"
		cp.Inner.Meta["x"] = 999
		cp.List[0].Tags[0] = "qqq"

		assert.Equal(t, "alpha", orig.Name)
		assert.Equal(t, "a", orig.Inner.Tags[0])
		assert.Equal(t, 1, orig.Inner.Meta["x"])
		assert.Equal(t, "c", orig.List[0].Tags[0])
	})

	t.Run("slice deep independence", func(t *testing.T) {
		orig := [][]string{{"a", "b"}, {"c"}}
		cp := DeepCopy(orig)
		require.Equal(t, orig, cp)

		cp[0][0] = "X"
		assert.Equal(t, "a", orig[0][0])
	})

	t.Run("map deep independence", func(t *testing.T) {
		orig := map[string][]int{"k": {1, 2, 3}}
		cp := DeepCopy(orig)
		require.Equal(t, orig, cp)

		cp["k"][0] = 99
		assert.Equal(t, 1, orig["k"][0])
	})

	t.Run("unmarshalable type returns zero", func(t *testing.T) {
		type withChan struct {
			C chan int
		}
		v := withChan{C: make(chan int)}
		cp := DeepCopy(v)
		assert.Nil(t, cp.C)
	})
}

// ──────────────────────────────────────────────
// CopyJSON
// ──────────────────────────────────────────────

func TestCopyJSON(t *testing.T) {
	t.Run("struct", func(t *testing.T) {
		type s struct {
			A int    `json:"a"`
			B string `json:"b"`
		}
		out, err := CopyJSON(s{A: 1, B: "hi"})
		require.NoError(t, err)
		m, ok := out.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(1), m["a"])
		assert.Equal(t, "hi", m["b"])
	})

	t.Run("map", func(t *testing.T) {
		out, err := CopyJSON(map[string]int{"k": 2})
		require.NoError(t, err)
		m, ok := out.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(2), m["k"])
	})

	t.Run("slice", func(t *testing.T) {
		out, err := CopyJSON([]int{1, 2, 3})
		require.NoError(t, err)
		s, ok := out.([]any)
		require.True(t, ok)
		assert.Equal(t, []any{float64(1), float64(2), float64(3)}, s)
	})

	t.Run("nil", func(t *testing.T) {
		out, err := CopyJSON(nil)
		require.NoError(t, err)
		assert.Nil(t, out)
	})

	t.Run("unmarshalable type returns error", func(t *testing.T) {
		_, err := CopyJSON(make(chan int))
		require.Error(t, err)
	})
}

// ──────────────────────────────────────────────
// Clone
// ──────────────────────────────────────────────

func TestClone(t *testing.T) {
	t.Run("struct deep independence", func(t *testing.T) {
		type item struct {
			Name string
			Tags []string
		}
		type box struct {
			Title string
			Items []item
		}
		orig := box{
			Title: "orig",
			Items: []item{{Name: "i1", Tags: []string{"t1"}}},
		}
		cp, err := Clone(orig)
		require.NoError(t, err)
		require.Equal(t, orig, cp)

		cp.Items[0].Tags[0] = "changed"
		assert.Equal(t, "t1", orig.Items[0].Tags[0])
	})

	t.Run("slice", func(t *testing.T) {
		orig := []int{1, 2, 3}
		cp, err := Clone(orig)
		require.NoError(t, err)
		assert.Equal(t, orig, cp)

		cp[0] = 99
		assert.Equal(t, 1, orig[0])
	})

	t.Run("map", func(t *testing.T) {
		orig := map[string]int{"a": 1, "b": 2}
		cp, err := Clone(orig)
		require.NoError(t, err)
		assert.Equal(t, orig, cp)

		cp["a"] = 100
		assert.Equal(t, 1, orig["a"])
	})

	t.Run("unmarshalable type returns error", func(t *testing.T) {
		type bad struct {
			C chan int
		}
		_, err := Clone(bad{C: make(chan int)})
		require.Error(t, err)
	})
}

// ──────────────────────────────────────────────
// ToMapStringString
// ──────────────────────────────────────────────

func TestToMapStringString(t *testing.T) {
	t.Run("map[string]int", func(t *testing.T) {
		out, err := ToMapStringString(map[string]int{"a": 1, "b": 2})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"a": "1", "b": "2"}, out)
	})

	t.Run("map[any]any", func(t *testing.T) {
		in := map[any]any{"a": 1, "b": true}
		out, err := ToMapStringString(in)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"a": "1", "b": "true"}, out)
	})

	t.Run("nil returns nil", func(t *testing.T) {
		out, err := ToMapStringString(nil)
		require.NoError(t, err)
		assert.Nil(t, out)
	})

	t.Run("non-map returns error", func(t *testing.T) {
		_, err := ToMapStringString([]int{1, 2})
		require.Error(t, err)
	})

	t.Run("map with non-string key converted via ToString", func(t *testing.T) {
		// int keys are converted to string keys by ToString.
		out, err := ToMapStringString(map[int]string{1: "one", 2: "two"})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"1": "one", "2": "two"}, out)
	})
}

// ──────────────────────────────────────────────
// ToMapStringInt
// ──────────────────────────────────────────────

func TestToMapStringInt(t *testing.T) {
	t.Run("map[string]int", func(t *testing.T) {
		out, err := ToMapStringInt(map[string]int{"a": 1, "b": 2})
		require.NoError(t, err)
		assert.Equal(t, map[string]int{"a": 1, "b": 2}, out)
	})

	t.Run("map[string]string values parsed", func(t *testing.T) {
		out, err := ToMapStringInt(map[string]string{"a": "10", "b": "20"})
		require.NoError(t, err)
		assert.Equal(t, map[string]int{"a": 10, "b": 20}, out)
	})

	t.Run("nil returns nil", func(t *testing.T) {
		out, err := ToMapStringInt(nil)
		require.NoError(t, err)
		assert.Nil(t, out)
	})

	t.Run("non-map returns error", func(t *testing.T) {
		_, err := ToMapStringInt("not a map")
		require.Error(t, err)
	})

	t.Run("map with non-int value returns error", func(t *testing.T) {
		_, err := ToMapStringInt(map[string]string{"a": "abc"})
		require.Error(t, err)
	})
}

// ──────────────────────────────────────────────
// JSONToMap
// ──────────────────────────────────────────────

func TestJSONToMap(t *testing.T) {
	t.Run("valid JSON object", func(t *testing.T) {
		out, err := JSONToMap([]byte(`{"a":1,"b":"hi"}`))
		require.NoError(t, err)
		assert.Equal(t, float64(1), out["a"])
		assert.Equal(t, "hi", out["b"])
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		_, err := JSONToMap([]byte(`{invalid`))
		require.Error(t, err)
	})

	t.Run("JSON array returns error (not a map)", func(t *testing.T) {
		_, err := JSONToMap([]byte(`[1,2,3]`))
		require.Error(t, err)
	})
}

// ──────────────────────────────────────────────
// JSONToSlice
// ──────────────────────────────────────────────

func TestJSONToSlice(t *testing.T) {
	t.Run("valid JSON array", func(t *testing.T) {
		out, err := JSONToSlice([]byte(`[1,2,3]`))
		require.NoError(t, err)
		assert.Equal(t, []any{float64(1), float64(2), float64(3)}, out)
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		_, err := JSONToSlice([]byte(`[invalid`))
		require.Error(t, err)
	})

	t.Run("JSON object returns error (not a slice)", func(t *testing.T) {
		_, err := JSONToSlice([]byte(`{"a":1}`))
		require.Error(t, err)
	})
}

// ──────────────────────────────────────────────
// MustMarshal
// ──────────────────────────────────────────────

func TestMustMarshal(t *testing.T) {
	t.Run("valid value JSON format", func(t *testing.T) {
		b := MustMarshal(FormatJSON, map[string]any{"a": 1})
		assert.JSONEq(t, `{"a":1}`, string(b))
	})

	t.Run("panic on unsupported format", func(t *testing.T) {
		assert.Panics(t, func() {
			MustMarshal(Format("xml"), map[string]any{"a": 1})
		})
	})
}

// ──────────────────────────────────────────────
// MustUnmarshal
// ──────────────────────────────────────────────

func TestMustUnmarshal(t *testing.T) {
	t.Run("valid data JSON format", func(t *testing.T) {
		var v map[string]any
		require.NotPanics(t, func() {
			MustUnmarshal(FormatJSON, []byte(`{"a":1}`), &v)
		})
		assert.Equal(t, float64(1), v["a"])
	})

	t.Run("panic on invalid data", func(t *testing.T) {
		var v map[string]any
		assert.Panics(t, func() {
			MustUnmarshal(FormatJSON, []byte(`{invalid`), &v)
		})
	})
}
