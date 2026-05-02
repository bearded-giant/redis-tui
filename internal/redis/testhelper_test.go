package redis

import (
	"strconv"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/bearded-giant/redis-tui/internal/types"
)

func setupTestClient(t *testing.T) (*Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := NewClient()
	port, _ := strconv.Atoi(mr.Port())
	if err := client.Connect(types.Connection{Name: "test", Host: mr.Host(), Port: port, Password: "", DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })

	return client, mr
}

func setupTestClientWithAuth(t *testing.T, username string, password string) (*Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	mr.RequireUserAuth(username, password)

	client := NewClient()
	port, _ := strconv.Atoi(mr.Port())
	if err := client.Connect(types.Connection{Name: "test", Host: mr.Host(), Port: port, Username: username, Password: password, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })

	return client, mr
}
