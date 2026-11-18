//go:build levad

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package vad

/*
#cgo CFLAGS: -I${SRCDIR}/native/levad/include
#cgo darwin LDFLAGS: ${SRCDIR}/native/levad/target/release/liblevad.a -ldl -lm -lpthread
#cgo linux LDFLAGS: ${SRCDIR}/native/levad/target/release/liblevad.a -ldl -lm -lpthread

#include "levad.h"
*/
import "C"

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"sync"
	"time"
	"unsafe"
)

// LevadEnabled reports whether the native Levad library is linked in.
func LevadEnabled() bool { return true }

// LevadVersion returns the native library version string.
func LevadVersion() string { return C.GoString(C.lv_version()) }

const (
	LevadKindSilero = uint8(C.LV_KIND_SILERO)
	LevadKindTen    = uint8(C.LV_KIND_TEN)
)

// LevadDetector wraps TinySilero or TinyTen from liblevad behind Detector.
type LevadDetector struct {
	h          *C.LvVad
	sampleRate int
	kind       uint8
	mu         sync.Mutex
}

// NewLevadDetector opens a native VAD instance (requires -tags levad and built liblevad).
func NewLevadDetector(kind uint8, sampleRate int, threshold float32) (*LevadDetector, error) {
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	h := C.lv_vad_open(C.uint8_t(kind), C.uint32_t(sampleRate), C.float(threshold))
	if h == nil {
		return nil, fmt.Errorf("levad: open kind=%d rate=%d failed (need 16kHz for TinyTen)", kind, sampleRate)
	}
	v := &LevadDetector{h: h, sampleRate: sampleRate, kind: kind}
	runtime.SetFinalizer(v, (*LevadDetector).Close)
	return v, nil
}

// NewTinySileroDetector creates TinySilero (512 samples @ 16kHz per decision).
func NewTinySileroDetector(sampleRate int) (*LevadDetector, error) {
	return NewLevadDetector(LevadKindSilero, sampleRate, 0.5)
}

// NewTinyTenDetector creates TinyTen (hop 256 @ 16kHz; requires 16kHz).
func NewTinyTenDetector(sampleRate int) (*LevadDetector, error) {
	if sampleRate > 0 && sampleRate != 16000 {
		return nil, fmt.Errorf("levad tinyten: only 16000 Hz supported, got %d", sampleRate)
	}
	return NewLevadDetector(LevadKindTen, 16000, 0.5)
}

func (v *LevadDetector) Kind() EngineKind {
	if v != nil && v.kind == LevadKindTen {
		return EngineTinyTen
	}
	return EngineTinySilero
}

func (v *LevadDetector) SampleRate() int {
	if v == nil {
		return 16000
	}
	return v.sampleRate
}

func (v *LevadDetector) FrameDuration() int { return 32 }

func (v *LevadDetector) ProcessFrame(pcm []byte) (FrameResult, error) {
	ok, err := v.IsSpeech(pcm)
	if err != nil {
		return FrameResult{}, err
	}
	prob := 0.0
	if ok {
		prob = 1.0
	}
	return FrameResult{Probability: prob, IsSpeech: ok, Timestamp: time.Now()}, nil
}

func (v *LevadDetector) Reset() {}

// IsSpeech returns true when any classified chunk in pcm is speech.
func (v *LevadDetector) IsSpeech(pcm []byte) (bool, error) {
	if v == nil || v.h == nil || len(pcm) < 2 {
		return false, nil
	}
	n := len(pcm) / 2
	samples := make([]int16, n)
	for i := 0; i < n; i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	var speech C.int
	rc := C.lv_vad_process(v.h, (*C.int16_t)(unsafe.Pointer(&samples[0])), C.size_t(n), &speech)
	switch rc {
	case C.LV_OK:
		return speech != 0, nil
	case C.LV_NEED_MORE:
		return false, nil
	case C.LV_ERR_PANIC:
		return false, fmt.Errorf("levad: panic in process")
	default:
		return false, fmt.Errorf("levad: process rc=%d", int(rc))
	}
}

// Close releases native resources.
func (v *LevadDetector) Close() error {
	if v == nil || v.h == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	runtime.SetFinalizer(v, nil)
	C.lv_vad_close(v.h)
	v.h = nil
	return nil
}
