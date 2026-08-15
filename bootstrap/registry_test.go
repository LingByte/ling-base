// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package bootstrap

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testComponent struct{ name string }
type anotherComponent struct{ val int }

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	c := &testComponent{name: "hello"}
	err := r.Register("comp1", c)
	assert.NoError(t, err)

	got, ok := r.Get("comp1")
	assert.True(t, ok)
	assert.Equal(t, c, got)
}

func TestRegistry_DuplicateName(t *testing.T) {
	r := NewRegistry()
	r.Register("comp", &testComponent{})
	err := r.Register("comp", &testComponent{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestRegistry_EmptyName(t *testing.T) {
	r := NewRegistry()
	err := r.Register("", &testComponent{})
	assert.Error(t, err)
}

func TestRegistry_NilComponent(t *testing.T) {
	r := NewRegistry()
	err := r.Register("comp", nil)
	assert.Error(t, err)
}

func TestRegistry_MustRegister(t *testing.T) {
	r := NewRegistry()
	assert.NotPanics(t, func() {
		r.MustRegister("comp", &testComponent{})
	})
	assert.Panics(t, func() {
		r.MustRegister("comp", &testComponent{})
	})
}

func TestRegistry_MustGet(t *testing.T) {
	r := NewRegistry()
	r.Register("comp", &testComponent{})
	assert.NotPanics(t, func() {
		r.MustGet("comp")
	})
	assert.Panics(t, func() {
		r.MustGet("nonexistent")
	})
}

func TestRegistry_GetNotFound(t *testing.T) {
	r := NewRegistry()
	_, ok := r.Get("nonexistent")
	assert.False(t, ok)
}

func TestRegistry_GetByType(t *testing.T) {
	r := NewRegistry()
	c := &testComponent{name: "test"}
	r.Register("comp", c)

	got, ok := r.GetByType(reflect.TypeOf(c))
	assert.True(t, ok)
	assert.Equal(t, c, got)
}

func TestRegistry_GetAllByType(t *testing.T) {
	r := NewRegistry()
	c1 := &testComponent{name: "a"}
	c2 := &testComponent{name: "b"}
	r.Register("c1", c1)
	r.Register("c2", c2)

	all := r.GetAllByType(reflect.TypeOf(c1))
	assert.Len(t, all, 2)
}

func TestRegistry_Names(t *testing.T) {
	r := NewRegistry()
	r.Register("c1", &testComponent{})
	r.Register("c2", &testComponent{})
	r.Register("c3", &anotherComponent{})

	names := r.Names()
	assert.Equal(t, []string{"c1", "c2", "c3"}, names)
}

func TestRegistry_Count(t *testing.T) {
	r := NewRegistry()
	r.Register("c1", &testComponent{})
	r.Register("c2", &testComponent{})
	assert.Equal(t, 2, r.Count())
}

func TestRegistry_Contains(t *testing.T) {
	r := NewRegistry()
	r.Register("c1", &testComponent{})
	assert.True(t, r.Contains("c1"))
	assert.False(t, r.Contains("c2"))
}

func TestRegistry_Unregister(t *testing.T) {
	r := NewRegistry()
	r.Register("c1", &testComponent{})
	r.Register("c2", &testComponent{})

	ok := r.Unregister("c1")
	assert.True(t, ok)
	assert.False(t, r.Contains("c1"))
	assert.True(t, r.Contains("c2"))
	assert.Equal(t, 1, r.Count())

	ok = r.Unregister("nonexistent")
	assert.False(t, ok)
}

func TestRegistry_Range(t *testing.T) {
	r := NewRegistry()
	r.Register("c1", &testComponent{name: "a"})
	r.Register("c2", &testComponent{name: "b"})

	var visited []string
	r.Range(func(name string, component any) bool {
		visited = append(visited, name)
		return true
	})
	assert.Equal(t, []string{"c1", "c2"}, visited)
}

func TestRegistry_RangeStop(t *testing.T) {
	r := NewRegistry()
	r.Register("c1", &testComponent{})
	r.Register("c2", &testComponent{})
	r.Register("c3", &testComponent{})

	var count int
	r.Range(func(name string, component any) bool {
		count++
		return count < 2 // stop after 2
	})
	assert.Equal(t, 2, count)
}
