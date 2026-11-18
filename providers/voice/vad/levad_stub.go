//go:build !levad

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package vad

import (
	"errors"
	"time"
)

var errLevadNotBuilt = errors.New("levad: rebuild with -tags levad (and make build-levad)")

// LevadEnabled reports whether the native Levad library is linked in.
func LevadEnabled() bool { return false }

// LevadVersion returns an empty string when Levad is not built.
func LevadVersion() string { return "" }

const (
	LevadKindSilero uint8 = 0
	LevadKindTen    uint8 = 1
)

// LevadDetector is unavailable unless built with -tags levad.
type LevadDetector struct{}

// NewLevadDetector returns an error when Levad is not compiled in.
func NewLevadDetector(_ uint8, _ int, _ float32) (*LevadDetector, error) {
	return nil, errLevadNotBuilt
}

// NewTinySileroDetector returns an error when Levad is not compiled in.
func NewTinySileroDetector(_ int) (*LevadDetector, error) {
	return nil, errLevadNotBuilt
}

// NewTinyTenDetector returns an error when Levad is not compiled in.
func NewTinyTenDetector(_ int) (*LevadDetector, error) {
	return nil, errLevadNotBuilt
}

func (v *LevadDetector) Kind() EngineKind   { return EngineTinySilero }
func (v *LevadDetector) SampleRate() int    { return 16000 }
func (v *LevadDetector) FrameDuration() int { return 32 }
func (v *LevadDetector) Reset()             {}
func (v *LevadDetector) Close() error       { return nil }

func (v *LevadDetector) ProcessFrame(_ []byte) (FrameResult, error) {
	return FrameResult{Timestamp: time.Now()}, errLevadNotBuilt
}

func (v *LevadDetector) IsSpeech(_ []byte) (bool, error) {
	return false, errLevadNotBuilt
}
