//go:build !cgo

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package vad

import (
	"errors"
	"fmt"
)

var errWebRTCVADNoCGO = errors.New("webrtc vad: unavailable when built with CGO_ENABLED=0")

// WebRTCDetector is a stub when CGO is disabled.
type WebRTCDetector struct {
	sampleRate int
	frameMs    int
	mode       int
}

// NewWebRTCDetector returns an error when the binary was built without CGO.
func NewWebRTCDetector(sampleRate, frameMs, mode int) (*WebRTCDetector, error) {
	return nil, errWebRTCVADNoCGO
}

func (w *WebRTCDetector) Kind() EngineKind   { return EngineWebRTC }
func (w *WebRTCDetector) SampleRate() int    { return w.sampleRate }
func (w *WebRTCDetector) FrameDuration() int { return w.frameMs }
func (w *WebRTCDetector) Close() error       { return nil }

func (w *WebRTCDetector) ProcessFrame(pcm []byte) (FrameResult, error) {
	return FrameResult{}, errWebRTCVADNoCGO
}

func (w *WebRTCDetector) Reset() {}

func (w *WebRTCDetector) SetMode(mode int) error {
	return fmt.Errorf("%w", errWebRTCVADNoCGO)
}

func (w *WebRTCDetector) Mode() int { return w.mode }
