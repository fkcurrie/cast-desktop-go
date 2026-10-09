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
	"github.com/mattn/go-isatty"
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

	// Worker internal flags
	workerDeviceIP   string
	workerDeviceName string
	workerDevicePort int
)

func addStartFlags(c *cobra.Command) {
	c.Flags().StringVarP(&flagDevice, "device", "d", "", "Target device name, IP address, or index (interactive picker if omitted)")
	c.Flags().StringVar(&flagDisplay, "display", "", "X11 display to capture (default :20 or $DISPLAY)")
	c.Flags().IntVar(&flagFPS, "fps", 30, "Capture framerate")
	c.Flags().StringVar(&flagResolution, "res", "1080p", "Output resolution (1080p, 720p, native)")
	c.Flags().StringVar(&flagFit, "fit", "pad", "Fit mode: pad (preserve aspect ratio with bars), crop (zoom to fill 16:9), stretch")
	c.Flags().IntVarP(&flagPort, "port", "p", 8088, "HTTP streaming server port")
	c.Flags().BoolVarP(&flagDaemon, "daemon", "b", false, "Run cast session in the background")
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
	workerCmd.Flags().StringVar(&flagDisplay, "display", "", "")
	workerCmd.Flags().IntVar(&flagFPS, "fps", 30, "")
	workerCmd.Flags().StringVar(&flagResolution, "res", "1080p", "")
	workerCmd.Flags().StringVar(&flagFit, "fit", "pad", "")
	workerCmd.Flags().IntVar(&flagPort, "port", 8088, "")
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

func runStart(cmd *cobra.Command, args []string) error {
	// First ensure any previous active session is stopped
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

	// 1. Discover devices
	fmt.Println("Discovering Chromecast devices...")
	devices := cast.DiscoverDevices(true)
	if len(devices) == 0 {
		return fmt.Errorf("no Chromecast devices discovered on the local network")
	}

	// 2. Resolve target device
	var targetDev *cast.Device
	if flagDevice != "" {
		var err error
		targetDev, err = cast.ResolveDevice(flagDevice, devices)
		if err != nil {
			return err
		}
	} else if flagDaemon && cfg.DefaultDevice != "" {
		// In background mode without explicit device flag, automatically use saved default
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
			display = ":20" // Default for current Chrome Remote Desktop session
		}
	}

	// Automatically ensure 16:9 mode if display :20 is detected in 3:2 (3840x2560)
	origW, origH := stream.DetectDisplayResolution(display)
	if origW == 3840 && origH == 2560 {
		fmt.Println("Detected 3:2 desktop resolution (3840x2560); switching display to 16:9 4K (3840x2160)...")
		_ = exec.Command("xrandr", "-display", display, "--output", "DUMMY0", "--mode", "3840x2160_60.00").Run()
	}

	// Remember settings for next session
	cfg.DefaultDevice = targetDev.Name
	cfg.Resolution = flagResolution
	cfg.Fit = flagFit
	cfg.FPS = flagFPS
	cfg.Display = display
	cfg.Port = flagPort
	_ = stream.SaveConfig(cfg)

	fmt.Printf("\nPreparing to cast to: %s (%s)\n", targetDev.Name, targetDev.IP)
	fmt.Printf("Display: %s | Framerate: %d fps | Resolution: %s | Fit: %s\n", display, flagFPS, flagResolution, flagFit)

	localIP, err := cast.GetRouteToTarget(targetDev.IP)
	if err != nil {
		return err
	}
	streamURL := fmt.Sprintf("http://%s:%d/live.m3u8", localIP, flagPort)
	fmt.Printf("Local stream endpoint: %s\n", streamURL)

	if flagDaemon {
		fmt.Println("Waiting for stream initialization...")
		st, err := StartBackgroundWorker(targetDev, display, flagFPS, flagResolution, flagFit, flagPort)
		if err != nil {
			return err
		}
		fmt.Printf("\nDesktop casting successfully started in background!\n")
		fmt.Printf("  Target:     %s (%s)\n", targetDev.Name, targetDev.IP)
		fmt.Printf("  Worker PID: %d\n", st.PID)
		fmt.Printf("  Run 'cast-desktop-go status' to inspect.\n")
		fmt.Printf("  Run 'cast-desktop-go stop' to end session.\n\n")
		return nil
	}

	// Foreground mode
	fmt.Println("\nStarting casting session in foreground (press Ctrl+C to stop)...")
	return executeCastLoop(targetDev, display, flagFPS, flagResolution, flagFit, flagPort, streamURL)
}

func runWorker(cmd *cobra.Command, args []string) error {
	targetDev := &cast.Device{
		Name: workerDeviceName,
		IP:   workerDeviceIP,
		Port: workerDevicePort,
	}

	display := flagDisplay
	if display == "" {
		display = ":20"
	}

	localIP, err := cast.GetRouteToTarget(targetDev.IP)
	if err != nil {
		return err
	}
	streamURL := fmt.Sprintf("http://%s:%d/live.m3u8", localIP, flagPort)

	return executeCastLoop(targetDev, display, flagFPS, flagResolution, flagFit, flagPort, streamURL)
}

func executeCastLoop(targetDev *cast.Device, display string, fps int, res string, fit string, port int, streamURL string) error {
	// 1. Start HTTP Server
	srv, err := stream.NewServer(port)
	if err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	defer srv.Stop()

	// 2. Start FFmpeg capture
	ffmpegLog := filepath.Join(os.Getenv("HOME"), ".config", "cast-desktop-go", "ffmpeg.log")
	opts := stream.FFmpegOptions{
		Display:    display,
		FPS:        fps,
		Resolution: res,
		Fit:        fit,
	}

	ffmpegCmd, err := stream.StartFFmpeg(opts, ffmpegLog)
	if err != nil {
		return err
	}
	defer stream.StopFFmpeg(ffmpegCmd)

	// 3. Record state once streaming infrastructure is ready
	st := &stream.State{
		PID:        os.Getpid(),
		DeviceName: targetDev.Name,
		DeviceIP:   targetDev.IP,
		Display:    display,
		Port:       port,
		StartedAt:  time.Now(),
	}
	_ = stream.SaveState(st)
	defer stream.ClearState()

	// 4. Connect to Chromecast and begin playback
	ctrl := cast.NewController(targetDev)
	if err := ctrl.StartCast(streamURL); err != nil {
		return fmt.Errorf("failed to launch Chromecast stream: %w", err)
	}
	defer ctrl.Close()

	fmt.Printf("\nNow casting desktop to '%s'!\n", targetDev.Name)

	// 5. Monitor signals and process health
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
func StartBackgroundWorker(targetDev *cast.Device, display string, fps int, res string, fit string, port int) (*stream.State, error) {
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
		"--device-ip", targetDev.IP,
		"--device-name", targetDev.Name,
		"--device-port", strconv.Itoa(targetDev.Port),
		"--display", display,
		"--fps", strconv.Itoa(fps),
		"--res", res,
		"--fit", fit,
		"--port", strconv.Itoa(port),
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
	origW, origH := stream.DetectDisplayResolution(display)
	if origW == 3840 && origH == 2560 {
		_ = exec.Command("xrandr", "-display", display, "--output", "DUMMY0", "--mode", "3840x2160_60.00").Run()
	}

	cfg.DefaultDevice = targetDev.Name
	cfg.Resolution = res
	cfg.Fit = fit
	cfg.FPS = fps
	cfg.Display = display
	cfg.Port = port
	_ = stream.SaveConfig(cfg)

	return StartBackgroundWorker(targetDev, display, fps, res, fit, port)
}

