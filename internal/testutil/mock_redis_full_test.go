package testutil

import (
	"errors"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
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

// --- Call tracking accumulation ---

func TestFullMockRedisClient_CallTracking(t *testing.T) {
	m := NewFullMockRedisClient()
	_ = m.Connect(types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
	m.SetIncludeTypes(true)
	_ = m.FlushDB()
	_ = m.SelectDB(0)

	expected := []string{"Connect", "SetIncludeTypes", "FlushDB", "SelectDB"}
	AssertSliceLen(t, m.Calls, len(expected), "total calls")
	for i, name := range expected {
		AssertEqual(t, m.Calls[i], name, "call order")
	}
}

// --- Connection methods ---

func TestFullMockRedisClient_Connect(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.Connect(types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
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
		err := m.Connect(types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_ConnectCluster(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.ConnectCluster([]string{"localhost:6379"}, types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
		AssertNoError(t, err, "ConnectCluster")
		AssertEqual(t, m.Calls[0], "ConnectCluster", "call name")
		if !m.IsConnected() {
			t.Error("expected connected")
		}
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ConnectClusterError = errTest
		err := m.ConnectCluster([]string{"localhost:6379"}, types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
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
		latency, err := m.TestConnection(types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
		AssertNoError(t, err, "TestConnection")
		AssertEqual(t, latency, 42*time.Millisecond, "latency")
		AssertEqual(t, m.Calls[0], "TestConnection", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.TestConnectionError = errTest
		_, err := m.TestConnection(types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}
