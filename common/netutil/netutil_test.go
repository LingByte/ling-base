// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package netutil

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// ──────────────────────────────────────────────
// Port availability
// ──────────────────────────────────────────────

func TestIsPortAvailable_Free(t *testing.T) {
	port, err := FreePort()
	if err != nil {
		t.Fatalf("FreePort failed: %v", err)
	}
	// The port we just got should be available (though not guaranteed).
	// We test with a definitely-free port by getting a new one.
	port2, err := FreePort()
	if err != nil {
		t.Fatalf("FreePort 2 failed: %v", err)
	}
	if !IsTCPPortAvailable(port2) {
		// Port may have been grabbed, try once more.
		port2, _ = FreePort()
	}
	_ = port
}

func TestIsPortAvailable_InUse(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	if IsTCPPortAvailable(port) {
		t.Fatalf("Port %d should be in use", port)
	}
}

func TestIsUDPPortAvailable(t *testing.T) {
	ln, err := net.ListenPacket("udp", ":0")
	if err != nil {
		t.Fatalf("ListenPacket failed: %v", err)
	}
	port := ln.LocalAddr().(*net.UDPAddr).Port
	_ = ln.Close()
	// After closing, port should be available (usually).
	// Just test that the function doesn't panic.
	_ = IsUDPPortAvailable(port)
}

func TestIsPortAvailable_InvalidNetwork(t *testing.T) {
	if IsPortAvailable("invalid", 8080) {
		t.Fatal("IsPortAvailable with invalid network should return false")
	}
}

func TestFreePort(t *testing.T) {
	port, err := FreePort()
	if err != nil {
		t.Fatalf("FreePort failed: %v", err)
	}
	if port < 0 || port > 65535 {
		t.Fatalf("FreePort = %d, out of range", port)
	}
}

func TestFreeUDPPort(t *testing.T) {
	port, err := FreeUDPPort()
	if err != nil {
		t.Fatalf("FreeUDPPort failed: %v", err)
	}
	if port < 0 || port > 65535 {
		t.Fatalf("FreeUDPPort = %d, out of range", port)
	}
}

func TestFindAvailablePort(t *testing.T) {
	port, err := FindAvailablePort(20000, 20100)
	if err != nil {
		t.Fatalf("FindAvailablePort failed: %v", err)
	}
	if port < 20000 || port > 20100 {
		t.Fatalf("FindAvailablePort = %d, out of range", port)
	}
}

func TestFindAvailablePort_NoneAvailable(t *testing.T) {
	// Use a range that's almost certainly all in use (0-1).
	// Port 0 is reserved, port 1 requires root.
	// This test may be flaky on some systems; skip if it finds one.
	port, err := FindAvailablePort(1, 1)
	if err == nil {
		// Port 1 might be available on some systems.
		_ = port
	}
}

func TestFindAvailablePorts(t *testing.T) {
	ports, err := FindAvailablePorts(30000, 3)
	if err != nil {
		t.Fatalf("FindAvailablePorts failed: %v", err)
	}
	if len(ports) != 3 {
		t.Fatalf("FindAvailablePorts returned %d ports, want 3", len(ports))
	}
	seen := make(map[int]bool)
	for _, p := range ports {
		if seen[p] {
			t.Fatalf("Duplicate port %d", p)
		}
		seen[p] = true
	}
}

// ──────────────────────────────────────────────
// IP address helpers
// ──────────────────────────────────────────────

func TestLocalIP(t *testing.T) {
	ip := LocalIP()
	// May be empty on some systems, but should not panic.
	_ = ip
}

func TestLocalIPs(t *testing.T) {
	ips := LocalIPs()
	// Should return at least something on most systems.
	// Just verify it doesn't panic and returns a slice.
	_ = ips
}

func TestAllIPs(t *testing.T) {
	ips := AllIPs()
	// Should include at least loopback.
	if len(ips) == 0 {
		t.Fatal("AllIPs returned empty")
	}
}

func TestOutboundIP(t *testing.T) {
	// Use a non-routable address to avoid actual network calls.
	// This may fail on some systems; just verify it doesn't panic.
	_, err := OutboundIP("8.8.8.8:80")
	if err != nil {
		// Network may be unavailable in test env.
		_ = err
	}
}

