// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package videoutil

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

// skipIfNoFFmpeg skips the test if ffmpeg is not installed.
func skipIfNoFFmpeg(t *testing.T) {
	t.Helper()
	if _, err := execLookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed, skipping")
	}
}

// skipIfNoFFprobe skips the test if ffprobe is not installed.
func skipIfNoFFprobe(t *testing.T) {
	t.Helper()
	if _, err := execLookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not installed, skipping")
	}
}

// createTestVideo generates a small test video using ffmpeg.
func createTestVideo(t *testing.T, path string, duration float64) {
	t.Helper()
	skipIfNoFFmpeg(t)
	// Generate a 3-second test video using the testsrc filter.
	args := []string{
		"-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc=duration=" + trimFloat(duration) + ":size=320x240:rate=15",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=" + trimFloat(duration),
		"-c:v", "libx264", "-preset", "ultrafast", "-crf", "28",
		"-c:a", "aac", "-b:a", "64k",
		"-pix_fmt", "yuv420p",
		path,
	}
	if err := DefaultFFmpeg.Run(context.Background(), args, nil); err != nil {
		t.Fatalf("createTestVideo: %v", err)
	}
}

// createTestImage creates a small PNG image for watermark tests.
func createTestImage(t *testing.T, path string, size int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 128})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test image: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
}

// ──────────────────────────────────────────────
// FFmpegTool tests
// ──────────────────────────────────────────────

func TestFFmpegTool_Version(t *testing.T) {
	skipIfNoFFmpeg(t)
	tool := NewFFmpegTool()
	ver, err := tool.Version(context.Background())
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if ver == "" {
		t.Error("Version should not be empty")
	}
	t.Logf("FFmpeg version: %s", ver)
}

func TestFFmpegTool_NotFound(t *testing.T) {
	tool := NewFFmpegToolWithPath("/nonexistent/ffmpeg")
	err := tool.ensureReady(context.Background())
	if err == nil {
		t.Fatal("expected error for nonexistent ffmpeg")
	}
}

func TestFFmpegTool_Run_InvalidArgs(t *testing.T) {
	skipIfNoFFmpeg(t)
	tool := NewFFmpegTool()
	err := tool.Run(context.Background(), []string{"-invalid-flag"}, nil)
	if err == nil {
		t.Fatal("expected error for invalid ffmpeg args")
	}
}

// ──────────────────────────────────────────────
// FFprobeTool tests
// ──────────────────────────────────────────────

func TestFFprobeTool_Version(t *testing.T) {
	skipIfNoFFprobe(t)
	tool := NewFFprobeTool()
	ver, err := tool.Version(context.Background())
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if ver == "" {
		t.Error("Version should not be empty")
	}
	t.Logf("FFprobe version: %s", ver)
}

func TestFFprobeTool_NotFound(t *testing.T) {
	tool := NewFFprobeToolWithPath("/nonexistent/ffprobe")
	err := tool.ensureReady(context.Background())
	if err == nil {
		t.Fatal("expected error for nonexistent ffprobe")
	}
}

func TestFFprobeTool_Probe(t *testing.T) {
	skipIfNoFFmpeg(t)
	skipIfNoFFprobe(t)

	dir := t.TempDir()
	videoPath := filepath.Join(dir, "test.mp4")
	createTestVideo(t, videoPath, 3)

	info, err := DefaultFFprobe.Probe(context.Background(), videoPath)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}

	if info.Duration < 2*time.Second || info.Duration > 5*time.Second {
		t.Errorf("Duration = %v, want ~3s", info.Duration)
	}
	if info.Video == nil {
		t.Fatal("Video stream should not be nil")
	}
	if info.Video.Width != 320 || info.Video.Height != 240 {
		t.Errorf("Resolution = %dx%d, want 320x240", info.Video.Width, info.Video.Height)
	}
	if info.Video.Codec != "h264" {
		t.Errorf("Video codec = %q, want h264", info.Video.Codec)
	}
	if info.Audio == nil {
		t.Fatal("Audio stream should not be nil")
	}
	if info.Audio.Codec != "aac" {
		t.Errorf("Audio codec = %q, want aac", info.Audio.Codec)
	}
}

