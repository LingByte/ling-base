package realtime

import (
	"strings"
	"testing"
)

func withDummyProvider(t *testing.T, slug string) func() {
	t.Helper()
	Register(func(cfg map[string]any, opts Options) (Agent, error) {
		return nil, nil
	}, slug)
	return func() {
		Unregister(slug)
	}
}

func TestNewAgentFromCredentialEmpty(t *testing.T) {
	_, err := NewAgentFromCredential(nil, Options{OnEvent: func(Event) {}})
	if err == nil {
		t.Fatal("expected error for empty config")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("err = %v, want empty", err)
	}
}

func TestNewAgentFromCredentialMissingProvider(t *testing.T) {
	_, err := NewAgentFromCredential(map[string]any{"apiKey": "x"}, Options{OnEvent: func(Event) {}})
	if err == nil {
		t.Fatal("expected error for missing provider")
	}
	if !strings.Contains(err.Error(), "missing provider") {
		t.Errorf("err = %v, want missing provider", err)
	}
}

func TestNewAgentFromCredentialUnknownProvider(t *testing.T) {
	_, err := NewAgentFromCredential(map[string]any{"provider": "nope"}, Options{OnEvent: func(Event) {}})
	if err == nil {
		t.Fatal("expected ErrUnknownProvider")
	}
	if _, ok := err.(*ErrUnknownProvider); !ok {
		t.Errorf("err type = %T, want *ErrUnknownProvider", err)
	}
}

func TestNewAgentFromCredentialMissingOnEvent(t *testing.T) {
	cleanup := withDummyProvider(t, "dummy_test")
	defer cleanup()
	_, err := NewAgentFromCredential(map[string]any{"provider": "dummy_test"}, Options{})
	if err == nil {
		t.Fatal("expected error for missing OnEvent")
	}
	if !strings.Contains(err.Error(), "OnEvent") {
		t.Errorf("err = %v, want OnEvent", err)
	}
}

func TestNewAgentFromCredentialDefaultSampleRates(t *testing.T) {
	cleanup := withDummyProvider(t, "dummy_test")
	defer cleanup()
	var captured Options
	Register(func(cfg map[string]any, opts Options) (Agent, error) {
		captured = opts
		return nil, nil
	}, "dummy_test")
	defer Unregister("dummy_test")

	_, _ = NewAgentFromCredential(map[string]any{"provider": "dummy_test"}, Options{OnEvent: func(Event) {}})
	if captured.InputSampleRate != 16000 {
		t.Errorf("InputSampleRate = %d, want 16000", captured.InputSampleRate)
	}
	if captured.OutputSampleRate != 24000 {
		t.Errorf("OutputSampleRate = %d, want 24000", captured.OutputSampleRate)
	}
}

func TestNewAgentFromCredentialPreservesSampleRates(t *testing.T) {
	cleanup := withDummyProvider(t, "dummy_test")
	defer cleanup()
	var captured Options
	Register(func(cfg map[string]any, opts Options) (Agent, error) {
		captured = opts
		return nil, nil
	}, "dummy_test")
	defer Unregister("dummy_test")

	_, _ = NewAgentFromCredential(map[string]any{"provider": "dummy_test"}, Options{
		OnEvent:           func(Event) {},
		InputSampleRate:   8000,
		OutputSampleRate:  48000,
	})
	if captured.InputSampleRate != 8000 {
		t.Errorf("InputSampleRate = %d, want 8000", captured.InputSampleRate)
	}
	if captured.OutputSampleRate != 48000 {
		t.Errorf("OutputSampleRate = %d, want 48000", captured.OutputSampleRate)
	}
}

func TestNewAgentFromCredentialCaseInsensitive(t *testing.T) {
	cleanup := withDummyProvider(t, "dummy_test")
	defer cleanup()
	_, err := NewAgentFromCredential(map[string]any{"provider": " DUMMY_TEST "}, Options{OnEvent: func(Event) {}})
	if err != nil {
		t.Errorf("err = %v, want nil for case-insensitive match", err)
	}
}

func TestMustCreatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustCreate(map[string]any{"provider": "nope"}, Options{OnEvent: func(Event) {}})
}

func TestRegisteredProvidersSorted(t *testing.T) {
	cleanup1 := withDummyProvider(t, "zzz_last")
	cleanup2 := withDummyProvider(t, "aaa_first")
	defer cleanup1()
	defer cleanup2()
	providers := RegisteredProviders()
	if len(providers) < 2 {
		t.Fatalf("len = %d, want >= 2", len(providers))
	}
	for i := 1; i < len(providers); i++ {
		if providers[i-1] > providers[i] {
			t.Errorf("providers not sorted: %v", providers)
			break
		}
	}
}

func TestAllProvidersMatchesRegistered(t *testing.T) {
	if len(AllProviders()) != len(RegisteredProviders()) {
		t.Error("AllProviders and RegisteredProviders disagree")
	}
}

func TestLookupNil(t *testing.T) {
	if p := Lookup("nonexistent_provider_xyz"); p != nil {
		t.Errorf("Lookup for unknown returned %v, want nil", p)
	}
}

func TestIsProviderRegistered(t *testing.T) {
	cleanup := withDummyProvider(t, "dummy_test")
	defer cleanup()
	if !IsProviderRegistered("dummy_test") {
		t.Error("expected dummy_test to be registered")
	}
	if IsProviderRegistered("not_registered") {
		t.Error("not_registered should not be registered")
	}
}

func TestErrUnknownProviderMessage(t *testing.T) {
	e := &ErrUnknownProvider{Provider: "foo"}
	if !strings.Contains(e.Error(), "foo") {
		t.Errorf("err = %v, want foo", e.Error())
	}
}
