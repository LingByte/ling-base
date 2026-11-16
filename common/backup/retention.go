// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package backup

import (
	"sort"
	"time"
)

// Retention selects which backups to delete during [Manager.Prune].
type Retention interface {
	// Victims returns backup names that should be removed.
	Victims(all []*Backup, now time.Time) []string
}

// KeepLastN keeps the N most recent backups (by CreatedAt); older ones are pruned.
type KeepLastN int

// Victims implements [Retention].
func (n KeepLastN) Victims(all []*Backup, _ time.Time) []string {
	if n <= 0 || len(all) <= int(n) {
		return nil
	}
	sorted := append([]*Backup(nil), all...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CreatedAt.After(sorted[j].CreatedAt)
	})
	out := make([]string, 0, len(sorted)-int(n))
	for _, bp := range sorted[int(n):] {
		out = append(out, bp.Name)
	}
	return out
}

// KeepNewerThan keeps backups created within the given duration before now.
type KeepNewerThan time.Duration

// Victims implements [Retention].
func (d KeepNewerThan) Victims(all []*Backup, now time.Time) []string {
	if d <= 0 {
		return nil
	}
	cutoff := now.Add(-time.Duration(d))
	var out []string
	for _, bp := range all {
		if bp.CreatedAt.Before(cutoff) {
			out = append(out, bp.Name)
		}
	}
	return out
}

// RetentionAll applies multiple policies; a backup is a victim if ANY
// policy marks it (union). Prefer composing carefully.
type RetentionAll []Retention

// Victims implements [Retention].
func (rs RetentionAll) Victims(all []*Backup, now time.Time) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, r := range rs {
		if r == nil {
			continue
		}
		for _, name := range r.Victims(all, now) {
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	return out
}
