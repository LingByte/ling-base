// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package vad

import "fmt"

// NewDetector creates a VAD detector by engine kind.
//
// Supported kinds:
//   - EngineEnergy:  RMS + ZCR energy-based (pure Go, no deps)
//   - EngineWebRTC:  WebRTC GMM (CGO, github.com/GanymedeNil/go-webrtcvad)
//   - EngineSilero:  Silero neural (pure Go, embedded weights, 16kHz only)
func NewDetector(kind EngineKind, cfg Config) (Detector, error) {
	cfg.validate()

	switch kind {
	case EngineEnergy:
		energyCfg := EnergyConfig{
			SampleRate:      cfg.SampleRate,
			FrameDurationMs: cfg.FrameDurationMs,
			Threshold:       cfg.Threshold * 3000, // map [0,1] → RMS scale
			MinSpeechFrames: cfg.MinSpeechFrames,
			HangoverFrames:  cfg.HangoverFrames,
			AdaptiveNoise:   true,
		}
		if energyCfg.Threshold < 100 {
			energyCfg.Threshold = 1500
		}
		return NewEnergyDetectorWithConfig(energyCfg), nil

	case EngineWebRTC:
		return NewWebRTCDetector(cfg.SampleRate, cfg.FrameDurationMs, cfg.Mode)

	case EngineSilero:
		if cfg.SampleRate != 16000 {
			return nil, fmt.Errorf("silero vad: requires 16kHz, got %d", cfg.SampleRate)
		}
		return NewSileroDetector(cfg.Threshold)

	default:
		return nil, fmt.Errorf("vad: unknown engine kind %q", kind)
	}
}

// NewStreamerWithConfig is a convenience function that creates a detector
// and wraps it in a Streamer in one call.
func NewStreamerWithConfig(kind EngineKind, cfg Config) (*Streamer, error) {
	det, err := NewDetector(kind, cfg)
	if err != nil {
		return nil, err
	}
	return NewStreamer(det, cfg), nil
}
