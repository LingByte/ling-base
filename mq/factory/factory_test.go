// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package factory

import (
	"testing"

	"github.com/LingByte/ling-base/mq"
	"github.com/LingByte/ling-base/mq/rabbitmq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBroker_UnsupportedType(t *testing.T) {
	_, err := NewBroker(mq.BrokerConfig{Type: "unknown"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported broker type")
}

func TestNewBroker_RabbitMQ_NilConfig(t *testing.T) {
	_, err := NewBroker(mq.BrokerConfig{Type: mq.BrokerRabbitMQ})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "RabbitMQConfig is required")
}

func TestNewBroker_RabbitMQ_WrongType(t *testing.T) {
	_, err := NewBroker(mq.BrokerConfig{
		Type:           mq.BrokerRabbitMQ,
		RabbitMQConfig: "not a config",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be *rabbitmq.Config")
}

func TestNewBroker_RabbitMQ_Success(t *testing.T) {
	broker, err := NewBroker(mq.BrokerConfig{
		Type:           mq.BrokerRabbitMQ,
		RabbitMQConfig: &rabbitmq.Config{URL: "amqp://localhost:5672/"},
	})
	require.NoError(t, err)
	assert.NotNil(t, broker)
	assert.False(t, broker.IsConnected())
	_ = broker.Close()
}

func TestNewBroker_Kafka_NilConfig(t *testing.T) {
	_, err := NewBroker(mq.BrokerConfig{Type: mq.BrokerKafka})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "KafkaConfig is required")
}

func TestNewBroker_RocketMQ_NilConfig(t *testing.T) {
	_, err := NewBroker(mq.BrokerConfig{Type: mq.BrokerRocketMQ})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "RocketMQConfig is required")
}

func TestNewBroker_ActiveMQ_NilConfig(t *testing.T) {
	_, err := NewBroker(mq.BrokerConfig{Type: mq.BrokerActiveMQ})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ActiveMQConfig is required")
}

func TestNewBroker_RedisStream_NilConfig(t *testing.T) {
	_, err := NewBroker(mq.BrokerConfig{Type: mq.BrokerRedisStream})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "RedisStreamConfig is required")
}

func TestBackendInfo(t *testing.T) {
	info := BackendInfo(mq.BrokerRabbitMQ)
	assert.Equal(t, "RabbitMQ", info.Name)
	assert.True(t, info.Persistent)
	assert.True(t, info.PubSub)

	info = BackendInfo(mq.BrokerKafka)
	assert.Equal(t, "Kafka", info.Name)
	assert.True(t, info.ConsumerGroup)

	info = BackendInfo(mq.BrokerRedisStream)
	assert.Equal(t, "Redis Streams", info.Name)
	assert.True(t, info.ConsumerGroup)
	assert.False(t, info.PubSub)

	info = BackendInfo("unknown")
	assert.Equal(t, "unknown", info.Name)
}

func TestAllBackendInfo(t *testing.T) {
	infos := AllBackendInfo()
	assert.Len(t, infos, len(mq.SupportedBrokers))
	for _, info := range infos {
		assert.NotEmpty(t, info.Name)
		assert.True(t, info.Persistent)
	}
}

func TestIsSupported(t *testing.T) {
	assert.True(t, mq.IsSupported(mq.BrokerRabbitMQ))
	assert.True(t, mq.IsSupported(mq.BrokerKafka))
	assert.True(t, mq.IsSupported(mq.BrokerRocketMQ))
	assert.True(t, mq.IsSupported(mq.BrokerActiveMQ))
	assert.True(t, mq.IsSupported(mq.BrokerRedisStream))
	assert.False(t, mq.IsSupported("unknown"))
}
