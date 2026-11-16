// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package backup is a pluggable logical backup/restore orchestrator.
//
// It streams data from a [Source] to a [Destination], optionally gzip-
// compressing and always SHA-256 checksumming the stored bytes. Restore
// reads the artifact back via [Destination.Open] and writes it to a [Sink].
//
// Built-in sources cover local files and CLI dumps (mysqldump, pg_dump,
// mongodump) — the same dump path works for self-hosted and cloud databases
// whenever the host is reachable with credentials. Cloud vendor snapshot
// APIs are intentionally out of scope; implement [Source]/[Destination]
// yourself if you need them.
package backup

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"
)

// Option configures a [Manager].
type Option func(*Manager)

// WithCompression enables gzip compression of the stored artifact.
func WithCompression() Option {
	return func(m *Manager) { m.compressed = true }
}

// WithCatalog sets the backup metadata store. Default is an in-memory catalog.
func WithCatalog(c Catalog) Option {
	return func(m *Manager) {
		if c != nil {
			m.catalog = c
		}
	}
}

// WithRetention sets the default retention policy used by [Manager.Prune].
func WithRetention(r Retention) Option {
	return func(m *Manager) { m.retention = r }
}

// WithSink sets the default restore target used when [Manager.Restore]
// is called with a nil sink.
func WithSink(s Sink) Option {
	return func(m *Manager) { m.sink = s }
}

// WithClock injects a clock (tests).
func WithClock(now func() time.Time) Option {
	return func(m *Manager) {
		if now != nil {
			m.now = now
		}
	}
}

// Manager coordinates backup and restore between a [Source] and [Destination].
type Manager struct {
	src        Source
	dst        Destination
	sink       Sink
	catalog    Catalog
	retention  Retention
	compressed bool
	now        func() time.Time
}

// New creates a Manager. Catalog defaults to [NewMemoryCatalog].
func New(src Source, dst Destination, opts ...Option) *Manager {
	m := &Manager{
		src:     src,
		dst:     dst,
		catalog: NewMemoryCatalog(),
		now:     time.Now,
	}
	for _, o := range opts {
		o(m)
	}
	if m.catalog == nil {
		m.catalog = NewMemoryCatalog()
	}
	if m.now == nil {
		m.now = time.Now
	}
	return m
}

// NewManager is an alias of [New] kept for callers migrating from the old API.
func NewManager(src Source, dst Destination, opts ...Option) *Manager {
	return New(src, dst, opts...)
}

// Backup streams the source to the destination and records metadata.
func (m *Manager) Backup(ctx context.Context, name string) (*Backup, error) {
	if m == nil || m.src == nil {
		return nil, ErrNilSource
	}
	if m.dst == nil {
		return nil, ErrNilDest
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, ErrEmptyName
	}

	srcReader, err := m.src.Open(ctx)
	if err != nil {
		return nil, fmt.Errorf("backup: open source: %w", err)
	}
	defer func() { _ = srcReader.Close() }()

	pr, pw := io.Pipe()
	done := make(chan struct{})
	var copyErr error
	go func() {
		defer close(done)
		var w io.WriteCloser = pw
		if m.compressed {
			gz := gzip.NewWriter(pw)
			w = &gzipWriteCloser{gz: gz, pipe: pw}
		} else {
			defer func() { _ = pw.Close() }()
		}
		copyErr = copyCtx(ctx, w, srcReader)
		if m.compressed {
			_ = w.Close()
		}
		if copyErr != nil {
			_ = pw.CloseWithError(copyErr)
		}
	}()

	hasher := sha256.New()
	var size int64
	tee := io.TeeReader(pr, io.MultiWriter(hasher, &countWriter{n: &size}))

	key, err := m.dst.Write(ctx, name, tee)
	if err != nil {
		_ = pr.CloseWithError(err)
		<-done
		return nil, fmt.Errorf("backup: write destination: %w", err)
	}
	<-done
	if copyErr != nil {
		return nil, fmt.Errorf("backup: read source: %w", copyErr)
	}

	bp := &Backup{
		Name:       name,
		Key:        key,
		CreatedAt:  m.now(),
		Size:       size,
		Checksum:   hex.EncodeToString(hasher.Sum(nil)),
		Compressed: m.compressed,
	}
	if err := m.catalog.Put(ctx, bp); err != nil {
		return nil, fmt.Errorf("backup: catalog put: %w", err)
	}
	return bp, nil
}

type gzipWriteCloser struct {
	gz   *gzip.Writer
	pipe *io.PipeWriter
}

func (g *gzipWriteCloser) Write(p []byte) (int, error) { return g.gz.Write(p) }

