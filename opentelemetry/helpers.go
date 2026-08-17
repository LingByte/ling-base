// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package opentelemetry

import (
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// setGlobalLoggerProvider registers the logger provider globally.
func setGlobalLoggerProvider(lp *sdklog.LoggerProvider) {
	if lp == nil {
		return
	}
	global.SetLoggerProvider(lp)
}
