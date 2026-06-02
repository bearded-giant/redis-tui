package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bearded-giant/redis-tui/internal/types"

	tea "github.com/charmbracelet/bubbletea"
)

func (c *Commands) ExportKeys(pattern, filename string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ExportCompleteMsg{Filename: filename, Err: nil}
		}
		data, err := c.redis.ExportKeys(pattern)
		if err != nil {
			return types.ExportCompleteMsg{Filename: filename, Err: err}
		}

		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return types.ExportCompleteMsg{Filename: filename, Err: err}
		}

		err = os.WriteFile(filename, jsonData, 0600)
		return types.ExportCompleteMsg{Filename: filename, KeyCount: len(data), Err: err}
	}
}

// ExportSingleKey dumps one key (type + ttl + raw + decoded value) to a JSON
// file under ~/redis-tui-exports/. Filename is auto-generated from connection
// name + sanitized key + timestamp.
func (c *Commands) ExportSingleKey(connName, key string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ExportSingleKeyCompleteMsg{Key: key}
		}
		data, err := c.redis.ExportSingleKey(key)
		if err != nil {
			return types.ExportSingleKeyCompleteMsg{Key: key, Err: err}
		}
		data["exported_at"] = time.Now().UTC().Format(time.RFC3339)

		filename, err := singleKeyExportPath(connName, key, time.Now())
		if err != nil {
			return types.ExportSingleKeyCompleteMsg{Key: key, Err: err}
		}
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			return types.ExportSingleKeyCompleteMsg{Key: key, Err: err}
		}

		blob, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return types.ExportSingleKeyCompleteMsg{Key: key, Err: err}
		}
		err = os.WriteFile(filename, blob, 0o600)
		return types.ExportSingleKeyCompleteMsg{Key: key, Filename: filename, Err: err}
	}
}

// singleKeyExportPath builds the on-disk path for a single-key export.
// Exposed at package level (not method) so tests can verify sanitization without
// hitting Redis.
func singleKeyExportPath(connName, key string, now time.Time) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	dir := filepath.Join(home, "redis-tui-exports")
	base := fmt.Sprintf("%s-%s-%s.json",
		sanitizeForFilename(connName),
		sanitizeForFilename(key),
		now.UTC().Format("20060102T150405Z"),
	)
	return filepath.Join(dir, base), nil
}

func sanitizeForFilename(s string) string {
	if s == "" {
		return "unnamed"
	}
	replacer := strings.NewReplacer(
		"/", "_", ":", "_", "*", "_", "?", "_",
		"\"", "_", "<", "_", ">", "_", "|", "_",
		"\\", "_", " ", "_",
	)
	out := replacer.Replace(s)
	if len(out) > 200 {
		out = out[:200]
	}
	return out
}

func (c *Commands) ImportKeys(filename string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ImportCompleteMsg{Filename: filename, Err: nil}
		}

		cleanPath := filepath.Clean(filename)
		jsonData, err := os.ReadFile(cleanPath) // #nosec G304 - user-provided import path is intentional
		if err != nil {
			return types.ImportCompleteMsg{Filename: filename, Err: err}
		}

		var data map[string]any
		if err := json.Unmarshal(jsonData, &data); err != nil {
			return types.ImportCompleteMsg{Filename: filename, Err: err}
		}

		count, err := c.redis.ImportKeys(data)
		return types.ImportCompleteMsg{Filename: filename, KeyCount: count, Err: err}
	}
}

func (c *Commands) BulkDelete(pattern string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.BulkDeleteMsg{Pattern: pattern, Err: nil}
		}
		deleted, err := c.redis.BulkDelete(pattern)
		return types.BulkDeleteMsg{Pattern: pattern, Deleted: deleted, Err: err}
	}
}

func (c *Commands) BatchSetTTL(pattern string, ttl time.Duration) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.BatchTTLSetMsg{Pattern: pattern, Err: nil}
		}
		count, err := c.redis.BatchSetTTL(pattern, ttl)
		return types.BatchTTLSetMsg{Pattern: pattern, Count: count, TTL: ttl, Err: err}
	}
}

// BatchSetTTLPreview counts matched keys + samples without applying TTL.
// Catastrophic-action guard: UI runs this first and only invokes BatchSetTTL
// after the user confirms.
func (c *Commands) BatchSetTTLPreview(pattern string, ttl time.Duration) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.BatchTTLPreviewMsg{Pattern: pattern, TTL: ttl}
		}
		matched, sample, err := c.redis.BatchSetTTLPreview(pattern, 10)
		return types.BatchTTLPreviewMsg{
			Pattern: pattern,
			TTL:     ttl,
			Matched: matched,
			Sample:  sample,
			Err:     err,
		}
	}
}

func (c *Commands) EvalLuaScript(script string, keys []string, args ...any) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.LuaScriptResultMsg{Err: nil}
		}
		result, err := c.redis.Eval(script, keys, args...)
		return types.LuaScriptResultMsg{Result: result, Err: err}
	}
}

func (c *Commands) JSONPathQuery(key, path string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.JSONPathResultMsg{Err: nil}
		}
		result, err := c.redis.JSONGetPath(key, path)
		return types.JSONPathResultMsg{Result: result, Err: err}
	}
}
