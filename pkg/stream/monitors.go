package stream

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// MonitorInfo represents an active physical or virtual display monitor.
type MonitorInfo struct {
	Index   int    `json:"index"`
	Name    string `json:"name"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Primary bool   `json:"primary"`
}

// WindowInfo represents an open top-level viewable application window.
type WindowInfo struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Class  string `json:"class,omitempty"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
}

// DetectMonitors queries xrandr to list all currently connected monitors.
func DetectMonitors(display string) ([]MonitorInfo, error) {
	cmd := exec.Command("xrandr", "-display", display, "--current")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run xrandr: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	// Example: DUMMY0 connected primary 3840x2160+0+0 412mm x 230mm
	// Or:      HDMI-1 connected 1920x1080+3840+0 ...
	re := regexp.MustCompile(`^(\S+)\s+connected\s+(primary\s+)?(\d+)x(\d+)\+(\d+)\+(\d+)`)

	var monitors []MonitorInfo
	idx := 1
	for _, l := range lines {
		m := re.FindStringSubmatch(strings.TrimSpace(l))
		if len(m) == 7 {
			w, _ := strconv.Atoi(m[3])
			h, _ := strconv.Atoi(m[4])
			x, _ := strconv.Atoi(m[5])
			y, _ := strconv.Atoi(m[6])
			isPrimary := strings.TrimSpace(m[2]) == "primary"

			mon := MonitorInfo{
				Index:   idx,
				Name:    m[1],
				Width:   w,
				Height:  h,
				X:       x,
				Y:       y,
				Primary: isPrimary,
			}
			monitors = append(monitors, mon)
			idx++
		}
	}

	if len(monitors) == 0 {
		// Fallback to single full-screen virtual monitor
		w, h := DetectDisplayResolution(display)
		monitors = append(monitors, MonitorInfo{
			Index:   1,
			Name:    "Default Screen",
			Width:   w,
			Height:  h,
			X:       0,
			Y:       0,
			Primary: true,
		})
	}

	return monitors, nil
}

// DetectWindows queries X11 net client list and returns viewable application windows.
func DetectWindows(display string) ([]WindowInfo, error) {
	cmd := exec.Command("xprop", "-display", display, "-root", "_NET_CLIENT_LIST")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query _NET_CLIENT_LIST: %w", err)
	}

	s := string(out)
	idx := strings.Index(s, "#")
	if idx == -1 {
		return nil, nil
	}

	rawIDs := strings.Split(s[idx+1:], ",")
	var windows []WindowInfo

	for _, rawID := range rawIDs {
		winID := strings.TrimSpace(rawID)
		if winID == "" || winID == "0x0" {
			continue
		}

		info, err := GetWindowInfo(display, winID)
		if err == nil && info != nil && info.Width > 50 && info.Height > 50 {
			windows = append(windows, *info)
		}
	}

	return windows, nil
}

var hexIDRe = regexp.MustCompile(`^0x[0-9a-fA-F]+$`)

// GetWindowInfo extracts window title, state, and coordinates using xwininfo.
func GetWindowInfo(display string, winID string) (*WindowInfo, error) {
	winID = strings.TrimSpace(winID)
	if !hexIDRe.MatchString(winID) {
		return nil, fmt.Errorf("invalid window hex ID format: %s", winID)
	}

	cmd := exec.Command("xwininfo", "-display", display, "-id", winID)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	s := string(out)
	if !strings.Contains(s, "Map State: IsViewable") {
		return nil, fmt.Errorf("window is not viewable")
	}

	titleRe := regexp.MustCompile(`xwininfo: Window id: 0x[0-9a-fA-F]+\s+"([^"]+)"`)
	titleMatch := titleRe.FindStringSubmatch(s)
	title := "Window " + winID
	if len(titleMatch) == 2 {
		title = titleMatch[1]
	}

	// Filter out desktop lockers, docks, or system panels if unnamed
	if title == "" || strings.HasPrefix(title, "Desktop —") || title == "Plasma" {
		return nil, fmt.Errorf("filtered desktop component")
	}

	wRe := regexp.MustCompile(`Width:\s+(\d+)`)
	hRe := regexp.MustCompile(`Height:\s+(\d+)`)
	xRe := regexp.MustCompile(`Absolute upper-left X:\s+(-?\d+)`)
	yRe := regexp.MustCompile(`Absolute upper-left Y:\s+(-?\d+)`)

	wM := wRe.FindStringSubmatch(s)
	hM := hRe.FindStringSubmatch(s)
	xM := xRe.FindStringSubmatch(s)
	yM := yRe.FindStringSubmatch(s)

	if len(wM) != 2 || len(hM) != 2 || len(xM) != 2 || len(yM) != 2 {
		return nil, fmt.Errorf("could not parse window dimensions")
	}

	w, _ := strconv.Atoi(wM[1])
	h, _ := strconv.Atoi(hM[1])
	x, _ := strconv.Atoi(xM[1])
	y, _ := strconv.Atoi(yM[1])

	// Ensure X and Y are non-negative for capture offset
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	return &WindowInfo{
		ID:     winID,
		Title:  title,
		Width:  w,
		Height: h,
		X:      x,
		Y:      y,
	}, nil
}

// ResolveMonitor finds a monitor by 1-based index or output name.
func ResolveMonitor(display string, target string) (*MonitorInfo, error) {
	monitors, err := DetectMonitors(display)
	if err != nil {
		return nil, err
	}

	target = strings.TrimSpace(target)
	if idx, err := strconv.Atoi(target); err == nil && idx >= 1 && idx <= len(monitors) {
		return &monitors[idx-1], nil
	}

	for _, m := range monitors {
		if strings.EqualFold(m.Name, target) {
			return &m, nil
		}
	}

	for _, m := range monitors {
		if strings.Contains(strings.ToLower(m.Name), strings.ToLower(target)) {
			return &m, nil
		}
	}

	return nil, fmt.Errorf("monitor '%s' not found (available: %d monitors)", target, len(monitors))
}

// ResolveWindow finds a window by hex ID or title substring.
func ResolveWindow(display string, target string) (*WindowInfo, error) {
	target = strings.TrimSpace(target)
	if strings.HasPrefix(target, "0x") {
		return GetWindowInfo(display, target)
	}

	windows, err := DetectWindows(display)
	if err != nil {
		return nil, err
	}

	for _, w := range windows {
		if strings.Contains(strings.ToLower(w.Title), strings.ToLower(target)) {
			return &w, nil
		}
	}

	return nil, fmt.Errorf("window matching '%s' not found", target)
}

