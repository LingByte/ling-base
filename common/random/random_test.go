// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package random

import (
	"strings"
	"testing"
	"time"
)

// ──────────────────────────────────────────────
// Secure random numbers
// ──────────────────────────────────────────────

func TestInt(t *testing.T) {
	for i := 0; i < 100; i++ {
		n := Int()
		if n < 0 {
			t.Fatalf("Int() returned negative: %d", n)
		}
	}
}

func TestIntn(t *testing.T) {
	for i := 0; i < 1000; i++ {
		n := Intn(100)
		if n < 0 || n >= 100 {
			t.Fatalf("Intn(100) out of range: %d", n)
		}
	}
	// Check that all values in a small range are eventually hit.
	seen := make(map[int]bool)
	for i := 0; i < 1000; i++ {
		seen[Intn(5)] = true
	}
	if len(seen) != 5 {
		t.Fatalf("Intn(5) didn't cover all values: %v", seen)
	}
}

func TestIntn_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Intn(0) should panic")
		}
	}()
	Intn(0)
}

func TestIntRange(t *testing.T) {
	for i := 0; i < 1000; i++ {
		n := IntRange(10, 20)
		if n < 10 || n > 20 {
			t.Fatalf("IntRange(10,20) out of range: %d", n)
		}
	}
	// Single value range.
	if v := IntRange(5, 5); v != 5 {
		t.Fatalf("IntRange(5,5) = %d, want 5", v)
	}
}

func TestIntRange_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("IntRange should panic when min > max")
		}
	}()
	IntRange(20, 10)
}

func TestInt64(t *testing.T) {
	for i := 0; i < 100; i++ {
		n := Int64()
		if n < 0 {
			t.Fatalf("Int64() returned negative: %d", n)
		}
	}
}

func TestInt64n(t *testing.T) {
	for i := 0; i < 1000; i++ {
		n := Int64n(1000)
		if n < 0 || n >= 1000 {
			t.Fatalf("Int64n(1000) out of range: %d", n)
		}
	}
}

func TestInt64n_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Int64n(0) should panic")
		}
	}()
	Int64n(0)
}

func TestInt64Range(t *testing.T) {
	for i := 0; i < 1000; i++ {
		n := Int64Range(-100, 100)
		if n < -100 || n > 100 {
			t.Fatalf("Int64Range(-100,100) out of range: %d", n)
		}
	}
}

func TestFloat64(t *testing.T) {
	for i := 0; i < 1000; i++ {
		f := Float64()
		if f < 0.0 || f >= 1.0 {
			t.Fatalf("Float64() out of range: %f", f)
		}
	}
}

func TestFloat64Range(t *testing.T) {
	for i := 0; i < 1000; i++ {
		f := Float64Range(10.0, 20.0)
		if f < 10.0 || f >= 20.0 {
			t.Fatalf("Float64Range(10,20) out of range: %f", f)
		}
	}
}

func TestFloat64Range_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Float64Range should panic when min >= max")
		}
	}()
	Float64Range(20, 10)
}

func TestBool(t *testing.T) {
	trueCount, falseCount := 0, 0
	for i := 0; i < 100; i++ {
		if Bool() {
			trueCount++
		} else {
			falseCount++
		}
	}
	if trueCount == 0 || falseCount == 0 {
		t.Fatalf("Bool() not producing both values: true=%d false=%d", trueCount, falseCount)
	}
}

func TestBytes(t *testing.T) {
	b := Bytes(32)
	if len(b) != 32 {
		t.Fatalf("Bytes(32) length = %d, want 32", len(b))
	}
	// Check that two calls produce different results.
	b2 := Bytes(32)
	if string(b) == string(b2) {
		t.Fatal("Bytes(32) produced identical results twice")
	}
}

func TestBytes_Zero(t *testing.T) {
	b := Bytes(0)
	if len(b) != 0 {
		t.Fatalf("Bytes(0) length = %d, want 0", len(b))
	}
}

func TestBytes_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Bytes(-1) should panic")
		}
	}()
	Bytes(-1)
}

func TestRead(t *testing.T) {
	b := make([]byte, 16)
	n, err := Read(b)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != 16 {
		t.Fatalf("Read returned n=%d, want 16", n)
	}
}

// ──────────────────────────────────────────────
// Random strings
// ──────────────────────────────────────────────

