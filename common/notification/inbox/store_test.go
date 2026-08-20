// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package inbox

// Compile-time interface checks.
var _ Store = (*MemoryStore)(nil)
