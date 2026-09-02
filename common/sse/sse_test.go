// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package sse

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Writer
// ──────────────────────────────────────────────

func TestNewWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	w, err := NewWriter(rec)
	require.NoError(t, err)
	assert.NotNil(t, w)
}

func TestNewWriter_NoFlusher(t *testing.T) {
	// A plain map-based ResponseWriter that does not implement Flusher.
	w := &noFlusher{}
	_, err := NewWriter(w)
	assert.ErrorIs(t, err, ErrFlusherNotSupported)
}

type noFlusher struct{}

func (*noFlusher) Header() http.Header             { return http.Header{} }
func (*noFlusher) Write([]byte) (int, error)       { return 0, nil }
func (*noFlusher) WriteHeader(int)                 {}

func TestSetHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	SetHeaders(rec)
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", rec.Header().Get("Connection"))
}

func TestWriter_WriteFullEvent(t *testing.T) {
	rec := httptest.NewRecorder()
	w, err := NewWriter(rec)
	require.NoError(t, err)

	err = w.Write(&Event{ID: "1", Event: "ping", Data: "pong", Retry: 5000})
	require.NoError(t, err)

	body := rec.Body.String()
	assert.Contains(t, body, "id: 1\n")
	assert.Contains(t, body, "event: ping\n")
	assert.Contains(t, body, "data: pong\n")
	assert.Contains(t, body, "retry: 5000\n")
	assert.True(t, strings.HasSuffix(body, "\n\n"), "event should end with blank line")
}

func TestWriter_WriteMultiLineData(t *testing.T) {
	rec := httptest.NewRecorder()
	w, err := NewWriter(rec)
	require.NoError(t, err)

	err = w.Write(&Event{Data: "line1\nline2\nline3"})
	require.NoError(t, err)

	body := rec.Body.String()
	assert.Contains(t, body, "data: line1\n")
	assert.Contains(t, body, "data: line2\n")
	assert.Contains(t, body, "data: line3\n")
}

func TestWriter_WriteNil(t *testing.T) {
	rec := httptest.NewRecorder()
	w, err := NewWriter(rec)
	require.NoError(t, err)

	err = w.Write(nil)
	require.NoError(t, err)
	assert.Empty(t, rec.Body.String())
}

func TestWriter_WriteData(t *testing.T) {
	rec := httptest.NewRecorder()
	w, err := NewWriter(rec)
	require.NoError(t, err)

	err = w.WriteData("hello")
	require.NoError(t, err)
	assert.Equal(t, "data: hello\n\n", rec.Body.String())
}

func TestWriter_WriteComment(t *testing.T) {
	rec := httptest.NewRecorder()
	w, err := NewWriter(rec)
	require.NoError(t, err)

	err = w.WriteComment("keep-alive")
	require.NoError(t, err)
	assert.Equal(t, ": keep-alive\n\n", rec.Body.String())
}

func TestWriter_WriteCommentMultiLine(t *testing.T) {
	rec := httptest.NewRecorder()
	w, err := NewWriter(rec)
	require.NoError(t, err)

	err = w.WriteComment("a\nb")
	require.NoError(t, err)
	assert.Equal(t, ": a\n: b\n\n", rec.Body.String())
}

func TestWriter_WriteRetry(t *testing.T) {
	rec := httptest.NewRecorder()
	w, err := NewWriter(rec)
	require.NoError(t, err)

	err = w.WriteRetry(3000)
	require.NoError(t, err)
	assert.Equal(t, "retry: 3000\n\n", rec.Body.String())
}

func TestWriter_Close(t *testing.T) {
	rec := httptest.NewRecorder()
	w, err := NewWriter(rec)
	require.NoError(t, err)
	assert.NoError(t, w.Close())
}

func TestWriter_WriteError(t *testing.T) {
	w, err := NewWriter(&errorWriter{})
	require.NoError(t, err)
	err = w.WriteData("x")
	assert.Error(t, err)
}

// errorWriter implements http.ResponseWriter and http.Flusher but always
// fails on Write.
type errorWriter struct{}

func (*errorWriter) Header() http.Header             { return http.Header{} }
func (*errorWriter) Write([]byte) (int, error)       { return 0, io.ErrShortWrite }
func (*errorWriter) WriteHeader(int)                 {}
func (*errorWriter) Flush()                          {}

// ──────────────────────────────────────────────
// ParseEvent
// ──────────────────────────────────────────────

func TestParseEvent(t *testing.T) {
	ev := ParseEvent("id: 42\nevent: update\ndata: hello\nretry: 1000\n\n")
	assert.Equal(t, "42", ev.ID)
	assert.Equal(t, "update", ev.Event)
	assert.Equal(t, "hello", ev.Data)
	assert.Equal(t, 1000, ev.Retry)
}

func TestParseEvent_MultiLineData(t *testing.T) {
	ev := ParseEvent("data: line1\ndata: line2\ndata: line3\n\n")
	assert.Equal(t, "line1\nline2\nline3", ev.Data)
}

func TestParseEvent_Comment(t *testing.T) {
	ev := ParseEvent(": a comment\ndata: hello\n\n")
	assert.Equal(t, "hello", ev.Data)
}

