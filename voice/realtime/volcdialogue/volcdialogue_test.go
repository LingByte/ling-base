package realtime

import (
	"strings"
	"testing"

	base "github.com/LingByte/ling-base/voice/realtime"
)

func TestVolcNewMissingAppID(t *testing.T) {
	_, err := New(map[string]any{"accessKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	if err == nil {
		t.Fatal("expected error for missing appId")
	}
	if !strings.Contains(err.Error(), "appId") {
		t.Errorf("err = %v, want appId error", err)
	}
}

func TestVolcNewMissingAccessKey(t *testing.T) {
	_, err := New(map[string]any{"appId": "a"}, base.Options{OnEvent: func(base.Event) {}})
	if err == nil {
		t.Fatal("expected error for missing accessKey")
	}
}

func TestVolcNewDefaults(t *testing.T) {
	a, err := New(map[string]any{"appId": "a", "accessKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	agent := a.(*Agent)
	if agent.cfg.AppKey != DefaultAppKey {
		t.Errorf("AppKey = %s, want %s", agent.cfg.AppKey, DefaultAppKey)
	}
	if agent.cfg.ResourceID != DefaultResourceID {
		t.Errorf("ResourceID = %s, want %s", agent.cfg.ResourceID, DefaultResourceID)
	}
	if agent.cfg.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %s, want %s", agent.cfg.BaseURL, DefaultBaseURL)
	}
	if agent.cfg.Model != DefaultModelO {
		t.Errorf("Model = %s, want %s", agent.cfg.Model, DefaultModelO)
	}
	if agent.cfg.Speaker != DefaultSpeaker {
		t.Errorf("Speaker = %s, want %s", agent.cfg.Speaker, DefaultSpeaker)
	}
	if agent.cfg.DialTimeoutMs != DefaultDialMs {
		t.Errorf("DialTimeoutMs = %d, want %d", agent.cfg.DialTimeoutMs, DefaultDialMs)
	}
	if agent.sessionID == "" {
		t.Error("sessionID should be auto-generated")
	}
}

func TestVolcNewSystemRoleMerge(t *testing.T) {
	a, err := New(
		map[string]any{"appId": "a", "accessKey": "k", "systemRole": "Be terse."},
		base.Options{SystemPrompt: "Be safe.", OnEvent: func(base.Event) {}},
	)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	agent := a.(*Agent)
	if !strings.Contains(agent.cfg.SystemRole, "Be terse.") {
		t.Errorf("SystemRole = %q, want cfg systemRole", agent.cfg.SystemRole)
	}
	if !strings.Contains(agent.cfg.SystemRole, "Be safe.") {
		t.Errorf("SystemRole = %q, want opts.SystemPrompt", agent.cfg.SystemRole)
	}
}

func TestVolcNewVoiceOverride(t *testing.T) {
	a, _ := New(map[string]any{"appId": "a", "accessKey": "k", "speaker": "default"}, base.Options{
		Voice:   "custom",
		OnEvent: func(base.Event) {},
	})
	agent := a.(*Agent)
	if agent.cfg.Speaker != "custom" {
		t.Errorf("Speaker = %s, want custom (opts.Voice override)", agent.cfg.Speaker)
	}
}

func TestVolcBuildStartSessionDefault(t *testing.T) {
	a, _ := New(map[string]any{"appId": "a", "accessKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	p := agent.buildStartSession()
	if p.ASR.Format != "pcm" {
		t.Errorf("ASR.Format = %s, want pcm", p.ASR.Format)
	}
	if p.ASR.Rate != 16000 {
		t.Errorf("ASR.Rate = %d, want 16000", p.ASR.Rate)
	}
	if p.TTS.Speaker != DefaultSpeaker {
		t.Errorf("TTS.Speaker = %s, want %s", p.TTS.Speaker, DefaultSpeaker)
	}
	if p.TTS.AudioConfig.SampleRate != 24000 {
		t.Errorf("TTS.AudioConfig.SampleRate = %d, want 24000", p.TTS.AudioConfig.SampleRate)
	}
	if p.Dialog.BotName != "豆包" {
		t.Errorf("Dialog.BotName = %s, want 豆包", p.Dialog.BotName)
	}
	if p.Dialog.Extra["model"] != DefaultModelO {
		t.Errorf("Dialog.Extra[model] = %v, want %s", p.Dialog.Extra["model"], DefaultModelO)
	}
}

func TestVolcBuildStartSessionCharacterManifest(t *testing.T) {
	a, _ := New(map[string]any{
		"appId": "a", "accessKey": "k",
		"characterManifest": "manifest-data",
	}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	p := agent.buildStartSession()
	if p.Dialog.CharacterManifest != "manifest-data" {
		t.Errorf("CharacterManifest = %s, want manifest-data", p.Dialog.CharacterManifest)
	}
	if p.Dialog.BotName != "" {
		t.Errorf("BotName = %s, want empty when using character_manifest", p.Dialog.BotName)
	}
}

func TestVolcBuildStartSessionCustomOutputRate(t *testing.T) {
	a, _ := New(map[string]any{"appId": "a", "accessKey": "k"}, base.Options{
		OutputSampleRate: 48000,
		OnEvent:          func(base.Event) {},
	})
	agent := a.(*Agent)
	p := agent.buildStartSession()
	if p.TTS.AudioConfig.SampleRate != 48000 {
		t.Errorf("SampleRate = %d, want 48000", p.TTS.AudioConfig.SampleRate)
	}
}

func TestVolcPushAudioAfterClose(t *testing.T) {
	a, _ := New(map[string]any{"appId": "a", "accessKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	agent.closed.Store(true)
	if err := agent.PushAudio([]byte{1, 2}); err == nil {
		t.Error("PushAudio after close should error")
	}
}

func TestVolcPushAudioEmpty(t *testing.T) {
	a, _ := New(map[string]any{"appId": "a", "accessKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	// Empty PCM should be a no-op (no error even without a connection).
	if err := agent.PushAudio(nil); err != nil {
		t.Errorf("PushAudio(nil) = %v, want nil", err)
	}
}

func TestVolcCommitInputAudioNoop(t *testing.T) {
	a, _ := New(map[string]any{"appId": "a", "accessKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	if err := agent.CommitInputAudio(); err != nil {
		t.Errorf("CommitInputAudio = %v, want nil (noop)", err)
	}
}

func TestVolcCancelNoop(t *testing.T) {
	a, _ := New(map[string]any{"appId": "a", "accessKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	if err := agent.Cancel(); err != nil {
		t.Errorf("Cancel = %v, want nil (noop)", err)
	}
}

func TestVolcUpdateInstructionsEmpty(t *testing.T) {
	a, _ := New(map[string]any{"appId": "a", "accessKey": "k"}, base.Options{OnEvent: func(base.Event) {}})
	agent := a.(*Agent)
	if err := agent.UpdateInstructions(""); err != nil {
		t.Errorf("UpdateInstructions('') = %v, want nil", err)
	}
}
