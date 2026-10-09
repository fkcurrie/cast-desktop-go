package stream

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// FFmpegOptions holds parameters for screen capture and streaming.
type FFmpegOptions struct {
	Display     string
	FPS         int
	Resolution  string // "1080p", "720p", "native"
	Fit         string // "pad" (default), "crop", "stretch"
	Preset      string // "balanced" (default), "ultra-low", "cinema"
	HWAccel     string // "auto" (default), "vaapi", "nvenc", "none"
	AudioOnly   bool   // Stream audio only (for smart speakers)
	AudioSource string // Pulse/PipeWire audio source (default "default")
	CaptureW    int    // Specific width for monitor or window capture (0 = full display)
	CaptureH    int    // Specific height for monitor or window capture
	OffsetX     int    // Left X coordinate offset
	OffsetY     int    // Top Y coordinate offset
}

// HasVAAPI checks if /dev/dri/renderD128 is available for hardware encoding.
func HasVAAPI() bool {
	info, err := os.Stat("/dev/dri/renderD128")
	return err == nil && (info.Mode()&os.ModeDevice != 0)
}

// ResolveHWAccel chooses the optimal encoding acceleration engine.
func ResolveHWAccel(requested string) string {
	switch strings.ToLower(requested) {
	case "vaapi":
		return "vaapi"
	case "nvenc":
		return "nvenc"
	case "none", "cpu":
		return "none"
	case "auto", "":
		if HasVAAPI() {
			return "vaapi"
		}
		return "none"
	default:
		return "none"
	}
}

// DetectDisplayResolution queries xwininfo to determine root window size.
func DetectDisplayResolution(display string) (int, int) {
	out, err := exec.Command("xwininfo", "-root", "-display", display).Output()
	if err == nil {
		s := string(out)
		wRe := regexp.MustCompile(`Width:\s+(\d+)`)
		hRe := regexp.MustCompile(`Height:\s+(\d+)`)
		wMatch := wRe.FindStringSubmatch(s)
		hMatch := hRe.FindStringSubmatch(s)
		if len(wMatch) == 2 && len(hMatch) == 2 {
			w, errW := strconv.Atoi(wMatch[1])
			h, errH := strconv.Atoi(hMatch[1])
			if errW == nil && errH == nil && w > 0 && h > 0 {
				return w, h
			}
		}
	}
	return 3840, 2160 // Default fallback for current host
}

func buildVideoFilter(opts FFmpegOptions, origW, origH int, useVAAPI bool) string {
	formatSuffix := "format=yuv420p"
	if useVAAPI {
		formatSuffix = "format=nv12,hwupload"
	}

	if opts.Resolution == "native" {
		return formatSuffix
	}

	targetW, targetH := 1920, 1080
	if opts.Resolution == "720p" {
		targetW, targetH = 1280, 720
	}

	switch opts.Fit {
	case "stretch":
		return fmt.Sprintf("scale=%d:%d,%s", targetW, targetH, formatSuffix)
	case "crop":
		// Scale and crop to fill the 16:9 frame without black bars
		if origW*targetH < origH*targetW {
			// Source is taller than target (e.g. 3:2 vs 16:9): crop top and bottom
			cropH := (origW * targetH) / targetW
			return fmt.Sprintf("crop=%d:%d:0:(in_h-out_h)/2,scale=%d:%d,%s", origW, cropH, targetW, targetH, formatSuffix)
		}
		// Source is wider than target: crop left and right
		cropW := (origH * targetW) / targetH
		return fmt.Sprintf("crop=%d:%d:(in_w-out_w)/2:0,scale=%d:%d,%s", cropW, origH, targetW, targetH, formatSuffix)
	default: // "pad"
		return fmt.Sprintf(
			"scale=w='min(%d,trunc(%d*%d/%d/2)*2)':h='min(%d,trunc(%d*%d/%d/2)*2)',pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black,%s",
			targetW, origW, targetH, origH,
			targetH, origH, targetW, origW,
			targetW, targetH,
			formatSuffix,
		)
	}
}