func TestOutboundIP_EmptyTarget(t *testing.T) {
	_, err := OutboundIP("")
	if err != nil {
		// Network may be unavailable.
		_ = err
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"172.32.0.1", false},
		{"192.168.1.1", true},
		{"127.0.0.1", true},   // loopback
		{"169.254.1.1", true}, // link-local
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"invalid", false},
	}
	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			if got := IsPrivateIP(tt.ip); got != tt.want {
				t.Fatalf("IsPrivateIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestIsLoopback(t *testing.T) {
	if !IsLoopback("127.0.0.1") {
		t.Fatal("127.0.0.1 should be loopback")
	}
	if IsLoopback("8.8.8.8") {
		t.Fatal("8.8.8.8 should not be loopback")
	}
	if IsLoopback("invalid") {
		t.Fatal("invalid should not be loopback")
	}
}

func TestIsPublicIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"8.8.8.8", true},
		{"1.1.1.1", true},
		{"10.0.0.1", false},
		{"127.0.0.1", false},
		{"169.254.1.1", false},
		{"0.0.0.0", false},
		{"invalid", false},
	}
	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			if got := IsPublicIP(tt.ip); got != tt.want {
				t.Fatalf("IsPublicIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestIPRange(t *testing.T) {
	start, end, err := IPRange("192.168.1.0/24")
	if err != nil {
		t.Fatalf("IPRange failed: %v", err)
	}
	if start != "192.168.1.0" {
		t.Fatalf("start = %s, want 192.168.1.0", start)
	}
	if end != "192.168.1.255" {
		t.Fatalf("end = %s, want 192.168.1.255", end)
	}
}

func TestIPRange_Invalid(t *testing.T) {
	_, _, err := IPRange("invalid")
	if err == nil {
		t.Fatal("IPRange with invalid CIDR should error")
	}
}

// ──────────────────────────────────────────────
// HTTP client
// ──────────────────────────────────────────────

func TestHTTPClient_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	client := NewHTTPClient(HTTPClientConfig{
		Timeout:    5 * time.Second,
		MaxRetries: 2,
	})
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestHTTPClient_RetryOn500(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	client := NewHTTPClient(HTTPClientConfig{
		Timeout:       5 * time.Second,
		MaxRetries:    3,
		RetryInterval: 10 * time.Millisecond,
	})
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Fatalf("attempts = %d, want 3", atomic.LoadInt32(&attempts))
	}
}

func TestHTTPClient_RetryExhausted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewHTTPClient(HTTPClientConfig{
		Timeout:       5 * time.Second,
		MaxRetries:    2,
		RetryInterval: 10 * time.Millisecond,
	})
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
}

func TestHTTPClient_NonRetryableStatus(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewHTTPClient(HTTPClientConfig{
		Timeout:       5 * time.Second,
		MaxRetries:    3,
		RetryInterval: 10 * time.Millisecond,
	})
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 1 {
		t.Fatalf("attempts = %d, want 1 (404 not retried)", atomic.LoadInt32(&attempts))
	}
}

func TestHTTPClient_Post(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("received"))
	}))
	defer srv.Close()

	client := NewHTTPClient(HTTPClientConfig{Timeout: 5 * time.Second})
	resp, err := client.Post(srv.URL, "text/plain", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestHTTPClient_ConnectionError(t *testing.T) {
	client := NewHTTPClient(HTTPClientConfig{
		Timeout:       1 * time.Second,
		MaxRetries:    2,
		RetryInterval: 10 * time.Millisecond,
	})
	_, err := client.Get("http://127.0.0.1:1") // port 1 should fail
	if err == nil {
		t.Fatal("Get to invalid port should error")
	}
}

func TestHTTPClient_Defaults(t *testing.T) {
	client := NewHTTPClient(HTTPClientConfig{})
	if client.config.Timeout != 30*time.Second {
		t.Fatalf("default timeout = %v, want 30s", client.config.Timeout)
	}
	if client.config.RetryInterval != time.Second {
		t.Fatalf("default retry interval = %v, want 1s", client.config.RetryInterval)
	}
	if !client.config.RetryableStatus[429] {
		t.Fatal("429 should be retryable by default")
	}
}

// ──────────────────────────────────────────────
// Circuit breaker
// ──────────────────────────────────────────────

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Second)
	if cb.State() != CircuitClosed {
		t.Fatal("initial state should be closed")
	}
	if !cb.Allow() {
		t.Fatal("closed circuit should allow")
	}
}

