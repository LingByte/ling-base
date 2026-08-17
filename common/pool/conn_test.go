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

var (
	errHealthCheck = errors.New("conn: health check failed")
	errFactory     = errors.New("conn: factory error")
)

type mockConn struct {
	id      int
	healthy bool
}

func newConnFactory(counter *atomic.Int32, healthy bool) Factory[*mockConn] {
	return func() (*mockConn, error) {
		id := counter.Add(1)
		return &mockConn{id: int(id), healthy: healthy}, nil
	}
}

func TestConnPoolGetPut(t *testing.T) {
	var created atomic.Int32
	factory := newConnFactory(&created, true)
	destroyer := func(c *mockConn) {}

	p := NewConnPool[*mockConn](factory, destroyer, ConnConfig{
		MaxOpen: 4,
		MaxIdle: 4,
	})
	defer p.Close()

	c1, err := p.Get()
	require.NoError(t, err)
	require.NotNil(t, c1)
	assert.Equal(t, int32(1), created.Load())

	p.Put(c1)

	c2, err := p.Get()
	require.NoError(t, err)
	require.Same(t, c1, c2)
	assert.Equal(t, int32(1), created.Load(), "expected reuse without new creation")

	p.Put(c2)

	stats := p.Stats()
	assert.Equal(t, int32(1), stats.OpenCount)
	assert.Equal(t, 1, stats.IdleCount)
	assert.Equal(t, int32(0), stats.InUse)
}

func TestConnPoolHealthCheckFailure(t *testing.T) {
	var created atomic.Int32
	var destroyed atomic.Int32
	factory := newConnFactory(&created, true)
	destroyer := func(c *mockConn) { destroyed.Add(1) }

	healthCheck := func(v any) error {
		c := v.(*mockConn)
		if !c.healthy {
			return errHealthCheck
		}
		return nil
	}

	p := NewConnPool[*mockConn](factory, destroyer, ConnConfig{
		MaxOpen:      4,
		MaxIdle:      4,
		HealthCheck:  healthCheck,
		HealthPeriod: 0,
	})
	defer p.Close()

	c1, err := p.Get()
	require.NoError(t, err)
	c1.healthy = false
	p.Put(c1)

	require.Equal(t, int32(1), created.Load())

	c2, err := p.Get()
	require.NoError(t, err)
	assert.NotSame(t, c1, c2, "unhealthy conn should be replaced")
	assert.Equal(t, int32(2), created.Load())
	assert.Equal(t, int32(1), destroyed.Load(), "unhealthy conn should be destroyed")
	p.Put(c2)
}

func TestConnPoolMaxLifetime(t *testing.T) {
	var created atomic.Int32
	var destroyed atomic.Int32
	factory := newConnFactory(&created, true)
	destroyer := func(c *mockConn) { destroyed.Add(1) }

	p := NewConnPool[*mockConn](factory, destroyer, ConnConfig{
		MaxOpen:     4,
		MaxIdle:     4,
		MaxLifetime: 30 * time.Millisecond,
	})
	defer p.Close()

	c1, err := p.Get()
	require.NoError(t, err)
	p.Put(c1)

	time.Sleep(60 * time.Millisecond)

	c2, err := p.Get()
	require.NoError(t, err)
	assert.NotSame(t, c1, c2, "expired conn should be replaced")
	assert.Equal(t, int32(2), created.Load())
	assert.Equal(t, int32(1), destroyed.Load(), "expired conn should be destroyed")
	p.Put(c2)
}

