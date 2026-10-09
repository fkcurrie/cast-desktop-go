# cast-desktop-go

High-performance Go CLI utility for casting the Linux desktop and audio to Chromecast devices on the local network (such as Living room TV and Bigscreen Basement TV).

## Features

- **Interactive Device Selection**: Seamlessly prompts you to pick which Chromecast to cast to if multiple devices are found on the network.
- **Fast Parallel Subnet Discovery**: Probes Google Cast endpoints across the subnet in parallel with goroutines.
- **Low-Latency Screen & Audio Capture**: Captures X11 desktop (`:20` or `$DISPLAY`) and PipeWire PulseAudio monitor using FFmpeg.
- **In-Memory HLS Streaming**: Serves live HLS chunks directly from ramdisk (`/dev/shm/cast_desktop_go`) with CORS headers.
- **Background & Foreground Modes**: Run interactively with `Ctrl+C` stop or detached in background with `-b` / `--daemon`.
- **Clean Release**: Sends stop command to the Chromecast upon exit to return the TV to its ambient backdrop.

## Installation

The binary is installed to `~/.local/bin/cast-desktop-go` (and symlinked as `cast-go`):

```bash
# Rebuild & install if modified:
cd ~/Projects/cast-desktop-go
go build -o ~/.local/bin/cast-desktop-go main.go
ln -sf ~/.local/bin/cast-desktop-go ~/.local/bin/cast-go
```

## Usage

### 1. List Available Chromecasts
```bash
cast-go list
```

### 2. Interactive Casting
Running `cast-go` or `cast-go start` without flags will discover devices and prompt you to pick:
```
$ cast-go
Scanning local network for Chromecast devices...

Discovered 2 Chromecast devices:
  [1] Living room TV (192.168.18.11)
  [2] Bigscreen Basement TV (192.168.18.17)

Select a device [1-2] (default: 1): 2
```

### 3. Specify Target Device Directly
You can specify device by name, IP, or index number:
```bash
# By name
cast-go start -d "Living room TV"
cast-go start -d "Basement"

# By index
cast-go start -d 1

# By IP
cast-go start -d 192.168.18.17
```

### 4. Background / Daemon Mode
```bash
# Start casting in the background
cast-go start -d "Bigscreen Basement TV" -b

# Check status
cast-go status

# Stop casting session and return TV to backdrop
cast-go stop
```

### 5. Custom Options
```bash
# 60 fps capture
cast-go start -d 1 --fps 60

# 720p scaled output
cast-go start -d 1 --res 720p

# Custom X11 display
cast-go start -d 1 --display :0
```
