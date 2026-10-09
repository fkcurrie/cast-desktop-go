package tray

import (
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
)

// Callbacks holds the function handlers triggered by user interactions with the system tray.
type Callbacks struct {
	OnOpenRemote func()
	OnVolumeStep func(delta float64)
	OnToggleMute func()
	OnTogglePlay func()
	OnStop       func()
}

// TrayManager manages the D-Bus StatusNotifierItem and dbusmenu registration.
type TrayManager struct {
	conn       *dbus.Conn
	busName    string
	callbacks  Callbacks
	deviceName string
	port       int
	mu         sync.Mutex
	closed     bool
}

type sniItem struct {
	tm *TrayManager
}

func (s *sniItem) Activate(x, y int32) *dbus.Error {
	if s.tm.callbacks.OnOpenRemote != nil {
		s.tm.callbacks.OnOpenRemote()
	} else {
		_ = exec.Command("xdg-open", fmt.Sprintf("http://localhost:%d/remote", s.tm.port)).Start()
	}
	return nil
}

func (s *sniItem) SecondaryActivate(x, y int32) *dbus.Error {
	if s.tm.callbacks.OnToggleMute != nil {
		s.tm.callbacks.OnToggleMute()
	}
	return nil
}

func (s *sniItem) Scroll(delta int32, orientation string) *dbus.Error {
	if s.tm.callbacks.OnVolumeStep != nil {
		if delta > 0 {
			s.tm.callbacks.OnVolumeStep(0.05)
		} else if delta < 0 {
			s.tm.callbacks.OnVolumeStep(-0.05)
		}
	}
	return nil
}

func (s *sniItem) ContextMenu(x, y int32) *dbus.Error {
	return nil
}

// DBusMenu types
type dBusMenuLayout struct {
	ID         int32
	Properties map[string]dbus.Variant
	Children   []dbus.Variant
}

type dBusMenuGroupProperty struct {
	ID         int32
	Properties map[string]dbus.Variant
}

type menuServer struct {
	tm *TrayManager
}

func (m *menuServer) AboutToShow(id int32) (bool, *dbus.Error) {
	return false, nil
}

func (m *menuServer) GetLayout(parentId int32, recursionDepth int32, propertyNames []string) (uint32, dBusMenuLayout, *dbus.Error) {
	items := []struct {
		id      int32
		label   string
		enabled bool
	}{
		{id: 1, label: fmt.Sprintf("Streaming: %s", m.tm.deviceName), enabled: false},
		{id: 2, label: "Open Web Remote", enabled: true},
		{id: 3, label: "Volume Up (+5%)", enabled: true},
		{id: 4, label: "Volume Down (-5%)", enabled: true},
		{id: 5, label: "Toggle Mute", enabled: true},
		{id: 6, label: "Pause / Resume", enabled: true},
		{id: 7, label: "Stop Casting", enabled: true},
	}

	var children []dbus.Variant
	for _, it := range items {
		child := dBusMenuLayout{
			ID: it.id,
			Properties: map[string]dbus.Variant{
				"label":   dbus.MakeVariant(it.label),
				"enabled": dbus.MakeVariant(it.enabled),
				"visible": dbus.MakeVariant(true),
			},
			Children: []dbus.Variant{},
		}
		children = append(children, dbus.MakeVariant(child))
	}

	root := dBusMenuLayout{
		ID:         0,
		Properties: map[string]dbus.Variant{},
		Children:   children,
	}
	return 1, root, nil
}

func (m *menuServer) GetGroupProperties(ids []int32, propertyNames []string) ([]dBusMenuGroupProperty, *dbus.Error) {
	return []dBusMenuGroupProperty{}, nil
}

func (m *menuServer) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	return dbus.MakeVariant(""), nil
}

func (m *menuServer) Event(id int32, eventId string, data dbus.Variant, timestamp uint32) *dbus.Error {
	if eventId != "clicked" {
		return nil
	}

	switch id {
	case 2: // Open Web Remote
		if m.tm.callbacks.OnOpenRemote != nil {
			m.tm.callbacks.OnOpenRemote()
		} else {
			_ = exec.Command("xdg-open", fmt.Sprintf("http://localhost:%d/remote", m.tm.port)).Start()
		}
	case 3: // Vol Up
		if m.tm.callbacks.OnVolumeStep != nil {
			m.tm.callbacks.OnVolumeStep(0.05)
		}
	case 4: // Vol Down
		if m.tm.callbacks.OnVolumeStep != nil {
			m.tm.callbacks.OnVolumeStep(-0.05)
		}
	case 5: // Mute
		if m.tm.callbacks.OnToggleMute != nil {
			m.tm.callbacks.OnToggleMute()
		}
	case 6: // Play/Pause
		if m.tm.callbacks.OnTogglePlay != nil {
			m.tm.callbacks.OnTogglePlay()
		}
	case 7: // Stop
		if m.tm.callbacks.OnStop != nil {
			m.tm.callbacks.OnStop()
		}
	}
	return nil
}

// Start launches the D-Bus StatusNotifierItem and connects it to KDE Plasma / AppIndicator tray.
func Start(deviceName string, port int, cb Callbacks) (*TrayManager, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect session bus: %w", err)
	}

	busName := fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid())
	reply, err := conn.RequestName(busName, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to acquire bus name %s: %w", busName, err)
	}

	tm := &TrayManager{
		conn:       conn,
		busName:    busName,
		callbacks:  cb,
		deviceName: deviceName,
		port:       port,
	}

	// 1. Export StatusNotifierItem
	sni := &sniItem{tm: tm}
	if err := conn.Export(sni, "/StatusNotifierItem", "org.kde.StatusNotifierItem"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to export StatusNotifierItem: %w", err)
	}

	// 2. Export DBusMenu
	menu := &menuServer{tm: tm}
	if err := conn.Export(menu, "/MenuBar", "com.canonical.dbusmenu"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to export dbusmenu: %w", err)
	}

	// 3. Export SNI Properties
	propsMap := prop.Map{
		"org.kde.StatusNotifierItem": {
			"Category":      {Value: "ApplicationStatus"},
			"Id":            {Value: "cast-desktop-go"},
			"Title":         {Value: "Cast Desktop"},
			"Status":        {Value: "Active"},
			"IconName":      {Value: "cast-desktop-go"},
			"IconThemePath": {Value: "/usr/share/icons/hicolor/scalable/apps"},
			"ItemIsMenu":    {Value: false},
			"Menu":          {Value: dbus.ObjectPath("/MenuBar")},
			"WindowId":      {Value: int32(0)},
		},
	}

	if _, err := prop.Export(conn, "/StatusNotifierItem", propsMap); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to export SNI properties: %w", err)
	}

	// 4. Register with StatusNotifierWatcher
	watcherObj := conn.Object("org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher")
	call := watcherObj.Call("org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem", 0, busName)
	if call.Err != nil {
		// Non-fatal if watcher is absent (e.g. headless)
		fmt.Printf("[Tray] Warning: StatusNotifierWatcher registration returned: %v\n", call.Err)
	}

	return tm, nil
}

// Stop closes the D-Bus connection and unregisters the tray item.
func (tm *TrayManager) Stop() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tm.closed {
		return
	}
	tm.closed = true
	if tm.conn != nil {
		_ = tm.conn.Close()
	}
}
