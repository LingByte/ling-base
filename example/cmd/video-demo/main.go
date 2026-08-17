// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Command video-demo demonstrates the ling-base common/videoutil package.
//
// It generates a small test video using FFmpeg's lavfi source, then
// performs several operations on it:
//
//   - Probe (inspect media metadata)
//   - Transcode (H.264/AAC MP4 with progress reporting)
//   - Screenshot (extract a frame as JPEG)
//   - Image watermark (overlay a PNG logo)
//   - Text watermark (overlay text using drawtext)
//   - GIF creation (animated GIF from a segment)
//   - Clip (extract a segment)
//   - Extract audio
//
// All outputs are written to an output directory.
//
// Usage:
//
//	go run ./cmd/video-demo
//	go run ./cmd/video-demo -out ./out/video
//	go run ./cmd/video-demo -ffmpeg /usr/local/bin/ffmpeg
package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/LingByte/ling-base/common/videoutil"
)

func main() {
	outDir := flag.String("out", "out/video", "output directory")
	ffmpegPath := flag.String("ffmpeg", "ffmpeg", "path to ffmpeg binary")
	ffprobePath := flag.String("ffprobe", "ffprobe", "path to ffprobe binary")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create out dir: %v\n", err)
		os.Exit(1)
	}

	// Configure tools with explicit binary paths.
	ffmpeg := videoutil.NewFFmpegToolWithPath(*ffmpegPath)
	ffprobe := videoutil.NewFFprobeToolWithPath(*ffprobePath)

	ctx := context.Background()

	// Check FFmpeg availability.
	ver, err := ffmpeg.Version(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ffmpeg not available: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("FFmpeg version: %s\n\n", ver)

	pver, err := ffprobe.Version(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ffprobe not available: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("FFprobe version: %s\n\n", pver)

	// Step 1: Generate a test video using lavfi.
	inputPath := filepath.Join(*outDir, "source.mp4")
	fmt.Println("=== Generating test video ===")
	if err := generateTestVideo(ctx, ffmpeg, inputPath); err != nil {
		fmt.Fprintf(os.Stderr, "generate test video: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created: %s\n\n", inputPath)

	// Step 2: Probe the video.
	fmt.Println("=== Probing media ===")
	info, err := ffprobe.Probe(ctx, inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "probe: %v\n", err)
		os.Exit(1)
	}
	printMediaInfo(info)

	// Step 3: Transcode with progress.
	fmt.Println("\n=== Transcoding (H.264/AAC MP4) ===")
	outputPath := filepath.Join(*outDir, "transcoded.mp4")
	err = ffmpeg.TranscodeWithProgress(ctx, inputPath, outputPath, func(p videoutil.Progress) {
		if p.Percent > 0 {
			fmt.Printf("\r  Progress: %.1f%% (frame=%d, fps=%.1f, speed=%s)    ",
				p.Percent, p.Frame, p.FPS, p.Speed)
		}
	}, videoutil.WithCRF(23), videoutil.WithPreset("fast"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "\ntranscode: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n  Created: %s\n", outputPath)

	// Step 4: Screenshot.
	fmt.Println("\n=== Screenshot extraction ===")
	thumbPath := filepath.Join(*outDir, "screenshot.jpg")
	if err := ffmpeg.Screenshot(ctx, inputPath, thumbPath, 1.0,
		videoutil.WithScreenshotSize(320, 240)); err != nil {
		fmt.Fprintf(os.Stderr, "screenshot: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Created: %s\n", thumbPath)

	// Step 5: Image watermark.
	fmt.Println("\n=== Image watermark ===")
	logoPath := filepath.Join(*outDir, "logo.png")
	if err := createLogoPNG(logoPath, 60); err != nil {
		fmt.Fprintf(os.Stderr, "create logo: %v\n", err)
		os.Exit(1)
	}
	wmPath := filepath.Join(*outDir, "watermarked.mp4")
	if err := ffmpeg.AddImageWatermark(ctx, inputPath, logoPath, wmPath,
		videoutil.WithWatermarkPosition(videoutil.PositionBottomRight),
		videoutil.WithWatermarkMargin(15),
		videoutil.WithWatermarkOpacity(0.8)); err != nil {
		fmt.Fprintf(os.Stderr, "image watermark: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Created: %s\n", wmPath)

	// Step 6: Text watermark.
	fmt.Println("\n=== Text watermark ===")
	textWMPath := filepath.Join(*outDir, "text_watermarked.mp4")
	if err := ffmpeg.AddTextWatermark(ctx, inputPath, textWMPath, "LingBase Demo", "",
		videoutil.WithWatermarkPosition(videoutil.PositionTopLeft),
		videoutil.WithWatermarkMargin(10),
		videoutil.WithWatermarkOpacity(0.7)); err != nil {
		fmt.Fprintf(os.Stderr, "text watermark: %v\n", err)
		fmt.Fprintf(os.Stderr, "  (drawtext filter may not be available in this ffmpeg build)\n")
	} else {
		fmt.Printf("  Created: %s\n", textWMPath)
	}

	// Step 7: GIF creation.
	fmt.Println("\n=== GIF creation ===")
	gifPath := filepath.Join(*outDir, "clip.gif")
	if err := ffmpeg.CreateGIF(ctx, inputPath, gifPath,
		videoutil.WithGIFDuration(2),
		videoutil.WithGIFSize(160, 120),
		videoutil.WithGIFFPS(10),
		videoutil.WithGIFOptimize()); err != nil {
		fmt.Fprintf(os.Stderr, "create gif: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Created: %s\n", gifPath)

	// Step 8: Clip extraction.
	fmt.Println("\n=== Clip extraction ===")
	clipPath := filepath.Join(*outDir, "clip.mp4")
	if err := ffmpeg.Clip(ctx, inputPath, clipPath,
		videoutil.WithClipStart(1.0),
		videoutil.WithClipDuration(2.0)); err != nil {
		fmt.Fprintf(os.Stderr, "clip: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Created: %s\n", clipPath)

	// Step 9: Extract audio.
	fmt.Println("\n=== Audio extraction ===")
	audioPath := filepath.Join(*outDir, "audio.aac")
	if err := ffmpeg.ExtractAudio(ctx, inputPath, audioPath, "aac", "128k"); err != nil {
		fmt.Fprintf(os.Stderr, "extract audio: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Created: %s\n", audioPath)

	fmt.Println("\n=== All operations completed! ===")
}

// generateTestVideo creates a 5-second test video using FFmpeg's lavfi source.
func generateTestVideo(ctx context.Context, tool *videoutil.FFmpegTool, path string) error {
	args := []string{
		"-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc=duration=5:size=320x240:rate=15",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=5",
		"-c:v", "libx264", "-preset", "ultrafast", "-crf", "28",
		"-c:a", "aac", "-b:a", "64k",
		"-pix_fmt", "yuv420p",
		path,
	}
	return tool.Run(ctx, args, nil)
}

// createLogoPNG creates a simple red semi-transparent PNG logo.
func createLogoPNG(path string, size int) error {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			// Create a simple pattern.
			if (x+y)%20 < 10 {
				img.Set(x, y, color.RGBA{R: 255, G: 100, B: 50, A: 200})
			} else {
				img.Set(x, y, color.RGBA{R: 50, G: 100, B: 255, A: 200})
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// printMediaInfo prints probed media metadata.
func printMediaInfo(info *videoutil.MediaInfo) {
	fmt.Printf("  Filename:   %s\n", info.Filename)
	fmt.Printf("  Format:     %s\n", info.Format)
	fmt.Printf("  Duration:   %s\n", info.Duration.Round(time.Millisecond))
	fmt.Printf("  Size:       %d bytes\n", info.Size)
	fmt.Printf("  BitRate:    %d bps\n", info.BitRate)
	if info.Video != nil {
		fmt.Printf("  Video:      %s %dx%d @ %.2ffps (pixel: %s)\n",
			info.Video.Codec, info.Video.Width, info.Video.Height,
			info.Video.FPS, info.Video.PixelFormat)
	}
	if info.Audio != nil {
		fmt.Printf("  Audio:      %s %dHz %dch (layout: %s)\n",
			info.Audio.Codec, info.Audio.SampleRate,
			info.Audio.Channels, info.Audio.ChannelLayout)
	}
	fmt.Printf("  Streams:    %d total\n", len(info.Streams))
}
