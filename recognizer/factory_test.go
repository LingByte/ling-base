package recognizer

import (
	"testing"
)

// mockConfig implements TranscriberConfig for testing.
type mockConfig struct {
	vendor Vendor
}

func (m *mockConfig) GetVendor() Vendor { return m.vendor }

func TestNewFactory(t *testing.T) {
	f := NewFactory()
	if f == nil {
		t.Fatal("NewFactory() returned nil")
	}
	if len(f.GetSupportedVendors()) != 0 {
		t.Errorf("new factory should have 0 vendors, got %d", len(f.GetSupportedVendors()))
	}
}

func TestFactoryRegisterAndCreate(t *testing.T) {
	f := NewFactory()

	// Register a mock creator
	f.RegisterCreator(VendorLocal, func(cfg TranscriberConfig) (Engine, error) {
		return &mockEngine{vendor: "local"}, nil
	})

	if !f.IsVendorSupported(VendorLocal) {
		t.Error("IsVendorSupported(VendorLocal) should be true")
	}
	if f.IsVendorSupported(VendorAWS) {
		t.Error("IsVendorSupported(VendorAWS) should be false")
	}

	vendors := f.GetSupportedVendors()
	if len(vendors) != 1 || vendors[0] != VendorLocal {
		t.Errorf("GetSupportedVendors() = %v, want [local]", vendors)
	}

	engine, err := f.CreateTranscriber(&mockConfig{vendor: VendorLocal})
	if err != nil {
		t.Fatalf("CreateTranscriber failed: %v", err)
	}
	if engine.Vendor() != "local" {
		t.Errorf("Vendor() = %q, want %q", engine.Vendor(), "local")
	}
}

func TestFactoryCreateUnsupported(t *testing.T) {
	f := NewFactory()
	_, err := f.CreateTranscriber(&mockConfig{vendor: VendorAWS})
	if err == nil {
		t.Fatal("CreateTranscriber should fail for unsupported vendor")
	}
}

func TestFactoryCreateNilConfig(t *testing.T) {
	f := NewFactory()
	_, err := f.CreateTranscriber(nil)
	if err == nil {
		t.Fatal("CreateTranscriber(nil) should fail")
	}
}

func TestGetGlobalFactory(t *testing.T) {
	f1 := GetGlobalFactory()
	f2 := GetGlobalFactory()
	if f1 != f2 {
		t.Error("GetGlobalFactory should return the same instance")
	}
}

func TestGetVendor(t *testing.T) {
	tests := []struct {
		input string
		want  Vendor
	}{
		{"tencent", VendorQCloud},
		{"qcloud", VendorQCloud},
		{"aws", VendorAWS},
		{"volcengine", VendorVolcengine},
		{"local", VendorLocal},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := GetVendor(tt.input)
			if got != tt.want {
				t.Errorf("GetVendor(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestVendorString(t *testing.T) {
	if VendorString(VendorAWS) != "aws" {
		t.Errorf("VendorString(VendorAWS) = %q, want %q", VendorString(VendorAWS), "aws")
	}
}

func TestAllVendors(t *testing.T) {
	vendors := AllVendors()
	if len(vendors) == 0 {
		t.Fatal("AllVendors() should not be empty")
	}
	// Verify all entries are non-empty
	for _, v := range vendors {
		if v == "" {
			t.Error("AllVendors() contains empty vendor")
		}
	}
}

func TestRegisterAllVendors(t *testing.T) {
	f := NewFactory()
	registrations := map[Vendor]Creator{
		VendorLocal: func(cfg TranscriberConfig) (Engine, error) {
			return &mockEngine{vendor: "local"}, nil
		},
		VendorAWS: func(cfg TranscriberConfig) (Engine, error) {
			return &mockEngine{vendor: "aws"}, nil
		},
		VendorWhisper: nil, // should be skipped
	}
	RegisterAllVendors(f, registrations)

	if !f.IsVendorSupported(VendorLocal) {
		t.Error("VendorLocal should be registered")
	}
	if !f.IsVendorSupported(VendorAWS) {
		t.Error("VendorAWS should be registered")
	}
	if f.IsVendorSupported(VendorWhisper) {
		t.Error("VendorWhisper should not be registered (nil creator)")
	}
}

func TestRegisterAllVendorsNilFactory(t *testing.T) {
	// Should not panic
	RegisterAllVendors(nil, map[Vendor]Creator{
		VendorLocal: func(cfg TranscriberConfig) (Engine, error) {
			return nil, nil
		},
	})
}

// mockEngine implements Engine for testing.
type mockEngine struct {
	vendor string
}

func (m *mockEngine) Init(_ ResultFunc, _ ErrorFunc)      {}
func (m *mockEngine) Vendor() string                       { return m.vendor }
func (m *mockEngine) ConnAndReceive(_ string) error        { return nil }
func (m *mockEngine) Activity() bool                       { return false }
func (m *mockEngine) RestartClient()                       {}
func (m *mockEngine) SendAudioBytes(_ []byte) error        { return nil }
func (m *mockEngine) SendEnd() error                       { return nil }
func (m *mockEngine) StopConn() error                      { return nil }
