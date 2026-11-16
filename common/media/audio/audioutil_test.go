// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package audioutil

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// ──────────────────────────────────────────────
// Test helpers
// ──────────────────────────────────────────────

// createTestTone creates a 1-second 440Hz sine wave at 8000Hz, 16-bit mono.
func createTestTone() *Audio {
	return GenerateTone(440, 0.1, 8000) // 0.1s for speed
}

// writeAndReadWAV writes an Audio to a temp WAV file and reads it back.
func writeAndReadWAV(t *testing.T, audio *Audio) *Audio {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.wav")
	if err := SaveWAV(audio, path); err != nil {
		t.Fatalf("SaveWAV failed: %v", err)
	}
	loaded, err := LoadWAV(path)
	if err != nil {
		t.Fatalf("LoadWAV failed: %v", err)
	}
	return loaded
}

// ──────────────────────────────────────────────
// Audio type tests
// ──────────────────────────────────────────────

func TestAudio_NumSamples(t *testing.T) {
	audio := NewAudio(8000, 1, Format16Bit, 100)
	if audio.NumSamples() != 100 {
		t.Fatalf("NumSamples = %d, want 100", audio.NumSamples())
	}
}

func TestAudio_Duration(t *testing.T) {
	audio := NewAudio(8000, 1, Format16Bit, 8000)
	if d := audio.Duration(); d != 1.0 {
		t.Fatalf("Duration = %f, want 1.0", d)
	}
}

func TestInfo_String(t *testing.T) {
	info := Info{SampleRate: 44100, Channels: 2, BitDepth: 16, DurationSec: 3.5}
	s := info.String()
	if s == "" {
		t.Fatal("Info.String() should not be empty")
	}
}

// ──────────────────────────────────────────────
// WAV read/write round-trip
// ──────────────────────────────────────────────

func TestWAVRoundTrip_16Bit(t *testing.T) {
	original := createTestTone()
	loaded := writeAndReadWAV(t, original)

	if loaded.SampleRate != original.SampleRate {
		t.Fatalf("SampleRate = %d, want %d", loaded.SampleRate, original.SampleRate)
	}
	if loaded.Channels != original.Channels {
		t.Fatalf("Channels = %d, want %d", loaded.Channels, original.Channels)
	}
	if loaded.Format != original.Format {
		t.Fatalf("Format = %d, want %d", loaded.Format, original.Format)
	}
	if loaded.NumSamples() != original.NumSamples() {
		t.Fatalf("NumSamples = %d, want %d", loaded.NumSamples(), original.NumSamples())
	}

	// Check sample values are close (16-bit quantization).
	for i := 0; i < original.NumSamples(); i++ {
		diff := math.Abs(original.Samples[0][i] - loaded.Samples[0][i])
		if diff > 2.0/32768 { // allow 2 quantization steps for round-trip
			t.Fatalf("sample %d: diff = %f", i, diff)
		}
	}
}

func TestWAVRoundTrip_8Bit(t *testing.T) {
	original := GenerateTone(440, 0.05, 8000)
	original.Format = Format8Bit
	loaded := writeAndReadWAV(t, original)

	if loaded.Format != Format8Bit {
		t.Fatalf("Format = %d, want 8", loaded.Format)
	}
	// 8-bit has lower precision, allow larger tolerance.
	for i := 0; i < original.NumSamples(); i++ {
		diff := math.Abs(original.Samples[0][i] - loaded.Samples[0][i])
		if diff > 2.0/128 {
			t.Fatalf("sample %d: diff = %f", i, diff)
		}
	}
}

func TestWAVRoundTrip_24Bit(t *testing.T) {
	original := GenerateTone(440, 0.05, 8000)
	original.Format = Format24Bit
	loaded := writeAndReadWAV(t, original)

	if loaded.Format != Format24Bit {
		t.Fatalf("Format = %d, want 24", loaded.Format)
	}
}

func TestWAVRoundTrip_32Bit(t *testing.T) {
	original := GenerateTone(440, 0.05, 8000)
	original.Format = Format32Bit
	loaded := writeAndReadWAV(t, original)

	if loaded.Format != Format32Bit {
		t.Fatalf("Format = %d, want 32", loaded.Format)
	}
}

