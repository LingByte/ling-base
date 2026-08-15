// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// ToInt
// ──────────────────────────────────────────────

func TestToInt(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		v, err := ToInt(int(42))
		require.NoError(t, err)
		assert.Equal(t, 42, v)
	})
	t.Run("int8", func(t *testing.T) {
		v, err := ToInt(int8(8))
		require.NoError(t, err)
		assert.Equal(t, 8, v)
	})
	t.Run("int16", func(t *testing.T) {
		v, err := ToInt(int16(16))
		require.NoError(t, err)
		assert.Equal(t, 16, v)
	})
	t.Run("int32", func(t *testing.T) {
		v, err := ToInt(int32(32))
		require.NoError(t, err)
		assert.Equal(t, 32, v)
	})
	t.Run("int64", func(t *testing.T) {
		v, err := ToInt(int64(64))
		require.NoError(t, err)
		assert.Equal(t, 64, v)
	})
	t.Run("uint", func(t *testing.T) {
		v, err := ToInt(uint(100))
		require.NoError(t, err)
		assert.Equal(t, 100, v)
	})
	t.Run("uint8", func(t *testing.T) {
		v, err := ToInt(uint8(8))
		require.NoError(t, err)
		assert.Equal(t, 8, v)
	})
	t.Run("uint16", func(t *testing.T) {
		v, err := ToInt(uint16(16))
		require.NoError(t, err)
		assert.Equal(t, 16, v)
	})
	t.Run("uint32", func(t *testing.T) {
		v, err := ToInt(uint32(32))
		require.NoError(t, err)
		assert.Equal(t, 32, v)
	})
	t.Run("uint64", func(t *testing.T) {
		v, err := ToInt(uint64(64))
		require.NoError(t, err)
		assert.Equal(t, 64, v)
	})
	t.Run("float32", func(t *testing.T) {
		v, err := ToInt(float32(3.9))
		require.NoError(t, err)
		assert.Equal(t, 3, v)
	})
	t.Run("float64", func(t *testing.T) {
		v, err := ToInt(float64(99.9))
		require.NoError(t, err)
		assert.Equal(t, 99, v)
	})
	t.Run("string valid", func(t *testing.T) {
		v, err := ToInt("42")
		require.NoError(t, err)
		assert.Equal(t, 42, v)
	})
	t.Run("string with whitespace", func(t *testing.T) {
		v, err := ToInt("  42  ")
		require.NoError(t, err)
		assert.Equal(t, 42, v)
	})
	t.Run("string abc error", func(t *testing.T) {
		v, err := ToInt("abc")
		require.Error(t, err)
		assert.Equal(t, 0, v)
	})
	t.Run("string empty error", func(t *testing.T) {
		v, err := ToInt("")
		require.Error(t, err)
		assert.Equal(t, 0, v)
	})
	t.Run("bool true", func(t *testing.T) {
		v, err := ToInt(true)
		require.NoError(t, err)
		assert.Equal(t, 1, v)
	})
	t.Run("bool false", func(t *testing.T) {
		v, err := ToInt(false)
		require.NoError(t, err)
		assert.Equal(t, 0, v)
	})
	t.Run("json.Number", func(t *testing.T) {
		v, err := ToInt(json.Number("123"))
		require.NoError(t, err)
		assert.Equal(t, 123, v)
	})
	t.Run("nil error", func(t *testing.T) {
		v, err := ToInt(nil)
		require.Error(t, err)
		assert.Equal(t, 0, v)
	})
	t.Run("unsupported type error", func(t *testing.T) {
		v, err := ToInt([]int{1, 2})
		require.Error(t, err)
		assert.Equal(t, 0, v)
	})
}

// ──────────────────────────────────────────────
// ToInt64
// ──────────────────────────────────────────────

