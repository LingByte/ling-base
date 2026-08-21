# relaymode

Relay mode constants and URL path-to-mode mapping functions.

## Key Constants

`RelayModeChatCompletions`, `RelayModeCompletions`, `RelayModeEmbeddings`,
`RelayModeModerations`, `RelayModeImagesGenerations`, `RelayModeImagesEdits`,
`RelayModeAudioSpeech`, `RelayModeAudioTranscription`, `RelayModeAudioTranslation`,
`RelayModeRerank`, `RelayModeResponses`, `RelayModeRealtime`, `RelayModeGemini`,
plus Midjourney and Suno task modes.

## Key Functions

- **`Path2RelayMode`** -- maps an HTTP request path to a relay mode int (e.g. `/v1/chat/completions` -> `RelayModeChatCompletions`).
- **`Path2RelayModeMidjourney`** -- maps Midjourney-specific paths to their relay modes.
- **`Path2RelaySuno`** -- maps Suno-specific paths and methods to their relay modes.

## Usage

```go
mode := relaymode.Path2RelayMode("/v1/chat/completions")
// mode == relaymode.RelayModeChatCompletions
```
