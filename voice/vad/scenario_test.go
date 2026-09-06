// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package vad

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

// ─── 真实场景模拟测试 ───

// generateSpeechLikeFrame 生成模拟语音帧：低频成分 + 幅度调制（模拟音节）
func generateSpeechLikeFrame(ms, sampleRate int, seed int64) []byte {
	samples := sampleRate * ms / 1000
	buf := make([]byte, samples*2)
	r := rand.New(rand.NewSource(seed))
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		// 基频 150Hz + 谐波 + 幅度调制（模拟音节节奏）
		fundamental := math.Sin(2 * math.Pi * 150 * t)
		harmonic2 := 0.3 * math.Sin(2*math.Pi*300*t)
		harmonic3 := 0.15 * math.Sin(2*math.Pi*450*t)
		// 5Hz 幅度调制模拟音节
		am := 0.7 + 0.3*math.Sin(2*math.Pi*5*t)
		// 少量噪声
		noise := r.Float64()*0.1 - 0.05
		v := (fundamental + harmonic2 + harmonic3) * am * 0.6
		v += noise
		sample := int16(v * 30000)
		if sample > 32767 {
			sample = 32767
		}
		if sample < -32768 {
			sample = -32768
		}
		buf[i*2] = byte(sample & 0xff)
		buf[i*2+1] = byte(sample >> 8)
	}
	return buf
}

// generateNoiseFrame 生成白噪声帧
func generateNoiseFrame(ms, sampleRate int, seed int64, amp float64) []byte {
	samples := sampleRate * ms / 1000
	buf := make([]byte, samples*2)
	r := rand.New(rand.NewSource(seed))
	for i := 0; i < samples; i++ {
		v := r.Float64()*2 - 1 // [-1, 1]
		sample := int16(v * amp * 32767)
		buf[i*2] = byte(sample & 0xff)
		buf[i*2+1] = byte(sample >> 8)
	}
	return buf
}

// generateToneFrame 生成纯音调帧
func generateToneFrame(ms, sampleRate int, freq float64, amp int16) []byte {
	samples := sampleRate * ms / 1000
	buf := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		v := int16(float64(amp) * math.Sin(2*math.Pi*freq*t))
		buf[i*2] = byte(v & 0xff)
		buf[i*2+1] = byte(v >> 8)
	}
	return buf
}

// ─── 场景1: 连续语音流（模拟 2 秒语音 + 1 秒静音 + 2 秒语音）───

func TestScenario_ContinuousSpeechStream(t *testing.T) {
	det, err := NewSileroDetector(0.5)
	if err != nil {
		t.Fatalf("failed to create Silero: %v", err)
	}
	defer det.Close()

	stream := NewStreamer(det, Config{
		SampleRate:      16000,
		FrameDurationMs: 32, // Silero = 32ms
		MinSpeechFrames: 3,
		HangoverFrames:  15,
	})
	defer stream.Close()

	// 生成 5 秒音频：2s 语音 + 1s 静音 + 2s 语音
	frameMs := 32
	framesPerSec := 1000 / frameMs

	var speechFrames, silenceFrames int
	var events []Event

	// 2 秒语音
	for i := 0; i < 2*framesPerSec; i++ {
		pcm := generateSpeechLikeFrame(frameMs, 16000, int64(i))
		result, _ := stream.ProcessFrame(pcm)
		if result.IsSpeech {
			speechFrames++
		}
	}

	// 1 秒静音
	for i := 0; i < framesPerSec; i++ {
		pcm := silentFrame(frameMs, 16000)
		result, _ := stream.ProcessFrame(pcm)
		if !result.IsSpeech {
			silenceFrames++
		}
	}

	// 2 秒语音
	for i := 0; i < 2*framesPerSec; i++ {
		pcm := generateSpeechLikeFrame(frameMs, 16000, int64(i+1000))
		result, _ := stream.ProcessFrame(pcm)
		if result.IsSpeech {
			speechFrames++
		}
	}

	// 收集事件
	for {
		select {
		case ev := <-stream.Events():
			events = append(events, ev)
		default:
			goto done
		}
	}
done:

	t.Logf("Speech frames: %d, Silence frames: %d", speechFrames, silenceFrames)
	t.Logf("Events: %d", len(events))
	for _, ev := range events {
		t.Logf("  %s at frame %d (prob=%.3f)", ev.String(), ev.FrameIndex, ev.Probability)
	}

	// 验证：应该检测到语音
	if speechFrames < 50 {
		t.Errorf("expected significant speech frames, got %d", speechFrames)
	}

	// 验证：应该有至少 2 个事件（SpeechStart + SpeechEnd）
	if len(events) < 2 {
		t.Errorf("expected at least 2 events, got %d", len(events))
	}

	// 验证：第一个事件应该是 SpeechStart
	if len(events) > 0 && events[0].Type != EventSpeechStart {
		t.Errorf("expected first event SpeechStart, got %v", events[0].Type)
	}
}