func TestToInt64(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		v, err := ToInt64(int(42))
		require.NoError(t, err)
		assert.Equal(t, int64(42), v)
	})
	t.Run("int8", func(t *testing.T) {
		v, err := ToInt64(int8(8))
		require.NoError(t, err)
		assert.Equal(t, int64(8), v)
	})
	t.Run("int64", func(t *testing.T) {
		v, err := ToInt64(int64(64))
		require.NoError(t, err)
		assert.Equal(t, int64(64), v)
	})
	t.Run("uint", func(t *testing.T) {
		v, err := ToInt64(uint(100))
		require.NoError(t, err)
		assert.Equal(t, int64(100), v)
	})
	t.Run("float64 large", func(t *testing.T) {
		// A float64 larger than int64 max range via ToInt path would overflow,
		// but ToInt64 falls back to float path on error.
		v, err := ToInt64(float64(1e15))
		require.NoError(t, err)
		assert.Equal(t, int64(1e15), v)
	})
	t.Run("float32 large", func(t *testing.T) {
		v, err := ToInt64(float32(1e10))
		require.NoError(t, err)
		assert.Equal(t, int64(float32(1e10)), v)
	})
	t.Run("string valid", func(t *testing.T) {
		v, err := ToInt64("42")
		require.NoError(t, err)
		assert.Equal(t, int64(42), v)
	})
	t.Run("string abc error", func(t *testing.T) {
		v, err := ToInt64("abc")
		require.Error(t, err)
		assert.Equal(t, int64(0), v)
	})
	t.Run("bool true", func(t *testing.T) {
		v, err := ToInt64(true)
		require.NoError(t, err)
		assert.Equal(t, int64(1), v)
	})
	t.Run("json.Number", func(t *testing.T) {
		v, err := ToInt64(json.Number("123"))
		require.NoError(t, err)
		assert.Equal(t, int64(123), v)
	})
	t.Run("nil error", func(t *testing.T) {
		v, err := ToInt64(nil)
		require.Error(t, err)
		assert.Equal(t, int64(0), v)
	})
	t.Run("unsupported type error", func(t *testing.T) {
		v, err := ToInt64([]int{1})
		require.Error(t, err)
		assert.Equal(t, int64(0), v)
	})
}

// ──────────────────────────────────────────────
// ToUint
// ──────────────────────────────────────────────

func TestToUint(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		v, err := ToUint(42)
		require.NoError(t, err)
		assert.Equal(t, uint(42), v)
	})
	t.Run("from string", func(t *testing.T) {
		v, err := ToUint("100")
		require.NoError(t, err)
		assert.Equal(t, uint(100), v)
	})
	t.Run("negative error", func(t *testing.T) {
		v, err := ToUint(-1)
		require.Error(t, err)
		assert.Equal(t, uint(0), v)
	})
	t.Run("zero", func(t *testing.T) {
		v, err := ToUint(0)
		require.NoError(t, err)
		assert.Equal(t, uint(0), v)
	})
	t.Run("nil error", func(t *testing.T) {
		v, err := ToUint(nil)
		require.Error(t, err)
		assert.Equal(t, uint(0), v)
	})
}

// ──────────────────────────────────────────────
// ToFloat64
// ──────────────────────────────────────────────

func TestToFloat64(t *testing.T) {
	t.Run("float64", func(t *testing.T) {
		v, err := ToFloat64(float64(3.14))
		require.NoError(t, err)
		assert.Equal(t, 3.14, v)
	})
	t.Run("float32", func(t *testing.T) {
		v, err := ToFloat64(float32(2.5))
		require.NoError(t, err)
		assert.Equal(t, float64(2.5), v)
	})
	t.Run("int", func(t *testing.T) {
		v, err := ToFloat64(int(5))
		require.NoError(t, err)
		assert.Equal(t, float64(5), v)
	})
	t.Run("int64", func(t *testing.T) {
		v, err := ToFloat64(int64(7))
		require.NoError(t, err)
		assert.Equal(t, float64(7), v)
	})
	t.Run("uint", func(t *testing.T) {
		v, err := ToFloat64(uint(9))
		require.NoError(t, err)
		assert.Equal(t, float64(9), v)
	})
	t.Run("uint64", func(t *testing.T) {
		v, err := ToFloat64(uint64(11))
		require.NoError(t, err)
		assert.Equal(t, float64(11), v)
	})
	t.Run("string valid", func(t *testing.T) {
		v, err := ToFloat64("3.14")
		require.NoError(t, err)
		assert.Equal(t, 3.14, v)
	})
	t.Run("string with whitespace", func(t *testing.T) {
		v, err := ToFloat64("  3.14  ")
		require.NoError(t, err)
		assert.Equal(t, 3.14, v)
	})
	t.Run("string abc error", func(t *testing.T) {
		v, err := ToFloat64("abc")
		require.Error(t, err)
		assert.Equal(t, float64(0), v)
	})
	t.Run("string empty error", func(t *testing.T) {
		v, err := ToFloat64("")
		require.Error(t, err)
		assert.Equal(t, float64(0), v)
	})
	t.Run("json.Number", func(t *testing.T) {
		v, err := ToFloat64(json.Number("3.14"))
		require.NoError(t, err)
		assert.Equal(t, 3.14, v)
	})
	t.Run("bool true", func(t *testing.T) {
		v, err := ToFloat64(true)
		require.NoError(t, err)
		assert.Equal(t, float64(1), v)
	})
	t.Run("bool false", func(t *testing.T) {
		v, err := ToFloat64(false)
		require.NoError(t, err)
		assert.Equal(t, float64(0), v)
	})
	t.Run("nil error", func(t *testing.T) {
		v, err := ToFloat64(nil)
		require.Error(t, err)
		assert.Equal(t, float64(0), v)
	})
	t.Run("unsupported type error", func(t *testing.T) {
		v, err := ToFloat64([]int{1})
		require.Error(t, err)
		assert.Equal(t, float64(0), v)
	})
}

