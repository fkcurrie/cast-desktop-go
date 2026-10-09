package stream

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

// Overlay manages the visual screen border indicating active broadcast.
type Overlay struct {
	cmd *exec.Cmd
}

// StartPrivacyOverlay launches the click-through border overlay process.
func StartPrivacyOverlay(display string, x, y, w, h int) *Overlay {
	var scriptPaths []string

	// Check standard distribution installation path first
	scriptPaths = append(scriptPaths,
		"/usr/share/cast-desktop-go/border_overlay.py",
		"/usr/local/share/cast-desktop-go/border_overlay.py",
	)

	// Check relative to current executable
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		scriptPaths = append(scriptPaths,
			filepath.Join(exeDir, "..", "share", "cast-desktop-go", "border_overlay.py"),
			filepath.Join(exeDir, "..", "scripts", "border_overlay.py"),
			filepath.Join(exeDir, "scripts", "border_overlay.py"),
		)
	}

	// Check relative to current working directory
	scriptPaths = append(scriptPaths, "scripts/border_overlay.py")

	var scriptPath string
	for _, p := range scriptPaths {
		if _, err := os.Stat(p); err == nil {
			scriptPath = p
			break
		}
	}

	if scriptPath == "" {
		return nil
	}

	args := []string{
		scriptPath,
		"--x", strconv.Itoa(x),
		"--y", strconv.Itoa(y),
		"--width", strconv.Itoa(w),
		"--height", strconv.Itoa(h),
	}

	cmd := exec.Command("python3", args...)
	cmd.Env = append(os.Environ(), "DISPLAY="+display)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return nil
	}

	return &Overlay{cmd: cmd}
}

// Stop cleanly terminates the overlay process.
func (o *Overlay) Stop() {
	if o == nil || o.cmd == nil || o.cmd.Process == nil {
		return
	}
	pgid, err := syscall.Getpgid(o.cmd.Process.Pid)
	if err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
	} else {
		_ = o.cmd.Process.Kill()
	}
}