// ─── 场景2: 噪声环境（白噪声不应被误判为语音）───

func TestScenario_NoiseRejection(t *testing.T) {
	tests := []struct {
		name    string
		engine  EngineKind
		amp     float64
	}{
		{"Silero_low_noise", EngineSilero, 0.05},
		{"Silero_med_noise", EngineSilero, 0.15},
		{"Energy_low_noise", EngineEnergy, 0.05},
		{"Energy_med_noise", EngineEnergy, 0.15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.MinSpeechFrames = 3
			cfg.HangoverFrames = 10

			stream, err := NewStreamerWithConfig(tt.engine, cfg)
			if err != nil {
				t.Fatalf("failed to create streamer: %v", err)
			}
			defer stream.Close()

			// 3 秒白噪声
			frameMs := cfg.FrameDurationMs
			if tt.engine == EngineSilero {
				frameMs = 32
			}
			framesPerSec := 1000 / frameMs

			speechCount := 0
			for i := 0; i < 3*framesPerSec; i++ {
				pcm := generateNoiseFrame(frameMs, cfg.SampleRate, int64(i), tt.amp)
				result, _ := stream.ProcessFrame(pcm)
				if result.IsSpeech {
					speechCount++
				}
			}

			totalFrames := 3 * framesPerSec
			speechRatio := float64(speechCount) / float64(totalFrames)
			t.Logf("Engine=%s, noise_amp=%.2f: speech_ratio=%.2f (%d/%d)",
				tt.engine, tt.amp, speechRatio, speechCount, totalFrames)

			// 噪声不应被大量误判为语音
			// Silero 应该几乎不误判；Energy 在高噪声时可能有一些
			maxRatio := 0.1 // 10% 上限
			if tt.engine == EngineEnergy && tt.amp >= 0.15 {
				maxRatio = 0.3 // Energy 在中噪声时容忍度高一些
			}
			if speechRatio > maxRatio {
				t.Errorf("noise rejection failed: speech_ratio=%.2f > %.2f", speechRatio, maxRatio)
			}
		})
	}
}

// ─── 场景3: 纯音调不应被 Silero 误判 ───

func TestScenario_ToneRejection(t *testing.T) {
	det, err := NewSileroDetector(0.5)
	if err != nil {
		t.Fatalf("failed to create Silero: %v", err)
	}
	defer det.Close()

	stream := NewStreamer(det, Config{
		SampleRate:      16000,
		FrameDurationMs: 32,
		MinSpeechFrames: 3,
		HangoverFrames:  10,
	})
	defer stream.Close()

	// 2 秒 440Hz 纯音调
	framesPerSec := 1000 / 32
	speechCount := 0
	for i := 0; i < 2*framesPerSec; i++ {
		pcm := generateToneFrame(32, 16000, 440, 30000)
		result, _ := stream.ProcessFrame(pcm)
		if result.IsSpeech {
			speechCount++
		}
	}

	totalFrames := 2 * framesPerSec
	speechRatio := float64(speechCount) / float64(totalFrames)
	t.Logf("Silero tone rejection: speech_ratio=%.2f (%d/%d)", speechRatio, speechCount, totalFrames)

	// Silero 应该能拒绝大部分纯音调
	if speechRatio > 0.3 {
		t.Errorf("Silero should reject most pure tones: speech_ratio=%.2f", speechRatio)
	}
}