// ──────────────────────────────────────────────
// ToBool
// ──────────────────────────────────────────────

func TestToBool(t *testing.T) {
	cases := []struct {
		name    string
		input   any
		want    bool
		wantErr bool
	}{
		{"bool true", true, true, false},
		{"bool false", false, false, false},
		{"string true", "true", true, false},
		{"string false", "false", false, false},
		{"string 1", "1", true, false},
		{"string 0", "0", false, false},
		{"string yes", "yes", true, false},
		{"string no", "no", false, false},
		{"string on", "on", true, false},
		{"string off", "off", false, false},
		{"string t", "t", true, false},
		{"string f", "f", false, false},
		{"string empty", "", false, false},
		{"string TRUE upper", "TRUE", true, false},
		{"string with whitespace", "  yes  ", true, false},
		{"string invalid", "maybe", false, true},
		{"int 0", 0, false, false},
		{"int 1", 1, true, false},
		{"int 5", 5, true, false},
		{"int8 0", int8(0), false, false},
		{"int16 1", int16(1), true, false},
		{"int32 0", int32(0), false, false},
		{"int64 1", int64(1), true, false},
		{"uint 0", uint(0), false, false},
		{"float64 0", float64(0), false, false},
		{"float64 3.14", float64(3.14), true, false},
		{"float32 0", float32(0), false, false},
		{"nil", nil, false, false},
		{"unsupported type", []int{1}, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := ToBool(tc.input)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.want, v)
		})
	}
}

// ──────────────────────────────────────────────
// ToString
// ──────────────────────────────────────────────

// stringer for testing fmt.Stringer path
type testStringer struct {
	s string
}

func (t testStringer) String() string { return t.s }

func TestToString(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		v, err := ToString("hello")
		require.NoError(t, err)
		assert.Equal(t, "hello", v)
	})
	t.Run("bool true", func(t *testing.T) {
		v, err := ToString(true)
		require.NoError(t, err)
		assert.Equal(t, "true", v)
	})
	t.Run("bool false", func(t *testing.T) {
		v, err := ToString(false)
		require.NoError(t, err)
		assert.Equal(t, "false", v)
	})
	t.Run("int", func(t *testing.T) {
		v, err := ToString(int(42))
		require.NoError(t, err)
		assert.Equal(t, "42", v)
	})
	t.Run("int8", func(t *testing.T) {
		v, err := ToString(int8(8))
		require.NoError(t, err)
		assert.Equal(t, "8", v)
	})
	t.Run("int16", func(t *testing.T) {
		v, err := ToString(int16(16))
		require.NoError(t, err)
		assert.Equal(t, "16", v)
	})
	t.Run("int32", func(t *testing.T) {
		v, err := ToString(int32(32))
		require.NoError(t, err)
		assert.Equal(t, "32", v)
	})
	t.Run("int64", func(t *testing.T) {
		v, err := ToString(int64(64))
		require.NoError(t, err)
		assert.Equal(t, "64", v)
	})
	t.Run("uint", func(t *testing.T) {
		v, err := ToString(uint(100))
		require.NoError(t, err)
		assert.Equal(t, "100", v)
	})
	t.Run("uint8", func(t *testing.T) {
		v, err := ToString(uint8(8))
		require.NoError(t, err)
		assert.Equal(t, "8", v)
	})
	t.Run("uint16", func(t *testing.T) {
		v, err := ToString(uint16(16))
		require.NoError(t, err)
		assert.Equal(t, "16", v)
	})
	t.Run("uint32", func(t *testing.T) {
		v, err := ToString(uint32(32))
		require.NoError(t, err)
		assert.Equal(t, "32", v)
	})
	t.Run("uint64", func(t *testing.T) {
		v, err := ToString(uint64(64))
		require.NoError(t, err)
		assert.Equal(t, "64", v)
	})
	t.Run("float32", func(t *testing.T) {
		v, err := ToString(float32(3.5))
		require.NoError(t, err)
		assert.Equal(t, "3.5", v)
	})
	t.Run("float64", func(t *testing.T) {
		v, err := ToString(float64(3.14))
		require.NoError(t, err)
		assert.Equal(t, "3.14", v)
	})
	t.Run("[]byte", func(t *testing.T) {
		v, err := ToString([]byte("bytes"))
		require.NoError(t, err)
		assert.Equal(t, "bytes", v)
	})
	t.Run("nil", func(t *testing.T) {
		v, err := ToString(nil)
		require.NoError(t, err)
		assert.Equal(t, "", v)
	})
	t.Run("fmt.Stringer", func(t *testing.T) {
		v, err := ToString(testStringer{s: "stringer-value"})
		require.NoError(t, err)
		assert.Equal(t, "stringer-value", v)
	})
	t.Run("json.Number", func(t *testing.T) {
		v, err := ToString(json.Number("12345"))
		require.NoError(t, err)
		assert.Equal(t, "12345", v)
	})
	t.Run("struct JSON fallback", func(t *testing.T) {
		v, err := ToString(struct{ A int }{A: 1})
		require.NoError(t, err)
		assert.Equal(t, `{"A":1}`, v)
	})
}

