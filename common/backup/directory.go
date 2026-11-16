// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DirectorySource archives a filesystem directory as a gzip tar stream.
// Useful for app data dirs, uploads, and config trees.
type DirectorySource struct {
	// Root is the directory to archive.
	Root string
	// Compress enables gzip (default true when zero-value Compress is unused —
	// set DisableGzip to skip compression at the source; Manager can still gzip).
	DisableGzip bool
}

// Open walks Root and returns a tar(.gz) stream.
func (s *DirectorySource) Open(ctx context.Context) (io.ReadCloser, error) {
	if s == nil || s.Root == "" {
		return nil, fmt.Errorf("backup: empty directory root")
	}
	info, err := os.Stat(s.Root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("backup: root is not a directory: %s", s.Root)
	}

	pr, pw := io.Pipe()
	go func() {
		var err error
		defer func() {
			if err != nil {
				_ = pw.CloseWithError(err)
			} else {
				_ = pw.Close()
			}
		}()
		if err = ctx.Err(); err != nil {
			return
		}
		var w io.Writer = pw
		var gzw *gzip.Writer
		if !s.DisableGzip {
			gzw = gzip.NewWriter(pw)
			w = gzw
			defer func() { _ = gzw.Close() }()
		}
		tw := tar.NewWriter(w)
		defer func() { _ = tw.Close() }()

		err = filepath.Walk(s.Root, func(path string, fi os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			rel, err := filepath.Rel(s.Root, path)
			if err != nil {
				return err
			}
			if rel == "." {
				return nil
			}
			hdr, err := tar.FileInfoHeader(fi, "")
			if err != nil {
				return err
			}
			hdr.Name = filepath.ToSlash(rel)
			if fi.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(path)
				if err != nil {
					return err
				}
				hdr.Linkname = target
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			if !fi.Mode().IsRegular() {
				return nil
			}
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(tw, f)
			_ = f.Close()
			return copyErr
		})
	}()
	return pr, nil
}

// DirectorySink extracts a tar(.gz) stream into Dest.
type DirectorySink struct {
	Dest string
	// ClearDest removes Dest contents before extract when true.
	ClearDest bool
}

// Write extracts r into Dest.
func (s *DirectorySink) Write(ctx context.Context, r io.Reader) error {
	if s == nil || s.Dest == "" {
		return fmt.Errorf("backup: empty directory dest")
	}
	if s.ClearDest {
		if err := os.RemoveAll(s.Dest); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(s.Dest, 0o755); err != nil {
		return err
	}

	var tr *tar.Reader
	buf := make([]byte, 2)
	n, _ := io.ReadFull(r, buf)
	rest := io.MultiReader(bytes.NewReader(buf[:n]), r)
	if n == 2 && buf[0] == 0x1f && buf[1] == 0x8b {
		gz, err := gzip.NewReader(rest)
		if err != nil {
			return fmt.Errorf("backup: gzip: %w", err)
		}
		defer func() { _ = gz.Close() }()
		tr = tar.NewReader(gz)
	} else {
		tr = tar.NewReader(rest)
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := safeJoin(s.Dest, hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			_, err = io.Copy(f, tr)
			_ = f.Close()
			if err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		default:
			// skip other types
		}
	}
}

func safeJoin(root, name string) (string, error) {
	clean := filepath.Clean("/" + name)
	rel := strings.TrimPrefix(clean, "/")
	full := filepath.Join(root, rel)
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	absFull, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	if absFull != absRoot && !strings.HasPrefix(absFull, absRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("backup: path escapes root: %s", name)
	}
	return full, nil
}
