package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cast-desktop-go",
	Short: "Cast Linux desktop and audio to Chromecast devices",
	Long: `cast-desktop-go is a high-performance CLI utility for casting the Linux desktop
and audio (X11 & PipeWire/PulseAudio) to Google Cast receivers (such as Living room TV
and Bigscreen Basement TV) over low-latency HLS.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStart(cmd, args)
	},
}

// Execute runs the root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Flags for root default execution (start)
	addStartFlags(rootCmd)
}
