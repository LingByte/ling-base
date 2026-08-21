# setting

Minimal configuration stubs for relay providers. In LingRein these are backed by a database; in library mode they return sensible defaults that the consuming application can override.

## Key Types

- **`GlobalSettings`** -- global relay settings (e.g. `PassThroughRequestEnabled`).
- **`ModelSetting`** -- model-specific settings (thinking adapter toggle and budget percentage).
- **`GeminiSetting`** -- Gemini-specific settings (thinking adapter, function response ID handling).

## Key Functions

- **`GetGlobalSettings`** / **`SetGlobalSettings`** -- access and replace global settings.
- **`GetModelSetting`** -- returns the model settings instance.
- **`GetGeminiSettings`** / **`GetGeminiVersionSetting`** -- Gemini settings and API version (defaults to `v1beta`).
- **`ShouldPreserveThinkingSuffix`** -- defaults to `false`.
- **`IsSyncImageModel`** -- defaults to `false`.
- **`WithCompactModelVariants`** / **`WithCompactModelSuffix`** -- identity stubs in library mode.

## Usage

```go
s := setting.GetGlobalSettings()
s.PassThroughRequestEnabled = true
```
