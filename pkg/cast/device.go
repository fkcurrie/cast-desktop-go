package cast

// Device represents a discovered Google Cast device.
type Device struct {
	Name    string `json:"name"`
	IP      string `json:"ip"`
	Port    int    `json:"port"`
	Model   string `json:"model"`
	UUID    string `json:"uuid"`
	Version string `json:"version"`
}
