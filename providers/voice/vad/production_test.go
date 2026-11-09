// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package vad

import (
	"math"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"
)

// ─── 生产环境压力测试 ───

// 生成 1 秒模拟语音（含自然停顿）
func genOneSecondSpeech(sr int) [][]byte {
	frameMs := 20
	frames := 1000 / frameMs
	out := make([][]byte, frames)
	r := rand.New(rand.NewSource(42))
	for i := 0; i < frames; i++ {
		// 模拟自然语音：80% 语音帧 + 20% 短停顿
		if i%5 == 4 {
			out[i] = make([]byte, sr*frameMs/1000*2) // 静音帧
		} else {
			samples := sr * frameMs / 1000
			buf := make([]byte, samples*2)
			for j := 0; j < samples; j++ {
				t := float64(j) / float64(sr)
				v := (math.Sin(2*math.Pi*150*t)+0.3*math.Sin(2*math.Pi*300*t)) *
					(0.7 + 0.3*math.Sin(2*math.Pi*5*t)) * 0.6
				v += r.Float64()*0.1 - 0.05
				s := int16(v * 30000)
				buf[j*2] = byte(s & 0xff)
				buf[j*2+1] = byte(s >> 8)
			}
			out[i] = buf
		}
	}
	return out
}

// ─── 测试1: 单路实时音频 CPU 占用 ───

func TestProduction_SingleStreamCPU(t *testing.T) {
	engines := []EngineKind{EngineEnergy, EngineWebRTC, EngineSilero}

	for _, engine := range engines {
		t.Run(string(engine), func(t *testing.T) {
			cfg := DefaultConfig()
			if engine == EngineSilero {
				cfg.FrameDurationMs = 32
			}

			det, err := NewDetector(engine, cfg)
			if err != nil {
				t.Fatalf("create detector: %v", err)
			}
			defer det.Close()

			frames := genOneSecondSpeech(cfg.SampleRate)
			frameMs := cfg.FrameDurationMs
			if engine == EngineSilero {
				frameMs = 32
			}

			// 处理 10 秒音频
			iterations := 10 * (1000 / frameMs)
			start := time.Now()
			for i := 0; i < iterations; i++ {
				det.ProcessFrame(frames[i%len(frames)])
			}
			elapsed := time.Since(start)
			audioDuration := time.Duration(iterations*frameMs) * time.Millisecond
			cpuPct := float64(elapsed) / float64(audioDuration) * 100

			t.Logf("%s: 10s audio processed in %v (CPU: %.2f%%), per-frame: %v",
				engine, elapsed, cpuPct, elapsed/time.Duration(iterations))

			// 生产要求：单路 CPU 占用 < 5%
			// race detector adds ~3× overhead, so allow higher budget under race.
			maxCPU := 5.0
			if engine == EngineSilero {
				maxCPU = 10.0 // 神经网络允许更高
			}
			if raceEnabled {
				maxCPU *= 3 // race detector overhead
			}
			if cpuPct > maxCPU {
				t.Errorf("CPU usage %.2f%% exceeds %.1f%% budget", cpuPct, maxCPU)
			}
		})
	}
}

// ─── 测试2: 多路并发 (模拟 100 路并发通话) ───

func TestProduction_Concurrent100Streams(t *testing.T) {
	engines := []EngineKind{EngineEnergy, EngineSilero}

	for _, engine := range engines {
		t.Run(string(engine), func(t *testing.T) {
			cfg := DefaultConfig()
			if engine == EngineSilero {
				cfg.FrameDurationMs = 32
			}

			frames := genOneSecondSpeech(cfg.SampleRate)
			frameMs := cfg.FrameDurationMs
			if engine == EngineSilero {
				frameMs = 32
			}
			framesPerSec := 1000 / frameMs

			const numStreams = 100
			var wg sync.WaitGroup
			start := time.Now()

			for s := 0; s < numStreams; s++ {
				wg.Add(1)
				go func(streamID int) {
					defer wg.Done()
					det, err := NewDetector(engine, cfg)
					if err != nil {
						t.Errorf("create detector: %v", err)
						return
					}
					defer det.Close()

					// 每路处理 2 秒音频
					for i := 0; i < 2*framesPerSec; i++ {
						det.ProcessFrame(frames[i%len(frames)])
					}
				}(s)
			}
			wg.Wait()
			elapsed := time.Since(start)

			// 总音频 = 100 路 × 2 秒 = 200 秒
			totalAudio := time.Duration(numStreams*2) * time.Second
			cpuPct := float64(elapsed) / float64(totalAudio) * 100

			t.Logf("%s: 100 concurrent × 2s = 200s audio in %v (CPU: %.2f%%)",
				engine, elapsed, cpuPct)

			// 100 路并发时 CPU 占用应 < 50%（有多核并行）
			maxConcurrentCPU := 50.0
			if raceEnabled {
				maxConcurrentCPU = 150.0 // race detector overhead
			}
			if cpuPct > maxConcurrentCPU {
				t.Errorf("100-stream CPU %.2f%% exceeds %.0f%% budget", cpuPct, maxConcurrentCPU)
			}
		})
	}
}

