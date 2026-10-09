package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"cast-desktop-go/pkg/cast"
	"cast-desktop-go/pkg/stream"
	"github.com/spf13/cobra"
)

var (
	guiQuickCast bool
	guiDevice    string
	guiStop      bool
	guiStatus    bool
	guiDisplay   string
)

type menuItem struct {
	Tag   string
	Label string
}

func ensureDisplay() string {
	disp := guiDisplay
	if disp == "" {
		disp = os.Getenv("DISPLAY")
	}
	if disp == "" {
		disp = ":20"
		os.Setenv("DISPLAY", disp)
	}
	return disp
}

func sendNotification(title, message, icon string) {
	disp := ensureDisplay()
	cmd := exec.Command("notify-send", "-a", "Cast Desktop", "-i", icon, title, message)
	cmd.Env = append(os.Environ(), "DISPLAY="+disp)
	_ = cmd.Run()
}

func showDialogMenu(title, prompt string, items []menuItem) (string, error) {
	disp := ensureDisplay()

	// Try kdialog first (native for KDE Plasma)
	if _, err := exec.LookPath("kdialog"); err == nil {
		args := []string{"--title", title, "--menu", prompt}
		for _, item := range items {
			args = append(args, item.Tag, item.Label)
		}
		cmd := exec.Command("kdialog", args...)
		cmd.Env = append(os.Environ(), "DISPLAY="+disp)
		out, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}
		// If user cancelled, kdialog exits with error 1
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", nil
		}
	}

	// Fallback to zenity
	if _, err := exec.LookPath("zenity"); err == nil {
		args := []string{"--list", "--title=" + title, "--text=" + prompt, "--column=ID", "--column=Option", "--hide-column=1"}
		for _, item := range items {
			args = append(args, item.Tag, item.Label)
		}
		cmd := exec.Command("zenity", args...)
		cmd.Env = append(os.Environ(), "DISPLAY="+disp)
		out, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}
		return "", nil
	}

	return "", fmt.Errorf("neither kdialog nor zenity is available for GUI dialogs")
}

func showInfoDialog(title, message string) {
	disp := ensureDisplay()
	if _, err := exec.LookPath("kdialog"); err == nil {
		cmd := exec.Command("kdialog", "--title", title, "--msgbox", message)
		cmd.Env = append(os.Environ(), "DISPLAY="+disp)
		_ = cmd.Run()
		return
	}
	if _, err := exec.LookPath("zenity"); err == nil {
		cmd := exec.Command("zenity", "--info", "--title="+title, "--text="+message)
		cmd.Env = append(os.Environ(), "DISPLAY="+disp)
		_ = cmd.Run()
		return
	}
}

func showErrorDialog(title, message string) {
	disp := ensureDisplay()
	if _, err := exec.LookPath("kdialog"); err == nil {
		cmd := exec.Command("kdialog", "--title", title, "--error", message)
		cmd.Env = append(os.Environ(), "DISPLAY="+disp)
		_ = cmd.Run()
		return
	}
	if _, err := exec.LookPath("zenity"); err == nil {
		cmd := exec.Command("zenity", "--error", "--title="+title, "--text="+message)
		cmd.Env = append(os.Environ(), "DISPLAY="+disp)
		_ = cmd.Run()
		return
	}
}

