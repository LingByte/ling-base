// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package mongodb

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/LingByte/ling-base/common/limiter"
)

// mockCollection is an in-memory MongoDB collection for testing.
type mockCollection struct {
	mu   sync.Mutex
	docs []map[string]interface{}
}

func newMockCollection() *mockCollection {
	return &mockCollection{}
}

func (m *mockCollection) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := extractKey(filter)
	if key == "" {
		return int64(len(m.docs)), nil
	}
	count := int64(0)
	for _, doc := range m.docs {
		if k, ok := doc["limiterKey"].(string); ok && k == key {
			count++
		}
	}
	return count, nil
}

func (m *mockCollection) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Handle bson.M (which is primitive.M, a named map[string]interface{}).
	switch doc := document.(type) {
	case map[string]interface{}:
		m.docs = append(m.docs, doc)
		return &mongo.InsertOneResult{}, nil
	default:
		// Try reflection-based conversion for bson.M etc.
		v := reflect.ValueOf(document)
		if v.Kind() == reflect.Map {
			converted := make(map[string]interface{})
			for _, key := range v.MapKeys() {
				converted[fmt.Sprintf("%v", key.Interface())] = v.MapIndex(key).Interface()
			}
			m.docs = append(m.docs, converted)
			return &mongo.InsertOneResult{}, nil
		}
		return nil, assert.AnError
	}
}

func (m *mockCollection) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := extractKey(filter)
	for i, doc := range m.docs {
		if k, ok := doc["limiterKey"].(string); ok && k == key {
			m.docs = append(m.docs[:i], m.docs[i+1:]...)
			return &mongo.DeleteResult{DeletedCount: 1}, nil
		}
	}
	return &mongo.DeleteResult{DeletedCount: 0}, nil
}

func (m *mockCollection) DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := extractKey(filter)
	deleted := int64(0)
	var remaining []map[string]interface{}
	for _, doc := range m.docs {
		if k, ok := doc["limiterKey"].(string); ok && k == key {
			deleted++
		} else {
			remaining = append(remaining, doc)
		}
	}
	m.docs = remaining
	return &mongo.DeleteResult{DeletedCount: deleted}, nil
}

// extractKey gets the "limiterKey" value from a filter (handles bson.M etc).
func extractKey(filter interface{}) string {
	v := reflect.ValueOf(filter)
	if v.Kind() != reflect.Map {
		return ""
	}
	keyVal := v.MapIndex(reflect.ValueOf("limiterKey"))
	if !keyVal.IsValid() {
		return ""
	}
	return fmt.Sprintf("%v", keyVal.Interface())
}

// ===== RateLimit =====

func TestRateLimit_Basic(t *testing.T) {
	col := newMockCollection()
	l := NewRateLimit(col, "test:rl", 3, time.Second)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		assert.NoError(t, l.Acquire(ctx, nil))
	}
	err := l.Acquire(ctx, nil)
	assert.Equal(t, limiter.ErrLimitExceeded, err)
}

func TestRateLimit_Running(t *testing.T) {
	col := newMockCollection()
	l := NewRateLimit(col, "test:rl:running", 10, time.Second)
	ctx := context.Background()

	l.Acquire(ctx, nil)
	l.Acquire(ctx, nil)
	assert.Equal(t, 2, l.Running())
}

func TestRateLimit_ReleaseNoOp(t *testing.T) {
	col := newMockCollection()
	l := NewRateLimit(col, "test:rl:release", 10, time.Second)
	assert.NotPanics(t, func() {
		l.Release(nil)
	})
}

// ===== Concurrency =====

func TestConcurrency_Basic(t *testing.T) {
	col := newMockCollection()
	l := NewConcurrency(col, "test:conc", 3)
	ctx := context.Background()

	assert.NoError(t, l.Acquire(ctx, nil))
	assert.NoError(t, l.Acquire(ctx, nil))
	assert.NoError(t, l.Acquire(ctx, nil))
	err := l.Acquire(ctx, nil)
	assert.Equal(t, limiter.ErrLimitExceeded, err)
}

func TestConcurrency_Release(t *testing.T) {
	col := newMockCollection()
	l := NewConcurrency(col, "test:conc:release", 2)
	ctx := context.Background()

	assert.NoError(t, l.Acquire(ctx, nil))
	assert.NoError(t, l.Acquire(ctx, nil))
	assert.Equal(t, limiter.ErrLimitExceeded, l.Acquire(ctx, nil))

	l.Release(nil)
	assert.NoError(t, l.Acquire(ctx, nil))
}

func TestConcurrency_Running(t *testing.T) {
	col := newMockCollection()
	l := NewConcurrency(col, "test:conc:running", 10)
	ctx := context.Background()

	l.Acquire(ctx, nil)
	l.Acquire(ctx, nil)
	assert.Equal(t, 2, l.Running())
}

func TestConcurrency_ReleaseUnknownKey(t *testing.T) {
	col := newMockCollection()
	l := NewConcurrency(col, "test:conc:unknown", 10)
	assert.NotPanics(t, func() {
		l.Release(nil)
	})
}

// ===== Reset =====

func TestReset(t *testing.T) {
	col := newMockCollection()
	l := NewConcurrency(col, "test:reset", 5)
	ctx := context.Background()
	l.Acquire(ctx, nil)
	l.Acquire(ctx, nil)
	assert.Equal(t, 2, l.Running())

	assert.NoError(t, Reset(ctx, col, "test:reset"))
	assert.Equal(t, 0, l.Running())
}

func TestNewRateLimit_NotNil(t *testing.T) {
	col := newMockCollection()
	l := NewRateLimit(col, "test:created", 10, time.Second)
	require.NotNil(t, l)
}

func TestNewConcurrency_NotNil(t *testing.T) {
	col := newMockCollection()
	l := NewConcurrency(col, "test:created", 10)
	require.NotNil(t, l)
}