func TestString(t *testing.T) {
	s := String(32)
	if len(s) != 32 {
		t.Fatalf("String(32) length = %d", len(s))
	}
	// All chars should be from CharsetAlphaNum.
	for _, c := range s {
		if !strings.ContainsRune(CharsetAlphaNum, c) {
			t.Fatalf("String(32) contains invalid char %q", c)
		}
	}
}

func TestString_Zero(t *testing.T) {
	if s := String(0); s != "" {
		t.Fatalf("String(0) = %q, want empty", s)
	}
}

func TestStringWithCharset(t *testing.T) {
	s := StringWithCharset(20, CharsetNumeric)
	if len(s) != 20 {
		t.Fatalf("length = %d, want 20", len(s))
	}
	for _, c := range s {
		if !strings.ContainsRune(CharsetNumeric, c) {
			t.Fatalf("StringWithCharset numeric contains %q", c)
		}
	}
}

func TestStringWithCharset_PanicEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("StringWithCharset with empty charset should panic")
		}
	}()
	StringWithCharset(10, "")
}

func TestStringWithCharset_PanicNegative(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("StringWithCharset with n<0 should panic")
		}
	}()
	StringWithCharset(-1, CharsetAlpha)
}

func TestLowerString(t *testing.T) {
	s := LowerString(20)
	for _, c := range s {
		if c < 'a' || c > 'z' {
			t.Fatalf("LowerString contains non-lowercase %q", c)
		}
	}
}

func TestUpperString(t *testing.T) {
	s := UpperString(20)
	for _, c := range s {
		if c < 'A' || c > 'Z' {
			t.Fatalf("UpperString contains non-uppercase %q", c)
		}
	}
}

func TestNumericString(t *testing.T) {
	s := NumericString(10)
	for _, c := range s {
		if c < '0' || c > '9' {
			t.Fatalf("NumericString contains non-digit %q", c)
		}
	}
}

func TestHexString(t *testing.T) {
	s := HexString(16)
	for _, c := range s {
		if !strings.ContainsRune(CharsetHexLower, c) {
			t.Fatalf("HexString contains non-hex %q", c)
		}
	}
}

func TestBase64URLString(t *testing.T) {
	s := Base64URLString(20)
	for _, c := range s {
		if !strings.ContainsRune(CharsetBase64URL, c) {
			t.Fatalf("Base64URLString contains invalid char %q", c)
		}
	}
}

func TestPassword(t *testing.T) {
	p := Password(16)
	if len(p) != 16 {
		t.Fatalf("Password(16) length = %d", len(p))
	}
	hasLower, hasUpper, hasDigit, hasSymbol := false, false, false, false
	for _, c := range p {
		switch {
		case strings.ContainsRune(CharsetAlphaLower, c):
			hasLower = true
		case strings.ContainsRune(CharsetAlphaUpper, c):
			hasUpper = true
		case strings.ContainsRune(CharsetNumeric, c):
			hasDigit = true
		case strings.ContainsRune(CharsetSymbol, c):
			hasSymbol = true
		}
	}
	if !hasLower || !hasUpper || !hasDigit || !hasSymbol {
		t.Fatalf("Password missing required classes: lower=%v upper=%v digit=%v symbol=%v",
			hasLower, hasUpper, hasDigit, hasSymbol)
	}
}

func TestPassword_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Password(3) should panic")
		}
	}()
	Password(3)
}

// ──────────────────────────────────────────────
// Sampling & shuffling
// ──────────────────────────────────────────────

func TestShuffleBytes(t *testing.T) {
	original := []byte("abcdefghij")
	shuffled := make([]byte, len(original))
	copy(shuffled, original)
	ShuffleBytes(shuffled)
	if len(shuffled) != len(original) {
		t.Fatalf("length changed after shuffle")
	}
	// Check all original elements are still present.
	seen := make(map[byte]bool)
	for _, b := range shuffled {
		seen[b] = true
	}
	for _, b := range original {
		if !seen[b] {
			t.Fatalf("byte %q lost after shuffle", b)
		}
	}
}

func TestShuffleInts(t *testing.T) {
	s := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	ShuffleInts(s)
	if len(s) != 10 {
		t.Fatal("length changed")
	}
	seen := make(map[int]bool)
	for _, v := range s {
		seen[v] = true
	}
	for i := 1; i <= 10; i++ {
		if !seen[i] {
			t.Fatalf("value %d lost after shuffle", i)
		}
	}
}

