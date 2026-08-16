package realtime

import (
	"strings"
	"testing"

	base "github.com/LingByte/ling-base/realtime"
)

func TestNewMissingCredentials(t *testing.T) {
	_, err := New(map[string]any{}, base.Options{OnEvent: func(base.Event) {}})
	if err == nil {
		t.Fatal("expected error for missing credentials")
	}
	if !strings.Contains(err.Error(), "appId") {
		t.Errorf("err = %v, want appId error", err)
	}
}

func TestNewMissingAPIKey(t *testing.T) {
	_, err := New(map[string]any{"appId": "id"}, base.Options{OnEvent: func(base.Event) {}})
	if err == nil {
		t.Fatal("expected error for missing apiKey")
	}
}

func TestNewMissingAPISecret(t *testing.T) {
	_, err := New(map[string]any{
		"appId":  "id",
		"apiKey": "key",
	}, base.Options{OnEvent: func(base.Event) {}})
	if err == nil {
		t.Fatal("expected error for missing apiSecret")
	}
}

func TestNewDefaults(t *testing.T) {
	a, err := New(map[string]any{
		"appId":     "id",
		"apiKey":    "key",
		"apiSecret": "secret",
	}, base.Options{OnEvent: func(base.Event) {}})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	agent := a.(*Agent)
	if agent.cfg.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %s, want %s", agent.cfg.BaseURL, DefaultBaseURL)
	}
	if agent.cfg.Voice != DefaultVoice {
		t.Errorf("Voice = %s, want %s", agent.cfg.Voice, DefaultVoice)
	}
	if agent.cfg.InteractMode != "continuous" {
		t.Errorf("InteractMode = %s, want continuous", agent.cfg.InteractMode)
	}
}

func TestNewCustomConfig(t *testing.T) {
	a, err := New(map[string]any{
		"appId":        "id",
		"apiKey":       "key",
		"apiSecret":    "secret",
		"voice":        "custom_voice",
		"interactMode": "continuous_vad",
	}, base.Options{OnEvent: func(base.Event) {}})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	agent := a.(*Agent)
	if agent.cfg.Voice != "custom_voice" {
		t.Errorf("Voice = %s, want custom_voice", agent.cfg.Voice)
	}
	if agent.cfg.InteractMode != "continuous_vad" {
		t.Errorf("InteractMode = %s, want continuous_vad", agent.cfg.InteractMode)
	}
}

func TestBuildSignedURL(t *testing.T) {
	u, err := BuildSignedURL(Config{
		AppID:     "id",
		APIKey:    "key",
		APISecret: "secret",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.HasPrefix(u, "wss://sparkos.xfyun.cn/v1/openapi/chat") {
		t.Errorf("u = %s, want wss://sparkos.xfyun.cn prefix", u)
	}
	if !strings.Contains(u, "authorization=") {
		t.Errorf("u = %s, want authorization query", u)
	}
	if !strings.Contains(u, "date=") {
		t.Errorf("u = %s, want date query", u)
	}
}

func TestBuildSignedURLBadScheme(t *testing.T) {
	_, err := BuildSignedURL(Config{
		AppID:     "id",
		APIKey:    "key",
		APISecret: "secret",
		BaseURL:   "http://example.com",
	})
	if err == nil {
		t.Fatal("expected error for http scheme")
	}
}

func TestDispatchHandshake(t *testing.T) {
	var got []base.Event
	a, _ := New(map[string]any{
		"appId":     "id",
		"apiKey":    "key",
		"apiSecret": "secret",
	}, base.Options{OnEvent: func(e base.Event) { got = append(got, e) }})
	agent := a.(*Agent)
	agent.ba.Dispatch = agent.dispatch
	agent.ba.Dispatch([]byte(`{"header":{"code":0,"type":"handshake"}}`))
	if len(got) != 1 || got[0].Type != base.EventSessionOpen {
		t.Fatalf("got = %+v, want EventSessionOpen", got)
	}
}

func TestDispatchErrorCode(t *testing.T) {
	var got []base.Event
	a, _ := New(map[string]any{
		"appId":     "id",
		"apiKey":    "key",
		"apiSecret": "secret",
	}, base.Options{OnEvent: func(e base.Event) { got = append(got, e) }})
	agent := a.(*Agent)
	agent.ba.FireOnce(base.Event{Type: base.EventSessionOpen})
	agent.ba.Dispatch = agent.dispatch
	agent.ba.Dispatch([]byte(`{"header":{"code":10001,"message":"auth failed"}}`))
	found := false
	for _, e := range got {
		if e.Type == base.EventError && e.Fatal {
			found = true
		}
	}
	if !found {
		t.Fatalf("no fatal EventError in %+v", got)
	}
}

func TestDispatchASRResult(t *testing.T) {
	var got []base.Event
	a, _ := New(map[string]any{
		"appId":     "id",
		"apiKey":    "key",
		"apiSecret": "secret",
	}, base.Options{OnEvent: func(e base.Event) { got = append(got, e) }})
	agent := a.(*Agent)
	agent.ba.FireOnce(base.Event{Type: base.EventSessionOpen})
	agent.ba.Dispatch = agent.dispatch
	agent.ba.Dispatch([]byte(`{"header":{"code":0,"type":"asr_result"},"payload":{"asr":{"text":"hello"}}}`))
	found := false
	for _, e := range got {
		if e.Type == base.EventUserTranscript && e.Text == "hello" {
			found = true
		}
	}
	if !found {
		t.Fatalf("no EventUserTranscript in %+v", got)
	}
}
