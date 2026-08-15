# mq

Broker-agnostic message queue abstraction with pluggable backends.

## Packages

- `mq/` — core interfaces: `Message`, `Delivery`, `Producer`, `Consumer`, `Broker`, `Handler`, `Middleware`
- `mq/rabbitmq/` — RabbitMQ backend (amqp091-go)

## Quick start

```go
import (
    "github.com/LingByte/ling-base/mq"
    "github.com/LingByte/ling-base/mq/rabbitmq"
)

broker, _ := rabbitmq.New(rabbitmq.DefaultConfig())
defer broker.Close()
broker.Connect()

// Declare topology
broker.DeclareExchange("events", mq.DefaultExchangeOptions())
broker.DeclareQueue("events.queue", mq.DefaultQueueOptions())
broker.Bind("events.queue", "events", "events.#")

// Produce
producer, _ := broker.Producer("events", mq.PublishOptions{Persistent: true})
producer.Publish(ctx, &mq.Message{
    RoutingKey:  "events.user.created",
    ContentType: "application/json",
    Body:        []byte(`{"user_id":123}`),
})

// Consume with middleware
consumer, _ := broker.Consumer("events.queue", mq.ConsumeOptions{
    Handler: func(ctx context.Context, d mq.Delivery) error {
        log.Println(string(d.Body()))
        return d.Ack()
    },
    Middleware: []mq.Middleware{
        mq.RecoverMiddleware(logger),
        mq.MetricsMiddleware(metrics),
        mq.RetryMiddleware(mq.RetryConfig{MaxAttempts: 3, Backoff: time.Second}),
    },
    QosPrefetchCount: 10,
    Concurrency:      4,
})
consumer.Start(ctx)
```

## Middleware

| Middleware | Description |
|---|---|
| `RecoverMiddleware` | Recovers from handler panics |
| `LoggingMiddleware` | Logs each delivery before/after |
| `MetricsMiddleware` | Records consume/error/redelivered metrics |
| `RetryMiddleware` | Retries failed deliveries with backoff |
| `DeadLetterMiddleware` | Routes failures to a dead-letter handler |
| `Chain` | Composes multiple middleware |

## Backends

| Backend | Status | Features |
|---|---|---|
| RabbitMQ | ✅ | Exchange/queue/binding, QoS, publisher confirms, auto-reconnect, concurrent consumers |
| Redis Streams | planned | |
| Kafka | planned | |
| NATS | planned | |
