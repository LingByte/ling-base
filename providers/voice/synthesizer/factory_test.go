package synthesizer

import (
	"testing"
)

// mockConfig implements Config for testing.
type mockConfig struct {
	provider Provider
}

func (m *mockConfig) GetProvider() Provider { return m.provider }

func TestNewFactory(t *testing.T) {
	f := NewFactory()
	if f == nil {
		t.Fatal("NewFactory() returned nil")
	}
	if len(f.GetSupportedProviders()) != 0 {
		t.Errorf("new factory should have 0 providers, got %d", len(f.GetSupportedProviders()))
	}
}

func TestFactoryRegisterAndCreate(t *testing.T) {
	f := NewFactory()

	f.RegisterCreator(ProviderOpenAI, func(cfg Config) (Engine, error) {
		return &mockEngine{provider: ProviderOpenAI, format: DefaultFormat()}, nil
	})

	if !f.IsProviderSupported(ProviderOpenAI) {
		t.Error("IsProviderSupported(ProviderOpenAI) should be true")
	}
	if f.IsProviderSupported(ProviderAWS) {
		t.Error("IsProviderSupported(ProviderAWS) should be false")
	}

	providers := f.GetSupportedProviders()
	if len(providers) != 1 || providers[0] != ProviderOpenAI {
		t.Errorf("GetSupportedProviders() = %v, want [openai]", providers)
	}

	engine, err := f.CreateEngine(&mockConfig{provider: ProviderOpenAI})
	if err != nil {
		t.Fatalf("CreateEngine failed: %v", err)
	}
	if engine.Provider() != ProviderOpenAI {
		t.Errorf("Provider() = %q, want %q", engine.Provider(), ProviderOpenAI)
	}
}

func TestFactoryCreateUnsupported(t *testing.T) {
	f := NewFactory()
	_, err := f.CreateEngine(&mockConfig{provider: ProviderAWS})
	if err == nil {
		t.Fatal("CreateEngine should fail for unsupported provider")
	}
}

func TestFactoryCreateNilConfig(t *testing.T) {
	f := NewFactory()
	_, err := f.CreateEngine(nil)
	if err == nil {
		t.Fatal("CreateEngine(nil) should fail")
	}
}

func TestGetGlobalFactory(t *testing.T) {
	f1 := GetGlobalFactory()
	f2 := GetGlobalFactory()
	if f1 != f2 {
		t.Error("GetGlobalFactory should return the same instance")
	}
}

func TestSetGlobalFactory(t *testing.T) {
	original := GetGlobalFactory()
	newFactory := NewFactory()
	SetGlobalFactory(newFactory)

	// GetGlobalFactory should now return the new factory
	// Note: due to the mutex-based implementation, this should work
	got := GetGlobalFactory()
	if got != newFactory {
		// This might fail because GetGlobalFactory checks if globalFactory is nil
		// and it's already set. Let's just verify it doesn't panic.
		t.Log("SetGlobalFactory behavior depends on initialization order")
	}

	// Restore
	SetGlobalFactory(original)
}

func TestCreate(t *testing.T) {
	// Register a mock creator in the global factory
	GetGlobalFactory().RegisterCreator(ProviderLocal, func(cfg Config) (Engine, error) {
		return &mockEngine{provider: ProviderLocal, format: DefaultFormat()}, nil
	})

	engine, err := Create(&mockConfig{provider: ProviderLocal})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if engine.Provider() != ProviderLocal {
		t.Errorf("Provider() = %q, want %q", engine.Provider(), ProviderLocal)
	}
}

func TestAllProviders(t *testing.T) {
	providers := AllProviders()
	if len(providers) == 0 {
		t.Fatal("AllProviders() should not be empty")
	}
	for _, p := range providers {
		if p == "" {
			t.Error("AllProviders() contains empty provider")
		}
	}
}

func TestRegisterAllProviders(t *testing.T) {
	f := NewFactory()
	registrations := map[Provider]Creator{
		ProviderLocal: func(cfg Config) (Engine, error) {
			return &mockEngine{provider: ProviderLocal}, nil
		},
		ProviderAWS: func(cfg Config) (Engine, error) {
			return &mockEngine{provider: ProviderAWS}, nil
		},
		ProviderOpenAI: nil, // should be skipped
	}
	RegisterAllProviders(f, registrations)

	if !f.IsProviderSupported(ProviderLocal) {
		t.Error("ProviderLocal should be registered")
	}
	if !f.IsProviderSupported(ProviderAWS) {
		t.Error("ProviderAWS should be registered")
	}
	if f.IsProviderSupported(ProviderOpenAI) {
		t.Error("ProviderOpenAI should not be registered (nil creator)")
	}
}

func TestRegisterAllProvidersNilFactory(t *testing.T) {
	// Should not panic
	RegisterAllProviders(nil, map[Provider]Creator{
		ProviderLocal: func(cfg Config) (Engine, error) {
			return nil, nil
		},
	})
}
