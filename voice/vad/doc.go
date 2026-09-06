// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package vad provides production-grade voice activity detection for Go.
//
// # Engines
//
// Three engines implement the unified [Detector] interface:
//
//   - [EngineEnergy]: RMS + ZCR energy-based detector. Pure Go, zero
//     dependencies, lowest latency (~20ms). Best for clean signals and
//     barge-in detection.
//   - [EngineWebRTC]: WebRTC GMM speech model (via CGO). Low latency
//     (~20ms), better speech/noise discrimination than energy. Good
//     general-purpose choice.
//   - [EngineSilero]: Silero VAD neural network (pure Go, embedded
//     weights, no CGO). Best discrimination — tones and noise don't
//     trigger it. Higher latency (~32ms per frame). 16kHz only.
//
// # Quick start
//
// Create a detector and wrap it in a [Streamer] for state-machine events:
//
//	det, _ := vad.NewDetector(vad.EngineSilero, vad.DefaultConfig())
//	stream := vad.NewStreamer(det, vad.DefaultConfig())
//	defer stream.Close()
//
//	// Feed PCM16 LE frames from your audio source
//	go func() {
//		for {
//			result, _ := stream.ProcessFrame(pcmFrame)
//			_ = result
//		}
//	}()
//
//	// Consume speech events from another goroutine
//	for ev := range stream.Events() {
//		switch ev.Type {
//		case vad.EventSpeechStart:
//			log.Println("speech started")
//		case vad.EventSpeechEnd:
//			log.Println("speech ended")
//		}
//	}
//
// # Barge-in (legacy API)
//
// For SIP/telephony barge-in (detecting user interruption during TTS
// playback), use [EnergyDetector] directly:
//
//	det := vad.NewEnergyDetector()
//	if det.CheckBargeIn(pcmFrame, ttsIsPlaying) {
//	    // user is speaking — stop TTS
//	}
//
// # Factory
//
// Use [NewDetector] or [NewStreamerWithConfig] to create detectors by
// engine kind with a unified [Config].
package vad
