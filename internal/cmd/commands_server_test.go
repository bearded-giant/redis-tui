package cmd

import (
	"errors"
	"testing"

	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestLoadServerInfo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ServerInfo = types.ServerInfo{Version: "7.0.0"}
		msg := cmds.LoadServerInfo()()
		result := msg.(types.ServerInfoLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Info.Version != "7.0.0" {
			t.Errorf("Version = %q, want %q", result.Info.Version, "7.0.0")
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ServerInfoError = errors.New("info error")
		msg := cmds.LoadServerInfo()()
		result := msg.(types.ServerInfoLoadedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.LoadServerInfo()()
		result := msg.(types.ServerInfoLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestFlushDB(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.FlushDB()()
		result := msg.(types.FlushDBMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.FlushDBError = errors.New("flush error")
		msg := cmds.FlushDB()()
		result := msg.(types.FlushDBMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.FlushDB()()
		result := msg.(types.FlushDBMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestGetMemoryUsage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.MemUsageResult = 1024
		msg := cmds.GetMemoryUsage("mykey")()
		result := msg.(types.MemoryUsageMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Bytes != 1024 {
			t.Errorf("Bytes = %d, want 1024", result.Bytes)
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.GetMemoryUsage("k")()
		result := msg.(types.MemoryUsageMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestGetSlowLog(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.SlowLogEntries = []types.SlowLogEntry{{ID: 1, Command: "GET key"}}
		msg := cmds.GetSlowLog(10)()
		result := msg.(types.SlowLogLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(result.Entries))
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.GetSlowLog(10)()
		result := msg.(types.SlowLogLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestGetClientList(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ClientListResult = []types.ClientInfo{{ID: 1, Addr: "127.0.0.1:1234"}}
		msg := cmds.GetClientList()()
		result := msg.(types.ClientListLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Clients) != 1 {
			t.Errorf("expected 1 client, got %d", len(result.Clients))
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.GetClientList()()
		result := msg.(types.ClientListLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestGetMemoryStats(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.MemStats = types.MemoryStats{UsedMemory: 1024}
		msg := cmds.GetMemoryStats()()
		result := msg.(types.MemoryStatsLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Stats.UsedMemory != 1024 {
			t.Errorf("UsedMemory = %d, want 1024", result.Stats.UsedMemory)
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.GetMemoryStats()()
		result := msg.(types.MemoryStatsLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestGetClusterInfo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ClusterNodesResult = []types.ClusterNode{{ID: "abc"}}
		mock.ClusterInfoResult = "cluster_state:ok"
		msg := cmds.GetClusterInfo()()
		result := msg.(types.ClusterInfoLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Nodes) != 1 {
			t.Errorf("expected 1 node, got %d", len(result.Nodes))
		}
		if result.Info != "cluster_state:ok" {
			t.Errorf("Info = %q, want %q", result.Info, "cluster_state:ok")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.GetClusterInfo()()
		result := msg.(types.ClusterInfoLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestFetchClusterNodes(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ClusterNodesResult = []types.ClusterNode{
			{ID: "node1", Addr: "127.0.0.1:7000"},
			{ID: "node2", Addr: "127.0.0.1:7001"},
		}
		msg := cmds.FetchClusterNodes()()
		result := msg.(types.ClusterNodesLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Nodes) != 2 {
			t.Errorf("expected 2 nodes, got %d", len(result.Nodes))
		}
		if result.Nodes[0].ID != "node1" {
			t.Errorf("Nodes[0].ID = %q, want %q", result.Nodes[0].ID, "node1")
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ClusterNodesError = errors.New("cluster nodes error")
		msg := cmds.FetchClusterNodes()()
		result := msg.(types.ClusterNodesLoadedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.FetchClusterNodes()()
		result := msg.(types.ClusterNodesLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestLoadLiveMetrics(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.LiveMetricsResult = types.LiveMetricsData{
			OpsPerSec:       1500,
			UsedMemoryBytes: 1024000,
		}
		msg := cmds.LoadLiveMetrics()()
		result := msg.(types.LiveMetricsMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Data.OpsPerSec != 1500 {
			t.Errorf("OpsPerSec = %f, want 1500", result.Data.OpsPerSec)
		}
		if result.Data.UsedMemoryBytes != 1024000 {
			t.Errorf("UsedMemoryBytes = %d, want 1024000", result.Data.UsedMemoryBytes)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.LiveMetricsError = errors.New("metrics error")
		msg := cmds.LoadLiveMetrics()()
		result := msg.(types.LiveMetricsMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.LoadLiveMetrics()()
		result := msg.(types.LiveMetricsMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}
