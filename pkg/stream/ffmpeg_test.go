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
	}
	argsStretch := strings.Join(BuildFFmpegArgs(optsStretch), " ")
	if !strings.Contains(argsStretch, "scale=1920:1080") {
		t.Errorf("expected scale=1920:1080 in stretch mode, got: %s", argsStretch)
	}
}
