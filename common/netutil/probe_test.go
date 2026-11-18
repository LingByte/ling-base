// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package netutil_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/netutil"
)

func TestProber_Success_200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Probe", "1")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	p := netutil.NewProber()
	result := p.Execute(context.Background(), netutil.ProbeRequest{
		URL:    srv.URL,
		Method: http.MethodGet,
	})
	if !result.Success {
		t.Fatalf("Success=false: %s", result.Error)
	}
	if result.StatusCode != 200 || result.Body != `{"status":"ok"}` {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Headers.Get("X-Probe") != "1" {
		t.Fatalf("missing response header")
	}
	if result.Truncated {
		t.Fatal("should not be truncated")
	}
}

func TestProber_UnsupportedMethod(t *testing.T) {
	p := netutil.NewProber()
	result := p.Execute(context.Background(), netutil.ProbeRequest{
		URL:    "http://example.com",
		Method: "TRACE",
	})
	if result.Success || !errors.Is(result.Err, netutil.ErrProbeUnsupportedMethod) {
		t.Fatalf("want ErrProbeUnsupportedMethod, got %#v %v", result.Err, result.Error)
	}
}

func TestProber_VariableKeyOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, r.URL.Query().Get("q"))
	}))
	defer srv.Close()

	p := netutil.NewProber()
	result := p.Execute(context.Background(), netutil.ProbeRequest{
		URL:    srv.URL + "?q=$${aa}",
		Params: map[string]string{},
		Variables: map[string]string{
			"a":  "SHORT",
			"aa": "LONG",
		},
		Expect: netutil.ProbeExpect{BodyContains: "LONG"},
	})
	if !result.Success {
		t.Fatalf("Success=false: %s body=%q", result.Error, result.Body)
	}
}

func TestProber_BodyJSONVariables(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["token"] == "abc" {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"ok":true}`)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	p := netutil.NewProber()
	result := p.Execute(context.Background(), netutil.ProbeRequest{
		URL:    srv.URL,
		Method: http.MethodPost,
		BodyJSON: map[string]string{
			"token": "$${tok}",
		},
		Variables: map[string]string{"tok": "abc"},
		Expect:    netutil.ProbeExpect{StatusCode: 201},
	})
	if !result.Success {
		t.Fatalf("Success=false: %s", result.Error)
	}
}

func TestProber_JSONPathNegativeIndex(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"items":[{"id":"a"},{"id":"b"},{"id":"c"}]}`)
	}))
	defer srv.Close()

	p := netutil.NewProber()
	result := p.Execute(context.Background(), netutil.ProbeRequest{
		URL:    srv.URL,
		Expect: netutil.ProbeExpect{BodyJSONPath: "items.-1.id"},
	})
	if !result.Success {
		t.Fatalf("Success=false: %s", result.Error)
	}
	if result.Extracted["items.-1.id"] != "c" {
		t.Fatalf("got %q", result.Extracted["items.-1.id"])
	}
}

func TestProber_BodyTruncationMarker(t *testing.T) {
	big := strings.Repeat("x", 100)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, big)
	}))
	defer srv.Close()

	p := netutil.NewProber(netutil.WithProbeMaxBodySize(10))
	result := p.Execute(context.Background(), netutil.ProbeRequest{URL: srv.URL})
	if !result.Success {
		t.Fatalf("Success=false: %s", result.Error)
	}
	if !result.Truncated || !strings.Contains(result.Body, "[truncated]") {
		t.Fatalf("expected truncation marker, body=%q truncated=%v", result.Body, result.Truncated)
	}
}

func TestProber_SkipTLSVerify_Reuse(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "secure")
	}))
	defer srv.Close()

	p := netutil.NewProber()
	for i := 0; i < 3; i++ {
		result := p.Execute(context.Background(), netutil.ProbeRequest{
			URL:           srv.URL,
			SkipTLSVerify: true,
			Expect:        netutil.ProbeExpect{BodyContains: "secure"},
		})
		if !result.Success {
			t.Fatalf("iter %d: %s", i, result.Error)
		}
	}
}

