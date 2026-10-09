package cast

import (
	"testing"
)

func TestResolveDevice(t *testing.T) {
	devices := []Device{
		{Name: "Living room TV", IP: "192.168.18.11", Port: 8009, Model: "Chromecast"},
		{Name: "Bigscreen Basement TV", IP: "192.168.18.17", Port: 8009, Model: "Chromecast"},
	}

	tests := []struct {
		name      string
		query     string
		expected  string
		expectErr bool
	}{
		{"empty defaults to first", "", "Living room TV", false},
		{"index 1", "1", "Living room TV", false},
		{"index 2", "2", "Bigscreen Basement TV", false},
		{"exact IP 1", "192.168.18.11", "Living room TV", false},
		{"exact IP 2", "192.168.18.17", "Bigscreen Basement TV", false},
		{"substring living", "living", "Living room TV", false},
		{"substring basement", "basement", "Bigscreen Basement TV", false},
		{"substring Bigscreen", "Bigscreen", "Bigscreen Basement TV", false},
		{"invalid index", "99", "", true},
		{"non-existent name", "bedroom", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dev, err := ResolveDevice(tc.query, devices)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error for query '%s', got nil", tc.query)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for query '%s': %v", tc.query, err)
			}
			if dev.Name != tc.expected {
				t.Fatalf("expected '%s', got '%s'", tc.expected, dev.Name)
			}
		})
	}
}
