// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package vad

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// ErrInvalidDataType is returned when a component receives unexpected data.
var ErrInvalidDataType = errors.New("vad: invalid data type")

// ──────────────────────────────────────────────
// RMS + ZCR calculation
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

// zcrPCM16LE computes the zero-crossing rate (fraction of samples that
// cross zero) of PCM16 little-endian audio. Voiced speech has low ZCR;
// noise has high ZCR. This helps distinguish speech from steady tones.
func zcrPCM16LE(pcm []byte) float64 {
	if len(pcm) < 4 {
		return 0
	}
	n := len(pcm) / 2
	if n < 2 {
		return 0
	}
	crossings := 0
	var prev int16
	for i := 0; i+1 < len(pcm); i += 2 {
		sample := int16(binary.LittleEndian.Uint16(pcm[i : i+2]))
		if i > 0 && (prev >= 0) != (sample >= 0) {
			crossings++
		}
		prev = sample
	}
	return float64(crossings) / float64(n-1)
}

// pcm16LEToFloat32 converts PCM16 LE bytes to normalized float32 samples [-1, 1].
func pcm16LEToFloat32(pcm []byte) []float32 {
	n := len(pcm) / 2
	if n == 0 {
		return nil
	}
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		s := int16(binary.LittleEndian.Uint16(pcm[i*2 : i*2+2]))
		out[i] = float32(s) / 32768.0
	}
	return out
}

// ──────────────────────────────────────────────
// PlaybackGate (for barge-in)
// ──────────────────────────────────────────────

// playbackGate tracks downlink TTS activity for echo suppression and barge-in.
type playbackGate struct {
	isPlaying   func() bool
	queueDepth  func() int
	tail        time.Duration
	lastActiveN atomic.Int64 // unix nanos
}

func newPlaybackGate(isPlaying func() bool, queueDepth func() int, tail time.Duration) *playbackGate {
	return &playbackGate{
		isPlaying:  isPlaying,
		queueDepth: queueDepth,
		tail:       tail,
	}
}

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

func (g *playbackGate) isQueued() bool {
	if g == nil || g.queueDepth == nil {
		return false
	}
	return g.queueDepth() > 0
}

func (g *playbackGate) isBargeInWindow() bool {
	if g == nil {
		return false
	}
	if g.isStreaming() || g.isQueued() {
		return true
	}
	return g.inTail()
}

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

func (g *playbackGate) reset() {
	if g != nil {
		g.lastActiveN.Store(0)
	}
}

// ──────────────────────────────────────────────
// EnergyDetector — RMS + ZCR based VAD
// ──────────────────────────────────────────────

// EnergyConfig configures the energy-based VAD detector.
type EnergyConfig struct {
	SampleRate      int    // 8000 or 16000
	FrameDurationMs int    // 10, 20, or 30
	Threshold       float64 // RMS threshold (0 = auto-calibrate)
	MinSpeechFrames int    // consecutive frames to trigger speech
	HangoverFrames  int    // consecutive silence frames before speech end
	MaxZCR          float64 // max zero-crossing rate for speech (0.0-1.0, 0 = disable ZCR check)
	AdaptiveNoise   bool   // enable adaptive noise floor
	MaxNoiseSamples int    // max samples for noise floor estimation
}

// DefaultEnergyConfig returns production-ready energy VAD defaults.
func DefaultEnergyConfig() EnergyConfig {
	return EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       1500.0,
		MinSpeechFrames: 3,
		HangoverFrames:  15,
		MaxZCR:          0.0, // disabled by default; enable for noisy environments
		AdaptiveNoise:   true,
	}
}

