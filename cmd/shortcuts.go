package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var shortcutsCmd = &cobra.Command{
	Use:   "shortcuts",
	Short: "Manage KDE Plasma global keyboard shortcuts for casting and TV controls",
}

var shortcutsInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install global shortcuts (Meta+Shift+C: Launch, Meta+Shift+X: Stop, Meta+Shift+M: Mute)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return installGlobalShortcuts()
	},
}

var shortcutsRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove registered global shortcuts",
	RunE: func(cmd *cobra.Command, args []string) error {
		return removeGlobalShortcuts()
	},
}

var shortcutsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured shortcuts",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Configured Global Shortcuts for Cast Desktop:")
		fmt.Println("  Meta + Shift + C   -> Launch Cast Desktop GUI")
		fmt.Println("  Meta + Shift + X   -> Stop Active Casting Session")
		fmt.Println("  Meta + Shift + M   -> Toggle TV Mute")
		fmt.Println("  Meta + Shift + Up  -> TV Volume Up (+5%)")
		fmt.Println("  Meta + Shift + Down-> TV Volume Down (-5%)")
	},
}

func init() {
	shortcutsCmd.AddCommand(shortcutsInstallCmd)
	shortcutsCmd.AddCommand(shortcutsRemoveCmd)
	shortcutsCmd.AddCommand(shortcutsListCmd)
	rootCmd.AddCommand(shortcutsCmd)
}

func installGlobalShortcuts() error {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		return fmt.Errorf("HOME environment variable not set")
	}

	// 1. Ensure ~/.local/share/applications/cast-desktop-go.desktop includes desktop actions
	appsDir := filepath.Join(homeDir, ".local", "share", "applications")
	_ = os.MkdirAll(appsDir, 0755)
	desktopFile := filepath.Join(appsDir, "cast-desktop-go.desktop")

	desktopContent := `[Desktop Entry]
Name=Cast Desktop
GenericName=Chromecast Desktop Streamer
Comment=Stream Linux desktop to Google Chromecast
Exec=cast-desktop-go gui
Icon=cast-desktop-go
Terminal=false
Type=Application
Categories=AudioVideo;Video;Network;
Actions=StopCast;ToggleMute;VolumeUp;VolumeDown;

[Desktop Action StopCast]
Name=Stop Casting
Exec=cast-desktop-go stop
Icon=media-playback-stop

[Desktop Action ToggleMute]
Name=Toggle TV Mute
Exec=cast-desktop-go mute toggle
Icon=audio-volume-muted

[Desktop Action VolumeUp]
Name=TV Volume Up (+5%)
Exec=cast-desktop-go volume +5
Icon=audio-volume-high

[Desktop Action VolumeDown]
Name=TV Volume Down (-5%)
Exec=cast-desktop-go volume -5
Icon=audio-volume-low
`
	if err := os.WriteFile(desktopFile, []byte(desktopContent), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", desktopFile, err)
	}

	// 2. Configure ~/.config/kglobalshortcutsrc
	kconfigPath := filepath.Join(homeDir, ".config", "kglobalshortcutsrc")
	lines := []string{}
	if data, err := os.ReadFile(kconfigPath); err == nil {
		lines = strings.Split(string(data), "\n")
	}

	// Filter out any existing [cast-desktop-go.desktop] section
	var filtered []string
	inSection := false
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			inSection = (trimmed == "[cast-desktop-go.desktop]")
		}
		if !inSection {
			filtered = append(filtered, l)
		}
	}

	// Append updated section
	section := []string{
		"[cast-desktop-go.desktop]",
		"_k_friendly_name=Cast Desktop",
		"_launch=Meta+Shift+C,Meta+Shift+C,Cast Desktop",
		"StopCast=Meta+Shift+X,Meta+Shift+X,Stop Active Casting",
		"ToggleMute=Meta+Shift+M,Meta+Shift+M,Toggle TV Mute",
		"VolumeUp=Meta+Shift+Up,Meta+Shift+Up,TV Volume Up (+5%)",
		"VolumeDown=Meta+Shift+Down,Meta+Shift+Down,TV Volume Down (-5%)",
	}

	newContent := strings.Join(filtered, "\n")
	if !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	newContent += strings.Join(section, "\n") + "\n"

	if err := os.WriteFile(kconfigPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to update %s: %w", kconfigPath, err)
	}

	// 3. Reload KDE global shortcuts daemon if available
	_ = exec.Command("kquitapp6", "kglobalaccel").Run()
	_ = exec.Command("dbus-send", "--session", "--dest=org.kde.KGlobalAccel", "--type=method_call", "/kglobalaccel", "org.kde.KGlobalAccel.blockGlobalShortcuts", "boolean:false").Run()

	fmt.Println("Global shortcuts successfully installed for KDE Plasma!")
	fmt.Println("  Meta + Shift + C   -> Launch Cast Desktop GUI")
	fmt.Println("  Meta + Shift + X   -> Stop Active Casting")
	fmt.Println("  Meta + Shift + M   -> Toggle TV Mute")
	fmt.Println("  Meta + Shift + Up  -> TV Volume Up (+5%)")
	fmt.Println("  Meta + Shift + Down-> TV Volume Down (-5%)")
	return nil
}

func removeGlobalShortcuts() error {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		return fmt.Errorf("HOME environment variable not set")
	}

	kconfigPath := filepath.Join(homeDir, ".config", "kglobalshortcutsrc")
	data, err := os.ReadFile(kconfigPath)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(data), "\n")
	var filtered []string
	inSection := false
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			inSection = (trimmed == "[cast-desktop-go.desktop]")
		}
		if !inSection {
			filtered = append(filtered, l)
		}
	}

	if err := os.WriteFile(kconfigPath, []byte(strings.Join(filtered, "\n")), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", kconfigPath, err)
	}

	fmt.Println("Global shortcuts removed.")
	return nil
}