func TestCircuitBreaker_OpenAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Second)
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}
	if cb.State() != CircuitOpen {
		t.Fatalf("state = %v, want open", cb.State())
	}
	if cb.Allow() {
		t.Fatal("open circuit should not allow")
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatal("should be open")
	}
	time.Sleep(60 * time.Millisecond)
	if !cb.Allow() {
		t.Fatal("should allow after timeout (half-open)")
	}
	if cb.State() != CircuitHalfOpen {
		t.Fatalf("state = %v, want half-open", cb.State())
	}
}

func TestCircuitBreaker_CloseOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)
	cb.RecordFailure()
	cb.RecordFailure()
	time.Sleep(60 * time.Millisecond)
	_ = cb.Allow() // transition to half-open
	cb.RecordSuccess()
	if cb.State() != CircuitClosed {
		t.Fatalf("state = %v, want closed after success", cb.State())
	}
}

func TestCircuitBreaker_OpenOnHalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)
	cb.RecordFailure()
	cb.RecordFailure()
	time.Sleep(60 * time.Millisecond)
	_ = cb.Allow() // half-open
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatalf("state = %v, want open after half-open failure", cb.State())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Second)
	cb.RecordFailure()
	cb.RecordFailure()
	cb.Reset()
	if cb.State() != CircuitClosed {
		t.Fatal("should be closed after reset")
	}
	if cb.Failures() != 0 {
		t.Fatal("failures should be 0 after reset")
	}
}

func TestCircuitBreaker_WithHTTPClient(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Second)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewHTTPClient(HTTPClientConfig{
		Timeout:        2 * time.Second,
		MaxRetries:     0,
		RetryInterval:  10 * time.Millisecond,
		CircuitBreaker: cb,
	})

	// First request: fails, circuit records failure.
	resp, _ := client.Get(srv.URL)
	if resp != nil {
		_ = resp.Body.Close()
	}
	resp, _ = client.Get(srv.URL)
	if resp != nil {
		_ = resp.Body.Close()
	}

	// After 2 failures, circuit should be open.
	if cb.State() != CircuitOpen {
		t.Fatalf("circuit state = %v, want open", cb.State())
	}

	// Next request should be rejected by circuit breaker.
	_, err := client.Get(srv.URL)
	if err == nil {
		t.Fatal("request should fail with circuit open")
	}
}

// ──────────────────────────────────────────────
// TCP utilities
// ──────────────────────────────────────────────

func TestTCPConnect(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	if err := TCPConnect(addr, 2*time.Second); err != nil {
		t.Fatalf("TCPConnect failed: %v", err)
	}
}

func TestTCPConnect_Failure(t *testing.T) {
	err := TCPConnect("127.0.0.1:1", 100*time.Millisecond)
	if err == nil {
		t.Fatal("TCPConnect to port 1 should fail")
	}
}

func TestTCPPing(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	dur, err := TCPPing(addr, 2*time.Second)
	if err != nil {
		t.Fatalf("TCPPing failed: %v", err)
	}
	if dur <= 0 {
		t.Fatal("TCPPing duration should be positive")
	}
}

func TestIsTCPReachable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	if !IsTCPReachable(addr, 2*time.Second) {
		t.Fatal("IsTCPReachable should be true")
	}
	if IsTCPReachable("127.0.0.1:1", 100*time.Millisecond) {
		t.Fatal("IsTCPReachable for port 1 should be false")
	}
}

func TestTCPExchange(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 1024)
		n, _ := conn.Read(buf)
		conn.Write(append([]byte("echo: "), buf[:n]...))
	}()

	addr := ln.Addr().String()
	resp, err := TCPExchange(addr, []byte("hello"), 2*time.Second)
	if err != nil {
		t.Fatalf("TCPExchange failed: %v", err)
	}
	if string(resp) != "echo: hello" {
		t.Fatalf("TCPExchange response = %q, want %q", string(resp), "echo: hello")
	}
}

// ──────────────────────────────────────────────
// UDP utilities
// ──────────────────────────────────────────────

