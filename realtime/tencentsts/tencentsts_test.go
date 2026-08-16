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
	if !strings.Contains(err.Error(), "secretId") {
		t.Errorf("err = %v, want secretId error", err)
	}
}

func TestNewMissingSecretKey(t *testing.T) {
	_, err := New(map[string]any{"secretId": "id"}, base.Options{OnEvent: func(base.Event) {}})
	if err == nil {
		t.Fatal("expected error for missing secretKey")
	}
	if !strings.Contains(err.Error(), "secretKey") {
		t.Errorf("err = %v, want secretKey error", err)
	}
}

func TestNewDefaults(t *testing.T) {
	a, err := New(map[string]any{
		"secretId":  "id",
		"secretKey": "key",
	}, base.Options{OnEvent: func(base.Event) {}})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	agent := a.(*Agent)
	if agent.cfg.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %s, want %s", agent.cfg.BaseURL, DefaultBaseURL)
	}
	if agent.cfg.VoiceID != DefaultVoice {
		t.Errorf("VoiceID = %s, want %s", agent.cfg.VoiceID, DefaultVoice)
	}
	if agent.cfg.SourceLang != "en" {
		t.Errorf("SourceLang = %s, want en", agent.cfg.SourceLang)
	}
}

func TestNewCustomConfig(t *testing.T) {
	a, err := New(map[string]any{
		"secretId":   "id",
		"secretKey":  "key",
		"voiceId":    "201001",
		"sourceLang": "zh",
	}, base.Options{OnEvent: func(base.Event) {}})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	agent := a.(*Agent)
	if agent.cfg.VoiceID != "201001" {
		t.Errorf("VoiceID = %s, want 201001", agent.cfg.VoiceID)
	}
	if agent.cfg.SourceLang != "zh" {
		t.Errorf("SourceLang = %s, want zh", agent.cfg.SourceLang)
	}
}

func TestBuildURLDefault(t *testing.T) {
	u, err := BuildURL(Config{SecretID: "test-id"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.HasPrefix(u, "wss://mps.cloud.tencent.com/sts/v1/") {
		t.Errorf("u = %s, want wss://mps.cloud.tencent.com prefix", u)
	}
	if !strings.Contains(u, "secretid=test-id") {
		t.Errorf("u = %s, want secretid query", u)
	}
}

func TestBuildURLBadScheme(t *testing.T) {
	_, err := BuildURL(Config{
		SecretID: "id",
		BaseURL:  "http://example.com",
	})
	if err == nil {
		t.Fatal("expected error for http scheme")
	}
}

func TestDispatchHandshakeResult(t *testing.T) {
	var got []base.Event
	a, _ := New(map[string]any{
		"secretId":  "id",
		"secretKey": "key",
	}, base.Options{OnEvent: func(e base.Event) { got = append(got, e) }})
	agent := a.(*Agent)
	agent.ba.Dispatch = agent.dispatch
	agent.ba.Dispatch([]byte(`{"type":"HandshakeResult"}`))
	if len(got) != 1 || got[0].Type != base.EventSessionOpen {
		t.Fatalf("got = %+v, want EventSessionOpen", got)
	}
}

func TestDispatchError(t *testing.T) {
	var got []base.Event
	a, _ := New(map[string]any{
		"secretId":  "id",
		"secretKey": "key",
	}, base.Options{OnEvent: func(e base.Event) { got = append(got, e) }})
	agent := a.(*Agent)
	agent.ba.FireOnce(base.Event{Type: base.EventSessionOpen})
	agent.ba.Dispatch = agent.dispatch
	agent.ba.Dispatch([]byte(`{"type":"Error","error":"auth failed"}`))
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
