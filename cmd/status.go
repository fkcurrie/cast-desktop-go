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
			fmt.Printf("  Display:       %s\n", st.Display)
			fmt.Printf("  Stream Port:   %d\n", st.Port)
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
