package synthesizer

import (
	"testing"
)

func TestDefaultCapabilities(t *testing.T) {
	caps := DefaultCapabilities()
	if caps.StreamingTTFB {
		t.Error("DefaultCapabilities().StreamingTTFB should be false")
	}
	if caps.SuggestedFirstMaxRunes != 12 {
		t.Errorf("SuggestedFirstMaxRunes = %d, want 12", caps.SuggestedFirstMaxRunes)
	}
}

func TestStreamingCapabilities(t *testing.T) {
	caps := StreamingCapabilities()
	if !caps.StreamingTTFB {
		t.Error("StreamingCapabilities().StreamingTTFB should be true")
	}
	if caps.SuggestedFirstMaxRunes != 24 {
		t.Errorf("SuggestedFirstMaxRunes = %d, want 24", caps.SuggestedFirstMaxRunes)
	}
}

// mockCapableEngine implements CapableEngine for testing.
type mockCapableEngine struct {
	mockEngine
	caps Capabilities
}

func (m *mockCapableEngine) Capabilities() Capabilities {
	return m.caps
}

func TestCapableEngineInterface(t *testing.T) {
	var e CapableEngine = &mockCapableEngine{
		mockEngine: mockEngine{provider: ProviderOpenAI, format: DefaultFormat()},
		caps:       StreamingCapabilities(),
	}

	if !e.Capabilities().StreamingTTFB {
		t.Error("Capabilities().StreamingTTFB should be true")
	}
	// Verify it also satisfies Engine
	if e.Provider() != ProviderOpenAI {
		t.Errorf("Provider() = %q, want %q", e.Provider(), ProviderOpenAI)
	}
}