func TestConnPoolIdleEviction(t *testing.T) {
	var created atomic.Int32
	var destroyed atomic.Int32
	factory := newConnFactory(&created, true)
	destroyer := func(c *mockConn) { destroyed.Add(1) }

	p := NewConnPool[*mockConn](factory, destroyer, ConnConfig{
		MaxOpen:     4,
		MaxIdle:     4,
		MaxIdleTime: 40 * time.Millisecond,
	})
	defer p.Close()

	c1, err := p.Get()
	require.NoError(t, err)
	p.Put(c1)

	require.Equal(t, int32(1), p.Stats().OpenCount)

	time.Sleep(150 * time.Millisecond)

	assert.Equal(t, int32(0), p.Stats().OpenCount, "idle conn should be evicted")
	assert.Equal(t, int32(1), destroyed.Load(), "evicted conn should be destroyed")
}

func TestConnPoolBackgroundHealthCheck(t *testing.T) {
	var created atomic.Int32
	var destroyed atomic.Int32
	factory := newConnFactory(&created, true)
	destroyer := func(c *mockConn) { destroyed.Add(1) }

	healthCheck := func(v any) error {
		c := v.(*mockConn)
		if !c.healthy {
			return errHealthCheck
		}
		return nil
	}

	p := NewConnPool[*mockConn](factory, destroyer, ConnConfig{
		MaxOpen:      4,
		MaxIdle:      4,
		HealthCheck:  healthCheck,
		HealthPeriod: 30 * time.Millisecond,
	})
	defer p.Close()

	c1, err := p.Get()
	require.NoError(t, err)
	c1.healthy = false
	p.Put(c1)

	time.Sleep(150 * time.Millisecond)

	assert.Equal(t, int32(0), p.Stats().OpenCount, "unhealthy idle conn should be evicted by janitor")
	assert.Equal(t, int32(1), destroyed.Load())
}

func TestConnPoolClose(t *testing.T) {
	var created atomic.Int32
	var destroyed atomic.Int32
	factory := newConnFactory(&created, true)
	destroyer := func(c *mockConn) { destroyed.Add(1) }

	p := NewConnPool[*mockConn](factory, destroyer, ConnConfig{
		MaxOpen:     4,
		MaxIdle:     4,
		MaxIdleTime: 0,
	})

	c1, err := p.Get()
	require.NoError(t, err)
	c2, err := p.Get()
	require.NoError(t, err)
	p.Put(c1)
	p.Put(c2)

	p.Close()

	c, err := p.Get()
	require.ErrorIs(t, err, ErrPoolClosed)
	assert.Nil(t, c)

	assert.Equal(t, int32(2), destroyed.Load(), "idle conns should be destroyed on close")
}

func TestConnPoolConcurrentAccess(t *testing.T) {
	const maxOpen = 8
	const goroutines = 40
	const iterations = 80

	var created atomic.Int32
	factory := newConnFactory(&created, true)
	p := NewConnPool[*mockConn](factory, nil, ConnConfig{
		MaxOpen:     maxOpen,
		MaxIdle:     maxOpen,
		MaxIdleTime: 0,
	})
	defer p.Close()

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				c, err := p.Get()
				if err != nil {
					continue
				}
				p.Put(c)
			}
		}()
	}
	wg.Wait()

	stats := p.Stats()
	assert.LessOrEqual(t, stats.OpenCount, int32(maxOpen))
	assert.Equal(t, int32(stats.IdleCount)+stats.InUse, stats.OpenCount)
}

func TestConnPoolFactoryError(t *testing.T) {
	factory := func() (*mockConn, error) { return nil, errFactory }
	p := NewConnPool[*mockConn](factory, nil, ConnConfig{MaxOpen: 2, MaxIdle: 2})
	defer p.Close()

	c, err := p.Get()
	require.ErrorIs(t, err, errFactory)
	assert.Nil(t, c)
	assert.Equal(t, int32(0), p.Stats().OpenCount)
}

func TestConnPoolDoubleClose(t *testing.T) {
	factory := newConnFactory(&atomic.Int32{}, true)
	p := NewConnPool[*mockConn](factory, nil, ConnConfig{MaxOpen: 2, MaxIdle: 2})

	require.NotPanics(t, func() {
		p.Close()
		p.Close()
	})
}

