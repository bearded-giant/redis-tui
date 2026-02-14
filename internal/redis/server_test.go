package redis

import (
	"testing"
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
