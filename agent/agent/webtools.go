package agent

// webToolBetas are the Anthropic beta headers required for server-side web tools.
var webToolBetas = []string{
	"web-search-2025-03-05",
	"web-fetch-2025-09-10",
}

// defaultBetas are the standard beta headers sent with every request.
var defaultBetas = []string{
	"claude-code-20250219",
	"context-management-2025-06-27",
}

// webToolParams returns the server-side tool definitions (web_search,
// web_fetch) appended to a request when WebTools is enabled. These are executed
// by the Anthropic API; the loop never dispatches them locally.
func webToolParams() []Tool {
	return []Tool{
		{Type: "web_search_20250305", Name: "web_search"},
		{Type: "web_fetch_20250910", Name: "web_fetch"},
	}
}
