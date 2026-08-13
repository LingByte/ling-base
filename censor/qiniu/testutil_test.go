// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package qiniu

import "context"

// contextWithCancel returns a pre-canceled context for testing request errors.
func contextWithCancel() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx, cancel
}
