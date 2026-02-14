package redis

import "testing"

func TestParseAddr(t *testing.T) {
	tests := []struct {
		name         string
		addr         string
		expectedHost string
		expectedPort int
	}{
		{"host:port", "localhost:6379", "localhost", 6379},
		{"custom port", "myhost:6380", "myhost", 6380},
		{"hostname only no port", "hostname", "hostname", 6379},
		{"empty string", "", "", 6379},
		{"ip with port", "192.168.1.1:7000", "192.168.1.1", 7000},
		{"ip without port", "192.168.1.1", "192.168.1.1", 6379},
		{"port 0", "host:0", "host", 0},
		{"non-numeric port uses default", "host:abc", "host", 6379},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port := parseAddr(tt.addr)
			if host != tt.expectedHost {
				t.Errorf("parseAddr(%q) host = %q, want %q", tt.addr, host, tt.expectedHost)
			}
			if port != tt.expectedPort {
				t.Errorf("parseAddr(%q) port = %d, want %d", tt.addr, port, tt.expectedPort)
			}
		})
	}
}
