package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"cast-desktop-go/pkg/cast"
	"cast-desktop-go/pkg/stream"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop active casting session and return Chromecast to home screen",
	RunE: func(cmd *cobra.Command, args []string) error {
		return stopActiveSession()
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

func stopActiveSession() error {
	st, err := stream.LoadState()
	if err != nil {
		// No state file, try fallback cleanup
		_ = exec.Command("pkill", "-f", "cast_desktop_go").Run()
		stream.CleanStreamDir()
		fmt.Println("No active casting session found.")
		return nil
	}

	fmt.Printf("Stopping casting session on %s (%s)...\n", st.DeviceName, st.DeviceIP)

	// 1. Tell Chromecast to stop playback and return to backdrop
	ctrl := cast.NewController(&cast.Device{
		Name: st.DeviceName,
		IP:   st.DeviceIP,
		Port: 8009,
	})
	if err := ctrl.StopCast(); err != nil {
		fmt.Printf("Notice: Could not send stop to Chromecast: %v\n", err)
	} else {
		fmt.Println("Chromecast released back to home screen.")
	}

	// 2. Terminate worker process if alive
	if st.PID > 0 && stream.IsProcessRunning(st.PID) {
		pgid, err := syscall.Getpgid(st.PID)
		if err == nil {
			_ = syscall.Kill(-pgid, syscall.SIGTERM)
			time.Sleep(400 * time.Millisecond)
			if stream.IsProcessRunning(st.PID) {
				_ = syscall.Kill(-pgid, syscall.SIGKILL)
			}
		} else {
			proc, err := os.FindProcess(st.PID)
			if err == nil {
				_ = proc.Signal(syscall.SIGTERM)
			}
		}
	}

	// 3. Cleanup ramdisk & state
	_ = exec.Command("pkill", "-f", "cast_desktop_go").Run()
	stream.CleanStreamDir()
	_ = stream.ClearState()

	fmt.Println("Casting session stopped successfully.")
	return nil
}
