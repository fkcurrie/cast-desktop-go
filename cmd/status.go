package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cast-desktop-go/pkg/stream"
	"github.com/spf13/cobra"
)

func readLastLines(path string, maxLines int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > maxLines {
			lines = lines[1:]
		}
	}
	return lines, scanner.Err()
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show active casting status",
	Run: func(cmd *cobra.Command, args []string) {
		st, err := stream.LoadState()
		if err != nil {
			fmt.Println("Status: No active desktop cast session.")
			return
		}

		if st.PID > 0 && stream.IsProcessRunning(st.PID) {
			uptime := time.Since(st.StartedAt).Truncate(time.Second)
			fmt.Println("Status: CASTING ACTIVE")
			fmt.Printf("  Target Device: %s (%s)\n", st.DeviceName, st.DeviceIP)
			if st.AudioOnly {
				fmt.Printf("  Mode:          Audio-Only (Smart Speaker / TV)\n")
			} else {
				fmt.Printf("  Display:       %s\n", st.Display)
				if st.Monitor != "" {
					fmt.Printf("  Monitor:       %s\n", st.Monitor)
				}
				if st.Window != "" {
					fmt.Printf("  Window:        %s\n", st.Window)
				}
			}
			if st.Preset != "" {
				fmt.Printf("  Latency Mode:  %s\n", st.Preset)
			}
			hwStr := "Software (libx264)"
			if st.HWAccel {
				hwStr = "AMD VA-API Hardware GPU"
			}
			fmt.Printf("  Encoder:       %s\n", hwStr)
			fmt.Printf("  Stream Port:   %d\n", st.Port)
			if st.RemoteURL != "" {
				fmt.Printf("  Web Remote:    %s\n", st.RemoteURL)
			}
			fmt.Printf("  Worker PID:    %d\n", st.PID)
			fmt.Printf("  Uptime:        %s\n", uptime)

			logDir := filepath.Join(os.Getenv("HOME"), ".config", "cast-desktop-go")
			httpLogPath := filepath.Join(logDir, "http.log")
			if lines, err := readLastLines(httpLogPath, 5); err == nil && len(lines) > 0 {
				fmt.Println("\nRecent Chromecast HTTP Activity:")
				for _, l := range lines {
					fmt.Printf("  %s\n", l)
				}
			}
			fmt.Printf("\nLog files:\n  HTTP:   %s\n  FFmpeg: %s/ffmpeg.log\n  Daemon: %s/daemon.log\n", httpLogPath, logDir, logDir)
		} else {
			fmt.Println("Status: No active session (cleaning up stale state file).")
			_ = stream.ClearState()
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
