// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package backup

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

// ReaderSource wraps an in-memory or pre-built reader factory.
type ReaderSource struct {
	// OpenFunc returns a new reader for each backup.
	OpenFunc func(ctx context.Context) (io.ReadCloser, error)
}

// Open implements [Source].
func (s *ReaderSource) Open(ctx context.Context) (io.ReadCloser, error) {
	if s == nil || s.OpenFunc == nil {
		return nil, fmt.Errorf("backup: nil reader source")
	}
	return s.OpenFunc(ctx)
}

// BytesSource is a [Source] backed by a fixed byte slice.
type BytesSource struct {
	Data []byte
}

// Open implements [Source].
func (s *BytesSource) Open(_ context.Context) (io.ReadCloser, error) {
	if s == nil {
		return nil, fmt.Errorf("backup: nil bytes source")
	}
	return io.NopCloser(bytes.NewReader(s.Data)), nil
}

// WriterSink writes restore bytes to an io.Writer.
type WriterSink struct {
	W io.Writer
}

// Write implements [Sink].
func (s *WriterSink) Write(_ context.Context, r io.Reader) error {
	if s == nil || s.W == nil {
		return fmt.Errorf("backup: nil writer sink")
	}
	_, err := io.Copy(s.W, r)
	return err
}

// BufferSink accumulates restore bytes in memory.
type BufferSink struct {
	Buf *bytes.Buffer
}

// Write implements [Sink].
func (s *BufferSink) Write(_ context.Context, r io.Reader) error {
	if s == nil {
		return fmt.Errorf("backup: nil buffer sink")
	}
	if s.Buf == nil {
		s.Buf = &bytes.Buffer{}
	}
	_, err := io.Copy(s.Buf, r)
	return err
}