var guiCmd = &cobra.Command{
	Use:   "gui",
	Short: "Desktop GUI launcher for desktop environments (KDE Plasma menu integration)",
	RunE: func(cmd *cobra.Command, args []string) error {
		disp := ensureDisplay()

		// 1. Handle Stop action
		if guiStop {
			_ = stopActiveSession()
			sendNotification("Cast Desktop", "Casting session stopped. Display released.", "media-playback-stop")
			return nil
		}

		// 2. Handle Status action
		if guiStatus {
			st, err := stream.LoadState()
			if err == nil && st.PID > 0 && stream.IsProcessRunning(st.PID) {
				uptime := time.Since(st.StartedAt).Truncate(time.Second)
				msg := fmt.Sprintf("Streaming to %s (%s)\nDisplay: %s | Port: %d | Uptime: %s",
					st.DeviceName, st.DeviceIP, st.Display, st.Port, uptime)
				sendNotification("Cast Desktop: ACTIVE", msg, "cast-desktop")
			} else {
				sendNotification("Cast Desktop", "No active desktop casting session.", "dialog-information")
			}
			return nil
		}

		// 3. Handle Quick Cast action
		if guiQuickCast {
			cfg := stream.LoadConfig()
			if cfg.DefaultDevice == "" {
				// No saved default; fall through to interactive device picker
			} else {
				sendNotification("Cast Desktop", fmt.Sprintf("Connecting to saved TV: %s...", cfg.DefaultDevice), "cast-desktop")
				devices := cast.DiscoverDevices(true)
				dev, err := cast.ResolveDevice(cfg.DefaultDevice, devices)
				if err != nil {
					showErrorDialog("Cast Desktop", fmt.Sprintf("Could not locate default device '%s': %v", cfg.DefaultDevice, err))
					return err
				}
				_, err = StartCastingTarget(dev, disp)
				if err != nil {
					showErrorDialog("Cast Desktop Failed", fmt.Sprintf("Failed to cast to %s: %v", dev.Name, err))
					sendNotification("Cast Desktop Failed", err.Error(), "dialog-error")
					return err
				}
				sendNotification("Cast Desktop Active", fmt.Sprintf("Now casting screen to %s", dev.Name), "cast-desktop")
				return nil
			}
		}

		// 4. Handle direct Device action
		if guiDevice != "" {
			sendNotification("Cast Desktop", fmt.Sprintf("Searching for '%s'...", guiDevice), "cast-desktop")
			devices := cast.DiscoverDevices(true)
			dev, err := cast.ResolveDevice(guiDevice, devices)
			if err != nil {
				showErrorDialog("Cast Desktop", fmt.Sprintf("Could not find device '%s': %v", guiDevice, err))
				return err
			}
			sendNotification("Cast Desktop", fmt.Sprintf("Connecting to %s...", dev.Name), "cast-desktop")
			_, err = StartCastingTarget(dev, disp)
			if err != nil {
				showErrorDialog("Cast Desktop Failed", fmt.Sprintf("Failed to cast to %s: %v", dev.Name, err))
				sendNotification("Cast Desktop Failed", err.Error(), "dialog-error")
				return err
			}
			sendNotification("Cast Desktop Active", fmt.Sprintf("Now casting screen to %s", dev.Name), "cast-desktop")
			return nil
		}

		// 5. Default action: interactive GUI manager
		st, err := stream.LoadState()
		if err == nil && st.PID > 0 && stream.IsProcessRunning(st.PID) {
			uptime := time.Since(st.StartedAt).Truncate(time.Second)
			prompt := fmt.Sprintf("Desktop casting is currently active on %s (uptime: %s).\n\nWhat would you like to do?", st.DeviceName, uptime)
			choice, err := showDialogMenu("Cast Desktop - Active Session", prompt, []menuItem{
				{"status", "View Status & Stream Details"},
				{"switch", "Switch to Another Chromecast TV"},
				{"stop", "Stop Active Casting Session"},
			})
			if err != nil || choice == "" {
				return nil // user closed dialog
			}

			switch choice {
			case "status":
				msg := fmt.Sprintf("Active Session:\n\nTarget TV: %s (%s)\nDisplay: %s\nStreaming Port: %d\nUptime: %s\nWorker PID: %d",
					st.DeviceName, st.DeviceIP, st.Display, st.Port, uptime, st.PID)
				showInfoDialog("Cast Desktop Status", msg)
				return nil
			case "stop":
				_ = stopActiveSession()
				sendNotification("Cast Desktop", fmt.Sprintf("Stopped casting to %s", st.DeviceName), "media-playback-stop")
				return nil
			case "switch":
				// proceed below to device picker
			}
		}

		sendNotification("Cast Desktop", "Scanning local network for Chromecast devices...", "cast-desktop")
		devices := cast.DiscoverDevices(true)
		if len(devices) == 0 {
			showErrorDialog("Cast Desktop", "No Chromecast devices found on the local network.")
			return nil
		}

		cfg := stream.LoadConfig()
		var items []menuItem
		for i, d := range devices {
			label := fmt.Sprintf("%s (%s)", d.Name, d.IP)
			if cfg.DefaultDevice != "" && strings.EqualFold(d.Name, cfg.DefaultDevice) {
				label += "  ★ Saved Default"
			}
			items = append(items, menuItem{
				Tag:   strconv.Itoa(i),
				Label: label,
			})
		}

		choice, err := showDialogMenu("Cast Desktop - Select Target TV", "Choose a Chromecast display to cast your screen to:", items)
		if err != nil || choice == "" {
			return nil // User cancelled
		}

		idx, err := strconv.Atoi(choice)
		if err != nil || idx < 0 || idx >= len(devices) {
			return nil
		}

		targetDev := &devices[idx]
		sendNotification("Cast Desktop", fmt.Sprintf("Starting screen cast to %s...", targetDev.Name), "cast-desktop")

		_, err = StartCastingTarget(targetDev, disp)
		if err != nil {
			showErrorDialog("Cast Desktop Failed", fmt.Sprintf("Failed to cast to %s: %v", targetDev.Name, err))
			sendNotification("Cast Desktop Failed", err.Error(), "dialog-error")
			return err
		}

		sendNotification("Cast Desktop Active", fmt.Sprintf("Now casting screen to %s", targetDev.Name), "cast-desktop")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(guiCmd)

	guiCmd.Flags().BoolVar(&guiQuickCast, "quick-cast", false, "Cast directly to saved default TV")
	guiCmd.Flags().StringVarP(&guiDevice, "device", "d", "", "Cast directly to named Chromecast device")
	guiCmd.Flags().BoolVar(&guiStop, "stop", false, "Stop active casting session")
	guiCmd.Flags().BoolVar(&guiStatus, "status", false, "Display current casting status")
	guiCmd.Flags().StringVar(&guiDisplay, "display", "", "X11 display to cast")
}
