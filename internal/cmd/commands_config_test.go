package cmd

import (
	"errors"
	"testing"

	"github.com/davidbudnick/redis-tui/internal/testutil"
	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestLoadFavorites(t *testing.T) {
	cfg := testutil.NewTestConfig(t)
	cmds := NewCommands(cfg, nil)
	msg := cmds.LoadFavorites(1)()
	result := msg.(types.FavoritesLoadedMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestAddFavorite(t *testing.T) {
	cfg := testutil.NewTestConfig(t)
	testutil.MustAddConnection(t, cfg, "test", "localhost", 6379, "", 0)
	cmds := NewCommands(cfg, nil)
	msg := cmds.AddFavorite(1, "mykey", "My Key")()
	result := msg.(types.FavoriteAddedMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestRemoveFavorite(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg := testutil.NewTestConfig(t)
		testutil.MustAddConnection(t, cfg, "test", "localhost", 6379, "", 0)
		_, _ = cfg.AddFavorite(1, "mykey", "label")
		cmds := NewCommands(cfg, nil)
		msg := cmds.RemoveFavorite(1, "mykey")()
		result := msg.(types.FavoriteRemovedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("nil config", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.RemoveFavorite(1, "mykey")()
		result := msg.(types.FavoriteRemovedMsg)
		if result.Err != nil {
			t.Errorf("nil config should not error: %v", result.Err)
		}
	})
}

func TestLoadRecentKeys(t *testing.T) {
	cfg := testutil.NewTestConfig(t)
	cmds := NewCommands(cfg, nil)
	msg := cmds.LoadRecentKeys(1)()
	result := msg.(types.RecentKeysLoadedMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestAddRecentKey(t *testing.T) {
	cfg := testutil.NewTestConfig(t)
	cmds := NewCommands(cfg, nil)
	msg := cmds.AddRecentKey(1, "mykey", types.KeyTypeString)()
	// AddRecentKey returns nil
	if msg != nil {
		t.Errorf("expected nil msg, got %T", msg)
	}
}

func TestLoadTemplates(t *testing.T) {
	cfg := testutil.NewTestConfig(t)
	cmds := NewCommands(cfg, nil)
	msg := cmds.LoadTemplates()()
	result := msg.(types.TemplatesLoadedMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestLoadValueHistory(t *testing.T) {
	cfg := testutil.NewTestConfig(t)
	cmds := NewCommands(cfg, nil)
	msg := cmds.LoadValueHistory("mykey")()
	result := msg.(types.ValueHistoryMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestSaveValueHistory(t *testing.T) {
	cfg := testutil.NewTestConfig(t)
	cmds := NewCommands(cfg, nil)
	msg := cmds.SaveValueHistory("mykey", types.RedisValue{StringValue: "val"}, "set")()
	if msg != nil {
		t.Errorf("expected nil msg, got %T", msg)
	}
}

func TestLoadRedisConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ConfigGetResult = map[string]string{"maxmemory": "0", "hz": "10"}
		msg := cmds.LoadRedisConfig("*")()
		result := msg.(types.ConfigLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Params) != 2 {
			t.Errorf("expected 2 params, got %d", len(result.Params))
		}
		if result.Params["maxmemory"] != "0" {
			t.Errorf("maxmemory = %q, want %q", result.Params["maxmemory"], "0")
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ConfigGetError = errors.New("config error")
		msg := cmds.LoadRedisConfig("*")()
		result := msg.(types.ConfigLoadedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.LoadRedisConfig("*")()
		result := msg.(types.ConfigLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestSetRedisConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.SetRedisConfig("hz", "20")()
		result := msg.(types.ConfigSetMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Param != "hz" {
			t.Errorf("Param = %q, want %q", result.Param, "hz")
		}
		if result.Value != "20" {
			t.Errorf("Value = %q, want %q", result.Value, "20")
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ConfigSetError = errors.New("config set error")
		msg := cmds.SetRedisConfig("hz", "bad")()
		result := msg.(types.ConfigSetMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.SetRedisConfig("hz", "10")()
		result := msg.(types.ConfigSetMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}
