package stream

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const StreamDir = "/dev/shm/cast_desktop_go"

// Server wraps the HTTP server providing in-memory HLS streaming.
type Server struct {
	httpServer *http.Server
	Port       int
	logFile    *os.File
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

// NewServer creates and starts an HTTP server serving the stream directory.
func NewServer(port int) (*Server, error) {
	if err := PrepareStreamDir(); err != nil {
		return nil, fmt.Errorf("failed to prepare stream dir: %w", err)
	}

	httpLogPath := filepath.Join(os.Getenv("HOME"), ".config", "cast-desktop-go", "http.log")
	_ = os.MkdirAll(filepath.Dir(httpLogPath), 0755)
	logFile, _ := os.OpenFile(httpLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir(StreamDir))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

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
			fmt.Print(logMsg)
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
		fmt.Print(logMsg)
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		_ = srv.ListenAndServe()
	}()

	return &Server{
		httpServer: srv,
		Port:       port,
		logFile:    logFile,
	}, nil
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
