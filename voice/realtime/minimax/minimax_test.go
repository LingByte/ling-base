package realtime

import (
	"encoding/base64"
	"strings"
	"testing"

	base "github.com/LingByte/ling-base/voice/realtime"
)

func TestBuildURLDefault(t *testing.T) {
	u, err := BuildURL("", "MiniMax-Text-01-realtime")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.HasPrefix(u, "wss://api.minimaxi.com/ws/v1/realtime") {
		t.Errorf("u = %s, want wss://api.minimaxi.com prefix", u)
	}
	if !strings.Contains(u, "model=MiniMax-Text-01-realtime") {
		t.Errorf("u = %s, want model query", u)
	}
}

func TestBuildURLBadScheme(t *testing.T) {
	_, err := BuildURL("http://example.com", "m1")
	if err == nil {
		t.Fatal("expected error for http scheme")
	}
}

func TestNewMissingAPIKey(t *testing.T) {
	_, err := New(map[string]any{}, base.Options{OnEvent: func(base.Event) {}})
	if err == nil {
		t.Fatal("expected error for missing apiKey")
	}
	if !strings.Contains(err.Error(), "apiKey") {
		t.Errorf("err = %v, want apiKey error", err)
	}
}

func TestNewDefaults(t *testing.T) {
	a, err := New(map[string]any{"apiKey": "test-key"}, base.Options{OnEvent: func(base.Event) {}})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	agent := a.(*Agent)
	if agent.cfg.Model != DefaultModel {
		t.Errorf("Model = %s, want %s", agent.cfg.Model, DefaultModel)
	}
	if agent.cfg.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %s, want %s", agent.cfg.BaseURL, DefaultBaseURL)
	}
}

func TestNewCustomVoice(t *testing.T) {
	a, err := New(map[string]any{
		"apiKey": "k",
		"voice":  "female-shaonv",
	}, base.Options{OnEvent: func(base.Event) {}})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	agent := a.(*Agent)
	if agent.cfg.Voice != "female-shaonv" {
		t.Errorf("Voice = %s, want female-shaonv", agent.cfg.Voice)
	}
}

func TestDispatchSessionCreated(t *testing.T) {
	var got []base.Event
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{
		OnEvent: func(e base.Event) { got = append(got, e) },
	})
	agent := a.(*Agent)
	agent.ba.Dispatch = agent.dispatch
	agent.ba.Dispatch([]byte(`{"type":"session.created"}`))
	if len(got) != 1 || got[0].Type != base.EventSessionOpen {
		t.Fatalf("got = %+v, want EventSessionOpen", got)
	}
}

func TestDispatchAssistantAudioDelta(t *testing.T) {
	var got []base.Event
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{
		OnEvent: func(e base.Event) { got = append(got, e) },
	})
	agent := a.(*Agent)
	agent.ba.FireOnce(base.Event{Type: base.EventSessionOpen})

	pcm := []byte{0x01, 0x02, 0x03, 0x04}
	encoded := base64.StdEncoding.EncodeToString(pcm)
	agent.ba.Dispatch = agent.dispatch
	agent.ba.Dispatch([]byte(`{"type":"response.audio.delta","delta":"` + encoded + `"}`))
	found := false
	for _, e := range got {
		if e.Type == base.EventAssistantAudio && len(e.AudioPC) == len(pcm) {
			found = true
		}
	}
	if !found {
		t.Fatalf("no EventAssistantAudio in %+v", got)
	}
}
