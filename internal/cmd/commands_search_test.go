package cmd

import (
	"testing"

	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestSearchByValue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.SearchByValueResult = []types.RedisKey{{Key: "found"}}
		msg := cmds.SearchByValue("*", "needle", 100)()
		result := msg.(types.KeysLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Keys) != 1 {
			t.Errorf("expected 1 key, got %d", len(result.Keys))
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.SearchByValue("*", "v", 10)()
		result := msg.(types.KeysLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestRegexSearch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.RegexSearchResult = []types.RedisKey{{Key: "user:123"}}
		msg := cmds.RegexSearch("user:\\d+", 100)()
		result := msg.(types.RegexSearchResultMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Keys) != 1 {
			t.Errorf("expected 1 key, got %d", len(result.Keys))
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.RegexSearch(".*", 10)()
		result := msg.(types.RegexSearchResultMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestFuzzySearch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.FuzzySearchResult = []types.RedisKey{{Key: "user:abc"}}
		msg := cmds.FuzzySearch("usr", 100)()
		result := msg.(types.FuzzySearchResultMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Keys) != 1 {
			t.Errorf("expected 1 key, got %d", len(result.Keys))
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.FuzzySearch("test", 10)()
		result := msg.(types.FuzzySearchResultMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestCompareKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.CompareValue1 = types.RedisValue{StringValue: "val1"}
		mock.CompareValue2 = types.RedisValue{StringValue: "val2"}
		msg := cmds.CompareKeys("k1", "k2")()
		result := msg.(types.CompareKeysResultMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Key1Value.StringValue != "val1" {
			t.Errorf("Key1Value.StringValue = %q, want %q", result.Key1Value.StringValue, "val1")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.CompareKeys("k1", "k2")()
		result := msg.(types.CompareKeysResultMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestLoadKeyPrefixes(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.KeyPrefixesResult = []string{"user:", "session:"}
		msg := cmds.LoadKeyPrefixes(":", 3)()
		result := msg.(types.TreeNodeExpandedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Children) != 2 {
			t.Errorf("expected 2 prefixes, got %d", len(result.Children))
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.LoadKeyPrefixes(":", 3)()
		result := msg.(types.TreeNodeExpandedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}