func TestWAVRoundTrip_Stereo(t *testing.T) {
	original := GenerateTone(440, 0.05, 8000)
	stereo := ToStereo(original)
	loaded := writeAndReadWAV(t, stereo)

	if loaded.Channels != 2 {
		t.Fatalf("Channels = %d, want 2", loaded.Channels)
	}
}

func TestWriteWAV_ToBuffer(t *testing.T) {
	audio := createTestTone()
	var buf bytes.Buffer
	if err := WriteWAV(&buf, audio); err != nil {
		t.Fatalf("WriteWAV failed: %v", err)
	}
	if buf.Len() < 44 {
		t.Fatalf("WAV data too small: %d bytes", buf.Len())
	}
	// Verify RIFF header.
	if string(buf.Bytes()[:4]) != "RIFF" {
		t.Fatal("Missing RIFF header")
	}
}

func TestLoadWAV_NotExist(t *testing.T) {
	_, err := LoadWAV("nonexistent.wav")
	if err == nil {
		t.Fatal("LoadWAV with nonexistent file should fail")
	}
}

func TestReadWAV_InvalidData(t *testing.T) {
	_, err := ReadWAV(bytes.NewReader([]byte("not a wav file")))
	if err == nil {
		t.Fatal("ReadWAV with invalid data should fail")
	}
}

func TestReadWAV_NotRIFF(t *testing.T) {
	_, err := ReadWAV(bytes.NewReader([]byte("RIFX\x00\x00\x00\x00WAVE")))
	if err == nil {
		t.Fatal("ReadWAV with non-RIFF should fail")
	}
}

// ──────────────────────────────────────────────
// Volume / gain
// ──────────────────────────────────────────────

func TestAdjustVolume(t *testing.T) {
	audio := createTestTone()
	louder := AdjustVolume(audio, 2.0)
	// Check that amplitude roughly doubled.
	origMax := maxAbs(audio.Samples[0])
	newMax := maxAbs(louder.Samples[0])
	if newMax < origMax*1.9 {
		t.Fatalf("volume not doubled: %f -> %f", origMax, newMax)
	}
}

func TestAdjustVolume_Zero(t *testing.T) {
	audio := createTestTone()
	silent := AdjustVolume(audio, 0)
	for _, s := range silent.Samples[0] {
		if s != 0 {
			t.Fatal("zero volume should produce silence")
		}
	}
}

func TestNormalize(t *testing.T) {
	audio := createTestTone()
	normalized := Normalize(audio, 1.0)
	peak := maxAbs(normalized.Samples[0])
	if peak < 0.99 {
		t.Fatalf("normalized peak = %f, want ~1.0", peak)
	}
}

func TestNormalize_Silence(t *testing.T) {
	audio := GenerateSilence(0.1, 8000, 1)
	normalized := Normalize(audio, 1.0)
	// Should not crash, should remain silent.
	for _, s := range normalized.Samples[0] {
		if s != 0 {
			t.Fatal("normalize of silence should remain silent")
		}
	}
}

func TestFadeIn(t *testing.T) {
	audio := createTestTone()
	faded := FadeIn(audio, 0.05) // half the duration
	// First sample should be ~0.
	if math.Abs(faded.Samples[0][0]) > 0.01 {
		t.Fatalf("first sample after fade-in = %f, should be ~0", faded.Samples[0][0])
	}
}

func TestFadeOut(t *testing.T) {
	audio := createTestTone()
	faded := FadeOut(audio, 0.05)
	// Last sample should be ~0.
	last := faded.Samples[0][faded.NumSamples()-1]
	if math.Abs(last) > 0.01 {
		t.Fatalf("last sample after fade-out = %f, should be ~0", last)
	}
}

// ──────────────────────────────────────────────
// Trim / cut
// ──────────────────────────────────────────────

func TestTrim(t *testing.T) {
	audio := createTestTone()            // 0.1s = 800 samples at 8000Hz
	trimmed := Trim(audio, 0.025, 0.025) // remove 25ms from each end
	want := 800 - 200 - 200
	if trimmed.NumSamples() != want {
		t.Fatalf("Trim NumSamples = %d, want %d", trimmed.NumSamples(), want)
	}
}