// ─── 场景4: 延迟测试 ───

func TestScenario_Latency(t *testing.T) {
	tests := []struct {
		name   string
		engine EngineKind
	}{
		{"Energy", EngineEnergy},
		{"WebRTC", EngineWebRTC},
		{"Silero", EngineSilero},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			if tt.engine == EngineSilero {
				cfg.FrameDurationMs = 32
			}

			det, err := NewDetector(tt.engine, cfg)
			if err != nil {
				t.Fatalf("failed to create detector: %v", err)
			}
			defer det.Close()

			frameMs := cfg.FrameDurationMs
			if tt.engine == EngineSilero {
				frameMs = 32
			}
			frameBytes := cfg.SampleRate * frameMs / 1000 * 2

			// 预热
			pcm := generateSpeechLikeFrame(frameMs, cfg.SampleRate, 1)
			for i := 0; i < 10; i++ {
				det.ProcessFrame(pcm)
			}

			// 测 100 帧平均延迟
			iterations := 100
			start := time.Now()
			for i := 0; i < iterations; i++ {
				det.ProcessFrame(pcm)
			}
			elapsed := time.Since(start)
			avgLatency := elapsed / time.Duration(iterations)

			t.Logf("Engine=%s: avg latency=%v per frame (%v for %d frames), frame_bytes=%d",
				tt.name, avgLatency, elapsed, iterations, frameBytes)

			// 延迟应该在合理范围内
			maxLatency := time.Duration(50 * time.Millisecond) // 50ms 上限
			if tt.engine == EngineSilero {
				maxLatency = 100 * time.Millisecond // Silero 神经网络推理更重
			}
			if avgLatency > maxLatency {
				t.Errorf("latency too high: %v > %v", avgLatency, maxLatency)
			}
		})
	}
}

// ─── 场景5: 短语音段（模拟 "嗯" 等短音节）───

func TestScenario_ShortUtterance(t *testing.T) {
	det, err := NewSileroDetector(0.5)
	if err != nil {
		t.Fatalf("failed to create Silero: %v", err)
	}
	defer det.Close()

	// 短语音段：300ms 语音 + 500ms 静音
	cfg := Config{
		SampleRate:      16000,
		FrameDurationMs: 32,
		MinSpeechFrames: 2,  // 降低门槛以检测短语音
		HangoverFrames:  10,
	}
	stream := NewStreamer(det, cfg)
	defer stream.Close()

	// 300ms 语音 (~9 frames)
	for i := 0; i < 9; i++ {
		pcm := generateSpeechLikeFrame(32, 16000, int64(i))
		stream.ProcessFrame(pcm)
	}

	// 500ms 静音 (~15 frames)
	for i := 0; i < 15; i++ {
		pcm := silentFrame(32, 16000)
		stream.ProcessFrame(pcm)
	}

	events := drainEvents(stream)
	t.Logf("Short utterance events: %d", len(events))
	for _, ev := range events {
		t.Logf("  %s at frame %d", ev.String(), ev.FrameIndex)
	}

	// 短语音应该能被检测到（至少 SpeechStart）
	if len(events) < 1 {
		t.Error("expected at least SpeechStart for short utterance")
	}
}

// ─── 场景6: 多轮对话模拟 ───

