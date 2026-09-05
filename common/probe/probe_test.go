// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package probe_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/probe"
)

func TestProbe_Success_200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	p := probe.New()
	result := p.Execute(context.Background(), probe.Request{
		URL:    srv.URL,
		Method: http.MethodGet,
	})

	if !result.Success {
		t.Errorf("Success = false, want true; error: %s", result.Error)
	}
	if result.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
	if result.Body != `{"status":"ok"}` {
		t.Errorf("Body = %q", result.Body)
	}
	if result.Duration < 0 {
		t.Error("Duration should be non-negative")
	}
}

func TestProbe_StatusCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := probe.New()
	result := p.Execute(context.Background(), probe.Request{
		URL:    srv.URL,
		Method: http.MethodGet,
	})

	if result.Success {
		t.Error("Success = true, want false for 500")
	}
	if result.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", result.StatusCode)
	}
}

func TestProbe_ExpectStatusCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	p := probe.New()
	result := p.Execute(context.Background(), probe.Request{
		URL:    srv.URL,
		Method: http.MethodGet,
		Expect: probe.Expect{StatusCode: 201},
	})

	if !result.Success {
		t.Errorf("Success = false, want true; error: %s", result.Error)
	}
}

func TestProbe_BodyContains(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"healthy","uptime":1234}`)
	}))
	defer srv.Close()

	p := probe.New()
	result := p.Execute(context.Background(), probe.Request{
		URL:    srv.URL,
		Method: http.MethodGet,
		Expect: probe.Expect{BodyContains: "healthy"},
	})

	if !result.Success {
		t.Errorf("Success = false; error: %s", result.Error)
	}

	// Negative case.
	result = p.Execute(context.Background(), probe.Request{
		URL:    srv.URL,
		Method: http.MethodGet,
		Expect: probe.Expect{BodyContains: "unhealthy"},
	})
	if result.Success {
		t.Error("Success = true, should fail on missing substring")
	}
}

func TestProbe_BodyNotContains(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	p := probe.New()
	result := p.Execute(context.Background(), probe.Request{
		URL:    srv.URL,
		Method: http.MethodGet,
		Expect: probe.Expect{BodyNotContains: "error"},
	})

	if !result.Success {
		t.Errorf("Success = false; error: %s", result.Error)
	}

	// Should fail when the body contains the forbidden string.
	result = p.Execute(context.Background(), probe.Request{
		URL:    srv.URL,
		Method: http.MethodGet,
		Expect: probe.Expect{BodyNotContains: "ok"},
	})
	if result.Success {
		t.Error("Success = true, should fail when body contains forbidden string")
	}
}

func TestProbe_JSONPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"token":"abc123","user":"alice"}}`)
	}))
	defer srv.Close()

	p := probe.New()
	result := p.Execute(context.Background(), probe.Request{
		URL:    srv.URL,
		Method: http.MethodGet,
		Expect: probe.Expect{BodyJSONPath: "data.token"},
	})

	if !result.Success {
		t.Errorf("Success = false; error: %s", result.Error)
	}
	if result.Extracted["data.token"] != "abc123" {
		t.Errorf("Extracted[data.token] = %q, want abc123", result.Extracted["data.token"])
	}
}

func TestProbe_CustomValidator(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"count":5}`)
	}))
	defer srv.Close()

	p := probe.New()
	result := p.Execute(context.Background(), probe.Request{
		URL:    srv.URL,
		Method: http.MethodGet,
		Expect: probe.Expect{
			Validator: func(status int, body []byte) error {
				var data struct {
					Count int `json:"count"`
				}
				if err := json.Unmarshal(body, &data); err != nil {
					return err
				}
				if data.Count < 10 {
					return fmt.Errorf("count %d < 10", data.Count)
				}
				return nil
			},
		},
	})

	if result.Success {
		t.Error("Success = true, validator should have failed")
	}
	if result.Error == "" {
		t.Error("Error should not be empty")
	}
}

func TestProbe_POST_WithJSONBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] == "test" {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"id":1}`)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer srv.Close()

	p := probe.New()
	result := p.Execute(context.Background(), probe.Request{
		URL:      srv.URL,
		Method:   http.MethodPost,
		BodyJSON: map[string]string{"name": "test"},
		Expect:   probe.Expect{StatusCode: 201},
	})

	if !result.Success {
		t.Errorf("Success = false; error: %s", result.Error)
	}
	if result.StatusCode != 201 {
		t.Errorf("StatusCode = %d, want 201", result.StatusCode)
	}
}

