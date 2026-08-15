// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package activemq

import (
	"testing"

	"github.com/LingByte/ling-base/mq"
	"github.com/go-stomp/stomp/v3"
	"github.com/go-stomp/stomp/v3/frame"
	"github.com/stretchr/testify/assert"
)

func newTestMessage(headers ...string) *stomp.Message {
	h := frame.NewHeader(headers...)
	return &stomp.Message{
		Destination:  "/queue/test",
		ContentType:  "text/plain",
		Header:       h,
		Body:         []byte("hello"),
		Subscription: nil, // no subscription => ShouldAck() == false
	}
}

func TestStompDelivery_Message(t *testing.T) {
	msg := newTestMessage(
		"amq-message-id", "msg-123",
		"correlation-id", "corr-1",
		"reply-to", "reply.queue",
		"type", "user.created",
		"app-id", "app-1",
		"content-encoding", "gzip",
		"priority", "5",
		"persistent", "true",
		"redelivered", "true",
		frame.MessageId, "stomp-id-42",
	)
	d := &stompDelivery{msg: msg, destination: "/queue/test"}

	m := d.Message()
	assert.Equal(t, "msg-123", m.ID)
	assert.Equal(t, "/queue/test", m.Exchange)
	assert.Equal(t, "/queue/test", m.RoutingKey)
	assert.Equal(t, []byte("hello"), m.Body)
	assert.Equal(t, "text/plain", m.ContentType)
	assert.Equal(t, "corr-1", m.CorrelationID)
	assert.Equal(t, "reply.queue", m.ReplyTo)
	assert.Equal(t, "user.created", m.Type)
	assert.Equal(t, "app-1", m.AppID)
	assert.Equal(t, "gzip", m.ContentEncoding)
	assert.Equal(t, uint8(5), m.Priority)
	assert.Equal(t, mq.Persistent, m.DeliveryMode)
}

func TestStompDelivery_Accessors(t *testing.T) {
	msg := newTestMessage("k", "v", frame.MessageId, "abc")
	d := &stompDelivery{msg: msg, destination: "/queue/events"}

	assert.Equal(t, []byte("hello"), d.Body())
	assert.Equal(t, "/queue/events", d.RoutingKey())
	assert.Equal(t, "/queue/events", d.Exchange())
	assert.Equal(t, "v", d.Headers()["k"])
}

func TestStompDelivery_Redelivered(t *testing.T) {
	msg := newTestMessage("redelivered", "true")
	d := &stompDelivery{msg: msg, destination: "/queue/test"}
	assert.True(t, d.Redelivered())

	msg2 := newTestMessage("redelivered", "false")
	d2 := &stompDelivery{msg: msg2, destination: "/queue/test"}
	assert.False(t, d2.Redelivered())

	msg3 := newTestMessage()
	d3 := &stompDelivery{msg: msg3, destination: "/queue/test"}
	assert.False(t, d3.Redelivered())
}

func TestStompDelivery_DeliveryTag(t *testing.T) {
	msg := newTestMessage(frame.MessageId, "id-1")
	d := &stompDelivery{msg: msg, destination: "/queue/test"}
	tag := d.DeliveryTag()
	assert.NotZero(t, tag)

	// Same message-id => same tag.
	d2 := &stompDelivery{msg: newTestMessage(frame.MessageId, "id-1"), destination: "/queue/test"}
	assert.Equal(t, tag, d2.DeliveryTag())

	// No message-id => zero tag.
	d3 := &stompDelivery{msg: newTestMessage(), destination: "/queue/test"}
	assert.Equal(t, uint64(0), d3.DeliveryTag())
}

func TestStompDelivery_AutoAck_NoOp(t *testing.T) {
	// Without a subscription/conn, ShouldAck() is false, so Ack/Nack/Reject
	// are no-ops and return nil.
	msg := newTestMessage()
	d := &stompDelivery{msg: msg, destination: "/queue/test"}

	assert.NoError(t, d.Ack())
	assert.True(t, d.ackedManually)

	d2 := &stompDelivery{msg: newTestMessage(), destination: "/queue/test"}
	assert.NoError(t, d2.Nack(true))
	assert.True(t, d2.ackedManually)

	d3 := &stompDelivery{msg: newTestMessage(), destination: "/queue/test"}
	assert.NoError(t, d3.Reject(false))
	assert.True(t, d3.ackedManually)
}

func TestStompDelivery_NilHeader(t *testing.T) {
	d := &stompDelivery{
		msg:         &stomp.Message{Header: nil},
		destination: "/queue/test",
	}
	assert.False(t, d.Redelivered())
	assert.Equal(t, uint64(0), d.DeliveryTag())
	assert.Nil(t, d.Headers())
}

func TestStompHeadersToMap(t *testing.T) {
	h := frame.NewHeader("a", "1", "b", "2", "a", "3")
	m := stompHeadersToMap(h)
	assert.Equal(t, "1", m["a"]) // first wins
	assert.Equal(t, "2", m["b"])

	assert.Nil(t, stompHeadersToMap(nil))
}

func TestStompDelivery_ImplementsDelivery(t *testing.T) {
	var _ mq.Delivery = (*stompDelivery)(nil)
}