func TestProber_NoFollowRedirects(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "final")
	}))
	defer final.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer srv.Close()

	follow := false
	p := netutil.NewProber()
	result := p.Execute(context.Background(), netutil.ProbeRequest{
		URL:             srv.URL,
		FollowRedirects: &follow,
		Expect:          netutil.ProbeExpect{StatusCode: 302},
	})
	if !result.Success {
		t.Fatalf("Success=false: %s", result.Error)
	}
}

func TestProber_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := netutil.NewProber()
	result := p.Execute(context.Background(), netutil.ProbeRequest{
		URL:     srv.URL,
		Timeout: 50 * time.Millisecond,
	})
	if result.Success || result.Err == nil || !errors.Is(result.Err, netutil.ErrProbeNetwork) {
		t.Fatalf("want network timeout, got success=%v err=%v", result.Success, result.Err)
	}
}

func TestProber_EmptyURL(t *testing.T) {
	result := netutil.NewProber().Execute(context.Background(), netutil.ProbeRequest{})
	if !errors.Is(result.Err, netutil.ErrProbeEmptyURL) {
		t.Fatalf("got %v", result.Err)
	}
}

func TestProber_Clone(t *testing.T) {
	r := netutil.ProbeRequest{
		URL:       "http://x",
		Headers:   map[string]string{"A": "1"},
		Variables: map[string]string{"k": "v"},
	}
	c := r.Clone()
	c.Headers["A"] = "2"
	c.Variables["k"] = "z"
	if r.Headers["A"] != "1" || r.Variables["k"] != "v" {
		t.Fatal("clone mutated original")
	}
}

func TestProbeSequence_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			fmt.Fprint(w, `{"data":{"token":"tok-abc"}}`)
		case "/profile":
			if r.Header.Get("Authorization") == "Bearer tok-abc" {
				fmt.Fprint(w, `{"data":{"name":"alice"}}`)
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	seq := netutil.NewProbeSequence(nil, []netutil.ProbeSequenceStep{
		{
			Name: "login",
			Request: netutil.ProbeRequest{
				URL:      srv.URL + "/login",
				Method:   http.MethodPost,
				BodyJSON: map[string]string{"user": "alice"},
				Expect:   netutil.ProbeExpect{BodyContains: "token"},
			},
			ExtractVars: map[string]string{"token": "data.token"},
		},
		{
			Name: "profile",
			Request: netutil.ProbeRequest{
				URL:     srv.URL + "/profile",
				Headers: map[string]string{"Authorization": "Bearer $${token}"},
				Expect:  netutil.ProbeExpect{BodyContains: "alice"},
			},
		},
	})
	result := seq.Execute(context.Background())
	if !result.Success {
		t.Fatalf("Success=false: %s", result.Error)
	}
	if result.Variables["token"] != "tok-abc" {
		t.Fatalf("token=%q", result.Variables["token"])
	}
}

func TestProbeSequence_SkipOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	seq := netutil.NewProbeSequence(nil, []netutil.ProbeSequenceStep{
		{Name: "step1", Request: netutil.ProbeRequest{URL: srv.URL}},
		{Name: "step2", Request: netutil.ProbeRequest{URL: srv.URL}},
	})
	result := seq.Execute(context.Background())
	if result.Success || !result.Steps[1].Skipped {
		t.Fatalf("expected skip, got %+v", result)
	}
}

func TestProber_CustomValidatorHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Count", "5")
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	p := netutil.NewProber()
	result := p.Execute(context.Background(), netutil.ProbeRequest{
		URL: srv.URL,
		Expect: netutil.ProbeExpect{
			Validator: func(status int, body []byte, headers http.Header) error {
				if headers.Get("X-Count") != "10" {
					return fmt.Errorf("bad count")
				}
				return nil
			},
		},
	})
	if result.Success || !errors.Is(result.Err, netutil.ErrProbeValidate) {
		t.Fatalf("want validation error, got %#v", result.Err)
	}
}
