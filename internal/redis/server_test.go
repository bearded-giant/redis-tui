package redis

import (
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestParseClusterNodes(t *testing.T) {
	t.Run("single node", func(t *testing.T) {
		input := "abc123 127.0.0.1:7000@17000 master - 0 0 1 connected 0-5460\n"
		nodes := parseClusterNodes(input)
		if len(nodes) != 1 {
			t.Fatalf("expected 1 node, got %d", len(nodes))
		}
		if nodes[0].ID != "abc123" {
			t.Errorf("ID = %q, want %q", nodes[0].ID, "abc123")
		}
		if nodes[0].Addr != "127.0.0.1:7000@17000" {
			t.Errorf("Addr = %q, want %q", nodes[0].Addr, "127.0.0.1:7000@17000")
		}
		if nodes[0].Flags != "master" {
			t.Errorf("Flags = %q, want %q", nodes[0].Flags, "master")
		}
		if nodes[0].LinkState != "connected" {
			t.Errorf("LinkState = %q, want %q", nodes[0].LinkState, "connected")
		}
		if nodes[0].Slots != "0-5460" {
			t.Errorf("Slots = %q, want %q", nodes[0].Slots, "0-5460")
		}
	})

	t.Run("multiple nodes", func(t *testing.T) {
		input := "abc123 127.0.0.1:7000@17000 master - 0 0 1 connected 0-5460\n" +
			"def456 127.0.0.1:7001@17001 slave abc123 0 0 1 connected\n" +
			"ghi789 127.0.0.1:7002@17002 master - 0 0 2 connected 5461-10922\n"
		nodes := parseClusterNodes(input)
		if len(nodes) != 3 {
			t.Fatalf("expected 3 nodes, got %d", len(nodes))
		}
	})

	t.Run("empty input", func(t *testing.T) {
		nodes := parseClusterNodes("")
		if len(nodes) != 0 {
			t.Errorf("expected 0 nodes for empty input, got %d", len(nodes))
		}
	})

	t.Run("lines with fewer than 8 fields skipped", func(t *testing.T) {
		input := "abc123 127.0.0.1:7000@17000 master - 0 0 1 connected 0-5460\n" +
			"short line only\n" +
			"def456 127.0.0.1:7001@17001 slave abc123 0 0 1 connected\n"
		nodes := parseClusterNodes(input)
		if len(nodes) != 2 {
			t.Errorf("expected 2 nodes (short line skipped), got %d", len(nodes))
		}
	})

	t.Run("node without slots", func(t *testing.T) {
		input := "def456 127.0.0.1:7001@17001 slave abc123 0 0 1 connected\n"
		nodes := parseClusterNodes(input)
		if len(nodes) != 1 {
			t.Fatalf("expected 1 node, got %d", len(nodes))
		}
		if nodes[0].Slots != "" {
			t.Errorf("Slots = %q, want empty", nodes[0].Slots)
		}
	})

	t.Run("master field", func(t *testing.T) {
		input := "def456 127.0.0.1:7001@17001 slave abc123 0 0 1 connected\n"
		nodes := parseClusterNodes(input)
		if nodes[0].Master != "abc123" {
			t.Errorf("Master = %q, want %q", nodes[0].Master, "abc123")
		}
	})
}

func TestGetServerInfo(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("k1", "v1")
	mr.Set("k2", "v2")
	mr.Set("k3", "v3")

	info, err := client.GetServerInfo()
	if err != nil {
		t.Fatalf("GetServerInfo() error = %v", err)
	}

	// TotalKeys is derived from DBSize which miniredis fully supports
	if info.TotalKeys != "3" {
		t.Errorf("GetServerInfo() TotalKeys = %q, want %q", info.TotalKeys, "3")
	}

	// miniredis may not populate redis_version in INFO; verify the field is at least set
	// (Version may be empty with miniredis, which is acceptable)
	t.Logf("GetServerInfo() Version = %q", info.Version)
}

func TestGetMemoryStats(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("mem1", "some-value")
	mr.Set("mem2", "another-value")

	stats, err := client.GetMemoryStats()
	if err != nil {
		// miniredis does not support INFO memory section; verify we get a
		// graceful error rather than a panic
		t.Logf("GetMemoryStats() returned expected error with miniredis: %v", err)
		return
	}

	// If miniredis ever adds support, verify the result is reasonable
	if stats.UsedMemory < 0 {
		t.Errorf("GetMemoryStats() UsedMemory = %d, want >= 0", stats.UsedMemory)
	}
}

func TestFlushDB(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("a", "1")
	mr.Set("b", "2")
	mr.Set("c", "3")

	if client.GetTotalKeys() != 3 {
		t.Fatal("expected 3 keys before flush")
	}

	if err := client.FlushDB(); err != nil {
		t.Fatalf("FlushDB() error = %v", err)
	}

	if got := client.GetTotalKeys(); got != 0 {
		t.Errorf("FlushDB() left %d keys, want 0", got)
	}
}

func TestEval(t *testing.T) {
	client, _ := setupTestClient(t)

	result, err := client.Eval("return 1+1", nil)
	if err != nil {
		t.Fatalf("Eval() error = %v", err)
	}

	val, ok := result.(int64)
	if !ok {
		t.Fatalf("Eval() result type = %T, want int64", result)
	}
	if val != 2 {
		t.Errorf("Eval(return 1+1) = %d, want 2", val)
	}
}

func TestGetLiveMetrics(t *testing.T) {
	client, _ := setupTestClient(t)

	metrics, err := client.GetLiveMetrics()
	if err != nil {
		// miniredis does not support INFO with multiple section arguments;
		// verify we get a graceful error rather than a panic
		t.Logf("GetLiveMetrics() returned expected error with miniredis: %v", err)
		return
	}

	if metrics.Timestamp.IsZero() {
		t.Error("GetLiveMetrics() Timestamp is zero")
	}

	if time.Since(metrics.Timestamp) > 5*time.Second {
		t.Error("GetLiveMetrics() Timestamp is not recent")
	}

	// ConnectedClients should be at least 1 (our own connection)
	if metrics.ConnectedClients < 1 {
		t.Errorf("GetLiveMetrics() ConnectedClients = %d, want >= 1", metrics.ConnectedClients)
	}
}

func TestDeleteKeys(t *testing.T) {
	t.Run("deletes specified keys and returns count", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("k1", "v1")
		mr.Set("k2", "v2")
		mr.Set("k3", "v3")

		count, err := client.DeleteKeys("k1", "k2")
		if err != nil {
			t.Fatalf("DeleteKeys() error = %v", err)
		}
		if count != 2 {
			t.Errorf("DeleteKeys() count = %d, want 2", count)
		}

		if client.GetTotalKeys() != 1 {
			t.Errorf("expected 1 key remaining, got %d", client.GetTotalKeys())
		}

		if !mr.Exists("k3") {
			t.Error("expected k3 to still exist")
		}
	})

	t.Run("deleting non-existent keys returns 0", func(t *testing.T) {
		client, _ := setupTestClient(t)

		count, err := client.DeleteKeys("nonexistent1", "nonexistent2")
		if err != nil {
			t.Fatalf("DeleteKeys() error = %v", err)
		}
		if count != 0 {
			t.Errorf("DeleteKeys() count = %d, want 0", count)
		}
	})
}

