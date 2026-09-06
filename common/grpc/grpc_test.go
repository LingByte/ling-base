// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package grpc_test

import (
	"context"
	"net"
	"testing"
	"time"

	grpchelper "github.com/LingByte/ling-base/common/grpc"
	stdgrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ──────────────────────────────────────────────
// Dial tests
// ──────────────────────────────────────────────

func TestDial_Insecure(t *testing.T) {
	// Dial a non-existent server — should not fail at dial time
	// because gRPC connections are lazy.
	conn, err := grpchelper.Dial("127.0.0.1:1",
		grpchelper.WithInsecure(),
		grpchelper.WithRoundRobin(),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	if conn == nil {
		t.Fatal("Dial returned nil conn")
	}
	_ = conn.Close()
}

func TestDial_WithTimeout(t *testing.T) {
	conn, err := grpchelper.Dial("127.0.0.1:1",
		grpchelper.WithInsecure(),
		grpchelper.WithTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	_ = conn.Close()
}

func TestDial_WithInterceptors(t *testing.T) {
	called := false
	interceptor := func(ctx context.Context, method string, req, reply interface{}, cc *stdgrpc.ClientConn, invoker stdgrpc.UnaryInvoker, opts ...stdgrpc.CallOption) error {
		called = true
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	conn, err := grpchelper.Dial("127.0.0.1:1",
		grpchelper.WithInsecure(),
		grpchelper.WithUnaryInterceptors(interceptor),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	_ = conn.Close()
	_ = called
}

// ──────────────────────────────────────────────
// ConnPool tests
// ──────────────────────────────────────────────

func TestConnPool_Get(t *testing.T) {
	pool := grpchelper.NewConnPool()
	defer pool.Close()

	conn1, err := pool.Get("127.0.0.1:9999", grpchelper.WithInsecure())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if conn1 == nil {
		t.Fatal("Get returned nil")
	}

	// Second Get should return the same connection.
	conn2, err := pool.Get("127.0.0.1:9999", grpchelper.WithInsecure())
	if err != nil {
		t.Fatalf("Get second: %v", err)
	}
	if conn1 != conn2 {
		t.Error("Pool should return same connection for same target")
	}
}

func TestConnPool_Close(t *testing.T) {
	pool := grpchelper.NewConnPool()

	_, _ = pool.Get("127.0.0.1:9998", grpchelper.WithInsecure())

	if err := pool.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}

	// Double close should not panic.
	if err := pool.Close(); err != nil {
		t.Errorf("Double close: %v", err)
	}
}

// ──────────────────────────────────────────────
// Server tests
// ──────────────────────────────────────────────

func TestNewServer(t *testing.T) {
	srv := grpchelper.NewServer()
	if srv == nil {
		t.Fatal("NewServer returned nil")
	}
	srv.Stop()
}

func TestNewServer_WithOptions(t *testing.T) {
	srv := grpchelper.NewServer(
		grpchelper.WithMaxRecvMsgSize(1024*1024),
		grpchelper.WithMaxSendMsgSize(1024*1024),
	)
	if srv == nil {
		t.Fatal("NewServer returned nil")
	}
	srv.Stop()
}

func TestNewServer_WithInterceptors(t *testing.T) {
	called := false
	interceptor := func(ctx context.Context, req interface{}, info *stdgrpc.UnaryServerInfo, handler stdgrpc.UnaryHandler) (interface{}, error) {
		called = true
		return handler(ctx, req)
	}

	srv := grpchelper.NewServer(
		grpchelper.WithUnaryServerInterceptors(interceptor),
	)
	if srv == nil {
		t.Fatal("NewServer returned nil")
	}
	srv.Stop()
	_ = called
}

// ──────────────────────────────────────────────
// Server start/stop lifecycle
// ──────────────────────────────────────────────

func TestServer_StartAndStop(t *testing.T) {
	srv := grpchelper.NewServer()
	defer srv.Stop()

	// Start listening on a random port.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	addr := lis.Addr().String()

	// Start serving in a goroutine.
	go func() {
		_ = srv.Serve(lis)
	}()

	// Give it a moment to start.
	time.Sleep(100 * time.Millisecond)

	// Connect to verify it's running.
	conn, err := grpchelper.Dial(addr,
		grpchelper.WithInsecure(),
		grpchelper.WithDialOption(stdgrpc.WithBlock()),
	)
	if err != nil {
		t.Fatalf("Dial to server: %v", err)
	}
	_ = conn.Close()

	// Graceful stop.
	srv.GracefulStop()
}

// ──────────────────────────────────────────────
// Interceptor tests
// ──────────────────────────────────────────────

func TestRecoveryUnaryInterceptor(t *testing.T) {
	// The recovery interceptor should convert panics to errors.
	info := &stdgrpc.UnaryServerInfo{FullMethod: "/test/method"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("test panic")
	}

	_, err := grpchelper.RecoveryUnaryInterceptor(context.Background(), nil, info, handler)
	if err == nil {
		t.Error("RecoveryInterceptor should return error on panic")
	}
}

func TestRecoveryUnaryInterceptor_NoPanic(t *testing.T) {
	info := &stdgrpc.UnaryServerInfo{FullMethod: "/test/method"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	resp, err := grpchelper.RecoveryUnaryInterceptor(context.Background(), nil, info, handler)
	if err != nil {
		t.Errorf("RecoveryInterceptor returned error: %v", err)
	}
	if resp != "ok" {
		t.Errorf("RecoveryInterceptor resp = %v, want ok", resp)
	}
}

func TestTimeoutUnaryInterceptor(t *testing.T) {
	// This interceptor adds a timeout if the context doesn't have one.
	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *stdgrpc.ClientConn, opts ...stdgrpc.CallOption) error {
		// Check that a deadline was set.
		_, ok := ctx.Deadline()
		if !ok {
			t.Error("TimeoutInterceptor should set deadline")
		}
		return nil
	}

	interceptor := grpchelper.TimeoutUnaryInterceptor(5 * time.Second)
	err := interceptor(context.Background(), "/test/method", nil, nil, nil, invoker)
	if err != nil {
		t.Errorf("TimeoutInterceptor: %v", err)
	}
}

func TestTimeoutUnaryInterceptor_PreservesExistingDeadline(t *testing.T) {
	// If the context already has a deadline, the interceptor should
	// preserve it.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	originalDeadline, _ := ctx.Deadline()
	defer cancel()

	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *stdgrpc.ClientConn, opts ...stdgrpc.CallOption) error {
		deadline, _ := ctx.Deadline()
		if !deadline.Equal(originalDeadline) {
			t.Error("TimeoutInterceptor should preserve existing deadline")
		}
		return nil
	}

	interceptor := grpchelper.TimeoutUnaryInterceptor(5 * time.Second)
	_ = interceptor(ctx, "/test/method", nil, nil, nil, invoker)
}

// ──────────────────────────────────────────────
// Errors
// ──────────────────────────────────────────────

func TestErrors(t *testing.T) {
	if grpchelper.ErrNotConnected == nil {
		t.Error("ErrNotConnected should not be nil")
	}
	if grpchelper.ErrAlreadyRunning == nil {
		t.Error("ErrAlreadyRunning should not be nil")
	}
}

// ──────────────────────────────────────────────
// Helper: use insecure credentials directly to verify
// the underlying grpc package works.
// ──────────────────────────────────────────────

func TestDirectGRPC(t *testing.T) {
	// Verify that the standard grpc package is accessible.
	_ = insecure.NewCredentials()
	_ = stdgrpc.WithTransportCredentials(insecure.NewCredentials())
}
