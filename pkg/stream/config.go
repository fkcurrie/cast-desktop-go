package stream

import (
	"encoding/json"
	"os"
	"path/filepath"
)

var configFile = filepath.Join(configDir, "config.json")

// Config holds persistent user preferences across casting sessions.
type Config struct {
	DefaultDevice string `json:"default_device,omitempty"`
	Display       string `json:"display,omitempty"`
	Resolution    string `json:"resolution,omitempty"`
	Fit           string `json:"fit,omitempty"`
	FPS           int    `json:"fps,omitempty"`
	Port          int    `json:"port,omitempty"`
	Preset        string `json:"preset,omitempty"`        // "balanced", "ultra-low", "cinema"
	HWAccel       string `json:"hw_accel,omitempty"`      // "auto", "vaapi", "nvenc", "none"
	Border        bool   `json:"border"`                  // privacy border overlay
	AudioOnly     bool   `json:"audio_only,omitempty"`    // stream audio only
	AudioSource   string `json:"audio_source,omitempty"`  // pulse/pipewire audio source
	Monitor       string `json:"monitor,omitempty"`       // monitor name or index
}

// DefaultConfig returns baseline configuration values.
func DefaultConfig() *Config {
	return &Config{
		DefaultDevice: "Bigscreen Basement TV",
		Display:       ":20",
		Resolution:    "1080p",
		Fit:           "pad",
		FPS:           30,
		Port:          8088,
		Preset:        "balanced",
		HWAccel:       "auto",
		Border:        true,
		AudioOnly:     false,
		AudioSource:   "default",
	}
}

// LoadConfig reads user preferences from ~/.config/cast-desktop-go/config.json.
// If the file does not exist, it returns default configuration.
func LoadConfig() *Config {
	cfg := DefaultConfig()
	data, err := os.ReadFile(configFile)
	if err == nil {
		_ = json.Unmarshal(data, cfg)
	}
	return cfg
}

// SaveConfig persists user preferences to disk.
func SaveConfig(cfg *Config) error {
	_ = os.MkdirAll(configDir, 0700)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0600)
}