func TestTrim_Full(t *testing.T) {
	audio := createTestTone()
	trimmed := Trim(audio, 0, 0)
	if trimmed.NumSamples() != audio.NumSamples() {
		t.Fatal("Trim(0, 0) should return full audio")
	}
}

func TestTrimSilence(t *testing.T) {
	// Create audio with silence at start and end.
	audio := GenerateTone(440, 0.1, 8000)
	full := NewAudio(8000, 1, Format16Bit, 1600) // 0.2s
	copy(full.Samples[0][400:1200], audio.Samples[0][:800])
	// First 400 and last 400 samples are silence.

	trimmed := TrimSilence(full, -40)
	// Allow off-by-one due to sine wave zero-crossings at boundaries.
	n := trimmed.NumSamples()
	if n < 798 || n > 800 {
		t.Fatalf("TrimSilence NumSamples = %d, want ~800", n)
	}
}

func TestTrimSilence_AllSilence(t *testing.T) {
	audio := GenerateSilence(0.1, 8000, 1)
	trimmed := TrimSilence(audio, -40)
	if trimmed.NumSamples() != 0 {
		t.Fatalf("TrimSilence of all silence = %d, want 0", trimmed.NumSamples())
	}
}

func TestCut(t *testing.T) {
	audio := createTestTone()
	cut := Cut(audio, 0.025, 0.075) // keep middle 50ms
	want := 400
	if cut.NumSamples() != want {
		t.Fatalf("Cut NumSamples = %d, want %d", cut.NumSamples(), want)
	}
}

// ──────────────────────────────────────────────
// Mix / merge
// ──────────────────────────────────────────────

func TestMix(t *testing.T) {
	a := GenerateTone(440, 0.05, 8000)
	b := GenerateTone(880, 0.05, 8000)
	mixed := Mix(a, b)
	if mixed.NumSamples() != a.NumSamples() {
		t.Fatalf("Mix NumSamples = %d, want %d", mixed.NumSamples(), a.NumSamples())
	}
}

func TestMix_DifferentLengths(t *testing.T) {
	a := GenerateTone(440, 0.05, 8000) // 400 samples
	b := GenerateTone(880, 0.1, 8000)  // 800 samples
	mixed := Mix(a, b)
	if mixed.NumSamples() != 800 {
		t.Fatalf("Mix NumSamples = %d, want 800", mixed.NumSamples())
	}
}

func TestConcatenate(t *testing.T) {
	a := GenerateTone(440, 0.05, 8000) // 400 samples
	b := GenerateTone(880, 0.05, 8000)
	result, err := Concatenate(a, b)
	if err != nil {
		t.Fatalf("Concatenate failed: %v", err)
	}
	if result.NumSamples() != 800 {
		t.Fatalf("Concatenate NumSamples = %d, want 800", result.NumSamples())
	}
}

func TestConcatenate_SampleRateMismatch(t *testing.T) {
	a := GenerateTone(440, 0.05, 8000)
	b := GenerateTone(880, 0.05, 16000)
	_, err := Concatenate(a, b)
	if err == nil {
		t.Fatal("Concatenate with mismatched sample rates should fail")
	}
}

func TestConcatenate_ChannelMismatch(t *testing.T) {
	a := GenerateTone(440, 0.05, 8000)
	b := ToStereo(GenerateTone(880, 0.05, 8000))
	_, err := Concatenate(a, b)
	if err == nil {
		t.Fatal("Concatenate with mismatched channels should fail")
	}
}

// ──────────────────────────────────────────────
// Channel conversion
// ──────────────────────────────────────────────

func TestToMono(t *testing.T) {
	stereo := ToStereo(createTestTone())
	mono := ToMono(stereo)
	if mono.Channels != 1 {
		t.Fatalf("ToMono Channels = %d, want 1", mono.Channels)
	}
}

func TestToMono_AlreadyMono(t *testing.T) {
	audio := createTestTone()
	mono := ToMono(audio)
	if mono != audio {
		t.Fatal("ToMono on mono should return same audio")
	}
}