func TestConnPoolStatsAccuracy(t *testing.T) {
	var created atomic.Int32
	factory := newConnFactory(&created, true)
	p := NewConnPool[*mockConn](factory, nil, ConnConfig{
		MaxOpen: 4,
		MaxIdle: 4,
	})
	defer p.Close()

	// Initial state.
	stats := p.Stats()
	assert.Equal(t, int32(0), stats.OpenCount)
	assert.Equal(t, 0, stats.IdleCount)
	assert.Equal(t, int32(0), stats.InUse)

	// Get two connections.
	c1, err := p.Get()
	require.NoError(t, err)
	c2, err := p.Get()
	require.NoError(t, err)

	stats = p.Stats()
	assert.Equal(t, int32(2), stats.OpenCount)
	assert.Equal(t, 0, stats.IdleCount)
	assert.Equal(t, int32(2), stats.InUse)

	// Return one.
	p.Put(c1)
	stats = p.Stats()
	assert.Equal(t, int32(2), stats.OpenCount)
	assert.Equal(t, 1, stats.IdleCount)
	assert.Equal(t, int32(1), stats.InUse)

	// Return the other.
	p.Put(c2)
	stats = p.Stats()
	assert.Equal(t, int32(2), stats.OpenCount)
	assert.Equal(t, 2, stats.IdleCount)
	assert.Equal(t, int32(0), stats.InUse)
}

func TestConnPoolCloseWithJanitorRunning(t *testing.T) {
	var created atomic.Int32
	var destroyed atomic.Int32
	factory := newConnFactory(&created, true)
	destroyer := func(c *mockConn) { destroyed.Add(1) }

	p := NewConnPool[*mockConn](factory, destroyer, ConnConfig{
		MaxOpen:     4,
		MaxIdle:     4,
		MaxIdleTime: 50 * time.Millisecond,
	})

	// Create some idle connections.
	c1, err := p.Get()
	require.NoError(t, err)
	c2, err := p.Get()
	require.NoError(t, err)
	p.Put(c1)
	p.Put(c2)

	// Close while janitor is running — should not block or panic.
	require.NotPanics(t, func() {
		p.Close()
	})

	// All idle connections should be destroyed.
	assert.Equal(t, int32(2), destroyed.Load(), "idle conns should be destroyed on close with janitor running")

	// Get after close should return error.
	c, err := p.Get()
	require.ErrorIs(t, err, ErrPoolClosed)
	assert.Nil(t, c)
}

func TestConnPoolPutOverflowDestroys(t *testing.T) {
	var created atomic.Int32
	var destroyed atomic.Int32
	factory := newConnFactory(&created, true)
	destroyer := func(c *mockConn) { destroyed.Add(1) }

	// MaxIdle=1, MaxOpen=4: idle channel capacity is 1.
	p := NewConnPool[*mockConn](factory, destroyer, ConnConfig{
		MaxOpen: 4,
		MaxIdle: 1,
	})
	defer p.Close()

	// Get two connections.
	c1, err := p.Get()
	require.NoError(t, err)
	c2, err := p.Get()
	require.NoError(t, err)

	// Put first — goes to idle (1 slot filled).
	p.Put(c1)
	require.Equal(t, int32(0), destroyed.Load())

	// Put second — idle is full, should overflow and destroy.
	p.Put(c2)
	assert.Equal(t, int32(1), destroyed.Load(), "overflow Put should destroy the connection")

	stats := p.Stats()
	assert.Equal(t, int32(1), stats.OpenCount, "openCount should reflect only the idle conn")
	assert.Equal(t, 1, stats.IdleCount)
	assert.Equal(t, int32(0), stats.InUse)
}

