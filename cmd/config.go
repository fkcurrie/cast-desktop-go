package cmd

import (
	"fmt"
	"path/filepath"
	"os"

	"cast-desktop-go/pkg/stream"
	"github.com/spf13/cobra"
)

var (
	configSetDevice     string
	configSetDisplay    string
	configSetResolution string
	configSetFit        string
	configSetFPS        int
	configSetPort       int
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or modify persistent casting settings",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := stream.LoadConfig()
		cfgPath := filepath.Join(os.Getenv("HOME"), ".config", "cast-desktop-go", "config.json")

		fmt.Println("Current Persistent Configuration:")
		fmt.Printf("  Default Device: %s\n", cfg.DefaultDevice)
		fmt.Printf("  Display:        %s\n", cfg.Display)
		fmt.Printf("  Resolution:     %s\n", cfg.Resolution)
		fmt.Printf("  Fit Mode:       %s\n", cfg.Fit)
		fmt.Printf("  Framerate:      %d fps\n", cfg.FPS)
		fmt.Printf("  Stream Port:    %d\n\n", cfg.Port)
		fmt.Printf("Configuration file: %s\n", cfgPath)
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Update default casting configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := stream.LoadConfig()

		if cmd.Flags().Changed("device") {
			cfg.DefaultDevice = configSetDevice
		}
		if cmd.Flags().Changed("display") {
			cfg.Display = configSetDisplay
		}
		if cmd.Flags().Changed("res") {
			cfg.Resolution = configSetResolution
		}
		if cmd.Flags().Changed("fit") {
			cfg.Fit = configSetFit
		}
		if cmd.Flags().Changed("fps") {
			cfg.FPS = configSetFPS
		}
		if cmd.Flags().Changed("port") {
			cfg.Port = configSetPort
		}

		if err := stream.SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Println("Configuration successfully updated:")
		fmt.Printf("  Default Device: %s\n", cfg.DefaultDevice)
		fmt.Printf("  Display:        %s\n", cfg.Display)
		fmt.Printf("  Resolution:     %s\n", cfg.Resolution)
		fmt.Printf("  Fit Mode:       %s\n", cfg.Fit)
		fmt.Printf("  Framerate:      %d fps\n", cfg.FPS)
		fmt.Printf("  Stream Port:    %d\n", cfg.Port)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)

	configSetCmd.Flags().StringVarP(&configSetDevice, "device", "d", "", "Default Chromecast device name or IP")
	configSetCmd.Flags().StringVar(&configSetDisplay, "display", "", "Default display (e.g. :20)")
	configSetCmd.Flags().StringVar(&configSetResolution, "res", "", "Default resolution (1080p, 720p, native)")
	configSetCmd.Flags().StringVar(&configSetFit, "fit", "", "Default fit mode (pad, crop, stretch)")
	configSetCmd.Flags().IntVar(&configSetFPS, "fps", 0, "Default framerate (e.g. 30 or 60)")
	configSetCmd.Flags().IntVarP(&configSetPort, "port", "p", 0, "Default streaming port")
}
