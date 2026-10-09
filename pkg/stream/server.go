package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

const StreamDir = "/dev/shm/cast_desktop_go"

// RemoteHandler provides callbacks for the Web Remote and API endpoints.
type RemoteHandler struct {
	GetStatus    func() map[string]interface{}
	SetVolume    func(vol float64) error
	SetMuted     func(muted bool) error
	ToggleMute   func() (bool, error)
	Pause        func() error
	Play         func() error
	Stop         func() error
	GetDevices   func() ([]string, error)
	SwitchDevice func(device string) error
}

// Server wraps the HTTP server providing in-memory HLS streaming and web remote control.
type Server struct {
	httpServer    *http.Server
	Port          int
	LocalIP       string
	logFile       *os.File
	remoteHandler *RemoteHandler
}

// PrepareStreamDir ensures the ramdisk directory exists and is clean.
func PrepareStreamDir() error {
	_ = os.RemoveAll(StreamDir)
	return os.MkdirAll(StreamDir, 0755)
}

// CleanStreamDir removes any leftover files in the stream directory.
func CleanStreamDir() {
	_ = os.RemoveAll(StreamDir)
}

// statusResponseWriter records the HTTP response status code.
type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// SetRemoteHandler attaches the interactive remote callbacks to the server.
func (s *Server) SetRemoteHandler(h *RemoteHandler) {
	s.remoteHandler = h
}

