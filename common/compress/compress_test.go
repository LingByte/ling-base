// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package compress

import (
	"bytes"
	"strings"
	"testing"
)

// ──────────────────────────────────────────────
// Gzip
// ──────────────────────────────────────────────

func TestGzipCompressDecompress(t *testing.T) {
	data := []byte("hello world hello world hello world hello world")
	compressed, err := GzipCompress(data, LevelDefault)
	if err != nil {
		t.Fatalf("GzipCompress failed: %v", err)
	}
	if len(compressed) >= len(data) {
		// Small data may not compress well, but let's check.
		t.Logf("warning: compressed (%d) >= original (%d)", len(compressed), len(data))
	}
	decompressed, err := GzipDecompress(compressed)
	if err != nil {
		t.Fatalf("GzipDecompress failed: %v", err)
	}
	if !bytes.Equal(data, decompressed) {
		t.Fatalf("decompressed = %q, want %q", decompressed, data)
	}
}

func TestGzipCompressDefault(t *testing.T) {
	data := []byte(strings.Repeat("test ", 100))
	compressed, err := GzipCompressDefault(data)
	if err != nil {
		t.Fatalf("GzipCompressDefault failed: %v", err)
	}
	decompressed, err := GzipDecompress(compressed)
	if err != nil {
		t.Fatalf("GzipDecompress failed: %v", err)
	}
	if !bytes.Equal(data, decompressed) {
		t.Fatal("round-trip failed")
	}
}

func TestGzipCompress_LargeData(t *testing.T) {
	data := make([]byte, 10000)
	for i := range data {
		data[i] = byte(i % 256)
	}
	compressed, err := GzipCompress(data, LevelBest)
	if err != nil {
		t.Fatalf("GzipCompress failed: %v", err)
	}
	decompressed, err := GzipDecompress(compressed)
	if err != nil {
		t.Fatalf("GzipDecompress failed: %v", err)
	}
	if !bytes.Equal(data, decompressed) {
		t.Fatal("round-trip failed for large data")
	}
}

func TestGzipCompress_Empty(t *testing.T) {
	compressed, err := GzipCompress(nil, LevelDefault)
	if err != nil {
		t.Fatalf("GzipCompress failed: %v", err)
	}
	decompressed, err := GzipDecompress(compressed)
	if err != nil {
		t.Fatalf("GzipDecompress failed: %v", err)
	}
	if len(decompressed) != 0 {
		t.Fatalf("decompressed = %d bytes, want 0", len(decompressed))
	}
}

func TestGzipDecompress_InvalidData(t *testing.T) {
	_, err := GzipDecompress([]byte("not gzip data"))
	if err == nil {
		t.Fatal("GzipDecompress of invalid data should fail")
	}
}

func TestGzipCompress_InvalidLevel(t *testing.T) {
	_, err := GzipCompress([]byte("test"), 100)
	if err == nil {
		t.Fatal("GzipCompress with invalid level should fail")
	}
}

func TestNewGzipWriterReader(t *testing.T) {
	var buf bytes.Buffer
	w, err := NewGzipWriter(&buf, LevelDefault)
	if err != nil {
		t.Fatalf("NewGzipWriter failed: %v", err)
	}
	w.Write([]byte("hello"))
	w.Close()

	r, err := NewGzipReader(&buf)
	if err != nil {
		t.Fatalf("NewGzipReader failed: %v", err)
	}
	defer r.Close()
	data := make([]byte, 5)
	r.Read(data)
	if string(data) != "hello" {
		t.Fatalf("data = %q", data)
	}
}

// ──────────────────────────────────────────────
// Zlib
// ──────────────────────────────────────────────

func TestZlibCompressDecompress(t *testing.T) {
	data := []byte(strings.Repeat("zlib test ", 50))
	compressed, err := ZlibCompress(data, LevelDefault)
	if err != nil {
		t.Fatalf("ZlibCompress failed: %v", err)
	}
	decompressed, err := ZlibDecompress(compressed)
	if err != nil {
		t.Fatalf("ZlibDecompress failed: %v", err)
	}
	if !bytes.Equal(data, decompressed) {
		t.Fatal("round-trip failed")
	}
}

func TestZlibDecompress_InvalidData(t *testing.T) {
	_, err := ZlibDecompress([]byte("not zlib data"))
	if err == nil {
		t.Fatal("ZlibDecompress of invalid data should fail")
	}
}

// ──────────────────────────────────────────────
// Flate
// ──────────────────────────────────────────────

func TestFlateCompressDecompress(t *testing.T) {
	data := []byte(strings.Repeat("flate test ", 50))
	compressed, err := FlateCompress(data, LevelDefault)
	if err != nil {
		t.Fatalf("FlateCompress failed: %v", err)
	}
	decompressed, err := FlateDecompress(compressed)
	if err != nil {
		t.Fatalf("FlateDecompress failed: %v", err)
	}
	if !bytes.Equal(data, decompressed) {
		t.Fatal("round-trip failed")
	}
}

// ──────────────────────────────────────────────
// Zstd (fallback)
// ──────────────────────────────────────────────

