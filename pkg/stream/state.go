package stream

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

var (
	configDir = filepath.Join(os.Getenv("HOME"), ".config", "cast-desktop-go")
	stateFile = filepath.Join(configDir, "state.json")
)

// State models the active casting session.
type State struct {
	PID        int       `json:"pid"`
	DeviceName string    `json:"device_name"`
	DeviceIP   string    `json:"device_ip"`
	Display    string    `json:"display"`
	Port       int       `json:"port"`
	Preset     string    `json:"preset,omitempty"`
	HWAccel    bool      `json:"hw_accel"`
	AudioOnly  bool      `json:"audio_only"`
	Monitor    string    `json:"monitor,omitempty"`
	Window     string    `json:"window,omitempty"`
	RemoteURL  string    `json:"remote_url,omitempty"`
	StartedAt  time.Time `json:"started_at"`
}

// SaveState writes active session state to disk.
func SaveState(st *State) error {
	_ = os.MkdirAll(configDir, 0700)
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(stateFile, data, 0600)
}

// LoadState reads the active session state from disk.
func LoadState() (*State, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

// ClearState removes the session state file.
func ClearState() error {
	return os.Remove(stateFile)
}

// IsProcessRunning checks if the given PID is currently active.
func IsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}