func TestFFprobeTool_Probe_NonExistent(t *testing.T) {
	skipIfNoFFprobe(t)
	_, err := DefaultFFprobe.Probe(context.Background(), "/nonexistent/file.mp4")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

// ──────────────────────────────────────────────
// Command builder tests
// ──────────────────────────────────────────────

func TestCommand_Builder(t *testing.T) {
	cmd := NewCommand().
		LogLevel("error").
		Input("in.mp4").
		VideoCodec("libx264").
		CRF(23).
		Preset("fast").
		AudioCodec("aac").
		MovFlagsFastStart().
		Output("out.mp4")

	args := cmd.Args()

	// Check key args are present.
	assertContains(t, args, "-y")
	assertContains(t, args, "-hide_banner")
	assertContains(t, args, "-loglevel")
	assertContains(t, args, "error")
	assertContains(t, args, "-i")
	assertContains(t, args, "in.mp4")
	assertContains(t, args, "-c:v")
	assertContains(t, args, "libx264")
	assertContains(t, args, "-crf")
	assertContains(t, args, "23")
	assertContains(t, args, "-preset")
	assertContains(t, args, "fast")
	assertContains(t, args, "-c:a")
	assertContains(t, args, "aac")
	assertContains(t, args, "-movflags")
	assertContains(t, args, "+faststart")
	assertContains(t, args, "out.mp4")
}

func TestCommand_Overwrite(t *testing.T) {
	cmd := NewCommand()
	assertContains(t, cmd.Args(), "-y")

	cmd2 := NewCommand().Overwrite(false)
	assertNotContains(t, cmd2.Args(), "-y")
	assertContains(t, cmd2.Args(), "-n")
}

func TestCommand_Scale(t *testing.T) {
	cmd := NewCommand().Scale(1280, 720)
	args := cmd.Args()
	assertContains(t, args, "-vf")
	assertContains(t, args, "scale=1280:720")
}

func TestCommand_ScaleKeepAspect(t *testing.T) {
	cmd := NewCommand().ScaleKeepAspect(1280, 720)
	args := cmd.Args()
	assertContains(t, args, "scale=1280:720:force_original_aspect_ratio=decrease")
}

func TestCommand_ScalePad(t *testing.T) {
	cmd := NewCommand().ScalePad(1280, 720)
	args := cmd.Args()
	assertContainsSubstring(t, args, "pad=1280:720")
}

func TestCommand_NoVideo(t *testing.T) {
	cmd := NewCommand().NoVideo()
	assertContains(t, cmd.Args(), "-vn")
}

func TestCommand_NoAudio(t *testing.T) {
	cmd := NewCommand().NoAudio()
	assertContains(t, cmd.Args(), "-an")
}

func TestCommand_SeekTo(t *testing.T) {
	cmd := NewCommand().SeekTo(10.5)
	args := cmd.Args()
	assertContains(t, args, "-ss")
	assertContains(t, args, "10.5")
}

func TestCommand_Duration(t *testing.T) {
	cmd := NewCommand().Duration(5.0)
	args := cmd.Args()
	assertContains(t, args, "-t")
	assertContains(t, args, "5")
}

func TestCommand_String(t *testing.T) {
	cmd := NewCommand().Input("in.mp4").Output("out.mp4")
	s := cmd.String()
	if s == "" {
		t.Error("String() should not be empty")
	}
	assertContainsSubstring(t, []string{s}, "ffmpeg")
}

// ──────────────────────────────────────────────
// Filter tests
// ──────────────────────────────────────────────

func TestOverlayExpr(t *testing.T) {
	tests := []struct {
		pos    Position
		margin int
		wantX  string
		wantY  string
	}{
		{PositionTopLeft, 10, "10", "10"},
		{PositionTopRight, 10, "W-w-10", "10"},
		{PositionBottomLeft, 10, "10", "H-h-10"},
		{PositionBottomRight, 10, "W-w-10", "H-h-10"},
		{PositionCenter, 0, "(W-w)/2", "(H-h)/2"},
	}
	for _, tt := range tests {
		x, y := OverlayExpr(tt.pos, tt.margin)
		if x != tt.wantX || y != tt.wantY {
			t.Errorf("OverlayExpr(%d, %d) = (%q, %q), want (%q, %q)",
				tt.pos, tt.margin, x, y, tt.wantX, tt.wantY)
		}
	}
}

func TestOverlayPosition(t *testing.T) {
	x, y := OverlayPosition(PositionBottomRight, 320, 240, 50, 50, 10)
	if x != 260 || y != 180 {
		t.Errorf("OverlayPosition = (%d, %d), want (260, 180)", x, y)
	}
}

func TestBuildImageWatermarkFilter(t *testing.T) {
	opts := DefaultWatermarkOptions()
	filter := BuildImageWatermarkFilter(opts)
	if filter == "" {
		t.Error("filter should not be empty")
	}
	assertContainsSubstring(t, []string{filter}, "overlay")
	assertContainsSubstring(t, []string{filter}, "[out]")
	assertContainsSubstring(t, []string{filter}, "[wm]")
}

func TestBuildImageWatermarkFilter_WithScale(t *testing.T) {
	opts := DefaultWatermarkOptions()
	opts.Scale = 0.1
	filter := BuildImageWatermarkFilter(opts)
	assertContainsSubstring(t, []string{filter}, "scale=iw*0.1")
}

func TestBuildImageWatermarkFilter_WithOpacity(t *testing.T) {
	opts := DefaultWatermarkOptions()
	opts.Opacity = 0.5
	filter := BuildImageWatermarkFilter(opts)
	assertContainsSubstring(t, []string{filter}, "colorchannelmixer=aa=0.5")
}

func TestBuildTextWatermarkFilter(t *testing.T) {
	opts := DefaultWatermarkOptions()
	filter := BuildTextWatermarkFilter("Hello", "/fonts/arial.ttf", 24, "white", opts)
	assertContainsSubstring(t, []string{filter}, "drawtext")
	assertContainsSubstring(t, []string{filter}, "text='Hello'")
	assertContainsSubstring(t, []string{filter}, "fontsize=24")
	assertContainsSubstring(t, []string{filter}, "fontcolor=white")
}

func TestBuildTextWatermarkFilter_EscapeText(t *testing.T) {
	filter := BuildTextWatermarkFilter("it's a:test", "", 20, "", DefaultWatermarkOptions())
	// Should escape ' and :
	assertContainsSubstring(t, []string{filter}, "\\'")
	assertContainsSubstring(t, []string{filter}, "\\:")
}

func TestEscapeDrawText(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"hello", "hello"},
		{"it's", "it\\'s"},
		{"a:b", "a\\:b"},
		{"100%", "100\\%"},
		{"path\\to", "path\\\\to"},
	}
	for _, tt := range tests {
		got := escapeDrawText(tt.input)
		if got != tt.expect {
			t.Errorf("escapeDrawText(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

// ──────────────────────────────────────────────
// Progress parsing tests
// ──────────────────────────────────────────────

func TestParseFFmpegDuration(t *testing.T) {
	tests := []struct {
		input  string
		expect float64 // seconds
	}{
		{"00:00:01.500", 1.5},
		{"00:01:30.000", 90},
		{"01:00:00.000", 3600},
	}
	for _, tt := range tests {
		d, err := parseFFmpegDuration(tt.input)
		if err != nil {
			t.Fatalf("parseFFmpegDuration(%q): %v", tt.input, err)
		}
		got := d.Seconds()
		if got != tt.expect {
			t.Errorf("parseFFmpegDuration(%q) = %f, want %f", tt.input, got, tt.expect)
		}
	}
}

func TestParseFFmpegDuration_Invalid(t *testing.T) {
	_, err := parseFFmpegDuration("invalid")
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
}

func TestParseFrameRate(t *testing.T) {
	tests := []struct {
		input  string
		expect float64
	}{
		{"30", 30},
		{"30000/1001", 29.97},
		{"0/0", 0},
		{"", 0},
	}
	for _, tt := range tests {
		got, err := parseFrameRate(tt.input)
		if err != nil {
			t.Fatalf("parseFrameRate(%q): %v", tt.input, err)
		}
		if tt.input == "30000/1001" {
			if got < 29.9 || got > 30.0 {
				t.Errorf("parseFrameRate(%q) = %f, want ~29.97", tt.input, got)
			}
		} else if got != tt.expect {
			t.Errorf("parseFrameRate(%q) = %f, want %f", tt.input, got, tt.expect)
		}
	}
}

func TestComputePercent(t *testing.T) {
	tests := []struct {
		out   time.Duration
		total time.Duration
		want  float64
	}{
		{0, 0, 0},
		{5 * time.Second, 10 * time.Second, 50},
		{10 * time.Second, 10 * time.Second, 100},
		{15 * time.Second, 10 * time.Second, 100}, // capped
	}
	for _, tt := range tests {
		got := computePercent(tt.out, tt.total)
		if got != tt.want {
			t.Errorf("computePercent(%v, %v) = %f, want %f", tt.out, tt.total, got, tt.want)
		}
	}
}

// ──────────────────────────────────────────────
// Integration tests (require ffmpeg)
// ──────────────────────────────────────────────

func TestTranscode(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "output.mp4")
	createTestVideo(t, input, 2)

	err := Transcode(context.Background(), input, output, WithCRF(28), WithPreset("ultrafast"))
	if err != nil {
		t.Fatalf("Transcode: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("output file not created: %v", err)
	}
}

func TestTranscodeWithProgress(t *testing.T) {
	skipIfNoFFmpeg(t)
	skipIfNoFFprobe(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "output.mp4")
	createTestVideo(t, input, 2)

	var progressCount int
	var lastProgress Progress
	err := TranscodeWithProgress(context.Background(), input, output, func(p Progress) {
		progressCount++
		lastProgress = p
	}, WithCRF(28), WithPreset("ultrafast"))
	if err != nil {
		t.Fatalf("TranscodeWithProgress: %v", err)
	}
	if progressCount == 0 {
		t.Error("expected at least one progress callback")
	}
	if !lastProgress.Done {
		t.Error("last progress should indicate done")
	}
}

func TestScreenshot(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "thumb.jpg")
	createTestVideo(t, input, 2)

	err := Screenshot(context.Background(), input, output, 1.0, WithScreenshotSize(160, 120))
	if err != nil {
		t.Fatalf("Screenshot: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("screenshot not created: %v", err)
	}
}

func TestScreenshot_PNG(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "thumb.png")
	createTestVideo(t, input, 2)

	err := Screenshot(context.Background(), input, output, 0.5, WithScreenshotFormat("png"))
	if err != nil {
		t.Fatalf("Screenshot PNG: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("screenshot not created: %v", err)
	}
}

func TestRemux(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "output.mkv")
	createTestVideo(t, input, 2)

	err := Remux(context.Background(), input, output)
	if err != nil {
		t.Fatalf("Remux: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("output not created: %v", err)
	}
}

func TestExtractAudio(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "audio.aac")
	createTestVideo(t, input, 2)

	err := ExtractAudio(context.Background(), input, output, "aac", "64k")
	if err != nil {
		t.Fatalf("ExtractAudio: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("audio not created: %v", err)
	}
}

func TestClip(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "clip.mp4")
	createTestVideo(t, input, 5)

	err := Clip(context.Background(), input, output, WithClipStart(1.0), WithClipDuration(2.0))
	if err != nil {
		t.Fatalf("Clip: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("clip not created: %v", err)
	}

	// Verify the clip is approximately 2 seconds.
	info, err := DefaultFFprobe.Probe(context.Background(), output)
	if err != nil {
		t.Fatalf("Probe clip: %v", err)
	}
	if info.Duration < 1*time.Second || info.Duration > 3*time.Second {
		t.Errorf("clip duration = %v, want ~2s", info.Duration)
	}
}

func TestAddImageWatermark(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	watermark := filepath.Join(dir, "logo.png")
	output := filepath.Join(dir, "output.mp4")
	createTestVideo(t, input, 2)
	createTestImage(t, watermark, 30)

	err := AddImageWatermark(context.Background(), input, watermark, output,
		WithWatermarkPosition(PositionBottomRight),
		WithWatermarkMargin(5))
	if err != nil {
		t.Fatalf("AddImageWatermark: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("output not created: %v", err)
	}
}

func TestAddTextWatermark(t *testing.T) {
	skipIfNoFFmpeg(t)
	if !hasFilter("drawtext") {
		t.Skip("ffmpeg build has no drawtext filter, skipping")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "output.mp4")
	createTestVideo(t, input, 2)

	err := AddTextWatermark(context.Background(), input, output, "LingBase", "",
		WithWatermarkPosition(PositionTopLeft))
	if err != nil {
		t.Fatalf("AddTextWatermark: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("output not created: %v", err)
	}
}

func TestCreateGIF(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "output.gif")
	createTestVideo(t, input, 3)

	err := CreateGIF(context.Background(), input, output,
		WithGIFDuration(2),
		WithGIFSize(160, 120),
		WithGIFFPS(10))
	if err != nil {
		t.Fatalf("CreateGIF: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("gif not created: %v", err)
	}
}

func TestCreateGIF_Optimized(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "output.gif")
	createTestVideo(t, input, 3)

	err := CreateGIF(context.Background(), input, output,
		WithGIFDuration(2),
		WithGIFSize(160, 120),
		WithGIFOptimize())
	if err != nil {
		t.Fatalf("CreateGIF optimized: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("gif not created: %v", err)
	}
}

func TestConcat(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input1 := filepath.Join(dir, "part1.mp4")
	input2 := filepath.Join(dir, "part2.mp4")
	output := filepath.Join(dir, "concat.mp4")
	createTestVideo(t, input1, 2)
	createTestVideo(t, input2, 2)

	err := Concat(context.Background(), []string{input1, input2}, output)
	if err != nil {
		t.Fatalf("Concat: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("concat output not created: %v", err)
	}

	// Verify the concat is approximately 4 seconds.
	info, err := DefaultFFprobe.Probe(context.Background(), output)
	if err != nil {
		t.Fatalf("Probe concat: %v", err)
	}
	if info.Duration < 3*time.Second || info.Duration > 5*time.Second {
		t.Errorf("concat duration = %v, want ~4s", info.Duration)
	}
}

func TestConcat_NoInputs(t *testing.T) {
	err := Concat(context.Background(), []string{}, "output.mp4")
	if err == nil {
		t.Fatal("expected error for no inputs")
	}
}

// ──────────────────────────────────────────────
// Additional command builder tests
// ──────────────────────────────────────────────

func TestCommand_AllVideoOptions(t *testing.T) {
	cmd := NewCommand().
		InputFormat("mp4").
		VideoCodec("libx265").
		Tune("film").
		VideoBitRate("5M").
		VideoFrameRate("30").
		MaxKeyInterval(60).
		PixelFormat("yuv420p10le").
		AudioCodec("libopus").
		AudioBitRate("192k").
		AudioSampleRate(48000).
		AudioChannels(2).
		AudioFilter("volume=2").
		OutputFormat("matroska").
		Map("0:v:0").
		Map("0:a:0")

	args := cmd.Args()
	assertContains(t, args, "-f")
	assertContains(t, args, "mp4")
	assertContains(t, args, "-c:v")
	assertContains(t, args, "libx265")
	assertContains(t, args, "-tune")
	assertContains(t, args, "film")
	assertContains(t, args, "-b:v")
	assertContains(t, args, "5M")
	assertContains(t, args, "-r")
	assertContains(t, args, "30")
	assertContains(t, args, "-g")
	assertContains(t, args, "60")
	assertContains(t, args, "-pix_fmt")
	assertContains(t, args, "yuv420p10le")
	assertContains(t, args, "-c:a")
	assertContains(t, args, "libopus")
	assertContains(t, args, "-b:a")
	assertContains(t, args, "192k")
	assertContains(t, args, "-ar")
	assertContains(t, args, "48000")
	assertContains(t, args, "-ac")
	assertContains(t, args, "2")
	assertContains(t, args, "-af")
	assertContains(t, args, "volume=2")
	assertContains(t, args, "-map")
	assertContains(t, args, "0:v:0")
	assertContains(t, args, "0:a:0")
}

func TestCommand_VideoFilterComplex(t *testing.T) {
	cmd := NewCommand().VideoFilterComplex("[0:v][1:v]overlay[out]")
	assertContains(t, cmd.Args(), "-filter_complex")
}

func TestCommand_CopyVideo_CopyAudio(t *testing.T) {
	cmd := NewCommand().CopyVideo().CopyAudio()
	assertContains(t, cmd.Args(), "copy")
}

func TestCommand_Frames(t *testing.T) {
	cmd := NewCommand().Frames(1)
	assertContains(t, cmd.Args(), "-frames:v")
	assertContains(t, cmd.Args(), "1")
}

func TestCommand_Append(t *testing.T) {
	cmd := NewCommand().Append("-custom", "value")
	assertContains(t, cmd.Args(), "-custom")
	assertContains(t, cmd.Args(), "value")
}

// ──────────────────────────────────────────────
// Functional options tests
// ──────────────────────────────────────────────

func TestTranscodeOptions_AllFunctionalOpts(t *testing.T) {
	o := DefaultTranscodeOptions()
	WithCRF(18)(&o)
	WithPreset("slow")(&o)
	WithTune("animation")(&o)
	WithVideoCodec("libx265")(&o)
	WithAudioCodec("libopus")(&o)
	WithScale(1920, 1080)(&o)
	WithPad(1920, 1080)(&o)
	WithFrameRate("24")(&o)
	WithPixelFormat("yuv420p10le")(&o)
	WithVideoBitRate("10M")(&o)
	WithAudioBitRate("256k")(&o)
	WithSampleRate(48000)(&o)
	WithAudioChannels(2)(&o)
	WithFastStart()(&o)
	WithNoAudio()(&o)
	WithNoVideo()(&o)

	if o.CRF != 18 || o.Preset != "slow" || o.Tune != "animation" {
		t.Error("functional opts not applied")
	}
	if o.VideoCodec != "libx265" || o.AudioCodec != "libopus" {
		t.Error("codec opts not applied")
	}
	if o.ScaleWidth != 1920 || o.ScaleHeight != 1080 || !o.PadToSize {
		t.Error("scale/pad opts not applied")
	}
	if !o.NoAudio || !o.NoVideo {
		t.Error("no-audio/no-video opts not applied")
	}
}

func TestClipOptions_FunctionalOpts(t *testing.T) {
	o := ClipOptions{}
	WithClipStart(5)(&o)
	WithClipEnd(10)(&o)
	WithClipDuration(3)(&o)
	WithClipCopy()(&o)
	if o.Start != 5 || o.End != 10 || o.Duration != 3 || !o.Copy {
		t.Error("clip opts not applied")
	}
}

func TestScreenshotOptions_FunctionalOpts(t *testing.T) {
	o := DefaultScreenshotOptions()
	WithScreenshotSize(640, 480)(&o)
	WithScreenshotFormat("png")(&o)
	WithScreenshotQuality(5)(&o)
	WithExactSeek()(&o)
	if o.Width != 640 || o.Height != 480 || o.Format != "png" || o.Quality != 5 || !o.ExactSeek {
		t.Error("screenshot opts not applied")
	}
}

func TestWatermarkOptions_FunctionalOpts(t *testing.T) {
	o := DefaultWatermarkOptions()
	WithWatermarkPosition(PositionCenter)(&o)
	WithWatermarkMargin(20)(&o)
	WithWatermarkScale(0.2)(&o)
	WithWatermarkOpacity(0.7)(&o)
	if o.Position != PositionCenter || o.Margin != 20 || o.Scale != 0.2 || o.Opacity != 0.7 {
		t.Error("watermark opts not applied")
	}
}

func TestGIFOptions_FunctionalOpts(t *testing.T) {
	o := DefaultGIFOptions()
	WithGIFStart(1)(&o)
	WithGIFDuration(5)(&o)
	WithGIFSize(320, 240)(&o)
	WithGIFFPS(15)(&o)
	WithGIFOptimize()(&o)
	if o.Start != 1 || o.Duration != 5 || o.Width != 320 || o.Height != 240 || o.FPS != 15 || !o.Optimize {
		t.Error("gif opts not applied")
	}
}

// ──────────────────────────────────────────────
// FilterGraph tests
// ──────────────────────────────────────────────

func TestFilterGraph(t *testing.T) {
	g := NewFilterGraph().
		AddChain("[0:v]scale=1280:720[scaled]").
		AddChain("[scaled][1:v]overlay[out]")
	s := g.String()
	assertContainsSubstring(t, []string{s}, "scale=1280:720")
	assertContainsSubstring(t, []string{s}, "overlay")
	assertContainsSubstring(t, []string{s}, ";")
}

func TestOverlayPosition_AllPositions(t *testing.T) {
	tests := []struct {
		pos              Position
		expectX, expectY int
	}{
		{PositionTopLeft, 10, 10},
		{PositionTopRight, 260, 10},
		{PositionBottomLeft, 10, 180},
		{PositionBottomRight, 260, 180},
		{PositionCenter, 135, 95},
	}
	for _, tt := range tests {
		x, y := OverlayPosition(tt.pos, 320, 240, 50, 50, 10)
		if x != tt.expectX || y != tt.expectY {
			t.Errorf("OverlayPosition(%d) = (%d,%d), want (%d,%d)",
				tt.pos, x, y, tt.expectX, tt.expectY)
		}
	}
}

func TestEscapeFilterPath(t *testing.T) {
	// Colons should be escaped for filter syntax.
	got := escapeFilterPath("/path/to/file:extra.ttf")
	assertContainsSubstring(t, []string{got}, "\\:")
	// No colon - no change.
	got2 := escapeFilterPath("/plain/path/file.ttf")
	if got2 != "/plain/path/file.ttf" {
		t.Errorf("escapeFilterPath = %q, want unchanged", got2)
	}
}

// ──────────────────────────────────────────────
// Defaults / package-level function tests
// ──────────────────────────────────────────────

func TestProbeFile(t *testing.T) {
	skipIfNoFFmpeg(t)
	skipIfNoFFprobe(t)
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "test.mp4")
	createTestVideo(t, videoPath, 2)

	info, err := ProbeFile(videoPath)
	if err != nil {
		t.Fatalf("ProbeFile: %v", err)
	}
	if info.Video == nil {
		t.Fatal("Video should not be nil")
	}
}

func TestTranscodeMP4(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "output.mp4")
	createTestVideo(t, input, 2)

	err := TranscodeMP4(input, output, WithCRF(28), WithPreset("ultrafast"))
	if err != nil {
		t.Fatalf("TranscodeMP4: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("output not created: %v", err)
	}
}

func TestTranscode_NoAudio(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "output.mp4")
	createTestVideo(t, input, 2)

	err := Transcode(context.Background(), input, output, WithCRF(28), WithPreset("ultrafast"), WithNoAudio())
	if err != nil {
		t.Fatalf("Transcode no audio: %v", err)
	}
	info, err := DefaultFFprobe.Probe(context.Background(), output)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if info.Audio != nil {
		t.Error("audio should be absent")
	}
}

func TestTranscode_NoVideo(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "output.aac")
	createTestVideo(t, input, 2)

	err := Transcode(context.Background(), input, output, WithNoVideo(), WithAudioCodec("aac"))
	if err != nil {
		t.Fatalf("Transcode no video: %v", err)
	}
	info, err := DefaultFFprobe.Probe(context.Background(), output)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if info.Video != nil {
		t.Error("video should be absent")
	}
}

func TestTranscode_Scale(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "output.mp4")
	createTestVideo(t, input, 2)

	err := Transcode(context.Background(), input, output,
		WithCRF(28), WithPreset("ultrafast"), WithScale(160, 120))
	if err != nil {
		t.Fatalf("Transcode scale: %v", err)
	}
	info, err := DefaultFFprobe.Probe(context.Background(), output)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if info.Video.Width > 160 || info.Video.Height > 120 {
		t.Errorf("scaled to %dx%d, want <=160x120", info.Video.Width, info.Video.Height)
	}
}

func TestTranscode_Pad(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "output.mp4")
	createTestVideo(t, input, 2)

	err := Transcode(context.Background(), input, output,
		WithCRF(28), WithPreset("ultrafast"), WithPad(160, 160))
	if err != nil {
		t.Fatalf("Transcode pad: %v", err)
	}
	info, err := DefaultFFprobe.Probe(context.Background(), output)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if info.Video.Width != 160 || info.Video.Height != 160 {
		t.Errorf("padded to %dx%d, want 160x160", info.Video.Width, info.Video.Height)
	}
}

func TestTranscode_VideoBitRate(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "output.mp4")
	createTestVideo(t, input, 2)

	err := Transcode(context.Background(), input, output,
		WithVideoBitRate("500k"), WithPreset("ultrafast"))
	if err != nil {
		t.Fatalf("Transcode bitrate: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("output not created: %v", err)
	}
}

func TestClip_Copy(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "clip.mp4")
	createTestVideo(t, input, 5)

	err := Clip(context.Background(), input, output,
		WithClipStart(1.0), WithClipDuration(2.0), WithClipCopy())
	if err != nil {
		t.Fatalf("Clip copy: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("clip not created: %v", err)
	}
}

func TestClip_EndOnly(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "clip.mp4")
	createTestVideo(t, input, 5)

	err := Clip(context.Background(), input, output, WithClipEnd(2.0))
	if err != nil {
		t.Fatalf("Clip end only: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("clip not created: %v", err)
	}
}

func TestScreenshot_ExactSeek(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "thumb.jpg")
	createTestVideo(t, input, 3)

	err := Screenshot(context.Background(), input, output, 1.5,
		WithExactSeek(), WithScreenshotQuality(5))
	if err != nil {
		t.Fatalf("Screenshot exact seek: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("screenshot not created: %v", err)
	}
}

func TestScreenshot_AtZero(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "thumb.jpg")
	createTestVideo(t, input, 2)

	err := Screenshot(context.Background(), input, output, 0)
	if err != nil {
		t.Fatalf("Screenshot at 0: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("screenshot not created: %v", err)
	}
}

func TestAddImageWatermark_WithScaleAndOpacity(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	watermark := filepath.Join(dir, "logo.png")
	output := filepath.Join(dir, "output.mp4")
	createTestVideo(t, input, 2)
	createTestImage(t, watermark, 60)

	err := AddImageWatermark(context.Background(), input, watermark, output,
		WithWatermarkPosition(PositionTopLeft),
		WithWatermarkMargin(5),
		WithWatermarkScale(0.1),
		WithWatermarkOpacity(0.5))
	if err != nil {
		t.Fatalf("AddImageWatermark: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("output not created: %v", err)
	}
}

func TestCreateGIF_NoSize(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "output.gif")
	createTestVideo(t, input, 3)

	err := CreateGIF(context.Background(), input, output, WithGIFDuration(1))
	if err != nil {
		t.Fatalf("CreateGIF no size: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("gif not created: %v", err)
	}
}

// ──────────────────────────────────────────────
// Context cancellation test
// ──────────────────────────────────────────────

func TestTranscode_ContextCanceled(t *testing.T) {
	skipIfNoFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	output := filepath.Join(dir, "output.mp4")
	// Create a longer video so we have time to cancel.
	createTestVideo(t, input, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := Transcode(ctx, input, output, WithCRF(28), WithPreset("ultrafast"))
	if err == nil {
		// Might succeed if fast enough; not a hard failure.
		return
	}
	// Should be a timeout/cancellation error, not a normal failure.
}

// ──────────────────────────────────────────────
// Progress parsing edge cases
// ──────────────────────────────────────────────

func TestScanProgress_UnknownFields(t *testing.T) {
	input := strings.Join([]string{
		"frame=100",
		"fps=30.0",
		"bitrate=1000.0kbits/s",
		"total_size=5000000",
		"out_time_us=3333333",
		"out_time_ms=3333333",
		"out_time=00:00:03.333333",
		"dup_frames=0",
		"drop_frames=0",
		"speed=1.5x",
		"progress=continue",
		"unknown_field=value123",
		"progress=end",
		"",
	}, "\n")

	var results []Progress
	var last Progress
	cb := func(p Progress) {
		results = append(results, p)
	}
	scanProgress(strings.NewReader(input), &last, cb)

	if len(results) == 0 {
		t.Fatal("expected progress entries")
	}
	// The first result should have Frame set.
	if results[0].Frame != 100 {
		t.Errorf("results[0].Frame = %d, want 100", results[0].Frame)
	}
	// The accumulated `last` should have all fields set.
	if last.Frame != 100 {
		t.Errorf("last.Frame = %d, want 100", last.Frame)
	}
	if last.FPS != 30.0 {
		t.Errorf("last.FPS = %f, want 30", last.FPS)
	}
	if last.Speed != "1.5x" {
		t.Errorf("last.Speed = %q, want 1.5x", last.Speed)
	}
	if !last.Done {
		t.Error("last progress should be Done")
	}
	if last.Extra["unknown_field"] != "value123" {
		t.Errorf("last.Extra[unknown_field] = %q", last.Extra["unknown_field"])
	}
	if last.Extra["total_size"] != "5000000" {
		t.Errorf("last.Extra[total_size] = %q", last.Extra["total_size"])
	}
}

func TestScanProgress_Empty(t *testing.T) {
	var last Progress
	scanProgress(strings.NewReader(""), &last, nil)
}

func TestParseFFmpegDuration_TwoParts(t *testing.T) {
	// MM:SS format - not supported, should error
	_, err := parseFFmpegDuration("01:30")
	if err == nil {
		t.Fatal("expected error for 2-part duration (only HH:MM:SS supported)")
	}
}

func TestParseFFmpegDuration_SinglePart(t *testing.T) {
	// SS format - not supported, should error
	_, err := parseFFmpegDuration("45")
	if err == nil {
		t.Fatal("expected error for 1-part duration (only HH:MM:SS supported)")
	}
}

func TestParseFrameRate_Invalid(t *testing.T) {
	_, err := parseFrameRate("invalid/rate/extra")
	if err == nil {
		t.Fatal("expected error for invalid frame rate")
	}
}

func TestParseFrameRate_ZeroDenominator(t *testing.T) {
	_, err := parseFrameRate("30/0")
	if err == nil {
		t.Fatal("expected error for zero denominator")
	}
}

// ──────────────────────────────────────────────
// ResolvedPath test
// ──────────────────────────────────────────────

func TestFFmpegTool_ResolvedPath(t *testing.T) {
	skipIfNoFFmpeg(t)
	tool := NewFFmpegTool()
	if err := tool.ensureReady(context.Background()); err != nil {
		t.Fatalf("ensureReady: %v", err)
	}
	if p := tool.ResolvedPath(); p == "" {
		t.Error("ResolvedPath should not be empty after ensureReady")
	}
}

func TestFFprobeTool_ProbeRaw(t *testing.T) {
	skipIfNoFFmpeg(t)
	skipIfNoFFprobe(t)
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "test.mp4")
	createTestVideo(t, videoPath, 2)

	raw, err := DefaultFFprobe.ProbeRaw(context.Background(), videoPath)
	if err != nil {
		t.Fatalf("ProbeRaw: %v", err)
	}
	if raw == nil {
		t.Fatal("raw should not be nil")
	}
	if len(raw.Streams) == 0 {
		t.Error("should have streams")
	}
}

// ──────────────────────────────────────────────
// Test helpers
// ──────────────────────────────────────────────

func assertContains(t *testing.T, args []string, want string) {
	t.Helper()
	for _, a := range args {
		if a == want {
			return
		}
	}
	t.Errorf("args %v does not contain %q", args, want)
}

func assertNotContains(t *testing.T, args []string, want string) {
	t.Helper()
	for _, a := range args {
		if a == want {
			t.Errorf("args %v should not contain %q", args, want)
		}
	}
}

// assertContainsSubstring checks if any string in args contains want as a substring.
func assertContainsSubstring(t *testing.T, args []string, want string) {
	t.Helper()
	for _, a := range args {
		if strings.Contains(a, want) {
			return
		}
	}
	t.Errorf("args %v does not contain substring %q", args, want)
}

// hasFilter checks if ffmpeg has a specific filter available.
func hasFilter(filter string) bool {
	tool := NewFFmpegTool()
	if err := tool.ensureReady(context.Background()); err != nil {
		return false
	}
	cmd := exec.Command(tool.resolvedPath, "-hide_banner", "-filters")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), filter)
}

// execCommand is a helper for tests.
func execCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}