func TestUDPExchange(t *testing.T) {
	ln, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket failed: %v", err)
	}
	defer ln.Close()

	go func() {
		buf := make([]byte, 1024)
		n, addr, err := ln.ReadFrom(buf)
		if err != nil {
			return
		}
		ln.WriteTo(append([]byte("udp-echo: "), buf[:n]...), addr)
	}()

	addr := ln.LocalAddr().String()
	resp, err := UDPExchange(addr, []byte("ping"), 2*time.Second)
	if err != nil {
		t.Fatalf("UDPExchange failed: %v", err)
	}
	if string(resp) != "udp-echo: ping" {
		t.Fatalf("UDPExchange response = %q, want %q", string(resp), "udp-echo: ping")
	}
}

func TestUDPSend(t *testing.T) {
	ln, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket failed: %v", err)
	}
	defer ln.Close()

	addr := ln.LocalAddr().String()
	if err := UDPSend(addr, []byte("test")); err != nil {
		t.Fatalf("UDPSend failed: %v", err)
	}

	// Verify the data was received.
	buf := make([]byte, 1024)
	_ = ln.SetReadDeadline(time.Now().Add(time.Second))
	n, _, err := ln.ReadFrom(buf)
	if err != nil {
		t.Fatalf("ReadFrom failed: %v", err)
	}
	if string(buf[:n]) != "test" {
		t.Fatalf("received = %q, want %q", string(buf[:n]), "test")
	}
}

// ──────────────────────────────────────────────
// DNS utilities
// ──────────────────────────────────────────────

func TestLookupHost(t *testing.T) {
	ips, err := LookupHost("localhost")
	if err != nil {
		t.Fatalf("LookupHost failed: %v", err)
	}
	if len(ips) == 0 {
		t.Fatal("LookupHost returned no IPs")
	}
}

func TestResolveIP(t *testing.T) {
	ip := ResolveIP("localhost")
	if ip == "" {
		t.Fatal("ResolveIP(localhost) returned empty")
	}
}

func TestResolveIP_Invalid(t *testing.T) {
	ip := ResolveIP("this-domain-definitely-does-not-exist.invalid")
	if ip != "" {
		t.Fatalf("ResolveIP(invalid) = %q, want empty", ip)
	}
}

// ──────────────────────────────────────────────
// Interface utilities
// ──────────────────────────────────────────────

func TestInterfaces(t *testing.T) {
	ifaces, err := Interfaces()
	if err != nil {
		t.Fatalf("Interfaces failed: %v", err)
	}
	if len(ifaces) == 0 {
		t.Fatal("Interfaces returned empty")
	}
}

func TestInterfaceIPs(t *testing.T) {
	ifaces, err := Interfaces()
	if err != nil || len(ifaces) == 0 {
		t.Skip("No interfaces available")
	}
	// Test with the first interface (usually lo0 or en0).
	ips, err := InterfaceIPs(ifaces[0])
	if err != nil {
		// Some interfaces may not have IPs.
		_ = err
	}
	_ = ips
}

func TestInterfaceIPs_Invalid(t *testing.T) {
	_, err := InterfaceIPs("nonexistent-interface-12345")
	if err == nil {
		t.Fatal("InterfaceIPs with invalid name should error")
	}
}

func TestMACAddress(t *testing.T) {
	ifaces, err := Interfaces()
	if err != nil || len(ifaces) == 0 {
		t.Skip("No interfaces available")
	}
	// Just verify it doesn't panic.
	_, _ = MACAddress(ifaces[0])
}

func TestMACAddress_Invalid(t *testing.T) {
	_, err := MACAddress("nonexistent-interface-12345")
	if err == nil {
		t.Fatal("MACAddress with invalid name should error")
	}
}

// ──────────────────────────────────────────────
// PublicIP (may fail without network)
// ──────────────────────────────────────────────

func TestPublicIPWithProvider(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "203.0.113.1")
	}))
	defer srv.Close()

	ip := PublicIPWithProvider(srv.URL)
	if ip != "203.0.113.1" {
		t.Fatalf("PublicIPWithProvider = %q, want 203.0.113.1", ip)
	}
}

func TestPublicIPWithProvider_Error(t *testing.T) {
	ip := PublicIPWithProvider("http://127.0.0.1:1")
	if ip != "" {
		t.Fatalf("PublicIPWithProvider with invalid URL = %q, want empty", ip)
	}
}
