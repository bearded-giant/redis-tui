package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestAddKeyFieldCount(t *testing.T) {
	tests := []struct {
		kt       types.KeyType
		expected int
	}{
		{types.KeyTypeString, 2},
		{types.KeyTypeList, 2},
		{types.KeyTypeSet, 2},
		{types.KeyTypeZSet, 3},
		{types.KeyTypeHash, 3},
		{types.KeyTypeStream, 3},
		{types.KeyTypeGeo, 3},
		{types.KeyTypeJSON, 2},
	}
	for _, tt := range tests {
		t.Run(string(tt.kt), func(t *testing.T) {
			m, _, _ := newTestModel(t)
			m.AddKeyType = tt.kt
			if got := m.addKeyFieldCount(); got != tt.expected {
				t.Errorf("%v: expected %d, got %d", tt.kt, tt.expected, got)
			}
		})
	}
}

func TestHandleAddKeyScreen(t *testing.T) {
	t.Run("tab advances", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleAddKeyScreen(tea.KeyMsg{Type: tea.KeyTab})
		if result.(Model).AddKeyFocusIdx != 1 {
			t.Errorf("expected 1, got %d", result.(Model).AddKeyFocusIdx)
		}
	})
	t.Run("shift+tab back wraps", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleAddKeyScreen(tea.KeyMsg{Type: tea.KeyShiftTab})
		if result.(Model).AddKeyFocusIdx != 1 {
			t.Errorf("expected wrap to 1, got %d", result.(Model).AddKeyFocusIdx)
		}
	})
	t.Run("ctrl+t cycles type", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.AddKeyType = types.KeyTypeString
		result, _ := m.handleAddKeyScreen(tea.KeyMsg{Type: tea.KeyCtrlT})
		if result.(Model).AddKeyType == types.KeyTypeString {
			t.Error("expected type to change")
		}
	})
	t.Run("ctrl+t adjusts focus on shrink", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.AddKeyType = types.KeyTypeHash // 3 fields
		m.AddKeyFocusIdx = 2
		_, _ = m.handleAddKeyScreen(tea.KeyMsg{Type: tea.KeyCtrlT}) // advances to Stream (still 3) — OK
		// Try again to eventually get a 2-field type
	})
	t.Run("enter submits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.AddKeyInputs[0].SetValue("key1")
		_, cmd := m.handleAddKeyScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected create cmd")
		}
	})
	t.Run("enter with 3 fields", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.AddKeyType = types.KeyTypeHash
		m.AddKeyInputs[0].SetValue("key1")
		_, cmd := m.handleAddKeyScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleAddKeyScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleAddKeyScreen(tea.KeyMsg{Type: tea.KeyEsc})
		if result.(Model).Screen != types.ScreenKeys {
			t.Errorf("expected ScreenKeys, got %v", result.(Model).Screen)
		}
	})
	t.Run("default input", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleAddKeyScreen(keyMsg('a'))
	})
}

