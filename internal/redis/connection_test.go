package redis

import (
	"strconv"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

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

func TestConnect(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := NewClient()
	port, _ := strconv.Atoi(mr.Port())

	if err := client.Connect(mr.Host(), port, "", 0); err != nil {
		t.Fatalf("Connect() returned error: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })

	// Verify the client is usable by setting and getting a key
	mr.Set("testkey", "testval")
	got, err := client.client.Get(client.ctx, "testkey").Result()
	if err != nil {
		t.Fatalf("Get after Connect failed: %v", err)
	}
	if got != "testval" {
		t.Errorf("Get = %q, want %q", got, "testval")
	}
}

func TestConnectWithTLS_NonTLSServer(t *testing.T) {
	// Connecting with TLS to a non-TLS miniredis should fail
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := NewClient()
	port, _ := strconv.Atoi(mr.Port())

	tlsCfg := &types.TLSConfig{InsecureSkipVerify: true}
	goTLS, _ := tlsCfg.BuildTLSConfig()

	err = client.ConnectWithTLS(mr.Host(), port, "", 0, goTLS)
	if err == nil {
		_ = client.Disconnect()
		t.Fatal("expected error when connecting with TLS to non-TLS server")
	}
}

func TestConnect_WrongPassword(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	mr.RequireAuth("correct-password")

	client := NewClient()
	port, _ := strconv.Atoi(mr.Port())

	err = client.Connect(mr.Host(), port, "wrong-password", 0)
	if err == nil {
		_ = client.Disconnect()
		t.Fatal("expected error when connecting with wrong password")
	}
}

func TestConnect_CorrectPassword(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	mr.RequireAuth("correct-password")

	client := NewClient()
	port, _ := strconv.Atoi(mr.Port())

	err = client.Connect(mr.Host(), port, "correct-password", 0)
	if err != nil {
		t.Fatalf("Connect() with correct password returned error: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })
}

func TestDisconnect(t *testing.T) {
	t.Run("disconnect after connect", func(t *testing.T) {
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to start miniredis: %v", err)
		}
		t.Cleanup(mr.Close)

		client := NewClient()
		port, _ := strconv.Atoi(mr.Port())

		if err := client.Connect(mr.Host(), port, "", 0); err != nil {
			t.Fatalf("Connect() returned error: %v", err)
		}

		if err := client.Disconnect(); err != nil {
			t.Fatalf("Disconnect() returned error: %v", err)
		}

		if client.client != nil {
			t.Error("client.client should be nil after Disconnect")
		}
		if client.isCluster {
			t.Error("client.isCluster should be false after Disconnect")
		}
	})

	t.Run("disconnect on fresh client", func(t *testing.T) {
		client := NewClient()
		if err := client.Disconnect(); err != nil {
			t.Errorf("Disconnect() on fresh client returned error: %v", err)
		}
	})
}

func TestSelectDB(t *testing.T) {
	t.Run("select db succeeds", func(t *testing.T) {
		client, _ := setupTestClient(t)

		if err := client.SelectDB(1); err != nil {
			t.Fatalf("SelectDB(1) returned error: %v", err)
		}
		if client.db != 1 {
			t.Errorf("client.db = %d, want 1", client.db)
		}
	})

	t.Run("select db on cluster returns error", func(t *testing.T) {
		client, _ := setupTestClient(t)
		client.isCluster = true

		err := client.SelectDB(1)
		if err == nil {
			t.Fatal("SelectDB on cluster should return error")
		}
		if err.Error() != "database selection not supported in cluster mode" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("select db on nil client returns error", func(t *testing.T) {
		client := NewClient()

		err := client.SelectDB(1)
		if err == nil {
			t.Fatal("SelectDB on nil client should return error")
		}
		if err.Error() != "not connected" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestCleanup(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := NewClient()
	port, _ := strconv.Atoi(mr.Port())

	if err := client.Connect(mr.Host(), port, "", 0); err != nil {
		t.Fatalf("Connect() returned error: %v", err)
	}

	client.cleanup()

	if client.client != nil {
		t.Error("client.client should be nil after cleanup")
	}
}

func TestReconnectCycle(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := NewClient()
	port, _ := strconv.Atoi(mr.Port())

	// First connect
	if err := client.Connect(mr.Host(), port, "", 0); err != nil {
		t.Fatalf("first Connect() returned error: %v", err)
	}

	// Disconnect
	if err := client.Disconnect(); err != nil {
		t.Fatalf("Disconnect() returned error: %v", err)
	}

	// Reconnect
	if err := client.Connect(mr.Host(), port, "", 0); err != nil {
		t.Fatalf("second Connect() returned error: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })

	// Verify it works after reconnect
	mr.Set("reconnectkey", "reconnectval")
	got, err := client.client.Get(client.ctx, "reconnectkey").Result()
	if err != nil {
		t.Fatalf("Get after reconnect failed: %v", err)
	}
	if got != "reconnectval" {
		t.Errorf("Get = %q, want %q", got, "reconnectval")
	}
}
