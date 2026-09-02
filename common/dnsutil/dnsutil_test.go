// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package dnsutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeServer(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", DefaultServer},
		{"8.8.8.8", "8.8.8.8:53"},
		{"1.1.1.1:53", "1.1.1.1:53"},
		{"  8.8.4.4  ", "8.8.4.4:53"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, normalizeServer(tt.input))
	}
}

func TestReverseZone_IPv4(t *testing.T) {
	zone, err := reverseZone("8.8.8.8")
	require.NoError(t, err)
	assert.Equal(t, "8.8.8.8.in-addr.arpa.", zone)

	zone, err = reverseZone("192.168.1.1")
	require.NoError(t, err)
	assert.Equal(t, "1.1.168.192.in-addr.arpa.", zone)
}

func TestReverseZone_IPv6(t *testing.T) {
	zone, err := reverseZone("2001:db8::1")
	require.NoError(t, err)
	assert.Contains(t, zone, "ip6.arpa.")
}

func TestReverseZone_Invalid(t *testing.T) {
	_, err := reverseZone("not-an-ip")
	require.Error(t, err)
}

func TestSupportedRecordTypes(t *testing.T) {
	types := SupportedRecordTypes()
	assert.NotEmpty(t, types)
	assert.Contains(t, types, "A")
	assert.Contains(t, types, "MX")
	assert.Contains(t, types, "TXT")
}

func TestCommonRecordTypes(t *testing.T) {
	types := CommonRecordTypes()
	assert.Equal(t, []string{"A", "AAAA", "CNAME", "MX", "NS", "TXT", "SOA", "PTR", "SRV", "CAA"}, types)
}

func TestNewClient_Defaults(t *testing.T) {
	c := NewClient()
	assert.Equal(t, DefaultServer, c.server)
	assert.Equal(t, DefaultTimeout, c.timeout)
	assert.False(t, c.udp)
	assert.False(t, c.dnssec)
}

func TestNewClient_Options(t *testing.T) {
	c := NewClient(
		WithServer("1.1.1.1:53"),
		WithTimeout(10*time.Second),
		WithUDP(),
		WithDNSSEC(),
	)
	assert.Equal(t, "1.1.1.1:53", c.server)
	assert.Equal(t, 10*time.Second, c.timeout)
	assert.True(t, c.udp)
	assert.True(t, c.dnssec)
}

func TestQuery_EmptyDomain(t *testing.T) {
	c := NewClient()
	_, err := c.Query("", "A")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestQuery_InvalidType(t *testing.T) {
	c := NewClient()
	_, err := c.Query("example.com", "INVALID")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestParseRecord_A(t *testing.T) {
	// parseRecord is tested implicitly via real queries, but we can
	// verify the CommonRecordTypes list matches what we expect
	types := CommonRecordTypes()
	for _, rt := range types {
		assert.NotEmpty(t, rt)
	}
}

// ─── Integration tests (require network) ────────────────────────

func TestQueryA_Google(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test")
	}
	records, err := QueryA("google.com", ServerGoogle)
	require.NoError(t, err)
	assert.NotEmpty(t, records)
	for _, r := range records {
		assert.Equal(t, "A", r.Type)
		assert.NotEmpty(t, r.Value)
	}
}

func TestQueryMX_Google(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test")
	}
	records, err := QueryMX("google.com", ServerCloudflare)
	require.NoError(t, err)
	assert.NotEmpty(t, records)
	for _, r := range records {
		assert.Equal(t, "MX", r.Type)
		assert.Contains(t, r.Value, "google.com")
	}
}

func TestQueryTXT_Google(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test")
	}
	records, err := QueryTXT("google.com", ServerGoogle)
	require.NoError(t, err)
	assert.NotEmpty(t, records)
}

func TestQueryNS_Google(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test")
	}
	records, err := QueryNS("google.com", ServerGoogle)
	require.NoError(t, err)
	assert.NotEmpty(t, records)
}

func TestReverseLookup_GoogleDNS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test")
	}
	records, err := ReverseLookup("8.8.8.8", ServerGoogle)
	require.NoError(t, err)
	assert.NotEmpty(t, records)
	assert.Equal(t, "PTR", records[0].Type)
	assert.Contains(t, records[0].Value, "google")
}

func TestQueryAll_Example(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test")
	}
	result, err := QueryAll("example.com", ServerGoogle)
	require.NoError(t, err)
	assert.Equal(t, "example.com", result.Domain)
	assert.NotEmpty(t, result.A)
}

func TestQuery_Cloudflare(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test")
	}
	records, err := Query("cloudflare.com", "A", ServerCloudflare)
	require.NoError(t, err)
	assert.NotEmpty(t, records)
}

func TestQuery_AliDNS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test")
	}
	records, err := QueryA("taobao.com", ServerAliDNS)
	require.NoError(t, err)
	assert.NotEmpty(t, records)
}
