package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"cast-desktop-go/pkg/cast"
	"cast-desktop-go/pkg/stream"
	"cast-desktop-go/pkg/tray"
	"github.com/mattn/go-isatty"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/spf13/cobra"
)

var (
	flagDevice     string
	flagDisplay    string
	flagFPS        int
	flagResolution string
	flagFit        string
	flagPort       int
	flagDaemon     bool
	flagPreset     string
	flagHWAccel    string
	flagMonitor    string
	flagWindow     string
	flagAudioOnly  bool
	flagBorder     bool
	flagNoBorder   bool

	// Worker internal flags
	workerDeviceIP   string
	workerDeviceName string
	workerDevicePort int
	workerDisplay    string
	workerFPS        int
	workerRes        string
	workerFit        string
	workerPort       int
	workerPreset     string
	workerHWAccel    string
	workerMonitor    string
	workerWindow     string
	workerAudioOnly  bool
	workerBorder     bool
)

func addStartFlags(c *cobra.Command) {
	c.Flags().StringVarP(&flagDevice, "device", "d", "", "Target device name, IP address, or index (interactive picker if omitted)")
	c.Flags().StringVar(&flagDisplay, "display", "", "X11 display to capture (default :20 or $DISPLAY)")
	c.Flags().IntVar(&flagFPS, "fps", 30, "Capture framerate")
	c.Flags().StringVar(&flagResolution, "res", "1080p", "Output resolution (1080p, 720p, native)")
	c.Flags().StringVar(&flagFit, "fit", "pad", "Fit mode: pad (preserve aspect ratio with bars), crop (zoom to fill 16:9), stretch")
	c.Flags().IntVarP(&flagPort, "port", "p", 8088, "HTTP streaming server port")
	c.Flags().BoolVarP(&flagDaemon, "daemon", "b", false, "Run cast session in the background")
	c.Flags().StringVar(&flagPreset, "preset", "balanced", "Latency preset: ultra-low (0.5s), balanced (1.0s), cinema (2.0s)")
	c.Flags().StringVar(&flagHWAccel, "hw-accel", "auto", "Hardware acceleration: auto (AMD VA-API GPU), vaapi, software")
	c.Flags().StringVarP(&flagMonitor, "monitor", "m", "", "Cast specific monitor (name or index, e.g. HDMI-A-0 or 0)")
	c.Flags().StringVarP(&flagWindow, "window", "w", "", "Cast specific application window by title or hex ID")
	c.Flags().BoolVar(&flagAudioOnly, "audio-only", false, "Stream audio only (ideal for smart speakers / Google Home)")
	c.Flags().BoolVar(&flagBorder, "border", true, "Show visual glowing border overlay around captured screen area")
	c.Flags().BoolVar(&flagNoBorder, "no-border", false, "Disable visual glowing border overlay")
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start casting the desktop to a Chromecast",
	RunE:  runStart,
}

var workerCmd = &cobra.Command{
	Use:    "_worker",
	Hidden: true,
	RunE:   runWorker,
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(workerCmd)

	addStartFlags(startCmd)

	workerCmd.Flags().StringVar(&workerDeviceIP, "device-ip", "", "")
	workerCmd.Flags().StringVar(&workerDeviceName, "device-name", "", "")
	workerCmd.Flags().IntVar(&workerDevicePort, "device-port", 8009, "")
	workerCmd.Flags().StringVar(&workerDisplay, "display", "", "")
	workerCmd.Flags().IntVar(&workerFPS, "fps", 30, "")
	workerCmd.Flags().StringVar(&workerRes, "res", "1080p", "")
	workerCmd.Flags().StringVar(&workerFit, "fit", "pad", "")
	workerCmd.Flags().IntVar(&workerPort, "port", 8088, "")
	workerCmd.Flags().StringVar(&workerPreset, "preset", "balanced", "")
	workerCmd.Flags().StringVar(&workerHWAccel, "hw-accel", "auto", "")
	workerCmd.Flags().StringVar(&workerMonitor, "monitor", "", "")
	workerCmd.Flags().StringVar(&workerWindow, "window", "", "")
	workerCmd.Flags().BoolVar(&workerAudioOnly, "audio-only", false, "")
	workerCmd.Flags().BoolVar(&workerBorder, "border", true, "")
}

