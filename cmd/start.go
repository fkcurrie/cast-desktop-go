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
	workerCmd.Flags().IntVar(&flagPort, "port", 8088, "")
}

func selectDeviceInteractive(devices []cast.Device) *cast.Device {
	if len(devices) == 1 {
		return &devices[0]
	}

	fmt.Printf("\nDiscovered %d Chromecast devices:\n", len(devices))
	for i, d := range devices {
		fmt.Printf("  [%d] %s (%s)\n", i+1, d.Name, d.IP)
	}

	if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		fmt.Printf("Non-interactive terminal detected; defaulting to [%d] %s\n", 1, devices[0].Name)
		return &devices[0]
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("\nSelect a device [1-%d] (default: 1): ", len(devices))
		input, err := reader.ReadString('\n')
		if err != nil {
			return &devices[0]
		}
		input = strings.TrimSpace(input)
		if input == "" {
			return &devices[0]
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
	} else {
		targetDev = selectDeviceInteractive(devices)
	}

	display := flagDisplay
	if display == "" {
		display = os.Getenv("DISPLAY")
		if display == "" {
			display = ":20" // Default for current Chrome Remote Desktop session
		}
	}

	fmt.Printf("\nPreparing to cast to: %s (%s)\n", targetDev.Name, targetDev.IP)
	fmt.Printf("Display: %s | Framerate: %d fps | Resolution: %s\n", display, flagFPS, flagResolution)

	localIP, err := cast.GetRouteToTarget(targetDev.IP)
	if err != nil {
		return err
	}
	streamURL := fmt.Sprintf("http://%s:%d/live.m3u8", localIP, flagPort)
	fmt.Printf("Local stream endpoint: %s\n", streamURL)

	if flagDaemon {
		// Launch background worker
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to determine executable path: %w", err)
		}

		daemonLogPath := filepath.Join(os.Getenv("HOME"), ".config", "cast-desktop-go", "daemon.log")
		logFile, err := os.OpenFile(daemonLogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("failed to open daemon log: %w", err)
		}

		workerArgs := []string{
			"_worker",
			"--device-ip", targetDev.IP,
			"--device-name", targetDev.Name,
			"--device-port", strconv.Itoa(targetDev.Port),
			"--display", display,
			"--fps", strconv.Itoa(flagFPS),
			"--res", flagResolution,
			"--port", strconv.Itoa(flagPort),
		}

		proc := exec.Command(exePath, workerArgs...)
		proc.Stdout = logFile
		proc.Stderr = logFile
		proc.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

		if err := proc.Start(); err != nil {
			return fmt.Errorf("failed to start background worker: %w", err)
		}

		fmt.Println("Waiting for stream initialization...")
		startWait := time.Now()
		var activeState *stream.State

		for time.Since(startWait) < 15*time.Second {
			st, err := stream.LoadState()
			if err == nil && st.PID > 0 && stream.IsProcessRunning(st.PID) {
				activeState = st
				break
			}
			time.Sleep(300 * time.Millisecond)
		}

		if activeState != nil {
			fmt.Printf("\nDesktop casting successfully started in background!\n")
			fmt.Printf("  Target:     %s (%s)\n", targetDev.Name, targetDev.IP)
			fmt.Printf("  Worker PID: %d\n", proc.Process.Pid)
			fmt.Printf("  Run 'cast-desktop-go status' to inspect.\n")
			fmt.Printf("  Run 'cast-desktop-go stop' to end session.\n\n")
			return nil
		}

		return fmt.Errorf("timed out waiting for background worker (check %s)", daemonLogPath)
	}

	// Foreground mode
	fmt.Println("\nStarting casting session in foreground (press Ctrl+C to stop)...")
	return executeCastLoop(targetDev, display, flagFPS, flagResolution, flagPort, streamURL)
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

	return executeCastLoop(targetDev, display, flagFPS, flagResolution, flagPort, streamURL)
}

func executeCastLoop(targetDev *cast.Device, display string, fps int, res string, port int, streamURL string) error {
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
	}

	ffmpegCmd, err := stream.StartFFmpeg(opts, ffmpegLog)
	if err != nil {
		return err
	}
	defer stream.StopFFmpeg(ffmpegCmd)

	// 3. Connect to Chromecast and begin playback
	ctrl := cast.NewController(targetDev)
	if err := ctrl.StartCast(streamURL); err != nil {
		return fmt.Errorf("failed to launch Chromecast stream: %w", err)
	}
	defer ctrl.Close()

	// 4. Save state
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