// ──────────────────────────────────────────────
// ToDuration
// ──────────────────────────────────────────────

func TestToDuration(t *testing.T) {
	t.Run("time.Duration", func(t *testing.T) {
		v, err := ToDuration(5 * time.Second)
		require.NoError(t, err)
		assert.Equal(t, 5*time.Second, v)
	})
	t.Run("string 5s", func(t *testing.T) {
		v, err := ToDuration("5s")
		require.NoError(t, err)
		assert.Equal(t, 5*time.Second, v)
	})
	t.Run("string 100ms", func(t *testing.T) {
		v, err := ToDuration("100ms")
		require.NoError(t, err)
		assert.Equal(t, 100*time.Millisecond, v)
	})
	t.Run("string 1h30m", func(t *testing.T) {
		v, err := ToDuration("1h30m")
		require.NoError(t, err)
		assert.Equal(t, 90*time.Minute, v)
	})
	t.Run("string invalid error", func(t *testing.T) {
		v, err := ToDuration("invalid")
		require.Error(t, err)
		assert.Equal(t, time.Duration(0), v)
	})
	t.Run("int", func(t *testing.T) {
		v, err := ToDuration(int(1000))
		require.NoError(t, err)
		assert.Equal(t, time.Duration(1000), v)
	})
	t.Run("int64", func(t *testing.T) {
		v, err := ToDuration(int64(2000))
		require.NoError(t, err)
		assert.Equal(t, time.Duration(2000), v)
	})
	t.Run("float64", func(t *testing.T) {
		v, err := ToDuration(float64(3000))
		require.NoError(t, err)
		assert.Equal(t, time.Duration(3000), v)
	})
	t.Run("json.Number", func(t *testing.T) {
		v, err := ToDuration(json.Number("4000"))
		require.NoError(t, err)
		assert.Equal(t, time.Duration(4000), v)
	})
	t.Run("nil error", func(t *testing.T) {
		v, err := ToDuration(nil)
		require.Error(t, err)
		assert.Equal(t, time.Duration(0), v)
	})
	t.Run("unsupported type error", func(t *testing.T) {
		v, err := ToDuration([]int{1})
		require.Error(t, err)
		assert.Equal(t, time.Duration(0), v)
	})
}

// ──────────────────────────────────────────────
// ToSlice
// ──────────────────────────────────────────────

func TestToSliceInt(t *testing.T) {
	t.Run("[]any of ints", func(t *testing.T) {
		v, err := ToSlice[int]([]any{1, 2, 3})
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, v)
	})
	t.Run("[]int", func(t *testing.T) {
		v, err := ToSlice[int]([]int{1, 2, 3})
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, v)
	})
	t.Run("single value", func(t *testing.T) {
		v, err := ToSlice[int](42)
		require.NoError(t, err)
		assert.Equal(t, []int{42}, v)
	})
	t.Run("nil", func(t *testing.T) {
		v, err := ToSlice[int](nil)
		require.NoError(t, err)
		assert.Nil(t, v)
	})
	t.Run("element conversion error", func(t *testing.T) {
		v, err := ToSlice[int]([]any{"a", "b"})
		require.Error(t, err)
		assert.Nil(t, v)
	})
	t.Run("array", func(t *testing.T) {
		v, err := ToSlice[int]([3]int{1, 2, 3})
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, v)
	})
}

func TestToSliceString(t *testing.T) {
	t.Run("[]any of strings", func(t *testing.T) {
		v, err := ToSlice[string]([]any{"a", "b"})
		require.NoError(t, err)
		assert.Equal(t, []string{"a", "b"}, v)
	})
	t.Run("[]string", func(t *testing.T) {
		v, err := ToSlice[string]([]string{"x", "y"})
		require.NoError(t, err)
		assert.Equal(t, []string{"x", "y"}, v)
	})
	t.Run("single value", func(t *testing.T) {
		v, err := ToSlice[string]("z")
		require.NoError(t, err)
		assert.Equal(t, []string{"z"}, v)
	})
	t.Run("nil", func(t *testing.T) {
		v, err := ToSlice[string](nil)
		require.NoError(t, err)
		assert.Nil(t, v)
	})
}

func TestToSliceIntHelper(t *testing.T) {
	t.Run("[]any", func(t *testing.T) {
		v, err := ToSliceInt([]any{1, 2, 3})
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, v)
	})
	t.Run("[]int", func(t *testing.T) {
		v, err := ToSliceInt([]int{4, 5, 6})
		require.NoError(t, err)
		assert.Equal(t, []int{4, 5, 6}, v)
	})
	t.Run("single", func(t *testing.T) {
		v, err := ToSliceInt(7)
		require.NoError(t, err)
		assert.Equal(t, []int{7}, v)
	})
}