func selectDeviceInteractive(devices []cast.Device, defaultDevice string) *cast.Device {
	if len(devices) == 1 {
		return &devices[0]
	}

	defaultIdx := 1
	fmt.Printf("\nDiscovered %d Chromecast devices:\n", len(devices))
	for i, d := range devices {
		suffix := ""
		if defaultDevice != "" && strings.EqualFold(d.Name, defaultDevice) {
			defaultIdx = i + 1
			suffix = " (saved default)"
		}
		fmt.Printf("  [%d] %s (%s)%s\n", i+1, d.Name, d.IP, suffix)
	}

	if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		fmt.Printf("Non-interactive terminal detected; defaulting to [%d] %s\n", defaultIdx, devices[defaultIdx-1].Name)
		return &devices[defaultIdx-1]
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("\nSelect a device [1-%d] (default: %d): ", len(devices), defaultIdx)
		input, err := reader.ReadString('\n')
		if err != nil {
			return &devices[defaultIdx-1]
		}
		input = strings.TrimSpace(input)
		if input == "" {
			return &devices[defaultIdx-1]
		}

		dev, err := cast.ResolveDevice(input, devices)
		if err == nil {
			return dev
		}
		fmt.Printf("Invalid selection: %v. Please try again.\n", err)
	}
}

// CastOptions encapsulates all configuration flags for launching a cast session.
type CastOptions struct {
	TargetDev  *cast.Device
	Display    string
	FPS        int
	Resolution string
	Fit        string
	Port       int
	Preset     string
	HWAccel    string
	Monitor    string
	Window     string
	AudioOnly  bool
	Border     bool
}

