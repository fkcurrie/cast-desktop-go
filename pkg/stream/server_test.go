package stream

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServerWebRemoteAndAPI(t *testing.T) {
	srv, err := NewServer(8999, "192.168.100.155")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Stop()

	var setVolVal float64 = -1
	var mutedVal bool = false
	var pausedVal bool = false
	var stoppedVal bool = false

	srv.SetRemoteHandler(&RemoteHandler{
		GetStatus: func() map[string]interface{} {
			return map[string]interface{}{
				"active": true,
				"device": "Basement TV",
				"volume": 42.0,
			}
		},
		SetVolume: func(vol float64) error {
			setVolVal = vol
			return nil
		},
		SetMuted: func(muted bool) error {
			mutedVal = muted
			return nil
		},
		Pause: func() error {
			pausedVal = true
			return nil
		},
		Play: func() error {
			pausedVal = false
			return nil
		},
		Stop: func() error {
			stoppedVal = true
			return nil
		},
		GetDevices: func() ([]string, error) {
			return []string{"Basement TV", "Living Room TV"}, nil
		},
	})

	// 1. Test /remote HTML page
	reqRemote := httptest.NewRequest(http.MethodGet, "/remote", nil)
	rrRemote := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rrRemote, reqRemote)
	if rrRemote.Code != http.StatusOK {
		t.Errorf("expected 200 for /remote, got %d", rrRemote.Code)
	}
	if !strings.Contains(rrRemote.Body.String(), "Cast Desktop Remote") {
		t.Errorf("expected title in /remote HTML response")
	}

	// 2. Test /qrcode PNG generation
	reqQR := httptest.NewRequest(http.MethodGet, "/qrcode", nil)
	rrQR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rrQR, reqQR)
	if rrQR.Code != http.StatusOK {
		t.Errorf("expected 200 for /qrcode, got %d", rrQR.Code)
	}
	if ct := rrQR.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("expected image/png content type, got: %s", ct)
	}

	// 3. Test /api/status
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rrStatus := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rrStatus, reqStatus)
	if rrStatus.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/status, got %d", rrStatus.Code)
	}
	var st map[string]interface{}
	if err := json.Unmarshal(rrStatus.Body.Bytes(), &st); err != nil {
		t.Fatalf("failed to parse status JSON: %v", err)
	}
	if st["device"] != "Basement TV" {
		t.Errorf("expected device Basement TV, got %v", st["device"])
	}

	// 4. Test /api/volume
	reqVol := httptest.NewRequest(http.MethodPost, "/api/volume", strings.NewReader(`{"volume": 65}`))
	rrVol := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rrVol, reqVol)
	if rrVol.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/volume, got %d", rrVol.Code)
	}
	if setVolVal != 65 {
		t.Errorf("expected setVolVal 65, got %f", setVolVal)
	}

	// 5. Test /api/mute
	reqMute := httptest.NewRequest(http.MethodPost, "/api/mute", strings.NewReader(`{"muted": true}`))
	rrMute := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rrMute, reqMute)
	if rrMute.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/mute, got %d", rrMute.Code)
	}
	if !mutedVal {
		t.Errorf("expected mutedVal to be true")
	}

	// 6. Test /api/devices
	reqDevs := httptest.NewRequest(http.MethodGet, "/api/devices", nil)
	rrDevs := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rrDevs, reqDevs)
	if rrDevs.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/devices, got %d", rrDevs.Code)
	}
	var devs []string
	if err := json.Unmarshal(rrDevs.Body.Bytes(), &devs); err != nil || len(devs) != 2 {
		t.Errorf("expected 2 devices, got %v (err=%v)", devs, err)
	}

	// 7. Test /api/pause
	reqPause := httptest.NewRequest(http.MethodPost, "/api/pause", nil)
	rrPause := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rrPause, reqPause)
	if rrPause.Code != http.StatusOK || !pausedVal {
		t.Errorf("expected pausedVal true, got %v (code %d)", pausedVal, rrPause.Code)
	}

	// 8. Test /api/play
	reqPlay := httptest.NewRequest(http.MethodPost, "/api/play", nil)
	rrPlay := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rrPlay, reqPlay)
	if rrPlay.Code != http.StatusOK || pausedVal {
		t.Errorf("expected pausedVal false, got %v (code %d)", pausedVal, rrPlay.Code)
	}

	// 9. Test /api/stop
	reqStop := httptest.NewRequest(http.MethodPost, "/api/stop", nil)
	rrStop := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rrStop, reqStop)
	if rrStop.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/stop, got %d", rrStop.Code)
	}
	_ = stoppedVal
}