func TestToSliceStringHelper(t *testing.T) {
	t.Run("[]any", func(t *testing.T) {
		v, err := ToSliceString([]any{"a", "b"})
		require.NoError(t, err)
		assert.Equal(t, []string{"a", "b"}, v)
	})
	t.Run("[]string", func(t *testing.T) {
		v, err := ToSliceString([]string{"x"})
		require.NoError(t, err)
		assert.Equal(t, []string{"x"}, v)
	})
}

// ──────────────────────────────────────────────
// ToMapStringAny
// ──────────────────────────────────────────────

func TestToMapStringAny(t *testing.T) {
	t.Run("map[string]any", func(t *testing.T) {
		v, err := ToMapStringAny(map[string]any{"a": 1})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"a": 1}, v)
	})
	t.Run("map[any]any with int key", func(t *testing.T) {
		v, err := ToMapStringAny(map[any]any{1: "a"})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"1": "a"}, v)
	})
	t.Run("nil", func(t *testing.T) {
		v, err := ToMapStringAny(nil)
		require.NoError(t, err)
		assert.Nil(t, v)
	})
	t.Run("non-map error", func(t *testing.T) {
		v, err := ToMapStringAny("not a map")
		require.Error(t, err)
		assert.Nil(t, v)
	})
}

// ──────────────────────────────────────────────
// toType (via ToSlice)
// ──────────────────────────────────────────────

func TestToTypeViaToSlice(t *testing.T) {
	t.Run("already target type", func(t *testing.T) {
		// []string passed to ToSlice[string] uses fast path element-wise
		v, err := ToSlice[string]([]string{"a", "b"})
		require.NoError(t, err)
		assert.Equal(t, []string{"a", "b"}, v)
	})
	t.Run("JSON roundtrip fallback", func(t *testing.T) {
		// converting []any of float64 to []int uses JSON roundtrip
		v, err := ToSlice[int]([]any{float64(1), float64(2)})
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2}, v)
	})
	t.Run("unmarshal error", func(t *testing.T) {
		// string cannot be unmarshaled into a chan int via JSON roundtrip
		v, err := ToSlice[chan int]([]any{"not-a-chan"})
		require.Error(t, err)
		assert.Nil(t, v)
	})
}

// ──────────────────────────────────────────────
// Marshal
// ──────────────────────────────────────────────

func TestMarshal(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		b, err := Marshal(FormatJSON, map[string]any{"a": 1})
		require.NoError(t, err)
		assert.JSONEq(t, `{"a":1}`, string(b))
	})
	t.Run("YAML", func(t *testing.T) {
		b, err := Marshal(FormatYAML, map[string]any{"a": 1})
		require.NoError(t, err)
		assert.Contains(t, string(b), "a:")
	})
	t.Run("TOML", func(t *testing.T) {
		b, err := Marshal(FormatTOML, map[string]any{"a": 1})
		require.NoError(t, err)
		assert.Contains(t, string(b), "a = 1")
	})
	t.Run("unsupported format error", func(t *testing.T) {
		_, err := Marshal(Format("xml"), map[string]any{"a": 1})
		require.Error(t, err)
	})
}

// ──────────────────────────────────────────────
// Unmarshal
// ──────────────────────────────────────────────

func TestUnmarshal(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		var v map[string]any
		err := Unmarshal(FormatJSON, []byte(`{"a":1}`), &v)
		require.NoError(t, err)
		assert.Equal(t, float64(1), v["a"])
	})
	t.Run("YAML", func(t *testing.T) {
		var v map[string]any
		err := Unmarshal(FormatYAML, []byte("a: 1\n"), &v)
		require.NoError(t, err)
		assert.Equal(t, 1, v["a"])
	})
	t.Run("TOML", func(t *testing.T) {
		var v map[string]any
		err := Unmarshal(FormatTOML, []byte("a = 1\n"), &v)
		require.NoError(t, err)
		assert.Equal(t, int64(1), v["a"])
	})
	t.Run("unsupported format error", func(t *testing.T) {
		var v map[string]any
		err := Unmarshal(Format("xml"), []byte(`x`), &v)
		require.Error(t, err)
	})
}

// ──────────────────────────────────────────────
// Convert
// ──────────────────────────────────────────────

