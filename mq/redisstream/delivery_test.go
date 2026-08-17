// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package redisstream

import (
	"testing"
	"time"

	"github.com/LingByte/ling-base/mq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestStreamDelivery_Message(t *testing.T) {
	ts := time.Unix(0, 1700000000000000000)
	d := &streamDelivery{
		msg: redis.XMessage{
			ID: "1700000000000-0",
			Values: map[string]any{
				"body":             "hello",
				"id":               "msg-123",
				"content_type":     "application/json",
				"content_encoding": "gzip",
				"correlation_id":   "corr-1",
				"reply_to":         "reply.queue",
				"type":             "user.created",
				"user_id":          "u1",
				"app_id":           "app1",
				"routing_key":      "rk",
				"priority":         int64(5),
				"delivery_mode":    int64(2),
				"timestamp":        int64(ts.UnixNano()),
				"h_x-custom":       "val",
			},
		},
		stream: "test-stream",
	}

	msg := d.Message()
	assert.Equal(t, "1700000000000-0", msg.ID)
	assert.Equal(t, "test-stream", msg.Exchange)
	assert.Equal(t, "rk", msg.RoutingKey)
	assert.Equal(t, []byte("hello"), msg.Body)
	assert.Equal(t, "application/json", msg.ContentType)
	assert.Equal(t, "gzip", msg.ContentEncoding)
	assert.Equal(t, "corr-1", msg.CorrelationID)
	assert.Equal(t, "reply.queue", msg.ReplyTo)
	assert.Equal(t, "user.created", msg.Type)
	assert.Equal(t, "u1", msg.UserID)
	assert.Equal(t, "app1", msg.AppID)
	assert.Equal(t, "rk", msg.RoutingKey)
	assert.Equal(t, uint8(5), msg.Priority)
	assert.Equal(t, mq.Persistent, msg.DeliveryMode)
	assert.Equal(t, ts.UnixNano(), msg.Timestamp.UnixNano())
	assert.Equal(t, "val", msg.Headers["x-custom"])
}

func TestStreamDelivery_Body(t *testing.T) {
	d := &streamDelivery{
		msg: redis.XMessage{
			Values: map[string]any{
				"body": "payload",
			},
		},
	}
	assert.Equal(t, []byte("payload"), d.Body())
}

func TestStreamDelivery_Body_Empty(t *testing.T) {
	d := &streamDelivery{
		msg: redis.XMessage{Values: map[string]any{}},
	}
	assert.Nil(t, d.Body())
}

func TestStreamDelivery_Headers(t *testing.T) {
	d := &streamDelivery{
		msg: redis.XMessage{
			Values: map[string]any{
				"body":       "x",
				"h_key1":     "val1",
				"h_key2":     int64(42),
				"not_header": "ignored",
			},
		},
	}
	h := d.Headers()
	assert.Equal(t, "val1", h["key1"])
	assert.Equal(t, int64(42), h["key2"])
	_, exists := h["not_header"]
	assert.False(t, exists)
}

func TestStreamDelivery_RoutingKey(t *testing.T) {
	d := &streamDelivery{stream: "my-stream"}
	assert.Equal(t, "my-stream", d.RoutingKey())
}

func TestStreamDelivery_Exchange(t *testing.T) {
	d := &streamDelivery{stream: "my-stream"}
	assert.Equal(t, "my-stream", d.Exchange())
}

func TestStreamDelivery_Redelivered(t *testing.T) {
	d := &streamDelivery{redelivered: true}
	assert.True(t, d.Redelivered())

	d2 := &streamDelivery{redelivered: false}
	assert.False(t, d2.Redelivered())
}

func TestStreamDelivery_DeliveryTag(t *testing.T) {
	d := &streamDelivery{
		msg: redis.XMessage{ID: "1700000000000-5"},
	}
	tag := d.DeliveryTag()
	assert.Equal(t, uint64(1700000000000), tag)
}

func TestStreamDelivery_DeliveryTag_EmptyID(t *testing.T) {
	d := &streamDelivery{
		msg: redis.XMessage{ID: ""},
	}
	assert.Equal(t, uint64(0), d.DeliveryTag())
}

func TestStreamDelivery_DeliveryTag_NoSequence(t *testing.T) {
	d := &streamDelivery{
		msg: redis.XMessage{ID: "1700000000000"},
	}
	tag := d.DeliveryTag()
	assert.Equal(t, uint64(1700000000000), tag)
}

func TestStreamDelivery_Ack_Standalone(t *testing.T) {
	// In standalone mode (no group), Ack is a no-op and should not error.
	d := &streamDelivery{
		msg:    redis.XMessage{ID: "1-0", Values: map[string]any{"body": "x"}},
		stream: "s",
		group:  "",
	}
	err := d.Ack()
	assert.NoError(t, err)
	assert.True(t, d.ackedManually)
}

func TestStreamDelivery_Nack_Standalone(t *testing.T) {
	d := &streamDelivery{
		msg:    redis.XMessage{ID: "1-0", Values: map[string]any{"body": "x"}},
		stream: "s",
		group:  "",
	}
	err := d.Nack(true)
	assert.NoError(t, err)
	assert.True(t, d.ackedManually)
}

func TestStreamDelivery_Reject_Standalone(t *testing.T) {
	d := &streamDelivery{
		msg:    redis.XMessage{ID: "1-0", Values: map[string]any{"body": "x"}},
		stream: "s",
		group:  "",
	}
	err := d.Reject(false)
	assert.NoError(t, err)
	assert.True(t, d.ackedManually)
}

func TestStreamDelivery_ImplementsDelivery(t *testing.T) {
	var _ mq.Delivery = (*streamDelivery)(nil)
}
