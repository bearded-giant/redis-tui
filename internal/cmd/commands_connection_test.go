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
		testutil.MustAddConnection(t, cfg, types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0})
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
		msg := cmds.AddConnection(types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})()
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
		msg := cmds.AddConnection(types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})()
		result := msg.(types.ConnectionAddedMsg)
		if result.Err != nil {
			t.Errorf("nil config should not error: %v", result.Err)
		}
	})
}

func TestUpdateConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg := testutil.NewTestConfig(t)
		conn := testutil.MustAddConnection(t, cfg, types.Connection{Name: "old", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
		cmds := NewCommands(cfg, nil)
		msg := cmds.UpdateConnection(types.Connection{ID: conn.ID, Name: "new", Host: "localhost", Port: 6380, Password: "pass", DB: 1, UseCluster: false})()
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
		msg := cmds.UpdateConnection(types.Connection{ID: 1, Name: "n", Host: "h", Port: 1, Password: "p", DB: 0, UseCluster: false})()
		result := msg.(types.ConnectionUpdatedMsg)
		if result.Err != nil {
			t.Errorf("nil config should not error: %v", result.Err)
		}
	})
}

func TestDeleteConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg := testutil.NewTestConfig(t)
		conn := testutil.MustAddConnection(t, cfg, types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
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
		msg := cmds.Connect(&types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})()
		result := msg.(types.ConnectedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ConnectError = errors.New("connection refused")
		msg := cmds.Connect(&types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})()
		result := msg.(types.ConnectedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("cluster mode", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.Connect(&types.Connection{Name: "test", Host: "localhost", Port: 7000, DB: 0, UseCluster: true})()
		result := msg.(types.ConnectedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.Connect(&types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})()
		result := msg.(types.ConnectedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
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
		msg := cmds.TestConnection(&types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})()
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
		msg := cmds.TestConnection(&types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})()
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
		msg := cmds.TestConnection(&types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})()
		result := msg.(types.ConnectionTestMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}