func TestParseEvent_NoSpaceAfterColon(t *testing.T) {
	ev := ParseEvent("data:hello\n\n")
	assert.Equal(t, "hello", ev.Data)
}

func TestParseEvent_UnknownField(t *testing.T) {
	ev := ParseEvent("foo: bar\ndata: hello\n\n")
	assert.Equal(t, "hello", ev.Data)
}

func TestParseEvent_NoColon(t *testing.T) {
	ev := ParseEvent("datahello\n\n")
	// field is the whole line, value is empty
	assert.Equal(t, "", ev.Data)
}

func TestParseEvent_Empty(t *testing.T) {
	ev := ParseEvent("")
	assert.NotNil(t, ev)
	assert.Empty(t, ev.Data)
}

func TestParseEvent_InvalidRetry(t *testing.T) {
	ev := ParseEvent("retry: abc\ndata: hello\n\n")
	assert.Equal(t, 0, ev.Retry)
	assert.Equal(t, "hello", ev.Data)
}

// ──────────────────────────────────────────────
// Client / StreamReader
// ──────────────────────────────────────────────

func TestClient_ConnectAndRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetHeaders(w)
		sw, err := NewWriter(w)
		if err != nil {
			t.Errorf("NewWriter: %v", err)
			return
		}
		_ = sw.Write(&Event{ID: "1", Event: "ping", Data: "pong"})
		_ = sw.WriteData("second")
		_ = sw.Write(&Event{ID: "2", Data: "multi\nline"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.Connect(ctx)
	require.NoError(t, err)
	defer stream.Close()

	ev, err := stream.Next()
	require.NoError(t, err)
	assert.Equal(t, "1", ev.ID)
	assert.Equal(t, "ping", ev.Event)
	assert.Equal(t, "pong", ev.Data)

	ev, err = stream.Next()
	require.NoError(t, err)
	assert.Equal(t, "second", ev.Data)
	assert.Empty(t, ev.ID)

	ev, err = stream.Next()
	require.NoError(t, err)
	assert.Equal(t, "2", ev.ID)
	assert.Equal(t, "multi\nline", ev.Data)

	_, err = stream.Next()
	assert.ErrorIs(t, err, io.EOF)
	assert.NoError(t, stream.Err())
}

func TestClient_ConnectWithRetryField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw, _ := NewWriter(w)
		_ = sw.Write(&Event{Retry: 2000, Data: "x"})
	}))
	defer srv.Close()

	stream, err := NewClient(srv.URL).Connect(context.Background())
	require.NoError(t, err)
	defer stream.Close()

	ev, err := stream.Next()
	require.NoError(t, err)
	assert.Equal(t, 2000, ev.Retry)
	assert.Equal(t, "x", ev.Data)
}

func TestClient_ConnectCommentOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw, _ := NewWriter(w)
		_ = sw.WriteComment("keepalive")
		_ = sw.WriteData("real")
	}))
	defer srv.Close()

	stream, err := NewClient(srv.URL).Connect(context.Background())
	require.NoError(t, err)
	defer stream.Close()

	ev, err := stream.Next()
	require.NoError(t, err)
	assert.Equal(t, "real", ev.Data)
}

func TestClient_ConnectUnknownField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "foo: bar\ndata: hello\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer srv.Close()

	stream, err := NewClient(srv.URL).Connect(context.Background())
	require.NoError(t, err)
	defer stream.Close()

	ev, err := stream.Next()
	require.NoError(t, err)
	assert.Equal(t, "hello", ev.Data)
}

func TestClient_BadStatusCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := NewClient(srv.URL).Connect(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestClient_BadURL(t *testing.T) {
	_, err := NewClient("http://127.0.0.1:0/events").Connect(context.Background())
	assert.Error(t, err)
}

func TestClient_InvalidURL(t *testing.T) {
	_, err := NewClient("://bad-url").Connect(context.Background())
	assert.Error(t, err)
}

func TestClient_WithContextHTTPClient(t *testing.T) {
	hc := &http.Client{Timeout: 10 * time.Second}
	c := NewClient("http://example.com").WithHTTPClient(hc)
	assert.Equal(t, hc, c.httpClient)
}

func TestStreamReader_CloseTwice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw, _ := NewWriter(w)
		_ = sw.WriteData("x")
	}))
	defer srv.Close()

	stream, err := NewClient(srv.URL).Connect(context.Background())
	require.NoError(t, err)
	require.NoError(t, stream.Close())
	// second close is a no-op
	require.NoError(t, stream.Close())
}

func TestStreamReader_CRLFLines(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "data: hello\r\n\r\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer srv.Close()

	stream, err := NewClient(srv.URL).Connect(context.Background())
	require.NoError(t, err)
	defer stream.Close()

	ev, err := stream.Next()
	require.NoError(t, err)
	assert.Equal(t, "hello", ev.Data)
}

// ──────────────────────────────────────────────
// strconv.Atoi helper (replaces former hand-written parseInt)
// ──────────────────────────────────────────────

func TestParseInt(t *testing.T) {
	n, err := strconv.Atoi("123")
	require.NoError(t, err)
	assert.Equal(t, 123, n)

	_, err = strconv.Atoi("abc")
	assert.Error(t, err)

	_, err = strconv.Atoi("")
	assert.Error(t, err)
}