// NewServer creates and starts an HTTP server serving both the HLS stream directory and the Web Remote.
func NewServer(port int, localIP string) (*Server, error) {
	if err := PrepareStreamDir(); err != nil {
		return nil, fmt.Errorf("failed to prepare stream dir: %w", err)
	}

	if localIP == "" {
		localIP = "127.0.0.1"
	}

	httpLogPath := filepath.Join(os.Getenv("HOME"), ".config", "cast-desktop-go", "http.log")
	_ = os.MkdirAll(filepath.Dir(httpLogPath), 0755)
	logFile, _ := os.OpenFile(httpLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

	s := &Server{
		Port:    port,
		LocalIP: localIP,
		logFile: logFile,
	}

	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir(StreamDir))

	// Web Remote UI
	mux.HandleFunc("/remote", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(RemoteHTML))
	})

	// QR Code for Remote UI
	mux.HandleFunc("/qrcode", func(w http.ResponseWriter, r *http.Request) {
		targetURL := fmt.Sprintf("http://%s:%d/remote", s.LocalIP, s.Port)
		pngBytes, err := qrcode.Encode(targetURL, qrcode.Medium, 256)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(pngBytes)
	})

	// API Status
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var st map[string]interface{}
		if s.remoteHandler != nil && s.remoteHandler.GetStatus != nil {
			st = s.remoteHandler.GetStatus()
		} else {
			st = map[string]interface{}{"active": true}
		}
		st["remote_url"] = fmt.Sprintf("http://%s:%d/remote", s.LocalIP, s.Port)
		_ = json.NewEncoder(w).Encode(st)
	})

	// API Volume
	mux.HandleFunc("/api/volume", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if s.remoteHandler == nil || s.remoteHandler.SetVolume == nil {
			http.Error(w, "Remote handler not configured", http.StatusServiceUnavailable)
			return
		}

		var volVal float64 = -1
		// Try parsing JSON body
		var payload struct {
			Volume float64 `json:"volume"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
			volVal = payload.Volume
		} else if q := r.URL.Query().Get("vol"); q != "" {
			if v, err := strconv.ParseFloat(q, 64); err == nil {
				volVal = v
			}
		}

		if volVal < 0 || volVal > 100 {
			http.Error(w, "Volume must be between 0 and 100", http.StatusBadRequest)
			return
		}

		if err := s.remoteHandler.SetVolume(volVal); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// API Mute
	mux.HandleFunc("/api/mute", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if s.remoteHandler == nil {
			http.Error(w, "Remote handler not configured", http.StatusServiceUnavailable)
			return
		}

		var payload struct {
			Muted *bool `json:"muted"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)

		var err error
		if payload.Muted != nil && s.remoteHandler.SetMuted != nil {
			err = s.remoteHandler.SetMuted(*payload.Muted)
		} else if s.remoteHandler.ToggleMute != nil {
			_, err = s.remoteHandler.ToggleMute()
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// API Pause
	mux.HandleFunc("/api/pause", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if s.remoteHandler == nil || s.remoteHandler.Pause == nil {
			http.Error(w, "Remote handler not configured", http.StatusServiceUnavailable)
			return
		}
		if err := s.remoteHandler.Pause(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// API Play
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if s.remoteHandler == nil || s.remoteHandler.Play == nil {
			http.Error(w, "Remote handler not configured", http.StatusServiceUnavailable)
			return
		}
		if err := s.remoteHandler.Play(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// API Stop
	mux.HandleFunc("/api/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if s.remoteHandler == nil || s.remoteHandler.Stop == nil {
			http.Error(w, "Remote handler not configured", http.StatusServiceUnavailable)
			return
		}
		go func() {
			time.Sleep(200 * time.Millisecond)
			_ = s.remoteHandler.Stop()
		}()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"stopping"}`))
	})

	// API Devices
	mux.HandleFunc("/api/devices", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var devs []string
		if s.remoteHandler != nil && s.remoteHandler.GetDevices != nil {
			var err error
			devs, err = s.remoteHandler.GetDevices()
			if err != nil {
				devs = []string{}
			}
		}
		_ = json.NewEncoder(w).Encode(devs)
	})

	// API Switch
	mux.HandleFunc("/api/switch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if s.remoteHandler == nil || s.remoteHandler.SwitchDevice == nil {
			http.Error(w, "Remote handler not configured", http.StatusServiceUnavailable)
			return
		}
		var payload struct {
			Device string `json:"device"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Device == "" {
			http.Error(w, "Invalid device parameter", http.StatusBadRequest)
			return
		}
		go func() {
			_ = s.remoteHandler.SwitchDevice(payload.Device)
		}()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"switching"}`))
	})

	// Main Stream & Static handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// If a browser visits the root URL directly, redirect to /remote
		if (r.URL.Path == "/" || r.URL.Path == "") && strings.Contains(r.Header.Get("Accept"), "text/html") {
			http.Redirect(w, r, "/remote", http.StatusTemporaryRedirect)
			return
		}

		// CORS and Private Network Access headers for Chromecast Shaka Player
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Private-Network", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			logMsg := fmt.Sprintf("[%s] %s OPTIONS %s -> 200 (preflight)\n", time.Now().Format("15:04:05.000"), r.RemoteAddr, r.URL.Path)
			if logFile != nil {
				_, _ = logFile.WriteString(logMsg)
			}
			return
		}

		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, ".m3u8"):
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		case strings.HasSuffix(path, ".m4s"):
			w.Header().Set("Content-Type", "video/iso.segment")
		case strings.HasSuffix(path, ".mp4"):
			w.Header().Set("Content-Type", "video/mp4")
		case strings.HasSuffix(path, ".ts"):
			w.Header().Set("Content-Type", "video/MP2T")
		}

		sw := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		fileServer.ServeHTTP(sw, r)

		duration := time.Since(start)
		logMsg := fmt.Sprintf("[%s] %s %s %s -> %d (%v)\n",
			time.Now().Format("15:04:05.000"),
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			sw.statusCode,
			duration.Round(time.Millisecond),
		)
		if logFile != nil {
			_, _ = logFile.WriteString(logMsg)
		}
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	s.httpServer = srv

	go func() {
		_ = srv.ListenAndServe()
	}()

	return s, nil
}

// Stop shuts down the HTTP server gracefully.
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if s.logFile != nil {
		_ = s.logFile.Close()
	}
	CleanStreamDir()
	return s.httpServer.Shutdown(ctx)
}
