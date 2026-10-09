package cast

import (
	"fmt"
	"time"

	"github.com/vishen/go-chromecast/application"
	chromecast "github.com/vishen/go-chromecast/cast"
)

// liveCastConn wraps the Chromecast TLS connection to inject StreamType = "LIVE"
// on LoadMediaCommand, ensuring the TV receiver treats the stream as a live broadcast
// rather than an on-demand video file, eliminating the bottom timeline scrubber bar.
type liveCastConn struct {
	*chromecast.Connection
}

func (c *liveCastConn) Send(requestID int, payload chromecast.Payload, sourceID, destinationID, namespace string) error {
	if cmd, ok := payload.(*chromecast.LoadMediaCommand); ok {
		cmd.Media.StreamType = "LIVE"
	}
	return c.Connection.Send(requestID, payload, sourceID, destinationID, namespace)
}

// Controller manages the Chromecast session and playback commands.
type Controller struct {
	app       *application.Application
	dev       *Device
	connected bool
}

// NewController creates a new Controller instance for target device.
func NewController(dev *Device) *Controller {
	app := application.NewApplication()
	conn := &liveCastConn{Connection: chromecast.NewConnection()}
	app.SetConn(conn)
	return &Controller{
		app: app,
		dev: dev,
	}
}

// StartCast connects to the Chromecast, launches the media receiver, and streams the HLS URL.
func (c *Controller) StartCast(streamURL string) error {
	port := c.dev.Port
	if port == 0 {
		port = 8009
	}

	if err := c.app.Start(c.dev.IP, port); err != nil {
		return fmt.Errorf("failed to connect to Chromecast %s (%s:%d): %w", c.dev.Name, c.dev.IP, port, err)
	}
	c.connected = true

	// Give the cast channel a brief moment to settle
	time.Sleep(300 * time.Millisecond)

	// Fetch receiver state so go-chromecast knows active app status
	_ = c.app.Update()

	// Load stream URL: url, startTime=0, contentType, transcode=false, detach=true, forceDetach=false
	// Detach must be true for live continuous streaming so the call returns without waiting for EOF
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if err := c.app.Load(streamURL, 0, "application/x-mpegURL", false, true, false); err != nil {
			lastErr = err
			time.Sleep(1 * time.Second)
			_ = c.app.Update()
			continue
		}
		return nil
	}

	return fmt.Errorf("failed to launch desktop stream on %s: %w", c.dev.Name, lastErr)
}

// StopCast sends a stop command and disconnects from Chromecast.
func (c *Controller) StopCast() error {
	if c.connected && c.app != nil {
		_ = c.app.Stop()
		_ = c.app.Close(true)
		c.connected = false
		c.app = nil
		return nil
	}

	port := c.dev.Port
	if port == 0 {
		port = 8009
	}

	app := application.NewApplication()
	conn := &liveCastConn{Connection: chromecast.NewConnection()}
	app.SetConn(conn)
	if err := app.Start(c.dev.IP, port); err != nil {
		return fmt.Errorf("failed to connect to Chromecast to stop: %w", err)
	}
	defer app.Close(true)

	_ = app.Stop()
	return nil
}

// Close closes the active connection.
func (c *Controller) Close() {
	if c.connected && c.app != nil {
		_ = c.app.Close(false)
		c.connected = false
		c.app = nil
	}
}