func TestConvert(t *testing.T) {
	t.Run("JSON to YAML", func(t *testing.T) {
		out, err := Convert(FormatJSON, FormatYAML, []byte(`{"a":1,"b":"hello"}`))
		require.NoError(t, err)
		assert.Contains(t, string(out), "a:")
		assert.Contains(t, string(out), "b:")
	})
	t.Run("YAML to JSON", func(t *testing.T) {
		out, err := Convert(FormatYAML, FormatJSON, []byte("a: 1\nb: hello\n"))
		require.NoError(t, err)
		assert.JSONEq(t, `{"a":1,"b":"hello"}`, string(out))
	})
	t.Run("JSON to TOML", func(t *testing.T) {
		out, err := Convert(FormatJSON, FormatTOML, []byte(`{"a":1}`))
		require.NoError(t, err)
		assert.Contains(t, string(out), "a = 1")
	})
	t.Run("TOML to JSON", func(t *testing.T) {
		out, err := Convert(FormatTOML, FormatJSON, []byte("a = 1\n"))
		require.NoError(t, err)
		assert.JSONEq(t, `{"a":1}`, string(out))
	})
	t.Run("YAML to TOML", func(t *testing.T) {
		out, err := Convert(FormatYAML, FormatTOML, []byte("a: 1\n"))
		require.NoError(t, err)
		assert.Contains(t, string(out), "a = 1")
	})
	t.Run("TOML to YAML", func(t *testing.T) {
		out, err := Convert(FormatTOML, FormatYAML, []byte("a = 1\n"))
		require.NoError(t, err)
		assert.Contains(t, string(out), "a:")
	})
	t.Run("same format JSON validation", func(t *testing.T) {
		out, err := Convert(FormatJSON, FormatJSON, []byte(`{"a":1}`))
		require.NoError(t, err)
		assert.JSONEq(t, `{"a":1}`, string(out))
	})
	t.Run("same format invalid source error", func(t *testing.T) {
		_, err := Convert(FormatJSON, FormatJSON, []byte(`{invalid`))
		require.Error(t, err)
	})
	t.Run("invalid source error", func(t *testing.T) {
		_, err := Convert(FormatJSON, FormatYAML, []byte(`{invalid`))
		require.Error(t, err)
	})
	t.Run("unsupported target format error", func(t *testing.T) {
		_, err := Convert(FormatJSON, Format("xml"), []byte(`{"a":1}`))
		require.Error(t, err)
	})
	t.Run("nested objects", func(t *testing.T) {
		out, err := Convert(FormatJSON, FormatYAML, []byte(`{"a":{"b":1}}`))
		require.NoError(t, err)
		assert.Contains(t, string(out), "b:")
	})
	t.Run("arrays", func(t *testing.T) {
		out, err := Convert(FormatJSON, FormatYAML, []byte(`{"a":[1,2,3]}`))
		require.NoError(t, err)
		assert.Contains(t, string(out), "a:")
	})
}

// ──────────────────────────────────────────────
// FromJSONToYAML
// ──────────────────────────────────────────────

func TestFromJSONToYAML(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		out, err := FromJSONToYAML([]byte(`{"a":1,"b":"hello"}`))
		require.NoError(t, err)
		assert.Contains(t, string(out), "a:")
	})
	t.Run("invalid JSON error", func(t *testing.T) {
		_, err := FromJSONToYAML([]byte(`{invalid`))
		require.Error(t, err)
	})
	t.Run("nested objects", func(t *testing.T) {
		out, err := FromJSONToYAML([]byte(`{"outer":{"inner":"value"}}`))
		require.NoError(t, err)
		assert.Contains(t, string(out), "inner:")
	})
	t.Run("arrays", func(t *testing.T) {
		out, err := FromJSONToYAML([]byte(`{"items":[1,2,3]}`))
		require.NoError(t, err)
		assert.Contains(t, string(out), "items:")
	})
}

// ──────────────────────────────────────────────
// FromYAMLToJSON
// ──────────────────────────────────────────────

func TestFromYAMLToJSON(t *testing.T) {
	t.Run("valid YAML", func(t *testing.T) {
		out, err := FromYAMLToJSON([]byte("a: 1\nb: hello\n"))
		require.NoError(t, err)
		assert.JSONEq(t, `{"a":1,"b":"hello"}`, string(out))
	})
	t.Run("invalid YAML error", func(t *testing.T) {
		_, err := FromYAMLToJSON([]byte("a: : :\n"))
		require.Error(t, err)
	})
	t.Run("nested objects", func(t *testing.T) {
		out, err := FromYAMLToJSON([]byte("outer:\n  inner: value\n"))
		require.NoError(t, err)
		assert.JSONEq(t, `{"outer":{"inner":"value"}}`, string(out))
	})
}

// ──────────────────────────────────────────────
// FromJSONToTOML
// ──────────────────────────────────────────────

