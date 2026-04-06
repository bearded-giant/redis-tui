package cmd

import (
	"errors"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestLoadKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		_ = mock.Connect("localhost", 6379, "", 0)
		mock.SetKey("k1", types.RedisValue{}, types.KeyTypeString, 0)
		msg := cmds.LoadKeys("*", 0, 100)()
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
		msg := cmds.LoadKeys("*", 0, 100)()
		result := msg.(types.KeysLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestLoadKeyValue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		_ = mock.Connect("localhost", 6379, "", 0)
		mock.SetKey("mykey", types.RedisValue{Type: types.KeyTypeString, StringValue: "val"}, types.KeyTypeString, 0)
		msg := cmds.LoadKeyValue("mykey")()
		result := msg.(types.KeyValueLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Key != "mykey" {
			t.Errorf("Key = %q, want %q", result.Key, "mykey")
		}
		if result.Value.StringValue != "val" {
			t.Errorf("StringValue = %q, want %q", result.Value.StringValue, "val")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.LoadKeyValue("mykey")()
		result := msg.(types.KeyValueLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestLoadKeyPreview(t *testing.T) {
	cmds, mock := newMockCmds()
	_ = mock.Connect("localhost", 6379, "", 0)
	mock.SetKey("pk", types.RedisValue{Type: types.KeyTypeString, StringValue: "preview"}, types.KeyTypeString, 0)
	msg := cmds.LoadKeyPreview("pk")()
	result := msg.(types.KeyPreviewLoadedMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
	if result.Key != "pk" {
		t.Errorf("Key = %q, want %q", result.Key, "pk")
	}
}

func TestDeleteKey(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		_ = mock.Connect("localhost", 6379, "", 0)
		msg := cmds.DeleteKey("mykey")()
		result := msg.(types.KeyDeletedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Key != "mykey" {
			t.Errorf("Key = %q, want %q", result.Key, "mykey")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.DeleteKey("mykey")()
		result := msg.(types.KeyDeletedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestSetTTL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.SetTTL("mykey", 60*time.Second)()
		result := msg.(types.TTLSetMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.TTL != 60*time.Second {
			t.Errorf("TTL = %v, want %v", result.TTL, 60*time.Second)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.SetTTLError = errors.New("ttl error")
		msg := cmds.SetTTL("mykey", time.Second)()
		result := msg.(types.TTLSetMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.SetTTL("mykey", time.Second)()
		result := msg.(types.TTLSetMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestCreateKey(t *testing.T) {
	keyTypes := []struct {
		name    string
		keyType types.KeyType
		value   string
		extra   string
	}{
		{"string", types.KeyTypeString, "hello", ""},
		{"list", types.KeyTypeList, "item1", ""},
		{"set", types.KeyTypeSet, "member1", ""},
		{"zset", types.KeyTypeZSet, "member1", "1.5"},
		{"hash", types.KeyTypeHash, "value1", "field1"},
		{"stream", types.KeyTypeStream, "value1", "data"},
	}

	for _, tt := range keyTypes {
		t.Run(tt.name, func(t *testing.T) {
			cmds, mock := newMockCmds()
			_ = mock.Connect("localhost", 6379, "", 0)
			msg := cmds.CreateKey("newkey", tt.keyType, tt.value, tt.extra, 0)()
			result := msg.(types.KeySetMsg)
			if result.Err != nil {
				t.Errorf("unexpected error for %s: %v", tt.name, result.Err)
			}
			if result.Key != "newkey" {
				t.Errorf("Key = %q, want %q", result.Key, "newkey")
			}
		})
	}

	t.Run("zset with empty extra defaults to 0", func(t *testing.T) {
		cmds, mock := newMockCmds()
		_ = mock.Connect("localhost", 6379, "", 0)
		msg := cmds.CreateKey("zkey", types.KeyTypeZSet, "member", "", 0)()
		result := msg.(types.KeySetMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("hash with empty extra defaults to 'field'", func(t *testing.T) {
		cmds, mock := newMockCmds()
		_ = mock.Connect("localhost", 6379, "", 0)
		msg := cmds.CreateKey("hkey", types.KeyTypeHash, "val", "", 0)()
		result := msg.(types.KeySetMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("stream with empty extra defaults to 'data'", func(t *testing.T) {
		cmds, mock := newMockCmds()
		_ = mock.Connect("localhost", 6379, "", 0)
		msg := cmds.CreateKey("skey", types.KeyTypeStream, "val", "", 0)()
		result := msg.(types.KeySetMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.CreateKey("k", types.KeyTypeString, "v", "", 0)()
		result := msg.(types.KeySetMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestEditStringValue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.EditStringValue("mykey", "newval")()
		result := msg.(types.ValueEditedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.SetStringError = errors.New("set error")
		msg := cmds.EditStringValue("mykey", "val")()
		result := msg.(types.ValueEditedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.EditStringValue("k", "v")()
		result := msg.(types.ValueEditedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestEditListElement(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.EditListElement("list", 0, "newval")()
	result := msg.(types.ValueEditedMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestEditJSONValue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.EditJSONValue("jsonkey", `{"key":"value"}`)()
		result := msg.(types.ValueEditedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.JSONSetError = errors.New("json set error")
		msg := cmds.EditJSONValue("jsonkey", `{"key":"value"}`)()
		result := msg.(types.ValueEditedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.EditJSONValue("k", "{}")()
		result := msg.(types.ValueEditedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestEditHashField(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.EditHashField("hash", "field", "val")()
	result := msg.(types.ValueEditedMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestRenameKey(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.RenameKey("old", "new")()
		result := msg.(types.KeyRenamedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.OldKey != "old" || result.NewKey != "new" {
			t.Errorf("OldKey=%q NewKey=%q, want old/new", result.OldKey, result.NewKey)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.RenameError = errors.New("rename error")
		msg := cmds.RenameKey("old", "new")()
		result := msg.(types.KeyRenamedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.RenameKey("old", "new")()
		result := msg.(types.KeyRenamedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestCopyKey(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.CopyKey("src", "dst", true)()
		result := msg.(types.KeyCopiedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.SourceKey != "src" || result.DestKey != "dst" {
			t.Errorf("got src=%q dst=%q", result.SourceKey, result.DestKey)
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.CopyKey("src", "dst", false)()
		result := msg.(types.KeyCopiedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestSwitchDB(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.SwitchDB(1)()
		result := msg.(types.DBSwitchedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.DB != 1 {
			t.Errorf("DB = %d, want 1", result.DB)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.SelectDBError = errors.New("select error")
		msg := cmds.SwitchDB(2)()
		result := msg.(types.DBSwitchedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.SwitchDB(0)()
		result := msg.(types.DBSwitchedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}
