// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package backup

import (
	"context"
	"errors"
	"io"
	"time"
)

var (
	// ErrNotFound is returned when a backup name is missing from the catalog.
	ErrNotFound = errors.New("backup: not found")
	// ErrEmptyName is returned when a backup name is empty.
	ErrEmptyName = errors.New("backup: empty name")
	// ErrNilBackup is returned when a nil *Backup is passed.
	ErrNilBackup = errors.New("backup: nil backup")
	// ErrChecksumMismatch is returned by Verify when digests differ.
	ErrChecksumMismatch = errors.New("backup: checksum mismatch")
	// ErrNoSink is returned when Restore is called without a Sink.
	ErrNoSink    = errors.New("backup: no sink configured")
	ErrNilSource = errors.New("backup: nil source")
	ErrNilDest   = errors.New("backup: nil destination")
)

// Backup describes a completed backup artifact.
type Backup struct {
	// Name is the logical name of the backup.
	Name string `json:"name"`
	// Key is the storage identifier returned by Destination.Write.
	Key string `json:"key"`
	// CreatedAt is when the backup was created.
	CreatedAt time.Time `json:"created_at"`
	// Size is the size of the stored (possibly compressed) artifact in bytes.
	Size int64 `json:"size"`
	// Checksum is the SHA-256 hex digest of the stored artifact bytes.
	Checksum string `json:"checksum"`
	// Compressed indicates whether gzip compression was applied.
	Compressed bool `json:"compressed"`
	// Labels are optional caller-supplied metadata.
	Labels map[string]string `json:"labels,omitempty"`
}

// Source produces a readable backup stream (file, dump command, …).
type Source interface {
	Open(ctx context.Context) (io.ReadCloser, error)
}

// Destination stores and retrieves backup artifacts.
type Destination interface {
	Write(ctx context.Context, name string, r io.Reader) (key string, err error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Remove(ctx context.Context, key string) error
}

// Sink receives restored data (file overwrite, mysql client, …).
type Sink interface {
	Write(ctx context.Context, r io.Reader) error
}

// Catalog persists backup metadata across process restarts.
type Catalog interface {
	Put(ctx context.Context, bp *Backup) error
	Get(ctx context.Context, name string) (*Backup, error)
	List(ctx context.Context) ([]*Backup, error)
	Delete(ctx context.Context, name string) error
}
