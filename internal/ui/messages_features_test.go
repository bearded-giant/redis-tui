package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/bearded-giant/redis-tui/internal/types"
)

func TestHandleExportCompleteMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ExportCompleteMsg{Filename: "out.json", KeyCount: 5}
		result, _ := m.handleExportCompleteMsg(msg)
		model := result.(Model)
		if !strings.Contains(model.StatusMsg, "out.json") {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
		if model.Screen != types.ScreenKeys {
			t.Errorf("expected ScreenKeys, got %v", model.Screen)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ExportCompleteMsg{Err: errors.New("boom")}
		result, _ := m.handleExportCompleteMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Export failed:") {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
	})
}

func TestHandleImportCompleteMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ImportCompleteMsg{Filename: "in.json", KeyCount: 10}
		_, cmd := m.handleImportCompleteMsg(msg)
		if cmd == nil {
			t.Error("expected load keys cmd")
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ImportCompleteMsg{Err: errors.New("boom")}
		result, cmd := m.handleImportCompleteMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Import failed:") {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
		if cmd != nil {
			t.Error("expected nil cmd on error")
		}
	})
}

func TestHandleBulkDeleteMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.BulkDeleteMsg{Deleted: 3}
		_, cmd := m.handleBulkDeleteMsg(msg)
		if cmd == nil {
			t.Error("expected reload cmd")
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.BulkDeleteMsg{Err: errors.New("boom")}
		result, cmd := m.handleBulkDeleteMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Bulk delete error:") {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
		if cmd != nil {
			t.Error("expected nil cmd on error")
		}
	})
}

func TestHandleFavoritesLoadedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.FavoritesLoadedMsg{Favorites: []types.Favorite{{Key: "a"}}}
		result, _ := m.handleFavoritesLoadedMsg(msg)
		model := result.(Model)
		if len(model.Favorites) != 1 {
			t.Errorf("expected 1 favorite, got %d", len(model.Favorites))
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.FavoritesLoadedMsg{Err: errors.New("boom")}
		result, _ := m.handleFavoritesLoadedMsg(msg)
		model := result.(Model)
		if len(model.Favorites) != 0 {
			t.Errorf("expected 0 favorites, got %d", len(model.Favorites))
		}
	})
}

func TestHandleFavoriteAddedMsg(t *testing.T) {
	t.Run("success updates key and current", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "foo"}}
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		msg := types.FavoriteAddedMsg{Favorite: types.Favorite{Key: "foo"}}
		result, _ := m.handleFavoriteAddedMsg(msg)
		model := result.(Model)
		if !model.Keys[0].IsFavorite {
			t.Error("expected key marked favorite")
		}
		if !model.CurrentKey.IsFavorite {
			t.Error("expected current key marked favorite")
		}
	})
	t.Run("error ignored", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.FavoriteAddedMsg{Err: errors.New("boom")}
		result, _ := m.handleFavoriteAddedMsg(msg)
		model := result.(Model)
		if model.StatusMsg != "" {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
	})
}

func TestHandleFavoriteRemovedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "foo", IsFavorite: true}}
		m.CurrentKey = &types.RedisKey{Key: "foo", IsFavorite: true}
		msg := types.FavoriteRemovedMsg{Key: "foo"}
		result, _ := m.handleFavoriteRemovedMsg(msg)
		model := result.(Model)
		if model.Keys[0].IsFavorite {
			t.Error("expected key unfavorited")
		}
		if model.CurrentKey.IsFavorite {
			t.Error("expected current key unfavorited")
		}
	})
	t.Run("error ignored", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.FavoriteRemovedMsg{Err: errors.New("boom")}
		_, _ = m.handleFavoriteRemovedMsg(msg)
	})
}

func TestHandleRecentKeysLoadedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.RecentKeysLoadedMsg{Keys: []types.RecentKey{{Key: "a"}}}
		result, _ := m.handleRecentKeysLoadedMsg(msg)
		model := result.(Model)
		if len(model.RecentKeys) != 1 {
			t.Errorf("expected 1 recent, got %d", len(model.RecentKeys))
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.RecentKeysLoadedMsg{Err: errors.New("boom")}
		_, _ = m.handleRecentKeysLoadedMsg(msg)
	})
}

func TestHandleTemplatesLoadedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.TemplatesLoadedMsg{Templates: []types.KeyTemplate{{Name: "t"}}}
		result, _ := m.handleTemplatesLoadedMsg(msg)
		model := result.(Model)
		if len(model.Templates) != 1 {
			t.Errorf("expected 1 template, got %d", len(model.Templates))
		}
		if model.Screen != types.ScreenTemplates {
			t.Errorf("expected ScreenTemplates, got %v", model.Screen)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.TemplatesLoadedMsg{Err: errors.New("boom")}
		_, _ = m.handleTemplatesLoadedMsg(msg)
	})
}

func TestHandleValueHistoryMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ValueHistoryMsg{History: []types.ValueHistoryEntry{{Key: "a"}}}
		result, _ := m.handleValueHistoryMsg(msg)
		model := result.(Model)
		if len(model.ValueHistory) != 1 {
			t.Errorf("expected 1 entry, got %d", len(model.ValueHistory))
		}
		if model.Screen != types.ScreenValueHistory {
			t.Errorf("expected ScreenValueHistory, got %v", model.Screen)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ValueHistoryMsg{Err: errors.New("boom")}
		_, _ = m.handleValueHistoryMsg(msg)
	})
}

func TestHandleRegexSearchResultMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.RegexSearchResultMsg{Keys: []types.RedisKey{{Key: "a"}}}
		result, _ := m.handleRegexSearchResultMsg(msg)
		model := result.(Model)
		if len(model.Keys) != 1 {
			t.Errorf("expected 1 key, got %d", len(model.Keys))
		}
		if model.Screen != types.ScreenKeys {
			t.Errorf("expected ScreenKeys, got %v", model.Screen)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.RegexSearchResultMsg{Err: errors.New("boom")}
		result, _ := m.handleRegexSearchResultMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Regex search error:") {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
	})
}

func TestHandleFuzzySearchResultMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.FuzzySearchResultMsg{Keys: []types.RedisKey{{Key: "a"}}}
		result, _ := m.handleFuzzySearchResultMsg(msg)
		model := result.(Model)
		if len(model.Keys) != 1 {
			t.Errorf("expected 1 key, got %d", len(model.Keys))
		}
		if model.Screen != types.ScreenKeys {
			t.Errorf("expected ScreenKeys, got %v", model.Screen)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.FuzzySearchResultMsg{Err: errors.New("boom")}
		result, _ := m.handleFuzzySearchResultMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Fuzzy search error:") {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
	})
}

func TestHandleCompareKeysResultMsg(t *testing.T) {
	t.Run("equal", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.CompareKeysResultMsg{
			Key1Value: types.RedisValue{StringValue: "same"},
			Key2Value: types.RedisValue{StringValue: "same"},
			Diff:      "",
		}
		result, _ := m.handleCompareKeysResultMsg(msg)
		model := result.(Model)
		if model.CompareResult == nil || !model.CompareResult.Equal {
			t.Error("expected equal comparison")
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.CompareKeysResultMsg{Err: errors.New("boom")}
		result, _ := m.handleCompareKeysResultMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Compare error:") {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
	})
}

func TestHandleJSONPathResultMsg(t *testing.T) {
	t.Run("string result", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.JSONPathResultMsg{Result: "hello"}
		result, _ := m.handleJSONPathResultMsg(msg)
		model := result.(Model)
		if model.JSONPathResult != "hello" {
			t.Errorf("unexpected result: %q", model.JSONPathResult)
		}
	})
	t.Run("non-string result", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.JSONPathResultMsg{Result: 42}
		result, _ := m.handleJSONPathResultMsg(msg)
		model := result.(Model)
		if model.JSONPathResult != "42" {
			t.Errorf("unexpected result: %q", model.JSONPathResult)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.JSONPathResultMsg{Err: errors.New("boom")}
		result, _ := m.handleJSONPathResultMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.JSONPathResult, "Error:") {
			t.Errorf("unexpected result: %q", model.JSONPathResult)
		}
	})
}

func TestHandleClipboardCopiedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ClipboardCopiedMsg{Content: "data"}
		result, _ := m.handleClipboardCopiedMsg(msg)
		model := result.(Model)
		if model.StatusMsg != "Copied to clipboard" {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ClipboardCopiedMsg{Err: errors.New("boom")}
		result, _ := m.handleClipboardCopiedMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Copy failed:") {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
	})
}