func runStart(cmd *cobra.Command, args []string) error {
	// 1. Ensure any previous active session is stopped
	_ = stopActiveSession()

	cfg := stream.LoadConfig()

	// Apply persisted settings if user did not pass explicit flags
	if !cmd.Flags().Changed("res") && cfg.Resolution != "" {
		flagResolution = cfg.Resolution
	}
	if !cmd.Flags().Changed("fit") && cfg.Fit != "" {
		flagFit = cfg.Fit
	}
	if !cmd.Flags().Changed("fps") && cfg.FPS > 0 {
		flagFPS = cfg.FPS
	}
	if !cmd.Flags().Changed("display") && cfg.Display != "" {
		flagDisplay = cfg.Display
	}
	if !cmd.Flags().Changed("port") && cfg.Port > 0 {
		flagPort = cfg.Port
	}
	if !cmd.Flags().Changed("preset") && cfg.Preset != "" {
		flagPreset = cfg.Preset
	}
	if !cmd.Flags().Changed("hw-accel") && cfg.HWAccel != "" {
		flagHWAccel = cfg.HWAccel
	}
	if !cmd.Flags().Changed("monitor") && cfg.Monitor != "" {
		flagMonitor = cfg.Monitor
	}

	showBorder := flagBorder
	if flagNoBorder || flagAudioOnly {
		showBorder = false
	}

	// 2. Discover devices
	fmt.Println("Discovering Chromecast devices...")
	devices := cast.DiscoverDevices(true)
	if len(devices) == 0 {
		return fmt.Errorf("no Chromecast devices discovered on the local network")
	}

	// 3. Resolve target device
	var targetDev *cast.Device
	if flagDevice != "" {
		var err error
		targetDev, err = cast.ResolveDevice(flagDevice, devices)
		if err != nil {
			return err
		}
	} else if flagDaemon && cfg.DefaultDevice != "" {
		dev, err := cast.ResolveDevice(cfg.DefaultDevice, devices)
		if err == nil {
			targetDev = dev
		} else {
			targetDev = selectDeviceInteractive(devices, cfg.DefaultDevice)
		}
	} else {
		targetDev = selectDeviceInteractive(devices, cfg.DefaultDevice)
	}

	display := flagDisplay
	if display == "" {
		display = os.Getenv("DISPLAY")
		if display == "" {
			display = ":20"
		}
	}

	// Automatically ensure 16:9 mode if display :20 is detected in 3:2 (3840x2560)
	if !flagAudioOnly && flagWindow == "" && flagMonitor == "" {
		origW, origH := stream.DetectDisplayResolution(display)
		if origW == 3840 && origH == 2560 {
			fmt.Println("Detected 3:2 desktop resolution (3840x2560); switching display to 16:9 4K (3840x2160)...")
			_ = exec.Command("xrandr", "-display", display, "--output", "DUMMY0", "--mode", "3840x2160_60.00").Run()
		}
	}

	// Remember settings for next session
	cfg.DefaultDevice = targetDev.Name
	cfg.Resolution = flagResolution
	cfg.Fit = flagFit
	cfg.FPS = flagFPS
	cfg.Display = display
	cfg.Port = flagPort
	cfg.Preset = flagPreset
	cfg.HWAccel = flagHWAccel
	cfg.Border = showBorder
	cfg.AudioOnly = flagAudioOnly
	cfg.Monitor = flagMonitor
	_ = stream.SaveConfig(cfg)

	opts := CastOptions{
		TargetDev:  targetDev,
		Display:    display,
		FPS:        flagFPS,
		Resolution: flagResolution,
		Fit:        flagFit,
		Port:       flagPort,
		Preset:     flagPreset,
		HWAccel:    flagHWAccel,
		Monitor:    flagMonitor,
		Window:     flagWindow,
		AudioOnly:  flagAudioOnly,
		Border:     showBorder,
	}

	fmt.Printf("\nPreparing to cast to: %s (%s)\n", targetDev.Name, targetDev.IP)
	if opts.AudioOnly {
		fmt.Printf("Mode: Audio-Only | Latency Preset: %s | Port: %d\n", opts.Preset, opts.Port)
	} else {
		fmt.Printf("Display: %s | Latency Preset: %s | Framerate: %d fps | Resolution: %s\n", display, opts.Preset, opts.FPS, opts.Resolution)
		if opts.Monitor != "" {
			fmt.Printf("Monitor Target: %s\n", opts.Monitor)
		}
		if opts.Window != "" {
			fmt.Printf("Window Target: %s\n", opts.Window)
		}
	}

	localIP, err := cast.GetRouteToTarget(targetDev.IP)
	if err != nil {
		return err
	}
	streamURL := fmt.Sprintf("http://%s:%d/live.m3u8", localIP, flagPort)
	remoteURL := fmt.Sprintf("http://%s:%d/remote", localIP, flagPort)

	fmt.Printf("Local stream endpoint: %s\n", streamURL)
	fmt.Printf("Mobile Web Remote:     %s\n", remoteURL)

	// Display Terminal ANSI QR Code for rapid mobile pairing
	if qr, err := qrcode.New(remoteURL, qrcode.Medium); err == nil {
		fmt.Printf("\nScan to open Web Remote Dashboard on phone / tablet:\n%s\n", qr.ToSmallString(false))
	}

	if flagDaemon {
		fmt.Println("Starting desktop casting in background...")
		st, err := StartBackgroundWorker(opts)
		if err != nil {
			return err
		}
		fmt.Printf("\nDesktop casting successfully started in background!\n")
		fmt.Printf("  Target:     %s (%s)\n", targetDev.Name, targetDev.IP)
		fmt.Printf("  Worker PID: %d\n", st.PID)
		fmt.Printf("  Web Remote: %s\n", remoteURL)
		fmt.Printf("  Run 'cast-desktop-go status' to inspect.\n")
		fmt.Printf("  Run 'cast-desktop-go stop' to end session.\n\n")
		return nil
	}

	// Foreground mode
	fmt.Println("\nStarting casting session in foreground (press Ctrl+C to stop)...")
	return executeCastLoop(opts, streamURL, localIP)
}

