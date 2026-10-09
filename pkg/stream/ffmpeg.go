package stream

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
	"time"
)

// FFmpegOptions holds parameters for screen capture.
type FFmpegOptions struct {
	Display    string
	FPS        int
	Resolution string // "1080p", "720p", "native"
	Fit        string // "pad" (default), "crop", "stretch"
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

func buildVideoFilter(opts FFmpegOptions, origW, origH int) string {
	if opts.Resolution == "native" {
		return "format=yuv420p"
	}

	targetW, targetH := 1920, 1080
	if opts.Resolution == "720p" {
		targetW, targetH = 1280, 720
	}

	switch opts.Fit {
	case "stretch":
		return fmt.Sprintf("scale=%d:%d,format=yuv420p", targetW, targetH)
	case "crop":
		// Scale and crop to fill the 16:9 frame without black bars
		if origW*targetH < origH*targetW {
			// Source is taller than target (e.g. 3:2 vs 16:9): crop top and bottom
			cropH := (origW * targetH) / targetW
			return fmt.Sprintf("crop=%d:%d:0:(in_h-out_h)/2,scale=%d:%d,format=yuv420p", origW, cropH, targetW, targetH)
		}
		// Source is wider than target: crop left and right
		cropW := (origH * targetW) / targetH
		return fmt.Sprintf("crop=%d:%d:(in_w-out_w)/2:0,scale=%d:%d,format=yuv420p", cropW, origH, targetW, targetH)
	default: // "pad"
		return fmt.Sprintf(
			"scale=w='min(%d,trunc(%d*%d/%d/2)*2)':h='min(%d,trunc(%d*%d/%d/2)*2)',pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black,format=yuv420p",
			targetW, origW, targetH, origH,
			targetH, origH, targetW, origW,
			targetW, targetH,
		)
	}
}

// BuildFFmpegArgs generates command-line arguments for FFmpeg.
func BuildFFmpegArgs(opts FFmpegOptions) []string {
	origW, origH := DetectDisplayResolution(opts.Display)
	playlistPath := filepath.Join(StreamDir, "live.m3u8")

	vfFilter := buildVideoFilter(opts, origW, origH)

	fpsStr := strconv.Itoa(opts.FPS)

	return []string{
		"-y", "-nostdin", "-loglevel", "warning",
		// Video grab
		"-f", "x11grab",
		"-draw_mouse", "1",
		"-video_size", fmt.Sprintf("%dx%d", origW, origH),
		"-framerate", fpsStr,
		"-i", fmt.Sprintf("%s.0", opts.Display),
		// Audio grab (PipeWire / Pulse monitor)
		"-f", "pulse",
		"-i", "default",
		// Video filters & encoding
		"-vf", vfFilter,
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-profile:v", "high",
		"-level", "4.1",
		"-pix_fmt", "yuv420p",
		"-g", fpsStr,
		"-keyint_min", fpsStr,
		"-sc_threshold", "0",
		"-b:v", "4500k",
		"-maxrate", "6000k",
		"-bufsize", "3000k",
		// Audio encoding
		"-c:a", "aac",
		"-b:a", "128k",
		"-ar", "44100",
		// HLS muxer (fMP4 CMAF for modern Chromecast Shaka player)
		"-f", "hls",
		"-hls_time", "1",
		"-hls_list_size", "5",
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

	// Setup log output
	if logFilePath != "" {
		_ = os.MkdirAll(filepath.Dir(logFilePath), 0755)
		logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err == nil {
			cmd.Stdout = logFile
			cmd.Stderr = logFile
		}
	}

	// Create new process group
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
			// Check if at least one .m4s segment exists
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
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	} else {
		_ = cmd.Process.Kill()
	}
	return nil
}
