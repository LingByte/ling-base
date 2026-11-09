// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package stores

import (
	"io"
	"testing"
	"time"
)

func TestStoreError_Error(t *testing.T) {
	e := &StoreError{Code: 404, Message: "not found"}
	if e.Error() != "not found" {
		t.Errorf("Error() = %q, want %q", e.Error(), "not found")
	}

	e2 := &StoreError{}
	if e2.Error() != "store error" {
		t.Errorf("Error() = %q, want %q", e2.Error(), "store error")
	}
}

func TestClampPresignTTL(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		def  time.Duration
		want time.Duration
	}{
		{"zero gets default", 0, time.Hour, time.Hour},
		{"negative gets default", -1, time.Hour, time.Hour},
		{"normal passes through", 30 * time.Minute, time.Hour, 30 * time.Minute},
		{"over max gets clamped", 48 * time.Hour, time.Hour, MaxPresignTTL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampPresignTTL(tt.d, tt.def)
			if got != tt.want {
				t.Errorf("clampPresignTTL(%v, %v) = %v, want %v", tt.d, tt.def, got, tt.want)
			}
		})
	}
}

func TestSignedURL_FallbackToPublicURL(t *testing.T) {
	s := &fallbackStore{url: "https://example.com/file.txt"}
	got, err := SignedURL(s, "key", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://example.com/file.txt" {
		t.Errorf("SignedURL = %q", got)
	}
}

func TestSignedURL_EmptyPublicURL(t *testing.T) {
	s := &fallbackStore{url: ""}
	_, err := SignedURL(s, "key", 0)
	if err != ErrInvalidPath {
		t.Errorf("expected ErrInvalidPath, got %v", err)
	}
}

func TestPresignUpload_Unsupported(t *testing.T) {
	s := &fallbackStore{}
	_, err := PresignUpload(s, "key", "image/png", 0)
	if err != ErrDirectUploadUnsupported {
		t.Errorf("expected ErrDirectUploadUnsupported, got %v", err)
	}
}

// fallbackStore implements Store but not PrivateURLSigner or DirectUploadPresigner.
type fallbackStore struct {
	url string
}

func (f *fallbackStore) Read(key string) (io.ReadCloser, int64, error) { return nil, 0, nil }
func (f *fallbackStore) Write(key string, r io.Reader) error           { return nil }
func (f *fallbackStore) Delete(key string) error                       { return nil }
func (f *fallbackStore) Exists(key string) (bool, error)               { return true, nil }
func (f *fallbackStore) PublicURL(key string) string                   { return f.url }