func runWorker(cmd *cobra.Command, args []string) error {
	targetDev := &cast.Device{
		Name: workerDeviceName,
		IP:   workerDeviceIP,
		Port: workerDevicePort,
	}

	display := workerDisplay
	if display == "" {
		display = ":20"
	}

	localIP, err := cast.GetRouteToTarget(targetDev.IP)
	if err != nil {
		return err
	}
	streamURL := fmt.Sprintf("http://%s:%d/live.m3u8", localIP, workerPort)

	opts := CastOptions{
		TargetDev:  targetDev,
		Display:    display,
		FPS:        workerFPS,
		Resolution: workerRes,
		Fit:        workerFit,
		Port:       workerPort,
		Preset:     workerPreset,
		HWAccel:    workerHWAccel,
		Monitor:    workerMonitor,
		Window:     workerWindow,
		AudioOnly:  workerAudioOnly,
		Border:     workerBorder,
	}

	return executeCastLoop(opts, streamURL, localIP)
}

func executeCastLoop(opts CastOptions, streamURL string, localIP string) error {
	targetDev := opts.TargetDev
	display := opts.Display
	port := opts.Port

	// 1. Resolve crop coordinates for Monitor or Window
	var cropX, cropY, cropW, cropH int
	if opts.Window != "" {
		win, err := stream.ResolveWindow(display, opts.Window)
		if err != nil {
			return fmt.Errorf("failed to locate window '%s': %w", opts.Window, err)
		}
		cropX, cropY, cropW, cropH = win.X, win.Y, win.Width, win.Height
		fmt.Printf("Casting Window: '%s' (%dx%d at +%d+%d)\n", win.Title, cropW, cropH, cropX, cropY)
	} else if opts.Monitor != "" {
		mon, err := stream.ResolveMonitor(display, opts.Monitor)
		if err != nil {
			return fmt.Errorf("failed to locate monitor '%s': %w", opts.Monitor, err)
		}
		cropX, cropY, cropW, cropH = mon.X, mon.Y, mon.Width, mon.Height
		fmt.Printf("Casting Monitor: '%s' (%dx%d at +%d+%d)\n", mon.Name, cropW, cropH, cropX, cropY)
	}

	// 2. Start Visual Privacy Border Overlay if enabled
	var overlay *stream.Overlay
	if opts.Border && !opts.AudioOnly {
		ox, oy, ow, oh := cropX, cropY, cropW, cropH
		if ow == 0 && oh == 0 {
			ow, oh = stream.DetectDisplayResolution(display)
		}
		overlay = stream.StartPrivacyOverlay(display, ox, oy, ow, oh)
		if overlay != nil {
			defer overlay.Stop()
		}
	}

	// 3. Start HTTP Server with Web Remote
	srv, err := stream.NewServer(port, localIP)
	if err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	defer srv.Stop()

	// 4. Start FFmpeg capture
	ffmpegLog := filepath.Join(os.Getenv("HOME"), ".config", "cast-desktop-go", "ffmpeg.log")
	ffOpts := stream.FFmpegOptions{
		Display:     display,
		FPS:         opts.FPS,
		Resolution:  opts.Resolution,
		Fit:         opts.Fit,
		Preset:      opts.Preset,
		HWAccel:     opts.HWAccel,
		AudioOnly:   opts.AudioOnly,
		AudioSource: "default",
		CaptureW:    cropW,
		CaptureH:    cropH,
		OffsetX:     cropX,
		OffsetY:     cropY,
	}

	ffmpegCmd, err := stream.StartFFmpeg(ffOpts, ffmpegLog)
	if err != nil {
		return err
	}
	defer stream.StopFFmpeg(ffmpegCmd)

	// 5. Connect to Chromecast and begin playback
	ctrl := cast.NewController(targetDev)
	if err := ctrl.StartCast(streamURL); err != nil {
		return fmt.Errorf("failed to launch Chromecast stream: %w", err)
	}
	defer ctrl.Close()

	// 6. Wire Web Remote & REST API Callbacks
	startTime := time.Now()
	remoteURL := fmt.Sprintf("http://%s:%d/remote", localIP, port)
	srv.SetRemoteHandler(&stream.RemoteHandler{
		GetStatus: func() map[string]interface{} {
			vol, muted, _ := ctrl.GetVolume()
			return map[string]interface{}{
				"active":     true,
				"device":     targetDev.Name,
				"preset":     opts.Preset,
				"hw_accel":   stream.HasVAAPI() && opts.HWAccel != "software",
				"audio_only": opts.AudioOnly,
				"resolution": opts.Resolution,
				"fps":        opts.FPS,
				"volume":     vol,
				"muted":      muted,
				"paused":     false,
				"uptime":     time.Since(startTime).Truncate(time.Second).String(),
			}
		},
		SetVolume: func(vol float64) error {
			return ctrl.SetVolume(float32(vol / 100.0))
		},
		SetMuted: func(muted bool) error {
			return ctrl.SetMuted(muted)
		},
		ToggleMute: func() (bool, error) {
			return ctrl.ToggleMute()
		},
		Pause: func() error {
			return ctrl.Pause()
		},
		Play: func() error {
			return ctrl.Unpause()
		},
		Stop: func() error {
			p, err := os.FindProcess(os.Getpid())
			if err == nil {
				_ = p.Signal(syscall.SIGTERM)
			}
			return nil
		},
		GetDevices: func() ([]string, error) {
			devs := cast.DiscoverDevices(false)
			var names []string
			for _, d := range devs {
				names = append(names, d.Name)
			}
			return names, nil
		},
		SwitchDevice: func(deviceName string) error {
			devs := cast.DiscoverDevices(false)
			dev, err := cast.ResolveDevice(deviceName, devs)
			if err != nil {
				return err
			}
			opts.TargetDev = dev
			targetDev = dev
			_ = ctrl.StopCast()
			ctrl.Close()
			ctrl = cast.NewController(targetDev)
			return ctrl.StartCast(streamURL)
		},
	})

	// 7. Start System Tray Item (SNI & DBusMenu)
	trayManager, err := tray.Start(targetDev.Name, port, tray.Callbacks{
		OnOpenRemote: func() {
			_ = exec.Command("xdg-open", remoteURL).Start()
		},
		OnVolumeStep: func(delta float64) {
			_ = ctrl.StepVolume(float32(delta))
		},
		OnToggleMute: func() {
			_, _ = ctrl.ToggleMute()
		},
		OnTogglePlay: func() {
			_ = ctrl.Pause()
		},
		OnStop: func() {
			p, err := os.FindProcess(os.Getpid())
			if err == nil {
				_ = p.Signal(syscall.SIGTERM)
			}
		},
	})
	if err == nil && trayManager != nil {
		defer trayManager.Stop()
	}

	// 8. Record state file
	st := &stream.State{
		PID:        os.Getpid(),
		DeviceName: targetDev.Name,
		DeviceIP:   targetDev.IP,
		Display:    display,
		Port:       port,
		Preset:     opts.Preset,
		HWAccel:    stream.HasVAAPI() && opts.HWAccel != "software",
		AudioOnly:  opts.AudioOnly,
		Monitor:    opts.Monitor,
		Window:     opts.Window,
		RemoteURL:  remoteURL,
		StartedAt:  startTime,
	}
	_ = stream.SaveState(st)
	defer stream.ClearState()

	fmt.Printf("\nNow casting desktop to '%s'!\n", targetDev.Name)

	// 9. Monitor termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	doneChan := make(chan error, 1)
	go func() {
		doneChan <- ffmpegCmd.Wait()
	}()

	select {
	case <-sigChan:
		fmt.Println("\nReceived stop signal. Stopping stream and releasing TV...")
		_ = ctrl.StopCast()
	case errWait := <-doneChan:
		if errWait != nil {
			fmt.Printf("\nFFmpeg process exited: %v\n", errWait)
		}
		_ = ctrl.StopCast()
	}

	return nil
}

