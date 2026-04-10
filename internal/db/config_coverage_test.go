package db

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestConfig_Close(t *testing.T) {
	cfg := newTestConfig(t)
	if err := cfg.Close(); err != nil {
		t.Errorf("Close() unexpected error: %v", err)
	}
}

func TestNewConfig_MkdirError(t *testing.T) {
	// Create a file where a directory is expected to force MkdirAll to fail.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	// configPath points to a file nested under the blocker file — MkdirAll
	// will fail because `blocker` is not a directory.
	configPath := filepath.Join(blocker, "sub", "config.json")
	if _, err := NewConfig(configPath); err == nil {
		t.Error("expected NewConfig to error when directory cannot be created")
	}
}

func TestNewConfig_LoadError(t *testing.T) {
	// Write an invalid JSON file so load() returns a non-IsNotExist error.
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte("not json at all"), 0o600); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if _, err := NewConfig(configPath); err == nil {
		t.Error("expected NewConfig to error on invalid JSON")
	}
}

func TestConfig_RemoveFavorite_NotFound(t *testing.T) {
	cfg := newTestConfig(t)
	if err := cfg.RemoveFavorite(1, "missing"); !os.IsNotExist(err) {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}
}

func TestConfig_ListFavorites_EmptyForUnknownConn(t *testing.T) {
	cfg := newTestConfig(t)
	if _, err := cfg.AddFavorite(1, "k", "l"); err != nil {
		t.Fatalf("AddFavorite failed: %v", err)
	}
	if favs := cfg.ListFavorites(99); len(favs) != 0 {
		t.Errorf("expected empty list for unknown conn, got %d", len(favs))
	}
}

func TestConfig_ListFavorites_SortedNewestFirst(t *testing.T) {
	cfg := newTestConfig(t)
	if _, err := cfg.AddFavorite(1, "first", ""); err != nil {
		t.Fatalf("AddFavorite failed: %v", err)
	}
	// Give the second entry a slightly later timestamp.
	if _, err := cfg.AddFavorite(1, "second", ""); err != nil {
		t.Fatalf("AddFavorite failed: %v", err)
	}
	favs := cfg.ListFavorites(1)
	if len(favs) != 2 {
		t.Fatalf("expected 2 favorites, got %d", len(favs))
	}
	// The sort comparator runs only when len>=2; the newest (second) should
	// come first because ListFavorites sorts AddedAt descending.
	if !favs[0].AddedAt.After(favs[1].AddedAt) && !favs[0].AddedAt.Equal(favs[1].AddedAt) {
		t.Errorf("expected newest-first ordering, got %+v", favs)
	}
}

func TestConfig_DeleteTemplate_NotFound(t *testing.T) {
	cfg := newTestConfig(t)
	if err := cfg.DeleteTemplate("Nonexistent"); !os.IsNotExist(err) {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}
}

func TestConfig_AddConnectionToGroup_NotFound(t *testing.T) {
	cfg := newTestConfig(t)
	if err := cfg.AddConnectionToGroup("nope", 1); !os.IsNotExist(err) {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}
}

func TestConfig_AddConnectionToGroup_AlreadyMember(t *testing.T) {
	cfg := newTestConfig(t)
	if err := cfg.AddGroup("g", ""); err != nil {
		t.Fatalf("AddGroup failed: %v", err)
	}
	if err := cfg.AddConnectionToGroup("g", 42); err != nil {
		t.Fatalf("AddConnectionToGroup failed: %v", err)
	}
	// Re-adding the same connection should be a no-op (exercises the early-return).
	if err := cfg.AddConnectionToGroup("g", 42); err != nil {
		t.Fatalf("idempotent add should not error, got %v", err)
	}
	groups := cfg.ListGroups()
	if len(groups) != 1 || len(groups[0].Connections) != 1 {
		t.Errorf("expected single membership, got %+v", groups)
	}
}

func TestConfig_RemoveConnectionFromGroup_NotFound(t *testing.T) {
	cfg := newTestConfig(t)
	if err := cfg.RemoveConnectionFromGroup("nope", 1); !os.IsNotExist(err) {
		t.Errorf("expected os.ErrNotExist for missing group, got %v", err)
	}

	if err := cfg.AddGroup("g", ""); err != nil {
		t.Fatalf("AddGroup failed: %v", err)
	}
	if err := cfg.RemoveConnectionFromGroup("g", 999); !os.IsNotExist(err) {
		t.Errorf("expected os.ErrNotExist for missing member, got %v", err)
	}
}

