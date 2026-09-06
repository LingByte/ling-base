// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package vad

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// ErrInvalidDataType is returned when a component receives unexpected data.
var ErrInvalidDataType = errors.New("vad: invalid data type")

// ──────────────────────────────────────────────
// RMS calculation (inlined from media.RMSPCM16LE)
// ──────────────────────────────────────────────

// rmsPCM16LE computes the RMS amplitude of PCM16 little-endian audio.
func rmsPCM16LE(pcm []byte) float64 {
	if len(pcm) < 2 {
		return 0
	}
	n := len(pcm) / 2
	var sum float64
	for i := 0; i+1 < len(pcm); i += 2 {
		v := int16(binary.LittleEndian.Uint16(pcm[i : i+2]))
		f := float64(v)
		sum += f * f
	}
	return math.Sqrt(sum / float64(n))
}

// ──────────────────────────────────────────────
// PlaybackGate (inlined from protocol/voice/asr.PlaybackGate)
// ──────────────────────────────────────────────

// playbackGate tracks downlink TTS activity for echo suppression and barge-in.
// It treats queued utterances and a configurable post-playback tail as "active"
// so uplink echo does not leak into ASR right after the speaker goes quiet.
type playbackGate struct {
	isPlaying   func() bool
	queueDepth  func() int
	tail        time.Duration
	lastActiveN atomic.Int64 // unix nanos
}

// newPlaybackGate creates a gate. tail is how long after playback ends uplink
// remains suppressed (room echo). 0 disables tail extension.
func newPlaybackGate(isPlaying func() bool, queueDepth func() int, tail time.Duration) *playbackGate {
	return &playbackGate{
		isPlaying:  isPlaying,
		queueDepth: queueDepth,
		tail:       tail,
	}
}

// isStreaming is true while audio frames are actively leaving the TTS pipeline.
func (g *playbackGate) isStreaming() bool {
	if g == nil {
		return false
	}
	if g.isPlaying != nil && g.isPlaying() {
		g.markActive()
		return true
	}
	return false
}

// isQueued is true when additional utterances wait on the speak queue.
func (g *playbackGate) isQueued() bool {
	if g == nil || g.queueDepth == nil {
		return false
	}
	return g.queueDepth() > 0
}

// isBargeInWindow is true when user interrupt should be considered: streaming,
// queued, or within the post-playback tail.
func (g *playbackGate) isBargeInWindow() bool {
	if g == nil {
		return false
	}
	if g.isStreaming() || g.isQueued() {
		return true
	}
	return g.inTail()
}

// isEchoSuppressActive is true when uplink should not be fed to ASR (echo tail
// included). Slightly longer than barge-in window when tail > 0.
func (g *playbackGate) isEchoSuppressActive() bool {
	return g.isBargeInWindow()
}

func (g *playbackGate) inTail() bool {
	if g.tail <= 0 {
		return false
	}
	last := g.lastActiveN.Load()
	if last == 0 {
		return false
	}
	return time.Since(time.Unix(0, last)) < g.tail
}

func (g *playbackGate) markActive() {
	g.lastActiveN.Store(time.Now().UnixNano())
}

// reset clears tail memory (e.g. on session teardown).
func (g *playbackGate) reset() {
	if g != nil {
		g.lastActiveN.Store(0)
	}
}

// ──────────────────────────────────────────────
// VADComponent (inlined from protocol/voice/asr.VADComponent)
// ──────────────────────────────────────────────

// vadConfig contains configuration for VAD component.
type vadConfig struct {
	Enabled                 bool
	Threshold               float64
	ConsecutiveFramesNeeded int
	MaxNoiseSamples         int
}

// defaultVADConfig returns general-purpose VAD settings (not barge-in tuned).
func defaultVADConfig() vadConfig {
	return vadConfig{
		Enabled:                 true,
		Threshold:               1500.0,
		ConsecutiveFramesNeeded: 1,
		MaxNoiseSamples:         20,
	}
}

// defaultBargeInVADConfig returns thresholds calibrated for interrupting TTS
// on uncancelled speakers.
func defaultBargeInVADConfig() vadConfig {
	return vadConfig{
		Enabled:                 true,
		Threshold:               5500.0,
		ConsecutiveFramesNeeded: 8,
		MaxNoiseSamples:         20,
	}
}

