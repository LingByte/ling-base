package realtime

import (
	"strings"
	"sync"
	"testing"

	base "github.com/LingByte/ling-base/voice/realtime"
)

func TestBuildURLDefault(t *testing.T) {
	u, err := BuildURL("", "gpt-4o-realtime-preview")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.HasPrefix(u, "wss://api.openai.com/v1/realtime") {
		t.Errorf("u = %s, want wss://api.openai.com prefix", u)
	}
	if !strings.Contains(u, "model=gpt-4o-realtime-preview") {
		t.Errorf("u = %s, want model query", u)
	}
}

func TestBuildURLCustomBase(t *testing.T) {
	u, err := BuildURL("wss://example.com/realtime", "m1")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(u, "model=m1") {
		t.Errorf("u = %s, want model=m1", u)
	}
}

func TestBuildURLBadScheme(t *testing.T) {
	_, err := BuildURL("http://example.com", "m1")
	if err == nil {
		t.Fatal("expected error for http scheme")
	}
	if !strings.Contains(err.Error(), "ws://") {
		t.Errorf("err = %v, want scheme error", err)
	}
}

func TestBuildURLPreservesExistingModel(t *testing.T) {
	u, err := BuildURL("wss://api.openai.com/v1/realtime?model=existing", "newmodel")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(u, "model=existing") {
		t.Errorf("u = %s, want existing model preserved", u)
	}
	if strings.Contains(u, "model=newmodel") {
		t.Errorf("u = %s, should not override existing model", u)
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
	if agent.cfg.Voice != DefaultVoice {
		t.Errorf("Voice = %s, want %s", agent.cfg.Voice, DefaultVoice)
	}
	if agent.cfg.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %s, want %s", agent.cfg.BaseURL, DefaultBaseURL)
	}
	if agent.cfg.DialTimeoutMs != DefaultDialMs {
		t.Errorf("DialTimeoutMs = %d, want %d", agent.cfg.DialTimeoutMs, DefaultDialMs)
	}
}

func TestNewSystemPromptMerge(t *testing.T) {
	a, err := New(
		map[string]any{"apiKey": "k", "instructions": "Be terse."},
		base.Options{SystemPrompt: "Be safe.", OnEvent: func(base.Event) {}},
	)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	agent := a.(*Agent)
	if !strings.Contains(agent.cfg.Instructions, "Be terse.") {
		t.Errorf("Instructions = %q, want cfg instructions", agent.cfg.Instructions)
	}
	if !strings.Contains(agent.cfg.Instructions, "Be safe.") {
		t.Errorf("Instructions = %q, want opts.SystemPrompt", agent.cfg.Instructions)
	}
}

func TestNewVoiceOverride(t *testing.T) {
	a, err := New(
		map[string]any{"apiKey": "k", "voice": "echo"},
		base.Options{Voice: "shimmer", OnEvent: func(base.Event) {}},
	)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	agent := a.(*Agent)
	if agent.cfg.Voice != "shimmer" {
		t.Errorf("Voice = %s, want shimmer (opts override)", agent.cfg.Voice)
	}
}

func TestPushAudioAfterClose(t *testing.T) {
	a, err := New(map[string]any{"apiKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	agent := a.(*Agent)
	// Simulate closed state without dialing.
	agent.ba.SetRootContext(nil, func() {})
	// Manually mark closed.
	agent.ba.Close()
	if err := agent.PushAudio([]byte{1, 2, 3}); err == nil {
		t.Error("PushAudio after Close should error")
	}
}

func TestCommitInputAudioAfterClose(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	agent.ba.Close()
	if err := agent.CommitInputAudio(); err == nil {
		t.Error("CommitInputAudio after Close should error")
	}
}

func TestCancelAfterClose(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	agent.ba.Close()
	if err := agent.Cancel(); err == nil {
		t.Error("Cancel after Close should error")
	}
}

func TestUpdateInstructionsEmpty(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	// Empty instructions should be a no-op (no error).
	if err := agent.UpdateInstructions(""); err != nil {
		t.Errorf("UpdateInstructions('') = %v, want nil", err)
	}
	if err := agent.UpdateInstructions("   "); err != nil {
		t.Errorf("UpdateInstructions('   ') = %v, want nil", err)
	}
}

func TestBuildSessionDefaults(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	sess := agent.buildSession()
	mods, _ := sess["modalities"].([]string)
	if len(mods) != 2 || mods[0] != "text" || mods[1] != "audio" {
		t.Errorf("modalities = %v, want [text audio]", mods)
	}
	if sess["input_audio_format"] != "pcm16" {
		t.Errorf("input_audio_format = %v, want pcm16", sess["input_audio_format"])
	}
	if sess["output_audio_format"] != "pcm16" {
		t.Errorf("output_audio_format = %v, want pcm16", sess["output_audio_format"])
	}
	td, _ := sess["turn_detection"].(map[string]any)
	if td == nil {
		t.Error("expected server_vad turn_detection by default")
	}
	if td["type"] != "server_vad" {
		t.Errorf("turn_detection.type = %v, want server_vad", td["type"])
	}
}

func TestBuildSessionDisableServerVAD(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{
		DisableServerVAD: true,
		OnEvent:          func(base.Event) {},
	})
	agent := a.(*Agent)
	sess := agent.buildSession()
	if sess["turn_detection"] != nil {
		t.Errorf("turn_detection = %v, want nil when VAD disabled", sess["turn_detection"])
	}
}

func TestBuildSessionTemperatureClamped(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{
		Temperature: 2.0,
		OnEvent:     func(base.Event) {},
	})
	agent := a.(*Agent)
	sess := agent.buildSession()
	temp, _ := sess["temperature"].(float64)
	if temp != 1.2 {
		t.Errorf("temperature = %v, want 1.2 (clamped)", temp)
	}
}