// ─── 测试3: 内存占用 (GC 压力) ───

func TestProduction_MemoryFootprint(t *testing.T) {
	engines := []EngineKind{EngineEnergy, EngineWebRTC, EngineSilero}

	for _, engine := range engines {
		t.Run(string(engine), func(t *testing.T) {
			cfg := DefaultConfig()
			if engine == EngineSilero {
				cfg.FrameDurationMs = 32
			}

			var mBefore, mAfter runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&mBefore)

			det, err := NewDetector(engine, cfg)
			if err != nil {
				t.Fatalf("create detector: %v", err)
			}
			defer det.Close()

			frames := genOneSecondSpeech(cfg.SampleRate)
			frameMs := cfg.FrameDurationMs
			if engine == EngineSilero {
				frameMs = 32
			}

			// 处理 60 秒音频
			for i := 0; i < 60*(1000/frameMs); i++ {
				det.ProcessFrame(frames[i%len(frames)])
			}

			runtime.ReadMemStats(&mAfter)
			detectorMem := mAfter.Alloc - mBefore.Alloc
			totalAlloc := mAfter.TotalAlloc - mBefore.TotalAlloc
			mallocs := mAfter.Mallocs - mBefore.Mallocs

			t.Logf("%s: detector=%dKB, total_alloc=%.2fMB, mallocs=%d, GC runs=%d",
				engine,
				detectorMem/1024,
				float64(totalAlloc)/1024/1024,
				mallocs,
				mAfter.NumGC-mBefore.NumGC)

			// 生产要求：60 秒处理总分配 < 10MB（避免 GC 压力）
			maxAlloc := uint64(10 * 1024 * 1024)
			if engine == EngineSilero {
				maxAlloc = 50 * 1024 * 1024 // Silero 神经网络允许更多
			}
			if totalAlloc > maxAlloc {
				t.Errorf("total alloc %.2fMB exceeds %.2fMB budget",
					float64(totalAlloc)/1024/1024, float64(maxAlloc)/1024/1024)
			}
		})
	}
}

// ─── 测试4: 尾延迟 (P50/P99/P999) ───

func TestProduction_TailLatency(t *testing.T) {
	engines := []EngineKind{EngineEnergy, EngineWebRTC, EngineSilero}

	for _, engine := range engines {
		t.Run(string(engine), func(t *testing.T) {
			cfg := DefaultConfig()
			if engine == EngineSilero {
				cfg.FrameDurationMs = 32
			}

			det, err := NewDetector(engine, cfg)
			if err != nil {
				t.Fatalf("create detector: %v", err)
			}
			defer det.Close()

			frames := genOneSecondSpeech(cfg.SampleRate)
			frameMs := cfg.FrameDurationMs
			if engine == EngineSilero {
				frameMs = 32
			}

			// 收集 1000 帧的延迟
			const N = 1000
			latencies := make([]time.Duration, N)

			// 预热
			for i := 0; i < 50; i++ {
				det.ProcessFrame(frames[i%len(frames)])
			}

			for i := 0; i < N; i++ {
				start := time.Now()
				det.ProcessFrame(frames[i%len(frames)])
				latencies[i] = time.Since(start)
			}

			// 排序计算百分位
			sortDurations(latencies)
			p50 := latencies[N/2]
			p99 := latencies[N*99/100]
			p999 := latencies[N*999/1000]
			maxLat := latencies[N-1]

			// 实时预算 = 帧时长
			frameBudget := time.Duration(frameMs) * time.Millisecond

			t.Logf("%s: P50=%v P99=%v P999=%v Max=%v (budget=%v)",
				engine, p50, p99, p999, maxLat, frameBudget)

			// 生产要求：P99 < 帧时长的 50%
			if p99 > frameBudget/2 {
				t.Errorf("P99 %v exceeds 50%% of frame budget %v", p99, frameBudget/2)
			}
			// P999 不应超过帧时长（否则会丢帧）
			if p999 > frameBudget {
				t.Errorf("P999 %v exceeds frame budget %v", p999, frameBudget)
			}
		})
	}
}

