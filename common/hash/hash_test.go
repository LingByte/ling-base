// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package hash

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// ──────────────────────────────────────────────
// Standard hashes
// ──────────────────────────────────────────────

func TestMD5(t *testing.T) {
	if got := MD5Hex([]byte("hello")); got != "5d41402abc4b2a76b9719d911017c592" {
		t.Fatalf("MD5Hex(hello) = %s", got)
	}
}

func TestMD5String(t *testing.T) {
	if got := MD5String("hello"); got != "5d41402abc4b2a76b9719d911017c592" {
		t.Fatalf("MD5String(hello) = %s", got)
	}
}

func TestMD5_Empty(t *testing.T) {
	if got := MD5Hex(nil); got != "d41d8cd98f00b204e9800998ecf8427e" {
		t.Fatalf("MD5Hex(empty) = %s", got)
	}
}

func TestSHA1(t *testing.T) {
	if got := SHA1Hex([]byte("hello")); got != "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d" {
		t.Fatalf("SHA1Hex(hello) = %s", got)
	}
}

func TestSHA1String(t *testing.T) {
	if got := SHA1String("hello"); got != "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d" {
		t.Fatalf("SHA1String(hello) = %s", got)
	}
}

func TestSHA256(t *testing.T) {
	if got := SHA256Hex([]byte("hello")); got != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("SHA256Hex(hello) = %s", got)
	}
}

func TestSHA256String(t *testing.T) {
	if got := SHA256String("hello"); got != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("SHA256String(hello) = %s", got)
	}
}

func TestSHA512(t *testing.T) {
	got := SHA512Hex([]byte("hello"))
	if len(got) != 128 {
		t.Fatalf("SHA512Hex length = %d, want 128", len(got))
	}
}

func TestSHA512String(t *testing.T) {
	got := SHA512String("hello")
	if len(got) != 128 {
		t.Fatalf("SHA512String length = %d, want 128", len(got))
	}
}

// ──────────────────────────────────────────────
// HMAC
// ──────────────────────────────────────────────

func TestHMACSHA256(t *testing.T) {
	// Known test vector
	data := []byte("hello")
	key := []byte("secret")
	got := HMACSHA256Hex(data, key)
	if len(got) != 64 {
		t.Fatalf("HMACSHA256Hex length = %d, want 64", len(got))
	}
}

func TestHMACSHA512(t *testing.T) {
	got := HMACSHA512Hex([]byte("hello"), []byte("secret"))
	if len(got) != 128 {
		t.Fatalf("HMACSHA512Hex length = %d, want 128", len(got))
	}
}

func TestHMACEqual(t *testing.T) {
	a := HMACSHA256([]byte("hello"), []byte("key"))
	b := HMACSHA256([]byte("hello"), []byte("key"))
	if !HMACEqual(a, b) {
		t.Fatal("HMACEqual should be true for same values")
	}
	c := HMACSHA256([]byte("hello"), []byte("different"))
	if HMACEqual(a, c) {
		t.Fatal("HMACEqual should be false for different values")
	}
}

// ──────────────────────────────────────────────
// CRC
// ──────────────────────────────────────────────

func TestCRC32(t *testing.T) {
	// CRC32 IEEE of "hello" = 0x3610a686
	got := CRC32([]byte("hello"))
	if got != 0x3610a686 {
		t.Fatalf("CRC32(hello) = 0x%08x, want 0x3610a686", got)
	}
}

func TestCRC32Hex(t *testing.T) {
	got := CRC32Hex([]byte("hello"))
	if got != "3610a686" {
		t.Fatalf("CRC32Hex(hello) = %s, want 3610a686", got)
	}
}

func TestCRC64(t *testing.T) {
	got := CRC64([]byte("hello"))
	if got == 0 {
		t.Fatal("CRC64 should not be 0")
	}
}

func TestCRC64Hex(t *testing.T) {
	got := CRC64Hex([]byte("hello"))
	if len(got) != 16 {
		t.Fatalf("CRC64Hex length = %d, want 16", len(got))
	}
}

func TestAdler32(t *testing.T) {
	got := Adler32([]byte("hello"))
	if got == 0 {
		t.Fatal("Adler32 should not be 0")
	}
}

