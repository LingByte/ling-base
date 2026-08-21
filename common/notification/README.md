# notification

Unified, provider-agnostic notification framework supporting email, SMS, instant messaging (IM), webhooks, in-app inbox, and mobile push.

## Architecture

The core abstraction is the `Channel` interface. Each channel type implements it with its own provider subpackages (`notification/email`, `notification/sms`, `notification/im`, `notification/webhook`, `notification/inbox`). A `Dispatcher` manages multiple channels and provides multi-channel failover: if one channel fails, the next enabled channel is tried.

Logging and template loading are pluggable via the `LogStore` and `TemplateStore` interfaces, so the package has no hard dependency on a specific database or ORM.

## Key types

- `Message` — unified notification payload with channel-specific fields
- `Channel` — interface implemented by each provider (`Name`, `Type`, `Send`, `Enabled`)
- `Dispatcher` — multi-channel failover dispatcher
- `LogStore` / `TemplateStore` — pluggable persistence interfaces
- `NewEmailMessage`, `NewSMSMessage`, `NewIMMessage`, `NewWebhookMessage`, `NewInboxMessage`, `NewPushMessage` — message constructors

## Quick start

```go
import "github.com/LingByte/ling-base/common/notification"

dispatcher := notification.NewDispatcher()
dispatcher.AddChannel(emailChannel)

err := dispatcher.Send(ctx, notification.Message{
    Type:    notification.TypeEmail,
    To:      "user@example.com",
    Subject: "Welcome",
    Body:    "Hello!",
})
```