func TestHandleConfirmDeleteScreen(t *testing.T) {
	t.Run("y confirm connection", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConfirmType = "connection"
		m.ConfirmData = types.Connection{ID: 1}
		_, cmd := m.handleConfirmDeleteScreen(keyMsg('y'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("y confirm key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConfirmType = "key"
		m.ConfirmData = types.RedisKey{Key: "foo"}
		_, cmd := m.handleConfirmDeleteScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("y confirm flushdb", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConfirmType = "flushdb"
		_, cmd := m.handleConfirmDeleteScreen(keyMsg('Y'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("n cancel connection", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConfirmType = "connection"
		result, _ := m.handleConfirmDeleteScreen(keyMsg('n'))
		if result.(Model).Screen != types.ScreenConnections {
			t.Errorf("expected ScreenConnections, got %v", result.(Model).Screen)
		}
	})
	t.Run("n cancel key with current", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConfirmType = "key"
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		result, _ := m.handleConfirmDeleteScreen(keyMsg('N'))
		if result.(Model).Screen != types.ScreenKeyDetail {
			t.Errorf("expected ScreenKeyDetail, got %v", result.(Model).Screen)
		}
	})
	t.Run("n cancel key without current", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConfirmType = "key"
		result, _ := m.handleConfirmDeleteScreen(tea.KeyMsg{Type: tea.KeyEsc})
		if result.(Model).Screen != types.ScreenKeys {
			t.Errorf("expected ScreenKeys, got %v", result.(Model).Screen)
		}
	})
	t.Run("n cancel flushdb", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConfirmType = "flushdb"
		result, _ := m.handleConfirmDeleteScreen(keyMsg('n'))
		if result.(Model).Screen != types.ScreenKeys {
			t.Errorf("expected ScreenKeys, got %v", result.(Model).Screen)
		}
	})
	t.Run("y bad type assertion", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConfirmType = "connection"
		m.ConfirmData = "not a connection"
		_, cmd := m.handleConfirmDeleteScreen(keyMsg('y'))
		if cmd != nil {
			t.Error("expected nil cmd for bad type")
		}
	})
	t.Run("y bad key type", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConfirmType = "key"
		m.ConfirmData = "not a key"
		_, cmd := m.handleConfirmDeleteScreen(keyMsg('y'))
		if cmd != nil {
			t.Error("expected nil cmd for bad type")
		}
	})
}

func TestHandleTTLEditorScreen(t *testing.T) {
	t.Run("enter valid ttl", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		m.TTLInput.SetValue("30")
		_, cmd := m.handleTTLEditorScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter invalid ttl", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		m.TTLInput.SetValue("abc")
		result, _ := m.handleTTLEditorScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if result.(Model).StatusMsg == "" {
			t.Error("expected error status")
		}
	})
	t.Run("enter no current", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleTTLEditorScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleTTLEditorScreen(tea.KeyMsg{Type: tea.KeyEsc})
	})
	t.Run("default input", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleTTLEditorScreen(keyMsg('5'))
	})
}

func TestHandleEditValueScreen(t *testing.T) {
	t.Run("ctrl+s string", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeString}
		m.VimEditor = createVimEditor("content", 80, 24, "")
		_, cmd := m.handleEditValueScreen(tea.KeyMsg{Type: tea.KeyCtrlS})
		if cmd == nil {
			t.Error("expected save cmd")
		}
	})
	t.Run("ctrl+s json", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeJSON}
		m.VimEditor = createVimEditor(`{"a":1}`, 80, 24, "v.json")
		_, cmd := m.handleEditValueScreen(tea.KeyMsg{Type: tea.KeyCtrlS})
		if cmd == nil {
			t.Error("expected save cmd")
		}
	})
	t.Run("ctrl+s nil key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleEditValueScreen(tea.KeyMsg{Type: tea.KeyCtrlS})
	})
	t.Run("ctrl+q quits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleEditValueScreen(tea.KeyMsg{Type: tea.KeyCtrlQ})
		if result.(Model).Screen != types.ScreenKeyDetail {
			t.Errorf("expected ScreenKeyDetail, got %v", result.(Model).Screen)
		}
	})
	t.Run("default delegates to vim", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.VimEditor = createVimEditor("x", 80, 24, "")
		_, _ = m.handleEditValueScreen(keyMsg('a'))
	})
	t.Run("default nil editor", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleEditValueScreen(keyMsg('a'))
	})
}