func TestScenario_MultiTurnConversation(t *testing.T) {
	det, err := NewSileroDetector(0.5)
	if err != nil {
		t.Fatalf("failed to create Silero: %v", err)
	}
	defer det.Close()

	cfg := Config{
		SampleRate:      16000,
		FrameDurationMs: 32,
		MinSpeechFrames: 3,
		HangoverFrames:  15,
	}
	stream := NewStreamer(det, cfg)
	defer stream.Close()

	// 模拟 3 轮对话：每轮 1.5s 语音 + 1s 静音
	turns := 3
	speechDuration := 1500 // ms
	silenceDuration := 1000 // ms
	frameMs := 32
	speechFrames := speechDuration / frameMs
	silenceFrames := silenceDuration / frameMs

	for turn := 0; turn < turns; turn++ {
		// 语音
		for i := 0; i < speechFrames; i++ {
			pcm := generateSpeechLikeFrame(frameMs, 16000, int64(turn*1000+i))
			stream.ProcessFrame(pcm)
		}
		// 静音
		for i := 0; i < silenceFrames; i++ {
			pcm := silentFrame(frameMs, 16000)
			stream.ProcessFrame(pcm)
		}
	}

	events := drainEvents(stream)
	t.Logf("Multi-turn events: %d", len(events))
	for _, ev := range events {
		t.Logf("  %s at frame %d", ev.String(), ev.FrameIndex)
	}

	// 应该有至少 6 个事件（3 SpeechStart + 3 SpeechEnd）
	if len(events) < 6 {
		t.Errorf("expected at least 6 events (3 turns × 2), got %d", len(events))
	}

	// 验证事件交替模式
	startCount := 0
	endCount := 0
	for _, ev := range events {
		if ev.Type == EventSpeechStart {
			startCount++
		} else if ev.Type == EventSpeechEnd {
			endCount++
		}
	}
	t.Logf("Starts: %d, Ends: %d", startCount, endCount)

	if startCount < 3 {
		t.Errorf("expected at least 3 SpeechStart, got %d", startCount)
	}
}

// ─── 场景7: Reset 后干净重启 ───

func TestScenario_ResetBetweenStreams(t *testing.T) {
	det, err := NewSileroDetector(0.5)
	if err != nil {
		t.Fatalf("failed to create Silero: %v", err)
	}
	defer det.Close()

	cfg := Config{
		SampleRate:      16000,
		FrameDurationMs: 32,
		MinSpeechFrames: 2,
		HangoverFrames:  10,
	}
	stream := NewStreamer(det, cfg)
	defer stream.Close()

	// 第一段语音
	for i := 0; i < 20; i++ {
		pcm := generateSpeechLikeFrame(32, 16000, int64(i))
		stream.ProcessFrame(pcm)
	}
	events1 := drainEvents(stream)
	t.Logf("Stream 1 events: %d", len(events1))

	// Reset
	stream.Reset()

	// 第二段语音
	for i := 0; i < 20; i++ {
		pcm := generateSpeechLikeFrame(32, 16000, int64(i+500))
		stream.ProcessFrame(pcm)
	}
	events2 := drainEvents(stream)
	t.Logf("Stream 2 events after reset: %d", len(events2))

	// Reset 后应该能正常检测
	if stream.FrameIndex() != 20 {
		t.Errorf("expected frame index 20 after reset, got %d", stream.FrameIndex())
	}
}

// ─── 场景8: 边界情况 — 极短帧、空帧、奇数字节 ───