// EnergyDetector is a production-grade energy + ZCR voice activity detector.
//
// It implements the Detector interface and can be used standalone or wrapped
// in a Streamer for state-machine event emission.
//
// For barge-in (interrupt detection during TTS playback), use the legacy
// CheckBargeIn method which integrates with a playbackGate.
type EnergyDetector struct {
	mu          sync.Mutex
	cfg         EnergyConfig
	sampleRate  int
	frameMs     int

	// Adaptive noise floor
	noiseSamples    []float64
	maxNoiseSamples int
	noiseLevel      float64
	adaptiveThreshold float64

	// State
	enabled           bool
	frameCounter      int
	armed             bool

	// Barge-in integration
	gate            *playbackGate
	bargeInCallback func()
	logger          func(string)

	// For legacy barge-in polling
	bargeIn atomic.Bool
	playing atomic.Bool
	threshold atomic.Uint64 // float64 bits
}

// NewEnergyDetectorWithConfig creates an energy VAD with explicit configuration.
func NewEnergyDetectorWithConfig(cfg EnergyConfig) *EnergyDetector {
	if cfg.SampleRate != 8000 && cfg.SampleRate != 16000 {
		cfg.SampleRate = 16000
	}
	if cfg.FrameDurationMs <= 0 {
		cfg.FrameDurationMs = 20
	}
	if cfg.Threshold <= 0 {
		cfg.Threshold = 1500.0
	}
	if cfg.MinSpeechFrames <= 0 {
		cfg.MinSpeechFrames = 3
	}
	if cfg.HangoverFrames <= 0 {
		cfg.HangoverFrames = 15
	}
	if cfg.MaxNoiseSamples <= 0 {
		cfg.MaxNoiseSamples = 20
	}
	d := &EnergyDetector{
		cfg:             cfg,
		sampleRate:      cfg.SampleRate,
		frameMs:         cfg.FrameDurationMs,
		enabled:         true,
		noiseSamples:    make([]float64, 0, cfg.MaxNoiseSamples),
		maxNoiseSamples: cfg.MaxNoiseSamples,
	}
	d.threshold.Store(math.Float64bits(cfg.Threshold))
	return d
}

// NewEnergyDetector builds a barge-in detector with sip-aligned defaults.
// Uses defaultBargeInVADConfig (higher RMS + multi-frame) so TTS echo / line
// noise is less likely to false-trigger interrupt during playback.
func NewEnergyDetector() *EnergyDetector {
	d := NewEnergyDetectorWithConfig(EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       5500.0, // barge-in tuned (higher)
		MinSpeechFrames: 8,
		HangoverFrames:  15,
		AdaptiveNoise:   true,
	})
	d.gate = newPlaybackGate(func() bool { return d.playing.Load() }, nil, 0)
	d.bargeIn.Store(false)
	// Set default barge-in callback that sets the atomic flag.
	d.bargeInCallback = func() { d.bargeIn.Store(true) }
	return d
}

// Kind returns the engine kind.
func (d *EnergyDetector) Kind() EngineKind { return EngineEnergy }

// SampleRate returns the configured sample rate.
func (d *EnergyDetector) SampleRate() int { return d.sampleRate }

// FrameDuration returns the frame duration in milliseconds.
func (d *EnergyDetector) FrameDuration() int { return d.frameMs }

// Close releases resources (no-op for energy detector).
func (d *EnergyDetector) Close() error { return nil }

// ProcessFrame implements the Detector interface.
// It analyzes a PCM16 LE frame and returns the detection result.
func (d *EnergyDetector) ProcessFrame(pcm []byte) (FrameResult, error) {
	if d == nil {
		return FrameResult{}, errors.New("vad: nil detector")
	}
	if len(pcm) < 2 {
		return FrameResult{Timestamp: time.Now()}, nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	rms := rmsPCM16LE(pcm)
	zcr := zcrPCM16LE(pcm)

	// Update adaptive noise floor
	if d.cfg.AdaptiveNoise {
		d.updateNoiseFloor(rms)
	}

	threshold := d.effectiveThreshold()
	isSpeech := rms > threshold

	// ZCR filter: voiced speech has low ZCR. If enabled and ZCR is very high,
	// it's likely noise even if RMS exceeds threshold.
	if isSpeech && d.cfg.MaxZCR > 0 && zcr > d.cfg.MaxZCR {
		isSpeech = false
	}

	prob := 0.0
	if isSpeech {
		prob = 1.0
	} else if threshold > 0 {
		prob = math.Min(1.0, rms/threshold*0.5)
	}

	return FrameResult{
		Probability: prob,
		IsSpeech:    isSpeech,
		RMS:         rms,
		Timestamp:   time.Now(),
	}, nil
}

// Reset clears internal state.
func (d *EnergyDetector) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.frameCounter = 0
	d.armed = false
	d.noiseSamples = d.noiseSamples[:0]
	d.noiseLevel = 0
	d.adaptiveThreshold = 0
	d.bargeIn.Store(false)
	if d.gate != nil {
		d.gate.reset()
	}
}

