// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package registry_test

import (
	"context"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/registry"
)

func TestMemoryRegistry_RegisterAndDiscover(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	defer reg.Close()

	ctx := context.Background()

	inst := registry.Instance{
		ServiceName: "order-service",
		ID:          "order-1",
		Host:        "10.0.0.5",
		Port:        8080,
		Tags:        []string{"v1", "primary"},
		HealthPath:  "/health",
	}

	if err := reg.Register(ctx, inst); err != nil {
		t.Fatalf("Register: %v", err)
	}

	instances, err := reg.Discover(ctx, "order-service")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("Discover count = %d, want 1", len(instances))
	}
	if instances[0].ID != "order-1" {
		t.Errorf("ID = %q, want order-1", instances[0].ID)
	}
	if instances[0].ServiceName != "order-service" {
		t.Errorf("ServiceName = %q", instances[0].ServiceName)
	}
	if instances[0].Host != "10.0.0.5" {
		t.Errorf("Host = %q", instances[0].Host)
	}
	if instances[0].Port != 8080 {
		t.Errorf("Port = %d, want 8080", instances[0].Port)
	}
}

func TestMemoryRegistry_MultipleInstances(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	defer reg.Close()
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		inst := registry.Instance{
			ServiceName: "user-service",
			ID:          "user-" + itoa(i),
			Host:        "10.0.0." + itoa(i),
			Port:        8080 + i,
		}
		if err := reg.Register(ctx, inst); err != nil {
			t.Fatalf("Register %d: %v", i, err)
		}
	}

	instances, err := reg.Discover(ctx, "user-service")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(instances) != 3 {
		t.Errorf("Discover count = %d, want 3", len(instances))
	}
}

func TestMemoryRegistry_Deregister(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	defer reg.Close()
	ctx := context.Background()

	inst := registry.Instance{
		ServiceName: "test-service",
		ID:          "test-1",
		Host:        "127.0.0.1",
		Port:        9090,
	}
	if err := reg.Register(ctx, inst); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := reg.Deregister(ctx, "test-1"); err != nil {
		t.Fatalf("Deregister: %v", err)
	}

	_, err := reg.Discover(ctx, "test-service")
	if err != registry.ErrNotFound {
		t.Errorf("Discover after deregister: err = %v, want ErrNotFound", err)
	}
}

func TestMemoryRegistry_Deregister_NotRegistered(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	defer reg.Close()
	ctx := context.Background()

	err := reg.Deregister(ctx, "nonexistent")
	if err != registry.ErrNotRegistered {
		t.Errorf("Deregister nonexistent: err = %v, want ErrNotRegistered", err)
	}
}

func TestMemoryRegistry_Discover_NotFound(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	defer reg.Close()
	ctx := context.Background()

	_, err := reg.Discover(ctx, "nonexistent-service")
	if err != registry.ErrNotFound {
		t.Errorf("Discover: err = %v, want ErrNotFound", err)
	}
}

func TestMemoryRegistry_AutoID(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	defer reg.Close()
	ctx := context.Background()

	inst := registry.Instance{
		ServiceName: "auto-id-service",
		Host:        "10.0.0.1",
		Port:        7070,
		// ID intentionally left empty
	}
	if err := reg.Register(ctx, inst); err != nil {
		t.Fatalf("Register: %v", err)
	}

	instances, err := reg.Discover(ctx, "auto-id-service")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("Discover count = %d", len(instances))
	}
	if instances[0].ID == "" {
		t.Error("Auto-generated ID should not be empty")
	}
}

func TestMemoryRegistry_Watch(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	defer reg.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := reg.Watch(ctx, "watch-service")
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}

	// Register an instance — should trigger a watch event.
	inst := registry.Instance{
		ServiceName: "watch-service",
		ID:          "watch-1",
		Host:        "10.0.0.1",
		Port:        6060,
	}
	if err := reg.Register(ctx, inst); err != nil {
		t.Fatalf("Register: %v", err)
	}

	select {
	case instances := <-ch:
		if len(instances) != 1 {
			t.Errorf("Watch event: count = %d, want 1", len(instances))
		}
		if len(instances) > 0 && instances[0].ID != "watch-1" {
			t.Errorf("Watch event: ID = %q", instances[0].ID)
		}
	case <-time.After(2 * time.Second):
		t.Error("Watch did not receive event")
	}
}

func TestMemoryRegistry_Close(t *testing.T) {
	reg := registry.NewMemoryRegistry()

	ctx := context.Background()
	inst := registry.Instance{
		ServiceName: "close-service",
		ID:          "close-1",
		Host:        "127.0.0.1",
		Port:        5050,
	}
	_ = reg.Register(ctx, inst)

	if err := reg.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Operations after close should fail.
	err := reg.Register(ctx, inst)
	if err != registry.ErrClosed {
		t.Errorf("Register after close: err = %v, want ErrClosed", err)
	}

	_, err = reg.Discover(ctx, "close-service")
	if err != registry.ErrClosed {
		t.Errorf("Discover after close: err = %v, want ErrClosed", err)
	}
}

func TestInstance_Address(t *testing.T) {
	inst := registry.Instance{
		Host: "10.0.0.1",
		Port: 8080,
	}
	if inst.Address() != "10.0.0.1:8080" {
		t.Errorf("Address = %q, want 10.0.0.1:8080", inst.Address())
	}
}

func TestRegistry_Errors(t *testing.T) {
	if registry.ErrNotFound == nil {
		t.Error("ErrNotFound should not be nil")
	}
	if registry.ErrAlreadyRegistered == nil {
		t.Error("ErrAlreadyRegistered should not be nil")
	}
	if registry.ErrNotRegistered == nil {
		t.Error("ErrNotRegistered should not be nil")
	}
	if registry.ErrClosed == nil {
		t.Error("ErrClosed should not be nil")
	}
}

// itoa is a local helper to avoid importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
