package realtime

import (
	"strings"
	"sync"
	"testing"

	base "github.com/LingByte/ling-base/realtime"
)

func TestAliyunBuildURLDefault(t *testing.T) {
	u, err := BuildURL("", "qwen-test")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(u, "model=qwen-test") {
		t.Errorf("u = %s, want model query", u)
	}
}

func TestAliyunBuildURLBadScheme(t *testing.T) {
	_, err := BuildURL("http://example.com", "m")
	if err == nil {
		t.Fatal("expected error for http scheme")
	}
}

func TestAliyunNewMissingAPIKey(t *testing.T) {
	_, err := New(map[string]any{}, base.Options{OnEvent: func(base.Event) {}})
	if err == nil {
		t.Fatal("expected error for missing apiKey")
	}
}

func TestAliyunNewDefaults(t *testing.T) {
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
	if agent.cfg.DialTimeoutMs != DefaultDialMs {
		t.Errorf("DialTimeoutMs = %d, want %d", agent.cfg.DialTimeoutMs, DefaultDialMs)
	}
}

func TestAliyunBuildSessionDefaults(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	sess := agent.buildSession()
	if sess["input_audio_format"] != "pcm" {
		t.Errorf("input_audio_format = %v, want pcm", sess["input_audio_format"])
	}
	if sess["output_audio_format"] != "pcm" {
		t.Errorf("output_audio_format = %v, want pcm", sess["output_audio_format"])
	}
	td, _ := sess["turn_detection"].(map[string]any)
	if td == nil {
		t.Fatal("expected turn_detection")
	}
	if td["threshold"] != 0.8 {
		t.Errorf("threshold = %v, want 0.8", td["threshold"])
	}
	if td["silence_duration_ms"] != 1000 {
		t.Errorf("silence_duration_ms = %v, want 1000", td["silence_duration_ms"])
	}
}

func TestAliyunBuildSessionDisableServerVAD(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{
		DisableServerVAD: true,
		OnEvent:          func(base.Event) {},
	})
	agent := a.(*Agent)
	sess := agent.buildSession()
	if sess["turn_detection"] != nil {
		t.Errorf("turn_detection = %v, want nil", sess["turn_detection"])
	}
}

func TestAliyunBuildSessionTemperatureClamped(t *testing.T) {
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

func TestAliyunDispatchSessionCreated(t *testing.T) {
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

func TestAliyunDispatchErrorNonFatal(t *testing.T) {
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
	agent.dispatch([]byte(`{"type":"error","error":{"message":"no active response"}}`))
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Fatal {
		t.Error("expected non-fatal for 'no active response'")
	}
}

func TestAliyunDispatchAudioDelta(t *testing.T) {
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

func TestAliyunDispatchFunctionCallDone(t *testing.T) {
	var mu sync.Mutex
	var got []base.Event
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{
		OnEvent: func(ev base.Event) {
			mu.Lock()
			got = append(got, ev)
			mu.Unlock()
		},
		Tools: []base.Tool{{Name: "get_weather", Description: "Get weather"}},
		ToolHandler: func(name string, args map[string]any) string {
			return "sunny"
		},
	})
	agent := a.(*Agent)
	// Function call arguments done.
	agent.dispatch([]byte(`{"type":"response.function_call_arguments.done","call_id":"call_1","name":"get_weather","arguments":"{\"location\":\"SF\"}"}`))
	// Response done.
	agent.dispatch([]byte(`{"type":"response.done"}`))
	mu.Lock()
	defer mu.Unlock()
	if len(got) == 0 {
		t.Fatal("no events emitted")
	}
	last := got[len(got)-1]
	if last.Type != base.EventAssistantTurnEnd {
		t.Errorf("last event = %v, want base.EventAssistantTurnEnd", last.Type)
	}
	if last.Text != "get_weather" {
		t.Errorf("last.Text = %q, want get_weather", last.Text)
	}
}

func TestAliyunDispatchBadJSON(t *testing.T) {
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

func TestAliyunPushAudioAfterClose(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	agent.ba.Close()
	if err := agent.PushAudio([]byte{1, 2}); err == nil {
		t.Error("PushAudio after Close should error")
	}
}

func TestAliyunUpdateInstructionsEmpty(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	if err := agent.UpdateInstructions(""); err != nil {
		t.Errorf("UpdateInstructions('') = %v, want nil", err)
	}
}

func TestAliyunCreateResponseAfterClose(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	agent.ba.Close()
	if err := agent.CreateResponse(""); err == nil {
		t.Error("CreateResponse after Close should error")
	}
}

func TestAliyunClearInputAudioAfterClose(t *testing.T) {
	a, _ := New(map[string]any{"apiKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	agent.ba.Close()
	if err := agent.ClearInputAudio(); err == nil {
		t.Error("ClearInputAudio after Close should error")
	}
}
