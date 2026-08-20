package synthesizer

import (
	"context"
	"testing"
	"time"
)

func TestPCMEmitConfigFromFormat(t *testing.T) {
	f := StreamFormat{
		SampleRate:    24000,
		BitDepth:      16,
		Channels:      1,
		FrameDuration: 20 * time.Millisecond,
	}
	cfg := PCMEmitConfigFromFormat(f)
	if cfg.SampleRate != 24000 {
		t.Errorf("SampleRate = %d, want 24000", cfg.SampleRate)
	}
	if cfg.BitDepth != 16 {
		t.Errorf("BitDepth = %d, want 16", cfg.BitDepth)
	}
	if cfg.Channels != 1 {
		t.Errorf("Channels = %d, want 1", cfg.Channels)
	}
	if cfg.FrameMS != 20 {
		t.Errorf("FrameMS = %d, want 20", cfg.FrameMS)
	}
}

func TestPCMEmitConfigFromFormatDefaults(t *testing.T) {
	f := StreamFormat{} // all zeros
	cfg := PCMEmitConfigFromFormat(f)
	if cfg.SampleRate != 16000 {
		t.Errorf("SampleRate = %d, want 16000 (default)", cfg.SampleRate)
	}
	if cfg.BitDepth != 16 {
		t.Errorf("BitDepth = %d, want 16 (default)", cfg.BitDepth)
	}
	if cfg.Channels != 1 {
		t.Errorf("Channels = %d, want 1 (default)", cfg.Channels)
	}
	if cfg.FrameMS != 20 {
		t.Errorf("FrameMS = %d, want 20 (default)", cfg.FrameMS)
	}
}

func TestFrameBytes(t *testing.T) {
	tests := []struct {
		name string
		cfg  PCMEmitConfig
		want int
	}{
		{"16k/16bit/mono/20ms", PCMEmitConfig{SampleRate: 16000, BitDepth: 16, Channels: 1, FrameMS: 20}, 640},
		{"24k/16bit/mono/20ms", PCMEmitConfig{SampleRate: 24000, BitDepth: 16, Channels: 1, FrameMS: 20}, 960},
		{"16k/16bit/stereo/20ms", PCMEmitConfig{SampleRate: 16000, BitDepth: 16, Channels: 2, FrameMS: 20}, 1280},
		{"16k/16bit/mono/50ms", PCMEmitConfig{SampleRate: 16000, BitDepth: 16, Channels: 1, FrameMS: 50}, 1600},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FrameBytes(tt.cfg)
			if got != tt.want {
				t.Errorf("FrameBytes() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFrameBytesDefaults(t *testing.T) {
	cfg := PCMEmitConfig{} // all zeros
	got := FrameBytes(cfg)
	// Should use defaults: 16000, 16, 1, 20ms → 640
	if got != 640 {
		t.Errorf("FrameBytes(empty) = %d, want 640 (defaults)", got)
	}
}

func TestEmitPCMChunks(t *testing.T) {
	cfg := PCMEmitConfig{SampleRate: 16000, BitDepth: 16, Channels: 1, FrameMS: 20}
	chunkSize := FrameBytes(cfg) // 640 bytes

	// Create 3 chunks worth of data
	pcm := make([]byte, chunkSize*3)

	var chunks [][]byte
	h := HandlerFunc{
		OnMessageFn: func(data []byte) {
			chunks = append(chunks, data)
		},
	}

	err := EmitPCMChunks(context.Background(), h, pcm, cfg)
	if err != nil {
		t.Fatalf("EmitPCMChunks failed: %v", err)
	}
	if len(chunks) != 3 {
		t.Errorf("chunks count = %d, want 3", len(chunks))
	}
	for i, c := range chunks {
		if len(c) != chunkSize {
			t.Errorf("chunk[%d] size = %d, want %d", i, len(c), chunkSize)
		}
	}
}

func TestEmitPCMChunksPartialLast(t *testing.T) {
	cfg := PCMEmitConfig{SampleRate: 16000, BitDepth: 16, Channels: 1, FrameMS: 20}
	chunkSize := FrameBytes(cfg) // 640 bytes

	// Create 2.5 chunks worth of data
	pcm := make([]byte, chunkSize*2+chunkSize/2)

	var chunks [][]byte
	h := HandlerFunc{
		OnMessageFn: func(data []byte) {
			chunks = append(chunks, data)
		},
	}

	err := EmitPCMChunks(context.Background(), h, pcm, cfg)
	if err != nil {
		t.Fatalf("EmitPCMChunks failed: %v", err)
	}
	if len(chunks) != 3 {
		t.Errorf("chunks count = %d, want 3", len(chunks))
	}
	// Last chunk should be partial
	if len(chunks[2]) != chunkSize/2 {
		t.Errorf("last chunk size = %d, want %d", len(chunks[2]), chunkSize/2)
	}
}

func TestEmitPCMChunksEmpty(t *testing.T) {
	var called bool
	h := HandlerFunc{
		OnMessageFn: func(data []byte) {
			called = true
		},
	}

	err := EmitPCMChunks(context.Background(), h, []byte{}, PCMEmitConfig{})
	if err != nil {
		t.Fatalf("EmitPCMChunks failed: %v", err)
	}
	if !called {
		t.Error("OnMessage should be called with nil for empty PCM")
	}
}

func TestEmitPCMChunksNilHandler(t *testing.T) {
	err := EmitPCMChunks(context.Background(), nil, []byte("test"), PCMEmitConfig{})
	if err != nil {
		t.Fatalf("EmitPCMChunks with nil handler should not error: %v", err)
	}
}

func TestEmitPCMChunksContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	cfg := PCMEmitConfig{SampleRate: 16000, BitDepth: 16, Channels: 1, FrameMS: 20}
	chunkSize := FrameBytes(cfg)
	pcm := make([]byte, chunkSize*10) // large enough to trigger multiple iterations

	var callCount int
	h := HandlerFunc{
		OnMessageFn: func(data []byte) {
			callCount++
		},
	}

	err := EmitPCMChunks(ctx, h, pcm, cfg)
	if err == nil {
		t.Error("EmitPCMChunks with canceled context should return error")
	}
	// The first chunk might be emitted before context is checked
	// but it should not complete all chunks
	if callCount >= 10 {
		t.Error("should not have emitted all chunks with canceled context")
	}
}