func (d *EnergyDetector) updateNoiseFloor(rms float64) {
	if rms >= 350 {
		return
	}
	d.noiseSamples = append(d.noiseSamples, rms)
	if len(d.noiseSamples) > d.maxNoiseSamples {
		d.noiseSamples = d.noiseSamples[1:]
	}
	var sum float64
	for _, s := range d.noiseSamples {
		sum += s
	}
	if len(d.noiseSamples) == 0 {
		return
	}
	d.noiseLevel = sum / float64(len(d.noiseSamples))
	d.adaptiveThreshold = d.noiseLevel * 4.0
	if d.adaptiveThreshold < 180 {
		d.adaptiveThreshold = 180
	}
	threshold := math.Float64frombits(d.threshold.Load())
	if d.adaptiveThreshold > threshold {
		d.adaptiveThreshold = threshold
	}
}

func (d *EnergyDetector) effectiveThreshold() float64 {
	threshold := math.Float64frombits(d.threshold.Load())
	if d.cfg.AdaptiveNoise && d.adaptiveThreshold > threshold {
		return d.adaptiveThreshold
	}
	return threshold
}

// ──────────────────────────────────────────────
// Barge-in API (legacy compatibility)
// ──────────────────────────────────────────────

// SetLogFunc attaches an optional log sink.
func (d *EnergyDetector) SetLogFunc(fn func(string)) {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.logger = fn
	d.mu.Unlock()
}

// CheckBargeIn returns true when uplink PCM suggests the user is speaking
// during synthesis playback. This is the legacy barge-in API.
//
// The detector uses a playbackGate to only consider barge-in while TTS is
// actively playing. The internal vadComponent state machine requires
// consecutiveFramesNeeded over-threshold frames before firing.
func (d *EnergyDetector) CheckBargeIn(pcmData []byte, synthPlaying bool) bool {
	if d == nil || len(pcmData) < 2 || !d.enabled {
		return false
	}
	d.playing.Store(synthPlaying)
	if !synthPlaying {
		d.bargeIn.Store(false)
		d.Reset()
		return false
	}

	// Process frame through the barge-in state machine
	d.mu.Lock()
	if d.gate == nil {
		d.mu.Unlock()
		return false
	}

	inWindow := d.gate.isBargeInWindow()
	if !inWindow {
		d.frameCounter = 0
		d.armed = false
		d.mu.Unlock()
		return false
	}

	rms := rmsPCM16LE(pcmData)
	if d.cfg.AdaptiveNoise {
		d.updateNoiseFloor(rms)
	}
	threshold := d.effectiveThreshold()

	var cb func()
	if rms > threshold {
		d.frameCounter++
		if d.frameCounter >= d.cfg.MinSpeechFrames && !d.armed {
			d.armed = true
			d.frameCounter = 0
			if d.logger != nil {
				d.logger(formatBargeInLog(rms, threshold))
			}
			cb = d.bargeInCallback
		}
	} else {
		d.frameCounter = 0
	}
	d.mu.Unlock()

	if cb != nil {
		cb()
	}

	// Check the barge-in flag (set by callback)
	if d.bargeIn.Load() {
		d.bargeIn.Store(false)
		return true
	}
	return false
}

