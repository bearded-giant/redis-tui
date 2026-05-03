package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/bearded-giant/redis-tui/internal/types"
)

func TestHandleServerInfoLoadedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ServerInfoLoadedMsg{Info: types.ServerInfo{Version: "7.0.0"}}
		result, _ := m.handleServerInfoLoadedMsg(msg)
		model := result.(Model)
		if model.ServerInfo.Version != "7.0.0" {
			t.Errorf("unexpected version: %q", model.ServerInfo.Version)
		}
		if model.Screen != types.ScreenServerInfo {
			t.Errorf("expected ScreenServerInfo, got %v", model.Screen)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ServerInfoLoadedMsg{Err: errors.New("boom")}
		_, _ = m.handleServerInfoLoadedMsg(msg)
	})
}

func TestHandleDBSwitchedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentConn = &types.Connection{DB: 0}
		msg := types.DBSwitchedMsg{DB: 3}
		result, cmd := m.handleDBSwitchedMsg(msg)
		model := result.(Model)
		if model.CurrentConn.DB != 3 {
			t.Errorf("expected DB=3, got %d", model.CurrentConn.DB)
		}
		if cmd == nil {
			t.Error("expected load keys cmd")
		}
	})
	t.Run("success nil conn", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.DBSwitchedMsg{DB: 3}
		_, cmd := m.handleDBSwitchedMsg(msg)
		if cmd == nil {
			t.Error("expected load keys cmd")
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.DBSwitchedMsg{Err: errors.New("boom")}
		result, cmd := m.handleDBSwitchedMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
		if cmd != nil {
			t.Error("expected nil cmd on error")
		}
	})
}

func TestHandleFlushDBMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "a"}}
		msg := types.FlushDBMsg{}
		result, _ := m.handleFlushDBMsg(msg)
		model := result.(Model)
		if len(model.Keys) != 0 {
			t.Errorf("expected keys cleared, got %d", len(model.Keys))
		}
		if model.StatusMsg != "Database flushed" {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "a"}}
		msg := types.FlushDBMsg{Err: errors.New("boom")}
		result, _ := m.handleFlushDBMsg(msg)
		model := result.(Model)
		if len(model.Keys) != 1 {
			t.Errorf("expected keys preserved, got %d", len(model.Keys))
		}
	})
}

func TestHandleSlowLogLoadedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.SlowLogLoadedMsg{Entries: []types.SlowLogEntry{{ID: 1}}}
		result, _ := m.handleSlowLogLoadedMsg(msg)
		model := result.(Model)
		if len(model.SlowLogEntries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(model.SlowLogEntries))
		}
		if model.Screen != types.ScreenSlowLog {
			t.Errorf("expected ScreenSlowLog, got %v", model.Screen)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.SlowLogLoadedMsg{Err: errors.New("boom")}
		result, _ := m.handleSlowLogLoadedMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
	})
}

func TestHandleClientListLoadedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ClientListLoadedMsg{Clients: []types.ClientInfo{{ID: 1}}}
		result, _ := m.handleClientListLoadedMsg(msg)
		model := result.(Model)
		if len(model.ClientList) != 1 {
			t.Errorf("expected 1 client, got %d", len(model.ClientList))
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ClientListLoadedMsg{Err: errors.New("boom")}
		_, _ = m.handleClientListLoadedMsg(msg)
	})
}

func TestHandleMemoryStatsLoadedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.MemoryStatsLoadedMsg{Stats: types.MemoryStats{UsedMemory: 1024}}
		result, _ := m.handleMemoryStatsLoadedMsg(msg)
		model := result.(Model)
		if model.MemoryStats == nil || model.MemoryStats.UsedMemory != 1024 {
			t.Error("expected stats set")
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.MemoryStatsLoadedMsg{Err: errors.New("boom")}
		_, _ = m.handleMemoryStatsLoadedMsg(msg)
	})
}

func TestHandleClusterInfoLoadedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ClusterInfoLoadedMsg{Nodes: []types.ClusterNode{{ID: "n1"}}}
		result, _ := m.handleClusterInfoLoadedMsg(msg)
		model := result.(Model)
		if !model.ClusterEnabled {
			t.Error("expected cluster enabled")
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ClusterInfoLoadedMsg{Err: errors.New("boom")}
		_, _ = m.handleClusterInfoLoadedMsg(msg)
	})
}

func TestHandleClusterNodesLoadedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ClusterNodesLoadedMsg{Nodes: []types.ClusterNode{{ID: "n1"}}}
		result, _ := m.handleClusterNodesLoadedMsg(msg)
		model := result.(Model)
		if !model.ClusterEnabled {
			t.Error("expected cluster enabled")
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ClusterNodesLoadedMsg{Err: errors.New("boom")}
		_, _ = m.handleClusterNodesLoadedMsg(msg)
	})
}

func TestHandleMemoryUsageMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.MemoryUsageMsg{Bytes: 42}
		result, _ := m.handleMemoryUsageMsg(msg)
		model := result.(Model)
		if model.MemoryUsage != 42 {
			t.Errorf("expected 42, got %d", model.MemoryUsage)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.MemoryUsageMsg{Err: errors.New("boom")}
		_, _ = m.handleMemoryUsageMsg(msg)
	})
}

func TestHandlePubSubChannelsLoadedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.PubSubChannelsLoadedMsg{Channels: []types.PubSubChannel{{Name: "ch"}}}
		result, _ := m.handlePubSubChannelsLoadedMsg(msg)
		model := result.(Model)
		if len(model.PubSubChannels) != 1 {
			t.Errorf("expected 1 channel, got %d", len(model.PubSubChannels))
		}
		if model.Screen != types.ScreenPubSubChannels {
			t.Errorf("expected ScreenPubSubChannels, got %v", model.Screen)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.PubSubChannelsLoadedMsg{Err: errors.New("boom")}
		result, _ := m.handlePubSubChannelsLoadedMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
	})
}

func TestHandleConfigLoadedMsg(t *testing.T) {
	t.Run("success sorts params", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ConfigLoadedMsg{Params: map[string]string{"b": "2", "a": "1"}}
		result, _ := m.handleConfigLoadedMsg(msg)
		model := result.(Model)
		if len(model.RedisConfigParams) != 2 {
			t.Errorf("expected 2 params, got %d", len(model.RedisConfigParams))
		}
		if model.RedisConfigParams[0].Name != "a" {
			t.Errorf("expected sorted, got %q first", model.RedisConfigParams[0].Name)
		}
		if model.Screen != types.ScreenRedisConfig {
			t.Errorf("expected ScreenRedisConfig, got %v", model.Screen)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ConfigLoadedMsg{Err: errors.New("boom")}
		result, _ := m.handleConfigLoadedMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
	})
}

func TestHandleConfigSetMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ConfigSetMsg{Param: "maxmemory", Value: "100mb"}
		_, cmd := m.handleConfigSetMsg(msg)
		if cmd == nil {
			t.Error("expected reload cmd")
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ConfigSetMsg{Err: errors.New("boom")}
		result, cmd := m.handleConfigSetMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
		if cmd != nil {
			t.Error("expected nil cmd on error")
		}
	})
}

func TestHandleLuaScriptResultMsg(t *testing.T) {
	t.Run("string result", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.LuaScriptResultMsg{Result: "hello"}
		result, _ := m.handleLuaScriptResultMsg(msg)
		model := result.(Model)
		if model.LuaResult != "hello" {
			t.Errorf("unexpected result: %q", model.LuaResult)
		}
	})
	t.Run("int64 result", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.LuaScriptResultMsg{Result: int64(42)}
		result, _ := m.handleLuaScriptResultMsg(msg)
		model := result.(Model)
		if model.LuaResult != "42" {
			t.Errorf("unexpected result: %q", model.LuaResult)
		}
	})
	t.Run("array result", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.LuaScriptResultMsg{Result: []any{1, 2, 3}}
		result, _ := m.handleLuaScriptResultMsg(msg)
		model := result.(Model)
		if !strings.Contains(model.LuaResult, "length: 3") {
			t.Errorf("unexpected result: %q", model.LuaResult)
		}
	})
	t.Run("default result", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.LuaScriptResultMsg{Result: map[string]int{"a": 1}}
		result, _ := m.handleLuaScriptResultMsg(msg)
		model := result.(Model)
		if model.LuaResult != "OK" {
			t.Errorf("unexpected result: %q", model.LuaResult)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.LuaScriptResultMsg{Err: errors.New("boom")}
		result, _ := m.handleLuaScriptResultMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.LuaResult, "Error:") {
			t.Errorf("unexpected result: %q", model.LuaResult)
		}
	})
}

func TestHandlePublishResultMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.PublishResultMsg{Receivers: 3}
		result, _ := m.handlePublishResultMsg(msg)
		model := result.(Model)
		if !strings.Contains(model.StatusMsg, "3 subscribers") {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
		if model.Screen != types.ScreenPubSubChannels {
			t.Errorf("expected ScreenPubSubChannels, got %v", model.Screen)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.PublishResultMsg{Err: errors.New("boom")}
		result, _ := m.handlePublishResultMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Publish failed:") {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
	})
}

func TestHandleKeyspaceEventMsg(t *testing.T) {
	t.Run("set event reloads keys", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.KeyspaceEventMsg{Event: types.KeyspaceEvent{Event: "set", Key: "foo"}}
		_, cmd := m.handleKeyspaceEventMsg(msg)
		if cmd == nil {
			t.Error("expected reload cmd")
		}
	})
	t.Run("del event reloads keys", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.KeyspaceEventMsg{Event: types.KeyspaceEvent{Event: "del", Key: "foo"}}
		_, cmd := m.handleKeyspaceEventMsg(msg)
		if cmd == nil {
			t.Error("expected reload cmd")
		}
	})
	t.Run("other event no cmd", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.KeyspaceEventMsg{Event: types.KeyspaceEvent{Event: "expire", Key: "foo"}}
		_, cmd := m.handleKeyspaceEventMsg(msg)
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("event buffer overflow", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		for range 105 {
			m.KeyspaceEvents = append(m.KeyspaceEvents, types.KeyspaceEvent{Key: "k"})
		}
		msg := types.KeyspaceEventMsg{Event: types.KeyspaceEvent{Event: "other"}}
		result, _ := m.handleKeyspaceEventMsg(msg)
		model := result.(Model)
		if len(model.KeyspaceEvents) > 100 {
			t.Errorf("expected truncated events, got %d", len(model.KeyspaceEvents))
		}
	})
}

func TestHandleLiveMetricsMsg(t *testing.T) {
	t.Run("success first data point", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.LiveMetricsMsg{Data: types.LiveMetricsData{OpsPerSec: 1.0}}
		result, _ := m.handleLiveMetricsMsg(msg)
		model := result.(Model)
		if model.LiveMetrics == nil || len(model.LiveMetrics.DataPoints) != 1 {
			t.Error("expected 1 data point")
		}
		if !model.LiveMetricsActive {
			t.Error("expected LiveMetricsActive=true")
		}
	})
	t.Run("success overflow truncates", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.LiveMetrics = &types.LiveMetrics{MaxDataPoints: 3}
		for range 3 {
			m.LiveMetrics.DataPoints = append(m.LiveMetrics.DataPoints, types.LiveMetricsData{})
		}
		msg := types.LiveMetricsMsg{Data: types.LiveMetricsData{OpsPerSec: 9}}
		result, _ := m.handleLiveMetricsMsg(msg)
		model := result.(Model)
		if len(model.LiveMetrics.DataPoints) != 3 {
			t.Errorf("expected 3 points, got %d", len(model.LiveMetrics.DataPoints))
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.LiveMetricsMsg{Err: errors.New("boom")}
		result, _ := m.handleLiveMetricsMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
	})
}

func TestHandleLiveMetricsTickMsg(t *testing.T) {
	t.Run("active", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.LiveMetricsActive = true
		_, cmd := m.handleLiveMetricsTickMsg()
		if cmd == nil {
			t.Error("expected refresh cmd")
		}
	})
	t.Run("inactive", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleLiveMetricsTickMsg()
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
}
