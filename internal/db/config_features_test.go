package db

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
)

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
	_, err = cfg1.AddConnection(types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
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
	conn, errConn := cfg.AddConnection(types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
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
