// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package stores

import (
	"io"
	"testing"
	"time"
)

// mockManagerStore implements both Store and ObjectStorageManager.
type mockManagerStore struct {
	fallbackStore
	buckets []BucketInfo
	files   []FileInfo
}

func (m *mockManagerStore) ListBuckets(req *ListBucketsRequest) (*ListBucketsResponse, error) {
	return &ListBucketsResponse{Buckets: m.buckets}, nil
}
func (m *mockManagerStore) CreateBucket(req *CreateBucketRequest) error       { return nil }
func (m *mockManagerStore) DeleteBucket(bucket string) error                  { return nil }
func (m *mockManagerStore) GetBucketInfo(bucket string) (*BucketInfo, error)  { return &BucketInfo{Name: bucket}, nil }
func (m *mockManagerStore) SetBucketPrivate(bucket string, p bool) error      { return nil }
func (m *mockManagerStore) GetBucketDomains(bucket string) ([]string, error)  { return []string{"a.com"}, nil }
func (m *mockManagerStore) ListFiles(bucket string, req *ListFilesRequest) (*ListFilesResponse, error) {
	return &ListFilesResponse{Files: m.files}, nil
}
func (m *mockManagerStore) GetFileInfo(bucket, key string) (*FileInfo, error) {
	return &FileInfo{Key: key, Size: 100}, nil
}
func (m *mockManagerStore) UploadFile(bucket, key string, r io.Reader, size int64) error { return nil }
func (m *mockManagerStore) DeleteFile(bucket, key string) error                          { return nil }
func (m *mockManagerStore) CopyFile(req *CopyObjectRequest) error                        { return nil }
func (m *mockManagerStore) MoveFile(req *CopyObjectRequest) error                        { return nil }
func (m *mockManagerStore) GetFileURL(bucket, key string, expires time.Duration) (string, error) {
	return "https://example.com/" + key, nil
}

// mockMultipartStore implements MultipartUploader.
type mockMultipartStore struct {
	fallbackStore
}

func (m *mockMultipartStore) InitiateMultipartUpload(bucket string, req *InitiateMultipartUploadRequest) (*InitiateMultipartUploadResponse, error) {
	return &InitiateMultipartUploadResponse{UploadID: "test-upload-id", Key: req.Key}, nil
}
func (m *mockMultipartStore) UploadPart(bucket, key string, req *UploadPartRequest) (*UploadPartResponse, error) {
	return &UploadPartResponse{ETag: "etag-1"}, nil
}
func (m *mockMultipartStore) CompleteMultipartUpload(bucket, key string, req *CompleteMultipartUploadRequest) (*CompleteMultipartUploadResponse, error) {
	return &CompleteMultipartUploadResponse{Bucket: bucket, Key: key, ETag: "final-etag"}, nil
}
func (m *mockMultipartStore) AbortMultipartUpload(bucket, key, uploadID string) error { return nil }
func (m *mockMultipartStore) ListParts(bucket, key string, req *ListPartsRequest) (*ListPartsResponse, error) {
	return &ListPartsResponse{Bucket: bucket, Key: key, UploadID: req.UploadID}, nil
}

func TestAsManager(t *testing.T) {
	// Store that implements ObjectStorageManager
	m := &mockManagerStore{buckets: []BucketInfo{{Name: "test"}}}
	got := AsManager(m)
	if got == nil {
		t.Fatal("AsManager returned nil for a manager store")
	}
	resp, err := got.ListBuckets(nil)
	if err != nil {
		t.Fatalf("ListBuckets error: %v", err)
	}
	if len(resp.Buckets) != 1 || resp.Buckets[0].Name != "test" {
		t.Errorf("unexpected buckets: %+v", resp.Buckets)
	}

	// Store that does NOT implement ObjectStorageManager
	s := &fallbackStore{}
	if AsManager(s) != nil {
		t.Error("AsManager should return nil for non-manager store")
	}
}

func TestAsMultipartUploader(t *testing.T) {
	m := &mockMultipartStore{}
	got := AsMultipartUploader(m)
	if got == nil {
		t.Fatal("AsMultipartUploader returned nil for a multipart store")
	}
	resp, err := got.InitiateMultipartUpload("bucket", &InitiateMultipartUploadRequest{Key: "test.txt"})
	if err != nil {
		t.Fatalf("InitiateMultipartUpload error: %v", err)
	}
	if resp.UploadID != "test-upload-id" {
		t.Errorf("UploadID = %q, want %q", resp.UploadID, "test-upload-id")
	}

	// Store that does NOT implement MultipartUploader
	s := &fallbackStore{}
	if AsMultipartUploader(s) != nil {
		t.Error("AsMultipartUploader should return nil for non-multipart store")
	}
}

func TestSupportsManagement(t *testing.T) {
	if !SupportsManagement(&mockManagerStore{}) {
		t.Error("SupportsManagement should return true for manager store")
	}
	if SupportsManagement(&fallbackStore{}) {
		t.Error("SupportsManagement should return false for non-manager store")
	}
}

func TestSupportsMultipart(t *testing.T) {
	if !SupportsMultipart(&mockMultipartStore{}) {
		t.Error("SupportsMultipart should return true for multipart store")
	}
	if SupportsMultipart(&fallbackStore{}) {
		t.Error("SupportsMultipart should return false for non-multipart store")
	}
}

func TestFileInfo_JSONRoundTrip(t *testing.T) {
	// Verify the types can be used without panicking.
	fi := FileInfo{
		Key:          "test.txt",
		Size:         1024,
		LastModified: time.Now(),
		ETag:         "abc123",
		ContentType:  "text/plain",
		Metadata:     map[string]string{"x": "y"},
		StorageClass: "STANDARD",
		PublicURL:    "https://example.com/test.txt",
	}
	if fi.Key != "test.txt" {
		t.Errorf("Key = %q", fi.Key)
	}
	if fi.Size != 1024 {
		t.Errorf("Size = %d", fi.Size)
	}
}

func TestBucketInfo_JSONRoundTrip(t *testing.T) {
	bi := BucketInfo{
		Name:         "my-bucket",
		Region:       "us-east-1",
		CreatedAt:    time.Now(),
		IsPrivate:    true,
		Domains:      []string{"cdn.example.com"},
		Tags:         map[string]string{"env": "prod"},
		StorageClass: "STANDARD",
		Versioning:   true,
	}
	if bi.Name != "my-bucket" {
		t.Errorf("Name = %q", bi.Name)
	}
	if !bi.IsPrivate {
		t.Error("IsPrivate should be true")
	}
}

func TestMultipartTypes(t *testing.T) {
	// Verify multipart types work correctly.
	initReq := InitiateMultipartUploadRequest{
		Key:         "large-file.bin",
		ContentType: "application/octet-stream",
		Metadata:    map[string]string{"author": "test"},
	}
	if initReq.Key != "large-file.bin" {
		t.Errorf("Key = %q", initReq.Key)
	}

	completeReq := CompleteMultipartUploadRequest{
		UploadID: "upload-123",
		Parts: []CompletedPart{
			{PartNumber: 1, ETag: "etag-1"},
			{PartNumber: 2, ETag: "etag-2"},
		},
	}
	if len(completeReq.Parts) != 2 {
		t.Errorf("Parts len = %d", len(completeReq.Parts))
	}
	if completeReq.Parts[0].PartNumber != 1 {
		t.Errorf("PartNumber = %d", completeReq.Parts[0].PartNumber)
	}
}
