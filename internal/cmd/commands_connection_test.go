package cmd

import (
	"errors"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/testutil"
	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestConnectionErrorPaths(t *testing.T) {
	t.Run("LoadConnections error", func(t *testing.T) {
		mc := testutil.NewMockConfigClient()
		mc.ListConnectionsError = errors.New("list failed")
		cmds := NewCommands(mc, nil)
		msg := cmds.LoadConnections()()
		result := msg.(types.ConnectionsLoadedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("AddConnection error", func(t *testing.T) {
		mc := testutil.NewMockConfigClient()
		mc.AddConnectionError = errors.New("add failed")
		cmds := NewCommands(mc, nil)
		msg := cmds.AddConnection("n", "h", 1, "", 0, false)()
		result := msg.(types.ConnectionAddedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("UpdateConnection error", func(t *testing.T) {
		mc := testutil.NewMockConfigClient()
		mc.UpdateConnectionError = errors.New("update failed")
		cmds := NewCommands(mc, nil)
		msg := cmds.UpdateConnection(1, "n", "h", 1, "", 0, false)()
		result := msg.(types.ConnectionUpdatedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})
}

func TestLoadConnections(t *testing.T) {
	t.Run("success empty", func(t *testing.T) {
		cfg := testutil.NewTestConfig(t)
		cmds := NewCommands(cfg, nil)
		msg := cmds.LoadConnections()()
		result, ok := msg.(types.ConnectionsLoadedMsg)
		if !ok {
			t.Fatalf("expected ConnectionsLoadedMsg, got %T", msg)
		}
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Connections) != 0 {
			t.Errorf("expected 0 connections, got %d", len(result.Connections))
		}
	})

	t.Run("success with connections", func(t *testing.T) {
		cfg := testutil.NewTestConfig(t)
		testutil.MustAddConnection(t, cfg, "test", "localhost", 6379, "", 0)
		cmds := NewCommands(cfg, nil)
		msg := cmds.LoadConnections()()
		result := msg.(types.ConnectionsLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Connections) != 1 {
			t.Errorf("expected 1 connection, got %d", len(result.Connections))
		}
	})

	t.Run("nil config", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.LoadConnections()()
		result := msg.(types.ConnectionsLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil config should not error: %v", result.Err)
		}
	})
}

func TestAddConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg := testutil.NewTestConfig(t)
		cmds := NewCommands(cfg, nil)
		msg := cmds.AddConnection("test", "localhost", 6379, "", 0, false)()
		result := msg.(types.ConnectionAddedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Connection.Name != "test" {
			t.Errorf("Name = %q, want %q", result.Connection.Name, "test")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.AddConnection("test", "localhost", 6379, "", 0, false)()
		result := msg.(types.ConnectionAddedMsg)
		if result.Err != nil {
			t.Errorf("nil config should not error: %v", result.Err)
		}
	})
}

func TestUpdateConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg := testutil.NewTestConfig(t)
		conn := testutil.MustAddConnection(t, cfg, "old", "localhost", 6379, "", 0)
		cmds := NewCommands(cfg, nil)
		msg := cmds.UpdateConnection(conn.ID, "new", "localhost", 6380, "pass", 1, false)()
		result := msg.(types.ConnectionUpdatedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Connection.Name != "new" {
			t.Errorf("Name = %q, want %q", result.Connection.Name, "new")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.UpdateConnection(1, "n", "h", 1, "p", 0, false)()
		result := msg.(types.ConnectionUpdatedMsg)
		if result.Err != nil {
			t.Errorf("nil config should not error: %v", result.Err)
		}
	})
}

func TestDeleteConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg := testutil.NewTestConfig(t)
		conn := testutil.MustAddConnection(t, cfg, "test", "localhost", 6379, "", 0)
		cmds := NewCommands(cfg, nil)
		msg := cmds.DeleteConnection(conn.ID)()
		result := msg.(types.ConnectionDeletedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.ID != conn.ID {
			t.Errorf("ID = %d, want %d", result.ID, conn.ID)
		}
	})

	t.Run("nil config", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.DeleteConnection(1)()
		result := msg.(types.ConnectionDeletedMsg)
		if result.Err != nil {
			t.Errorf("nil config should not error: %v", result.Err)
		}
	})
}

func TestConnect(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.Connect("localhost", 6379, "", 0, false)()
		result := msg.(types.ConnectedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ConnectError = errors.New("connection refused")
		msg := cmds.Connect("localhost", 6379, "", 0, false)()
		result := msg.(types.ConnectedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("cluster mode", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.Connect("localhost", 7000, "", 0, true)()
		result := msg.(types.ConnectedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.Connect("localhost", 6379, "", 0, false)()
		result := msg.(types.ConnectedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestAutoConnect(t *testing.T) {
	t.Run("success standard", func(t *testing.T) {
		cmds, mock := newMockCmds()
		conn := types.Connection{Host: "localhost", Port: 6379, DB: 0}
		msg := cmds.AutoConnect(conn)()
		result := msg.(types.ConnectedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(mock.Calls) == 0 || mock.Calls[0] != "Connect" {
			t.Errorf("expected Connect call, got %v", mock.Calls)
		}
	})

	t.Run("success cluster", func(t *testing.T) {
		cmds, mock := newMockCmds()
		conn := types.Connection{Host: "localhost", Port: 7000, UseCluster: true}
		msg := cmds.AutoConnect(conn)()
		result := msg.(types.ConnectedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(mock.Calls) == 0 || mock.Calls[0] != "ConnectCluster" {
			t.Errorf("expected ConnectCluster call, got %v", mock.Calls)
		}
	})

	t.Run("success TLS", func(t *testing.T) {
		cmds, mock := newMockCmds()
		conn := types.Connection{
			Host:   "localhost",
			Port:   6380,
			UseTLS: true,
			TLSConfig: &types.TLSConfig{
				InsecureSkipVerify: true,
			},
		}
		msg := cmds.AutoConnect(conn)()
		result := msg.(types.ConnectedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(mock.Calls) == 0 || mock.Calls[0] != "ConnectWithTLS" {
			t.Errorf("expected ConnectWithTLS call, got %v", mock.Calls)
		}
	})

	t.Run("TLS without config returns error", func(t *testing.T) {
		cmds, _ := newMockCmds()
		conn := types.Connection{Host: "localhost", Port: 6379, UseTLS: true}
		msg := cmds.AutoConnect(conn)()
		result := msg.(types.ConnectedMsg)
		if result.Err == nil {
			t.Error("expected error for TLS without config")
		}
		if result.Err.Error() != "TLS requested but TLS configuration is missing" {
			t.Errorf("unexpected error message: %v", result.Err)
		}
	})

	t.Run("TLS bad cert file", func(t *testing.T) {
		cmds, _ := newMockCmds()
		conn := types.Connection{
			Host:   "localhost",
			Port:   6380,
			UseTLS: true,
			TLSConfig: &types.TLSConfig{
				CertFile: "/nonexistent/cert.pem",
				KeyFile:  "/nonexistent/key.pem",
			},
		}
		msg := cmds.AutoConnect(conn)()
		result := msg.(types.ConnectedMsg)
		if result.Err == nil {
			t.Error("expected error for bad TLS cert file")
		}
	})

	t.Run("connect error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ConnectError = errors.New("connection refused")
		conn := types.Connection{Host: "localhost", Port: 6379}
		msg := cmds.AutoConnect(conn)()
		result := msg.(types.ConnectedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("cluster error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ConnectClusterError = errors.New("cluster unavailable")
		conn := types.Connection{Host: "localhost", Port: 7000, UseCluster: true}
		msg := cmds.AutoConnect(conn)()
		result := msg.(types.ConnectedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("TLS connect error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ConnectWithTLSError = errors.New("TLS handshake failed")
		conn := types.Connection{
			Host:      "localhost",
			Port:      6380,
			UseTLS:    true,
			TLSConfig: &types.TLSConfig{InsecureSkipVerify: true},
		}
		msg := cmds.AutoConnect(conn)()
		result := msg.(types.ConnectedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		conn := types.Connection{Host: "localhost", Port: 6379}
		msg := cmds.AutoConnect(conn)()
		result := msg.(types.ConnectedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestAutoConnect_ClusterWithTLS_IgnoresTLS(t *testing.T) {
	// Documents current behavior: when both UseCluster and UseTLS are set,
	// the cluster branch runs first and TLS is silently ignored.
	// This test guards against accidental changes and documents the gap.
	cmds, mock := newMockCmds()
	conn := types.Connection{
		Host:       "localhost",
		Port:       7000,
		UseCluster: true,
		UseTLS:     true,
		TLSConfig:  &types.TLSConfig{InsecureSkipVerify: true},
	}
	msg := cmds.AutoConnect(conn)()
	result := msg.(types.ConnectedMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
	// Cluster path takes priority — ConnectCluster is called, not ConnectWithTLS
	if len(mock.Calls) == 0 || mock.Calls[0] != "ConnectCluster" {
		t.Errorf("expected ConnectCluster call (cluster takes priority over TLS), got %v", mock.Calls)
	}
}

func TestDisconnect(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.Disconnect()()
		if _, ok := msg.(types.DisconnectedMsg); !ok {
			t.Fatalf("expected DisconnectedMsg, got %T", msg)
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.Disconnect()()
		if _, ok := msg.(types.DisconnectedMsg); !ok {
			t.Fatalf("expected DisconnectedMsg, got %T", msg)
		}
	})
}

func TestTestConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.TestConnectionLatency = 5 * time.Millisecond
		msg := cmds.TestConnection("localhost", 6379, "", 0)()
		result := msg.(types.ConnectionTestMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if !result.Success {
			t.Error("expected Success=true")
		}
		if result.Latency != 5*time.Millisecond {
			t.Errorf("Latency = %v, want %v", result.Latency, 5*time.Millisecond)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.TestConnectionError = errors.New("connection failed")
		msg := cmds.TestConnection("localhost", 6379, "", 0)()
		result := msg.(types.ConnectionTestMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
		if result.Success {
			t.Error("expected Success=false on error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.TestConnection("localhost", 6379, "", 0)()
		result := msg.(types.ConnectionTestMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}
