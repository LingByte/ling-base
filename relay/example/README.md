# example

Runnable example programs demonstrating how to use the relay library.

## Examples

| Directory | Description |
|-----------|-------------|
| `basic/` | Chat completion with the OpenAI provider and in-memory usage metering |
| `gemini/` | Chat completion using the Gemini provider |
| `midjourney/` | Midjourney image generation task submission and polling |
| `realtime/` | OpenAI Realtime API WebSocket session bridging |
| `suno/` | Suno music generation task submission and polling |

## Running

```bash
export OPENAI_API_KEY="sk-..."
cd example/basic && go run main.go
```

Each example creates a `relay.Client` with a provider and optional `meter.MemoryMeter`, sends a request, and prints the response and aggregated usage statistics.
