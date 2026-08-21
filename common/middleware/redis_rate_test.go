package middleware

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRedisRateEnabled_DefaultFalse(t *testing.T) {
	// Without init, should be false
	redisRateMu.Lock()
	redisRateClient = nil
	redisRateMu.Unlock()

	assert.False(t, RedisRateEnabled())
}

func TestRedisAllow_NoClient(t *testing.T) {
	redisRateMu.Lock()
	old := redisRateClient
	redisRateClient = nil
	redisRateMu.Unlock()
	defer func() {
		redisRateMu.Lock()
		redisRateClient = old
		redisRateMu.Unlock()
	}()

	// Without redis client, should fail open (return true)
	assert.True(t, RedisAllow("test-key", 10, time.Minute))
}

func TestRedisAllow_LimitZero(t *testing.T) {
	assert.True(t, RedisAllow("test-key", 0, time.Minute))
}

func TestRedisAllow_NegativeLimit(t *testing.T) {
	assert.True(t, RedisAllow("test-key", -1, time.Minute))
}

func TestInitRedisRateBackend_InvalidAddr(t *testing.T) {
	err := InitRedisRateBackend("localhost:1", "")
	assert.Error(t, err)

	// Should not have set the client
	assert.False(t, RedisRateEnabled())
}