func TestScenario_EdgeCases(t *testing.T) {
	det := NewEnergyDetectorWithConfig(EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       500,
	})
	defer det.Close()

	// 空帧
	result, err := det.ProcessFrame(nil)
	if err != nil {
		t.Errorf("nil frame should not error: %v", err)
	}
	if result.IsSpeech {
		t.Error("nil frame should not be speech")
	}

	// 1 字节（奇数，不完整样本）
	result, err = det.ProcessFrame([]byte{0x00})
	if err != nil {
		t.Errorf("odd byte frame should not error: %v", err)
	}
	if result.IsSpeech {
		t.Error("odd byte frame should not be speech")
	}

	// 2 字节（1 个样本）
	result, err = det.ProcessFrame([]byte{0xff, 0x7f})
	if err != nil {
		t.Errorf("2-byte frame should not error: %v", err)
	}
	// 应该不崩溃，结果合理
	_ = result

	// 超大帧
	bigFrame := make([]byte, 16000*2) // 1 秒
	for i := 0; i < len(bigFrame); i += 2 {
		bigFrame[i] = 0xff
		bigFrame[i+1] = 0x7f
	}
	result, err = det.ProcessFrame(bigFrame)
	if err != nil {
		t.Errorf("big frame should not error: %v", err)
	}
	if !result.IsSpeech {
		t.Error("big loud frame should be speech")
	}
}

// ─── 场景9: 并发安全（Streamer 多 goroutine 读写）───

func TestScenario_ConcurrentAccess(t *testing.T) {
	det := NewEnergyDetectorWithConfig(EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       500,
	})
	defer det.Close()

	stream := NewStreamer(det, Config{
		SampleRate:      16000,
		FrameDurationMs: 20,
		MinSpeechFrames: 2,
		HangoverFrames:  5,
	})
	defer stream.Close()

	done := make(chan struct{})

	// 写 goroutine
	go func() {
		for i := 0; i < 100; i++ {
			pcm := generateSpeechLikeFrame(20, 16000, int64(i))
			stream.ProcessFrame(pcm)
		}
		close(done)
	}()

	// 读 goroutine（轮询状态）
	for {
		select {
		case <-done:
			_ = stream.IsSpeech()
			_ = stream.FrameIndex()
			_ = stream.LastEvent()
			return
		default:
			_ = stream.IsSpeech()
			_ = stream.FrameIndex()
			runtime_Gosched()
		}
	}
}

// runtime_Gosched avoids importing runtime in test file top-level
func runtime_Gosched() {
	// small sleep to avoid tight loop
	time.Sleep(100 * time.Microsecond)
}

// ─── 场景10: 性能基准 ───

func BenchmarkSileroDetector_ProcessFrame(b *testing.B) {
	det, err := NewSileroDetector(0.5)
	if err != nil {
		b.Fatalf("failed to create Silero: %v", err)
	}
	defer det.Close()

	pcm := generateSpeechLikeFrame(32, 16000, 1)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		det.ProcessFrame(pcm)
	}
}

func BenchmarkEnergyDetector_ProcessFrame(b *testing.B) {
	det := NewEnergyDetectorWithConfig(EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       1500,
	})
	defer det.Close()

	pcm := generateSpeechLikeFrame(20, 16000, 1)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		det.ProcessFrame(pcm)
	}
}

func BenchmarkWebRTCDetector_ProcessFrame(b *testing.B) {
	det, err := NewWebRTCDetector(16000, 20, 2)
	if err != nil {
		b.Fatalf("failed to create WebRTC: %v", err)
	}
	defer det.Close()

	pcm := generateSpeechLikeFrame(20, 16000, 1)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		det.ProcessFrame(pcm)
	}
}

func BenchmarkStreamer_ProcessFrame(b *testing.B) {
	det := NewEnergyDetectorWithConfig(EnergyConfig{
		SampleRate:      16000,
		FrameDurationMs: 20,
		Threshold:       1500,
	})
	defer det.Close()

	stream := NewStreamer(det, Config{
		SampleRate:      16000,
		FrameDurationMs: 20,
		MinSpeechFrames: 3,
		HangoverFrames:  15,
	})
	defer stream.Close()

	pcm := generateSpeechLikeFrame(20, 16000, 1)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		stream.ProcessFrame(pcm)
	}
}

// Ensure fmt is used
var _ = fmt.Sprintf