func formatBargeInLog(rms, threshold float64) string {
	return fmtBargeIn(rms, threshold)
}

// SetBargeInCallback sets the callback invoked on barge-in detection.
func (d *EnergyDetector) SetBargeInCallback(fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.bargeInCallback = func() {
		if fn != nil {
			fn()
		}
		d.bargeIn.Store(true)
	}
}

// SetEnabled turns detection on/off.
func (d *EnergyDetector) SetEnabled(enabled bool) {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.enabled = enabled
	if !enabled {
		d.frameCounter = 0
		d.armed = false
		d.bargeIn.Store(false)
	}
	d.mu.Unlock()
}

// Enabled returns whether detection is enabled.
func (d *EnergyDetector) Enabled() bool {
	return d != nil && d.enabled
}

// Threshold returns the current RMS threshold.
func (d *EnergyDetector) Threshold() float64 {
	if d == nil {
		return 0
	}
	return math.Float64frombits(d.threshold.Load())
}

// SetThreshold sets the RMS ceiling.
func (d *EnergyDetector) SetThreshold(threshold float64) {
	if d != nil {
		d.threshold.Store(math.Float64bits(threshold))
	}
}

// SetConsecutiveFrames sets how many consecutive over-threshold frames
// trigger barge-in.
func (d *EnergyDetector) SetConsecutiveFrames(frames int) {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.cfg.MinSpeechFrames = frames
	d.mu.Unlock()
}

// UserSpeechLikely reports uplink speech activity during listen windows.
func (d *EnergyDetector) UserSpeechLikely(pcmData []byte) bool {
	if d == nil || len(pcmData) < 2 || !d.enabled {
		return false
	}
	thr := math.Float64frombits(d.threshold.Load())
	if thr <= 0 {
		thr = 5500.0
	}
	return CalculateRMS(pcmData) > thr
}

// CalculateRMS computes RMS for PCM16LE frames.
func CalculateRMS(pcmData []byte) float64 {
	if len(pcmData) < 2 {
		return 0
	}
	return rmsPCM16LE(pcmData)
}

// CalculateZCR computes the zero-crossing rate for PCM16LE frames.
func CalculateZCR(pcmData []byte) float64 {
	return zcrPCM16LE(pcmData)
}

// ──────────────────────────────────────────────
// Barge-in internal component (compatibility)
// ──────────────────────────────────────────────

// bargeInConfig is the legacy config type for barge-in VAD.
type bargeInConfig struct {
	Enabled                 bool
	Threshold               float64
	ConsecutiveFramesNeeded int
	MaxNoiseSamples         int
}

func defaultBargeInVADConfig() bargeInConfig {
	return bargeInConfig{
		Enabled:                 true,
		Threshold:               5500.0,
		ConsecutiveFramesNeeded: 8,
		MaxNoiseSamples:         20,
	}
}

// fmtBargeIn formats a barge-in log line.
func fmtBargeIn(rms, threshold float64) string {
	return formatLog("[VAD] barge-in: rms=%.0f threshold=%.0f", rms, threshold)
}

func formatLog(format string, args ...interface{}) string {
	return sprintf(format, args...)
}

// sprintf is a thin wrapper around fmt.Sprintf to avoid importing fmt at
// package level when not needed. Kept as a function for testability.
func sprintf(format string, args ...interface{}) string {
	return sprintfImpl(format, args...)
}

// processBargeInFrame is the internal barge-in processing used by
// CheckBargeIn. It exists to keep the legacy API working without the
// old vadComponent/playbackGate indirection.
func (d *EnergyDetector) processBargeInFrame(ctx context.Context, pcmData []byte) (bool, error) {
	_ = ctx
	if d == nil || len(pcmData) < 2 {
		return false, nil
	}
	result, err := d.ProcessFrame(pcmData)
	if err != nil {
		return false, err
	}
	return result.IsSpeech, nil
}
