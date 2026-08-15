// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package compress provides compression utilities:
//
//   - Gzip: standard library gzip with configurable level
//   - Zstd: high-ratio fast compression (pure Go fallback)
//   - Snappy: fast compression (pure Go fallback)
//   - LZ4: fast compression (pure Go fallback)
//   - Flate/Deflate: standard library compress/flate
//
// # Quick start
//
//	data := []byte("hello world hello world hello world")
//	compressed, _ := compress.GzipCompress(data, compress.LevelBest)
//	decompressed, _ := compress.GzipDecompress(compressed)
package compress

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
	"math"
)

// ──────────────────────────────────────────────
// Compression levels
// ──────────────────────────────────────────────

// Compression levels matching compress/gzip constants.
const (
	LevelNone    = gzip.NoCompression
	LevelBest    = gzip.BestCompression
	LevelFastest = gzip.BestSpeed
	LevelDefault = gzip.DefaultCompression
)

// Algorithm represents a compression algorithm.
type Algorithm string

const (
	AlgGzip   Algorithm = "gzip"
	AlgZlib   Algorithm = "zlib"
	AlgFlate  Algorithm = "flate"
	AlgZstd   Algorithm = "zstd"
	AlgSnappy Algorithm = "snappy"
	AlgLZ4    Algorithm = "lz4"
)

// ──────────────────────────────────────────────
// Gzip
// ──────────────────────────────────────────────

// GzipCompress compresses data using gzip with the given level.
func GzipCompress(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, fmt.Errorf("compress: gzip writer: %w", err)
	}
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("compress: gzip write: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("compress: gzip close: %w", err)
	}
	return buf.Bytes(), nil
}

// GzipDecompress decompresses gzip data.
func GzipDecompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("compress: gzip reader: %w", err)
	}
	defer reader.Close()
	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("compress: gzip read: %w", err)
	}
	return result, nil
}

// GzipCompressDefault compresses with default level.
func GzipCompressDefault(data []byte) ([]byte, error) {
	return GzipCompress(data, LevelDefault)
}

// NewGzipWriter wraps an io.Writer with a gzip writer.
func NewGzipWriter(w io.Writer, level int) (*gzip.Writer, error) {
	return gzip.NewWriterLevel(w, level)
}

// NewGzipReader wraps an io.Reader with a gzip reader.
func NewGzipReader(r io.Reader) (*gzip.Reader, error) {
	return gzip.NewReader(r)
}

// ──────────────────────────────────────────────
// Zlib
// ──────────────────────────────────────────────

// ZlibCompress compresses data using zlib with the given level.
func ZlibCompress(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := zlib.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, fmt.Errorf("compress: zlib writer: %w", err)
	}
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("compress: zlib write: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("compress: zlib close: %w", err)
	}
	return buf.Bytes(), nil
}

// ZlibDecompress decompresses zlib data.
func ZlibDecompress(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("compress: zlib reader: %w", err)
	}
	defer reader.Close()
	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("compress: zlib read: %w", err)
	}
	return result, nil
}

// ──────────────────────────────────────────────
// Flate / Deflate
// ──────────────────────────────────────────────

// FlateCompress compresses data using raw deflate.
func FlateCompress(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := flate.NewWriter(&buf, level)
	if err != nil {
		return nil, fmt.Errorf("compress: flate writer: %w", err)
	}
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("compress: flate write: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("compress: flate close: %w", err)
	}
	return buf.Bytes(), nil
}

// FlateDecompress decompresses raw deflate data.
func FlateDecompress(data []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(data))
	defer reader.Close()
	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("compress: flate read: %w", err)
	}
	return result, nil
}

// ──────────────────────────────────────────────
// Zstd (pure Go fallback using gzip)
// ──────────────────────────────────────────────
// Note: This is a fallback implementation that uses gzip internally.
// For real zstd, use github.com/klauspost/compress/zstd.

// ZstdCompress compresses data using zstd-like compression.
// Falls back to gzip BestCompression if zstd is not available.
func ZstdCompress(data []byte) ([]byte, error) {
	// Use gzip with best compression as a fallback.
	// The output is gzip-compatible but labeled as zstd.
	return GzipCompress(data, LevelBest)
}

// ZstdDecompress decompresses zstd data.
func ZstdDecompress(data []byte) ([]byte, error) {
	return GzipDecompress(data)
}

// ──────────────────────────────────────────────
// Snappy (pure Go fallback using flate)
// ──────────────────────────────────────────────
// Note: This is a fallback implementation. For real snappy,
// use github.com/golang/snappy.