func TestBuildSessionTemperatureLowClamped(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{
		Temperature: 0.1,
		OnEvent:     func(base.Event) {},
	})
	agent := a.(*Agent)
	sess := agent.buildSession()
	temp, _ := sess["temperature"].(float64)
	if temp != 0.6 {
		t.Errorf("temperature = %v, want 0.6 (clamped low)", temp)
	}
}

func TestDispatchSessionCreated(t *testing.T) {
	var mu sync.Mutex
	var got []base.EventType
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{
		OnEvent: func(ev base.Event) {
			mu.Lock()
			got = append(got, ev.Type)
			mu.Unlock()
		},
	})
	agent := a.(*Agent)
	agent.dispatch([]byte(`{"type":"session.created"}`))
	mu.Lock()
	defer mu.Unlock()
	if len(got) == 0 || got[0] != base.EventSessionOpen {
		t.Errorf("got = %v, want base.EventSessionOpen", got)
	}
}

func TestDispatchErrorNonFatal(t *testing.T) {
	var mu sync.Mutex
	var got []base.Event
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{
		OnEvent: func(ev base.Event) {
			mu.Lock()
			got = append(got, ev)
			mu.Unlock()
		},
	})
	agent := a.(*Agent)
	agent.dispatch([]byte(`{"type":"error","error":{"message":"none active response"}}`))
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Fatal {
		t.Error("expected non-fatal for 'none active response'")
	}
}

func TestDispatchErrorFatal(t *testing.T) {
	var mu sync.Mutex
	var got []base.Event
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{
		OnEvent: func(ev base.Event) {
			mu.Lock()
			got = append(got, ev)
			mu.Unlock()
		},
	})
	agent := a.(*Agent)
	agent.dispatch([]byte(`{"type":"error","error":{"message":"rate limit exceeded"}}`))
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if !got[0].Fatal {
		t.Error("expected fatal for rate limit")
	}
}

func TestDispatchBadJSON(t *testing.T) {
	var mu sync.Mutex
	var got []base.Event
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{
		OnEvent: func(ev base.Event) {
			mu.Lock()
			got = append(got, ev)
			mu.Unlock()
		},
	})
	agent := a.(*Agent)
	agent.dispatch([]byte(`{not json`))
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 || got[0].Type != base.EventError {
		t.Errorf("got = %v, want base.EventError", got)
	}
}

func TestDispatchAudioDelta(t *testing.T) {
	var mu sync.Mutex
	var got []base.Event
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{
		OnEvent: func(ev base.Event) {
			mu.Lock()
			got = append(got, ev)
			mu.Unlock()
		},
	})
	agent := a.(*Agent)
	// "hello" in base64
	agent.dispatch([]byte(`{"type":"response.audio.delta","delta":"aGVsbG8="}`))
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 || got[0].Type != base.EventAssistantAudio {
		t.Errorf("got = %v, want base.EventAssistantAudio", got)
	}
	if string(got[0].AudioPC) != "hello" {
		t.Errorf("AudioPC = %q, want hello", string(got[0].AudioPC))
	}
}

func TestDispatchBadBase64Audio(t *testing.T) {
	var mu sync.Mutex
	var got []base.Event
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{
		OnEvent: func(ev base.Event) {
			mu.Lock()
			got = append(got, ev)
			mu.Unlock()
		},
	})
	agent := a.(*Agent)
	agent.dispatch([]byte(`{"type":"response.audio.delta","delta":"!!!notbase64!!!"}`))
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 || got[0].Type != base.EventError {
		t.Errorf("got = %v, want base.EventError for bad base64", got)
	}
}
