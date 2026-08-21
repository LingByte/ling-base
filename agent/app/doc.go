// Package app is the embeddable ling-agent runtime.
//
// Configure an [Options] value (provider, model, prompt, …) and call [Run] or
// [Execute]. File-based settings under ~/.ling-agent and ./.ling-agent still
// load; non-empty fields on Options override them.
//
//	os.Exit(app.Execute(app.Options{
//		Provider:  "openai",
//		Model:     "gpt-5.4",
//		BaseURL:   "https://rightapi.ai/codex/v1",
//		APIKeyEnv: "RIGHTAPI_API_KEY",
//		Prompt:    "hello",
//		Print:     true,
//	}))
package app