func TestHandleAddToCollectionScreen(t *testing.T) {
	setup := func(t *testing.T, kt types.KeyType) Model {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: kt}
		return m
	}

	t.Run("tab advances", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyTab})
		if result.(Model).AddCollFocusIdx != 1 {
			t.Errorf("expected 1, got %d", result.(Model).AddCollFocusIdx)
		}
	})
	t.Run("shift+tab wraps", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyShiftTab})
		if result.(Model).AddCollFocusIdx != 1 {
			t.Errorf("expected 1, got %d", result.(Model).AddCollFocusIdx)
		}
	})
	t.Run("enter list", func(t *testing.T) {
		m := setup(t, types.KeyTypeList)
		m.AddCollectionInput[0].SetValue("v")
		_, cmd := m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter set", func(t *testing.T) {
		m := setup(t, types.KeyTypeSet)
		m.AddCollectionInput[0].SetValue("v")
		_, cmd := m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter zset valid", func(t *testing.T) {
		m := setup(t, types.KeyTypeZSet)
		m.AddCollectionInput[0].SetValue("m")
		m.AddCollectionInput[1].SetValue("3.14")
		_, cmd := m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter zset no score", func(t *testing.T) {
		m := setup(t, types.KeyTypeZSet)
		m.AddCollectionInput[0].SetValue("m")
		_, cmd := m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter zset invalid score", func(t *testing.T) {
		m := setup(t, types.KeyTypeZSet)
		m.AddCollectionInput[0].SetValue("m")
		m.AddCollectionInput[1].SetValue("bad")
		result, _ := m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if result.(Model).StatusMsg == "" {
			t.Error("expected error status")
		}
	})
	t.Run("enter hash with value", func(t *testing.T) {
		m := setup(t, types.KeyTypeHash)
		m.AddCollectionInput[0].SetValue("f")
		m.AddCollectionInput[1].SetValue("v")
		_, cmd := m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter hash default value", func(t *testing.T) {
		m := setup(t, types.KeyTypeHash)
		m.AddCollectionInput[0].SetValue("f")
		_, cmd := m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter stream", func(t *testing.T) {
		m := setup(t, types.KeyTypeStream)
		m.AddCollectionInput[0].SetValue("f")
		m.AddCollectionInput[1].SetValue("v")
		_, cmd := m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter hll", func(t *testing.T) {
		m := setup(t, types.KeyTypeHyperLogLog)
		m.AddCollectionInput[0].SetValue("v")
		_, cmd := m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter bitmap valid", func(t *testing.T) {
		m := setup(t, types.KeyTypeBitmap)
		m.AddCollectionInput[0].SetValue("5")
		_, cmd := m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter bitmap invalid", func(t *testing.T) {
		m := setup(t, types.KeyTypeBitmap)
		m.AddCollectionInput[0].SetValue("bad")
		result, _ := m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if result.(Model).StatusMsg == "" {
			t.Error("expected error status")
		}
	})
	t.Run("enter geo valid", func(t *testing.T) {
		m := setup(t, types.KeyTypeGeo)
		m.AddCollectionInput[0].SetValue("place")
		m.AddCollectionInput[1].SetValue("-122.4, 37.7")
		_, cmd := m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter geo bad lon", func(t *testing.T) {
		m := setup(t, types.KeyTypeGeo)
		m.AddCollectionInput[0].SetValue("place")
		m.AddCollectionInput[1].SetValue("bad,37.7")
		result, _ := m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if result.(Model).StatusMsg == "" {
			t.Error("expected error")
		}
	})
	t.Run("enter geo bad lat", func(t *testing.T) {
		m := setup(t, types.KeyTypeGeo)
		m.AddCollectionInput[0].SetValue("place")
		m.AddCollectionInput[1].SetValue("-122.4,bad")
		result, _ := m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if result.(Model).StatusMsg == "" {
			t.Error("expected error")
		}
	})
	t.Run("enter geo no coords", func(t *testing.T) {
		m := setup(t, types.KeyTypeGeo)
		m.AddCollectionInput[0].SetValue("place")
		_, cmd := m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter nil current key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.AddCollectionInput[0].SetValue("x")
		_, cmd := m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleAddToCollectionScreen(tea.KeyMsg{Type: tea.KeyEsc})
	})
	t.Run("default input", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleAddToCollectionScreen(keyMsg('a'))
	})
}