func TestZstdCompressDecompress(t *testing.T) {
	data := []byte(strings.Repeat("zstd test ", 50))
	compressed, err := ZstdCompress(data)
	if err != nil {
		t.Fatalf("ZstdCompress failed: %v", err)
	}
	decompressed, err := ZstdDecompress(compressed)
	if err != nil {
		t.Fatalf("ZstdDecompress failed: %v", err)
	}
	if !bytes.Equal(data, decompressed) {
		t.Fatal("round-trip failed")
	}
}

// ──────────────────────────────────────────────
// Snappy (fallback)
// ──────────────────────────────────────────────

func TestSnappyCompressDecompress(t *testing.T) {
	data := []byte(strings.Repeat("snappy test ", 50))
	compressed, err := SnappyCompress(data)
	if err != nil {
		t.Fatalf("SnappyCompress failed: %v", err)
	}
	decompressed, err := SnappyDecompress(compressed)
	if err != nil {
		t.Fatalf("SnappyDecompress failed: %v", err)
	}
	if !bytes.Equal(data, decompressed) {
		t.Fatal("round-trip failed")
	}
}

// ──────────────────────────────────────────────
// LZ4 (fallback)
// ──────────────────────────────────────────────

func TestLZ4CompressDecompress(t *testing.T) {
	data := []byte(strings.Repeat("lz4 test ", 50))
	compressed, err := LZ4Compress(data)
	if err != nil {
		t.Fatalf("LZ4Compress failed: %v", err)
	}
	decompressed, err := LZ4Decompress(compressed)
	if err != nil {
		t.Fatalf("LZ4Decompress failed: %v", err)
	}
	if !bytes.Equal(data, decompressed) {
		t.Fatal("round-trip failed")
	}
}

// ──────────────────────────────────────────────
// Generic Compress/Decompress
// ──────────────────────────────────────────────

func TestCompressDecompress_AllAlgorithms(t *testing.T) {
	data := []byte(strings.Repeat("generic test data ", 30))
	algorithms := []Algorithm{AlgGzip, AlgZlib, AlgFlate, AlgZstd, AlgSnappy, AlgLZ4}

	for _, alg := range algorithms {
		t.Run(string(alg), func(t *testing.T) {
			compressed, err := Compress(alg, data, LevelDefault)
			if err != nil {
				t.Fatalf("Compress(%s) failed: %v", alg, err)
			}
			decompressed, err := Decompress(alg, compressed)
			if err != nil {
				t.Fatalf("Decompress(%s) failed: %v", alg, err)
			}
			if !bytes.Equal(data, decompressed) {
				t.Fatalf("Decompress(%s) round-trip failed", alg)
			}
		})
	}
}

func TestCompress_UnknownAlgorithm(t *testing.T) {
	_, err := Compress(Algorithm("unknown"), []byte("test"), LevelDefault)
	if err == nil {
		t.Fatal("Compress with unknown algorithm should fail")
	}
}

func TestDecompress_UnknownAlgorithm(t *testing.T) {
	_, err := Decompress(Algorithm("unknown"), []byte("test"))
	if err == nil {
		t.Fatal("Decompress with unknown algorithm should fail")
	}
}

// ──────────────────────────────────────────────
// Stream helpers
// ──────────────────────────────────────────────

func TestGzipCompressStream(t *testing.T) {
	input := []byte(strings.Repeat("stream test ", 50))
	var compressed bytes.Buffer
	err := GzipCompressStream(&compressed, bytes.NewReader(input), LevelDefault)
	if err != nil {
		t.Fatalf("GzipCompressStream failed: %v", err)
	}

	var decompressed bytes.Buffer
	err = GzipDecompressStream(&decompressed, &compressed)
	if err != nil {
		t.Fatalf("GzipDecompressStream failed: %v", err)
	}
	if !bytes.Equal(input, decompressed.Bytes()) {
		t.Fatal("stream round-trip failed")
	}
}

// ──────────────────────────────────────────────
// Utility
// ──────────────────────────────────────────────

func TestRatio(t *testing.T) {
	r := Ratio(1000, 300)
	if r != 30 {
		t.Fatalf("Ratio(1000, 300) = %f, want 30", r)
	}
}

func TestRatio_ZeroOriginal(t *testing.T) {
	if Ratio(0, 100) != 0 {
		t.Fatal("Ratio(0, 100) should be 0")
	}
}

func TestBestAlgorithm(t *testing.T) {
	data := []byte(strings.Repeat("best algorithm test ", 100))
	alg, compressed, err := BestAlgorithm(data)
	if err != nil {
		t.Fatalf("BestAlgorithm failed: %v", err)
	}
	if alg == "" {
		t.Fatal("BestAlgorithm returned empty algorithm")
	}
	if len(compressed) == 0 {
		t.Fatal("BestAlgorithm returned empty compressed data")
	}
	// Verify we can decompress with the returned algorithm.
	decompressed, err := Decompress(alg, compressed)
	if err != nil {
		t.Fatalf("Decompress with best alg failed: %v", err)
	}
	if !bytes.Equal(data, decompressed) {
		t.Fatal("BestAlgorithm round-trip failed")
	}
}

func TestBestAlgorithm_SmallData(t *testing.T) {
	// Very small data may not compress well.
	_, _, err := BestAlgorithm([]byte("x"))
	if err != nil {
		t.Fatalf("BestAlgorithm with small data failed: %v", err)
	}
}
