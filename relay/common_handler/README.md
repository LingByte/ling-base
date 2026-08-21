# common_handler

Shared response handlers used by multiple provider adaptors for response formats that are identical across providers.

## Key Functions

- **`RerankHandler`** -- parses a rerank API response, normalizes usage (sets `PromptTokens` from `TotalTokens`), writes the JSON response to the `http.ResponseWriter`, and returns the extracted `dto.Usage`.

## Usage

```go
import "github.com/LingByte/ling-base/relay/common_handler"

// Inside an adaptor's DoResponse method:
usage, apiErr := common_handler.RerankHandler(ctx, info, resp, w)
if apiErr != nil {
    return nil, apiErr
}
return usage, nil
```
