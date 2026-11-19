package denoise

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"testing"
)

// TestProcessHandlesInputLargerThanTheInternalBuffer is a regression test for
// a heap overflow: the AEC/AGC scratch buffers only hold a single 20ms frame,
// but the processor used to write the whole input into them regardless of its
// length. Any chunk longer than 20ms corrupted the heap.
func TestProcessHandlesInputLargerThanTheInternalBuffer(t *testing.T) {
	processor, err := NewDenoiseProcessor(&DenoiseConfig{
		AECEnable:     true,
		AGCEnable:     true,
		SampleRate:    16000,
		Channels:      1,
		BitsPerSample: 16,
	})
	if err != nil {
		t.Fatalf("NewDenoiseProcessor: %v", err)
	}
	defer func() { _ = processor.Close() }()

	// 16kHz, 16 bit, mono: one 20ms frame is 320 samples = 640 bytes.
	// Feed ten times that so the input no longer fits in the scratch buffers.
	const frameBytes = 640
	const frames = 10
	input := make([]byte, frameBytes*frames)
	for i := range input {
		input[i] = byte(i % 256)
	}

	output, err := processor.Process(input)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(output) != len(input) {
		t.Fatalf("expected %d bytes out, got %d", len(input), len(output))
	}
}

// TestProcessInPlaceHandlesInputLargerThanTheInternalBuffer covers the same
// overflow for the in-place variant.
func TestProcessInPlaceHandlesInputLargerThanTheInternalBuffer(t *testing.T) {
	processor, err := NewDenoiseProcessor(&DenoiseConfig{
		AECEnable:     true,
		AGCEnable:     true,
		SampleRate:    16000,
		Channels:      1,
		BitsPerSample: 16,
	})
	if err != nil {
		t.Fatalf("NewDenoiseProcessor: %v", err)
	}
	defer func() { _ = processor.Close() }()

	const frameBytes = 640
	const frames = 10
	data := make([]byte, frameBytes*frames)
	for i := range data {
		data[i] = byte(i % 256)
	}

	if err := processor.ProcessInPlace(data); err != nil {
		t.Fatalf("ProcessInPlace: %v", err)
	}
}

// TestProcessWithoutAECOrAGC verifies the pass-through path still copies the
// input byte for byte.
func TestProcessWithoutAECOrAGC(t *testing.T) {
	processor, err := NewDenoiseProcessor(&DenoiseConfig{
		AECEnable:     false,
		AGCEnable:     false,
		SampleRate:    16000,
		Channels:      1,
		BitsPerSample: 16,
	})
	if err != nil {
		t.Fatalf("NewDenoiseProcessor: %v", err)
	}
	defer func() { _ = processor.Close() }()

	input := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	output, err := processor.Process(input)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(output) != len(input) {
		t.Fatalf("expected %d bytes out, got %d", len(input), len(output))
	}
	for i := range input {
		if output[i] != input[i] {
			t.Fatalf("byte %d: expected %d, got %d", i, input[i], output[i])
		}
	}
}

func TestProcessRejectsEmptyInput(t *testing.T) {
	processor, err := NewDenoiseProcessor(nil)
	if err != nil {
		t.Fatalf("NewDenoiseProcessor: %v", err)
	}
	defer func() { _ = processor.Close() }()

	if _, err := processor.Process(nil); err == nil {
		t.Fatal("expected an error for empty input")
	}
}

func TestVersion(t *testing.T) {
	if Version() == "" {
		t.Fatal("Version must not be empty")
	}
}