// BuildFFmpegArgs generates command-line arguments for FFmpeg with hardware acceleration and latency presets.
func BuildFFmpegArgs(opts FFmpegOptions) []string {
	playlistPath := filepath.Join(StreamDir, "live.m3u8")

	audioSrc := opts.AudioSource
	if audioSrc == "" {
		audioSrc = "default"
	}

	// 1. Audio-Only Mode
	if opts.AudioOnly {
		return []string{
			"-y", "-nostdin", "-loglevel", "warning",
			"-f", "pulse", "-i", audioSrc,
			"-vn",
			"-c:a", "aac", "-b:a", "192k", "-ar", "44100",
			"-f", "hls",
			"-hls_time", "1",
			"-hls_list_size", "5",
			"-hls_flags", "delete_segments+split_by_time",
			"-hls_segment_type", "fmp4",
			"-hls_fmp4_init_filename", "init.mp4",
			playlistPath,
		}
	}

	// 2. Video Dimensions
	origW, origH := opts.CaptureW, opts.CaptureH
	if origW <= 0 || origH <= 0 {
		origW, origH = DetectDisplayResolution(opts.Display)
	}

	inputSpec := fmt.Sprintf("%s.0+%d,%d", opts.Display, opts.OffsetX, opts.OffsetY)
	fpsStr := strconv.Itoa(opts.FPS)

	// 3. Latency Preset Configuration
	hlsTime := "1"
	hlsListSize := "5"
	gopSize := opts.FPS
	videoBitrate := "4500k"
	audioBitrate := "128k"

	switch opts.Preset {
	case "ultra-low":
		hlsTime = "0.5"
		hlsListSize = "4"
		gopSize = opts.FPS / 2
		if gopSize < 10 {
			gopSize = 15
		}
		videoBitrate = "4000k"
	case "cinema":
		hlsTime = "2"
		hlsListSize = "6"
		gopSize = opts.FPS * 2
		videoBitrate = "7500k"
		audioBitrate = "256k"
	default: // balanced
		hlsTime = "1"
		hlsListSize = "5"
		gopSize = opts.FPS
		videoBitrate = "4500k"
		audioBitrate = "128k"
	}

	gopStr := strconv.Itoa(gopSize)
	accel := ResolveHWAccel(opts.HWAccel)

	// 4. Hardware-Accelerated VA-API
	if accel == "vaapi" {
		vfFilter := buildVideoFilter(opts, origW, origH, true)
		return []string{
			"-y", "-nostdin", "-loglevel", "warning",
			"-vaapi_device", "/dev/dri/renderD128",
			"-f", "x11grab",
			"-draw_mouse", "1",
			"-video_size", fmt.Sprintf("%dx%d", origW, origH),
			"-framerate", fpsStr,
			"-i", inputSpec,
			"-f", "pulse",
			"-i", audioSrc,
			"-vf", vfFilter,
			"-c:v", "h264_vaapi",
			"-b:v", videoBitrate,
			"-g", gopStr,
			"-keyint_min", gopStr,
			"-c:a", "aac",
			"-b:a", audioBitrate,
			"-ar", "44100",
			"-f", "hls",
			"-hls_time", hlsTime,
			"-hls_list_size", hlsListSize,
			"-hls_flags", "delete_segments+split_by_time",
			"-hls_segment_type", "fmp4",
			"-hls_fmp4_init_filename", "init.mp4",
			playlistPath,
		}
	}

	// 5. Software CPU Fallback (libx264)
	vfFilter := buildVideoFilter(opts, origW, origH, false)
	cpuPreset := "ultrafast"
	if opts.Preset == "cinema" {
		cpuPreset = "fast"
	}

	return []string{
		"-y", "-nostdin", "-loglevel", "warning",
		"-f", "x11grab",
		"-draw_mouse", "1",
		"-video_size", fmt.Sprintf("%dx%d", origW, origH),
		"-framerate", fpsStr,
		"-i", inputSpec,
		"-f", "pulse",
		"-i", audioSrc,
		"-vf", vfFilter,
		"-c:v", "libx264",
		"-preset", cpuPreset,
		"-tune", "zerolatency",
		"-profile:v", "high",
		"-level", "4.1",
		"-pix_fmt", "yuv420p",
		"-g", gopStr,
		"-keyint_min", gopStr,
		"-sc_threshold", "0",
		"-b:v", videoBitrate,
		"-c:a", "aac",
		"-b:a", audioBitrate,
		"-ar", "44100",
		"-f", "hls",
		"-hls_time", hlsTime,
		"-hls_list_size", hlsListSize,
		"-hls_flags", "delete_segments+split_by_time",
		"-hls_segment_type", "fmp4",
		"-hls_fmp4_init_filename", "init.mp4",
		playlistPath,
	}
}

// StartFFmpeg launches the FFmpeg capture process and waits for the first segment.
func StartFFmpeg(opts FFmpegOptions, logFilePath string) (*exec.Cmd, error) {
	args := BuildFFmpegArgs(opts)
	cmd := exec.Command("ffmpeg", args...)

	if logFilePath != "" {
		_ = os.MkdirAll(filepath.Dir(logFilePath), 0755)
		logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err == nil {
			cmd.Stdout = logFile
			cmd.Stderr = logFile
		}
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Wait for init.mp4 and first segment to be generated
	initMp4 := filepath.Join(StreamDir, "init.mp4")
	m3u8Path := filepath.Join(StreamDir, "live.m3u8")
	startWait := time.Now()
	ready := false

	for time.Since(startWait) < 10*time.Second {
		infoInit, errInit := os.Stat(initMp4)
		infoM3u8, errM3u8 := os.Stat(m3u8Path)
		if errInit == nil && infoInit.Size() > 0 && errM3u8 == nil && infoM3u8.Size() > 0 {
			matches, _ := filepath.Glob(filepath.Join(StreamDir, "*.m4s"))
			if len(matches) > 0 {
				ready = true
				break
			}
		}
		time.Sleep(200 * time.Millisecond)
	}

	if !ready {
		_ = StopFFmpeg(cmd)
		return nil, fmt.Errorf("timeout waiting for FFmpeg stream generation (check %s)", logFilePath)
	}

	return cmd, nil
}

// StopFFmpeg terminates the FFmpeg process group gracefully.
func StopFFmpeg(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
		time.Sleep(500 * time.Millisecond)
		if IsProcessRunning(cmd.Process.Pid) {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		}
	} else {
		_ = cmd.Process.Kill()
	}
	return nil
}
