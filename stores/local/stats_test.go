// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package local

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LingByte/ling-base/stores"
)

func TestGetBucketStats(t *testing.T) {
	dir := t.TempDir()
	s := New(Config{Root: dir})

	// Create a bucket with some files.
	bucketDir := filepath.Join(dir, "buckets", "test-bucket")
	if err := os.MkdirAll(bucketDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bucketDir, "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bucketDir, "b.txt"), []byte("world!"), 0644); err != nil {
		t.Fatal(err)
	}
	// Subdirectory with another file.
	subDir := filepath.Join(bucketDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "c.txt"), []byte("foo"), 0644); err != nil {
		t.Fatal(err)
	}

	stats, err := s.GetBucketStats("test-bucket")
	if err != nil {
		t.Fatalf("GetBucketStats failed: %v", err)
	}
	if stats.Bucket != "test-bucket" {
		t.Errorf("bucket = %q, want %q", stats.Bucket, "test-bucket")
	}
	if stats.Size != 14 { // "hello" (5) + "world!" (6) + "foo" (3) = 14
		t.Errorf("size = %d, want 14", stats.Size)
	}
	if stats.ObjectCount != 3 {
		t.Errorf("objectCount = %d, want 3", stats.ObjectCount)
	}
	if len(stats.StorageClasses) != 1 {
		t.Errorf("storageClasses len = %d, want 1", len(stats.StorageClasses))
	} else {
		sc := stats.StorageClasses[0]
		if sc.Class != "LOCAL" {
			t.Errorf("class = %q, want %q", sc.Class, "LOCAL")
		}
		if sc.Size != 14 {
			t.Errorf("class size = %d, want 14", sc.Size)
		}
		if sc.ObjectCount != 3 {
			t.Errorf("class objectCount = %d, want 3", sc.ObjectCount)
		}
	}
}

func TestGetBucketStats_NotExist(t *testing.T) {
	dir := t.TempDir()
	s := New(Config{Root: dir})

	_, err := s.GetBucketStats("nonexistent")
	if err != stores.ErrAttachmentNotExist {
		t.Errorf("expected ErrAttachmentNotExist, got %v", err)
	}
}

func TestGetBucketStats_EmptyBucket(t *testing.T) {
	dir := t.TempDir()
	s := New(Config{Root: dir})

	bucketDir := filepath.Join(dir, "buckets", "empty")
	if err := os.MkdirAll(bucketDir, 0755); err != nil {
		t.Fatal(err)
	}

	stats, err := s.GetBucketStats("empty")
	if err != nil {
		t.Fatalf("GetBucketStats failed: %v", err)
	}
	if stats.Size != 0 {
		t.Errorf("size = %d, want 0", stats.Size)
	}
	if stats.ObjectCount != 0 {
		t.Errorf("objectCount = %d, want 0", stats.ObjectCount)
	}
}

func TestGetCDNStats_Unsupported(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	_, err := s.GetCDNStats(&stores.CDNStatsRequest{})
	if err != stores.ErrStatsUnsupported {
		t.Errorf("expected ErrStatsUnsupported, got %v", err)
	}
}

func TestGetAPIRequestStats_Unsupported(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	_, err := s.GetAPIRequestStats(&stores.APIStatsRequest{})
	if err != stores.ErrStatsUnsupported {
		t.Errorf("expected ErrStatsUnsupported, got %v", err)
	}
}

func TestGetOriginFetchStats_Unsupported(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	_, err := s.GetOriginFetchStats(&stores.OriginStatsRequest{})
	if err != stores.ErrStatsUnsupported {
		t.Errorf("expected ErrStatsUnsupported, got %v", err)
	}
}

func TestLocalSupportsStats(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	if !stores.SupportsStats(s) {
		t.Error("local store should support stats")
	}
}