func TestConnPoolGetHealthCheckFailsFromIdle(t *testing.T) {
	var created atomic.Int32
	var destroyed atomic.Int32
	factory := newConnFactory(&created, true)
	destroyer := func(c *mockConn) { destroyed.Add(1) }

	healthCheck := func(v any) error {
		c := v.(*mockConn)
		if !c.healthy {
			return errHealthCheck
		}
		return nil
	}

	p := NewConnPool[*mockConn](factory, destroyer, ConnConfig{
		MaxOpen:     2,
		MaxIdle:     2,
		HealthCheck: healthCheck,
	})
	defer p.Close()

	// Get a connection, mark it unhealthy, return it.
	c1, err := p.Get()
	require.NoError(t, err)
	c1.healthy = false
	p.Put(c1)

	require.Equal(t, int32(1), created.Load())
	require.Equal(t, int32(0), destroyed.Load())

	// Get — should detect unhealthy idle conn, destroy it, and create a new one.
	c2, err := p.Get()
	require.NoError(t, err)
	assert.NotSame(t, c1, c2, "unhealthy idle conn should be replaced")
	assert.Equal(t, int32(2), created.Load())
	assert.Equal(t, int32(1), destroyed.Load(), "unhealthy conn should be destroyed on Get")
	p.Put(c2)
}

func TestConnPoolMaxIdleTimeEvictionWithMultipleConns(t *testing.T) {
	var created atomic.Int32
	var destroyed atomic.Int32
	factory := newConnFactory(&created, true)
	destroyer := func(c *mockConn) { destroyed.Add(1) }

	p := NewConnPool[*mockConn](factory, destroyer, ConnConfig{
		MaxOpen:     4,
		MaxIdle:     4,
		MaxIdleTime: 40 * time.Millisecond,
	})
	defer p.Close()

	// Create and return 3 idle connections.
	c1, err := p.Get()
	require.NoError(t, err)
	c2, err := p.Get()
	require.NoError(t, err)
	c3, err := p.Get()
	require.NoError(t, err)
	p.Put(c1)
	p.Put(c2)
	p.Put(c3)

	require.Equal(t, int32(3), p.Stats().OpenCount)
	require.Equal(t, int32(0), destroyed.Load())

	// Wait for janitor to evict idle connections.
	time.Sleep(150 * time.Millisecond)

	assert.Equal(t, int32(0), p.Stats().OpenCount, "all idle conns should be evicted")
	assert.Equal(t, int32(3), destroyed.Load(), "all evicted conns should be destroyed")
}

func TestConnPoolPutAfterCloseDestroys(t *testing.T) {
	var created atomic.Int32
	var destroyed atomic.Int32
	factory := newConnFactory(&created, true)
	destroyer := func(c *mockConn) { destroyed.Add(1) }

	p := NewConnPool[*mockConn](factory, destroyer, ConnConfig{
		MaxOpen: 2,
		MaxIdle: 2,
	})

	c1, err := p.Get()
	require.NoError(t, err)

	p.Close()
	p.Put(c1)

	assert.Equal(t, int32(1), destroyed.Load(), "Put after close should destroy the connection")
}

func TestConnPoolGetBlocksAtCapacity(t *testing.T) {
	var created atomic.Int32
	factory := newConnFactory(&created, true)
	p := NewConnPool[*mockConn](factory, nil, ConnConfig{
		MaxOpen: 1,
		MaxIdle: 1,
	})
	defer p.Close()

	c1, err := p.Get()
	require.NoError(t, err)

	done := make(chan *mockConn, 1)
	go func() {
		c, err := p.Get()
		if err != nil {
			done <- nil
			return
		}
		done <- c
	}()

	// Should block at capacity.
	select {
	case <-done:
		t.Fatal("Get should block when at capacity")
	case <-time.After(50 * time.Millisecond):
	}

	// Return the connection — blocked Get should unblock.
	p.Put(c1)

	select {
	case c2 := <-done:
		require.NotNil(t, c2)
		assert.Same(t, c1, c2, "blocked Get should receive the returned connection")
	case <-time.After(time.Second):
		t.Fatal("Get did not unblock after Put")
	}
}
