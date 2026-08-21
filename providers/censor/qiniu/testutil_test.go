// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package qiniu

import (
	"context"
	"net/http"
	"net/http/httptest"
)

// contextWithCancel returns a pre-canceled context for testing request errors.
func contextWithCancel() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx, cancel
}

// newTestServer creates a TLS test server that responds with the given status
// code and body. It returns the server and an HTTP client configured to trust
// the server's certificate. The host should be set to srv.Listener.Addr().String().
func newTestServer(statusCode int, body string) (*httptest.Server, *http.Client) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	}))
	return srv, srv.Client()
}

// newTestServerFunc creates a TLS test server with a custom handler function.
func newTestServerFunc(handler http.HandlerFunc) (*httptest.Server, *http.Client) {
	srv := httptest.NewTLSServer(handler)
	return srv, srv.Client()
}

// testConfig builds a Config that points at the given test server.
func testConfig(srv *httptest.Server, client *http.Client) Config {
	return Config{
		AccessKey: "ak",
		SecretKey: "sk",
		Host:      srv.Listener.Addr().String(),
		Client:    client,
	}
}
