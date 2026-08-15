// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package keysize

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_BasicAcquireRelease(t *testing.T) {
	l := New(1000)
	assert.Equal(t, int64(0), l.Running())

	assert.NoError(t, l.Acquire(context.Background(), []byte("a"), 100))
	assert.NoError(t, l.Acquire(context.Background(), []byte("a"), 200))
	assert.Equal(t, int64(300), l.Running())
}

func TestNew_PerKeyLimit(t *testing.T) {
	l := New(100)
	assert.NoError(t, l.Acquire(context.Background(), []byte("a"), 60))
	err := l.Acquire(context.Background(), []byte("a"), 50)
	assert.Error(t, err)

	// Different key should still work.
	assert.NoError(t, l.Acquire(context.Background(), []byte("b"), 60))
}

func TestNew_ReleaseRestores(t *testing.T) {
	l := New(100)
	assert.NoError(t, l.Acquire(context.Background(), []byte("a"), 80))
	assert.Error(t, l.Acquire(context.Background(), []byte("a"), 30))
	l.Release([]byte("a"), 80)
	assert.NoError(t, l.Acquire(context.Background(), []byte("a"), 30))
}

func TestNew_Remaining(t *testing.T) {
	l := New(1000)
	assert.Equal(t, int64(1000), l.Remaining([]byte("a")))
	l.Acquire(context.Background(), []byte("a"), 300)
	assert.Equal(t, int64(700), l.Remaining([]byte("a")))
}

func TestNew_InvalidSize(t *testing.T) {
	l := New(100)
	err := l.Acquire(context.Background(), []byte("a"), 0)
	assert.Error(t, err)
	err = l.Acquire(context.Background(), []byte("a"), -1)
	assert.Error(t, err)
}

func TestNew_ReleaseCleansUp(t *testing.T) {
	l := New(100)
	l.Acquire(context.Background(), []byte("a"), 50)
	l.Release([]byte("a"), 50)
	assert.Equal(t, int64(0), l.Running())
}

func TestNew_ReleaseUnknownKey(t *testing.T) {
	l := New(100)
	assert.NotPanics(t, func() {
		l.Release([]byte("nonexistent"), 50)
	})
}

func TestNew_Concurrent(t *testing.T) {
	l := New(10000)
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			l.Acquire(context.Background(), []byte("k"), 50)
			l.Release([]byte("k"), 50)
			done <- true
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}
	assert.Equal(t, int64(0), l.Running())
}

// ===== KeySizeWriter =====

func TestKeySizeWriter_Basic(t *testing.T) {
	l := New(100)
	var buf bytes.Buffer
	w := NewWriter(l, []byte("a"), &nopCloser{&buf})

	n, err := w.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, int64(5), w.Written())
	assert.Equal(t, int64(95), l.Remaining([]byte("a")))

	assert.NoError(t, w.Close())
	// After close, the quota is released.
	assert.Equal(t, int64(100), l.Remaining([]byte("a")))
}

func TestKeySizeWriter_LimitExceeded(t *testing.T) {
	l := New(10)
	var buf bytes.Buffer
	w := NewWriter(l, []byte("a"), &nopCloser{&buf})

	_, err := w.Write([]byte("0123456789")) // exactly 10
	assert.NoError(t, err)

	_, err = w.Write([]byte("X")) // exceeds
	assert.Error(t, err)
	w.Close()
}

func TestKeySizeWriter_CloseIdempotent(t *testing.T) {
	l := New(100)
	var buf bytes.Buffer
	w := NewWriter(l, []byte("a"), &nopCloser{&buf})
	w.Write([]byte("test"))
	assert.NoError(t, w.Close())
	assert.NoError(t, w.Close()) // double close is safe
}

func TestKeySizeWriter_Written(t *testing.T) {
	l := New(1000)
	var buf bytes.Buffer
	w := NewWriter(l, []byte("a"), &nopCloser{&buf})
	w.Write([]byte("hello world"))
	assert.Equal(t, int64(11), w.Written())
	w.Close()
}

// nopCloser wraps a Writer to make it a WriteCloser.
type nopCloser struct {
	io.Writer
}

func (n *nopCloser) Close() error { return nil }
