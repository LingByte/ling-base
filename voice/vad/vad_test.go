// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package vad

import (
	"bytes"
	"testing"
	"time"
)

func loudPCMFrame() []byte {
	buf := make([]byte, 320)
	for i := 0; i < len(buf); i += 2 {
		buf[i] = 0xff
		buf[i+1] = 0x7f // int16 LE ≈ 32767
	}
	return buf
}

func quietPCMFrame() []byte {
	return bytes.Repeat([]byte{0x00, 0x00}, 80)
}

func silentFrame(ms int, sampleRate int) []byte {
	samples := sampleRate * ms / 1000
	return bytes.Repeat([]byte{0x00, 0x00}, samples)
}

func toneFrame(ms int, sampleRate int, freq float64, amp int16) []byte {
	samples := sampleRate * ms / 1000
	buf := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		// Simple sine wave
		t := float64(i) / float64(sampleRate)
		v := int16(float64(amp) * sinApprox(2*PI*freq*t))
		buf[i*2] = byte(v & 0xff)
		buf[i*2+1] = byte(v >> 8)
	}
	return buf
}

// sinApprox is a fast sine approximation to avoid importing math in tests.
const PI = 3.14159265358979323846

func sinApprox(x float64) float64 {
	// Reduce to [-PI, PI]
	for x > PI {
		x -= 2 * PI
	}
	for x < -PI {
		x += 2 * PI
	}
	// Taylor series (good enough for test signal generation)
	x2 := x * x
	x3 := x2 * x
	x5 := x3 * x2
	x7 := x5 * x2
	return x - x3/6 + x5/120 - x7/5040
}

// ─── RMS / ZCR tests ───

func TestCalculateRMS_Edges(t *testing.T) {
	if CalculateRMS(nil) != 0 || CalculateRMS([]byte{1}) != 0 {
		t.Fatal("edge cases should return 0")
	}
	if v := CalculateRMS([]byte{0x00, 0x10}); v <= 0 {
		t.Fatalf("expected positive RMS, got %v", v)
	}
}

func TestCalculateRMS_LoudFrame(t *testing.T) {
	v := CalculateRMS(loudPCMFrame())
	if v < 30000 {
		t.Fatalf("expected high RMS for loud frame, got %.0f", v)
	}
}

func TestCalculateRMS_QuietFrame(t *testing.T) {
	v := CalculateRMS(quietPCMFrame())
	if v != 0 {
		t.Fatalf("expected 0 RMS for silence, got %v", v)
	}
}

func TestCalculateZCR_Silence(t *testing.T) {
	zcr := CalculateZCR(silentFrame(20, 16000))
	if zcr != 0 {
		t.Fatalf("expected 0 ZCR for silence, got %v", zcr)
	}
}

func TestCalculateZCR_Tone(t *testing.T) {
	// A 440Hz tone at 16kHz should have a non-zero ZCR
	zcr := CalculateZCR(toneFrame(20, 16000, 440, 16000))
	if zcr <= 0 {
		t.Fatalf("expected positive ZCR for tone, got %v", zcr)
	}
}

// ─── EnergyDetector tests ───

func TestEnergyDetector_NewWithConfig(t *testing.T) {
	d := NewEnergyDetectorWithConfig(EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       1000,
	})
	if d == nil {
		t.Fatal("expected detector")
	}
	if d.Kind() != EngineEnergy {
		t.Errorf("expected EngineEnergy, got %v", d.Kind())
	}
	if d.SampleRate() != 16000 {
		t.Errorf("expected 16000, got %d", d.SampleRate())
	}
	if d.FrameDuration() != 20 {
		t.Errorf("expected 20ms, got %d", d.FrameDuration())
	}
}

func TestEnergyDetector_ProcessFrame_Loud(t *testing.T) {
	d := NewEnergyDetectorWithConfig(EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       500,
		MinSpeechFrames: 1,
	})
	result, err := d.ProcessFrame(loudPCMFrame())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsSpeech {
		t.Error("expected speech for loud frame")
	}
	if result.Probability < 0.5 {
		t.Errorf("expected high probability, got %v", result.Probability)
	}
	if result.RMS < 30000 {
		t.Errorf("expected high RMS, got %v", result.RMS)
	}
}

