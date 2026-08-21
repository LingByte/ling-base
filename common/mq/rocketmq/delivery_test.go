// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package rocketmq

import (
	"strconv"
	"testing"
	"time"

	"github.com/LingByte/ling-base/mq"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/stretchr/testify/assert"
)

func newTestMessageExt() *primitive.MessageExt {
	ext := &primitive.MessageExt{}
	ext.Topic = "test.topic"
	ext.Body = []byte("hello")
	ext.WithTag("test.tag")
	ext.WithProperty("ContentType", "application/json")
	ext.WithProperty("ContentEncoding", "gzip")
	ext.WithProperty(primitive.PropertyCorrelationID, "corr-1")
	ext.WithProperty("ReplyTo", "reply.queue")
	ext.WithProperty("Type", "user.created")
	ext.WithProperty("UserID", "u1")
	ext.WithProperty("AppID", "app1")
	ext.WithProperty("MessageID", "msg-123")
	ext.WithProperty("Timestamp", strconv.FormatInt(time.Unix(1700000000, 0).UnixNano(), 10))
	ext.WithProperty("Expiration", "5000")
	ext.WithProperty("Priority", "3")
	ext.WithProperty("x-custom", "val")

	ext.MsgId = "msg-123"
	ext.QueueOffset = 42
	ext.ReconsumeTimes = 1
	ext.BornTimestamp = time.UnixMilli(1700000000000).UnixMilli()
	return ext
}

func TestDelivery_Message(t *testing.T) {
	d := newDelivery(newTestMessageExt())
	msg := d.Message()

	assert.Equal(t, "msg-123", msg.ID)
	assert.Equal(t, "test.topic", msg.Exchange)
	assert.Equal(t, "test.tag", msg.RoutingKey)
	assert.Equal(t, []byte("hello"), msg.Body)
	assert.Equal(t, "application/json", msg.ContentType)
	assert.Equal(t, "gzip", msg.ContentEncoding)
	assert.Equal(t, "corr-1", msg.CorrelationID)
	assert.Equal(t, "reply.queue", msg.ReplyTo)
	assert.Equal(t, "user.created", msg.Type)
	assert.Equal(t, "u1", msg.UserID)
	assert.Equal(t, "app1", msg.AppID)
	assert.Equal(t, uint8(3), msg.Priority)
	assert.Equal(t, 5*time.Second, msg.Expiration)
	assert.False(t, msg.Timestamp.IsZero())
	assert.Equal(t, mq.Persistent, msg.DeliveryMode)
	assert.Equal(t, "val", msg.Headers["x-custom"])
}

func TestDelivery_Accessors(t *testing.T) {
	d := newDelivery(newTestMessageExt())

	assert.Equal(t, []byte("hello"), d.Body())
	assert.Equal(t, "test.tag", d.RoutingKey())
	assert.Equal(t, "test.topic", d.Exchange())
	assert.True(t, d.Redelivered())
	assert.Equal(t, uint64(42), d.DeliveryTag())
	assert.Equal(t, "val", d.Headers()["x-custom"])
	assert.Equal(t, "application/json", d.Headers()["ContentType"])
}

func TestDelivery_NotRedelivered(t *testing.T) {
	ext := &primitive.MessageExt{}
	ext.Topic = "t"
	ext.Body = []byte("b")
	d := newDelivery(ext)
	assert.False(t, d.Redelivered())
}

func TestDelivery_Ack(t *testing.T) {
	d := newDelivery(newTestMessageExt())
	assert.False(t, d.ackedManually)
	assert.NoError(t, d.Ack())
	assert.True(t, d.ackedManually)
	assert.False(t, d.nacked)
}

func TestDelivery_Nack(t *testing.T) {
	d := newDelivery(newTestMessageExt())
	assert.NoError(t, d.Nack(true))
	assert.True(t, d.ackedManually)
	assert.True(t, d.nacked)
	assert.True(t, d.requeue)
}

func TestDelivery_Nack_NoRequeue(t *testing.T) {
	d := newDelivery(newTestMessageExt())
	assert.NoError(t, d.Nack(false))
	assert.True(t, d.nacked)
	assert.False(t, d.requeue)
}

func TestDelivery_Reject(t *testing.T) {
	d := newDelivery(newTestMessageExt())
	assert.NoError(t, d.Reject(true))
	assert.True(t, d.ackedManually)
	assert.True(t, d.rejected)
	assert.True(t, d.requeue)
}

func TestDelivery_Reject_NoRequeue(t *testing.T) {
	d := newDelivery(newTestMessageExt())
	assert.NoError(t, d.Reject(false))
	assert.True(t, d.rejected)
	assert.False(t, d.requeue)
}

func TestDelivery_ImplementsDelivery(t *testing.T) {
	var _ mq.Delivery = (*Delivery)(nil)
}

func TestDelivery_TimestampFromBornTimestamp(t *testing.T) {
	ext := &primitive.MessageExt{}
	ext.Topic = "t"
	ext.Body = []byte("b")
	ext.BornTimestamp = time.UnixMilli(1700000000000).UnixMilli()
	d := newDelivery(ext)
	msg := d.Message()
	assert.False(t, msg.Timestamp.IsZero())
}

func TestDelivery_EmptyProperties(t *testing.T) {
	ext := &primitive.MessageExt{}
	ext.Topic = "t"
	ext.Body = []byte("b")
	d := newDelivery(ext)
	msg := d.Message()
	assert.NotNil(t, msg.Headers)
	assert.Empty(t, msg.ContentType)
	assert.Equal(t, []byte("b"), d.Body())
}