func TestProbe_VariableSubstitution(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Token")
		if token == "secret123" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "authorized")
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer srv.Close()

	p := probe.New()
	result := p.Execute(context.Background(), probe.Request{
		URL:    srv.URL,
		Method: http.MethodGet,
		Headers: map[string]string{
			"X-Token": "$${authToken}",
		},
		Variables: map[string]string{
			"authToken": "secret123",
		},
		Expect: probe.Expect{BodyContains: "authorized"},
	})

	if !result.Success {
		t.Errorf("Success = false; error: %s", result.Error)
	}
}

func TestProbe_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := probe.New()
	result := p.Execute(context.Background(), probe.Request{
		URL:     srv.URL,
		Method:  http.MethodGet,
		Timeout: 50 * time.Millisecond,
	})

	if result.Success {
		t.Error("Success = true, should timeout")
	}
	if result.Error == "" {
		t.Error("Error should mention timeout")
	}
}

func TestProbe_EmptyURL(t *testing.T) {
	p := probe.New()
	result := p.Execute(context.Background(), probe.Request{})
	if result.Success {
		t.Error("Success = true, want false for empty URL")
	}
}

func TestProbe_TransportError(t *testing.T) {
	p := probe.New()
	result := p.Execute(context.Background(), probe.Request{
		URL:    "http://127.0.0.1:0/nonexistent",
		Method: http.MethodGet,
	})
	if result.Success {
		t.Error("Success = true, should fail on connection error")
	}
}

func TestProbe_QueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			fmt.Fprint(w, "page2")
		} else {
			fmt.Fprint(w, "other")
		}
	}))
	defer srv.Close()

	p := probe.New()
	result := p.Execute(context.Background(), probe.Request{
		URL:    srv.URL,
		Method: http.MethodGet,
		Params: map[string]string{"page": "2"},
		Expect: probe.Expect{BodyContains: "page2"},
	})

	if !result.Success {
		t.Errorf("Success = false; error: %s", result.Error)
	}
}

func TestProbe_SkipTLSVerify(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "secure")
	}))
	defer srv.Close()

	p := probe.New()
	// Without SkipTLSVerify, this would fail due to self-signed cert.
	result := p.Execute(context.Background(), probe.Request{
		URL:           srv.URL,
		Method:        http.MethodGet,
		SkipTLSVerify: true,
		Expect:        probe.Expect{BodyContains: "secure"},
	})

	if !result.Success {
		t.Errorf("Success = false; error: %s", result.Error)
	}
}

// ──────────────────────────────────────────────
// Sequence tests
// ──────────────────────────────────────────────

func TestSequence_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"token":"tok-abc"}}`)
		case "/profile":
			token := r.Header.Get("Authorization")
			if token == "Bearer tok-abc" {
				fmt.Fprint(w, `{"data":{"name":"alice"}}`)
			} else {
				w.WriteHeader(http.StatusUnauthorized)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	steps := []probe.SequenceStep{
		{
			Name: "login",
			Request: probe.Request{
				URL:      srv.URL + "/login",
				Method:   http.MethodPost,
				BodyJSON: map[string]string{"user": "alice"},
				Expect:   probe.Expect{BodyContains: "token"},
			},
			ExtractVars: map[string]string{
				"token": "data.token",
			},
		},
		{
			Name: "fetch-profile",
			Request: probe.Request{
				URL:    srv.URL + "/profile",
				Method: http.MethodGet,
				Headers: map[string]string{
					"Authorization": "Bearer $${token}",
				},
				Expect: probe.Expect{BodyContains: "alice"},
			},
		},
	}

	seq := probe.NewSequence(nil, steps)
	result := seq.Execute(context.Background())

	if !result.Success {
		t.Errorf("Success = false; error: %s", result.Error)
	}
	if len(result.Steps) != 2 {
		t.Errorf("Steps count = %d, want 2", len(result.Steps))
	}
	if result.Variables["token"] != "tok-abc" {
		t.Errorf("Variables[token] = %q, want tok-abc", result.Variables["token"])
	}
}

func TestSequence_StepFailure_SkipsRest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	steps := []probe.SequenceStep{
		{Name: "step1", Request: probe.Request{URL: srv.URL, Method: http.MethodGet}},
		{Name: "step2", Request: probe.Request{URL: srv.URL, Method: http.MethodGet}},
	}

	seq := probe.NewSequence(nil, steps)
	result := seq.Execute(context.Background())

	if result.Success {
		t.Error("Success = true, want false")
	}
	if !result.Steps[0].Result.Success {
		// step1 should fail
	}
	if !result.Steps[1].Skipped {
		t.Error("Step 2 should be skipped")
	}
}