func TestEnergyDetector_ProcessFrame_Silence(t *testing.T) {
	d := NewEnergyDetectorWithConfig(EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       500,
	})
	result, err := d.ProcessFrame(silentFrame(20, 16000))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsSpeech {
		t.Error("expected no speech for silence")
	}
	if result.Probability != 0 {
		t.Errorf("expected 0 probability, got %v", result.Probability)
	}
}

func TestEnergyDetector_ProcessFrame_HighThreshold(t *testing.T) {
	d := NewEnergyDetectorWithConfig(EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       40000, // above max possible RMS
	})
	result, _ := d.ProcessFrame(loudPCMFrame())
	if result.IsSpeech {
		t.Error("high threshold should block speech detection")
	}
}

func TestEnergyDetector_ProcessFrame_ZCRFilter(t *testing.T) {
	// A high-frequency tone has high ZCR; with ZCR filter enabled,
	// it should not be classified as speech even if RMS is high.
	d := NewEnergyDetectorWithConfig(EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       500,
		MaxZCR:          0.1, // low ZCR threshold
	})
	tone := toneFrame(20, 16000, 4000, 30000) // 4kHz tone, high amplitude
	result, _ := d.ProcessFrame(tone)
	if result.IsSpeech {
		t.Error("ZCR filter should reject high-frequency tone")
	}
}

func TestEnergyDetector_Reset(t *testing.T) {
	d := NewEnergyDetectorWithConfig(EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       500,
	})
	// Feed some frames to build up state
	_, _ = d.ProcessFrame(loudPCMFrame())
	d.Reset()
	// After reset, internal state should be clean
	result, _ := d.ProcessFrame(silentFrame(20, 16000))
	if result.IsSpeech {
		t.Error("silence after reset should not be speech")
	}
}

func TestEnergyDetector_Close(t *testing.T) {
	d := NewEnergyDetectorWithConfig(EnergyConfig{})
	if err := d.Close(); err != nil {
		t.Errorf("close should not error: %v", err)
	}
}

// ─── Barge-in (legacy API) tests ───

func TestEnergyDetector_CheckBargeIn_DisabledOrNotPlaying(t *testing.T) {
	d := NewEnergyDetector()
	d.SetEnabled(false)
	if d.CheckBargeIn(loudPCMFrame(), true) {
		t.Fatal("disabled detector should not trigger")
	}
	d.SetEnabled(true)
	if d.CheckBargeIn(loudPCMFrame(), false) {
		t.Fatal("not playing should not trigger barge-in")
	}
}

func TestEnergyDetector_CheckBargeIn_Triggers(t *testing.T) {
	d := NewEnergyDetector()
	d.SetThreshold(500)
	d.SetConsecutiveFrames(1)
	if !d.CheckBargeIn(loudPCMFrame(), true) {
		t.Fatal("expected barge-in")
	}
}

func TestEnergyDetector_CheckBargeIn_ConsecutiveFrames(t *testing.T) {
	d := NewEnergyDetector()
	d.SetThreshold(500)
	d.SetConsecutiveFrames(2)
	frame := loudPCMFrame()
	if d.CheckBargeIn(frame, true) {
		t.Fatal("first frame should not trigger")
	}
	if !d.CheckBargeIn(frame, true) {
		t.Fatal("second frame should trigger")
	}
}

func TestEnergyDetector_AdaptiveNoiseFloor(t *testing.T) {
	d := NewEnergyDetectorWithConfig(EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       2000,
		MinSpeechFrames: 1,
		AdaptiveNoise:   true,
	})
	quiet := quietPCMFrame()
	for i := 0; i < 25; i++ {
		_, _ = d.ProcessFrame(quiet)
	}
	result, _ := d.ProcessFrame(quiet)
	if result.IsSpeech {
		t.Fatal("quiet should stay below adaptive threshold")
	}
}

func TestEnergyDetector_SetThreshold(t *testing.T) {
	d := NewEnergyDetector()
	d.SetThreshold(40000)
	d.SetConsecutiveFrames(1)
	if d.CheckBargeIn(loudPCMFrame(), true) {
		t.Fatal("high threshold should block")
	}
}

