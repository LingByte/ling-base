// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package audioutil

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestIsMP3Supported(t *testing.T) {
	if !IsMP3Supported() {
		t.Fatal("MP3 should be supported")
	}
}

func TestLoad_UnsupportedFormat(t *testing.T) {
	_, err := Load("test.xyz")
	if err == nil {
		t.Fatal("Load with unsupported format should fail")
	}
}

func TestSave_UnsupportedFormat(t *testing.T) {
	audio := createTestTone()
	err := Save(audio, filepath.Join(t.TempDir(), "test.xyz"))
	if err == nil {
		t.Fatal("Save with unsupported format should fail")
	}
}

func TestSave_MP3NotSupported(t *testing.T) {
	audio := createTestTone()
	err := Save(audio, filepath.Join(t.TempDir(), "test.mp3"))
	if err == nil {
		t.Fatal("Save as MP3 should fail (encoding not supported)")
	}
}

func TestLoad_MP3_NotExist(t *testing.T) {
	_, err := LoadMP3("nonexistent.mp3")
	if err == nil {
		t.Fatal("LoadMP3 with nonexistent file should fail")
	}
}

func TestReadMP3_InvalidData(t *testing.T) {
	_, err := ReadMP3(bytes.NewReader([]byte("not mp3 data")))
	if err == nil {
		t.Fatal("ReadMP3 with invalid data should fail")
	}
}

func TestLoadMP3_RealFile(t *testing.T) {
	audio, err := LoadMP3("testdata/test.mp3")
	if err != nil {
		t.Fatalf("LoadMP3 failed: %v", err)
	}
	if audio.SampleRate != 44100 {
		t.Fatalf("SampleRate = %d, want 44100", audio.SampleRate)
	}
	if audio.Channels < 1 {
		t.Fatalf("Channels = %d, want >= 1", audio.Channels)
	}
	if audio.NumSamples() == 0 {
		t.Fatal("NumSamples = 0, expected decoded audio")
	}
}

func TestReadMP3_EmptyData(t *testing.T) {
	_, err := ReadMP3(bytes.NewReader([]byte{}))
	if err == nil {
		t.Fatal("ReadMP3 with empty data should fail")
	}
}

func TestLoad_MP3ViaLoad(t *testing.T) {
	audio, err := Load("testdata/test.mp3")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if audio.SampleRate != 44100 {
		t.Fatalf("SampleRate = %d, want 44100", audio.SampleRate)
	}
}

func TestLoad_WAVViaLoad(t *testing.T) {
	audio := createTestTone()
	path := filepath.Join(t.TempDir(), "test.wav")
	if err := SaveWAV(audio, path); err != nil {
		t.Fatalf("SaveWAV failed: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.SampleRate != audio.SampleRate {
		t.Fatalf("SampleRate = %d, want %d", loaded.SampleRate, audio.SampleRate)
	}
}

func TestSave_WAVViaSave(t *testing.T) {
	audio := createTestTone()
	path := filepath.Join(t.TempDir(), "test.wav")
	if err := Save(audio, path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	loaded, err := LoadWAV(path)
	if err != nil {
		t.Fatalf("LoadWAV failed: %v", err)
	}
	if loaded.NumSamples() != audio.NumSamples() {
		t.Fatalf("NumSamples = %d, want %d", loaded.NumSamples(), audio.NumSamples())
	}
}
