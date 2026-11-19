//go:build !lingcodec && !cgo

package encoder

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"

	"github.com/LingByte/ling-base/media"
)

var errOpusNoCGO = errors.New("opus: unavailable when built with CGO_ENABLED=0 (use CGO or -tags lingcodec)")

func createOPUSDecode(src, pcm media.CodecConfig) media.EncoderFunc {
	return func(pkt media.MediaPacket) ([]media.MediaPacket, error) {
		return nil, errOpusNoCGO
	}
}

func createOPUSEncode(src, pcm media.CodecConfig) media.EncoderFunc {
	return func(pkt media.MediaPacket) ([]media.MediaPacket, error) {
		return nil, errOpusNoCGO
	}
}
