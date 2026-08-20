// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package geoip

import (
	"net"
	"testing"
)

func TestIsIP(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"192.168.1.1", true},
		{"::1", true},
		{"invalid", false},
		{"", false},
		{"256.0.0.1", false},
	}
	for _, tt := range tests {
		if got := IsIP(tt.input); got != tt.want {
			t.Errorf("IsIP(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseIP(t *testing.T) {
	if ParseIP("192.168.1.1") == nil {
		t.Fatal("ParseIP should return non-nil for valid IP")
	}
	if ParseIP("invalid") != nil {
		t.Fatal("ParseIP should return nil for invalid IP")
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"127.0.0.1", true},
		{"8.8.8.8", false},
	}
	for _, tt := range tests {
		ip := net.ParseIP(tt.input)
		if got := IsPrivateIP(ip); got != tt.want {
			t.Errorf("IsPrivateIP(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsInternalIP(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"192.168.1.1", true},
		{"8.8.8.8", false},
		{"invalid", false},
	}
	for _, tt := range tests {
		if got := IsInternalIP(tt.input); got != tt.want {
			t.Errorf("IsInternalIP(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsIpInCIDRList(t *testing.T) {
	ip := net.ParseIP("192.168.1.100")
	if !IsIpInCIDRList(ip, []string{"192.168.1.0/24", "10.0.0.0/8"}) {
		t.Error("should be in CIDR list")
	}
	if IsIpInCIDRList(net.ParseIP("8.8.8.8"), []string{"192.168.1.0/24"}) {
		t.Error("should not be in CIDR list")
	}
}

func TestGetRealAddressByIP_Internal(t *testing.T) {
	if got := GetRealAddressByIP("127.0.0.1"); got != INTERNAL_IP {
		t.Errorf("got %q, want %q", got, INTERNAL_IP)
	}
}

func TestGetIPLocation_Internal(t *testing.T) {
	country, city, loc, err := GetIPLocation("127.0.0.1")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if country != "Local" || city != "Local" || loc != LocalNetwork {
		t.Errorf("got (%q, %q, %q)", country, city, loc)
	}
}

func TestGetIPLocationCN_Internal(t *testing.T) {
	country, city, loc, err := GetIPLocationCN("10.0.0.1")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if country != "Local" || city != "Local" || loc != LocalNetwork {
		t.Errorf("got (%q, %q, %q)", country, city, loc)
	}
}

func TestGetIPLocationGlobal_Internal(t *testing.T) {
	country, city, loc, err := GetIPLocationGlobal("192.168.1.1")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if country != "Local" || city != "Local" || loc != LocalNetwork {
		t.Errorf("got (%q, %q, %q)", country, city, loc)
	}
}

func TestIsChinaIP(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"112.0.0.1", true},
		{"183.0.0.1", true},
		{"8.8.8.8", false},
		{"invalid", false},
	}
	for _, tt := range tests {
		if got := isChinaIP(tt.input); got != tt.want {
			t.Errorf("isChinaIP(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
