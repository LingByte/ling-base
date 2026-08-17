// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package stores

import (
	"io"
	"time"
)

// ──────────────────────────────────────────────
// ObjectStorageManager — bucket & object management
// ──────────────────────────────────────────────

// ObjectStorageManager extends Store with administrative operations for
// managing buckets, listing objects with metadata, copying/moving
// objects, and generating expiring URLs. Not all backends implement
// every operation — callers should check with a type assertion or use
// the helper functions.
//
// The bucket parameter may be empty for backends that operate on a
// single pre-configured bucket (e.g. the local store or a single-bucket
// cloud config); in that case the backend uses its default bucket.
type ObjectStorageManager interface {
	Store

	// ── Bucket management ──

	// ListBuckets returns all buckets the configured credentials can
	// access, optionally filtered by the request parameters.
	ListBuckets(req *ListBucketsRequest) (*ListBucketsResponse, error)

	// CreateBucket creates a new bucket.
	CreateBucket(req *CreateBucketRequest) error

	// DeleteBucket deletes an empty bucket. Returns an error if the
	// bucket is not empty or does not exist.
	DeleteBucket(bucket string) error

	// GetBucketInfo returns metadata about a bucket.
	GetBucketInfo(bucket string) (*BucketInfo, error)

	// SetBucketPrivate sets the access control of a bucket.
	SetBucketPrivate(bucket string, isPrivate bool) error

	// GetBucketDomains returns the domain names bound to a bucket.
	GetBucketDomains(bucket string) ([]string, error)

	// ── Object management ──

	// ListFiles lists objects in a bucket with optional prefix,
	// delimiter, and pagination.
	ListFiles(bucket string, req *ListFilesRequest) (*ListFilesResponse, error)

	// GetFileInfo returns metadata for a single object.
	GetFileInfo(bucket, key string) (*FileInfo, error)

	// UploadFile uploads data to a bucket. Unlike Write, this method
	// accepts an explicit size hint and bucket parameter.
	UploadFile(bucket, key string, reader io.Reader, size int64) error

	// DeleteFile deletes an object from a bucket.
	DeleteFile(bucket, key string) error

	// CopyFile copies an object within or across buckets.
	CopyFile(req *CopyObjectRequest) error

	// MoveFile moves (renames) an object within or across buckets.
	MoveFile(req *CopyObjectRequest) error

	// GetFileURL returns an expiring URL for accessing a (possibly
	// private) object.
	GetFileURL(bucket, key string, expires time.Duration) (string, error)
}

// ──────────────────────────────────────────────
// MultipartUploader — large file multipart upload
// ──────────────────────────────────────────────

// MultipartUploader is implemented by backends that support multipart
// upload for large files. This allows uploading files in parts,
// resuming interrupted uploads, and parallel part uploads.
type MultipartUploader interface {
	// InitiateMultipartUpload starts a multipart upload and returns
	// an upload ID.
	InitiateMultipartUpload(bucket string, req *InitiateMultipartUploadRequest) (*InitiateMultipartUploadResponse, error)

	// UploadPart uploads a single part of a multipart upload.
	UploadPart(bucket, key string, req *UploadPartRequest) (*UploadPartResponse, error)

	// CompleteMultipartUpload finalizes a multipart upload by combining
	// all uploaded parts.
	CompleteMultipartUpload(bucket, key string, req *CompleteMultipartUploadRequest) (*CompleteMultipartUploadResponse, error)

	// AbortMultipartUpload cancels a multipart upload and discards
	// uploaded parts.
	AbortMultipartUpload(bucket, key, uploadID string) error

	// ListParts lists the parts that have been uploaded for a multipart
	// upload.
	ListParts(bucket, key string, req *ListPartsRequest) (*ListPartsResponse, error)
}

// ──────────────────────────────────────────────
// Helper functions
// ──────────────────────────────────────────────

// AsManager returns the given store as an ObjectStorageManager, or nil
// if the store does not implement the management interface.
func AsManager(s Store) ObjectStorageManager {
	if m, ok := s.(ObjectStorageManager); ok {
		return m
	}
	return nil
}

// AsMultipartUploader returns the given store as a MultipartUploader,
// or nil if the store does not support multipart upload.
func AsMultipartUploader(s Store) MultipartUploader {
	if m, ok := s.(MultipartUploader); ok {
		return m
	}
	return nil
}

// SupportsManagement reports whether the store implements
// ObjectStorageManager.
func SupportsManagement(s Store) bool {
	_, ok := s.(ObjectStorageManager)
	return ok
}

// SupportsMultipart reports whether the store implements
// MultipartUploader.
func SupportsMultipart(s Store) bool {
	_, ok := s.(MultipartUploader)
	return ok
}