func TestToStereo(t *testing.T) {
	audio := createTestTone()
	stereo := ToStereo(audio)
	if stereo.Channels != 2 {
		t.Fatalf("ToStereo Channels = %d, want 2", stereo.Channels)
	}
}

func TestToStereo_AlreadyStereo(t *testing.T) {
	stereo := ToStereo(createTestTone())
	result := ToStereo(stereo)
	if result != stereo {
		t.Fatal("ToStereo on stereo should return same audio")
	}
}

// ──────────────────────────────────────────────
// Resample
// ──────────────────────────────────────────────

func TestResample_Upsample(t *testing.T) {
	audio := createTestTone() // 8000Hz
	resampled := Resample(audio, 16000)
	if resampled.SampleRate != 16000 {
		t.Fatalf("SampleRate = %d, want 16000", resampled.SampleRate)
	}
	want := audio.NumSamples() * 2
	if resampled.NumSamples() != want {
		t.Fatalf("Resample NumSamples = %d, want %d", resampled.NumSamples(), want)
	}
}

func TestResample_Downsample(t *testing.T) {
	audio := createTestTone() // 8000Hz, 800 samples
	resampled := Resample(audio, 4000)
	if resampled.SampleRate != 4000 {
		t.Fatalf("SampleRate = %d, want 4000", resampled.SampleRate)
	}
	want := audio.NumSamples() / 2
	if resampled.NumSamples() != want {
		t.Fatalf("Resample NumSamples = %d, want %d", resampled.NumSamples(), want)
	}
}

func TestResample_SameRate(t *testing.T) {
	audio := createTestTone()
	result := Resample(audio, 8000)
	if result != audio {
		t.Fatal("Resample with same rate should return same audio")
	}
}

// ──────────────────────────────────────────────
// Effects
// ──────────────────────────────────────────────

func TestReverse(t *testing.T) {
	audio := createTestTone()
	reversed := Reverse(audio)
	// First sample of reversed should equal last sample of original.
	if reversed.Samples[0][0] != audio.Samples[0][audio.NumSamples()-1] {
		t.Fatal("Reverse: first sample should equal last of original")
	}
}

func TestChangeSpeed(t *testing.T) {
	audio := createTestTone()
	faster := ChangeSpeed(audio, 2.0)
	// Faster playback means fewer samples (at same perceived rate).
	// Actually, ChangeSpeed resamples to lower rate, so NumSamples stays same
	// but SampleRate changes. Let's check SampleRate.
	if faster.SampleRate != 4000 {
		t.Fatalf("ChangeSpeed SampleRate = %d, want 4000", faster.SampleRate)
	}
}

// ──────────────────────────────────────────────
// Bit depth conversion
// ──────────────────────────────────────────────

func TestConvertFormat(t *testing.T) {
	audio := createTestTone()
	converted := ConvertFormat(audio, Format24Bit)
	if converted.Format != Format24Bit {
		t.Fatalf("Format = %d, want 24", converted.Format)
	}
}

// ──────────────────────────────────────────────
// Info
// ──────────────────────────────────────────────

func TestGetInfo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.wav")
	audio := createTestTone()
	if err := SaveWAV(audio, path); err != nil {
		t.Fatalf("SaveWAV failed: %v", err)
	}
	info, err := GetInfo(path)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}
	if info.SampleRate != 8000 {
		t.Fatalf("SampleRate = %d, want 8000", info.SampleRate)
	}
	if info.Channels != 1 {
		t.Fatalf("Channels = %d, want 1", info.Channels)
	}
	if info.BitDepth != 16 {
		t.Fatalf("BitDepth = %d, want 16", info.BitDepth)
	}
}

func TestGetInfo_NotExist(t *testing.T) {
	_, err := GetInfo("nonexistent.wav")
	if err == nil {
		t.Fatal("GetInfo with nonexistent file should fail")
	}
}

// ──────────────────────────────────────────────
// Generation
// ──────────────────────────────────────────────

