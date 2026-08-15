// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package null

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_AlwaysAllows(t *testing.T) {
	l := New()
	for i := 0; i < 1000; i++ {
		assert.NoError(t, l.Acquire(context.Background(), nil))
	}
}

func TestNew_Running(t *testing.T) {
	l := New()
	assert.Equal(t, -1, l.Running())
}

func TestNew_ReleaseNoOp(t *testing.T) {
	l := New()
	assert.NotPanics(t, func() {
		l.Release(nil)
		l.Release([]byte("anything"))
	})
}

func TestNewString_AlwaysAllows(t *testing.T) {
	l := NewString()
	for i := 0; i < 100; i++ {
		assert.NoError(t, l.Acquire(context.Background(), "key"))
	}
}

func TestNewString_ReleaseNoOp(t *testing.T) {
	l := NewString()
	assert.NotPanics(t, func() {
		l.Release("key")
	})
}

func TestNewSize_AlwaysAllows(t *testing.T) {
	l := NewSize()
	assert.NoError(t, l.Acquire(context.Background(), nil, 1<<30))
	assert.NoError(t, l.Acquire(context.Background(), []byte("k"), 999999))
}

func TestNewSize_Remaining(t *testing.T) {
	l := NewSize()
	assert.Equal(t, int64(-1), l.Remaining(nil))
}

func TestNewSize_ReleaseNoOp(t *testing.T) {
	l := NewSize()
	assert.NotPanics(t, func() {
		l.Release(nil, 100)
	})
}