func TestShuffle(t *testing.T) {
	s := []int{1, 2, 3, 4, 5}
	Shuffle(len(s), func(i, j int) {
		s[i], s[j] = s[j], s[i]
	})
	if len(s) != 5 {
		t.Fatal("length changed")
	}
}

func TestChoice(t *testing.T) {
	s := []string{"a", "b", "c", "d"}
	for i := 0; i < 100; i++ {
		c := Choice(s)
		found := false
		for _, v := range s {
			if v == c {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Choice returned %q not in slice", c)
		}
	}
}

func TestChoice_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Choice on empty slice should panic")
		}
	}()
	Choice[int](nil)
}

func TestChoices(t *testing.T) {
	s := []int{10, 20, 30}
	result := Choices(s, 5)
	if len(result) != 5 {
		t.Fatalf("Choices length = %d, want 5", len(result))
	}
	for _, v := range result {
		if v != 10 && v != 20 && v != 30 {
			t.Fatalf("Choices returned invalid value %d", v)
		}
	}
}

func TestChoices_PanicEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Choices on empty slice should panic")
		}
	}()
	Choices[int](nil, 1)
}

func TestChoices_PanicNegative(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Choices with k<0 should panic")
		}
	}()
	Choices([]int{1}, -1)
}

func TestSample(t *testing.T) {
	s := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	result := Sample(s, 5)
	if len(result) != 5 {
		t.Fatalf("Sample length = %d, want 5", len(result))
	}
	// Check uniqueness.
	seen := make(map[int]bool)
	for _, v := range result {
		if seen[v] {
			t.Fatalf("Sample returned duplicate %d", v)
		}
		seen[v] = true
	}
	// Check all values are from the original slice.
	for _, v := range result {
		found := false
		for _, orig := range s {
			if v == orig {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Sample returned %d not in original", v)
		}
	}
}

func TestSample_Zero(t *testing.T) {
	result := Sample([]int{1, 2, 3}, 0)
	if len(result) != 0 {
		t.Fatalf("Sample(_, 0) length = %d, want 0", len(result))
	}
}

func TestSample_All(t *testing.T) {
	s := []int{1, 2, 3}
	result := Sample(s, 3)
	if len(result) != 3 {
		t.Fatalf("Sample length = %d, want 3", len(result))
	}
}

func TestSample_PanicTooMany(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Sample with k > len should panic")
		}
	}()
	Sample([]int{1, 2}, 5)
}

func TestSample_PanicNegative(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Sample with k < 0 should panic")
		}
	}()
	Sample([]int{1}, -1)
}

func TestPermutation(t *testing.T) {
	p := Permutation(10)
	if len(p) != 10 {
		t.Fatalf("Permutation(10) length = %d", len(p))
	}
	seen := make(map[int]bool)
	for _, v := range p {
		if v < 0 || v >= 10 {
			t.Fatalf("Permutation value %d out of range", v)
		}
		if seen[v] {
			t.Fatalf("Permutation has duplicate %d", v)
		}
		seen[v] = true
	}
}

func TestPermutation_Zero(t *testing.T) {
	p := Permutation(0)
	if len(p) != 0 {
		t.Fatalf("Permutation(0) length = %d", len(p))
	}
}

func TestPermutation_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Permutation(-1) should panic")
		}
	}()
	Permutation(-1)
}

// ──────────────────────────────────────────────
// Random colors
// ──────────────────────────────────────────────

func TestRGBColor(t *testing.T) {
	c := RGBColor()
	// Values are uint8, always 0-255. Just check it runs.
	_ = c
}

func TestHexColor(t *testing.T) {
	h := HexColor()
	if len(h) != 7 || h[0] != '#' {
		t.Fatalf("HexColor() = %q, want format #rrggbb", h)
	}
}

func TestRGB_Hex(t *testing.T) {
	c := RGB{R: 255, G: 0, B: 128}
	if c.Hex() != "#ff0080" {
		t.Fatalf("RGB.Hex() = %q, want #ff0080", c.Hex())
	}
}