func TestConfig_GetTreeSeparator_EmptyFallback(t *testing.T) {
	cfg := newTestConfig(t)
	cfg.TreeSeparator = ""
	if got := cfg.GetTreeSeparator(); got != ":" {
		t.Errorf("empty separator should fall back to \":\", got %q", got)
	}
}

func TestConfig_GetWatchInterval_ZeroFallback(t *testing.T) {
	cfg := newTestConfig(t)
	cfg.WatchInterval = 0
	if got := cfg.GetWatchInterval(); got.String() != "1s" {
		t.Errorf("zero interval should fall back to 1s, got %v", got)
	}

	cfg.WatchInterval = -5
	if got := cfg.GetWatchInterval(); got.String() != "1s" {
		t.Errorf("negative interval should fall back to 1s, got %v", got)
	}
}

// TestConfig_SaveRollback verifies that when save() fails, mutating operations
// roll back in-memory state. We make the config path a directory so WriteFile
// fails reliably.
func TestConfig_SaveRollback(t *testing.T) {
	dir := t.TempDir()
	cfg, err := NewConfig(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}

	// Replace the config path with a directory so subsequent save()s fail.
	badPath := filepath.Join(dir, "nowdir")
	if err := os.Mkdir(badPath, 0o750); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	cfg.path = badPath

	t.Run("AddConnection rollback", func(t *testing.T) {
		before := len(cfg.Connections)
		if _, err := cfg.AddConnection(types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false}); err == nil {
			t.Error("expected save error")
		}
		if got := len(cfg.Connections); got != before {
			t.Errorf("expected rollback, len=%d want %d", got, before)
		}
	})

	t.Run("AddFavorite rollback", func(t *testing.T) {
		before := len(cfg.Favorites)
		if _, err := cfg.AddFavorite(1, "k", "l"); err == nil {
			t.Error("expected save error")
		}
		if got := len(cfg.Favorites); got != before {
			t.Errorf("expected rollback, len=%d want %d", got, before)
		}
	})
}

// TestConfig_UpdateConnection_SaveError confirms UpdateConnection rolls back
// the in-memory mutation when save() fails.
func TestConfig_UpdateConnection_SaveError(t *testing.T) {
	dir := t.TempDir()
	cfg, err := NewConfig(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}
	conn, err := cfg.AddConnection(types.Connection{Name: "orig", Host: "h", Port: 1, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	// Break save() by pointing path to a directory.
	badPath := filepath.Join(dir, "blocked")
	if err := os.Mkdir(badPath, 0o750); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	cfg.path = badPath

	conn.Name = "new"
	conn.Host = "h2"
	conn.Port = 2
	conn.DB = 0
	conn.UseCluster = false

	if _, err := cfg.UpdateConnection(conn); err == nil {
		t.Error("expected save error")
	}
	if cfg.Connections[0].Name != "orig" {
		t.Errorf("expected rollback to 'orig', got %q", cfg.Connections[0].Name)
	}
}

// TestConfig_Save_MarshalError forces json.MarshalIndent to fail by injecting
// a NaN float score into ValueHistory. encoding/json rejects NaN/Inf, so save()
// returns an error from the MarshalIndent branch.
func TestConfig_Save_MarshalError(t *testing.T) {
	cfg := newTestConfig(t)

	// Directly seed an unmarshalable entry.
	cfg.ValueHistory = []types.ValueHistoryEntry{{
		Key: "bad",
		Value: types.RedisValue{
			Type: types.KeyTypeZSet,
			ZSetValue: []types.ZSetMember{
				{Member: "nan", Score: math.NaN()},
			},
		},
		Action: "set",
	}}

	// SetTreeSeparator calls save() which should surface the marshal error.
	if err := cfg.SetTreeSeparator("/"); err == nil {
		t.Error("expected save to fail with NaN ZSet score")
	}
}

// TestConfig_IsFavorite_False hits the non-matching branch that existing tests
// cover only in the positive case via the comparison loop.
func TestConfig_IsFavorite_DifferentConn(t *testing.T) {
	cfg := newTestConfig(t)
	if _, err := cfg.AddFavorite(1, "k", "l"); err != nil {
		t.Fatalf("AddFavorite failed: %v", err)
	}
	if cfg.IsFavorite(2, "k") {
		t.Error("favorite scoped to conn=1 should not match conn=2")
	}
	_ = types.KeyTypeString // retain import parity
}
