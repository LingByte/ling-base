package ocr

import (
	"context"
	"errors"
	"testing"
)

// fakeProvider is a test double for Provider.
type fakeProvider struct {
	name      string
	text      string
	err       error
	lastImage []byte
	lastOpts  *Options
}

func (f *fakeProvider) Name() string { return f.name }
func (f *fakeProvider) Recognize(ctx context.Context, imageBytes []byte, opts *Options) (string, error) {
	f.lastImage = imageBytes
	f.lastOpts = opts
	if f.err != nil {
		return "", f.err
	}
	return f.text, nil
}

func TestRegisterAndSetProvider(t *testing.T) {
	// Save and restore state
	defer func() {
		mu.Lock()
		active = nil
		registry = make(map[string]Provider)
		mu.Unlock()
	}()

	p := &fakeProvider{name: "test", text: "hello"}
	RegisterProvider("test", p)

	err := SetProviderByDriver("test")
	if err != nil {
		t.Fatalf("SetProviderByDriver: %v", err)
	}
	if GetProvider() == nil {
		t.Fatal("expected active provider")
	}
	if GetProvider().Name() != "test" {
		t.Errorf("got %s, want test", GetProvider().Name())
	}
}

func TestSetProviderByDriverNotFound(t *testing.T) {
	defer func() {
		mu.Lock()
		active = nil
		registry = make(map[string]Provider)
		mu.Unlock()
	}()

	err := SetProviderByDriver("nonexistent")
	if err == nil {
		t.Fatal("expected error for unregistered driver")
	}
}

func TestSetProviderDirect(t *testing.T) {
	defer func() {
		mu.Lock()
		active = nil
		registry = make(map[string]Provider)
		mu.Unlock()
	}()

	p := &fakeProvider{name: "direct", text: "world"}
	SetProvider(p)
	if GetProvider() == nil || GetProvider().Name() != "direct" {
		t.Fatal("SetProvider did not work")
	}

	SetProvider(nil)
	if GetProvider() != nil {
		t.Fatal("SetProvider(nil) should clear")
	}
}

func TestRegisteredDrivers(t *testing.T) {
	defer func() {
		mu.Lock()
		active = nil
		registry = make(map[string]Provider)
		mu.Unlock()
	}()

	RegisterProvider("a", &fakeProvider{name: "a"})
	RegisterProvider("b", &fakeProvider{name: "b"})
	drivers := RegisteredDrivers()
	if len(drivers) != 2 {
		t.Fatalf("got %d drivers, want 2", len(drivers))
	}
}

func TestRegisterNilProvider(t *testing.T) {
	defer func() {
		mu.Lock()
		active = nil
		registry = make(map[string]Provider)
		mu.Unlock()
	}()

	RegisterProvider("nil-test", nil)
	err := SetProviderByDriver("nil-test")
	if err == nil {
		t.Fatal("expected error for nil provider")
	}
}

func TestErrNoProvider(t *testing.T) {
	if !errors.Is(ErrNoProvider, ErrNoProvider) {
		t.Fatal("ErrNoProvider should match itself")
	}
}
