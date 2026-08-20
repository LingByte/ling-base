// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package stores

import (
	"io"
	"testing"
	"time"
)

// stubStore is a minimal Store used for testing the helper functions.
type stubStore struct{}

func (s *stubStore) Read(key string) (io.ReadCloser, int64, error) { return nil, 0, nil }
func (s *stubStore) Write(key string, r io.Reader) error           { return nil }
func (s *stubStore) Delete(key string) error                       { return nil }
func (s *stubStore) Exists(key string) (bool, error)               { return true, nil }
func (s *stubStore) PublicURL(key string) string                   { return "" }

// stubStatsStore implements both Store and StorageStatsProvider.
type stubStatsStore struct {
	stubStore
	bucketStats *BucketStats
	cdnStats    *CDNStatsResponse
}

func (s *stubStatsStore) GetBucketStats(bucket string) (*BucketStats, error) {
	if s.bucketStats != nil {
		return s.bucketStats, nil
	}
	return &BucketStats{Bucket: bucket, Size: 100, ObjectCount: 5, UpdatedAt: time.Now()}, nil
}
func (s *stubStatsStore) GetCDNStats(req *CDNStatsRequest) (*CDNStatsResponse, error) {
	if s.cdnStats != nil {
		return s.cdnStats, nil
	}
	return &CDNStatsResponse{Points: []CDNStatsPoint{{Traffic: 1000}}}, nil
}
func (s *stubStatsStore) GetAPIRequestStats(req *APIStatsRequest) (*APIStatsResponse, error) {
	return &APIStatsResponse{}, nil
}
func (s *stubStatsStore) GetOriginFetchStats(req *OriginStatsRequest) (*OriginStatsResponse, error) {
	return &OriginStatsResponse{}, nil
}

func TestSupportsStats(t *testing.T) {
	plain := &stubStore{}
	if SupportsStats(plain) {
		t.Error("plain store should not support stats")
	}

	stats := &stubStatsStore{}
	if !SupportsStats(stats) {
		t.Error("stubStatsStore should support stats")
	}
}

func TestAsStatsProvider(t *testing.T) {
	plain := &stubStore{}
	if p := AsStatsProvider(plain); p != nil {
		t.Error("AsStatsProvider should return nil for plain store")
	}

	stats := &stubStatsStore{}
	if p := AsStatsProvider(stats); p == nil {
		t.Error("AsStatsProvider should return non-nil for stats store")
	}
}

func TestGetBucketStatsHelper(t *testing.T) {
	stats := &stubStatsStore{}
	result, err := GetBucketStats(stats, "test-bucket")
	if err != nil {
		t.Fatalf("GetBucketStats failed: %v", err)
	}
	if result.Bucket != "test-bucket" {
		t.Errorf("bucket = %q, want %q", result.Bucket, "test-bucket")
	}
	if result.Size != 100 {
		t.Errorf("size = %d, want 100", result.Size)
	}
	if result.ObjectCount != 5 {
		t.Errorf("objectCount = %d, want 5", result.ObjectCount)
	}
}

func TestGetBucketStatsUnsupported(t *testing.T) {
	plain := &stubStore{}
	_, err := GetBucketStats(plain, "test-bucket")
	if err != ErrStatsUnsupported {
		t.Errorf("expected ErrStatsUnsupported, got %v", err)
	}
}

func TestGetCDNStatsHelper(t *testing.T) {
	stats := &stubStatsStore{}
	result, err := GetCDNStats(stats, &CDNStatsRequest{
		Range: TimeRange{Start: time.Now().Add(-time.Hour), End: time.Now()},
	})
	if err != nil {
		t.Fatalf("GetCDNStats failed: %v", err)
	}
	if len(result.Points) != 1 {
		t.Errorf("points = %d, want 1", len(result.Points))
	}
}

func TestGetCDNStatsUnsupported(t *testing.T) {
	plain := &stubStore{}
	_, err := GetCDNStats(plain, &CDNStatsRequest{})
	if err != ErrStatsUnsupported {
		t.Errorf("expected ErrStatsUnsupported, got %v", err)
	}
}

func TestGetAPIRequestStatsHelper(t *testing.T) {
	stats := &stubStatsStore{}
	_, err := GetAPIRequestStats(stats, &APIStatsRequest{})
	if err != nil {
		t.Fatalf("GetAPIRequestStats failed: %v", err)
	}
}

func TestGetAPIRequestStatsUnsupported(t *testing.T) {
	plain := &stubStore{}
	_, err := GetAPIRequestStats(plain, &APIStatsRequest{})
	if err != ErrStatsUnsupported {
		t.Errorf("expected ErrStatsUnsupported, got %v", err)
	}
}

func TestGetOriginFetchStatsHelper(t *testing.T) {
	stats := &stubStatsStore{}
	_, err := GetOriginFetchStats(stats, &OriginStatsRequest{})
	if err != nil {
		t.Fatalf("GetOriginFetchStats failed: %v", err)
	}
}

func TestGetOriginFetchStatsUnsupported(t *testing.T) {
	plain := &stubStore{}
	_, err := GetOriginFetchStats(plain, &OriginStatsRequest{})
	if err != ErrStatsUnsupported {
		t.Errorf("expected ErrStatsUnsupported, got %v", err)
	}
}

func TestErrStatsUnsupported(t *testing.T) {
	if ErrStatsUnsupported.Code != 501 {
		t.Errorf("code = %d, want 501", ErrStatsUnsupported.Code)
	}
	if ErrStatsUnsupported.Error() != "statistics not supported by this storage backend" {
		t.Errorf("Error() = %q", ErrStatsUnsupported.Error())
	}
}
