package cmd

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"cast-desktop-go/pkg/cast"
	"cast-desktop-go/pkg/stream"
	"github.com/spf13/cobra"
)

var volumeCmd = &cobra.Command{
	Use:   "volume [0-100 | +N | -N]",
	Short: "Get or set Chromecast TV volume level",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := stream.LoadState()
		var targetIP string
		var targetName string
		port := 8088

		if err == nil && st.PID > 0 && stream.IsProcessRunning(st.PID) {
			targetIP = st.DeviceIP
			targetName = st.DeviceName
			port = st.Port
		} else {
			cfg := stream.LoadConfig()
			if cfg.DefaultDevice == "" {
				return fmt.Errorf("no active cast session and no default device configured")
			}
			devices := cast.DiscoverDevices(true)
			dev, err := cast.ResolveDevice(cfg.DefaultDevice, devices)
			if err != nil {
				return fmt.Errorf("failed to locate default device %s: %w", cfg.DefaultDevice, err)
			}
			targetIP = dev.IP
			targetName = dev.Name
		}

		if len(args) == 0 {
			// Query current volume
			lvl, muted, err := cast.GetDeviceVolume(&cast.Device{Name: targetName, IP: targetIP, Port: 8009})
			if err != nil {
				return err
			}
			muteStr := ""
			if muted {
				muteStr = " [MUTED]"
			}
			fmt.Printf("Volume for '%s': %.0f%%%s\n", targetName, lvl*100, muteStr)
			return nil
		}

		arg := strings.TrimSpace(args[0])
		var newLevel float32

		if strings.HasPrefix(arg, "+") || strings.HasPrefix(arg, "-") {
			curLvl, _, err := cast.GetDeviceVolume(&cast.Device{Name: targetName, IP: targetIP, Port: 8009})
			if err != nil {
				return err
			}
			delta, err := strconv.ParseFloat(arg, 32)
			if err != nil {
				return fmt.Errorf("invalid volume offset: %s", arg)
			}
			newLevel = curLvl + float32(delta)/100.0
		} else {
			val, err := strconv.ParseFloat(arg, 32)
			if err != nil {
				return fmt.Errorf("invalid volume level: %s (expected 0-100)", arg)
			}
			newLevel = float32(val) / 100.0
		}

		if newLevel < 0 {
			newLevel = 0
		}
		if newLevel > 1.0 {
			newLevel = 1.0
		}

		// Try sending via worker HTTP API if active
		apiURL := fmt.Sprintf("http://127.0.0.1:%d/api/volume?level=%.2f", port, newLevel)
		resp, err := http.Post(apiURL, "application/json", nil)
		if err == nil && resp.StatusCode == http.StatusOK {
			fmt.Printf("Volume set to %.0f%% on %s (via active session)\n", newLevel*100, targetName)
			return nil
		}

		// Fallback to direct Chromecast command
		err = cast.SendVolumeCommand(&cast.Device{Name: targetName, IP: targetIP, Port: 8009}, newLevel)
		if err != nil {
			return err
		}
		fmt.Printf("Volume set to %.0f%% on %s\n", newLevel*100, targetName)
		return nil
	},
}

var muteCmd = &cobra.Command{
	Use:   "mute [on | off | toggle]",
	Short: "Toggle or set Chromecast TV audio mute",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := stream.LoadState()
		var targetIP string
		var targetName string
		port := 8088

		if err == nil && st.PID > 0 && stream.IsProcessRunning(st.PID) {
			targetIP = st.DeviceIP
			targetName = st.DeviceName
			port = st.Port
		} else {
			cfg := stream.LoadConfig()
			if cfg.DefaultDevice == "" {
				return fmt.Errorf("no active cast session and no default device configured")
			}
			devices := cast.DiscoverDevices(true)
			dev, err := cast.ResolveDevice(cfg.DefaultDevice, devices)
			if err != nil {
				return err
			}
			targetIP = dev.IP
			targetName = dev.Name
		}

		action := "toggle"
		if len(args) > 0 {
			action = strings.ToLower(args[0])
		}

		var targetMute bool
		if action == "toggle" {
			_, curMuted, err := cast.GetDeviceVolume(&cast.Device{Name: targetName, IP: targetIP, Port: 8009})
			if err != nil {
				return err
			}
			targetMute = !curMuted
		} else if action == "on" || action == "true" || action == "yes" {
			targetMute = true
		} else {
			targetMute = false
		}

		apiURL := fmt.Sprintf("http://127.0.0.1:%d/api/mute?val=%t", port, targetMute)
		resp, err := http.Post(apiURL, "application/json", nil)
		if err == nil && resp.StatusCode == http.StatusOK {
			status := "Muted"
			if !targetMute {
				status = "Unmuted"
			}
			fmt.Printf("%s audio on %s\n", status, targetName)
			return nil
		}

		err = cast.SendMuteCommand(&cast.Device{Name: targetName, IP: targetIP, Port: 8009}, targetMute)
		if err != nil {
			return err
		}
		status := "Muted"
		if !targetMute {
			status = "Unmuted"
		}
		fmt.Printf("%s audio on %s\n", status, targetName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(volumeCmd)
	rootCmd.AddCommand(muteCmd)
}
