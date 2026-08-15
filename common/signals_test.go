// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package common

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignal(t *testing.T) {
	var val string
	var eid uint
	eid = Sig().Connect("mock_test", func(sender any, params ...any) {
		val = sender.(string)
		// Handlers unsubscribe themselves mid-Emit; must not deadlock.
		Sig().Disconnect("mock_test", eid)
	})
	Sig().Emit("mock_test", "unittest")
	assert.Equal(t, val, "unittest")
	Sig().Clear("mock_test")
	assert.Equal(t, 0, len(Sig().sigHandlers))
}

// TestSignalsReentrantConnectDuringEmit pins the copy-on-write contract: a
// handler that subscribes during Emit must not disturb the in-flight iteration
// and must not be invoked by it.
func TestSignalsReentrantConnectDuringEmit(t *testing.T) {
	s := NewSignals()
	var order []string

	s.Connect("ev", func(sender any, params ...any) {
		order = append(order, "first")
		s.Connect("ev", func(sender any, params ...any) {
			order = append(order, "late")
		})
	})
	s.Connect("ev", func(sender any, params ...any) {
		order = append(order, "second")
	})

	s.Emit("ev", nil)
	assert.Equal(t, []string{"first", "second"}, order)

	// Second Emit sees the handler subscribed during the first one. It also
	// subscribes another, which again only becomes visible to the next Emit.
	order = nil
	s.Emit("ev", nil)
	assert.Equal(t, []string{"first", "second", "late"}, order)
}

// TestSignalsConcurrentEmitConnectDisconnect is the regression test for the
// unsynchronised bus: before the mutex/copy-on-write fix this reproduced
// "fatal error: concurrent map read and map write" under -race.
func TestSignalsConcurrentEmitConnectDisconnect(t *testing.T) {
	s := NewSignals()
	var hits int64
	var hitsMu sync.Mutex

	s.Connect("hot", func(sender any, params ...any) {
		hitsMu.Lock()
		hits++
		hitsMu.Unlock()
	})

	const workers = 16
	const iterations = 200

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				switch w % 4 {
				case 0:
					s.Emit("hot", "sender")
				case 1:
					id := s.Connect("hot", func(sender any, params ...any) {})
					s.Disconnect("hot", id)
				case 2:
					s.Query("hot", "param")
				case 3:
					s.Emit("cold", nil)
				}
			}
		}(w)
	}
	wg.Wait()

	hitsMu.Lock()
	defer hitsMu.Unlock()
	assert.Greater(t, hits, int64(0))
}

func TestSignalsQuery(t *testing.T) {
	s := NewSignals()
	s.Connect("query_test", func(sender any, params ...any) {
		if len(params) < 1 {
			return
		}
		r, ok := params[len(params)-1].(*QueryResult)
		if !ok {
			return
		}
		r.Value = "result_value"
		r.Err = nil
	})

	r := s.Query("query_test", "input")
	assert.Equal(t, "result_value", r.Value)
	assert.NoError(t, r.Err)
}

func TestSignalsDisconnectNonexistent(t *testing.T) {
	s := NewSignals()
	s.Connect("ev", func(sender any, params ...any) {})
	// Disconnecting a non-existent ID should be a no-op.
	s.Disconnect("ev", 999)
	assert.Equal(t, 1, len(s.sigHandlers["ev"]))
}

func TestSignalsClearMultiple(t *testing.T) {
	s := NewSignals()
	s.Connect("ev1", func(sender any, params ...any) {})
	s.Connect("ev2", func(sender any, params ...any) {})
	s.Connect("ev3", func(sender any, params ...any) {})

	s.Clear("ev1", "ev3")
	assert.Equal(t, 0, len(s.sigHandlers["ev1"]))
	assert.Equal(t, 0, len(s.sigHandlers["ev3"]))
	assert.Equal(t, 1, len(s.sigHandlers["ev2"]))
}

func TestSignalsEmitNoHandlers(t *testing.T) {
	s := NewSignals()
	// Emitting an event with no handlers should be safe.
	s.Emit("nonexistent", "sender")
}

func TestSignalsQueryNoHandlers(t *testing.T) {
	s := NewSignals()
	r := s.Query("nonexistent", "param")
	assert.NotNil(t, r)
	assert.Nil(t, r.Value)
	assert.Nil(t, r.Err)
}

func TestSignalsMultipleHandlersOrder(t *testing.T) {
	s := NewSignals()
	var order []string

	s.Connect("ev", func(sender any, params ...any) {
		order = append(order, "a")
	})
	s.Connect("ev", func(sender any, params ...any) {
		order = append(order, "b")
	})
	s.Connect("ev", func(sender any, params ...any) {
		order = append(order, "c")
	})

	s.Emit("ev", nil)
	assert.Equal(t, []string{"a", "b", "c"}, order)
}

func TestSignalsParams(t *testing.T) {
	s := NewSignals()
	var received []any

	s.Connect("ev", func(sender any, params ...any) {
		received = params
	})

	s.Emit("ev", "sender", "p1", "p2", "p3")
	assert.Equal(t, []any{"p1", "p2", "p3"}, received)
}

func TestSignalsSingleton(t *testing.T) {
	a := Sig()
	b := Sig()
	assert.Same(t, a, b, "Sig() should return the same singleton instance")
}
