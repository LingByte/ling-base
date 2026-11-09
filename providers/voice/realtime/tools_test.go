package realtime

import (
	"encoding/json"
	"testing"
)

func TestToolsForSessionEmpty(t *testing.T) {
	if got := ToolsForSession(nil); got != nil {
		t.Errorf("ToolsForSession(nil) = %v, want nil", got)
	}
	if got := ToolsForSession([]Tool{}); got != nil {
		t.Errorf("ToolsForSession([]) = %v, want nil", got)
	}
}

func TestToolsForSessionSkipsEmptyName(t *testing.T) {
	tools := []Tool{
		{Name: "", Description: "skip me"},
		{Name: "get_weather", Description: "Get weather"},
	}
	got := ToolsForSession(tools)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	fn, _ := got[0]["function"].(map[string]any)
	if fn["name"] != "get_weather" {
		t.Errorf("name = %v, want get_weather", fn["name"])
	}
	if got[0]["type"] != "function" {
		t.Errorf("type = %v, want function", got[0]["type"])
	}
}

func TestToolsForSessionDefaultParams(t *testing.T) {
	tools := []Tool{{Name: "noop", Description: "no params"}}
	got := ToolsForSession(tools)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	fn, _ := got[0]["function"].(map[string]any)
	params, _ := fn["parameters"].(map[string]any)
	if params["type"] != "object" {
		t.Errorf("default params type = %v, want object", params["type"])
	}
}

func TestToolsForSessionValidJSON(t *testing.T) {
	tools := []Tool{{
		Name:        "search",
		Description: "Search",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"q":{"type":"string"}},"required":["q"]}`),
	}}
	got := ToolsForSession(tools)
	fn, _ := got[0]["function"].(map[string]any)
	params, _ := fn["parameters"].(map[string]any)
	if params["type"] != "object" {
		t.Errorf("params type = %v, want object", params["type"])
	}
	props, _ := params["properties"].(map[string]any)
	if _, ok := props["q"]; !ok {
		t.Errorf("expected q property, got %v", props)
	}
}

func TestToolsForSessionInvalidJSONFallback(t *testing.T) {
	tools := []Tool{{
		Name:        "bad",
		Description: "Bad params",
		Parameters:  json.RawMessage(`{not valid json`),
	}}
	got := ToolsForSession(tools)
	fn, _ := got[0]["function"].(map[string]any)
	params, _ := fn["parameters"].(map[string]any)
	if params["type"] != "object" {
		t.Errorf("fallback params type = %v, want object", params["type"])
	}
	if _, ok := params["properties"].(map[string]any); !ok {
		t.Errorf("fallback params missing properties map")
	}
}