// vadComponent performs energy-based barge-in detection during downlink playback.
type vadComponent struct {
	mu                      sync.RWMutex
	enabled                 bool
	threshold               float64
	adaptiveThreshold       float64
	consecutiveFramesNeeded int
	frameCounter            int
	lastLogTime             time.Time
	noiseLevel              float64
	noiseSamples            []float64
	maxNoiseSamples         int

	gate            *playbackGate
	bargeInCallback func()
	logger          func(string)

	// armed prevents repeated barge-in fires until playback window closes.
	armed bool
}

// newVADComponent creates a VAD stage. gate may be nil (detection disabled).
func newVADComponent(config vadConfig, gate *playbackGate) *vadComponent {
	if config.Threshold == 0 {
		config.Threshold = 1500.0
	}
	if config.ConsecutiveFramesNeeded == 0 {
		config.ConsecutiveFramesNeeded = 1
	}
	if config.MaxNoiseSamples == 0 {
		config.MaxNoiseSamples = 20
	}
	return &vadComponent{
		enabled:                 config.Enabled,
		threshold:               config.Threshold,
		consecutiveFramesNeeded: config.ConsecutiveFramesNeeded,
		noiseSamples:            make([]float64, 0, config.MaxNoiseSamples),
		maxNoiseSamples:         config.MaxNoiseSamples,
		lastLogTime:             time.Now(),
		gate:                    gate,
	}
}

// process analyzes PCM for barge-in; audio passes through unchanged.
func (v *vadComponent) process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	pcmData, ok := data.([]byte)
	if !ok {
		return nil, false, fmt.Errorf("%w: expected []byte, got %T", ErrInvalidDataType, data)
	}
	if len(pcmData) < 2 {
		return pcmData, true, nil
	}

	v.mu.Lock()

	if !v.enabled || v.gate == nil {
		v.mu.Unlock()
		return pcmData, true, nil
	}

	inWindow := v.gate.isBargeInWindow()
	if !inWindow {
		v.frameCounter = 0
		v.armed = false
		v.mu.Unlock()
		return pcmData, true, nil
	}

	rms := rmsPCM16LE(pcmData)
	v.updateNoiseFloor(rms)
	effective := v.effectiveThreshold()

	var cb func()
	if rms > effective {
		v.frameCounter++
		if v.frameCounter >= v.consecutiveFramesNeeded && !v.armed {
			v.armed = true
			v.frameCounter = 0
			if v.logger != nil {
				v.logger(fmt.Sprintf("[VAD] barge-in: rms=%.0f threshold=%.0f", rms, effective))
			}
			cb = v.bargeInCallback
		}
	} else {
		v.frameCounter = 0
	}
	v.mu.Unlock()
	// Invoke synchronously.
	if cb != nil {
		cb()
	}

	return pcmData, true, nil
}

func (v *vadComponent) updateNoiseFloor(rms float64) {
	if rms >= 350 {
		return
	}
	v.noiseSamples = append(v.noiseSamples, rms)
	if len(v.noiseSamples) > v.maxNoiseSamples {
		v.noiseSamples = v.noiseSamples[1:]
	}
	var sum float64
	for _, s := range v.noiseSamples {
		sum += s
	}
	if len(v.noiseSamples) == 0 {
		return
	}
	v.noiseLevel = sum / float64(len(v.noiseSamples))
	v.adaptiveThreshold = v.noiseLevel * 4.0
	if v.adaptiveThreshold < 180 {
		v.adaptiveThreshold = 180
	}
	if v.adaptiveThreshold > v.threshold {
		v.adaptiveThreshold = v.threshold
	}
}

func (v *vadComponent) effectiveThreshold() float64 {
	if v.adaptiveThreshold > v.threshold {
		return v.adaptiveThreshold
	}
	return v.threshold
}

// setBargeInCallback sets the callback invoked on barge-in (edge-triggered per window).
func (v *vadComponent) setBargeInCallback(callback func()) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.bargeInCallback = callback
}

// setLogger sets the logging callback.
func (v *vadComponent) setLogger(callback func(string)) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.logger = callback
}

// setEnabled enables or disables VAD.
func (v *vadComponent) setEnabled(enabled bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.enabled = enabled
	if !enabled {
		v.frameCounter = 0
		v.armed = false
	}
}

// setThreshold sets the RMS energy threshold.
func (v *vadComponent) setThreshold(threshold float64) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.threshold = threshold
}

// setConsecutiveFrames sets frames required before barge-in fires.
func (v *vadComponent) setConsecutiveFrames(frames int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.consecutiveFramesNeeded = frames
}

// reset clears internal state between turns.
func (v *vadComponent) reset() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.frameCounter = 0
	v.armed = false
}