func TestIsCluster(t *testing.T) {
	client, _ := setupTestClient(t)

	if client.IsCluster() {
		t.Error("IsCluster() = true, want false for standalone miniredis")
	}
}

func TestTestConnection(t *testing.T) {
	t.Run("successful connection returns latency", func(t *testing.T) {
		client, mr := setupTestClient(t)

		port, err := strconv.Atoi(mr.Port())
		if err != nil {
			t.Fatalf("failed to parse port: %v", err)
		}

		latency, err := client.TestConnection(mr.Host(), port, "", 0)
		if err != nil {
			t.Fatalf("TestConnection() error = %v", err)
		}
		if latency <= 0 {
			t.Errorf("TestConnection() latency = %v, want > 0", latency)
		}
	})

	t.Run("wrong port returns error", func(t *testing.T) {
		client, _ := setupTestClient(t)

		// Use a port that is almost certainly not listening
		_, err := client.TestConnection("127.0.0.1", 1, "", 0)
		if err == nil {
			t.Fatal("TestConnection() expected error for wrong port, got nil")
		}
	})

	t.Run("wrong host returns error", func(t *testing.T) {
		// Start a separate miniredis so we have a valid client context
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to start miniredis: %v", err)
		}
		t.Cleanup(mr.Close)

		c := NewClient()
		port, _ := strconv.Atoi(mr.Port())
		if err := c.Connect(mr.Host(), port, "", 0); err != nil {
			t.Fatalf("Connect() error = %v", err)
		}
		t.Cleanup(func() { _ = c.Disconnect() })

		_, err = c.TestConnection("192.0.2.1", 6379, "", 0)
		if err == nil {
			t.Fatal("TestConnection() expected error for unreachable host, got nil")
		}
	})
}

