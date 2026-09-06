// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package vad

import "fmt"

// sprintfImpl wraps fmt.Sprintf for use by the energy detector's log formatting.
func sprintfImpl(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}
