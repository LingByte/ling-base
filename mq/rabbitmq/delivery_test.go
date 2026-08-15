// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package rabbitmq

import (
	"testing"

	"github.com/LingByte/ling-base/mq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func TestAmqpDelivery_Message(t *testing.T) {
	d := amqp.Delivery{
		MessageId:     "msg-123",
		Exchange:      "test.ex",
		RoutingKey:    "test.key",
		Body:          []byte("hello"),
		ContentType:   "application/json",
		Priority:      5,
		CorrelationId: "corr-1",
		ReplyTo:       "reply.queue",
		Redelivered:   true,
		DeliveryTag:   42,
		Headers:       amqp.Table{"x-custom": "val"},
	}
	a := &amqpDelivery{d: d}

	msg := a.Message()
	assert.Equal(t, "msg-123", msg.ID)
	assert.Equal(t, "test.ex", msg.Exchange)
	assert.Equal(t, "test.key", msg.RoutingKey)
	assert.Equal(t, []byte("hello"), msg.Body)
	assert.Equal(t, "application/json", msg.ContentType)
	assert.Equal(t, uint8(5), msg.Priority)
	assert.Equal(t, "corr-1", msg.CorrelationID)
	assert.Equal(t, "reply.queue", msg.ReplyTo)
}

func TestAmqpDelivery_Accessors(t *testing.T) {
	d := amqp.Delivery{
		Body:        []byte("payload"),
		Headers:     amqp.Table{"k": "v"},
		RoutingKey:  "rk",
		Exchange:    "ex",
		Redelivered: true,
		DeliveryTag: 99,
	}
	a := &amqpDelivery{d: d}

	assert.Equal(t, []byte("payload"), a.Body())
	assert.Equal(t, "rk", a.RoutingKey())
	assert.Equal(t, "ex", a.Exchange())
	assert.True(t, a.Redelivered())
	assert.Equal(t, uint64(99), a.DeliveryTag())
	assert.Equal(t, "v", a.Headers()["k"])
}

func TestAmqpDelivery_ManualAckFlag(t *testing.T) {
	a := &amqpDelivery{d: amqp.Delivery{}}
	assert.False(t, a.ackedManually)
	// Ack/Nack/Reject set the flag even though the underlying channel
	// is nil (will error, but flag is set first).
	_ = a.Ack()
	assert.True(t, a.ackedManually)
}

func TestAmqpDelivery_ManualNackFlag(t *testing.T) {
	a := &amqpDelivery{d: amqp.Delivery{}}
	_ = a.Nack(true)
	assert.True(t, a.ackedManually)
}

func TestAmqpDelivery_ManualRejectFlag(t *testing.T) {
	a := &amqpDelivery{d: amqp.Delivery{}}
	_ = a.Reject(false)
	assert.True(t, a.ackedManually)
}

// Verify that amqpDelivery satisfies mq.Delivery.
func TestAmqpDelivery_ImplementsDelivery(t *testing.T) {
	var _ mq.Delivery = (*amqpDelivery)(nil)
}

// Verify that Broker implements mq.Broker.
func TestBroker_ImplementsBroker(t *testing.T) {
	var _ mq.Broker = (*Broker)(nil)
}

// Verify that Producer implements mq.Producer.
func TestProducer_ImplementsProducer(t *testing.T) {
	var _ mq.Producer = (*Producer)(nil)
}

// Verify that Consumer implements mq.Consumer.
func TestConsumer_ImplementsConsumer(t *testing.T) {
	var _ mq.Consumer = (*Consumer)(nil)
}
