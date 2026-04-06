package cmd

import (
	"testing"

	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestCheckVersion(t *testing.T) {
	t.Run("empty version returns empty msg", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.CheckVersion("")()
		result := msg.(types.UpdateAvailableMsg)
		if result.LatestVersion != "" {
			t.Errorf("LatestVersion = %q, want empty", result.LatestVersion)
		}
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("dev version returns empty msg", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.CheckVersion("dev")()
		result := msg.(types.UpdateAvailableMsg)
		if result.LatestVersion != "" {
			t.Errorf("LatestVersion = %q, want empty", result.LatestVersion)
		}
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})
}

func TestWatchKeyTick(t *testing.T) {
	t.Run("returns non-nil cmd", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		cmd := cmds.WatchKeyTick()
		if cmd == nil {
			t.Error("expected non-nil cmd from WatchKeyTick")
		}
	})
}

func TestCopyToClipboard(t *testing.T) {
	t.Run("returns cmd", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		cmd := cmds.CopyToClipboard("test content")
		if cmd == nil {
			t.Fatal("expected non-nil cmd from CopyToClipboard")
		}
		// Execute the command - it may fail in CI if pbcopy is not available
		msg := cmd()
		result := msg.(types.ClipboardCopiedMsg)
		if result.Content != "test content" {
			t.Errorf("Content = %q, want %q", result.Content, "test content")
		}
		// Note: result.Err may be non-nil if pbcopy is unavailable (e.g. in CI)
	})
}
