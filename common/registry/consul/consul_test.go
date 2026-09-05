// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package consul_test

import (
	"testing"

	"github.com/LingByte/ling-base/common/registry/consul"
)

func TestNew_MissingAddress(t *testing.T) {
	_, err := consul.New(consul.Config{})
	if err == nil {
		t.Error("New should fail without address")
	}
}

func TestNew_ValidConfig(t *testing.T) {
	c, err := consul.New(consul.Config{
		Address:                 "127.0.0.1:8500",
		Scheme:                  "http",
		DefaultHealthInterval:   0, // test default
		DefaultHealthTimeout:    0, // test default
		DeregisterCriticalAfter: 0, // test default
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c == nil {
		t.Fatal("New returned nil")
	}
	_ = c.Close()
}

func TestClose_Twice(t *testing.T) {
	c, err := consul.New(consul.Config{Address: "127.0.0.1:8500"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := c.Close(); err != nil {
		t.Errorf("First Close: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Errorf("Second Close: %v", err)
	}
}

func TestGetOutboundIP(t *testing.T) {
	// This may fail in CI without network access, so we just test
	// that it doesn't panic. We don't assert on the result.
	_, _ = consul.GetOutboundIP()
}
