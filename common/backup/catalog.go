// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// MemoryCatalog is an in-process catalog (lost on restart).
type MemoryCatalog struct {
	mu   sync.RWMutex
	data map[string]*Backup
}

// NewMemoryCatalog creates an empty memory catalog.
func NewMemoryCatalog() *MemoryCatalog {
	return &MemoryCatalog{data: make(map[string]*Backup)}
}

func (c *MemoryCatalog) Put(_ context.Context, bp *Backup) error {
	if bp == nil || bp.Name == "" {
		return ErrEmptyName
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := *bp
	if bp.Labels != nil {
		cp.Labels = make(map[string]string, len(bp.Labels))
		for k, v := range bp.Labels {
			cp.Labels[k] = v
		}
	}
	c.data[bp.Name] = &cp
	return nil
}

func (c *MemoryCatalog) Get(_ context.Context, name string) (*Backup, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	bp, ok := c.data[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrNotFound, name)
	}
	cp := *bp
	return &cp, nil
}

func (c *MemoryCatalog) List(_ context.Context) ([]*Backup, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]*Backup, 0, len(c.data))
	for _, bp := range c.data {
		cp := *bp
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (c *MemoryCatalog) Delete(_ context.Context, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.data[name]; !ok {
		return fmt.Errorf("%w: %q", ErrNotFound, name)
	}
	delete(c.data, name)
	return nil
}

// FileCatalog persists backups as a JSON file (atomic replace on write).
type FileCatalog struct {
	Path string

	mu sync.Mutex
}

type fileCatalogDoc struct {
	Backups []*Backup `json:"backups"`
}

func (c *FileCatalog) load() (map[string]*Backup, error) {
	data := make(map[string]*Backup)
	if c.Path == "" {
		return nil, fmt.Errorf("backup: empty catalog path")
	}
	b, err := os.ReadFile(c.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return nil, err
	}
	var doc fileCatalogDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("backup: catalog decode: %w", err)
	}
	for _, bp := range doc.Backups {
		if bp != nil && bp.Name != "" {
			data[bp.Name] = bp
		}
	}
	return data, nil
}

func (c *FileCatalog) save(data map[string]*Backup) error {
	if c.Path == "" {
		return fmt.Errorf("backup: empty catalog path")
	}
	if err := os.MkdirAll(filepath.Dir(c.Path), 0o755); err != nil {
		return err
	}
	doc := fileCatalogDoc{Backups: make([]*Backup, 0, len(data))}
	for _, bp := range data {
		doc.Backups = append(doc.Backups, bp)
	}
	sort.Slice(doc.Backups, func(i, j int) bool { return doc.Backups[i].Name < doc.Backups[j].Name })
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	tmp := c.Path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, c.Path)
}

func (c *FileCatalog) Put(_ context.Context, bp *Backup) error {
	if bp == nil || bp.Name == "" {
		return ErrEmptyName
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := c.load()
	if err != nil {
		return err
	}
	cp := *bp
	if bp.Labels != nil {
		cp.Labels = make(map[string]string, len(bp.Labels))
		for k, v := range bp.Labels {
			cp.Labels[k] = v
		}
	}
	data[bp.Name] = &cp
	return c.save(data)
}

func (c *FileCatalog) Get(_ context.Context, name string) (*Backup, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := c.load()
	if err != nil {
		return nil, err
	}
	bp, ok := data[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrNotFound, name)
	}
	cp := *bp
	return &cp, nil
}

func (c *FileCatalog) List(_ context.Context) ([]*Backup, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := c.load()
	if err != nil {
		return nil, err
	}
	out := make([]*Backup, 0, len(data))
	for _, bp := range data {
		cp := *bp
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (c *FileCatalog) Delete(_ context.Context, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := c.load()
	if err != nil {
		return err
	}
	if _, ok := data[name]; !ok {
		return fmt.Errorf("%w: %q", ErrNotFound, name)
	}
	delete(data, name)
	return c.save(data)
}
