# Cast Desktop (`cast-desktop-go`)

[![Release](https://img.shields.io/github/v/release/fkcurrie/cast-desktop-go?color=blue&logo=github)](https://github.com/fkcurrie/cast-desktop-go/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/fkcurrie/cast-desktop-go)](https://goreportcard.com/report/github.com/fkcurrie/cast-desktop-go)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20Ubuntu%20%7C%20Mint-orange)](https://github.com/fkcurrie/cast-desktop-go)

High-performance utility for streaming the Linux desktop display and PipeWire/PulseAudio audio to Google Chromecast, Android TV, Google TV, and Google Home smart speakers over low-latency Fragmented MP4 (fMP4/CMAF) HTTP Live Streaming.

---

## Features

- ⚡ **Low-Latency fMP4/CMAF Streaming**: Uses Fragmented MP4 (`init.mp4` + `.m4s`) live chunking compatible with modern Chromecast with Google TV firmware, Chromecast Ultra, and Shaka Player.
- 🚀 **Hardware-Accelerated Encoding (VA-API)**: Offload H.264 video compression to AMD Radeon and Intel GPUs via `/dev/dri/renderD128`, reducing CPU usage to near zero.
- 🎛️ **System Tray / StatusNotifierItem (D-Bus)**: Native desktop tray icon with live status, mouse wheel volume scrolling ($\pm 5\%$), middle-click mute toggle, and full right-click context menu.
- 📱 **Mobile Web Remote & Scannable QR Code**: Touch-friendly web dashboard hosted at `http://<host-ip>:8088/remote` with volume sliders, mute, pause/play, device switching, and terminal ANSI QR code.
- ⏱️ **Latency Presets**:
  - `ultra-low` (0.5s segments): Near real-time responsiveness for general desktop tasks.
  - `balanced` (1.0s segments): Smooth desktop streaming (default).
  - `cinema` (2.0s segments): Enhanced buffering for stutter-free video and movie playback.
- 🖥️ **Multi-Monitor & Single Window Capture**:
  - Select specific physical monitors (`--monitor 0` or `--monitor HDMI-A-0`).
  - Target single application windows (`--window "Firefox"` or `--window 0x03800007`).
- 🔊 **Audio-Only Streaming**: Route desktop audio directly to Google Home, Nest Audio, and Home Mini speakers without video overhead (`--audio-only`).
- 📺 **TV Volume & Playback Controls**: Control TV volume, toggle mute, pause, and resume playback directly from CLI, D-Bus, or the Web Remote.
- 🛡️ **Visual Privacy Border Overlay**: Click-through glowing border framing the captured screen or window area with zero mouse interference.
- ⌨️ **KDE Plasma & Cinnamon Shortcuts**: Integrated global keybindings (`Meta+Shift+C`, `Meta+Shift+X`, etc.) and right-click desktop launcher actions.

---

## Installation

### Method 1: Install Debian Package (Ubuntu & Linux Mint)

Download and install the latest `.deb` release:

```bash
wget https://github.com/fkcurrie/cast-desktop-go/releases/download/v1.2.0/cast-desktop-go_1.2.0_amd64.deb
sudo apt install ./cast-desktop-go_1.2.0_amd64.deb
```

### Method 2: Build from Source

**Prerequisites**: Go 1.21+, `ffmpeg`, `libnotify-bin`, `python3-gi`, `python3-cairo`.

```bash
git clone https://github.com/fkcurrie/cast-desktop-go.git
cd cast-desktop-go

# Build binary
go build -ldflags="-s -w" -o bin/cast-desktop-go main.go

# Install locally
sudo install -m 0755 bin/cast-desktop-go /usr/local/bin/
sudo ln -sf /usr/local/bin/cast-desktop-go /usr/local/bin/cast-go
```

---

## Usage

### 1. Launch Interactive GUI or Menu
Click **Cast Desktop** in your application launcher (under *Sound & Video* / *Multimedia*), or run:
```bash
cast-desktop-go gui
```

### 2. Quick Cast from CLI
Run `cast-desktop-go` or `cast-go`:
```bash
cast-go
```
If multiple Chromecasts are found, you will be prompted with an interactive picker:
```text
Scanning local network for Chromecast devices...

Discovered 2 Chromecast devices:
  [1] Living room TV (192.168.18.11)
  [2] Bigscreen Basement TV (192.168.18.17)

Select a device [1-2] (default: 1): 2
```

### 3. Target Specific Device
Target by name substring, IP, or index:
```bash
cast-go start -d "Basement"
cast-go start -d 192.168.18.17
```

### 4. Background / Daemon Mode
```bash
# Start detached in the background
cast-go start -d "Living room TV" -b

# Check active streaming status
cast-go status

# Stop casting session and return TV to backdrop
cast-go stop
```

### 5. Advanced Flags & Modes

```bash
# Hardware-accelerated encoding (AMD/Intel VA-API)
cast-go start --hw-accel vaapi

# Ultra-low latency preset (0.5s segments)
cast-go start --preset ultra-low

# Capture specific monitor in multi-head setup
cast-go start --monitor 0

# Capture only a single application window
cast-go start --window "Firefox"

# Audio-only streaming to Google Home / Nest Audio speakers
cast-go start -d "Living room speaker" --audio-only

# Disable visual glowing border overlay
cast-go start --no-border
```

### 6. TV Volume & Playback Controls

```bash
# View current TV volume
cast-go volume

# Set volume percentage (0-100)
cast-go volume 45

# Relative adjustment
cast-go volume +5
cast-go volume -5

# Toggle audio mute
cast-go mute

# Playback controls
cast-go pause
cast-go play
```

### 7. Keyboard Shortcuts (KDE Plasma & Desktop)

```bash
# View configured shortcuts
cast-go shortcuts list

# Install shortcuts (Meta+Shift+C, Meta+Shift+X, Meta+Shift+M, Meta+Shift+Up/Down)
cast-go shortcuts install
```

---

## Mobile Web Remote

When casting starts, a local HTTP server is launched (default: port 8088).

- Terminal displays a scannable **ANSI QR code** on startup.
- Visit `http://<host-ip>:8088/remote` in any mobile browser to adjust TV volume, mute, pause, or switch target devices.
- Scannable PNG QR code available at `http://<host-ip>:8088/qrcode`.

---

## License

This project is licensed under the [MIT License](LICENSE).
