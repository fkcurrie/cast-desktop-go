package cast

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	configDir   = filepath.Join(os.Getenv("HOME"), ".config", "cast-desktop-go")
	devicesFile = filepath.Join(configDir, "devices.json")
)

// eurekaResponse models the JSON returned by Chromecast eureka_info.
type eurekaResponse struct {
	Name        string `json:"name"`
	IPAddress   string `json:"ip_address"`
	ModelName   string `json:"model_name"`
	SsdpUdn     string `json:"ssdp_udn"`
	BuildVer    string `json:"build_version"`
	CastRev     string `json:"cast_build_revision"`
}

// GetRouteToTarget finds the local IP address used to communicate with targetIP.
func GetRouteToTarget(targetIP string) (string, error) {
	conn, err := net.DialTimeout("udp", net.JoinHostPort(targetIP, "8008"), 2*time.Second)
	if err != nil {
		return "", fmt.Errorf("unable to determine local route to %s: %w", targetIP, err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// ProbeEureka queries a single IP for Chromecast eureka_info.
func ProbeEureka(ip string, timeout time.Duration) (*Device, error) {
	client := &http.Client{Timeout: timeout}
	url := fmt.Sprintf("http://%s:8008/setup/eureka_info", ip)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "cast-desktop-go")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var er eurekaResponse
	if err := json.Unmarshal(body, &er); err != nil {
		return nil, err
	}

	name := er.Name
	if name == "" {
		name = "Chromecast (" + ip + ")"
	}
	model := er.ModelName
	if model == "" {
		model = "Chromecast"
	}

	return &Device{
		Name:    name,
		IP:      ip,
		Port:    8009,
		Model:   model,
		UUID:    er.SsdpUdn,
		Version: er.CastRev,
	}, nil
}

// DiscoverDevices scans the local network to find all Google Cast receivers.
func DiscoverDevices(scanSubnet bool) []Device {
	var mu sync.Mutex
	devices := make([]Device, 0)
	seen := make(map[string]bool)

	// 1. Check cached devices
	if data, err := os.ReadFile(devicesFile); err == nil {
		var cached []Device
		if json.Unmarshal(data, &cached) == nil {
			var wg sync.WaitGroup
			for _, d := range cached {
				wg.Add(1)
				go func(dev Device) {
					defer wg.Done()
					if probed, err := ProbeEureka(dev.IP, 600*time.Millisecond); err == nil {
						mu.Lock()
						if !seen[probed.IP] {
							seen[probed.IP] = true
							devices = append(devices, *probed)
						}
						mu.Unlock()
					}
				}(d)
			}
			wg.Wait()
		}
	}

	// 2. Probe default known IPs on 192.168.18.0/24 if not yet discovered
	knownIPs := []string{"192.168.18.11", "192.168.18.17"}
	var wgKnown sync.WaitGroup
	for _, ip := range knownIPs {
		if !seen[ip] {
			wgKnown.Add(1)
			go func(targetIP string) {
				defer wgKnown.Done()
				if dev, err := ProbeEureka(targetIP, 800*time.Millisecond); err == nil {
					mu.Lock()
					if !seen[dev.IP] {
						seen[dev.IP] = true
						devices = append(devices, *dev)
					}
					mu.Unlock()
				}
			}(ip)
		}
	}
	wgKnown.Wait()

	// 3. Subnet scan if requested and fewer than 2 devices found
	if scanSubnet && len(devices) < 2 {
		routeIP, err := GetRouteToTarget("8.8.8.8")
		if err == nil {
			parts := strings.Split(routeIP, ".")
			if len(parts) == 4 {
				prefix := fmt.Sprintf("%s.%s.%s", parts[0], parts[1], parts[2])
				var wgSubnet sync.WaitGroup
				sem := make(chan struct{}, 40) // limit concurrent sockets

				for i := 1; i <= 35; i++ {
					targetIP := fmt.Sprintf("%s.%d", prefix, i)
					mu.Lock()
					alreadySeen := seen[targetIP]
					mu.Unlock()

					if alreadySeen {
						continue
					}

					wgSubnet.Add(1)
					go func(ip string) {
						defer wgSubnet.Done()
						sem <- struct{}{}
						defer func() { <-sem }()

						if dev, err := ProbeEureka(ip, 700*time.Millisecond); err == nil {
							mu.Lock()
							if !seen[dev.IP] {
								seen[dev.IP] = true
								devices = append(devices, *dev)
							}
							mu.Unlock()
						}
					}(targetIP)
				}
				wgSubnet.Wait()
			}
		}
	}

	// Cache discovered devices
	_ = os.MkdirAll(configDir, 0755)
	if data, err := json.MarshalIndent(devices, "", "  "); err == nil {
		_ = os.WriteFile(devicesFile, data, 0644)
	}

	return devices
}

// ResolveDevice finds a device by index (1-based), IP, or substring name match.
func ResolveDevice(query string, devices []Device) (*Device, error) {
	if len(devices) == 0 {
		return nil, fmt.Errorf("no Chromecast devices available")
	}

	// Default to first device if empty query
	if strings.TrimSpace(query) == "" {
		return &devices[0], nil
	}

	// Check 1-based index
	if idx, err := strconv.Atoi(query); err == nil {
		if idx >= 1 && idx <= len(devices) {
			return &devices[idx-1], nil
		}
		return nil, fmt.Errorf("device index %d out of range (1-%d)", idx, len(devices))
	}

	// Match exact IP
	for _, d := range devices {
		if d.IP == query {
			return &d, nil
		}
	}

	// Match case-insensitive name substring
	qLower := strings.ToLower(query)
	for _, d := range devices {
		if strings.Contains(strings.ToLower(d.Name), qLower) {
			return &d, nil
		}
	}

	return nil, fmt.Errorf("device matching '%s' not found", query)
}