func TestGenerateTone(t *testing.T) {
	audio := GenerateTone(440, 0.1, 8000)
	if audio.NumSamples() != 800 {
		t.Fatalf("NumSamples = %d, want 800", audio.NumSamples())
	}
	if audio.SampleRate != 8000 {
		t.Fatalf("SampleRate = %d, want 8000", audio.SampleRate)
	}
	// Check that it's actually a sine wave (non-zero amplitude).
	peak := maxAbs(audio.Samples[0])
	if peak < 0.1 {
		t.Fatalf("tone peak = %f, should be ~0.5", peak)
	}
}

func TestGenerateSilence(t *testing.T) {
	audio := GenerateSilence(0.1, 8000, 2)
	if audio.NumSamples() != 800 {
		t.Fatalf("NumSamples = %d, want 800", audio.NumSamples())
	}
	if audio.Channels != 2 {
		t.Fatalf("Channels = %d, want 2", audio.Channels)
	}
	for _, ch := range audio.Samples {
		for _, s := range ch {
			if s != 0 {
				t.Fatal("silence should be all zeros")
			}
		}
	}
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

func TestCopy(t *testing.T) {
	audio := createTestTone()
	copied := Copy(audio)
	if copied == audio {
		t.Fatal("Copy should return a different pointer")
	}
	if copied.NumSamples() != audio.NumSamples() {
		t.Fatal("Copy should have same number of samples")
	}
	// Modify copy, original should be unchanged.
	copied.Samples[0][0] = 999
	if audio.Samples[0][0] == 999 {
		t.Fatal("Copy should be deep copy")
	}
}

func TestNewAudio(t *testing.T) {
	audio := NewAudio(44100, 2, Format16Bit, 1000)
	if audio.SampleRate != 44100 {
		t.Fatalf("SampleRate = %d", audio.SampleRate)
	}
	if audio.Channels != 2 {
		t.Fatalf("Channels = %d", audio.Channels)
	}
	if audio.NumSamples() != 1000 {
		t.Fatalf("NumSamples = %d", audio.NumSamples())
	}
}

func TestDescribe(t *testing.T) {
	audio := createTestTone()
	s := Describe(audio)
	if s == "" {
		t.Fatal("Describe should not be empty")
	}
}

func TestSaveWAV_CreateError(t *testing.T) {
	audio := createTestTone()
	// Try to save in a nonexistent directory.
	err := SaveWAV(audio, "/nonexistent/path/test.wav")
	if err == nil {
		t.Fatal("SaveWAV to nonexistent path should fail")
	}
}

// ──────────────────────────────────────────────
// Integration: full pipeline
// ──────────────────────────────────────────────

func TestFullPipeline(t *testing.T) {
	// Generate → Fade in → Normalize → Save → Load → Trim → Save
	audio := GenerateTone(440, 0.2, 8000)
	audio = FadeIn(audio, 0.05)
	audio = Normalize(audio, 0.9)

	dir := t.TempDir()
	path1 := filepath.Join(dir, "step1.wav")
	if err := SaveWAV(audio, path1); err != nil {
		t.Fatalf("SaveWAV step1: %v", err)
	}

	loaded, err := LoadWAV(path1)
	if err != nil {
		t.Fatalf("LoadWAV: %v", err)
	}

	trimmed := Trim(loaded, 0.05, 0.05)
	path2 := filepath.Join(dir, "step2.wav")
	if err := SaveWAV(trimmed, path2); err != nil {
		t.Fatalf("SaveWAV step2: %v", err)
	}

	// Verify final file.
	info, err := GetInfo(path2)
	if err != nil {
		t.Fatalf("GetInfo: %v", err)
	}
	if info.NumSamples != 800 { // 0.2s - 0.1s = 0.1s = 800 samples
		t.Fatalf("final NumSamples = %d, want 800", info.NumSamples)
	}

	// Cleanup.
	os.Remove(path1)
	os.Remove(path2)
}

// maxAbs returns the maximum absolute value in a slice.
func maxAbs(s []float64) float64 {
	max := 0.0
	for _, v := range s {
		abs := math.Abs(v)
		if abs > max {
			max = abs
		}
	}
	return max
}