func TestFNV1a32(t *testing.T) {
	got := FNV1a32([]byte("hello"))
	if got == 0 {
		t.Fatal("FNV1a32 should not be 0")
	}
}

func TestFNV1a64(t *testing.T) {
	got := FNV1a64([]byte("hello"))
	if got == 0 {
		t.Fatal("FNV1a64 should not be 0")
	}
}

// ──────────────────────────────────────────────
// MurmurHash3
// ──────────────────────────────────────────────

func TestMurmur3_32(t *testing.T) {
	// Known test vector: MurmurHash3_32("", 0) = 0
	if got := Murmur3_32(nil, 0); got != 0 {
		t.Fatalf("Murmur3_32(empty, 0) = 0x%08x, want 0", got)
	}
	// MurmurHash3_32("hello", 0) should be deterministic
	h1 := Murmur3_32([]byte("hello"), 0)
	h2 := Murmur3_32([]byte("hello"), 0)
	if h1 != h2 {
		t.Fatal("Murmur3_32 should be deterministic")
	}
	// Different seeds should (usually) produce different hashes
	h3 := Murmur3_32([]byte("hello"), 1)
	if h1 == h3 {
		t.Fatal("Murmur3_32 with different seeds should differ")
	}
}

func TestMurmur3_32Hex(t *testing.T) {
	got := Murmur3_32Hex([]byte("hello"), 0)
	if len(got) != 8 {
		t.Fatalf("Murmur3_32Hex length = %d, want 8", len(got))
	}
}

func TestMurmur3_128(t *testing.T) {
	// MurmurHash3_128("", 0) = (0, 0)
	h1, h2 := Murmur3_128(nil, 0)
	if h1 != 0 || h2 != 0 {
		t.Fatalf("Murmur3_128(empty, 0) = (%d, %d), want (0, 0)", h1, h2)
	}
	// Deterministic
	a1, a2 := Murmur3_128([]byte("hello"), 0)
	b1, b2 := Murmur3_128([]byte("hello"), 0)
	if a1 != b1 || a2 != b2 {
		t.Fatal("Murmur3_128 should be deterministic")
	}
}

func TestMurmur3_128Hex(t *testing.T) {
	got := Murmur3_128Hex([]byte("hello"), 0)
	if len(got) != 32 {
		t.Fatalf("Murmur3_128Hex length = %d, want 32", len(got))
	}
}

func TestMurmur3_DifferentInput(t *testing.T) {
	h1 := Murmur3_32([]byte("hello"), 0)
	h2 := Murmur3_32([]byte("world"), 0)
	if h1 == h2 {
		t.Fatal("Murmur3_32 should differ for different inputs")
	}
}

// ──────────────────────────────────────────────
// xxHash
// ──────────────────────────────────────────────

func TestXXHash64(t *testing.T) {
	// Known: xxHash64("", 0) = 0xef46db3751d8e999
	got := XXHash64(nil, 0)
	if got != 0xef46db3751d8e999 {
		t.Fatalf("XXHash64(empty, 0) = 0x%016x, want 0xef46db3751d8e999", got)
	}
	// Deterministic
	h1 := XXHash64([]byte("hello"), 0)
	h2 := XXHash64([]byte("hello"), 0)
	if h1 != h2 {
		t.Fatal("XXHash64 should be deterministic")
	}
}

func TestXXHash64Hex(t *testing.T) {
	got := XXHash64Hex(nil, 0)
	if got != "ef46db3751d8e999" {
		t.Fatalf("XXHash64Hex(empty, 0) = %s, want ef46db3751d8e999", got)
	}
}

func TestXXHash64_Seeded(t *testing.T) {
	h1 := XXHash64([]byte("hello"), 0)
	h2 := XXHash64([]byte("hello"), 1)
	if h1 == h2 {
		t.Fatal("XXHash64 with different seeds should differ")
	}
}

func TestXXHash64_LongInput(t *testing.T) {
	// Input >= 32 bytes to test the main loop.
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i)
	}
	h := XXHash64(data, 0)
	if h == 0 {
		t.Fatal("XXHash64 of long input should not be 0")
	}
}