func TestHSLColor(t *testing.T) {
	c := HSLColor()
	if c.H < 0 || c.H >= 360 {
		t.Fatalf("HSL H out of range: %f", c.H)
	}
	if c.S < 0.5 || c.S > 0.9 {
		t.Fatalf("HSL S out of range: %f", c.S)
	}
	if c.L < 0.4 || c.L > 0.8 {
		t.Fatalf("HSL L out of range: %f", c.L)
	}
}

func TestHSLToRGB(t *testing.T) {
	// Gray (S=0).
	gray := HSLToRGB(HSL{H: 0, S: 0, L: 0.5})
	if gray.R != 127 || gray.G != 127 || gray.B != 127 {
		t.Fatalf("HSLToRGB gray = %v, want ~127,127,127", gray)
	}
	// Red (H=0, S=1, L=0.5).
	red := HSLToRGB(HSL{H: 0, S: 1, L: 0.5})
	if red.R != 255 {
		t.Fatalf("HSLToRGB red R = %d, want 255", red.R)
	}
	// Just check it produces valid values for a random HSL.
	rgb := HSLToRGB(HSLColor())
	_ = rgb
}

// ──────────────────────────────────────────────
// UUID
// ──────────────────────────────────────────────

func TestUUID(t *testing.T) {
	u := UUID()
	if len(u) != 36 {
		t.Fatalf("UUID() length = %d, want 36", len(u))
	}
	if u[8] != '-' || u[13] != '-' || u[18] != '-' || u[23] != '-' {
		t.Fatalf("UUID() format invalid: %q", u)
	}
	// Version 4.
	if u[14] != '4' {
		t.Fatalf("UUID() version = %c, want 4", u[14])
	}
	// Variant (8, 9, a, or b).
	v := u[19]
	if v != '8' && v != '9' && v != 'a' && v != 'b' {
		t.Fatalf("UUID() variant = %c, want 8/9/a/b", v)
	}
}

func TestUUID_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		u := UUID()
		if seen[u] {
			t.Fatalf("UUID collision: %s", u)
		}
		seen[u] = true
	}
}

func TestUUIDBytes(t *testing.T) {
	b := UUIDBytes()
	if b[6]&0xf0 != 0x40 {
		t.Fatalf("UUIDBytes version bits wrong: %02x", b[6])
	}
	if b[8]&0xc0 != 0x80 {
		t.Fatalf("UUIDBytes variant bits wrong: %02x", b[8])
	}
}

func TestUUIDNoDashes(t *testing.T) {
	u := UUIDNoDashes()
	if len(u) != 32 {
		t.Fatalf("UUIDNoDashes() length = %d, want 32", len(u))
	}
	if strings.Contains(u, "-") {
		t.Fatalf("UUIDNoDashes() contains dashes: %q", u)
	}
}

// ──────────────────────────────────────────────
// Math convenience
// ──────────────────────────────────────────────

func TestMathShuffle(t *testing.T) {
	s := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	MathShuffle(s)
	if len(s) != 10 {
		t.Fatal("length changed")
	}
}

func TestMathIntn(t *testing.T) {
	for i := 0; i < 100; i++ {
		n := MathIntn(50)
		if n < 0 || n >= 50 {
			t.Fatalf("MathIntn(50) = %d, out of range", n)
		}
	}
}

// ──────────────────────────────────────────────
// Concurrency safety (race detector)
// ──────────────────────────────────────────────

func TestConcurrentSafety(t *testing.T) {
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = Intn(100)
				_ = String(16)
				_ = UUID()
				_ = HexColor()
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

// ──────────────────────────────────────────────
// Distribution sanity check
// ──────────────────────────────────────────────

func TestIntn_Distribution(t *testing.T) {
	const n = 10
	const samples = 10000
	counts := make([]int, n)
	for i := 0; i < samples; i++ {
		counts[Intn(n)]++
	}
	expected := samples / n
	for i, c := range counts {
		// Allow 50% deviation from expected.
		if c < expected/2 || c > expected*2 {
			t.Fatalf("Intn(%d) distribution skewed for %d: count=%d expected~%d", n, i, c, expected)
		}
	}
}

func TestString_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		s := String(16)
		if seen[s] {
			t.Fatalf("String(16) collision: %s", s)
		}
		seen[s] = true
	}
}

// Ensure time import is used (for potential future timing tests).
var _ = time.Now
