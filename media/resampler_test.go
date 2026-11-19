package media

import "testing"

func TestStreamResampler_SameRateAliases(t *testing.T) {
	rs := NewStreamResampler(16000, 16000)
	in := []byte{1, 2, 3, 4}
	out, err := rs.Resample(in)
	if err != nil {
		t.Fatal(err)
	}
	if &out[0] != &in[0] {
		t.Fatal("same-rate should alias input")
	}
}

func TestStreamResampler_8kTo16k(t *testing.T) {
	rs := NewStreamResampler(8000, 16000)
	// 20ms @ 8kHz = 160 samples = 320 bytes
	in := make([]byte, 320)
	out, err := rs.Resample(in)
	if err != nil {
		t.Fatal(err)
	}
	// ~40ms worth at 16k ≈ 640 bytes (interpolator float rounding)
	if len(out) < 600 || len(out) > 680 {
		t.Fatalf("unexpected out len %d", len(out))
	}
	// Second call should still work (persistent converter).
	out2, err := rs.Resample(in)
	if err != nil || len(out2) < 600 {
		t.Fatalf("second resample: err=%v len=%d", err, len(out2))
	}
}

func TestResamplePCM_WriteNoDoubleCopy(t *testing.T) {
	// Smoke: one-shot still works after Write optimization.
	in := make([]byte, 320)
	out, err := ResamplePCM(in, 8000, 16000)
	if err != nil || len(out) < 600 {
		t.Fatalf("err=%v len=%d", err, len(out))
	}
}

func BenchmarkResamplePCM_PerFrame(b *testing.B) {
	in := make([]byte, 320) // 20ms @ 8k
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := ResamplePCM(in, 8000, 16000)
		if err != nil || len(out) == 0 {
			b.Fatal(err)
		}
	}
}

func BenchmarkStreamResampler_Reuse(b *testing.B) {
	rs := NewStreamResampler(8000, 16000)
	in := make([]byte, 320)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := rs.Resample(in)
		if err != nil || len(out) == 0 {
			b.Fatal(err)
		}
	}
}