func TestXXHash32(t *testing.T) {
	// Known: xxHash32("", 0) = 0x02cc5d05
	got := XXHash32(nil, 0)
	if got != 0x02cc5d05 {
		t.Fatalf("XXHash32(empty, 0) = 0x%08x, want 0x02cc5d05", got)
	}
}

func TestXXHash32Hex(t *testing.T) {
	got := XXHash32Hex(nil, 0)
	if got != "02cc5d05" {
		t.Fatalf("XXHash32Hex(empty, 0) = %s, want 02cc5d05", got)
	}
}

func TestXXHash32_Deterministic(t *testing.T) {
	h1 := XXHash32([]byte("hello"), 0)
	h2 := XXHash32([]byte("hello"), 0)
	if h1 != h2 {
		t.Fatal("XXHash32 should be deterministic")
	}
}

func TestXXHash32_LongInput(t *testing.T) {
	data := make([]byte, 64)
	for i := range data {
		data[i] = byte(i)
	}
	h := XXHash32(data, 0)
	if h == 0 {
		t.Fatal("XXHash32 of long input should not be 0")
	}
}

// ──────────────────────────────────────────────
// File/stream hashing
// ──────────────────────────────────────────────

func TestMD5File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	got, err := MD5FileHex(path)
	if err != nil {
		t.Fatalf("MD5FileHex failed: %v", err)
	}
	if got != "5d41402abc4b2a76b9719d911017c592" {
		t.Fatalf("MD5FileHex = %s", got)
	}
}

func TestMD5File_NotExist(t *testing.T) {
	_, err := MD5File("nonexistent-file-12345")
	if err == nil {
		t.Fatal("MD5File with nonexistent file should error")
	}
}

func TestSHA256File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	got, err := SHA256FileHex(path)
	if err != nil {
		t.Fatalf("SHA256FileHex failed: %v", err)
	}
	if got != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("SHA256FileHex = %s", got)
	}
}

func TestMD5Reader(t *testing.T) {
	got, err := MD5Reader(bytes.NewReader([]byte("hello")))
	if err != nil {
		t.Fatalf("MD5Reader failed: %v", err)
	}
	if hex := bytesToHex(got); hex != "5d41402abc4b2a76b9719d911017c592" {
		t.Fatalf("MD5Reader = %s", hex)
	}
}

func TestSHA256Reader(t *testing.T) {
	got, err := SHA256Reader(bytes.NewReader([]byte("hello")))
	if err != nil {
		t.Fatalf("SHA256Reader failed: %v", err)
	}
	if len(got) != 32 {
		t.Fatalf("SHA256Reader length = %d, want 32", len(got))
	}
}

func bytesToHex(b []byte) string {
	const hexChars = "0123456789abcdef"
	result := make([]byte, len(b)*2)
	for i, v := range b {
		result[i*2] = hexChars[v>>4]
		result[i*2+1] = hexChars[v&0x0f]
	}
	return string(result)
}

// ──────────────────────────────────────────────
// Generic hash helper
// ──────────────────────────────────────────────

func TestHex(t *testing.T) {
	tests := []struct {
		ht   HashType
		want int // expected hex length
	}{
		{HashMD5, 32},
		{HashSHA1, 40},
		{HashSHA256, 64},
		{HashSHA512, 128},
	}
	for _, tt := range tests {
		t.Run(string(tt.ht), func(t *testing.T) {
			got, err := Hex(tt.ht, []byte("hello"))
			if err != nil {
				t.Fatalf("Hex(%s) failed: %v", tt.ht, err)
			}
			if len(got) != tt.want {
				t.Fatalf("Hex(%s) length = %d, want %d", tt.ht, len(got), tt.want)
			}
		})
	}
}

func TestHex_UnknownType(t *testing.T) {
	_, err := Hex(HashType("unknown"), []byte("hello"))
	if err == nil {
		t.Fatal("Hex with unknown type should error")
	}
}

func TestHexUpper(t *testing.T) {
	got, err := HexUpper(HashMD5, []byte("hello"))
	if err != nil {
		t.Fatalf("HexUpper failed: %v", err)
	}
	if got != "5D41402ABC4B2A76B9719D911017C592" {
		t.Fatalf("HexUpper(MD5, hello) = %s", got)
	}
}