// StartBackgroundWorker spawns the internal _worker process in the background, detached from the terminal.
func StartBackgroundWorker(opts CastOptions) (*stream.State, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to determine executable path: %w", err)
	}

	daemonLogPath := filepath.Join(os.Getenv("HOME"), ".config", "cast-desktop-go", "daemon.log")
	_ = os.MkdirAll(filepath.Dir(daemonLogPath), 0755)
	logFile, err := os.OpenFile(daemonLogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open daemon log: %w", err)
	}
	defer logFile.Close()

	workerArgs := []string{
		"_worker",
		"--device-ip", opts.TargetDev.IP,
		"--device-name", opts.TargetDev.Name,
		"--device-port", strconv.Itoa(opts.TargetDev.Port),
		"--display", opts.Display,
		"--fps", strconv.Itoa(opts.FPS),
		"--res", opts.Resolution,
		"--fit", opts.Fit,
		"--port", strconv.Itoa(opts.Port),
		"--preset", opts.Preset,
		"--hw-accel", opts.HWAccel,
		"--monitor", opts.Monitor,
		"--window", opts.Window,
	}
	if opts.AudioOnly {
		workerArgs = append(workerArgs, "--audio-only")
	}
	if !opts.Border {
		workerArgs = append(workerArgs, "--border=false")
	}

	proc := exec.Command(exePath, workerArgs...)
	proc.Stdout = logFile
	proc.Stderr = logFile
	proc.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := proc.Start(); err != nil {
		return nil, fmt.Errorf("failed to start background worker: %w", err)
	}

	startWait := time.Now()
	var activeState *stream.State

	for time.Since(startWait) < 30*time.Second {
		if !stream.IsProcessRunning(proc.Process.Pid) {
			break
		}
		st, err := stream.LoadState()
		if err == nil && st.PID > 0 && stream.IsProcessRunning(st.PID) {
			activeState = st
			break
		}
		time.Sleep(300 * time.Millisecond)
	}

	if activeState != nil {
		return activeState, nil
	}

	logBytes, _ := os.ReadFile(daemonLogPath)
	if len(logBytes) > 0 {
		return nil, fmt.Errorf("background worker failed to start:\n%s", string(logBytes))
	}
	return nil, fmt.Errorf("timed out waiting for background worker (check %s)", daemonLogPath)
}