func (g *gzipWriteCloser) Close() error {
	err1 := g.gz.Close()
	err2 := g.pipe.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

type countWriter struct{ n *int64 }

func (c *countWriter) Write(p []byte) (int, error) {
	*c.n += int64(len(p))
	return len(p), nil
}

// Restore loads a backup artifact into sink. If sink is nil, the manager's
// default sink from [WithSink] is used.
func (m *Manager) Restore(ctx context.Context, bp *Backup, sink Sink) error {
	if m == nil || m.dst == nil {
		return ErrNilDest
	}
	if bp == nil {
		return ErrNilBackup
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if sink == nil {
		sink = m.sink
	}
	if sink == nil {
		return ErrNoSink
	}
	key := bp.Key
	if key == "" {
		key = bp.Name
	}
	rc, err := m.dst.Open(ctx, key)
	if err != nil {
		return fmt.Errorf("backup: open artifact: %w", err)
	}
	defer func() { _ = rc.Close() }()

	var r io.Reader = rc
	if bp.Compressed {
		gz, err := gzip.NewReader(rc)
		if err != nil {
			return fmt.Errorf("backup: open gzip: %w", err)
		}
		defer func() { _ = gz.Close() }()
		r = gz
	}
	if err := sink.Write(ctx, r); err != nil {
		return fmt.Errorf("backup: write sink: %w", err)
	}
	return nil
}

// Verify recomputes the SHA-256 of the stored artifact and compares it
// to bp.Checksum.
func (m *Manager) Verify(ctx context.Context, bp *Backup) error {
	if m == nil || m.dst == nil {
		return ErrNilDest
	}
	if bp == nil {
		return ErrNilBackup
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	key := bp.Key
	if key == "" {
		key = bp.Name
	}
	rc, err := m.dst.Open(ctx, key)
	if err != nil {
		return fmt.Errorf("backup: open artifact: %w", err)
	}
	defer func() { _ = rc.Close() }()

	hasher := sha256.New()
	if err := copyCtx(ctx, hasher, rc); err != nil {
		return fmt.Errorf("backup: hash artifact: %w", err)
	}
	sum := hex.EncodeToString(hasher.Sum(nil))
	if sum != bp.Checksum {
		return fmt.Errorf("%w: got %s want %s", ErrChecksumMismatch, sum, bp.Checksum)
	}
	return nil
}

// List returns catalogued backups sorted by name.
func (m *Manager) List(ctx context.Context) ([]*Backup, error) {
	if m == nil || m.catalog == nil {
		return nil, ErrNotFound
	}
	return m.catalog.List(ctx)
}

// Get returns a named backup from the catalog.
func (m *Manager) Get(ctx context.Context, name string) (*Backup, error) {
	if m == nil || m.catalog == nil {
		return nil, ErrNotFound
	}
	return m.catalog.Get(ctx, name)
}

// Delete removes the catalog entry and the artifact from the destination.
func (m *Manager) Delete(ctx context.Context, name string) error {
	if m == nil || m.dst == nil {
		return ErrNilDest
	}
	bp, err := m.catalog.Get(ctx, name)
	if err != nil {
		return err
	}
	key := bp.Key
	if key == "" {
		key = bp.Name
	}
	if err := m.dst.Remove(ctx, key); err != nil {
		return fmt.Errorf("backup: remove artifact: %w", err)
	}
	if err := m.catalog.Delete(ctx, name); err != nil {
		return fmt.Errorf("backup: catalog delete: %w", err)
	}
	return nil
}

// Prune deletes backups that violate the retention policy (manager default
// or the override argument). Returns the names that were removed.
func (m *Manager) Prune(ctx context.Context, override Retention) ([]string, error) {
	if m == nil {
		return nil, ErrNilSource
	}
	policy := override
	if policy == nil {
		policy = m.retention
	}
	if policy == nil {
		return nil, nil
	}
	list, err := m.catalog.List(ctx)
	if err != nil {
		return nil, err
	}
	victims := policy.Victims(list, m.now())
	removed := make([]string, 0, len(victims))
	for _, name := range victims {
		if err := m.Delete(ctx, name); err != nil {
			return removed, err
		}
		removed = append(removed, name)
	}
	return removed, nil
}

// copyCtx copies src to dst, aborting when ctx is cancelled.
func copyCtx(ctx context.Context, dst io.Writer, src io.Reader) error {
	buf := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			if ew != nil {
				return ew
			}
			if nr != nw {
				return io.ErrShortWrite
			}
		}
		if errors.Is(er, io.EOF) {
			return nil
		}
		if er != nil {
			return er
		}
	}
}
