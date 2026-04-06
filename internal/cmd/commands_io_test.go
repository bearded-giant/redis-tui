package cmd

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestExportKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ExportResult = map[string]any{"key1": "val1", "key2": "val2"}
		dir := t.TempDir()
		filename := dir + "/export.json"
		msg := cmds.ExportKeys("*", filename)()
		result := msg.(types.ExportCompleteMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.KeyCount != 2 {
			t.Errorf("KeyCount = %d, want 2", result.KeyCount)
		}
		if result.Filename != filename {
			t.Errorf("Filename = %q, want %q", result.Filename, filename)
		}
		// Verify the file was actually written
		data, err := os.ReadFile(filename)
		if err != nil {
			t.Fatalf("failed to read exported file: %v", err)
		}
		if len(data) == 0 {
			t.Error("exported file is empty")
		}
	})

	t.Run("export error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ExportError = errors.New("scan failed")
		dir := t.TempDir()
		filename := dir + "/export.json"
		msg := cmds.ExportKeys("*", filename)()
		result := msg.(types.ExportCompleteMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("write error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ExportResult = map[string]any{"key1": "val1"}
		// Use a path that cannot be written to
		filename := "/nonexistent-dir/export.json"
		msg := cmds.ExportKeys("*", filename)()
		result := msg.(types.ExportCompleteMsg)
		if result.Err == nil {
			t.Error("expected error for invalid path")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.ExportKeys("*", "file.json")()
		result := msg.(types.ExportCompleteMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestImportKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ImportResult = 3
		dir := t.TempDir()
		filename := dir + "/import.json"
		err := os.WriteFile(filename, []byte(`{"key1":"val1","key2":"val2","key3":"val3"}`), 0600)
		if err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		msg := cmds.ImportKeys(filename)()
		result := msg.(types.ImportCompleteMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.KeyCount != 3 {
			t.Errorf("KeyCount = %d, want 3", result.KeyCount)
		}
		if result.Filename != filename {
			t.Errorf("Filename = %q, want %q", result.Filename, filename)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.ImportKeys("/nonexistent/import.json")()
		result := msg.(types.ImportCompleteMsg)
		if result.Err == nil {
			t.Error("expected error for missing file")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		cmds, _ := newMockCmds()
		dir := t.TempDir()
		filename := dir + "/bad.json"
		err := os.WriteFile(filename, []byte(`not valid json`), 0600)
		if err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		msg := cmds.ImportKeys(filename)()
		result := msg.(types.ImportCompleteMsg)
		if result.Err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("import error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ImportError = errors.New("import failed")
		dir := t.TempDir()
		filename := dir + "/import.json"
		err := os.WriteFile(filename, []byte(`{"key1":"val1"}`), 0600)
		if err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		msg := cmds.ImportKeys(filename)()
		result := msg.(types.ImportCompleteMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.ImportKeys("file.json")()
		result := msg.(types.ImportCompleteMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestBulkDelete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.BulkDeleteResult = 5
		msg := cmds.BulkDelete("user:*")()
		result := msg.(types.BulkDeleteMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Deleted != 5 {
			t.Errorf("Deleted = %d, want 5", result.Deleted)
		}
		if result.Pattern != "user:*" {
			t.Errorf("Pattern = %q, want %q", result.Pattern, "user:*")
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.BulkDeleteError = errors.New("bulk error")
		msg := cmds.BulkDelete("*")()
		result := msg.(types.BulkDeleteMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.BulkDelete("*")()
		result := msg.(types.BulkDeleteMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestBatchSetTTL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.BatchTTLResult = 3
		msg := cmds.BatchSetTTL("user:*", 60*time.Second)()
		result := msg.(types.BatchTTLSetMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Count != 3 {
			t.Errorf("Count = %d, want 3", result.Count)
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.BatchSetTTL("*", time.Second)()
		result := msg.(types.BatchTTLSetMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestEvalLuaScript(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.EvalResult = "OK"
		msg := cmds.EvalLuaScript("return 'OK'", nil)()
		result := msg.(types.LuaScriptResultMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Result != "OK" {
			t.Errorf("Result = %v, want %q", result.Result, "OK")
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.EvalError = errors.New("script error")
		msg := cmds.EvalLuaScript("bad", nil)()
		result := msg.(types.LuaScriptResultMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.EvalLuaScript("return 1", nil)()
		result := msg.(types.LuaScriptResultMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestJSONPathQuery(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.JSONGetResult = `{"name":"test"}`
		msg := cmds.JSONPathQuery("mykey", "$.name")()
		result := msg.(types.JSONPathResultMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Result != `{"name":"test"}` {
			t.Errorf("Result = %q, want %q", result.Result, `{"name":"test"}`)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.JSONGetError = errors.New("json path error")
		msg := cmds.JSONPathQuery("mykey", "$.bad")()
		result := msg.(types.JSONPathResultMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.JSONPathQuery("mykey", "$.name")()
		result := msg.(types.JSONPathResultMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}
