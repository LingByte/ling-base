// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package pool

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testObject struct {
	id int
}

func newObjectFactory(counter *atomic.Int32) Factory[*testObject] {
	return func() (*testObject, error) {
		id := counter.Add(1)
		return &testObject{id: int(id)}, nil
	}
}

func TestObjectPoolGetPut(t *testing.T) {
	var created atomic.Int32
	var destroyed atomic.Int32
	factory := newObjectFactory(&created)
	destroyer := func(o *testObject) { destroyed.Add(1) }

	p := NewObjectPool[*testObject](factory, 4, destroyer)
	defer p.Close()

	o1, err := p.Get()
	require.NoError(t, err)
	require.NotNil(t, o1)
	assert.Equal(t, int32(1), created.Load())

	p.Put(o1)

	o2, err := p.Get()
	require.NoError(t, err)
	require.Same(t, o1, o2)
	assert.Equal(t, int32(1), created.Load(), "expected reuse without new creation")

	p.Put(o2)
	assert.Equal(t, int32(0), destroyed.Load())

	stats := p.Stats()
	assert.Equal(t, int32(1), stats.OpenCount)
	assert.Equal(t, 1, stats.IdleCount)
	assert.Equal(t, int32(0), stats.InUse)
}

func TestObjectPoolFactoryError(t *testing.T) {
	factoryErr := errors.New("factory boom")
	factory := func() (*testObject, error) { return nil, factoryErr }
	p := NewObjectPool[*testObject](factory, 2, nil)
	defer p.Close()

	o, err := p.Get()
	require.ErrorIs(t, err, factoryErr)
	assert.Nil(t, o)

	assert.Equal(t, int32(0), p.Stats().OpenCount)
}

func TestObjectPoolMaxCapacityBlocking(t *testing.T) {
	var created atomic.Int32
	factory := newObjectFactory(&created)
	p := NewObjectPool[*testObject](factory, 1, nil)
	defer p.Close()

	o1, err := p.Get()
	require.NoError(t, err)
	assert.Equal(t, int32(1), created.Load())

	done := make(chan *testObject, 1)
	go func() {
		o, err := p.Get()
		if err != nil {
			done <- nil
			return
		}
		done <- o
	}()

	select {
	case <-done:
		t.Fatal("Get should block when at capacity")
	case <-time.After(50 * time.Millisecond):
	}

	p.Put(o1)

	select {
	case o2 := <-done:
		require.NotNil(t, o2)
		assert.Same(t, o1, o2, "blocked Get should receive the returned object")
	case <-time.After(time.Second):
		t.Fatal("Get did not unblock after Put")
	}
}

func TestObjectPoolCloseBehavior(t *testing.T) {
	var created atomic.Int32
	var destroyed atomic.Int32
	factory := newObjectFactory(&created)
	destroyer := func(o *testObject) { destroyed.Add(1) }

	p := NewObjectPool[*testObject](factory, 2, destroyer)

	o1, err := p.Get()
	require.NoError(t, err)
	o2, err := p.Get()
	require.NoError(t, err)
	p.Put(o1)
	p.Put(o2)

	p.Close()

	o, err := p.Get()
	require.ErrorIs(t, err, ErrPoolClosed)
	assert.Nil(t, o)

	assert.Equal(t, int32(2), destroyed.Load(), "idle objects should be destroyed on close")
}

func TestObjectPoolPutAfterCloseDestroys(t *testing.T) {
	var created atomic.Int32
	var destroyed atomic.Int32
	factory := newObjectFactory(&created)
	destroyer := func(o *testObject) { destroyed.Add(1) }

	p := NewObjectPool[*testObject](factory, 2, destroyer)

	o1, err := p.Get()
	require.NoError(t, err)

	p.Close()
	p.Put(o1)

	assert.Equal(t, int32(1), destroyed.Load(), "Put after close should destroy the object")
}

func TestObjectPoolConcurrentAccess(t *testing.T) {
	const maxOpen = 8
	const goroutines = 50
	const iterations = 100

	var created atomic.Int32
	factory := newObjectFactory(&created)
	p := NewObjectPool[*testObject](factory, maxOpen, nil)
	defer p.Close()

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				o, err := p.Get()
				if err != nil {
					continue
				}
				p.Put(o)
			}
		}()
	}
	wg.Wait()

	stats := p.Stats()
	assert.LessOrEqual(t, stats.OpenCount, int32(maxOpen))
	assert.Equal(t, stats.IdleCount+int(stats.InUse), int(stats.OpenCount))
	assert.Greater(t, created.Load(), int32(0))
}

