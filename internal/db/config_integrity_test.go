package db

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/bearded-giant/redis-tui/internal/types"
)

// --- Integration: Full Connection Lifecycle ---

// TestConfig_Integration_FullConnectionLifecycle exercises the complete lifecycle
// of a connection across persistence boundaries: add -> reload -> update -> reload -> delete -> reload.
// This catches cross-layer breakage that unit tests miss.
func TestConfig_Integration_FullConnectionLifecycle(t *testing.T) {
	cfg := newTestConfig(t)

	// Step 1: Add a connection with all features
	conn, err := cfg.AddConnection(types.Connection{Name: "prod", Host: "redis.example.com", Username: "default", Password: "secret", Port: 6380, DB: 2, UseCluster: true})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	// Set SSH and TLS config
	cfg.mu.Lock()
	for i := range cfg.Connections {
		if cfg.Connections[i].ID == conn.ID {
			cfg.Connections[i].UseSSH = true
			cfg.Connections[i].SSHConfig = &types.SSHConfig{
				Host: "bastion.example.com",
				Port: 22,
				User: "deploy",
			}
			cfg.Connections[i].UseTLS = true
			cfg.Connections[i].TLSConfig = &types.TLSConfig{
				CertFile:   "/certs/client.pem",
				KeyFile:    "/certs/client-key.pem",
				ServerName: "redis.internal",
			}
		}
	}
	cfg.mu.Unlock()
	if err := cfg.save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Add favorites and groups for this connection
	_, err = cfg.AddFavorite(conn.ID, "cache:main", "Main Cache")
	if err != nil {
		t.Fatalf("AddFavorite failed: %v", err)
	}
	err = cfg.AddGroup("Production", "#ff0000")
	if err != nil {
		t.Fatalf("AddGroup failed: %v", err)
	}
	err = cfg.AddConnectionToGroup("Production", conn.ID)
	if err != nil {
		t.Fatalf("AddConnectionToGroup failed: %v", err)
	}
	cfg.AddRecentKey(conn.ID, "user:1", types.KeyTypeHash)

	// Step 2: Reload and verify everything survived
	cfg2 := reloadConfig(t, cfg)

	connections, errList := cfg2.ListConnections()
	if errList != nil {
		t.Fatalf("ListConnections failed: %v", errList)
	}
	if len(connections) != 1 {
		t.Fatalf("expected 1 connection after first reload, got %d", len(connections))
	}
	got := connections[0]
	if got.Name != "prod" || got.Host != "redis.example.com" || got.Port != 6380 || got.Username != "default" || got.DB != 2 || !got.UseCluster {
		t.Errorf("basic fields corrupted after reload: %+v", got)
	}
	if !got.UseSSH || got.SSHConfig == nil || got.SSHConfig.Host != "bastion.example.com" {
		t.Error("SSH config lost after reload")
	}
	if !got.UseTLS || got.TLSConfig == nil || got.TLSConfig.CertFile != "/certs/client.pem" {
		t.Error("TLS config lost after reload")
	}
	if got.Password != "" {
		t.Error("password should have been stripped from disk")
	}

	favs := cfg2.ListFavorites(got.ID)
	if len(favs) != 1 || favs[0].Label != "Main Cache" {
		t.Errorf("favorites lost after reload: %v", favs)
	}

	groups := cfg2.ListGroups()
	if len(groups) != 1 || len(groups[0].Connections) != 1 {
		t.Errorf("groups lost after reload: %v", groups)
	}

	recents := cfg2.ListRecentKeys(got.ID)
	if len(recents) != 1 || recents[0].Type != types.KeyTypeHash {
		t.Errorf("recent keys lost after reload: %v", recents)
	}

	// Step 3: Update the connection and verify preserved fields
	got.Name = "prod-updated"
	got.Host = "redis2.example.com"
	got.Port = 6381
	got.Username = "admin"
	got.DB = 3
	got.UseCluster = false
	updated, err := cfg2.UpdateConnection(got)
	if err != nil {
		t.Fatalf("UpdateConnection failed: %v", err)
	}
	if updated.Name != "prod-updated" {
		t.Errorf("Name not updated: %q", updated.Name)
	}
	if updated.Host != "redis2.example.com" {
		t.Errorf("Host not updated: %q", updated.Host)
	}
	if updated.Port != 6381 {
		t.Errorf("Port not updated: %d", updated.Port)
	}
	if updated.Username != "admin" {
		t.Errorf("Username not updated: %q", updated.Username)
	}
	if updated.DB != 3 {
		t.Errorf("DB not updated: %d", updated.DB)
	}
	if updated.UseCluster {
		t.Error("UseCluster should have been updated to false")
	}
	if !updated.UseSSH || updated.SSHConfig == nil {
		t.Error("SSH config lost after update")
	}
	if !updated.UseTLS || updated.TLSConfig == nil {
		t.Error("TLS config lost after update")
	}

	// Step 4: Reload again and verify update persisted
	cfg3 := reloadConfig(t, cfg2)
	connections3, errList3 := cfg3.ListConnections()
	if errList3 != nil {
		t.Fatalf("ListConnections failed: %v", errList3)
	}
	if connections3[0].Name != "prod-updated" || connections3[0].Host != "redis2.example.com" || connections3[0].Port != 6381 || connections3[0].Username != "admin" || connections3[0].DB != 3 || connections3[0].UseCluster {
		t.Error("update not persisted after reload")
	}
	if !connections3[0].UseSSH || !connections3[0].UseTLS {
		t.Error("SSH/TLS flags lost after update + reload")
	}

	// Step 5: Delete and verify
	err = cfg3.DeleteConnection(connections3[0].ID)
	if err != nil {
		t.Fatalf("DeleteConnection failed: %v", err)
	}

	cfg4 := reloadConfig(t, cfg3)
	connections4, errList4 := cfg4.ListConnections()
	if errList4 != nil {
		t.Fatalf("ListConnections failed: %v", errList4)
	}
	if len(connections4) != 0 {
		t.Errorf("expected 0 connections after delete + reload, got %d", len(connections4))
	}
}

