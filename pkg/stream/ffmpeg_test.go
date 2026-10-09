package stream

import (
	"strings"
	"testing"
)

func TestBuildFFmpegArgs(t *testing.T) {
	opts := FFmpegOptions{
		Display:    ":20",
		FPS:        30,
		Resolution: "1080p",
		Preset:     "balanced",
		HWAccel:    "none",
	}

	args := BuildFFmpegArgs(opts)
	cmdLine := strings.Join(args, " ")

	if !strings.Contains(cmdLine, "-f x11grab") {
		t.Errorf("expected x11grab in args, got: %s", cmdLine)
	}
	if !strings.Contains(cmdLine, ":20.0") {
		t.Errorf("expected :20.0 in args, got: %s", cmdLine)
	}
	if !strings.Contains(cmdLine, "-f pulse") {
		t.Errorf("expected pulse audio in args, got: %s", cmdLine)
	}
	if !strings.Contains(cmdLine, "-f hls") {
		t.Errorf("expected hls in args, got: %s", cmdLine)
	}
	if !strings.Contains(cmdLine, "live.m3u8") {
		t.Errorf("expected live.m3u8 in args, got: %s", cmdLine)
	}

	// Test crop mode
	optsCrop := FFmpegOptions{
		Display:    ":20",
		FPS:        30,
		Resolution: "1080p",
		Fit:        "crop",
		HWAccel:    "none",
	}
	argsCrop := strings.Join(BuildFFmpegArgs(optsCrop), " ")
	if !strings.Contains(argsCrop, "scale=1920:1080") {
		t.Errorf("expected scale=1920:1080 in crop mode, got: %s", argsCrop)
	}

	// Test stretch mode
	optsStretch := FFmpegOptions{
		Display:    ":20",
		FPS:        30,
		Resolution: "1080p",
		Fit:        "stretch",
		HWAccel:    "none",
	}
	argsStretch := strings.Join(BuildFFmpegArgs(optsStretch), " ")
	if !strings.Contains(argsStretch, "scale=1920:1080") {
		t.Errorf("expected scale=1920:1080 in stretch mode, got: %s", argsStretch)
	}
}

func TestBuildFFmpegArgs_Presets(t *testing.T) {
	// Ultra-low preset
	optsUL := FFmpegOptions{
		Display:    ":20",
		FPS:        30,
		Resolution: "1080p",
		Preset:     "ultra-low",
		HWAccel:    "none",
	}
	argsUL := strings.Join(BuildFFmpegArgs(optsUL), " ")
	if !strings.Contains(argsUL, "-hls_time 0.5") {
		t.Errorf("expected -hls_time 0.5 for ultra-low preset, got: %s", argsUL)
	}

	// Cinema preset
	optsCinema := FFmpegOptions{
		Display:    ":20",
		FPS:        30,
		Resolution: "1080p",
		Preset:     "cinema",
		HWAccel:    "none",
	}
	argsCinema := strings.Join(BuildFFmpegArgs(optsCinema), " ")
	if !strings.Contains(argsCinema, "-hls_time 2") {
		t.Errorf("expected -hls_time 2 for cinema preset, got: %s", argsCinema)
	}
}

func TestBuildFFmpegArgs_AudioOnly(t *testing.T) {
	opts := FFmpegOptions{
		AudioOnly: true,
		Preset:    "balanced",
	}
	args := strings.Join(BuildFFmpegArgs(opts), " ")
	if !strings.Contains(args, "-vn") {
		t.Errorf("expected -vn for audio only, got: %s", args)
	}
	if strings.Contains(args, "x11grab") {
		t.Errorf("expected no x11grab for audio only, got: %s", args)
	}
	if !strings.Contains(args, "-c:a aac") {
		t.Errorf("expected aac audio encoding, got: %s", args)
	}
}

func TestBuildFFmpegArgs_WindowCrop(t *testing.T) {
	opts := FFmpegOptions{
		Display:  ":20",
		FPS:      30,
		HWAccel:  "none",
		CaptureW: 800,
		CaptureH: 600,
		OffsetX:  100,
		OffsetY:  50,
	}
	args := strings.Join(BuildFFmpegArgs(opts), " ")
	if !strings.Contains(args, "-video_size 800x600") {
		t.Errorf("expected -video_size 800x600, got: %s", args)
	}
	if !strings.Contains(args, ":20.0+100,50") {
		t.Errorf("expected input :20.0+100,50, got: %s", args)
	}
}
