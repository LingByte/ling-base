# service

Shared utility functions for relay providers, bridging provider adaptors to the relaykit conversion system and providing common helpers for response handling and usage estimation.

## Key Functions

- **`ConvertRequest`** -- converts a request to a target relay format (openai/claude/gemini) via relaykit.
- **`ConvertRequestByID`** -- converts a request using a named converter.
- **`ResponseText2Usage`** -- estimates `dto.Usage` from response text when the provider does not return token counts.
- **`EstimateTokenByModel`** -- heuristic token estimation (~4 chars/token for English, ~2 chars/token for CJK).
- **`ValidUsage`** -- checks whether a usage struct has non-zero values.
- **`GetImageFromUrl`** -- downloads an image from a URL and returns its data and MIME type.
- **`GetBase64Data`** -- returns base64 data from a URL or data URI.
- **`ShouldCopyUpstreamHeader`** -- reports whether a header should be forwarded to the client (skips hop-by-hop headers).
- **`IOCopyBytesGracefully`** -- writes bytes to a ResponseWriter with JSON content type.
- **`CloseResponseBodyGracefully`** -- closes a response body, ignoring errors.
- **`TaskErrorWrapper`** / **`TaskErrorWrapperLocal`** -- wrap errors into `common.TaskError`.

## Usage

```go
usage := service.ResponseText2Usage(responseText, "gpt-4o", 100)
```
