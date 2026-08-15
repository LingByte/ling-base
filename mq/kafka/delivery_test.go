// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package kafka

import (
	"testing"

	"github.com/LingByte/ling-base/mq"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
)

// Verify that kafkaDelivery satisfies mq.Delivery.
func TestKafkaDelivery_ImplementsDelivery(t *testing.T) {
	var _ mq.Delivery = (*kafkaDelivery)(nil)
}

func TestKafkaDelivery_Message(t *testing.T) {
	km := kafkago.Message{
		Topic:     "test-topic",
		Partition: 2,
		Offset:    42,
		Key:       []byte("msg-key-123"),
		Value:     []byte(`{"hello":"world"}`),
		Headers: []kafkago.Header{
			{Key: "content-type", Value: []byte("application/json")},
			{Key: "x-custom", Value: []byte("val")},
		},
	}
	d := &kafkaDelivery{msg: km}

	msg := d.Message()
	assert.Equal(t, "msg-key-123", msg.ID)
	assert.Equal(t, "test-topic", msg.Exchange)
	assert.Equal(t, "2", msg.RoutingKey)
	assert.Equal(t, []byte(`{"hello":"world"}`), msg.Body)
	assert.Equal(t, mq.Persistent, msg.DeliveryMode)
	assert.Equal(t, "application/json", msg.Headers["content-type"])
	assert.Equal(t, "val", msg.Headers["x-custom"])
}

func TestKafkaDelivery_Body(t *testing.T) {
	d := &kafkaDelivery{msg: kafkago.Message{Value: []byte("payload")}}
	assert.Equal(t, []byte("payload"), d.Body())
}

func TestKafkaDelivery_Headers(t *testing.T) {
	km := kafkago.Message{
		Headers: []kafkago.Header{
			{Key: "k1", Value: []byte("v1")},
			{Key: "k2", Value: []byte("v2")},
		},
	}
	d := &kafkaDelivery{msg: km}
	h := d.Headers()
	assert.Equal(t, "v1", h["k1"])
	assert.Equal(t, "v2", h["k2"])
	assert.Len(t, h, 2)
}

func TestKafkaDelivery_RoutingKey(t *testing.T) {
	d := &kafkaDelivery{msg: kafkago.Message{Partition: 5}}
	assert.Equal(t, "5", d.RoutingKey())
}

func TestKafkaDelivery_Exchange(t *testing.T) {
	d := &kafkaDelivery{msg: kafkago.Message{Topic: "my-topic"}}
	assert.Equal(t, "my-topic", d.Exchange())
}

func TestKafkaDelivery_Redelivered(t *testing.T) {
	d := &kafkaDelivery{msg: kafkago.Message{}}
	// Kafka does not track redelivery; always false.
	assert.False(t, d.Redelivered())
}

func TestKafkaDelivery_DeliveryTag(t *testing.T) {
	d := &kafkaDelivery{msg: kafkago.Message{Offset: 99}}
	assert.Equal(t, uint64(99), d.DeliveryTag())
}

func TestKafkaDelivery_Ack_NoReader(t *testing.T) {
	// Without a reader, Ack should be a no-op (returns nil).
	c := &Consumer{}
	d := &kafkaDelivery{
		msg:      kafkago.Message{Offset: 1},
		consumer: c,
	}
	err := d.Ack()
	assert.NoError(t, err)
	assert.True(t, d.ackedManually)
}

func TestKafkaDelivery_Nack_NoReader(t *testing.T) {
	c := &Consumer{}
	d := &kafkaDelivery{
		msg:      kafkago.Message{Offset: 1},
		consumer: c,
	}
	err := d.Nack(true)
	assert.NoError(t, err)
	assert.True(t, d.ackedManually)
}

func TestKafkaDelivery_Nack_NoRequeue_NoReader(t *testing.T) {
	c := &Consumer{}
	d := &kafkaDelivery{
		msg:      kafkago.Message{Offset: 1},
		consumer: c,
	}
	// Nack with requeue=false and no reader should still return nil.
	err := d.Nack(false)
	assert.NoError(t, err)
	assert.True(t, d.ackedManually)
}

func TestKafkaDelivery_Reject_NoReader(t *testing.T) {
	c := &Consumer{}
	d := &kafkaDelivery{
		msg:      kafkago.Message{Offset: 1},
		consumer: c,
	}
	err := d.Reject(false)
	assert.NoError(t, err)
	assert.True(t, d.ackedManually)
}

func TestKafkaDelivery_Reject_Requeue_NoReader(t *testing.T) {
	c := &Consumer{}
	d := &kafkaDelivery{
		msg:      kafkago.Message{Offset: 1},
		consumer: c,
	}
	err := d.Reject(true)
	assert.NoError(t, err)
	assert.True(t, d.ackedManually)
}

func TestKafkaDelivery_EmptyHeaders(t *testing.T) {
	d := &kafkaDelivery{msg: kafkago.Message{}}
	h := d.Headers()
	assert.NotNil(t, h)
	assert.Empty(t, h)
}

func TestKafkaDelivery_Message_EmptyKey(t *testing.T) {
	d := &kafkaDelivery{msg: kafkago.Message{Topic: "t", Partition: 0}}
	msg := d.Message()
	assert.Equal(t, "", msg.ID)
	assert.Equal(t, "0", msg.RoutingKey)
}
