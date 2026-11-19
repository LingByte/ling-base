//go:build !lingcodec

package encoder

import "testing"

func TestG711_IntoRoundTrip_PCMA(t *testing.T) {
	pcm := []byte{0x00, 0x10, 0xFF, 0x7F, 0x00, 0x80, 0x10, 0x00} // 4 samples
	alaw, err := Pcm2pcma(pcm)
	if err != nil || len(alaw) != 4 {
		t.Fatalf("encode: %v len=%d", err, len(alaw))
	}
	dst := make([]byte, 8)
	pcma2pcmInto(dst, alaw)
	out, err := pcma2pcm(alaw)
	if err != nil {
		t.Fatal(err)
	}
	if string(dst) != string(out) {
		t.Fatalf("Into vs alloc mismatch: %v vs %v", dst, out)
	}
}

func TestG711_IntoRoundTrip_PCMU(t *testing.T) {
	pcm := []byte{0x00, 0x10, 0xFF, 0x7F, 0x00, 0x80, 0x10, 0x00}
	ulaw, err := pcm2pcmu(pcm)
	if err != nil || len(ulaw) != 4 {
		t.Fatalf("encode: %v len=%d", err, len(ulaw))
	}
	dst := make([]byte, 8)
	pcmu2pcmInto(dst, ulaw)
	out, err := pcmu2pcm(ulaw)
	if err != nil {
		t.Fatal(err)
	}
	if string(dst) != string(out) {
		t.Fatalf("Into vs alloc mismatch")
	}
}

func BenchmarkPCMA_DecodeAlloc(b *testing.B) {
	alaw := make([]byte, 160)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := pcma2pcm(alaw)
		if err != nil || len(out) != 320 {
			b.Fatal(err)
		}
	}
}

func BenchmarkPCMA_DecodeIntoReused(b *testing.B) {
	alaw := make([]byte, 160)
	dst := make([]byte, 320)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pcma2pcmInto(dst, alaw)
	}
}
