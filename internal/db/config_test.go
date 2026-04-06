package db

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestNewConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("NewConfig returned nil")
	}

	// Check defaults
	if cfg.TreeSeparator != ":" {
		t.Errorf("TreeSeparator = %q, want \":\"", cfg.TreeSeparator)
	}
	if cfg.MaxRecentKeys != 20 {
		t.Errorf("MaxRecentKeys = %d, want 20", cfg.MaxRecentKeys)
	}
	if cfg.MaxValueHistory != 50 {
		t.Errorf("MaxValueHistory = %d, want 50", cfg.MaxValueHistory)
	}
	if cfg.WatchInterval != 1000 {
		t.Errorf("WatchInterval = %d, want 1000", cfg.WatchInterval)
	}
}

func TestNewConfig_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	nestedPath := filepath.Join(dir, "subdir", "config.json")

	_, err := NewConfig(nestedPath)
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}

	// Check directory was created
	if _, err := os.Stat(filepath.Dir(nestedPath)); os.IsNotExist(err) {
		t.Error("NewConfig did not create directory")
	}
}

func TestConfig_AddConnection(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection("test", "localhost", 6379, "secret", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	if conn.ID == 0 {
		t.Error("Connection ID should not be 0")
	}
	if conn.Name != "test" {
		t.Errorf("Name = %q, want \"test\"", conn.Name)
	}
	if conn.Host != "localhost" {
		t.Errorf("Host = %q, want \"localhost\"", conn.Host)
	}
	if conn.Port != 6379 {
		t.Errorf("Port = %d, want 6379", conn.Port)
	}
	if conn.Password != "secret" {
		t.Errorf("Password = %q, want \"secret\"", conn.Password)
	}
	if conn.DB != 0 {
		t.Errorf("DB = %d, want 0", conn.DB)
	}
	if conn.Created.IsZero() {
		t.Error("Created should not be zero")
	}
}

func TestConfig_AddConnection_IncrementingIDs(t *testing.T) {
	cfg := newTestConfig(t)

	conn1, err := cfg.AddConnection("test1", "localhost", 6379, "", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	conn2, err := cfg.AddConnection("test2", "localhost", 6380, "", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	conn3, err := cfg.AddConnection("test3", "localhost", 6381, "", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	if conn2.ID <= conn1.ID {
		t.Errorf("conn2.ID (%d) should be greater than conn1.ID (%d)", conn2.ID, conn1.ID)
	}
	if conn3.ID <= conn2.ID {
		t.Errorf("conn3.ID (%d) should be greater than conn2.ID (%d)", conn3.ID, conn2.ID)
	}
}

func TestConfig_ListConnections(t *testing.T) {
	cfg := newTestConfig(t)

	// Add connections in non-alphabetical order
	_, err := cfg.AddConnection("zebra", "localhost", 6379, "", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	_, err = cfg.AddConnection("alpha", "localhost", 6380, "", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	_, err = cfg.AddConnection("beta", "localhost", 6381, "", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	connections, errList := cfg.ListConnections()
	if errList != nil {
		t.Fatalf("ListConnections failed: %v", errList)
	}

	if len(connections) != 3 {
		t.Fatalf("Expected 3 connections, got %d", len(connections))
	}

	// Check sorted by ID (insertion order)
	if connections[0].Name != "zebra" {
		t.Errorf("First connection = %q, want \"zebra\"", connections[0].Name)
	}
	if connections[1].Name != "alpha" {
		t.Errorf("Second connection = %q, want \"alpha\"", connections[1].Name)
	}
	if connections[2].Name != "beta" {
		t.Errorf("Third connection = %q, want \"beta\"", connections[2].Name)
	}
}

func TestConfig_UpdateConnection(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection("original", "localhost", 6379, "old", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	originalCreated := conn.Created

	updated, err := cfg.UpdateConnection(conn.ID, "updated", "newhost", 6380, "new", 1, false)
	if err != nil {
		t.Fatalf("UpdateConnection failed: %v", err)
	}

	if updated.Name != "updated" {
		t.Errorf("Name = %q, want \"updated\"", updated.Name)
	}
	if updated.Host != "newhost" {
		t.Errorf("Host = %q, want \"newhost\"", updated.Host)
	}
	if updated.Port != 6380 {
		t.Errorf("Port = %d, want 6380", updated.Port)
	}
	if updated.Password != "new" {
		t.Errorf("Password = %q, want \"new\"", updated.Password)
	}
	if updated.DB != 1 {
		t.Errorf("DB = %d, want 1", updated.DB)
	}
	if !updated.Created.Equal(originalCreated) {
		t.Error("Created timestamp should not change")
	}
	if !updated.Updated.After(updated.Created) {
		t.Error("Updated should be after Created")
	}
}

func TestConfig_UpdateConnection_NotFound(t *testing.T) {
	cfg := newTestConfig(t)

	_, err := cfg.UpdateConnection(999, "test", "localhost", 6379, "", 0, false)
	if !os.IsNotExist(err) {
		t.Errorf("Expected os.ErrNotExist, got %v", err)
	}
}

func TestConfig_DeleteConnection(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection("test", "localhost", 6379, "", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	err = cfg.DeleteConnection(conn.ID)
	if err != nil {
		t.Fatalf("DeleteConnection failed: %v", err)
	}

	connections, errList := cfg.ListConnections()
	if errList != nil {
		t.Fatalf("ListConnections failed: %v", errList)
	}
	if len(connections) != 0 {
		t.Errorf("Expected 0 connections, got %d", len(connections))
	}
}

func TestConfig_DeleteConnection_NotFound(t *testing.T) {
	cfg := newTestConfig(t)

	err := cfg.DeleteConnection(999)
	if !os.IsNotExist(err) {
		t.Errorf("Expected os.ErrNotExist, got %v", err)
	}
}

func TestConfig_Favorites(t *testing.T) {
	cfg := newTestConfig(t)

	// Add favorite
	fav, err := cfg.AddFavorite(1, "user:123", "Test User")
	if err != nil {
		t.Fatalf("AddFavorite failed: %v", err)
	}
	if fav.Key != "user:123" {
		t.Errorf("Key = %q, want \"user:123\"", fav.Key)
	}

	// Check is favorite
	if !cfg.IsFavorite(1, "user:123") {
		t.Error("IsFavorite should return true")
	}
	if cfg.IsFavorite(1, "other:key") {
		t.Error("IsFavorite should return false for non-favorite")
	}

	// List favorites
	favs := cfg.ListFavorites(1)
	if len(favs) != 1 {
		t.Errorf("Expected 1 favorite, got %d", len(favs))
	}

	// Remove favorite
	err = cfg.RemoveFavorite(1, "user:123")
	if err != nil {
		t.Fatalf("RemoveFavorite failed: %v", err)
	}

	if cfg.IsFavorite(1, "user:123") {
		t.Error("IsFavorite should return false after removal")
	}
}

func TestConfig_Favorites_NoDuplicates(t *testing.T) {
	cfg := newTestConfig(t)

	// Add same favorite twice
	_, err := cfg.AddFavorite(1, "user:123", "Label 1")
	if err != nil {
		t.Fatalf("AddFavorite failed: %v", err)
	}
	_, err = cfg.AddFavorite(1, "user:123", "Label 2")
	if err != nil {
		t.Fatalf("AddFavorite failed: %v", err)
	}

	favs := cfg.ListFavorites(1)
	if len(favs) != 1 {
		t.Errorf("Expected 1 favorite (no duplicates), got %d", len(favs))
	}
}

func TestConfig_RecentKeys(t *testing.T) {
	cfg := newTestConfig(t)

	// Add recent keys
	cfg.AddRecentKey(1, "key1", types.KeyTypeString)
	cfg.AddRecentKey(1, "key2", types.KeyTypeHash)
	cfg.AddRecentKey(1, "key3", types.KeyTypeList)

	recents := cfg.ListRecentKeys(1)
	if len(recents) != 3 {
		t.Errorf("Expected 3 recent keys, got %d", len(recents))
	}

	// Most recent should be first
	if recents[0].Key != "key3" {
		t.Errorf("Most recent key = %q, want \"key3\"", recents[0].Key)
	}
}

func TestConfig_RecentKeys_MaxLimit(t *testing.T) {
	cfg := newTestConfig(t)
	cfg.MaxRecentKeys = 3

	// Add more than max
	for i := 0; i < 5; i++ {
		cfg.AddRecentKey(1, "key"+string(rune('a'+i)), types.KeyTypeString)
	}

	recents := cfg.ListRecentKeys(1)
	if len(recents) != 3 {
		t.Errorf("Expected max 3 recent keys, got %d", len(recents))
	}
}

func TestConfig_RecentKeys_MovesToFront(t *testing.T) {
	cfg := newTestConfig(t)

	cfg.AddRecentKey(1, "key1", types.KeyTypeString)
	cfg.AddRecentKey(1, "key2", types.KeyTypeString)
	cfg.AddRecentKey(1, "key1", types.KeyTypeString) // Re-add key1

	recents := cfg.ListRecentKeys(1)
	if recents[0].Key != "key1" {
		t.Errorf("Most recent key = %q, want \"key1\"", recents[0].Key)
	}
	if len(recents) != 2 {
		t.Errorf("Expected 2 recent keys (no duplicates), got %d", len(recents))
	}
}

func TestConfig_Templates(t *testing.T) {
	cfg := newTestConfig(t)

	// List default templates
	templates := cfg.ListTemplates()
	if len(templates) == 0 {
		t.Fatal("Expected default templates")
	}

	// Add new template
	newTemplate := types.KeyTemplate{
		Name:       "Custom",
		KeyPattern: "custom:{id}",
		Type:       types.KeyTypeString,
	}
	err := cfg.AddTemplate(newTemplate)
	if err != nil {
		t.Fatalf("AddTemplate failed: %v", err)
	}

	templates = cfg.ListTemplates()
	found := false
	for _, tmpl := range templates {
		if tmpl.Name == "Custom" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Custom template not found")
	}

	// Delete template
	err = cfg.DeleteTemplate("Custom")
	if err != nil {
		t.Fatalf("DeleteTemplate failed: %v", err)
	}
}

func TestConfig_ValueHistory(t *testing.T) {
	cfg := newTestConfig(t)

	value := types.RedisValue{
		Type:        types.KeyTypeString,
		StringValue: "test value",
	}

	cfg.AddValueHistory("user:123", value, "update")

	history := cfg.GetValueHistory("user:123")
	if len(history) != 1 {
		t.Fatalf("Expected 1 history entry, got %d", len(history))
	}
	if history[0].Action != "update" {
		t.Errorf("Action = %q, want \"update\"", history[0].Action)
	}
}

func TestConfig_ValueHistory_MaxLimit(t *testing.T) {
	cfg := newTestConfig(t)
	cfg.MaxValueHistory = 3

	value := types.RedisValue{Type: types.KeyTypeString, StringValue: "test"}

	for i := 0; i < 5; i++ {
		cfg.AddValueHistory("key", value, "action")
	}

	// Note: GetValueHistory filters by key, but the max limit applies globally
	// The implementation stores all history together
	if len(cfg.ValueHistory) > 3 {
		t.Errorf("Expected max 3 history entries, got %d", len(cfg.ValueHistory))
	}
}

func TestConfig_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Create config and add data
	cfg1, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}
	_, err = cfg1.AddConnection("test", "localhost", 6379, "pass", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	_, err = cfg1.AddFavorite(1, "key1", "label")
	if err != nil {
		t.Fatalf("AddFavorite failed: %v", err)
	}

	// Create new config from same file
	cfg2, err2 := NewConfig(path)
	if err2 != nil {
		t.Fatalf("NewConfig failed: %v", err2)
	}

	// Verify data persisted
	connections, errList := cfg2.ListConnections()
	if errList != nil {
		t.Fatalf("ListConnections failed: %v", errList)
	}
	if len(connections) != 1 {
		t.Errorf("Expected 1 connection after reload, got %d", len(connections))
	}
	if connections[0].Name != "test" {
		t.Errorf("Connection name = %q, want \"test\"", connections[0].Name)
	}
}

func TestConfig_TreeSeparator(t *testing.T) {
	cfg := newTestConfig(t)

	// Default
	if cfg.GetTreeSeparator() != ":" {
		t.Errorf("Default separator = %q, want \":\"", cfg.GetTreeSeparator())
	}

	// Set new separator
	err := cfg.SetTreeSeparator("/")
	if err != nil {
		t.Fatalf("SetTreeSeparator failed: %v", err)
	}

	if cfg.GetTreeSeparator() != "/" {
		t.Errorf("Separator = %q, want \"/\"", cfg.GetTreeSeparator())
	}
}

func TestConfig_WatchInterval(t *testing.T) {
	cfg := newTestConfig(t)

	interval := cfg.GetWatchInterval()
	expected := time.Duration(1000) * time.Millisecond
	if interval != expected {
		t.Errorf("WatchInterval = %v, want %v", interval, expected)
	}
}

func TestConfig_KeyBindings(t *testing.T) {
	cfg := newTestConfig(t)

	// Get default bindings
	bindings := cfg.GetKeyBindings()
	if bindings.Quit == "" {
		t.Error("Quit keybinding should not be empty")
	}

	// Modify and save
	bindings.Quit = "ctrl+x"
	err := cfg.SetKeyBindings(bindings)
	if err != nil {
		t.Fatalf("SetKeyBindings failed: %v", err)
	}

	if cfg.GetKeyBindings().Quit != "ctrl+x" {
		t.Errorf("Quit = %q, want \"ctrl+x\"", cfg.GetKeyBindings().Quit)
	}

	// Reset
	err = cfg.ResetKeyBindings()
	if err != nil {
		t.Fatalf("ResetKeyBindings failed: %v", err)
	}

	if cfg.GetKeyBindings().Quit == "ctrl+x" {
		t.Error("Keybindings should be reset to defaults")
	}
}

func TestConfig_Groups(t *testing.T) {
	cfg := newTestConfig(t)

	// Add group
	err := cfg.AddGroup("Production", "#ff0000")
	if err != nil {
		t.Fatalf("AddGroup failed: %v", err)
	}

	groups := cfg.ListGroups()
	if len(groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(groups))
	}
	if groups[0].Name != "Production" {
		t.Errorf("Group name = %q, want \"Production\"", groups[0].Name)
	}

	// Add connection to group
	conn, errConn := cfg.AddConnection("test", "localhost", 6379, "", 0, false)
	if errConn != nil {
		t.Fatalf("AddConnection failed: %v", errConn)
	}
	err = cfg.AddConnectionToGroup("Production", conn.ID)
	if err != nil {
		t.Fatalf("AddConnectionToGroup failed: %v", err)
	}

	groups = cfg.ListGroups()
	if len(groups[0].Connections) != 1 {
		t.Errorf("Expected 1 connection in group, got %d", len(groups[0].Connections))
	}

	// Remove connection from group
	err = cfg.RemoveConnectionFromGroup("Production", conn.ID)
	if err != nil {
		t.Fatalf("RemoveConnectionFromGroup failed: %v", err)
	}

	groups = cfg.ListGroups()
	if len(groups[0].Connections) != 0 {
		t.Errorf("Expected 0 connections in group, got %d", len(groups[0].Connections))
	}
}

func TestDefaultTemplates(t *testing.T) {
	templates := defaultTemplates()

	if len(templates) == 0 {
		t.Fatal("Expected default templates")
	}

	// Check required templates exist
	requiredNames := []string{"Session", "Cache", "Rate Limit", "Queue", "Leaderboard"}
	for _, name := range requiredNames {
		found := false
		for _, tmpl := range templates {
			if tmpl.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing required template: %q", name)
		}
	}
}

// newTestConfig creates a config for testing with a temp file
func newTestConfig(t *testing.T) *Config {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}
	return cfg
}

// reloadConfig creates a fresh Config from the same file path, forcing a full JSON round-trip.
func reloadConfig(t *testing.T, cfg *Config) *Config {
	t.Helper()
	cfg2, err := NewConfig(cfg.path)
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	return cfg2
}

// --- Phase 1: Connection Persistence Round-Trip ---

func TestConfig_Persistence_AllConnectionFields(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection("prod-redis", "redis.example.com", 6380, "", 2, true)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	cfg2 := reloadConfig(t, cfg)
	connections, err := cfg2.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections failed: %v", err)
	}
	if len(connections) != 1 {
		t.Fatalf("expected 1 connection after reload, got %d", len(connections))
	}

	got := connections[0]
	if got.ID != conn.ID {
		t.Errorf("ID = %d, want %d", got.ID, conn.ID)
	}
	if got.Name != "prod-redis" {
		t.Errorf("Name = %q, want %q", got.Name, "prod-redis")
	}
	if got.Host != "redis.example.com" {
		t.Errorf("Host = %q, want %q", got.Host, "redis.example.com")
	}
	if got.Port != 6380 {
		t.Errorf("Port = %d, want %d", got.Port, 6380)
	}
	if got.DB != 2 {
		t.Errorf("DB = %d, want %d", got.DB, 2)
	}
	if got.UseCluster != true {
		t.Errorf("UseCluster = %v, want true", got.UseCluster)
	}
	if got.Created.IsZero() {
		t.Error("Created should not be zero after reload")
	}
	if got.Updated.IsZero() {
		t.Error("Updated should not be zero after reload")
	}
}

func TestConfig_Persistence_PasswordStripping(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection("secure", "localhost", 6379, "s3cr3t_p@ss", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	// In-memory connection should have the password
	if conn.Password != "s3cr3t_p@ss" {
		t.Errorf("in-memory password = %q, want %q", conn.Password, "s3cr3t_p@ss")
	}

	// Raw JSON file should NOT contain the password
	data, err := os.ReadFile(cfg.path)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	if string(data) == "" {
		t.Fatal("config file is empty")
	}
	if contains(string(data), "s3cr3t_p@ss") {
		t.Error("password should NOT be written to the config file")
	}

	// Reloaded connection should have empty password
	cfg2 := reloadConfig(t, cfg)
	connections, errList := cfg2.ListConnections()
	if errList != nil {
		t.Fatalf("ListConnections failed: %v", errList)
	}
	if connections[0].Password != "" {
		t.Errorf("password after reload = %q, want empty", connections[0].Password)
	}
}

func TestConfig_PasswordFieldCanBeLoaded(t *testing.T) {
	// Verifies that the Password JSON tag can deserialize passwords.
	// If someone changes the tag to json:"-", this test fails because
	// the field can no longer be read from JSON — even though save()
	// intentionally strips it before writing.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Write a config file with a password directly in the JSON
	raw := `{
		"connections": [{
			"id": 1,
			"name": "manual",
			"host": "localhost",
			"port": 6379,
			"password": "loaded_from_json",
			"db": 0,
			"created_at": "2025-01-01T00:00:00Z",
			"updated_at": "2025-01-01T00:00:00Z"
		}]
	}`
	err := os.WriteFile(path, []byte(raw), 0600)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}

	connections, err := cfg.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections failed: %v", err)
	}
	if len(connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(connections))
	}

	// The password field must be readable from JSON — save() strips it,
	// but the struct tag must still support deserialization
	if connections[0].Password != "loaded_from_json" {
		t.Errorf("Password = %q, want %q — the json tag may have been changed to json:\"-\"",
			connections[0].Password, "loaded_from_json")
	}
}

func TestConfig_SSHPasswordFieldCanBeLoaded(t *testing.T) {
	// Same principle: SSH password and passphrase must be loadable from JSON
	// even though save() strips them before writing.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	raw := `{
		"connections": [{
			"id": 1,
			"name": "ssh-test",
			"host": "localhost",
			"port": 6379,
			"db": 0,
			"use_ssh": true,
			"ssh_config": {
				"host": "bastion",
				"port": 22,
				"user": "deploy",
				"password": "ssh_pass_from_json",
				"passphrase": "key_pass_from_json"
			},
			"created_at": "2025-01-01T00:00:00Z",
			"updated_at": "2025-01-01T00:00:00Z"
		}]
	}`
	err := os.WriteFile(path, []byte(raw), 0600)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}

	connections, err := cfg.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections failed: %v", err)
	}
	if connections[0].SSHConfig == nil {
		t.Fatal("SSHConfig should not be nil")
	}
	if connections[0].SSHConfig.Password != "ssh_pass_from_json" {
		t.Errorf("SSHConfig.Password = %q, want %q — the json tag may have been changed to json:\"-\"",
			connections[0].SSHConfig.Password, "ssh_pass_from_json")
	}
	if connections[0].SSHConfig.Passphrase != "key_pass_from_json" {
		t.Errorf("SSHConfig.Passphrase = %q, want %q — the json tag may have been changed to json:\"-\"",
			connections[0].SSHConfig.Passphrase, "key_pass_from_json")
	}
}

func TestConfig_Persistence_SSHPasswordStripping(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection("ssh-conn", "localhost", 6379, "", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	// Manually set SSH config with sensitive fields
	cfg.mu.Lock()
	for i := range cfg.Connections {
		if cfg.Connections[i].ID == conn.ID {
			cfg.Connections[i].UseSSH = true
			cfg.Connections[i].SSHConfig = &types.SSHConfig{
				Host:           "bastion.example.com",
				Port:           22,
				User:           "deploy",
				Password:       "ssh_s3cr3t",
				PrivateKeyPath: "/home/user/.ssh/id_rsa",
				Passphrase:     "k3y_p@ss",
			}
		}
	}
	cfg.mu.Unlock()
	if err := cfg.save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Raw JSON should NOT contain SSH password or passphrase
	data, err := os.ReadFile(cfg.path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if contains(string(data), "ssh_s3cr3t") {
		t.Error("SSH password should NOT be written to config file")
	}
	if contains(string(data), "k3y_p@ss") {
		t.Error("SSH passphrase should NOT be written to config file")
	}

	// Non-sensitive SSH fields should survive reload
	cfg2 := reloadConfig(t, cfg)
	connections, errList := cfg2.ListConnections()
	if errList != nil {
		t.Fatalf("ListConnections failed: %v", errList)
	}
	got := connections[0]
	if !got.UseSSH {
		t.Error("UseSSH should be true after reload")
	}
	if got.SSHConfig == nil {
		t.Fatal("SSHConfig should not be nil after reload")
	}
	if got.SSHConfig.Host != "bastion.example.com" {
		t.Errorf("SSHConfig.Host = %q, want %q", got.SSHConfig.Host, "bastion.example.com")
	}
	if got.SSHConfig.Port != 22 {
		t.Errorf("SSHConfig.Port = %d, want %d", got.SSHConfig.Port, 22)
	}
	if got.SSHConfig.User != "deploy" {
		t.Errorf("SSHConfig.User = %q, want %q", got.SSHConfig.User, "deploy")
	}
	if got.SSHConfig.PrivateKeyPath != "/home/user/.ssh/id_rsa" {
		t.Errorf("SSHConfig.PrivateKeyPath = %q, want %q", got.SSHConfig.PrivateKeyPath, "/home/user/.ssh/id_rsa")
	}
	if got.SSHConfig.Password != "" {
		t.Errorf("SSHConfig.Password should be empty after reload, got %q", got.SSHConfig.Password)
	}
	if got.SSHConfig.Passphrase != "" {
		t.Errorf("SSHConfig.Passphrase should be empty after reload, got %q", got.SSHConfig.Passphrase)
	}
}

func TestConfig_Persistence_TLSConfig(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection("tls-conn", "localhost", 6380, "", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	// Manually set TLS config
	cfg.mu.Lock()
	for i := range cfg.Connections {
		if cfg.Connections[i].ID == conn.ID {
			cfg.Connections[i].UseTLS = true
			cfg.Connections[i].TLSConfig = &types.TLSConfig{
				CertFile:           "/certs/client.pem",
				KeyFile:            "/certs/client-key.pem",
				CAFile:             "/certs/ca.pem",
				InsecureSkipVerify: true,
				ServerName:         "redis.internal",
			}
		}
	}
	cfg.mu.Unlock()
	if err := cfg.save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	cfg2 := reloadConfig(t, cfg)
	connections, err := cfg2.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections failed: %v", err)
	}
	got := connections[0]

	if !got.UseTLS {
		t.Error("UseTLS should be true after reload")
	}
	if got.TLSConfig == nil {
		t.Fatal("TLSConfig should not be nil after reload")
	}
	if got.TLSConfig.CertFile != "/certs/client.pem" {
		t.Errorf("TLSConfig.CertFile = %q, want %q", got.TLSConfig.CertFile, "/certs/client.pem")
	}
	if got.TLSConfig.KeyFile != "/certs/client-key.pem" {
		t.Errorf("TLSConfig.KeyFile = %q, want %q", got.TLSConfig.KeyFile, "/certs/client-key.pem")
	}
	if got.TLSConfig.CAFile != "/certs/ca.pem" {
		t.Errorf("TLSConfig.CAFile = %q, want %q", got.TLSConfig.CAFile, "/certs/ca.pem")
	}
	if !got.TLSConfig.InsecureSkipVerify {
		t.Error("TLSConfig.InsecureSkipVerify should be true after reload")
	}
	if got.TLSConfig.ServerName != "redis.internal" {
		t.Errorf("TLSConfig.ServerName = %q, want %q", got.TLSConfig.ServerName, "redis.internal")
	}
}

// --- Phase 2: UpdateConnection Field Preservation ---

func TestConfig_UpdateConnection_PreservesGroupAndColor(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection("test", "localhost", 6379, "", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	// Set Group and Color directly (not exposed via AddConnection API)
	cfg.mu.Lock()
	for i := range cfg.Connections {
		if cfg.Connections[i].ID == conn.ID {
			cfg.Connections[i].Group = "Production"
			cfg.Connections[i].Color = "#ff0000"
		}
	}
	cfg.mu.Unlock()

	// Update basic fields
	updated, err := cfg.UpdateConnection(conn.ID, "renamed", "newhost", 6380, "pass", 1, false)
	if err != nil {
		t.Fatalf("UpdateConnection failed: %v", err)
	}

	if updated.Group != "Production" {
		t.Errorf("Group = %q, want %q", updated.Group, "Production")
	}
	if updated.Color != "#ff0000" {
		t.Errorf("Color = %q, want %q", updated.Color, "#ff0000")
	}
}

func TestConfig_UpdateConnection_PreservesSSH(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection("test", "localhost", 6379, "", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	sshCfg := &types.SSHConfig{
		Host:           "bastion.example.com",
		Port:           22,
		User:           "deploy",
		PrivateKeyPath: "/home/user/.ssh/id_rsa",
	}
	cfg.mu.Lock()
	for i := range cfg.Connections {
		if cfg.Connections[i].ID == conn.ID {
			cfg.Connections[i].UseSSH = true
			cfg.Connections[i].SSHConfig = sshCfg
		}
	}
	cfg.mu.Unlock()

	updated, err := cfg.UpdateConnection(conn.ID, "renamed", "newhost", 6380, "", 0, false)
	if err != nil {
		t.Fatalf("UpdateConnection failed: %v", err)
	}

	if !updated.UseSSH {
		t.Error("UseSSH should be preserved after update")
	}
	if updated.SSHConfig == nil {
		t.Fatal("SSHConfig should be preserved after update")
	}
	if updated.SSHConfig.Host != "bastion.example.com" {
		t.Errorf("SSHConfig.Host = %q, want %q", updated.SSHConfig.Host, "bastion.example.com")
	}
	if updated.SSHConfig.User != "deploy" {
		t.Errorf("SSHConfig.User = %q, want %q", updated.SSHConfig.User, "deploy")
	}
}

func TestConfig_UpdateConnection_PreservesTLS(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection("test", "localhost", 6379, "", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	tlsCfg := &types.TLSConfig{
		CertFile:           "/certs/client.pem",
		KeyFile:            "/certs/client-key.pem",
		CAFile:             "/certs/ca.pem",
		InsecureSkipVerify: true,
		ServerName:         "redis.internal",
	}
	cfg.mu.Lock()
	for i := range cfg.Connections {
		if cfg.Connections[i].ID == conn.ID {
			cfg.Connections[i].UseTLS = true
			cfg.Connections[i].TLSConfig = tlsCfg
		}
	}
	cfg.mu.Unlock()

	updated, err := cfg.UpdateConnection(conn.ID, "renamed", "newhost", 6380, "", 0, false)
	if err != nil {
		t.Fatalf("UpdateConnection failed: %v", err)
	}

	if !updated.UseTLS {
		t.Error("UseTLS should be preserved after update")
	}
	if updated.TLSConfig == nil {
		t.Fatal("TLSConfig should be preserved after update")
	}
	if updated.TLSConfig.CertFile != "/certs/client.pem" {
		t.Errorf("TLSConfig.CertFile = %q, want %q", updated.TLSConfig.CertFile, "/certs/client.pem")
	}
	if updated.TLSConfig.ServerName != "redis.internal" {
		t.Errorf("TLSConfig.ServerName = %q, want %q", updated.TLSConfig.ServerName, "redis.internal")
	}
}

// --- Phase 3: Remaining Config Persistence ---

func TestConfig_Persistence_Favorites(t *testing.T) {
	cfg := newTestConfig(t)

	_, err := cfg.AddConnection("test", "localhost", 6379, "", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	fav, err := cfg.AddFavorite(1, "user:123", "My User Key")
	if err != nil {
		t.Fatalf("AddFavorite failed: %v", err)
	}

	cfg2 := reloadConfig(t, cfg)
	favs := cfg2.ListFavorites(1)
	if len(favs) != 1 {
		t.Fatalf("expected 1 favorite after reload, got %d", len(favs))
	}

	got := favs[0]
	if got.ConnectionID != 1 {
		t.Errorf("ConnectionID = %d, want %d", got.ConnectionID, 1)
	}
	if got.Key != "user:123" {
		t.Errorf("Key = %q, want %q", got.Key, "user:123")
	}
	if got.Label != "My User Key" {
		t.Errorf("Label = %q, want %q", got.Label, "My User Key")
	}
	if got.AddedAt.IsZero() {
		t.Error("AddedAt should not be zero after reload")
	}
	if got.AddedAt.Sub(fav.AddedAt).Abs() > time.Second {
		t.Errorf("AddedAt drifted: got %v, want ~%v", got.AddedAt, fav.AddedAt)
	}
}

func TestConfig_Persistence_RecentKeys(t *testing.T) {
	cfg := newTestConfig(t)

	cfg.AddRecentKey(1, "key1", types.KeyTypeString)
	cfg.AddRecentKey(1, "key2", types.KeyTypeHash)
	cfg.AddRecentKey(2, "key3", types.KeyTypeList)

	cfg2 := reloadConfig(t, cfg)

	// Check connID=1 keys
	recents1 := cfg2.ListRecentKeys(1)
	if len(recents1) != 2 {
		t.Fatalf("expected 2 recent keys for connID=1 after reload, got %d", len(recents1))
	}
	// Most recent first
	if recents1[0].Key != "key2" {
		t.Errorf("recents1[0].Key = %q, want %q", recents1[0].Key, "key2")
	}
	if recents1[0].Type != types.KeyTypeHash {
		t.Errorf("recents1[0].Type = %q, want %q", recents1[0].Type, types.KeyTypeHash)
	}
	if recents1[1].Key != "key1" {
		t.Errorf("recents1[1].Key = %q, want %q", recents1[1].Key, "key1")
	}
	if recents1[1].Type != types.KeyTypeString {
		t.Errorf("recents1[1].Type = %q, want %q", recents1[1].Type, types.KeyTypeString)
	}

	// Check connID=2 isolation
	recents2 := cfg2.ListRecentKeys(2)
	if len(recents2) != 1 {
		t.Fatalf("expected 1 recent key for connID=2 after reload, got %d", len(recents2))
	}
	if recents2[0].Type != types.KeyTypeList {
		t.Errorf("recents2[0].Type = %q, want %q", recents2[0].Type, types.KeyTypeList)
	}
}

func TestConfig_Persistence_Groups(t *testing.T) {
	cfg := newTestConfig(t)

	err := cfg.AddGroup("Production", "#ff0000")
	if err != nil {
		t.Fatalf("AddGroup failed: %v", err)
	}
	conn, err := cfg.AddConnection("test", "localhost", 6379, "", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	err = cfg.AddConnectionToGroup("Production", conn.ID)
	if err != nil {
		t.Fatalf("AddConnectionToGroup failed: %v", err)
	}

	cfg2 := reloadConfig(t, cfg)
	groups := cfg2.ListGroups()
	if len(groups) != 1 {
		t.Fatalf("expected 1 group after reload, got %d", len(groups))
	}

	got := groups[0]
	if got.Name != "Production" {
		t.Errorf("Name = %q, want %q", got.Name, "Production")
	}
	if got.Color != "#ff0000" {
		t.Errorf("Color = %q, want %q", got.Color, "#ff0000")
	}
	if len(got.Connections) != 1 {
		t.Fatalf("expected 1 connection in group, got %d", len(got.Connections))
	}
	if got.Connections[0] != conn.ID {
		t.Errorf("Connections[0] = %d, want %d", got.Connections[0], conn.ID)
	}
}

func TestConfig_Persistence_Templates(t *testing.T) {
	cfg := newTestConfig(t)

	custom := types.KeyTemplate{
		Name:         "Custom",
		Description:  "A custom template",
		KeyPattern:   "custom:{id}",
		Type:         types.KeyTypeHash,
		DefaultTTL:   5 * time.Minute,
		DefaultValue: "default",
		Fields:       map[string]string{"field1": "val1", "field2": "val2"},
	}
	err := cfg.AddTemplate(custom)
	if err != nil {
		t.Fatalf("AddTemplate failed: %v", err)
	}

	cfg2 := reloadConfig(t, cfg)
	templates := cfg2.ListTemplates()

	var got *types.KeyTemplate
	for _, tmpl := range templates {
		if tmpl.Name == "Custom" {
			got = &tmpl
			break
		}
	}
	if got == nil {
		t.Fatal("custom template not found after reload")
	}
	if got.Description != "A custom template" {
		t.Errorf("Description = %q, want %q", got.Description, "A custom template")
	}
	if got.KeyPattern != "custom:{id}" {
		t.Errorf("KeyPattern = %q, want %q", got.KeyPattern, "custom:{id}")
	}
	if got.Type != types.KeyTypeHash {
		t.Errorf("Type = %q, want %q", got.Type, types.KeyTypeHash)
	}
	if got.DefaultValue != "default" {
		t.Errorf("DefaultValue = %q, want %q", got.DefaultValue, "default")
	}
	if len(got.Fields) != 2 {
		t.Errorf("Fields count = %d, want 2", len(got.Fields))
	}
	if got.Fields["field1"] != "val1" {
		t.Errorf("Fields[field1] = %q, want %q", got.Fields["field1"], "val1")
	}
}

func TestConfig_Persistence_KeyBindings(t *testing.T) {
	cfg := newTestConfig(t)

	bindings := cfg.GetKeyBindings()
	bindings.Quit = "ctrl+x"
	err := cfg.SetKeyBindings(bindings)
	if err != nil {
		t.Fatalf("SetKeyBindings failed: %v", err)
	}

	cfg2 := reloadConfig(t, cfg)
	got := cfg2.GetKeyBindings()
	if got.Quit != "ctrl+x" {
		t.Errorf("Quit = %q, want %q after reload", got.Quit, "ctrl+x")
	}
}

func TestConfig_Persistence_TreeSeparator(t *testing.T) {
	cfg := newTestConfig(t)

	err := cfg.SetTreeSeparator("/")
	if err != nil {
		t.Fatalf("SetTreeSeparator failed: %v", err)
	}

	cfg2 := reloadConfig(t, cfg)
	if cfg2.GetTreeSeparator() != "/" {
		t.Errorf("TreeSeparator = %q, want %q after reload", cfg2.GetTreeSeparator(), "/")
	}
}

func TestConfig_Persistence_ValueHistory(t *testing.T) {
	cfg := newTestConfig(t)

	value := types.RedisValue{
		Type:        types.KeyTypeString,
		StringValue: "hello world",
	}
	cfg.AddValueHistory("user:123", value, "set")

	cfg2 := reloadConfig(t, cfg)
	history := cfg2.GetValueHistory("user:123")
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry after reload, got %d", len(history))
	}

	got := history[0]
	if got.Key != "user:123" {
		t.Errorf("Key = %q, want %q", got.Key, "user:123")
	}
	if got.Action != "set" {
		t.Errorf("Action = %q, want %q", got.Action, "set")
	}
	if got.Value.StringValue != "hello world" {
		t.Errorf("Value.StringValue = %q, want %q", got.Value.StringValue, "hello world")
	}
	if got.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero after reload")
	}
}

func TestConfig_Persistence_Settings(t *testing.T) {
	cfg := newTestConfig(t)

	cfg.mu.Lock()
	cfg.MaxRecentKeys = 50
	cfg.MaxValueHistory = 100
	cfg.WatchInterval = 2000
	cfg.mu.Unlock()
	if err := cfg.save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	cfg2 := reloadConfig(t, cfg)
	if cfg2.MaxRecentKeys != 50 {
		t.Errorf("MaxRecentKeys = %d, want 50 after reload", cfg2.MaxRecentKeys)
	}
	if cfg2.MaxValueHistory != 100 {
		t.Errorf("MaxValueHistory = %d, want 100 after reload", cfg2.MaxValueHistory)
	}
	if cfg2.WatchInterval != 2000 {
		t.Errorf("WatchInterval = %d, want 2000 after reload", cfg2.WatchInterval)
	}
}

// --- Phase 4: Untested Methods ---

func TestConfig_ClearRecentKeys(t *testing.T) {
	cfg := newTestConfig(t)

	cfg.AddRecentKey(1, "key1", types.KeyTypeString)
	cfg.AddRecentKey(1, "key2", types.KeyTypeHash)
	cfg.AddRecentKey(2, "key3", types.KeyTypeList)

	cfg.ClearRecentKeys(1)

	// connID=1 should be empty
	if len(cfg.ListRecentKeys(1)) != 0 {
		t.Errorf("expected 0 recent keys for connID=1 after clear, got %d", len(cfg.ListRecentKeys(1)))
	}

	// connID=2 should be untouched
	if len(cfg.ListRecentKeys(2)) != 1 {
		t.Errorf("expected 1 recent key for connID=2 after clear, got %d", len(cfg.ListRecentKeys(2)))
	}
}

func TestConfig_ClearRecentKeys_Persistence(t *testing.T) {
	cfg := newTestConfig(t)

	cfg.AddRecentKey(1, "key1", types.KeyTypeString)
	cfg.ClearRecentKeys(1)

	cfg2 := reloadConfig(t, cfg)
	if len(cfg2.ListRecentKeys(1)) != 0 {
		t.Errorf("expected 0 recent keys after reload, got %d", len(cfg2.ListRecentKeys(1)))
	}
}

func TestConfig_ClearValueHistory(t *testing.T) {
	cfg := newTestConfig(t)

	value := types.RedisValue{Type: types.KeyTypeString, StringValue: "test"}
	cfg.AddValueHistory("key1", value, "set")
	cfg.AddValueHistory("key2", value, "set")

	cfg.ClearValueHistory()

	if len(cfg.GetValueHistory("key1")) != 0 {
		t.Error("expected empty history for key1 after clear")
	}
	if len(cfg.GetValueHistory("key2")) != 0 {
		t.Error("expected empty history for key2 after clear")
	}
}

func TestConfig_ClearValueHistory_Persistence(t *testing.T) {
	cfg := newTestConfig(t)

	value := types.RedisValue{Type: types.KeyTypeString, StringValue: "test"}
	cfg.AddValueHistory("key1", value, "set")
	cfg.ClearValueHistory()

	cfg2 := reloadConfig(t, cfg)
	if len(cfg2.GetValueHistory("key1")) != 0 {
		t.Errorf("expected empty history after reload, got %d entries", len(cfg2.GetValueHistory("key1")))
	}
}

// --- Integration: Full Connection Lifecycle ---

// TestConfig_Integration_FullConnectionLifecycle exercises the complete lifecycle
// of a connection across persistence boundaries: add → reload → update → reload → delete → reload.
// This catches cross-layer breakage that unit tests miss.
func TestConfig_Integration_FullConnectionLifecycle(t *testing.T) {
	cfg := newTestConfig(t)

	// Step 1: Add a connection with all features
	conn, err := cfg.AddConnection("prod", "redis.example.com", 6380, "secret", 2, true)
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
	if got.Name != "prod" || got.Host != "redis.example.com" || got.Port != 6380 || got.DB != 2 || !got.UseCluster {
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
	updated, err := cfg2.UpdateConnection(got.ID, "prod-updated", "redis2.example.com", 6381, "", 3, false)
	if err != nil {
		t.Fatalf("UpdateConnection failed: %v", err)
	}
	if updated.Name != "prod-updated" {
		t.Errorf("Name not updated: %q", updated.Name)
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
	if connections3[0].Name != "prod-updated" || connections3[0].Host != "redis2.example.com" {
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
	err := os.WriteFile(path, []byte(`{broken json!!!`), 0600)
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

	err := os.WriteFile(path, []byte(``), 0600)
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
	conn1, err := cfg.AddConnection("a", "localhost", 6379, "", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	_, err = cfg.AddConnection("b", "localhost", 6380, "", 0, false)
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	conn3, err := cfg.AddConnection("c", "localhost", 6381, "", 0, false)
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
	conn4, err := cfg2.AddConnection("d", "localhost", 6382, "", 0, false)
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
	err := os.WriteFile(path, []byte(raw), 0600)
	if err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}

	conn, err := cfg.AddConnection("new", "localhost", 6380, "", 0, false)
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
		_, err := cfg.AddConnection("conn"+string(rune('a'+i)), "localhost", 6379+i, "", 0, false)
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
			conn, err := cfg.AddConnection("concurrent"+string(rune('a'+i)), "localhost", 6379+i, "", 0, false)
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

	conn, err := cfg.AddConnection("test", "localhost", 6379, "", 0, false)
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

// contains checks if a string contains a substring (test helper to avoid importing strings).
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