// StartCastingTarget coordinates stopping previous sessions, checking display ratio, and launching background casting to targetDev.
func StartCastingTarget(targetDev *cast.Device, display string) (*stream.State, error) {
	_ = stopActiveSession()

	cfg := stream.LoadConfig()
	res := cfg.Resolution
	if res == "" {
		res = "1080p"
	}
	fit := cfg.Fit
	if fit == "" {
		fit = "pad"
	}
	fps := cfg.FPS
	if fps <= 0 {
		fps = 30
	}
	port := cfg.Port
	if port <= 0 {
		port = 8088
	}
	preset := cfg.Preset
	if preset == "" {
		preset = "balanced"
	}
	hwAccel := cfg.HWAccel
	if hwAccel == "" {
		hwAccel = "auto"
	}
	if display == "" {
		display = cfg.Display
	}
	if display == "" {
		display = os.Getenv("DISPLAY")
	}
	if display == "" {
		display = ":20"
	}

	// 16:9 auto check
	if !cfg.AudioOnly {
		origW, origH := stream.DetectDisplayResolution(display)
		if origW == 3840 && origH == 2560 {
			_ = exec.Command("xrandr", "-display", display, "--output", "DUMMY0", "--mode", "3840x2160_60.00").Run()
		}
	}

	cfg.DefaultDevice = targetDev.Name
	cfg.Resolution = res
	cfg.Fit = fit
	cfg.FPS = fps
	cfg.Display = display
	cfg.Port = port
	cfg.Preset = preset
	cfg.HWAccel = hwAccel
	_ = stream.SaveConfig(cfg)

	opts := CastOptions{
		TargetDev:  targetDev,
		Display:    display,
		FPS:        fps,
		Resolution: res,
		Fit:        fit,
		Port:       port,
		Preset:     preset,
		HWAccel:    hwAccel,
		Monitor:    cfg.Monitor,
		AudioOnly:  cfg.AudioOnly,
		Border:     cfg.Border,
	}

	return StartBackgroundWorker(opts)
}