// ─── 测试5: 长时间运行稳定性 (10 分钟模拟) ───

func TestProduction_LongRunStability(t *testing.T) {
	det, err := NewSileroDetector(0.5)
	if err != nil {
		t.Fatalf("create Silero: %v", err)
	}
	defer det.Close()

	stream := NewStreamer(det, Config{
		SampleRate:      16000,
		FrameDurationMs: 32,
		MinSpeechFrames: 3,
		HangoverFrames:  15,
	})
	defer stream.Close()

	frames := genOneSecondSpeech(16000)
	frameMs := 32
	// 10 分钟 = 600 秒
	totalFrames := 600 * (1000 / frameMs)

	var speechCount, eventCount int
	var lastFrameIndex int

	for i := 0; i < totalFrames; i++ {
		result, _ := stream.ProcessFrame(frames[i%len(frames)])
		if result.IsSpeech {
			speechCount++
		}
		lastFrameIndex = stream.FrameIndex()
	}

	// 收集残留事件
	for {
		select {
		case <-stream.Events():
			eventCount++
		default:
			goto done
		}
	}
done:

	t.Logf("10min: frames=%d, speech_frames=%d (%.1f%%), events=%d, last_index=%d",
		totalFrames, speechCount, float64(speechCount)/float64(totalFrames)*100,
		eventCount, lastFrameIndex)

	// 验证：所有帧都被处理
	if lastFrameIndex != totalFrames {
		t.Errorf("frame count mismatch: expected %d, got %d", totalFrames, lastFrameIndex)
	}

	// 验证：没有 panic 或死锁
	if totalFrames > 0 && speechCount == 0 {
		t.Error("expected some speech frames in 10 minutes")
	}
}

// ─── 测试6: Streamer 并发读写压力 ───

func TestProduction_StreamerConcurrentStress(t *testing.T) {
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

	frames := genOneSecondSpeech(16000)

	// 1 个写 goroutine + 10 个读 goroutine
	var wg sync.WaitGroup
	done := make(chan struct{})

	// 写
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10000; i++ {
			stream.ProcessFrame(frames[i%len(frames)])
		}
		close(done)
	}()

	// 10 个并发读
	for r := 0; r < 10; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					_ = stream.IsSpeech()
					_ = stream.FrameIndex()
					_ = stream.LastEvent()
				}
			}
		}()
	}

	wg.Wait()
	t.Logf("10000 frames + 10 concurrent readers: OK, final index=%d", stream.FrameIndex())
}

// ─── 测试7: 内存泄漏检测 (创建/销毁循环) ───

func TestProduction_NoMemoryLeak(t *testing.T) {
	var mBefore, mAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&mBefore)

	// 创建/销毁 500 个 detector
	for i := 0; i < 500; i++ {
		det, err := NewSileroDetector(0.5)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		det.ProcessFrame(silentFrame(32, 16000))
		det.Close()
	}

	runtime.GC()
	runtime.ReadMemStats(&mAfter)
	leak := int64(mAfter.Alloc) - int64(mBefore.Alloc)

	t.Logf("500 create/destroy cycles: memory delta = %d KB", leak/1024)

	// 允许一些 GC 残留，但不应超过 5MB
	if leak > 5*1024*1024 {
		t.Errorf("possible memory leak: %d KB after 500 cycles", leak/1024)
	}
}

// ─── 辅助函数 ───

func sortDurations(d []time.Duration) {
	// 简单插入排序（N=1000 足够快）
	for i := 1; i < len(d); i++ {
		key := d[i]
		j := i - 1
		for j >= 0 && d[j] > key {
			d[j+1] = d[j]
			j--
		}
		d[j+1] = key
	}
}