func TestHandleRemoveFromCollectionScreen(t *testing.T) {
	t.Run("up navigates", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.SelectedItemIdx = 1
		result, _ := m.handleRemoveFromCollectionScreen(keyMsg('k'))
		if result.(Model).SelectedItemIdx != 0 {
			t.Errorf("expected 0, got %d", result.(Model).SelectedItemIdx)
		}
	})
	t.Run("down navigates", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeList, ListValue: []string{"a", "b", "c"}}
		result, _ := m.handleRemoveFromCollectionScreen(keyMsg('j'))
		if result.(Model).SelectedItemIdx != 1 {
			t.Errorf("expected 1, got %d", result.(Model).SelectedItemIdx)
		}
	})
	t.Run("remove list", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeList}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeList, ListValue: []string{"a"}}
		_, cmd := m.handleRemoveFromCollectionScreen(keyMsg('d'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("remove set", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeSet}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeSet, SetValue: []string{"a"}}
		_, cmd := m.handleRemoveFromCollectionScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("remove zset", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeZSet}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeZSet, ZSetValue: []types.ZSetMember{{Member: "a"}}}
		_, cmd := m.handleRemoveFromCollectionScreen(keyMsg('d'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("remove hash", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeHash}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeHash, HashValue: map[string]string{"k": "v"}}
		_, cmd := m.handleRemoveFromCollectionScreen(keyMsg('d'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("remove stream", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeStream}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeStream, StreamValue: []types.StreamEntry{{ID: "1"}}}
		_, cmd := m.handleRemoveFromCollectionScreen(keyMsg('d'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("remove geo", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeGeo}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeGeo, GeoValue: []types.GeoMember{{Name: "a"}}}
		_, cmd := m.handleRemoveFromCollectionScreen(keyMsg('d'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("remove out of bounds", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeList}
		m.SelectedItemIdx = 10
		_, _ = m.handleRemoveFromCollectionScreen(keyMsg('d'))
	})
	t.Run("remove nil key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleRemoveFromCollectionScreen(keyMsg('d'))
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleRemoveFromCollectionScreen(tea.KeyMsg{Type: tea.KeyEsc})
	})
}

func TestHandleRenameKeyScreen(t *testing.T) {
	t.Run("enter valid rename", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "old"}
		m.RenameInput.SetValue("new")
		_, cmd := m.handleRenameKeyScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter same name", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		m.RenameInput.SetValue("foo")
		_, cmd := m.handleRenameKeyScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd for same name")
		}
	})
	t.Run("enter empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		_, cmd := m.handleRenameKeyScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleRenameKeyScreen(tea.KeyMsg{Type: tea.KeyEsc})
	})
	t.Run("default", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleRenameKeyScreen(keyMsg('a'))
	})
}

func TestHandleCopyKeyScreen(t *testing.T) {
	t.Run("enter valid", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		m.CopyInput.SetValue("foo_copy")
		_, cmd := m.handleCopyKeyScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		_, cmd := m.handleCopyKeyScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleCopyKeyScreen(tea.KeyMsg{Type: tea.KeyEsc})
	})
	t.Run("default", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleCopyKeyScreen(keyMsg('a'))
	})
}

func TestHandleBulkDeleteScreen(t *testing.T) {
	t.Run("enter valid", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.BulkDeleteInput.SetValue("user:*")
		_, cmd := m.handleBulkDeleteScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleBulkDeleteScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleBulkDeleteScreen(tea.KeyMsg{Type: tea.KeyEsc})
	})
	t.Run("default", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleBulkDeleteScreen(keyMsg('a'))
	})
}

func TestHandleBatchTTLScreen(t *testing.T) {
	t.Run("tab toggles focus", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.BatchTTLInput.Focus()
		_, _ = m.handleBatchTTLScreen(tea.KeyMsg{Type: tea.KeyTab})
	})
	t.Run("tab from pattern", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.BatchTTLPattern.Focus()
		_, _ = m.handleBatchTTLScreen(tea.KeyMsg{Type: tea.KeyTab})
	})
	t.Run("enter valid", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.BatchTTLInput.SetValue("30")
		m.BatchTTLPattern.SetValue("user:*")
		_, cmd := m.handleBatchTTLScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter invalid ttl", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.BatchTTLInput.SetValue("bad")
		m.BatchTTLPattern.SetValue("user:*")
		_, cmd := m.handleBatchTTLScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("enter empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleBatchTTLScreen(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleBatchTTLScreen(tea.KeyMsg{Type: tea.KeyEsc})
	})
	t.Run("default input to focused", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.BatchTTLInput.Focus()
		_, _ = m.handleBatchTTLScreen(keyMsg('5'))
	})
	t.Run("default input to pattern", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.BatchTTLPattern.Focus()
		_, _ = m.handleBatchTTLScreen(keyMsg('a'))
	})
}