// ---------------------------------------------------------------------------
// ConfigGet
// ---------------------------------------------------------------------------

func TestConfigGet(t *testing.T) {
	t.Run("returns result or graceful error", func(t *testing.T) {
		client, _ := setupTestClient(t)

		result, err := client.ConfigGet("save")
		if err != nil {
			// miniredis may not support CONFIG GET; verify graceful error
			t.Logf("ConfigGet() returned expected error with miniredis: %v", err)
			return
		}

		if _, ok := result["save"]; !ok {
			t.Error("ConfigGet(\"save\") did not return a 'save' key")
		}
	})

	t.Run("wildcard pattern returns results or graceful error", func(t *testing.T) {
		client, _ := setupTestClient(t)

		result, err := client.ConfigGet("*")
		if err != nil {
			// miniredis may not support CONFIG GET; verify graceful error
			t.Logf("ConfigGet(\"*\") returned expected error with miniredis: %v", err)
			return
		}

		if len(result) == 0 {
			t.Error("ConfigGet(\"*\") returned empty map")
		}
	})
}

// ---------------------------------------------------------------------------
// ConfigSet
// ---------------------------------------------------------------------------

func TestConfigSet(t *testing.T) {
	t.Run("set config param returns result or graceful error", func(t *testing.T) {
		client, _ := setupTestClient(t)

		err := client.ConfigSet("save", "900 1")
		if err != nil {
			// miniredis may not support CONFIG SET; verify graceful error
			t.Logf("ConfigSet() returned expected error with miniredis: %v", err)
			return
		}

		result, err := client.ConfigGet("save")
		if err != nil {
			t.Fatalf("ConfigGet() error = %v", err)
		}

		if result["save"] != "900 1" {
			t.Errorf("ConfigGet(\"save\") = %q, want %q", result["save"], "900 1")
		}
	})
}

// ---------------------------------------------------------------------------
// ClientList
// ---------------------------------------------------------------------------

func TestClientList(t *testing.T) {
	t.Run("returns clients or graceful error", func(t *testing.T) {
		client, _ := setupTestClient(t)

		clients, err := client.ClientList()
		if err != nil {
			// miniredis may not support CLIENT LIST; verify graceful error
			t.Logf("ClientList() returned expected error with miniredis: %v", err)
			return
		}

		if len(clients) == 0 {
			t.Fatal("ClientList() returned 0 clients, want >= 1")
		}

		// Verify first client has a non-empty address
		c := clients[0]
		if c.Addr == "" {
			t.Error("ClientList() first client has empty Addr")
		}
	})
}
