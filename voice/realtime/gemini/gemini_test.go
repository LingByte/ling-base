package realtime

import (
	"strings"
	"sync"
	"testing"

	base "github.com/LingByte/ling-base/voice/realtime"
)

func TestGeminiBuildURLDefault(t *testing.T) {
	u, err := BuildURL("", "test-key")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(u, "key=test-key") {
		t.Errorf("u = %s, want key=test-key", u)
	}
}

func TestGeminiBuildURLBadScheme(t *testing.T) {
	_, err := BuildURL("http://example.com", "k")
	if err == nil {
		t.Fatal("expected error for http scheme")
	}
}

func TestGeminiNewMissingAPIKey(t *testing.T) {
	_, err := New(map[string]any{}, base.Options{OnEvent: func(base.Event) {}})
	if err == nil {
		t.Fatal("expected error for missing apiKey")
	}
	if !strings.Contains(err.Error(), "apiKey") {
		t.Errorf("err = %v, want apiKey error", err)
	}
}

func TestGeminiNewDefaults(t *testing.T) {
	a, err := New(map[string]any{"apiKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
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
}

func TestGeminiBuildSetupDefaults(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	setup := agent.buildSetup()
	setupInner, _ := setup["setup"].(map[string]any)
	if setupInner == nil {
		t.Fatal("missing setup")
	}
	model, _ := setupInner["model"].(string)
	if !strings.HasPrefix(model, "models/") {
		t.Errorf("model = %s, want models/ prefix", model)
	}
	genCfg, _ := setupInner["generationConfig"].(map[string]any)
	mods, _ := genCfg["responseModalities"].([]string)
	if len(mods) != 1 || mods[0] != "AUDIO" {
		t.Errorf("responseModalities = %v, want [AUDIO]", mods)
	}
}

func TestGeminiBuildSetupDisableServerVAD(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{
		DisableServerVAD: true,
		OnEvent:          func(base.Event) {},
	})
	agent := a.(*Agent)
	setup := agent.buildSetup()
	setupInner, _ := setup["setup"].(map[string]any)
	rtCfg, _ := setupInner["realtimeInputConfig"].(map[string]any)
	if rtCfg == nil {
		t.Fatal("expected realtimeInputConfig when VAD disabled")
	}
	aad, _ := rtCfg["automaticActivityDetection"].(map[string]any)
	if aad == nil || aad["disabled"] != true {
		t.Errorf("automaticActivityDetection = %v, want disabled=true", aad)
	}
}

func TestGeminiBuildSetupSystemInstruction(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k", "instructions": "Be helpful"}, base.Options{
		OnEvent: func(base.Event) {},
	})
	agent := a.(*Agent)
	setup := agent.buildSetup()
	setupInner, _ := setup["setup"].(map[string]any)
	si, _ := setupInner["systemInstruction"].(map[string]any)
	if si == nil {
		t.Fatal("expected systemInstruction")
	}
	parts, _ := si["parts"].([]map[string]any)
	if len(parts) == 0 {
		t.Fatal("expected at least one part")
	}
	if !strings.Contains(parts[0]["text"].(string), "Be helpful") {
		t.Errorf("systemInstruction text = %v, want 'Be helpful'", parts[0]["text"])
	}
}

func TestGeminiDispatchSetupComplete(t *testing.T) {
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
	agent.dispatch([]byte(`{"setupComplete":{}}`))
	mu.Lock()
	defer mu.Unlock()
	if len(got) == 0 || got[0] != base.EventSessionOpen {
		t.Errorf("got = %v, want base.EventSessionOpen", got)
	}
}

func TestGeminiDispatchServerError(t *testing.T) {
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
	agent.dispatch([]byte(`{"error":{"code":400,"message":"bad request","status":"INVALID_ARGUMENT"}}`))
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 || got[0].Type != base.EventError || !got[0].Fatal {
		t.Errorf("got = %v, want fatal base.EventError", got)
	}
}

func TestGeminiDispatchServerContentAudio(t *testing.T) {
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
	agent.dispatch([]byte(`{"serverContent":{"modelTurn":{"parts":[{"inlineData":{"mimeType":"audio/pcm","data":"aGVsbG8="}}]},"turnComplete":true}}`))
	mu.Lock()
	defer mu.Unlock()
	if len(got) < 2 {
		t.Fatalf("got = %v, want at least 2 events (audio + turn end)", got)
	}
	if got[0].Type != base.EventAssistantAudio || string(got[0].AudioPC) != "hello" {
		t.Errorf("first event = %v, want audio 'hello'", got[0])
	}
	if got[len(got)-1].Type != base.EventAssistantTurnEnd {
		t.Errorf("last event = %v, want turn end", got[len(got)-1])
	}
}

func TestGeminiDispatchInterrupted(t *testing.T) {
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
	agent.dispatch([]byte(`{"serverContent":{"interrupted":true}}`))
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 || got[0] != base.EventUserSpeechStarted {
		t.Errorf("got = %v, want base.EventUserSpeechStarted", got)
	}
}

func TestGeminiDispatchBadJSON(t *testing.T) {
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

func TestGeminiPushAudioAfterClose(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	agent.ba.Close()
	if err := agent.PushAudio([]byte{1, 2}); err == nil {
		t.Error("PushAudio after Close should error")
	}
}

func TestGeminiCancelNoopWithServerVAD(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	// With server VAD enabled (default), Cancel is a no-op and should not error
	// even when closed (returns nil).
	_ = agent.Cancel()
}

func TestGeminiUpdateInstructionsEmpty(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	if err := agent.UpdateInstructions(""); err != nil {
		t.Errorf("UpdateInstructions('') = %v, want nil", err)
	}
}
