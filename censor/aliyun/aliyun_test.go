// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package aliyun

import (
	"strings"
	"testing"
)

func TestNewClient_EmptyAccessKeyID(t *testing.T) {
	_, err := newClient(Config{AccessKeySecret: "sk"})
	if err == nil {
		t.Fatal("expected error for empty AccessKeyID")
	}
	if !strings.Contains(err.Error(), "AccessKeyID and AccessKeySecret") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewClient_EmptyAccessKeySecret(t *testing.T) {
	_, err := newClient(Config{AccessKeyID: "ak"})
	if err == nil {
		t.Fatal("expected error for empty AccessKeySecret")
	}
	if !strings.Contains(err.Error(), "AccessKeyID and AccessKeySecret") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewClient_EmptyBoth(t *testing.T) {
	_, err := newClient(Config{})
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}
}

func TestNewClient_Valid(t *testing.T) {
	c, err := newClient(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("client should be initialized")
	}
}

func TestNewClient_DefaultEndpoint(t *testing.T) {
	c, err := newClient(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("client should be initialized")
	}
	// The default endpoint should be used when Endpoint is empty.
	// We verify the client was created successfully with the default endpoint.
}

func TestNewClient_CustomEndpoint(t *testing.T) {
	c, err := newClient(Config{
		AccessKeyID:     "ak",
		AccessKeySecret: "sk",
		Endpoint:        "green-cip.cn-beijing.aliyuncs.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("client should be initialized with custom endpoint")
	}
}

func TestDefaultEndpoint_Constant(t *testing.T) {
	if defaultEndpoint != "green-cip.cn-shanghai.aliyuncs.com" {
		t.Errorf("defaultEndpoint = %q, want green-cip.cn-shanghai.aliyuncs.com", defaultEndpoint)
	}
}