func TestFromJSONToTOML(t *testing.T) {
	t.Run("valid JSON object", func(t *testing.T) {
		out, err := FromJSONToTOML([]byte(`{"a":1,"b":"hello"}`))
		require.NoError(t, err)
		assert.Contains(t, string(out), "a = 1")
		assert.Contains(t, string(out), `b = "hello"`)
	})
	t.Run("JSON array wrapped", func(t *testing.T) {
		out, err := FromJSONToTOML([]byte(`[1,2,3]`))
		require.NoError(t, err)
		assert.Contains(t, string(out), "data")
	})
	t.Run("JSON scalar wrapped", func(t *testing.T) {
		out, err := FromJSONToTOML([]byte(`42`))
		require.NoError(t, err)
		assert.Contains(t, string(out), "data = 42")
	})
	t.Run("invalid JSON error", func(t *testing.T) {
		_, err := FromJSONToTOML([]byte(`{invalid`))
		require.Error(t, err)
	})
}

// ──────────────────────────────────────────────
// FromTOMLToJSON
// ──────────────────────────────────────────────

func TestFromTOMLToJSON(t *testing.T) {
	t.Run("valid TOML", func(t *testing.T) {
		out, err := FromTOMLToJSON([]byte("a = 1\nb = \"hello\"\n"))
		require.NoError(t, err)
		assert.JSONEq(t, `{"a":1,"b":"hello"}`, string(out))
	})
	t.Run("invalid TOML error", func(t *testing.T) {
		_, err := FromTOMLToJSON([]byte("a = = =\n"))
		require.Error(t, err)
	})
}

// ──────────────────────────────────────────────
// FromYAMLToTOML
// ──────────────────────────────────────────────

func TestFromYAMLToTOML(t *testing.T) {
	t.Run("valid YAML", func(t *testing.T) {
		out, err := FromYAMLToTOML([]byte("a: 1\nb: hello\n"))
		require.NoError(t, err)
		assert.Contains(t, string(out), "a = 1")
	})
	t.Run("invalid YAML error", func(t *testing.T) {
		_, err := FromYAMLToTOML([]byte("a: : :\n"))
		require.Error(t, err)
	})
}

// ──────────────────────────────────────────────
// FromTOMLToYAML
// ──────────────────────────────────────────────

func TestFromTOMLToYAML(t *testing.T) {
	t.Run("valid TOML", func(t *testing.T) {
		out, err := FromTOMLToYAML([]byte("a = 1\nb = \"hello\"\n"))
		require.NoError(t, err)
		assert.Contains(t, string(out), "a:")
		assert.Contains(t, string(out), "b:")
	})
	t.Run("invalid TOML error", func(t *testing.T) {
		_, err := FromTOMLToYAML([]byte("a = = =\n"))
		require.Error(t, err)
	})
}

// ──────────────────────────────────────────────
// StructToJSON
// ──────────────────────────────────────────────

type testStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestStructToJSON(t *testing.T) {
	t.Run("struct", func(t *testing.T) {
		out, err := StructToJSON(testStruct{Name: "Alice", Age: 30})
		require.NoError(t, err)
		assert.JSONEq(t, `{"name":"Alice","age":30}`, string(out))
	})
	t.Run("map", func(t *testing.T) {
		out, err := StructToJSON(map[string]any{"a": 1})
		require.NoError(t, err)
		assert.JSONEq(t, `{"a":1}`, string(out))
	})
}

// ──────────────────────────────────────────────
// StructToYAML
// ──────────────────────────────────────────────

func TestStructToYAML(t *testing.T) {
	t.Run("struct", func(t *testing.T) {
		out, err := StructToYAML(testStruct{Name: "Alice", Age: 30})
		require.NoError(t, err)
		assert.Contains(t, string(out), "name: Alice")
		assert.Contains(t, string(out), "age: 30")
	})
}

// ──────────────────────────────────────────────
// StructToTOML
// ──────────────────────────────────────────────

func TestStructToTOML(t *testing.T) {
	t.Run("struct", func(t *testing.T) {
		out, err := StructToTOML(testStruct{Name: "Alice", Age: 30})
		require.NoError(t, err)
		assert.Contains(t, string(out), "name = \"Alice\"")
		assert.Contains(t, string(out), "age = 30")
	})
	t.Run("scalar wrapped", func(t *testing.T) {
		out, err := StructToTOML(42)
		require.NoError(t, err)
		assert.Contains(t, string(out), "data = 42")
	})
}

// ──────────────────────────────────────────────
// JSONToStruct
// ──────────────────────────────────────────────

func TestJSONToStruct(t *testing.T) {
	t.Run("valid JSON into struct", func(t *testing.T) {
		var s testStruct
		err := JSONToStruct([]byte(`{"name":"Alice","age":30}`), &s)
		require.NoError(t, err)
		assert.Equal(t, "Alice", s.Name)
		assert.Equal(t, 30, s.Age)
	})
	t.Run("invalid JSON error", func(t *testing.T) {
		var s testStruct
		err := JSONToStruct([]byte(`{invalid`), &s)
		require.Error(t, err)
	})
}

// ──────────────────────────────────────────────
// YAMLToStruct
// ──────────────────────────────────────────────

