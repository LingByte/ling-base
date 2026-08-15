// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package mq

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// BatchPublisher tests
// ============================================================

type mockProducer struct {
	published []*Message
	err       error
	closed    bool
}

func (m *mockProducer) Publish(ctx context.Context, msg *Message) error {
	if m.err != nil {
		return m.err
	}
	m.published = append(m.published, msg)
	return nil
}

func (m *mockProducer) Close() error { m.closed = true; return nil }

func TestPublishBatch_Success(t *testing.T) {
	p := &mockProducer{}
	msgs := []*Message{
		{Body: []byte("msg1")},
		{Body: []byte("msg2")},
		{Body: []byte("msg3")},
	}
	err := PublishBatch(context.Background(), p, msgs)
	assert.NoError(t, err)
	assert.Len(t, p.published, 3)
}

func TestPublishBatch_NilProducer(t *testing.T) {
	err := PublishBatch(context.Background(), nil, []*Message{{Body: []byte("x")}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "producer is nil")
}

func TestPublishBatch_Error(t *testing.T) {
	p := &mockProducer{err: errors.New("publish failed")}
	err := PublishBatch(context.Background(), p, []*Message{{Body: []byte("x")}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "batch publish failed at index 0")
}

func TestPublishBatch_Empty(t *testing.T) {
	p := &mockProducer{}
	err := PublishBatch(context.Background(), p, nil)
	assert.NoError(t, err)
	assert.Empty(t, p.published)
}

// ============================================================
// JSON helpers tests
// ============================================================

func TestNewJSONMessage_Success(t *testing.T) {
	msg, err := NewJSONMessage(map[string]any{"key": "value"})
	require.NoError(t, err)
	assert.Equal(t, "application/json", msg.ContentType)
	assert.NotEmpty(t, msg.Body)
	assert.False(t, msg.Timestamp.IsZero())

	var m map[string]any
	err = json.Unmarshal(msg.Body, &m)
	assert.NoError(t, err)
	assert.Equal(t, "value", m["key"])
}

func TestNewJSONMessage_NilPayload(t *testing.T) {
	msg, err := NewJSONMessage(nil)
	require.NoError(t, err)
	assert.Equal(t, "application/json", msg.ContentType)
	assert.Equal(t, "null", string(msg.Body))
}

func TestNewJSONMessage_InvalidPayload(t *testing.T) {
	_, err := NewJSONMessage(make(chan int))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshal json")
}

func TestDecodeJSON_Success(t *testing.T) {
	d := newFakeDelivery([]byte(`{"name":"test","value":42}`))
	var result struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	err := DecodeJSON(d, &result)
	assert.NoError(t, err)
	assert.Equal(t, "test", result.Name)
	assert.Equal(t, 42, result.Value)
}

func TestDecodeJSON_NilDelivery(t *testing.T) {
	err := DecodeJSON(nil, &struct{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delivery is nil")
}

func TestDecodeJSON_InvalidJSON(t *testing.T) {
	d := newFakeDelivery([]byte(`not json`))
	err := DecodeJSON(d, &struct{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal json")
}

func TestDecodeJSONBody_Success(t *testing.T) {
	var result struct {
		X int `json:"x"`
	}
	err := DecodeJSONBody([]byte(`{"x":99}`), &result)
	assert.NoError(t, err)
	assert.Equal(t, 99, result.X)
}

func TestDecodeJSONBody_InvalidJSON(t *testing.T) {
	err := DecodeJSONBody([]byte(`bad`), &struct{}{})
	assert.Error(t, err)
}

// ============================================================
// HealthChecker tests
// ============================================================

type mockHealthChecker struct {
	connected bool
	checkErr  error
}

func (m *mockHealthChecker) Connect() error                    { return nil }
func (m *mockHealthChecker) IsConnected() bool                 { return m.connected }
func (m *mockHealthChecker) Close() error                       { return nil }
func (m *mockHealthChecker) Producer(string, PublishOptions) (Producer, error) { return nil, nil }
func (m *mockHealthChecker) Consumer(string, ConsumeOptions) (Consumer, error) { return nil, nil }
func (m *mockHealthChecker) DeclareExchange(string, ExchangeOptions) error     { return nil }
func (m *mockHealthChecker) DeclareQueue(string, QueueOptions) error           { return nil }
func (m *mockHealthChecker) Bind(string, string, string) error                 { return nil }
func (m *mockHealthChecker) Unbind(string, string, string) error               { return nil }
func (m *mockHealthChecker) DeleteQueue(string) error                           { return nil }
func (m *mockHealthChecker) DeleteExchange(string) error                        { return nil }
func (m *mockHealthChecker) Check(ctx context.Context) error                    { return m.checkErr }

func TestCheckHealth_Healthy(t *testing.T) {
	b := &mockHealthChecker{connected: true}
	status := CheckHealth(context.Background(), b, "test")
	assert.True(t, status.Healthy)
	assert.Equal(t, "test", status.Backend)
	assert.True(t, status.Connected)
	assert.True(t, status.Latency >= 0)
}

func TestCheckHealth_Unhealthy(t *testing.T) {
	b := &mockHealthChecker{connected: false, checkErr: errors.New("connection lost")}
	status := CheckHealth(context.Background(), b, "test")
	assert.False(t, status.Healthy)
	assert.Contains(t, status.Error, "connection lost")
}

func TestCheckHealth_FallbackIsConnected(t *testing.T) {
	// mockProducer doesn't implement HealthChecker, so fallback to IsConnected
	// But we need a Broker... use a simple mock
	b := &fallbackBroker{connected: true}
	status := CheckHealth(context.Background(), b, "fallback")
	assert.True(t, status.Healthy)
	assert.True(t, status.Connected)
}

func TestCheckHealth_FallbackNotConnected(t *testing.T) {
	b := &fallbackBroker{connected: false}
	status := CheckHealth(context.Background(), b, "fallback")
	assert.False(t, status.Healthy)
	assert.False(t, status.Connected)
}

func TestHealthStatus_String(t *testing.T) {
	h := HealthStatus{Healthy: true, Backend: "rabbitmq", Connected: true, Latency: 5 * time.Millisecond}
	s := h.String()
	assert.Contains(t, s, "healthy")
	assert.Contains(t, s, "rabbitmq")

	h2 := HealthStatus{Healthy: false, Backend: "kafka", Error: "timeout"}
	s2 := h2.String()
	assert.Contains(t, s2, "unhealthy")
	assert.Contains(t, s2, "timeout")
}

// fallbackBroker implements Broker but NOT HealthChecker
type fallbackBroker struct{ connected bool }

func (f *fallbackBroker) Connect() error                    { return nil }
func (f *fallbackBroker) IsConnected() bool                 { return f.connected }
func (f *fallbackBroker) Close() error                       { return nil }
func (f *fallbackBroker) Producer(string, PublishOptions) (Producer, error) { return nil, nil }
func (f *fallbackBroker) Consumer(string, ConsumeOptions) (Consumer, error) { return nil, nil }
func (f *fallbackBroker) DeclareExchange(string, ExchangeOptions) error     { return nil }
func (f *fallbackBroker) DeclareQueue(string, QueueOptions) error           { return nil }
func (f *fallbackBroker) Bind(string, string, string) error                 { return nil }
func (f *fallbackBroker) Unbind(string, string, string) error               { return nil }
func (f *fallbackBroker) DeleteQueue(string) error                           { return nil }
func (f *fallbackBroker) DeleteExchange(string) error                        { return nil }

// ============================================================
// Topology builder tests
// ============================================================

type recordingBroker struct {
	exchanges []string
	queues    []string
	binds     [][3]string // queue, exchange, routingKey
	failAt    string
}

func (r *recordingBroker) Connect() error                    { return nil }
func (r *recordingBroker) IsConnected() bool                 { return true }
func (r *recordingBroker) Close() error                       { return nil }
func (r *recordingBroker) Producer(string, PublishOptions) (Producer, error) { return nil, nil }
func (r *recordingBroker) Consumer(string, ConsumeOptions) (Consumer, error) { return nil, nil }
func (r *recordingBroker) DeclareExchange(name string, _ ExchangeOptions) error {
	if r.failAt == "exchange:"+name {
		return errors.New("exchange failed")
	}
	r.exchanges = append(r.exchanges, name)
	return nil
}
func (r *recordingBroker) DeclareQueue(name string, _ QueueOptions) error {
	if r.failAt == "queue:"+name {
		return errors.New("queue failed")
	}
	r.queues = append(r.queues, name)
	return nil
}
func (r *recordingBroker) Bind(queue, exchange, routingKey string) error {
	if r.failAt == "bind:"+queue {
		return errors.New("bind failed")
	}
	r.binds = append(r.binds, [3]string{queue, exchange, routingKey})
	return nil
}
func (r *recordingBroker) Unbind(string, string, string) error { return nil }
func (r *recordingBroker) DeleteQueue(string) error             { return nil }
func (r *recordingBroker) DeleteExchange(string) error          { return nil }

func TestTopology_FullChain(t *testing.T) {
	b := &recordingBroker{}
	topo := NewTopology(b).
		Exchange("events").
		Queue("events.queue").
		Bind("events.queue", "events", "events.#")

	err := topo.Apply()
	assert.NoError(t, err)
	assert.Equal(t, []string{"events"}, b.exchanges)
	assert.Equal(t, []string{"events.queue"}, b.queues)
	assert.Equal(t, [][3]string{{"events.queue", "events", "events.#"}}, b.binds)
}

func TestTopology_MultipleExchanges(t *testing.T) {
	b := &recordingBroker{}
	topo := NewTopology(b).
		Exchange("ex1", ExchangeOptions{Kind: "direct"}).
		Exchange("ex2", ExchangeOptions{Kind: "fanout"}).
		Queue("q1").
		Queue("q2").
		Bind("q1", "ex1", "rk1").
		Bind("q2", "ex2", "")

	err := topo.Apply()
	assert.NoError(t, err)
	assert.Equal(t, []string{"ex1", "ex2"}, b.exchanges)
	assert.Equal(t, []string{"q1", "q2"}, b.queues)
	assert.Len(t, b.binds, 2)
}

func TestTopology_ExchangeError(t *testing.T) {
	b := &recordingBroker{failAt: "exchange:bad"}
	topo := NewTopology(b).Exchange("bad")
	err := topo.Apply()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "declare exchange")
}

func TestTopology_QueueError(t *testing.T) {
	b := &recordingBroker{failAt: "queue:bad"}
	topo := NewTopology(b).Exchange("ok").Queue("bad")
	err := topo.Apply()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "declare queue")
}

func TestTopology_BindError(t *testing.T) {
	b := &recordingBroker{failAt: "bind:q1"}
	topo := NewTopology(b).Exchange("ex").Queue("q1").Bind("q1", "ex", "rk")
	err := topo.Apply()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bind")
}

func TestTopology_NilBroker(t *testing.T) {
	topo := NewTopology(nil).Exchange("test")
	err := topo.Apply()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "broker is nil")
}

func TestTopology_MustApply_Panics(t *testing.T) {
	b := &recordingBroker{failAt: "exchange:bad"}
	topo := NewTopology(b).Exchange("bad")
	assert.Panics(t, func() { topo.MustApply() })
}

func TestTopology_MustApply_Success(t *testing.T) {
	b := &recordingBroker{}
	topo := NewTopology(b).Exchange("ok")
	assert.NotPanics(t, func() { topo.MustApply() })
}

func TestTopology_Reset(t *testing.T) {
	b := &recordingBroker{}
	topo := NewTopology(b).Exchange("ex").Queue("q")
	topo.Reset()
	assert.Empty(t, topo.exchanges)
	assert.Empty(t, topo.queues)
	assert.Empty(t, topo.binds)
}

func TestTopology_ChainAfterError(t *testing.T) {
	// When err is set, subsequent calls should be no-ops
	b := &recordingBroker{failAt: "exchange:bad"}
	topo := NewTopology(b)
	topo.err = errors.New("preset error")
	topo.Exchange("should-skip").Queue("should-skip").Bind("q", "ex", "rk")
	assert.Empty(t, topo.exchanges)
	assert.Empty(t, topo.queues)
	assert.Empty(t, topo.binds)
}

func TestTopology_CustomOptions(t *testing.T) {
	b := &recordingBroker{}
	topo := NewTopology(b).
		Exchange("custom", ExchangeOptions{Kind: "headers", Durable: false}).
		Queue("custom.q", QueueOptions{Durable: false, AutoDelete: true})
	err := topo.Apply()
	assert.NoError(t, err)
}

// ============================================================
// Dead-letter / TTL helpers tests
// ============================================================

func TestWithDeadLetter(t *testing.T) {
	opts := WithDeadLetter(DefaultQueueOptions(), "dlx", "dlx.key")
	assert.Equal(t, "dlx", opts.Args["x-dead-letter-exchange"])
	assert.Equal(t, "dlx.key", opts.Args["x-dead-letter-routing-key"])
}

func TestWithDeadLetter_EmptyRoutingKey(t *testing.T) {
	opts := WithDeadLetter(DefaultQueueOptions(), "dlx", "")
	assert.Equal(t, "dlx", opts.Args["x-dead-letter-exchange"])
	_, exists := opts.Args["x-dead-letter-routing-key"]
	assert.False(t, exists)
}

func TestWithDeadLetter_PreservesExistingArgs(t *testing.T) {
	opts := QueueOptions{Args: map[string]any{"existing": "val"}}
	opts = WithDeadLetter(opts, "dlx", "")
	assert.Equal(t, "val", opts.Args["existing"])
	assert.Equal(t, "dlx", opts.Args["x-dead-letter-exchange"])
}

func TestWithMessageTTL(t *testing.T) {
	opts := WithMessageTTL(DefaultQueueOptions(), 30*time.Second)
	assert.Equal(t, int64(30000), opts.Args["x-message-ttl"])
}

func TestWithQueueTTL(t *testing.T) {
	opts := WithQueueTTL(DefaultQueueOptions(), 1*time.Hour)
	assert.Equal(t, int64(3600000), opts.Args["x-expires"])
}

func TestWithMaxPriority(t *testing.T) {
	opts := WithMaxPriority(DefaultQueueOptions(), 10)
	assert.Equal(t, 10, opts.Args["x-max-priority"])
}

// ============================================================
// types.go tests
// ============================================================

func TestBrokerType_String(t *testing.T) {
	tests := []struct {
		bt   BrokerType
		want string
	}{
		{BrokerRabbitMQ, "rabbitmq"},
		{BrokerKafka, "kafka"},
		{BrokerRocketMQ, "rocketmq"},
		{BrokerActiveMQ, "activemq"},
		{BrokerRedisStream, "redisstream"},
		{BrokerType("custom"), "custom"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.bt.String())
		})
	}
}

func TestIsSupported(t *testing.T) {
	assert.True(t, IsSupported(BrokerRabbitMQ))
	assert.True(t, IsSupported(BrokerKafka))
	assert.True(t, IsSupported(BrokerRocketMQ))
	assert.True(t, IsSupported(BrokerActiveMQ))
	assert.True(t, IsSupported(BrokerRedisStream))
	assert.False(t, IsSupported("unknown"))
	assert.False(t, IsSupported(""))
}

func TestSupportedBrokers(t *testing.T) {
	assert.Len(t, SupportedBrokers, 5)
}

// ============================================================
// atomicCounter.add test
// ============================================================

func TestAtomicCounter_Add(t *testing.T) {
	var c atomicCounter
	c.add(5)
	assert.Equal(t, int64(5), c.load())
	c.add(3)
	assert.Equal(t, int64(8), c.load())
	c.add(-2)
	assert.Equal(t, int64(6), c.load())
}
