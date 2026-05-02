package testutil

import (
	"errors"
	"testing"

	"github.com/bearded-giant/redis-tui/internal/types"
)

func TestFullMockRedisClient_GetServerInfo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ServerInfo = types.ServerInfo{Version: "7.0.0", Mode: "standalone"}
		info, err := m.GetServerInfo()
		AssertNoError(t, err, "GetServerInfo")
		AssertEqual(t, info.Version, "7.0.0", "Version")
		AssertEqual(t, info.Mode, "standalone", "Mode")
		AssertEqual(t, m.Calls[0], "GetServerInfo", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ServerInfoError = errTest
		_, err := m.GetServerInfo()
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_GetMemoryStats(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.MemStats = types.MemoryStats{UsedMemory: 1024, PeakMemory: 2048}
		stats, err := m.GetMemoryStats()
		AssertNoError(t, err, "GetMemoryStats")
		AssertEqual(t, stats.UsedMemory, int64(1024), "UsedMemory")
		AssertEqual(t, stats.PeakMemory, int64(2048), "PeakMemory")
		AssertEqual(t, m.Calls[0], "GetMemoryStats", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.MemStatsError = errTest
		_, err := m.GetMemoryStats()
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_MemoryUsage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.MemUsageResult = 256
		usage, err := m.MemoryUsage("key")
		AssertNoError(t, err, "MemoryUsage")
		AssertEqual(t, usage, int64(256), "MemoryUsage result")
		AssertEqual(t, m.Calls[0], "MemoryUsage", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.MemUsageError = errTest
		_, err := m.MemoryUsage("key")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_SlowLogGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		entries := []types.SlowLogEntry{{ID: 1, Command: "KEYS *"}}
		m.SlowLogEntries = entries
		got, err := m.SlowLogGet(10)
		AssertNoError(t, err, "SlowLogGet")
		AssertSliceLen(t, got, 1, "SlowLogGet result")
		AssertEqual(t, got[0].Command, "KEYS *", "SlowLogEntry command")
		AssertEqual(t, m.Calls[0], "SlowLogGet", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.SlowLogError = errTest
		_, err := m.SlowLogGet(10)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_ClientList(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		clients := []types.ClientInfo{{ID: 1, Addr: "127.0.0.1:1234"}}
		m.ClientListResult = clients
		got, err := m.ClientList()
		AssertNoError(t, err, "ClientList")
		AssertSliceLen(t, got, 1, "ClientList result")
		AssertEqual(t, got[0].Addr, "127.0.0.1:1234", "client addr")
		AssertEqual(t, m.Calls[0], "ClientList", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ClientListError = errTest
		_, err := m.ClientList()
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_FlushDB(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.FlushDB()
		AssertNoError(t, err, "FlushDB")
		AssertEqual(t, m.Calls[0], "FlushDB", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.FlushDBError = errTest
		err := m.FlushDB()
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_SelectDB(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.SelectDB(1)
		AssertNoError(t, err, "SelectDB")
		AssertEqual(t, m.Calls[0], "SelectDB", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.SelectDBError = errTest
		err := m.SelectDB(1)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_ClusterNodes(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ClusterNodesResult = []types.ClusterNode{{ID: "abc", Role: "master"}}
		got, err := m.ClusterNodes()
		AssertNoError(t, err, "ClusterNodes")
		AssertSliceLen(t, got, 1, "ClusterNodes result")
		AssertEqual(t, got[0].Role, "master", "node role")
		AssertEqual(t, m.Calls[0], "ClusterNodes", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ClusterNodesError = errTest
		_, err := m.ClusterNodes()
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_ClusterInfo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ClusterInfoResult = "cluster_state:ok"
		got, err := m.ClusterInfo()
		AssertNoError(t, err, "ClusterInfo")
		AssertEqual(t, got, "cluster_state:ok", "ClusterInfo result")
		AssertEqual(t, m.Calls[0], "ClusterInfo", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ClusterInfoError = errTest
		_, err := m.ClusterInfo()
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_GetLiveMetrics(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.LiveMetricsResult = types.LiveMetricsData{OpsPerSec: 42.0, ConnectedClients: 10}
		got, err := m.GetLiveMetrics()
		AssertNoError(t, err, "GetLiveMetrics")
		AssertEqual(t, got.OpsPerSec, 42.0, "OpsPerSec")
		AssertEqual(t, got.ConnectedClients, int64(10), "ConnectedClients")
		AssertEqual(t, m.Calls[0], "GetLiveMetrics", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.LiveMetricsError = errTest
		_, err := m.GetLiveMetrics()
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_Eval(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.EvalResult = "OK"
		got, err := m.Eval("return 'OK'", nil)
		AssertNoError(t, err, "Eval")
		AssertEqual(t, got.(string), "OK", "Eval result")
		AssertEqual(t, m.Calls[0], "Eval", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.EvalError = errTest
		_, err := m.Eval("return 1", nil)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}