// --- Data Integrity Tests ---

func TestConfig_CorruptedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Write invalid JSON
	err := os.WriteFile(path, []byte(`{broken json!!!`), 0o600)
	if err != nil {
		t.Fatalf("failed to write corrupt config: %v", err)
	}

	_, err = NewConfig(path)
	if err == nil {
		t.Fatal("expected error when loading corrupted JSON config")
	}
}

func TestConfig_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	err := os.WriteFile(path, []byte(``), 0o600)
	if err != nil {
		t.Fatalf("failed to write empty config: %v", err)
	}

	// Empty file should produce an error (invalid JSON), not silently succeed
	_, err = NewConfig(path)
	if err == nil {
		t.Fatal("expected error when loading empty config file")
	}
}

func TestConfig_NextID_AfterReload(t *testing.T) {
	cfg := newTestConfig(t)

	// Add 3 connections, IDs should be 1, 2, 3
	conn1, err := cfg.AddConnection(types.Connection{Name: "a", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	_, err = cfg.AddConnection(types.Connection{Name: "b", Host: "localhost", Port: 6380, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	conn3, err := cfg.AddConnection(types.Connection{Name: "c", Host: "localhost", Port: 6381, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	// Delete the middle one
	err = cfg.DeleteConnection(conn1.ID + 1)
	if err != nil {
		t.Fatalf("DeleteConnection failed: %v", err)
	}

	// Reload — nextID must be > highest existing ID (conn3.ID)
	cfg2 := reloadConfig(t, cfg)

	// Add a new connection — its ID must be higher than conn3
	conn4, err := cfg2.AddConnection(types.Connection{Name: "d", Host: "localhost", Port: 6382, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection after reload failed: %v", err)
	}

	if conn4.ID <= conn3.ID {
		t.Errorf("new connection ID (%d) should be > highest existing ID (%d) — nextID calculation broken", conn4.ID, conn3.ID)
	}
}

func TestConfig_NextID_WithGaps(t *testing.T) {
	// Write a config with a high ID to simulate gaps
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	raw := `{
		"connections": [
			{"id": 100, "name": "high-id", "host": "localhost", "port": 6379, "db": 0, "created_at": "2025-01-01T00:00:00Z", "updated_at": "2025-01-01T00:00:00Z"}
		]
	}`
	err := os.WriteFile(path, []byte(raw), 0o600)
	if err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}

	conn, err := cfg.AddConnection(types.Connection{Name: "new", Host: "localhost", Port: 6380, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	if conn.ID <= 100 {
		t.Errorf("new ID (%d) should be > 100 — nextID must account for existing high IDs", conn.ID)
	}
}

func TestConfig_ConcurrentReadWrite(t *testing.T) {
	cfg := newTestConfig(t)

	// Seed some data
	for i := range 5 {
		_, err := cfg.AddConnection(types.Connection{Name: "conn" + string(rune('a'+i)), Host: "localhost", Port: 6379 + i, DB: 0, UseCluster: false})
		if err != nil {
			t.Fatalf("AddConnection failed: %v", err)
		}
	}

	var wg sync.WaitGroup
	errs := make(chan error, 100)

	// Concurrent writers
	for i := range 10 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cfg.AddRecentKey(1, "key"+string(rune('0'+i)), types.KeyTypeString)
		}(i)
	}

	// Concurrent readers
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cfg.ListConnections()
			if err != nil {
				errs <- err
			}
		}()
	}

	// Concurrent favorites
	for i := range 5 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := cfg.AddFavorite(1, "fav"+string(rune('a'+i)), "label")
			if err != nil {
				errs <- err
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent operation failed: %v", err)
	}
}

func TestConfig_ConcurrentAddDelete(t *testing.T) {
	cfg := newTestConfig(t)

	var wg sync.WaitGroup

	// Add and delete connections concurrently
	for i := range 10 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			conn, err := cfg.AddConnection(types.Connection{Name: "concurrent" + string(rune('a'+i)), Host: "localhost", Port: 6379 + i, DB: 0, UseCluster: false})
			if err != nil {
				return
			}
			_ = cfg.DeleteConnection(conn.ID)
		}(i)
	}

	wg.Wait()

	// After all adds+deletes, state should be consistent
	connections, err := cfg.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections failed: %v", err)
	}

	// Some connections may remain if delete raced with add — that's fine.
	// The key assertion is no panic, no data corruption, and IDs are unique.
	ids := make(map[int64]bool)
	for _, conn := range connections {
		if ids[conn.ID] {
			t.Errorf("duplicate connection ID %d — concurrent operations corrupted state", conn.ID)
		}
		ids[conn.ID] = true
	}
}

func TestConfig_DeleteConnection_OrphanedFavorites(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection(types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	_, err = cfg.AddFavorite(conn.ID, "key1", "label1")
	if err != nil {
		t.Fatalf("AddFavorite failed: %v", err)
	}
	cfg.AddRecentKey(conn.ID, "key1", types.KeyTypeString)

	// Delete the connection
	err = cfg.DeleteConnection(conn.ID)
	if err != nil {
		t.Fatalf("DeleteConnection failed: %v", err)
	}

	// Document current behavior: favorites and recent keys are NOT cleaned up
	// They become orphaned. This test documents the behavior so changes are intentional.
	favs := cfg.ListFavorites(conn.ID)
	recents := cfg.ListRecentKeys(conn.ID)

	// If cleanup is ever added, update these assertions
	if len(favs) != 1 {
		t.Errorf("expected 1 orphaned favorite (current behavior), got %d — was cleanup added?", len(favs))
	}
	if len(recents) != 1 {
		t.Errorf("expected 1 orphaned recent key (current behavior), got %d — was cleanup added?", len(recents))
	}
}