func TestEnergyDetector_SetBargeInCallback(t *testing.T) {
	d := NewEnergyDetector()
	d.SetThreshold(500)
	d.SetConsecutiveFrames(1)
	called := false
	d.SetBargeInCallback(func() {
		called = true
	})
	d.CheckBargeIn(loudPCMFrame(), true)
	if !called {
		t.Fatal("callback should have been called")
	}
}

func TestEnergyDetector_UserSpeechLikely(t *testing.T) {
	d := NewEnergyDetector()
	d.SetThreshold(500)
	if !d.UserSpeechLikely(loudPCMFrame()) {
		t.Error("expected speech likely for loud frame")
	}
	if d.UserSpeechLikely(quietPCMFrame()) {
		t.Error("expected no speech for quiet frame")
	}
}

// ─── Streamer tests ───

func TestStreamer_SpeechStartEnd(t *testing.T) {
	det := NewEnergyDetectorWithConfig(EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       500,
		MinSpeechFrames: 2,
		HangoverFrames:  3,
	})
	cfg := Config{
		SampleRate:      16000,
		FrameDurationMs: 20,
		MinSpeechFrames: 2,
		HangoverFrames:  3,
	}
	stream := NewStreamer(det, cfg)
	defer stream.Close()

	loud := loudPCMFrame()
	silent := silentFrame(20, 16000)

	// Feed silence — no event
	for i := 0; i < 5; i++ {
		stream.ProcessFrame(silent)
	}
	if stream.IsSpeech() {
		t.Fatal("should not be in speech state")
	}

	// Feed loud frames — should trigger SpeechStart
	for i := 0; i < 3; i++ {
		stream.ProcessFrame(loud)
	}
	if !stream.IsSpeech() {
		t.Fatal("should be in speech state after loud frames")
	}

	// Feed silence — should trigger SpeechEnd after hangover
	for i := 0; i < 4; i++ {
		stream.ProcessFrame(silent)
	}
	if stream.IsSpeech() {
		t.Fatal("should not be in speech state after hangover")
	}

	// Check events
	events := drainEvents(stream)
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}
	if events[0].Type != EventSpeechStart {
		t.Errorf("expected SpeechStart, got %v", events[0].Type)
	}
	if events[len(events)-1].Type != EventSpeechEnd {
		t.Errorf("expected SpeechEnd, got %v", events[len(events)-1].Type)
	}
}

func TestStreamer_HangoverDebounce(t *testing.T) {
	det := NewEnergyDetectorWithConfig(EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       500,
	})
	cfg := Config{
		SampleRate:      16000,
		FrameDurationMs: 20,
		MinSpeechFrames: 1,
		HangoverFrames:  5,
	}
	stream := NewStreamer(det, cfg)
	defer stream.Close()

	loud := loudPCMFrame()
	silent := silentFrame(20, 16000)

	// Start speech
	stream.ProcessFrame(loud)
	if !stream.IsSpeech() {
		t.Fatal("should be in speech state")
	}

	// Brief silence (less than hangover) — should NOT end speech
	for i := 0; i < 3; i++ {
		stream.ProcessFrame(silent)
	}
	if !stream.IsSpeech() {
		t.Fatal("brief silence should not end speech (hangover)")
	}

	// More silence — should end speech
	for i := 0; i < 3; i++ {
		stream.ProcessFrame(silent)
	}
	if stream.IsSpeech() {
		t.Fatal("long silence should end speech")
	}
}

func TestStreamer_Reset(t *testing.T) {
	det := NewEnergyDetectorWithConfig(EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       500,
	})
	stream := NewStreamer(det, Config{
		SampleRate:      16000,
		FrameDurationMs: 20,
		MinSpeechFrames: 1,
		HangoverFrames:  2,
	})
	defer stream.Close()

	stream.ProcessFrame(loudPCMFrame())
	if !stream.IsSpeech() {
		t.Fatal("should be in speech state")
	}

	stream.Reset()
	if stream.IsSpeech() {
		t.Fatal("should not be in speech state after reset")
	}
	if stream.FrameIndex() != 0 {
		t.Errorf("expected frame index 0, got %d", stream.FrameIndex())
	}
}

