package stream

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigSaveLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cast_config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldConfigFile := configFile
	configFile = filepath.Join(tmpDir, "config.json")
	defer func() { configFile = oldConfigFile }()

	cfg := &Config{
		DefaultDevice: "Bigscreen Basement TV",
		Display:       ":20",
		Resolution:    "1080p",
		Fit:           "pad",
		FPS:           30,
		Port:          8088,
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loaded := LoadConfig()
	if loaded.DefaultDevice != cfg.DefaultDevice {
		t.Errorf("expected device %s, got %s", cfg.DefaultDevice, loaded.DefaultDevice)
	}
	if loaded.Fit != cfg.Fit {
		t.Errorf("expected fit %s, got %s", cfg.Fit, loaded.Fit)
	}
	if loaded.Resolution != cfg.Resolution {
		t.Errorf("expected resolution %s, got %s", cfg.Resolution, loaded.Resolution)
	}
}
