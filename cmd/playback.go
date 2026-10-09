package cmd

import (
	"fmt"
	"net/http"

	"cast-desktop-go/pkg/stream"
	"github.com/spf13/cobra"
)

var pauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause Chromecast stream playback",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := stream.LoadState()
		if err != nil || st.PID <= 0 || !stream.IsProcessRunning(st.PID) {
			return fmt.Errorf("no active cast session found")
		}

		apiURL := fmt.Sprintf("http://127.0.0.1:%d/api/pause", st.Port)
		resp, err := http.Post(apiURL, "application/json", nil)
		if err != nil || resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to pause playback")
		}

		fmt.Printf("Playback paused on %s.\n", st.DeviceName)
		return nil
	},
}

var playCmd = &cobra.Command{
	Use:   "play",
	Short: "Resume Chromecast stream playback",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := stream.LoadState()
		if err != nil || st.PID <= 0 || !stream.IsProcessRunning(st.PID) {
			return fmt.Errorf("no active cast session found")
		}

		apiURL := fmt.Sprintf("http://127.0.0.1:%d/api/play", st.Port)
		resp, err := http.Post(apiURL, "application/json", nil)
		if err != nil || resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to resume playback")
		}

		fmt.Printf("Playback resumed on %s.\n", st.DeviceName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pauseCmd)
	rootCmd.AddCommand(playCmd)
}