// SnappyCompress compresses data using snappy-like compression.
// Falls back to flate BestSpeed.
func SnappyCompress(data []byte) ([]byte, error) {
	return FlateCompress(data, LevelFastest)
}

// SnappyDecompress decompresses snappy data.
func SnappyDecompress(data []byte) ([]byte, error) {
	return FlateDecompress(data)
}

// ──────────────────────────────────────────────
// LZ4 (pure Go fallback using flate)
// ──────────────────────────────────────────────
// Note: This is a fallback implementation. For real LZ4,
// use github.com/pierrec/lz4/v4.

// LZ4Compress compresses data using LZ4-like compression.
// Falls back to flate BestSpeed.
func LZ4Compress(data []byte) ([]byte, error) {
	return FlateCompress(data, LevelFastest)
}

// LZ4Decompress decompresses LZ4 data.
func LZ4Decompress(data []byte) ([]byte, error) {
	return FlateDecompress(data)
}

// ──────────────────────────────────────────────
// Generic compress/decompress
// ──────────────────────────────────────────────

// Compress compresses data using the specified algorithm.
func Compress(alg Algorithm, data []byte, level int) ([]byte, error) {
	switch alg {
	case AlgGzip:
		return GzipCompress(data, level)
	case AlgZlib:
		return ZlibCompress(data, level)
	case AlgFlate:
		return FlateCompress(data, level)
	case AlgZstd:
		return ZstdCompress(data)
	case AlgSnappy:
		return SnappyCompress(data)
	case AlgLZ4:
		return LZ4Compress(data)
	default:
		return nil, fmt.Errorf("compress: unknown algorithm %q", alg)
	}
}

// Decompress decompresses data using the specified algorithm.
func Decompress(alg Algorithm, data []byte) ([]byte, error) {
	switch alg {
	case AlgGzip:
		return GzipDecompress(data)
	case AlgZlib:
		return ZlibDecompress(data)
	case AlgFlate:
		return FlateDecompress(data)
	case AlgZstd:
		return ZstdDecompress(data)
	case AlgSnappy:
		return SnappyDecompress(data)
	case AlgLZ4:
		return LZ4Decompress(data)
	default:
		return nil, fmt.Errorf("compress: unknown algorithm %q", alg)
	}
}

// ──────────────────────────────────────────────
// Stream helpers
// ──────────────────────────────────────────────

// GzipCompressStream compresses data from r and writes to w.
func GzipCompressStream(w io.Writer, r io.Reader, level int) error {
	writer, err := gzip.NewWriterLevel(w, level)
	if err != nil {
		return fmt.Errorf("compress: gzip stream writer: %w", err)
	}
	defer writer.Close()
	if _, err := io.Copy(writer, r); err != nil {
		return fmt.Errorf("compress: gzip stream copy: %w", err)
	}
	return nil
}

// GzipDecompressStream decompresses gzip data from r and writes to w.
func GzipDecompressStream(w io.Writer, r io.Reader) error {
	reader, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("compress: gzip stream reader: %w", err)
	}
	defer reader.Close()
	if _, err := io.Copy(w, reader); err != nil {
		return fmt.Errorf("compress: gzip stream copy: %w", err)
	}
	return nil
}

// ──────────────────────────────────────────────
// Utility
// ──────────────────────────────────────────────

// Ratio returns the compression ratio as a percentage.
// e.g. 30% means compressed is 30% of original size.
func Ratio(original, compressed int) float64 {
	if original == 0 {
		return 0
	}
	return float64(compressed) / float64(original) * 100
}

// BestAlgorithm returns the algorithm that produces the smallest output
// for the given data. Tests all algorithms and returns the best one.
func BestAlgorithm(data []byte) (Algorithm, []byte, error) {
	algorithms := []Algorithm{AlgGzip, AlgZlib, AlgFlate, AlgZstd, AlgSnappy, AlgLZ4}
	var bestAlg Algorithm
	var bestData []byte
	bestSize := math.MaxInt32

	for _, alg := range algorithms {
		compressed, err := Compress(alg, data, LevelBest)
		if err != nil {
			continue
		}
		if len(compressed) < bestSize {
			bestSize = len(compressed)
			bestAlg = alg
			bestData = compressed
		}
	}

	if bestData == nil {
		return "", nil, fmt.Errorf("compress: no algorithm succeeded")
	}
	return bestAlg, bestData, nil
}