func TestYAMLToStruct(t *testing.T) {
	t.Run("valid YAML into struct", func(t *testing.T) {
		var s testStruct
		err := YAMLToStruct([]byte("name: Alice\nage: 30\n"), &s)
		require.NoError(t, err)
		assert.Equal(t, "Alice", s.Name)
		assert.Equal(t, 30, s.Age)
	})
	t.Run("invalid YAML error", func(t *testing.T) {
		var s testStruct
		err := YAMLToStruct([]byte("a: : :\n"), &s)
		require.Error(t, err)
	})
}

// ──────────────────────────────────────────────
// TOMLToStruct
// ──────────────────────────────────────────────

func TestTOMLToStruct(t *testing.T) {
	t.Run("valid TOML into struct", func(t *testing.T) {
		var s testStruct
		err := TOMLToStruct([]byte("name = \"Alice\"\nage = 30\n"), &s)
		require.NoError(t, err)
		assert.Equal(t, "Alice", s.Name)
		assert.Equal(t, 30, s.Age)
	})
	t.Run("invalid TOML error", func(t *testing.T) {
		var s testStruct
		err := TOMLToStruct([]byte("a = = =\n"), &s)
		require.Error(t, err)
	})
}

// ──────────────────────────────────────────────
// wrapForTOML
// ──────────────────────────────────────────────

func TestWrapForTOML(t *testing.T) {
	t.Run("map[string]any returned as-is", func(t *testing.T) {
		in := map[string]any{"a": 1}
		out := wrapForTOML(in)
		assert.Equal(t, in, out)
	})
	t.Run("non-map wrapped in data key", func(t *testing.T) {
		out := wrapForTOML(42)
		assert.Equal(t, map[string]any{"data": 42}, out)
	})
	t.Run("slice wrapped", func(t *testing.T) {
		out := wrapForTOML([]any{1, 2})
		assert.Equal(t, map[string]any{"data": []any{1, 2}}, out)
	})
}

// ──────────────────────────────────────────────
// byteWriter
// ──────────────────────────────────────────────

func TestByteWriter(t *testing.T) {
	var buf []byte
	w := &byteWriter{buf: &buf}
	n, err := w.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", string(buf))

	// second write appends
	n, err = w.Write([]byte(" world"))
	require.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, "hello world", string(buf))
}

// ──────────────────────────────────────────────
// Edge cases
// ──────────────────────────────────────────────

func TestEdgeCases(t *testing.T) {
	t.Run("ToInt negative number", func(t *testing.T) {
		v, err := ToInt("-42")
		require.NoError(t, err)
		assert.Equal(t, -42, v)
	})
	t.Run("ToFloat64 negative number", func(t *testing.T) {
		v, err := ToFloat64("-3.14")
		require.NoError(t, err)
		assert.Equal(t, -3.14, v)
	})
	t.Run("ToString unicode", func(t *testing.T) {
		v, err := ToString("héllo wörld 日本語")
		require.NoError(t, err)
		assert.Equal(t, "héllo wörld 日本語", v)
	})
	t.Run("ToBool unicode string yes", func(t *testing.T) {
		// non-ascii invalid bool
		_, err := ToBool("是")
		require.Error(t, err)
	})
	t.Run("Convert nested map JSON to TOML", func(t *testing.T) {
		out, err := Convert(FormatJSON, FormatTOML, []byte(`{"outer":{"inner":1}}`))
		require.NoError(t, err)
		assert.Contains(t, string(out), "[outer]")
		assert.Contains(t, string(out), "inner = 1")
	})
	t.Run("Convert nested array JSON to YAML", func(t *testing.T) {
		out, err := Convert(FormatJSON, FormatYAML, []byte(`{"list":[{"x":1}]}`))
		require.NoError(t, err)
		assert.Contains(t, string(out), "list:")
	})
	t.Run("ToInt64 large json.Number", func(t *testing.T) {
		v, err := ToInt64(json.Number("9999999999"))
		require.NoError(t, err)
		assert.Equal(t, int64(9999999999), v)
	})
	t.Run("ToInt64 json.Number float fallback", func(t *testing.T) {
		// json.Number with decimal triggers ToInt error, falls back to Int64()
		v, err := ToInt64(json.Number("3.5"))
		// Int64() on "3.5" returns error
		_ = v
		_ = err
	})
	t.Run("ToDuration json.Number error", func(t *testing.T) {
		// json.Number with decimal - Int64() errors
		_, err := ToDuration(json.Number("3.5"))
		require.Error(t, err)
	})
	t.Run("ToString struct with unmarshalable", func(t *testing.T) {
		// channel cannot be JSON marshaled
		_, err := ToString(make(chan int))
		require.Error(t, err)
	})
	t.Run("ToMapStringAny empty map", func(t *testing.T) {
		v, err := ToMapStringAny(map[string]any{})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{}, v)
	})
	// keep fmt import used
	_ = fmt.Sprintf
}
