// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package qcloud

import (
	"strings"
	"testing"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

func TestNewCredential_EmptySecretID(t *testing.T) {
	_, err := newCredential(Config{SecretKey: "sk"})
	if err == nil {
		t.Fatal("expected error for empty SecretID")
	}
	if !strings.Contains(err.Error(), "SecretID and SecretKey") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewCredential_EmptySecretKey(t *testing.T) {
	_, err := newCredential(Config{SecretID: "sid"})
	if err == nil {
		t.Fatal("expected error for empty SecretKey")
	}
	if !strings.Contains(err.Error(), "SecretID and SecretKey") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewCredential_EmptyBoth(t *testing.T) {
	_, err := newCredential(Config{})
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}
}

func TestNewCredential_Valid(t *testing.T) {
	cred, err := newCredential(Config{SecretID: "sid", SecretKey: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred == nil {
		t.Fatal("credential should be initialized")
	}
}

func TestNewProfile_SetsEndpoint(t *testing.T) {
	pf := newProfile("tms.tencentcloudapi.com")
	if pf == nil {
		t.Fatal("profile should not be nil")
	}
	if pf.HttpProfile == nil {
		t.Fatal("HttpProfile should not be nil")
	}
	if pf.HttpProfile.Endpoint != "tms.tencentcloudapi.com" {
		t.Errorf("endpoint = %q, want tms.tencentcloudapi.com", pf.HttpProfile.Endpoint)
	}
}

func TestNewProfile_EmptyEndpoint(t *testing.T) {
	pf := newProfile("")
	if pf == nil {
		t.Fatal("profile should not be nil")
	}
	if pf.HttpProfile.Endpoint != "" {
		t.Errorf("endpoint = %q, want empty", pf.HttpProfile.Endpoint)
	}
}

func TestResolveRegion_Default(t *testing.T) {
	if got := resolveRegion(""); got != defaultRegion {
		t.Errorf("resolveRegion(\"\") = %q, want %q", got, defaultRegion)
	}
}

func TestResolveRegion_Custom(t *testing.T) {
	if got := resolveRegion("ap-shanghai"); got != "ap-shanghai" {
		t.Errorf("resolveRegion(\"ap-shanghai\") = %q, want ap-shanghai", got)
	}
}

func TestResolveBizType_Default(t *testing.T) {
	if got := resolveBizType(""); got != "default" {
		t.Errorf("resolveBizType(\"\") = %q, want default", got)
	}
}

func TestResolveBizType_Custom(t *testing.T) {
	if got := resolveBizType("custom_biz"); got != "custom_biz" {
		t.Errorf("resolveBizType(\"custom_biz\") = %q, want custom_biz", got)
	}
}

func TestDefaultRegion_Constant(t *testing.T) {
	if defaultRegion != "ap-guangzhou" {
		t.Errorf("defaultRegion = %q, want ap-guangzhou", defaultRegion)
	}
}

func TestNewProfile_ReturnsClientProfile(t *testing.T) {
	pf := newProfile("ams.tencentcloudapi.com")
	if _, ok := interface{}(pf).(*profile.ClientProfile); !ok {
		t.Errorf("newProfile should return *profile.ClientProfile")
	}
}
