package testutil

import (
	"errors"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
)

var errConfig = errors.New("config error")

func TestMockConfigClient_NewDefaults(t *testing.T) {
	m := NewMockConfigClient()
	if m == nil {
		t.Fatal("NewMockConfigClient returned nil")
	}
	AssertEqual(t, m.TreeSeparatorResult, ":", "default TreeSeparator")
	AssertEqual(t, m.WatchIntervalResult, time.Second, "default WatchInterval")
}

func TestMockConfigClient_Close(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewMockConfigClient()
		AssertNoError(t, m.Close(), "Close")
	})

	t.Run("error", func(t *testing.T) {
		m := NewMockConfigClient()
		m.CloseError = errConfig
		err := m.Close()
		if !errors.Is(err, errConfig) {
			t.Errorf("expected errConfig, got %v", err)
		}
	})
}

func TestMockConfigClient_ListConnections(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewMockConfigClient()
		conns := []types.Connection{{ID: 1, Name: "test"}}
		m.ListConnectionsResult = conns
		got, err := m.ListConnections()
		AssertNoError(t, err, "ListConnections")
		AssertSliceLen(t, got, 1, "ListConnections result")
		AssertEqual(t, got[0].Name, "test", "connection name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewMockConfigClient()
		m.ListConnectionsError = errConfig
		_, err := m.ListConnections()
		if !errors.Is(err, errConfig) {
			t.Errorf("expected errConfig, got %v", err)
		}
	})

	t.Run("default nil", func(t *testing.T) {
		m := NewMockConfigClient()
		got, err := m.ListConnections()
		AssertNoError(t, err, "ListConnections default")
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}

func TestMockConfigClient_AddConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewMockConfigClient()
		m.AddConnectionResult = types.Connection{ID: 1, Name: "new"}
		got, err := m.AddConnection("new", "localhost", 6379, "", 0, false)
		AssertNoError(t, err, "AddConnection")
		AssertEqual(t, got.Name, "new", "connection name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewMockConfigClient()
		m.AddConnectionError = errConfig
		_, err := m.AddConnection("new", "localhost", 6379, "", 0, false)
		if !errors.Is(err, errConfig) {
			t.Errorf("expected errConfig, got %v", err)
		}
	})
}

func TestMockConfigClient_UpdateConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewMockConfigClient()
		m.UpdateConnectionResult = types.Connection{ID: 1, Name: "updated"}
		got, err := m.UpdateConnection(1, "updated", "localhost", 6379, "", 0, false)
		AssertNoError(t, err, "UpdateConnection")
		AssertEqual(t, got.Name, "updated", "connection name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewMockConfigClient()
		m.UpdateConnectionError = errConfig
		_, err := m.UpdateConnection(1, "updated", "localhost", 6379, "", 0, false)
		if !errors.Is(err, errConfig) {
			t.Errorf("expected errConfig, got %v", err)
		}
	})
}

func TestMockConfigClient_DeleteConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewMockConfigClient()
		AssertNoError(t, m.DeleteConnection(1), "DeleteConnection")
	})

	t.Run("error", func(t *testing.T) {
		m := NewMockConfigClient()
		m.DeleteConnectionError = errConfig
		err := m.DeleteConnection(1)
		if !errors.Is(err, errConfig) {
			t.Errorf("expected errConfig, got %v", err)
		}
	})
}

func TestMockConfigClient_AddFavorite(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewMockConfigClient()
		m.AddFavoriteResult = types.Favorite{Key: "mykey", Label: "test"}
		got, err := m.AddFavorite(1, "mykey", "test")
		AssertNoError(t, err, "AddFavorite")
		AssertEqual(t, got.Key, "mykey", "favorite key")
		AssertEqual(t, got.Label, "test", "favorite label")
	})

	t.Run("error", func(t *testing.T) {
		m := NewMockConfigClient()
		m.AddFavoriteError = errConfig
		_, err := m.AddFavorite(1, "mykey", "test")
		if !errors.Is(err, errConfig) {
			t.Errorf("expected errConfig, got %v", err)
		}
	})
}

func TestMockConfigClient_RemoveFavorite(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewMockConfigClient()
		AssertNoError(t, m.RemoveFavorite(1, "mykey"), "RemoveFavorite")
	})

	t.Run("error", func(t *testing.T) {
		m := NewMockConfigClient()
		m.RemoveFavoriteError = errConfig
		err := m.RemoveFavorite(1, "mykey")
		if !errors.Is(err, errConfig) {
			t.Errorf("expected errConfig, got %v", err)
		}
	})
}

