// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package response

import "sync"

// ErrorMapper maps a domain error to *AppError. Return ok=false to skip.
type ErrorMapper func(err error) (ae *AppError, ok bool)

var (
	mapperMu sync.RWMutex
	mappers  []ErrorMapper
)

// RegisterErrorMapper appends a host-app mapper consulted by From.
// Safe for init()-time registration from the embedding application.
func RegisterErrorMapper(fn ErrorMapper) {
	if fn == nil {
		return
	}
	mapperMu.Lock()
	mappers = append(mappers, fn)
	mapperMu.Unlock()
}

func errorMappers() []ErrorMapper {
	mapperMu.RLock()
	defer mapperMu.RUnlock()
	if len(mappers) == 0 {
		return nil
	}
	out := make([]ErrorMapper, len(mappers))
	copy(out, mappers)
	return out
}
