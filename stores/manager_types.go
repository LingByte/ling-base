// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package stores

import (
	"io"
	"time"
)

// ──────────────────────────────────────────────
// Bucket management types
// ──────────────────────────────────────────────

// BucketInfo holds metadata about a storage bucket.
type BucketInfo struct {
	Name         string            `json:"name"`         // bucket name
	Region       string            `json:"region"`       // bucket region
	CreatedAt    time.Time         `json:"createdAt"`    // creation time
	IsPrivate    bool              `json:"isPrivate"`    // whether the bucket is private
	Domains      []string          `json:"domains"`      // bound domain names
	Tags         map[string]string `json:"tags"`         // bucket tags
	StorageClass string            `json:"storageClass"` // storage class (e.g. STANDARD)
	Versioning   bool              `json:"versioning"`   // whether versioning is enabled
}

// ListBucketsRequest holds parameters for listing buckets.
type ListBucketsRequest struct {
	Region string `json:"region,omitempty"` // filter by region
	Prefix string `json:"prefix,omitempty"` // filter by name prefix
	MaxKeys int   `json:"maxKeys,omitempty"`// max buckets to return (0 = all)
}

// ListBucketsResponse holds the result of listing buckets.
type ListBucketsResponse struct {
	Buckets     []BucketInfo `json:"buckets"`
	IsTruncated bool         `json:"isTruncated"`
	NextMarker  string       `json:"nextMarker,omitempty"`
}

// CreateBucketRequest holds parameters for creating a bucket.
type CreateBucketRequest struct {
	Name         string            `json:"name"`
	Region       string            `json:"region,omitempty"`
	StorageClass string            `json:"storageClass,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
	IsPrivate    bool              `json:"isPrivate"`
}

// ──────────────────────────────────────────────
// File/object management types
// ──────────────────────────────────────────────

// FileInfo holds metadata about a stored object.
type FileInfo struct {
	Key          string            `json:"key"`          // object key
	Size         int64             `json:"size"`         // object size in bytes
	LastModified time.Time         `json:"lastModified"` // last modification time
	ETag         string            `json:"etag"`         // ETag / hash
	ContentType  string            `json:"contentType"`  // MIME type
	Metadata     map[string]string `json:"metadata"`     // user-defined metadata
	StorageClass string            `json:"storageClass"` // storage class
	VersionID    string            `json:"versionId"`    // version ID (if versioning enabled)
	IsLatest     bool              `json:"isLatest"`     // whether this is the latest version
	PublicURL    string            `json:"publicURL"`    // public access URL
}

// ListFilesRequest holds parameters for listing objects in a bucket.
type ListFilesRequest struct {
	Prefix    string `json:"prefix,omitempty"`    // prefix filter
	Marker    string `json:"marker,omitempty"`    // pagination marker
	Limit     int    `json:"limit,omitempty"`     // max objects to return
	Delimiter string `json:"delimiter,omitempty"` // directory delimiter (e.g. "/")
}

// ListFilesResponse holds the result of listing objects.
type ListFilesResponse struct {
	Files          []FileInfo `json:"files"`
	Marker         string     `json:"marker,omitempty"`
	IsTruncated    bool       `json:"isTruncated"`
	CommonPrefixes []string   `json:"commonPrefixes,omitempty"` // virtual directories
}

// CopyObjectRequest holds parameters for copying an object.
type CopyObjectRequest struct {
	SrcBucket    string `json:"srcBucket"`
	SrcKey       string `json:"srcKey"`
	DestBucket   string `json:"destBucket"`
	DestKey      string `json:"destKey"`
	ContentType  string `json:"contentType,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ──────────────────────────────────────────────
// Multipart upload types
// ──────────────────────────────────────────────

// InitiateMultipartUploadRequest holds parameters for initiating a
// multipart upload.
type InitiateMultipartUploadRequest struct {
	Key         string            `json:"key"`
	ContentType string            `json:"contentType,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// InitiateMultipartUploadResponse holds the result of initiating a
// multipart upload.
type InitiateMultipartUploadResponse struct {
	UploadID string `json:"uploadId"`
	Key      string `json:"key"`
}

// UploadPartRequest holds parameters for uploading a single part.
type UploadPartRequest struct {
	UploadID   string    `json:"uploadId"`
	PartNumber int       `json:"partNumber"`
	Body       io.Reader `json:"-"`
}

// UploadPartResponse holds the result of uploading a part.
type UploadPartResponse struct {
	ETag string `json:"etag"`
}

// CompletedPart represents a single completed part in a multipart upload.
type CompletedPart struct {
	PartNumber int    `json:"partNumber"`
	ETag       string `json:"etag"`
}

// CompleteMultipartUploadRequest holds parameters for completing a
// multipart upload.
type CompleteMultipartUploadRequest struct {
	UploadID string          `json:"uploadId"`
	Parts    []CompletedPart `json:"parts"`
}

// CompleteMultipartUploadResponse holds the result of completing a
// multipart upload.
type CompleteMultipartUploadResponse struct {
	Location string `json:"location"`
	Bucket   string `json:"bucket"`
	Key      string `json:"key"`
	ETag     string `json:"etag"`
}

// ListPartsRequest holds parameters for listing parts of a multipart upload.
type ListPartsRequest struct {
	UploadID         string `json:"uploadId"`
	MaxParts         int    `json:"maxParts,omitempty"`
	PartNumberMarker int    `json:"partNumberMarker,omitempty"`
}

// ListPartsResponse holds the result of listing parts.
type ListPartsResponse struct {
	Bucket               string          `json:"bucket"`
	Key                  string          `json:"key"`
	UploadID             string          `json:"uploadId"`
	MaxParts             int             `json:"maxParts"`
	IsTruncated          bool            `json:"isTruncated"`
	PartNumberMarker     int             `json:"partNumberMarker"`
	NextPartNumberMarker int             `json:"nextPartNumberMarker"`
	Parts                []CompletedPart `json:"parts"`
}