func TestMockConfigClient_ListFavorites(t *testing.T) {
	t.Run("with results", func(t *testing.T) {
		m := NewMockConfigClient()
		m.ListFavoritesResult = []types.Favorite{{Key: "k1"}, {Key: "k2"}}
		got := m.ListFavorites(1)
		AssertSliceLen(t, got, 2, "ListFavorites result")
	})

	t.Run("default nil", func(t *testing.T) {
		m := NewMockConfigClient()
		got := m.ListFavorites(1)
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}

func TestMockConfigClient_IsFavorite(t *testing.T) {
	t.Run("default false", func(t *testing.T) {
		m := NewMockConfigClient()
		AssertEqual(t, m.IsFavorite(1, "key"), false, "default IsFavorite")
	})

	t.Run("configured true", func(t *testing.T) {
		m := NewMockConfigClient()
		m.IsFavoriteResult = true
		AssertEqual(t, m.IsFavorite(1, "key"), true, "IsFavorite set true")
	})
}

func TestMockConfigClient_RecentKeys(t *testing.T) {
	t.Run("AddRecentKey does not panic", func(t *testing.T) {
		m := NewMockConfigClient()
		m.AddRecentKey(1, "key", types.KeyTypeString)
	})

	t.Run("ListRecentKeys with results", func(t *testing.T) {
		m := NewMockConfigClient()
		m.ListRecentKeysResult = []types.RecentKey{{Key: "recent1"}}
		got := m.ListRecentKeys(1)
		AssertSliceLen(t, got, 1, "ListRecentKeys result")
		AssertEqual(t, got[0].Key, "recent1", "recent key name")
	})

	t.Run("ListRecentKeys default nil", func(t *testing.T) {
		m := NewMockConfigClient()
		got := m.ListRecentKeys(1)
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("ClearRecentKeys does not panic", func(t *testing.T) {
		m := NewMockConfigClient()
		m.ClearRecentKeys(1)
	})
}

func TestMockConfigClient_ValueHistory(t *testing.T) {
	t.Run("AddValueHistory does not panic", func(t *testing.T) {
		m := NewMockConfigClient()
		m.AddValueHistory("key", types.RedisValue{StringValue: "val"}, "set")
	})

	t.Run("GetValueHistory with results", func(t *testing.T) {
		m := NewMockConfigClient()
		m.GetValueHistoryResult = []types.ValueHistoryEntry{{Key: "k", Action: "set"}}
		got := m.GetValueHistory("k")
		AssertSliceLen(t, got, 1, "GetValueHistory result")
		AssertEqual(t, got[0].Action, "set", "history action")
	})

	t.Run("GetValueHistory default nil", func(t *testing.T) {
		m := NewMockConfigClient()
		got := m.GetValueHistory("k")
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("ClearValueHistory does not panic", func(t *testing.T) {
		m := NewMockConfigClient()
		m.ClearValueHistory()
	})
}

func TestMockConfigClient_Templates(t *testing.T) {
	t.Run("ListTemplates with results", func(t *testing.T) {
		m := NewMockConfigClient()
		m.ListTemplatesResult = []types.KeyTemplate{{Name: "tmpl1"}}
		got := m.ListTemplates()
		AssertSliceLen(t, got, 1, "ListTemplates result")
		AssertEqual(t, got[0].Name, "tmpl1", "template name")
	})

	t.Run("ListTemplates default nil", func(t *testing.T) {
		m := NewMockConfigClient()
		got := m.ListTemplates()
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("AddTemplate success", func(t *testing.T) {
		m := NewMockConfigClient()
		AssertNoError(t, m.AddTemplate(types.KeyTemplate{Name: "t"}), "AddTemplate")
	})

	t.Run("AddTemplate error", func(t *testing.T) {
		m := NewMockConfigClient()
		m.AddTemplateError = errConfig
		err := m.AddTemplate(types.KeyTemplate{})
		if !errors.Is(err, errConfig) {
			t.Errorf("expected errConfig, got %v", err)
		}
	})

	t.Run("DeleteTemplate success", func(t *testing.T) {
		m := NewMockConfigClient()
		AssertNoError(t, m.DeleteTemplate("tmpl1"), "DeleteTemplate")
	})

	t.Run("DeleteTemplate error", func(t *testing.T) {
		m := NewMockConfigClient()
		m.DeleteTemplateError = errConfig
		err := m.DeleteTemplate("tmpl1")
		if !errors.Is(err, errConfig) {
			t.Errorf("expected errConfig, got %v", err)
		}
	})
}

func TestMockConfigClient_Groups(t *testing.T) {
	t.Run("ListGroups with results", func(t *testing.T) {
		m := NewMockConfigClient()
		m.ListGroupsResult = []types.ConnectionGroup{{Name: "prod"}}
		got := m.ListGroups()
		AssertSliceLen(t, got, 1, "ListGroups result")
		AssertEqual(t, got[0].Name, "prod", "group name")
	})

	t.Run("ListGroups default nil", func(t *testing.T) {
		m := NewMockConfigClient()
		got := m.ListGroups()
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("AddGroup success", func(t *testing.T) {
		m := NewMockConfigClient()
		AssertNoError(t, m.AddGroup("prod", "red"), "AddGroup")
	})

	t.Run("AddGroup error", func(t *testing.T) {
		m := NewMockConfigClient()
		m.AddGroupError = errConfig
		err := m.AddGroup("prod", "red")
		if !errors.Is(err, errConfig) {
			t.Errorf("expected errConfig, got %v", err)
		}
	})

	t.Run("AddConnectionToGroup success", func(t *testing.T) {
		m := NewMockConfigClient()
		AssertNoError(t, m.AddConnectionToGroup("prod", 1), "AddConnectionToGroup")
	})

	t.Run("AddConnectionToGroup error", func(t *testing.T) {
		m := NewMockConfigClient()
		m.AddConnectionToGrpError = errConfig
		err := m.AddConnectionToGroup("prod", 1)
		if !errors.Is(err, errConfig) {
			t.Errorf("expected errConfig, got %v", err)
		}
	})

	t.Run("RemoveConnectionFromGroup success", func(t *testing.T) {
		m := NewMockConfigClient()
		AssertNoError(t, m.RemoveConnectionFromGroup("prod", 1), "RemoveConnectionFromGroup")
	})

	t.Run("RemoveConnectionFromGroup error", func(t *testing.T) {
		m := NewMockConfigClient()
		m.RemoveConnFromGrpError = errConfig
		err := m.RemoveConnectionFromGroup("prod", 1)
		if !errors.Is(err, errConfig) {
			t.Errorf("expected errConfig, got %v", err)
		}
	})
}

func TestMockConfigClient_KeyBindings(t *testing.T) {
	t.Run("GetKeyBindings default", func(t *testing.T) {
		m := NewMockConfigClient()
		got := m.GetKeyBindings()
		// Default is zero value
		AssertEqual(t, got.Up, "", "default Up binding")
	})

	t.Run("GetKeyBindings configured", func(t *testing.T) {
		m := NewMockConfigClient()
		m.KeyBindingsResult = types.DefaultKeyBindings()
		got := m.GetKeyBindings()
		AssertEqual(t, got.Up, "k", "Up binding")
		AssertEqual(t, got.Quit, "q", "Quit binding")
	})

	t.Run("SetKeyBindings success", func(t *testing.T) {
		m := NewMockConfigClient()
		AssertNoError(t, m.SetKeyBindings(types.DefaultKeyBindings()), "SetKeyBindings")
	})

	t.Run("SetKeyBindings error", func(t *testing.T) {
		m := NewMockConfigClient()
		m.SetKeyBindingsError = errConfig
		err := m.SetKeyBindings(types.KeyBindings{})
		if !errors.Is(err, errConfig) {
			t.Errorf("expected errConfig, got %v", err)
		}
	})

	t.Run("ResetKeyBindings success", func(t *testing.T) {
		m := NewMockConfigClient()
		AssertNoError(t, m.ResetKeyBindings(), "ResetKeyBindings")
	})

	t.Run("ResetKeyBindings error", func(t *testing.T) {
		m := NewMockConfigClient()
		m.ResetKeyBindingsError = errConfig
		err := m.ResetKeyBindings()
		if !errors.Is(err, errConfig) {
			t.Errorf("expected errConfig, got %v", err)
		}
	})
}

func TestMockConfigClient_TreeSeparator(t *testing.T) {
	t.Run("GetTreeSeparator default", func(t *testing.T) {
		m := NewMockConfigClient()
		AssertEqual(t, m.GetTreeSeparator(), ":", "default separator")
	})

	t.Run("GetTreeSeparator configured", func(t *testing.T) {
		m := NewMockConfigClient()
		m.TreeSeparatorResult = "/"
		AssertEqual(t, m.GetTreeSeparator(), "/", "configured separator")
	})

	t.Run("SetTreeSeparator success", func(t *testing.T) {
		m := NewMockConfigClient()
		AssertNoError(t, m.SetTreeSeparator("/"), "SetTreeSeparator")
	})

	t.Run("SetTreeSeparator error", func(t *testing.T) {
		m := NewMockConfigClient()
		m.SetTreeSeparatorError = errConfig
		err := m.SetTreeSeparator("/")
		if !errors.Is(err, errConfig) {
			t.Errorf("expected errConfig, got %v", err)
		}
	})
}

func TestMockConfigClient_WatchInterval(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		m := NewMockConfigClient()
		AssertEqual(t, m.GetWatchInterval(), time.Second, "default WatchInterval")
	})

	t.Run("configured", func(t *testing.T) {
		m := NewMockConfigClient()
		m.WatchIntervalResult = 5 * time.Second
		AssertEqual(t, m.GetWatchInterval(), 5*time.Second, "configured WatchInterval")
	})
}
