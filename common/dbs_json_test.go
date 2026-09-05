// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package common

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
)

// ──────────────────────────────────────────────
// StringArray tests
// ──────────────────────────────────────────────

func TestStringArray_Value(t *testing.T) {
	tests := []struct {
		name  string
		input StringArray
		want  driver.Value
	}{
		{"nil", nil, "[]"},
		{"empty", StringArray{}, "[]"},
		{"single", StringArray{"a"}, `["a"]`},
		{"multiple", StringArray{"a", "b", "c"}, `["a","b","c"]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.input.Value()
			if err != nil {
				t.Fatalf("Value: %v", err)
			}
			if got != tt.want {
				t.Errorf("Value = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringArray_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    StringArray
		wantErr bool
	}{
		{"nil", nil, nil, false},
		{"empty bytes", []byte{}, nil, false},
		{"empty string", "", nil, false},
		{"valid bytes", []byte(`["a","b"]`), StringArray{"a", "b"}, false},
		{"valid string", `["x","y","z"]`, StringArray{"x", "y", "z"}, false},
		{"invalid json", []byte(`not json`), nil, true},
		{"unsupported type", 123, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s StringArray
			err := s.Scan(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("Scan should return error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}
			if len(s) != len(tt.want) {
				t.Errorf("Scan result len = %d, want %d", len(s), len(tt.want))
				return
			}
			for i := range s {
				if s[i] != tt.want[i] {
					t.Errorf("Scan[%d] = %q, want %q", i, s[i], tt.want[i])
				}
			}
		})
	}
}

func TestStringArray_MarshalJSON(t *testing.T) {
	s := StringArray{"a", "b", "c"}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(data) != `["a","b","c"]` {
		t.Errorf("MarshalJSON = %s", data)
	}

	// nil should marshal as null
	var nilS StringArray
	data, err = json.Marshal(nilS)
	if err != nil {
		t.Fatalf("MarshalJSON nil: %v", err)
	}
	if string(data) != "null" {
		t.Errorf("MarshalJSON nil = %s, want null", data)
	}
}

func TestStringArray_UnmarshalJSON(t *testing.T) {
	var s StringArray
	err := json.Unmarshal([]byte(`["a","b","c"]`), &s)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if len(s) != 3 || s[0] != "a" || s[1] != "b" || s[2] != "c" {
		t.Errorf("UnmarshalJSON result = %v", s)
	}

	// null should set nil
	err = json.Unmarshal([]byte("null"), &s)
	if err != nil {
		t.Fatalf("UnmarshalJSON null: %v", err)
	}
	if s != nil {
		t.Errorf("UnmarshalJSON null: s = %v, want nil", s)
	}
}

func TestStringArray_RoundTrip(t *testing.T) {
	original := StringArray{"alpha", "beta", "gamma"}

	// Value → Scan round trip.
	val, err := original.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}

	var decoded StringArray
	if err := decoded.Scan(val); err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(decoded) != len(original) {
		t.Fatalf("Round trip: len = %d, want %d", len(decoded), len(original))
	}
	for i := range original {
		if decoded[i] != original[i] {
			t.Errorf("Round trip[%d]: %q != %q", i, decoded[i], original[i])
		}
	}
}

// ──────────────────────────────────────────────
// IntArray tests
// ──────────────────────────────────────────────

func TestIntArray_Value(t *testing.T) {
	a := IntArray{1, 2, 3}
	val, err := a.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if val != "[1,2,3]" {
		t.Errorf("Value = %v, want [1,2,3]", val)
	}

	var nilA IntArray
	val, err = nilA.Value()
	if err != nil {
		t.Fatalf("Value nil: %v", err)
	}
	if val != "[]" {
		t.Errorf("Value nil = %v, want []", val)
	}
}

func TestIntArray_Scan(t *testing.T) {
	var a IntArray
	err := a.Scan([]byte("[1,2,3]"))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(a) != 3 || a[0] != 1 || a[1] != 2 || a[2] != 3 {
		t.Errorf("Scan result = %v", a)
	}

	err = a.Scan(nil)
	if err != nil {
		t.Fatalf("Scan nil: %v", err)
	}
	if a != nil {
		t.Errorf("Scan nil: a = %v, want nil", a)
	}

	err = a.Scan(123)
	if err == nil {
		t.Error("Scan int should fail")
	}
}

func TestIntArray_MarshalJSON(t *testing.T) {
	a := IntArray{1, 2, 3}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(data) != "[1,2,3]" {
		t.Errorf("MarshalJSON = %s", data)
	}
}

func TestIntArray_UnmarshalJSON(t *testing.T) {
	var a IntArray
	err := json.Unmarshal([]byte("[10,20,30]"), &a)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if len(a) != 3 || a[0] != 10 || a[1] != 20 || a[2] != 30 {
		t.Errorf("UnmarshalJSON result = %v", a)
	}
}

// ──────────────────────────────────────────────
// JSONMap tests
// ──────────────────────────────────────────────

func TestJSONMap_Value(t *testing.T) {
	m := JSONMap{"key": "value", "num": float64(42)}
	val, err := m.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if val == nil {
		t.Error("Value should not be nil")
	}

	var nilM JSONMap
	val, err = nilM.Value()
	if err != nil {
		t.Fatalf("Value nil: %v", err)
	}
	if val != nil {
		t.Errorf("Value nil = %v, want nil", val)
	}
}

func TestJSONMap_Scan(t *testing.T) {
	var m JSONMap
	err := m.Scan([]byte(`{"name":"test","count":5}`))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if m["name"] != "test" {
		t.Errorf("Scan name = %v", m["name"])
	}

	err = m.Scan(nil)
	if err != nil {
		t.Fatalf("Scan nil: %v", err)
	}
	if m != nil {
		t.Errorf("Scan nil: m = %v, want nil", m)
	}

	err = m.Scan(123)
	if err == nil {
		t.Error("Scan int should fail")
	}
}

func TestJSONMap_MarshalJSON(t *testing.T) {
	m := JSONMap{"key": "value"}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(data) != `{"key":"value"}` {
		t.Errorf("MarshalJSON = %s", data)
	}
}

func TestJSONMap_UnmarshalJSON(t *testing.T) {
	var m JSONMap
	err := json.Unmarshal([]byte(`{"a":1,"b":"hello"}`), &m)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if m["a"] == nil || m["b"] != "hello" {
		t.Errorf("UnmarshalJSON result = %v", m)
	}
}

// ──────────────────────────────────────────────
// JSONRaw tests
// ──────────────────────────────────────────────

func TestJSONRaw_Value(t *testing.T) {
	r := JSONRaw(`{"key":"value"}`)
	val, err := r.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if val != `{"key":"value"}` {
		t.Errorf("Value = %v", val)
	}

	var nilR JSONRaw
	val, err = nilR.Value()
	if err != nil {
		t.Fatalf("Value nil: %v", err)
	}
	if val != nil {
		t.Errorf("Value nil = %v, want nil", val)
	}
}

func TestJSONRaw_Scan(t *testing.T) {
	var r JSONRaw
	err := r.Scan([]byte(`{"name":"test"}`))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if string(r) != `{"name":"test"}` {
		t.Errorf("Scan result = %s", r)
	}

	err = r.Scan([]byte(`not valid json`))
	if err == nil {
		t.Error("Scan invalid JSON should fail")
	}

	err = r.Scan(nil)
	if err != nil {
		t.Fatalf("Scan nil: %v", err)
	}
	if r != nil {
		t.Errorf("Scan nil: r = %v, want nil", r)
	}
}

func TestJSONRaw_MarshalJSON(t *testing.T) {
	r := JSONRaw(`{"key":"value"}`)
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(data) != `{"key":"value"}` {
		t.Errorf("MarshalJSON = %s", data)
	}
}

func TestJSONRaw_UnmarshalJSON(t *testing.T) {
	var r JSONRaw
	err := json.Unmarshal([]byte(`{"a":1}`), &r)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if string(r) != `{"a":1}` {
		t.Errorf("UnmarshalJSON result = %s", r)
	}
}