func TestStreamer_FrameIndex(t *testing.T) {
	det := NewEnergyDetectorWithConfig(EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       500,
	})
	stream := NewStreamer(det, Config{
		SampleRate:      16000,
		FrameDurationMs: 20,
	})
	defer stream.Close()

	for i := 0; i < 10; i++ {
		stream.ProcessFrame(silentFrame(20, 16000))
	}
	if stream.FrameIndex() != 10 {
		t.Errorf("expected frame index 10, got %d", stream.FrameIndex())
	}
}

func TestStreamer_NilDetector(t *testing.T) {
	stream := NewStreamer(nil, Config{})
	defer stream.Close()
	_, err := stream.ProcessFrame(loudPCMFrame())
	if err == nil {
		t.Fatal("expected error for nil detector")
	}
}

// ─── Factory tests ───

func TestNewDetector_Energy(t *testing.T) {
	det, err := NewDetector(EngineEnergy, DefaultConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if det.Kind() != EngineEnergy {
		t.Errorf("expected EngineEnergy, got %v", det.Kind())
	}
}

func TestNewDetector_WebRTC(t *testing.T) {
	det, err := NewDetector(EngineWebRTC, DefaultConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if det.Kind() != EngineWebRTC {
		t.Errorf("expected EngineWebRTC, got %v", det.Kind())
	}
}

func TestNewDetector_Silero(t *testing.T) {
	det, err := NewDetector(EngineSilero, DefaultConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if det.Kind() != EngineSilero {
		t.Errorf("expected EngineSilero, got %v", det.Kind())
	}
}

func TestNewDetector_Silero_8kHz(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SampleRate = 8000
	_, err := NewDetector(EngineSilero, cfg)
	if err == nil {
		t.Fatal("expected error for 8kHz Silero")
	}
}

func TestNewDetector_UnknownKind(t *testing.T) {
	_, err := NewDetector("unknown", DefaultConfig())
	if err == nil {
		t.Fatal("expected error for unknown engine kind")
	}
}

func TestNewStreamerWithConfig(t *testing.T) {
	stream, err := NewStreamerWithConfig(EngineEnergy, DefaultConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer stream.Close()
	if stream == nil {
		t.Fatal("expected streamer")
	}
}

// ─── Silero detector tests ───

func TestSileroDetector_Basic(t *testing.T) {
	det, err := NewSileroDetector(0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer det.Close()

	if det.Kind() != EngineSilero {
		t.Errorf("expected EngineSilero, got %v", det.Kind())
	}
	if det.SampleRate() != 16000 {
		t.Errorf("expected 16000, got %d", det.SampleRate())
	}
	if det.FrameDuration() != 32 {
		t.Errorf("expected 32ms, got %d", det.FrameDuration())
	}
}

func TestSileroDetector_ProcessSilence(t *testing.T) {
	det, err := NewSileroDetector(0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer det.Close()

	// Feed silence frames (32ms @ 16kHz = 512 samples = 1024 bytes)
	silent := silentFrame(32, 16000)
	result, err := det.ProcessFrame(silent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Silero should report low probability for silence
	if result.IsSpeech {
		t.Error("silence should not be detected as speech")
	}
}

func TestSileroDetector_ProcessLoud(t *testing.T) {
	det, err := NewSileroDetector(0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer det.Close()

	// Feed a loud tone (simulating speech-like signal)
	tone := toneFrame(32, 16000, 200, 30000) // 200Hz, loud
	result, err := det.ProcessFrame(tone)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Note: a pure tone may or may not trigger Silero (it's designed for
	// real speech), but the result should be valid
	if result.Probability < 0 || result.Probability > 1 {
		t.Errorf("probability out of range: %v", result.Probability)
	}
}

func TestSileroDetector_PartialFrame(t *testing.T) {
	det, err := NewSileroDetector(0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer det.Close()

	// Feed a small frame (less than 512 samples)
	small := silentFrame(10, 16000) // 160 samples
	result, err := det.ProcessFrame(small)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsSpeech {
		t.Error("partial frame should not trigger speech")
	}
}

func TestSileroDetector_Reset(t *testing.T) {
	det, err := NewSileroDetector(0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer det.Close()

	// Feed some frames then reset
	_, _ = det.ProcessFrame(silentFrame(32, 16000))
	det.Reset()
	// After reset, should work normally
	result, _ := det.ProcessFrame(silentFrame(32, 16000))
	if result.Probability < 0 || result.Probability > 1 {
		t.Errorf("probability out of range after reset: %v", result.Probability)
	}
}

func TestSileroDetector_SetThreshold(t *testing.T) {
	det, err := NewSileroDetector(0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer det.Close()

	det.SetThreshold(0.8)
	if det.Threshold() != 0.8 {
		t.Errorf("expected 0.8, got %v", det.Threshold())
	}
}

// ─── WebRTC detector tests ───

func TestWebRTCDetector_Basic(t *testing.T) {
	det, err := NewWebRTCDetector(16000, 20, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer det.Close()

	if det.Kind() != EngineWebRTC {
		t.Errorf("expected EngineWebRTC, got %v", det.Kind())
	}
	if det.SampleRate() != 16000 {
		t.Errorf("expected 16000, got %d", det.SampleRate())
	}
	if det.FrameDuration() != 20 {
		t.Errorf("expected 20ms, got %d", det.FrameDuration())
	}
}

func TestWebRTCDetector_ProcessSilence(t *testing.T) {
	det, err := NewWebRTCDetector(16000, 20, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer det.Close()

	silent := silentFrame(20, 16000)
	result, err := det.ProcessFrame(silent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsSpeech {
		t.Error("silence should not be detected as speech")
	}
}

func TestWebRTCDetector_ProcessLoud(t *testing.T) {
	det, err := NewWebRTCDetector(16000, 20, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer det.Close()

	// Feed a loud tone (WebRTC may or may not classify pure tones as speech)
	tone := toneFrame(20, 16000, 200, 30000)
	result, err := det.ProcessFrame(tone)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Just verify it doesn't crash; WebRTC's behavior on pure tones varies
	_ = result
}

func TestWebRTCDetector_InvalidSampleRate(t *testing.T) {
	_, err := NewWebRTCDetector(11025, 20, 2)
	if err == nil {
		t.Fatal("expected error for invalid sample rate")
	}
}

func TestWebRTCDetector_SetMode(t *testing.T) {
	det, err := NewWebRTCDetector(16000, 20, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer det.Close()

	if err := det.SetMode(3); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if det.Mode() != 3 {
		t.Errorf("expected mode 3, got %d", det.Mode())
	}
}

func TestWebRTCDetector_InvalidMode(t *testing.T) {
	det, err := NewWebRTCDetector(16000, 20, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer det.Close()

	if err := det.SetMode(5); err == nil {
		t.Error("expected error for invalid mode")
	}
}

// ─── Config tests ───

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.SampleRate != 16000 {
		t.Errorf("expected 16000, got %d", cfg.SampleRate)
	}
	if cfg.Threshold != 0.5 {
		t.Errorf("expected 0.5, got %v", cfg.Threshold)
	}
}

func TestConfig_Validate(t *testing.T) {
	cfg := Config{}
	cfg.validate()
	if cfg.SampleRate != 16000 {
		t.Errorf("expected 16000, got %d", cfg.SampleRate)
	}
	if cfg.Threshold != 0.5 {
		t.Errorf("expected 0.5, got %v", cfg.Threshold)
	}
	if cfg.MinSpeechFrames != 3 {
		t.Errorf("expected 3, got %d", cfg.MinSpeechFrames)
	}
	if cfg.HangoverFrames != 15 {
		t.Errorf("expected 15, got %d", cfg.HangoverFrames)
	}
}

func TestConfig_FrameSampleCount(t *testing.T) {
	cfg := Config{SampleRate: 16000, FrameDurationMs: 20}
	cfg.validate()
	if cfg.frameSampleCount() != 320 {
		t.Errorf("expected 320, got %d", cfg.frameSampleCount())
	}
	if cfg.frameBytes() != 640 {
		t.Errorf("expected 640, got %d", cfg.frameBytes())
	}
}

// ─── AssistantConfig tests ───

func TestParseAssistantConfig_Empty(t *testing.T) {
	cfg := ParseAssistantConfig(nil)
	if cfg.EnergyThreshold != 0 {
		t.Errorf("expected 0, got %d", cfg.EnergyThreshold)
	}
}

func TestParseAssistantConfig_Values(t *testing.T) {
	cfg := ParseAssistantConfig(map[string]any{
		"energyThreshold": float64(5000),
		"minSpeechFrames": float64(5),
		"speechStartMs":   float64(100),
		"ratio":           float64(1.5),
	})
	if cfg.EnergyThreshold != 5000 {
		t.Errorf("expected 5000, got %d", cfg.EnergyThreshold)
	}
	if cfg.MinSpeechFrames != 5 {
		t.Errorf("expected 5, got %d", cfg.MinSpeechFrames)
	}
	if cfg.Ratio != 1.5 {
		t.Errorf("expected 1.5, got %v", cfg.Ratio)
	}
}

func TestParseAssistantConfigBytes(t *testing.T) {
	cfg := ParseAssistantConfigBytes([]byte(`{"energyThreshold": 3000}`))
	if cfg.EnergyThreshold != 3000 {
		t.Errorf("expected 3000, got %d", cfg.EnergyThreshold)
	}
}

func TestParseAssistantConfigBytes_Invalid(t *testing.T) {
	cfg := ParseAssistantConfigBytes([]byte(`invalid`))
	if cfg.EnergyThreshold != 0 {
		t.Errorf("expected 0 for invalid JSON, got %d", cfg.EnergyThreshold)
	}
}

func TestApplyAssistantConfig(t *testing.T) {
	d := NewEnergyDetector()
	cfg := AssistantConfig{
		EnergyThreshold: 6000,
		MinSpeechFrames: 5,
	}
	d.ApplyAssistantConfig(cfg, 5500, 8)
	if d.Threshold() != 6000 {
		t.Errorf("expected 6000, got %v", d.Threshold())
	}
}

// ─── Helpers ───

func drainEvents(s *Streamer) []Event {
	var events []Event
	for {
		select {
		case ev, ok := <-s.Events():
			if !ok {
				return events
			}
			events = append(events, ev)
		default:
			return events
		}
	}
}

// Ensure time import is used.
var _ = time.Now

// ─── PreSpeechBuffer tests ───

func TestStreamer_PreSpeechBuffer(t *testing.T) {
	det := NewEnergyDetectorWithConfig(EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       500,
	})
	defer det.Close()

	stream := NewStreamerExplicit(det, StreamerConfig{
		MinSpeechFrames:       3,
		HangoverFrames:        5,
		PreSpeechBufferFrames: 5,
	})
	defer stream.Close()

	loud := loudPCMFrame()
	silent := silentFrame(20, 16000)

	// Feed 5 silent frames (fill pre-speech buffer)
	for i := 0; i < 5; i++ {
		stream.ProcessFrame(silent)
	}

	// Feed 5 loud frames (trigger speech start after 3)
	for i := 0; i < 5; i++ {
		stream.ProcessFrame(loud)
	}

	// Should have speech start event
	if !stream.IsSpeech() {
		t.Fatal("should be in speech state")
	}

	// Pre-speech audio should contain the last 5 frames before speech start
	preAudio := stream.PreSpeechAudio()
	if preAudio == nil {
		t.Fatal("expected pre-speech audio")
	}
	// Should have 5 frames (the buffered silent frames + the first 2 loud frames
	// before speech was declared)
	if len(preAudio) != 5 {
		t.Errorf("expected 5 pre-speech frames, got %d", len(preAudio))
	}
}

func TestStreamer_PreSpeechBuffer_Disabled(t *testing.T) {
	det := NewEnergyDetectorWithConfig(EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       500,
	})
	defer det.Close()

	stream := NewStreamerExplicit(det, StreamerConfig{
		MinSpeechFrames:       2,
		HangoverFrames:        5,
		PreSpeechBufferFrames: 0, // disabled
	})
	defer stream.Close()

	stream.ProcessFrame(loudPCMFrame())
	stream.ProcessFrame(loudPCMFrame())

	if stream.PreSpeechAudio() != nil {
		t.Error("expected nil when PreSpeechBufferFrames=0")
	}
}

func TestStreamerExplicit_NilDetector(t *testing.T) {
	stream := NewStreamerExplicit(nil, StreamerConfig{
		MinSpeechFrames: 2,
		HangoverFrames:  5,
	})
	defer stream.Close()
	_, err := stream.ProcessFrame(loudPCMFrame())
	if err == nil {
		t.Fatal("expected error for nil detector")
	}
}