func TestObjectPoolDoubleClose(t *testing.T) {
	factory := newObjectFactory(&atomic.Int32{})
	p := NewObjectPool[*testObject](factory, 2, nil)

	require.NotPanics(t, func() {
		p.Close()
		p.Close()
	})
}

func TestObjectPoolStatsAfterOperations(t *testing.T) {
	var created atomic.Int32
	var destroyed atomic.Int32
	factory := newObjectFactory(&created)
	destroyer := func(o *testObject) { destroyed.Add(1) }

	p := NewObjectPool[*testObject](factory, 4, destroyer)
	defer p.Close()

	// Initial stats — nothing created yet.
	stats := p.Stats()
	assert.Equal(t, int32(0), stats.OpenCount)
	assert.Equal(t, 0, stats.IdleCount)
	assert.Equal(t, int32(0), stats.InUse)

	// Get two objects — both in use.
	o1, err := p.Get()
	require.NoError(t, err)
	o2, err := p.Get()
	require.NoError(t, err)

	stats = p.Stats()
	assert.Equal(t, int32(2), stats.OpenCount)
	assert.Equal(t, 0, stats.IdleCount)
	assert.Equal(t, int32(2), stats.InUse)

	// Put one back — one idle, one in use.
	p.Put(o1)
	stats = p.Stats()
	assert.Equal(t, int32(2), stats.OpenCount)
	assert.Equal(t, 1, stats.IdleCount)
	assert.Equal(t, int32(1), stats.InUse)

	// Put the other back — both idle.
	p.Put(o2)
	stats = p.Stats()
	assert.Equal(t, int32(2), stats.OpenCount)
	assert.Equal(t, 2, stats.IdleCount)
	assert.Equal(t, int32(0), stats.InUse)
}

func TestObjectPoolPutOverflowDestroys(t *testing.T) {
	var created atomic.Int32
	var destroyed atomic.Int32
	factory := newObjectFactory(&created)
	destroyer := func(o *testObject) { destroyed.Add(1) }

	// maxOpen=1, idle channel capacity=1.
	p := NewObjectPool[*testObject](factory, 1, destroyer)
	defer p.Close()

	// Get the only object, then return it — idle is now full.
	o1, err := p.Get()
	require.NoError(t, err)
	p.Put(o1)
	require.Equal(t, int32(0), destroyed.Load())

	// Put the same object again — idle is full, so it overflows and destroys.
	p.Put(o1)
	assert.Equal(t, int32(1), destroyed.Load(), "overflow Put should destroy the object")

	// openCount should have been decremented.
	stats := p.Stats()
	assert.Equal(t, int32(0), stats.OpenCount)
}

func TestObjectPoolGetAfterCloseBlocks(t *testing.T) {
	var created atomic.Int32
	factory := newObjectFactory(&created)
	p := NewObjectPool[*testObject](factory, 1, nil)

	// Get the only object, don't return it.
	o, err := p.Get()
	require.NoError(t, err)

	// Close the pool — this should unblock any waiting Get.
	p.Close()

	// Now Get should return ErrPoolClosed immediately.
	o2, err := p.Get()
	require.ErrorIs(t, err, ErrPoolClosed)
	assert.Nil(t, o2)

	// Return the object we held — should be destroyed, not pooled.
	p.Put(o)
}

func TestObjectPoolStatsInUseNonNegative(t *testing.T) {
	var created atomic.Int32
	factory := newObjectFactory(&created)
	p := NewObjectPool[*testObject](factory, 2, nil)

	// Get an object, then manually put it back and close — Stats should
	// never report negative InUse.
	o, err := p.Get()
	require.NoError(t, err)
	p.Put(o)
	p.Close()

	stats := p.Stats()
	assert.GreaterOrEqual(t, stats.InUse, int32(0), "InUse should never be negative")
	assert.Equal(t, int32(0), stats.OpenCount)
	assert.Equal(t, 0, stats.IdleCount)
}
