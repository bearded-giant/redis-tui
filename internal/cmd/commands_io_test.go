package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bearded-giant/redis-tui/internal/types"
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

	t.Run("marshal error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		// channel values cannot be JSON-marshaled
		mock.ExportResult = map[string]any{"bad": make(chan int)}
		dir := t.TempDir()
		filename := dir + "/export.json"
		msg := cmds.ExportKeys("*", filename)()
		result := msg.(types.ExportCompleteMsg)
		if result.Err == nil {
			t.Error("expected marshal error")
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


func TestSanitizeForFilename(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", "unnamed"},
		{"simple", "simple"},
		{"path/with/slashes", "path_with_slashes"},
		{"user:1234", "user_1234"},
		{"weird?name*here", "weird_name_here"},
		{"has spaces", "has_spaces"},
		{`<>|"\:`, "______"},
		{strings.Repeat("a", 300), strings.Repeat("a", 200)},
	}
	for _, c := range cases {
		got := sanitizeForFilename(c.in)
		if got != c.want {
			t.Errorf("sanitizeForFilename(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSingleKeyExportPath(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 30, 45, 0, time.UTC)
	path, err := singleKeyExportPath("prod-redis", "user:1234", now)
	if err != nil {
		t.Fatalf("singleKeyExportPath: %v", err)
	}
	if !strings.Contains(path, "redis-tui-exports") {
		t.Errorf("path missing redis-tui-exports: %q", path)
	}
	base := filepath.Base(path)
	want := "prod-redis-user_1234-20260601T123045Z.json"
	if base != want {
		t.Errorf("basename = %q, want %q", base, want)
	}
}

func TestExportSingleKey(t *testing.T) {
	t.Run("success writes file with decoded payload", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ExportSingleResult = map[string]any{
			"key":           "user:1",
			"type":          "string",
			"ttl_seconds":   nil,
			"value_raw":     "alice",
			"decoded_format": "raw",
		}

		home := t.TempDir()
		t.Setenv("HOME", home)

		msg := cmds.ExportSingleKey("conn1", "user:1")()
		result := msg.(types.ExportSingleKeyCompleteMsg)
		if result.Err != nil {
			t.Fatalf("unexpected error: %v", result.Err)
		}
		if result.Key != "user:1" {
			t.Errorf("Key = %q, want user:1", result.Key)
		}
		if !strings.HasPrefix(result.Filename, filepath.Join(home, "redis-tui-exports")) {
			t.Errorf("Filename prefix wrong: %q", result.Filename)
		}

		data, err := os.ReadFile(result.Filename)
		if err != nil {
			t.Fatalf("read exported file: %v", err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("unmarshal exported file: %v", err)
		}
		if parsed["key"] != "user:1" {
			t.Errorf("file.key = %v, want user:1", parsed["key"])
		}
		if _, ok := parsed["exported_at"]; !ok {
			t.Error("exported_at missing from file")
		}
	})

	t.Run("export error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ExportSingleError = errors.New("nope")
		t.Setenv("HOME", t.TempDir())

		msg := cmds.ExportSingleKey("conn1", "ghost")()
		result := msg.(types.ExportSingleKeyCompleteMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
		if result.Filename != "" {
			t.Errorf("Filename should be empty on error, got %q", result.Filename)
		}
	})

	t.Run("nil redis no-op", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.ExportSingleKey("conn1", "anything")()
		result := msg.(types.ExportSingleKeyCompleteMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}
