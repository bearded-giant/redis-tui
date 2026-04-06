package testutil

import (
	"errors"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/redis/go-redis/v9"
)

var errTest = errors.New("test error")

func TestFullMockRedisClient_NewDefaults(t *testing.T) {
	m := NewFullMockRedisClient()
	if m == nil {
		t.Fatal("NewFullMockRedisClient returned nil")
	}
	if m.MockRedisClient == nil {
		t.Fatal("embedded MockRedisClient is nil")
	}
	if m.IsCluster() {
		t.Error("IsCluster should default to false")
	}
	if len(m.Calls) != 0 {
		t.Errorf("Calls should be empty, got %d", len(m.Calls))
	}
}

// --- Connection methods ---

func TestFullMockRedisClient_Connect(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.Connect("localhost", 6379, "", 0)
		AssertNoError(t, err, "Connect")
		AssertSliceLen(t, m.Calls, 1, "Calls after Connect")
		AssertEqual(t, m.Calls[0], "Connect", "call name")
		if !m.IsConnected() {
			t.Error("expected connected after Connect")
		}
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ConnectError = errTest
		err := m.Connect("localhost", 6379, "", 0)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_ConnectWithTLS(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.ConnectWithTLS("localhost", 6379, "", 0, nil)
		AssertNoError(t, err, "ConnectWithTLS")
		AssertEqual(t, m.Calls[0], "ConnectWithTLS", "call name")
		if !m.IsConnected() {
			t.Error("expected connected")
		}
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ConnectWithTLSError = errTest
		err := m.ConnectWithTLS("localhost", 6379, "", 0, nil)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_ConnectCluster(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.ConnectCluster([]string{"localhost:6379"}, "")
		AssertNoError(t, err, "ConnectCluster")
		AssertEqual(t, m.Calls[0], "ConnectCluster", "call name")
		if !m.IsConnected() {
			t.Error("expected connected")
		}
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ConnectClusterError = errTest
		err := m.ConnectCluster([]string{"localhost:6379"}, "")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_IsCluster(t *testing.T) {
	m := NewFullMockRedisClient()
	AssertEqual(t, m.IsCluster(), false, "default IsCluster")
	m.IsClusterResult = true
	AssertEqual(t, m.IsCluster(), true, "IsCluster after set")
}

func TestFullMockRedisClient_TestConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.TestConnectionLatency = 42 * time.Millisecond
		latency, err := m.TestConnection("localhost", 6379, "", 0)
		AssertNoError(t, err, "TestConnection")
		AssertEqual(t, latency, 42*time.Millisecond, "latency")
		AssertEqual(t, m.Calls[0], "TestConnection", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.TestConnectionError = errTest
		_, err := m.TestConnection("localhost", 6379, "", 0)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

// --- Key operations ---

func TestFullMockRedisClient_SetString(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.SetString("key", "value", 0)
		AssertNoError(t, err, "SetString")
		AssertEqual(t, m.Calls[0], "SetString", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.SetStringError = errTest
		err := m.SetString("key", "value", 0)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_SetTTL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.SetTTL("key", 10*time.Second)
		AssertNoError(t, err, "SetTTL")
		AssertEqual(t, m.Calls[0], "SetTTL", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.SetTTLError = errTest
		err := m.SetTTL("key", 10*time.Second)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_Rename(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.Rename("old", "new")
		AssertNoError(t, err, "Rename")
		AssertEqual(t, m.Calls[0], "Rename", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.RenameError = errTest
		err := m.Rename("old", "new")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_Copy(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.Copy("src", "dst", false)
		AssertNoError(t, err, "Copy")
		AssertEqual(t, m.Calls[0], "Copy", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.CopyError = errTest
		err := m.Copy("src", "dst", true)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_DeleteKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		n, err := m.DeleteKeys("a", "b")
		AssertNoError(t, err, "DeleteKeys")
		AssertEqual(t, n, int64(0), "DeleteKeys result")
		AssertEqual(t, m.Calls[0], "DeleteKeys", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.DeleteKeysError = errTest
		_, err := m.DeleteKeys("a")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_BulkDelete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.BulkDeleteResult = 5
		n, err := m.BulkDelete("user:*")
		AssertNoError(t, err, "BulkDelete")
		AssertEqual(t, n, 5, "BulkDelete result")
		AssertEqual(t, m.Calls[0], "BulkDelete", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.BulkDeleteError = errTest
		_, err := m.BulkDelete("user:*")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

// --- Collection operations ---

func TestFullMockRedisClient_ListOps(t *testing.T) {
	tests := []struct {
		name    string
		setErr  func(*FullMockRedisClient)
		call    func(*FullMockRedisClient) error
		logName string
	}{
		{"RPush success", nil, func(m *FullMockRedisClient) error { return m.RPush("list", "a", "b") }, "RPush"},
		{"RPush error", func(m *FullMockRedisClient) { m.RPushError = errTest }, func(m *FullMockRedisClient) error { return m.RPush("list", "a") }, "RPush"},
		{"LSet success", nil, func(m *FullMockRedisClient) error { return m.LSet("list", 0, "val") }, "LSet"},
		{"LSet error", func(m *FullMockRedisClient) { m.LSetError = errTest }, func(m *FullMockRedisClient) error { return m.LSet("list", 0, "val") }, "LSet"},
		{"LRem success", nil, func(m *FullMockRedisClient) error { return m.LRem("list", 1, "val") }, "LRem"},
		{"LRem error", func(m *FullMockRedisClient) { m.LRemError = errTest }, func(m *FullMockRedisClient) error { return m.LRem("list", 1, "val") }, "LRem"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewFullMockRedisClient()
			if tt.setErr != nil {
				tt.setErr(m)
			}
			err := tt.call(m)
			if tt.setErr != nil {
				if !errors.Is(err, errTest) {
					t.Errorf("expected errTest, got %v", err)
				}
			} else {
				AssertNoError(t, err, tt.logName)
			}
			AssertEqual(t, m.Calls[0], tt.logName, "call name")
		})
	}
}

func TestFullMockRedisClient_SetOps(t *testing.T) {
	tests := []struct {
		name    string
		setErr  func(*FullMockRedisClient)
		call    func(*FullMockRedisClient) error
		logName string
	}{
		{"SAdd success", nil, func(m *FullMockRedisClient) error { return m.SAdd("set", "a", "b") }, "SAdd"},
		{"SAdd error", func(m *FullMockRedisClient) { m.SAddError = errTest }, func(m *FullMockRedisClient) error { return m.SAdd("set", "a") }, "SAdd"},
		{"SRem success", nil, func(m *FullMockRedisClient) error { return m.SRem("set", "a") }, "SRem"},
		{"SRem error", func(m *FullMockRedisClient) { m.SRemError = errTest }, func(m *FullMockRedisClient) error { return m.SRem("set", "a") }, "SRem"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewFullMockRedisClient()
			if tt.setErr != nil {
				tt.setErr(m)
			}
			err := tt.call(m)
			if tt.setErr != nil {
				if !errors.Is(err, errTest) {
					t.Errorf("expected errTest, got %v", err)
				}
			} else {
				AssertNoError(t, err, tt.logName)
			}
			AssertEqual(t, m.Calls[0], tt.logName, "call name")
		})
	}
}

func TestFullMockRedisClient_ZSetOps(t *testing.T) {
	t.Run("ZAdd success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.ZAdd("zset", 1.5, "member")
		AssertNoError(t, err, "ZAdd")
		AssertEqual(t, m.Calls[0], "ZAdd", "call name")
	})
	t.Run("ZAdd error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ZAddError = errTest
		err := m.ZAdd("zset", 1.5, "member")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
	t.Run("ZRem success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.ZRem("zset", "member")
		AssertNoError(t, err, "ZRem")
		AssertEqual(t, m.Calls[0], "ZRem", "call name")
	})
	t.Run("ZRem error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ZRemError = errTest
		err := m.ZRem("zset", "member")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_HashOps(t *testing.T) {
	t.Run("HSet success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.HSet("hash", "field", "value")
		AssertNoError(t, err, "HSet")
		AssertEqual(t, m.Calls[0], "HSet", "call name")
	})
	t.Run("HSet error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.HSetError = errTest
		err := m.HSet("hash", "field", "value")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
	t.Run("HDel success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.HDel("hash", "field1", "field2")
		AssertNoError(t, err, "HDel")
		AssertEqual(t, m.Calls[0], "HDel", "call name")
	})
	t.Run("HDel error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.HDelError = errTest
		err := m.HDel("hash", "field1")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_StreamOps(t *testing.T) {
	t.Run("XAdd success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.XAddResult = "1234-0"
		id, err := m.XAdd("stream", map[string]any{"k": "v"})
		AssertNoError(t, err, "XAdd")
		AssertEqual(t, id, "1234-0", "XAdd result")
		AssertEqual(t, m.Calls[0], "XAdd", "call name")
	})
	t.Run("XAdd error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.XAddError = errTest
		_, err := m.XAdd("stream", map[string]any{"k": "v"})
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
	t.Run("XDel success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.XDel("stream", "1234-0")
		AssertNoError(t, err, "XDel")
		AssertEqual(t, m.Calls[0], "XDel", "call name")
	})
	t.Run("XDel error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.XDelError = errTest
		err := m.XDel("stream", "1234-0")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

// --- Server operations ---

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

// --- Pub/Sub ---

func TestFullMockRedisClient_Publish(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.PublishResult = 3
		n, err := m.Publish("channel", "msg")
		AssertNoError(t, err, "Publish")
		AssertEqual(t, n, int64(3), "Publish result")
		AssertEqual(t, m.Calls[0], "Publish", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.PublishError = errTest
		_, err := m.Publish("channel", "msg")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_Subscribe(t *testing.T) {
	m := NewFullMockRedisClient()
	result := m.Subscribe("channel")
	if result != nil {
		t.Error("Subscribe should return nil")
	}
	AssertEqual(t, m.Calls[0], "Subscribe", "call name")
}

func TestFullMockRedisClient_PubSubChannels(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.PubSubChannelsResult = []string{"ch1", "ch2"}
		got, err := m.PubSubChannels("*")
		AssertNoError(t, err, "PubSubChannels")
		AssertSliceLen(t, got, 2, "PubSubChannels result")
		AssertEqual(t, got[0], "ch1", "channel 0")
		AssertEqual(t, m.Calls[0], "PubSubChannels", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.PubSubChannelsError = errTest
		_, err := m.PubSubChannels("*")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_SubscribeKeyspace(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.SubscribeKeyspace("*", func(_ types.KeyspaceEvent) {})
		AssertNoError(t, err, "SubscribeKeyspace")
		AssertEqual(t, m.Calls[0], "SubscribeKeyspace", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.SubscribeKeyspaceError = errTest
		err := m.SubscribeKeyspace("*", func(_ types.KeyspaceEvent) {})
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_UnsubscribeKeyspace(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.UnsubscribeKeyspace()
		AssertNoError(t, err, "UnsubscribeKeyspace")
		AssertEqual(t, m.Calls[0], "UnsubscribeKeyspace", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.UnsubscribeKSError = errTest
		err := m.UnsubscribeKeyspace()
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

// --- Config operations ---

func TestFullMockRedisClient_ConfigGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ConfigGetResult = map[string]string{"maxmemory": "100mb"}
		got, err := m.ConfigGet("maxmemory")
		AssertNoError(t, err, "ConfigGet")
		AssertEqual(t, got["maxmemory"], "100mb", "config value")
		AssertEqual(t, m.Calls[0], "ConfigGet", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ConfigGetError = errTest
		_, err := m.ConfigGet("maxmemory")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_ConfigSet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.ConfigSet("maxmemory", "100mb")
		AssertNoError(t, err, "ConfigSet")
		AssertEqual(t, m.Calls[0], "ConfigSet", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ConfigSetError = errTest
		err := m.ConfigSet("maxmemory", "100mb")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

// --- Search operations ---

func TestFullMockRedisClient_ScanKeysWithRegex(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.RegexSearchResult = []types.RedisKey{{Key: "user:1"}}
		got, err := m.ScanKeysWithRegex("user:.*", 100)
		AssertNoError(t, err, "ScanKeysWithRegex")
		AssertSliceLen(t, got, 1, "ScanKeysWithRegex result")
		AssertEqual(t, got[0].Key, "user:1", "key name")
		AssertEqual(t, m.Calls[0], "ScanKeysWithRegex", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.RegexSearchError = errTest
		_, err := m.ScanKeysWithRegex("user:.*", 100)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_FuzzySearchKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.FuzzySearchResult = []types.RedisKey{{Key: "session:abc"}}
		got, err := m.FuzzySearchKeys("sess", 50)
		AssertNoError(t, err, "FuzzySearchKeys")
		AssertSliceLen(t, got, 1, "FuzzySearchKeys result")
		AssertEqual(t, m.Calls[0], "FuzzySearchKeys", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.FuzzySearchError = errTest
		_, err := m.FuzzySearchKeys("sess", 50)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_SearchByValue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.SearchByValueResult = []types.RedisKey{{Key: "k1"}, {Key: "k2"}}
		got, err := m.SearchByValue("*", "needle", 100)
		AssertNoError(t, err, "SearchByValue")
		AssertSliceLen(t, got, 2, "SearchByValue result")
		AssertEqual(t, m.Calls[0], "SearchByValue", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.SearchByValueError = errTest
		_, err := m.SearchByValue("*", "needle", 100)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_CompareKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.CompareValue1 = types.RedisValue{StringValue: "a"}
		m.CompareValue2 = types.RedisValue{StringValue: "b"}
		v1, v2, err := m.CompareKeys("key1", "key2")
		AssertNoError(t, err, "CompareKeys")
		AssertEqual(t, v1.StringValue, "a", "CompareValue1")
		AssertEqual(t, v2.StringValue, "b", "CompareValue2")
		AssertEqual(t, m.Calls[0], "CompareKeys", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.CompareKeysError = errTest
		_, _, err := m.CompareKeys("key1", "key2")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_GetKeyPrefixes(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.KeyPrefixesResult = []string{"user:", "session:"}
		got, err := m.GetKeyPrefixes(":", 10)
		AssertNoError(t, err, "GetKeyPrefixes")
		AssertSliceLen(t, got, 2, "GetKeyPrefixes result")
		AssertEqual(t, m.Calls[0], "GetKeyPrefixes", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.KeyPrefixesError = errTest
		_, err := m.GetKeyPrefixes(":", 10)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

// --- IO operations ---

func TestFullMockRedisClient_ExportKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ExportResult = map[string]any{"key1": "val1"}
		got, err := m.ExportKeys("*")
		AssertNoError(t, err, "ExportKeys")
		if got["key1"] != "val1" {
			t.Errorf("expected key1=val1, got %v", got["key1"])
		}
		AssertEqual(t, m.Calls[0], "ExportKeys", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ExportError = errTest
		_, err := m.ExportKeys("*")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_ImportKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ImportResult = 3
		n, err := m.ImportKeys(map[string]any{"a": "1", "b": "2", "c": "3"})
		AssertNoError(t, err, "ImportKeys")
		AssertEqual(t, n, 3, "ImportKeys result")
		AssertEqual(t, m.Calls[0], "ImportKeys", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ImportError = errTest
		_, err := m.ImportKeys(map[string]any{})
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

// --- Metrics ---

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

// --- Special types: JSON ---

func TestFullMockRedisClient_JSONGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.JSONGetResult = `{"name":"test"}`
		got, err := m.JSONGet("key")
		AssertNoError(t, err, "JSONGet")
		AssertEqual(t, got, `{"name":"test"}`, "JSONGet result")
		AssertEqual(t, m.Calls[0], "JSONGet", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.JSONGetError = errTest
		_, err := m.JSONGet("key")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_JSONGetPath(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.JSONGetResult = `"test"`
		got, err := m.JSONGetPath("key", "$.name")
		AssertNoError(t, err, "JSONGetPath")
		AssertEqual(t, got, `"test"`, "JSONGetPath result")
		AssertEqual(t, m.Calls[0], "JSONGetPath", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.JSONGetError = errTest
		_, err := m.JSONGetPath("key", "$.name")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_JSONSet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.JSONSet("key", `{"a":1}`)
		AssertNoError(t, err, "JSONSet")
		AssertEqual(t, m.Calls[0], "JSONSet", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.JSONSetError = errTest
		err := m.JSONSet("key", `{"a":1}`)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

// --- HyperLogLog ---

func TestFullMockRedisClient_PFAdd(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.PFAdd("hll", "a", "b", "c")
		AssertNoError(t, err, "PFAdd")
		AssertEqual(t, m.Calls[0], "PFAdd", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.PFAddError = errTest
		err := m.PFAdd("hll", "a")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_PFCount(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.PFCountResult = 42
		got, err := m.PFCount("hll")
		AssertNoError(t, err, "PFCount")
		AssertEqual(t, got, int64(42), "PFCount result")
		AssertEqual(t, m.Calls[0], "PFCount", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.PFCountError = errTest
		_, err := m.PFCount("hll")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

// --- Bitmap ---

func TestFullMockRedisClient_SetBit(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.SetBit("bitmap", 7, 1)
		AssertNoError(t, err, "SetBit")
		AssertEqual(t, m.Calls[0], "SetBit", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.SetBitError = errTest
		err := m.SetBit("bitmap", 7, 1)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_GetBit(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		got, err := m.GetBit("bitmap", 7)
		AssertNoError(t, err, "GetBit")
		AssertEqual(t, got, int64(0), "GetBit result")
		AssertEqual(t, m.Calls[0], "GetBit", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.GetBitError = errTest
		_, err := m.GetBit("bitmap", 7)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_BitCount(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.BitCountResult = 5
		got, err := m.BitCount("bitmap")
		AssertNoError(t, err, "BitCount")
		AssertEqual(t, got, int64(5), "BitCount result")
		AssertEqual(t, m.Calls[0], "BitCount", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.BitCountError = errTest
		_, err := m.BitCount("bitmap")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

// --- Geo ---

func TestFullMockRedisClient_GeoAdd(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		loc := &redis.GeoLocation{Name: "place", Longitude: 1.0, Latitude: 2.0}
		err := m.GeoAdd("geo", loc)
		AssertNoError(t, err, "GeoAdd")
		AssertEqual(t, m.Calls[0], "GeoAdd", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.GeoAddError = errTest
		err := m.GeoAdd("geo", &redis.GeoLocation{})
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_GeoPos(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		pos := &redis.GeoPos{Longitude: 1.0, Latitude: 2.0}
		m.GeoPosResult = []*redis.GeoPos{pos}
		got, err := m.GeoPos("geo", "place")
		AssertNoError(t, err, "GeoPos")
		AssertSliceLen(t, got, 1, "GeoPos result")
		AssertEqual(t, got[0].Longitude, 1.0, "longitude")
		AssertEqual(t, got[0].Latitude, 2.0, "latitude")
		AssertEqual(t, m.Calls[0], "GeoPos", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.GeoPosError = errTest
		_, err := m.GeoPos("geo", "place")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

// --- Misc ---

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

func TestFullMockRedisClient_BatchSetTTL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.BatchTTLResult = 10
		n, err := m.BatchSetTTL("user:*", 60*time.Second)
		AssertNoError(t, err, "BatchSetTTL")
		AssertEqual(t, n, 10, "BatchSetTTL result")
		AssertEqual(t, m.Calls[0], "BatchSetTTL", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.BatchSetTTLError = errTest
		_, err := m.BatchSetTTL("user:*", 60*time.Second)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_SetIncludeTypes(t *testing.T) {
	m := NewFullMockRedisClient()
	m.SetIncludeTypes(true)
	AssertEqual(t, m.Calls[0], "SetIncludeTypes", "call name")
}

// --- Cluster ---

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

// --- Call tracking accumulation ---

func TestFullMockRedisClient_CallTracking(t *testing.T) {
	m := NewFullMockRedisClient()
	_ = m.Connect("localhost", 6379, "", 0)
	m.SetIncludeTypes(true)
	_ = m.FlushDB()
	_ = m.SelectDB(0)

	expected := []string{"Connect", "SetIncludeTypes", "FlushDB", "SelectDB"}
	AssertSliceLen(t, m.Calls, len(expected), "total calls")
	for i, name := range expected {
		AssertEqual(t, m.Calls[i], name, "call order")
	}
}
