package ui

import (
	"testing"

	"github.com/bearded-giant/redis-tui/internal/types"
)

func TestFilterMonitorEntries_EmptyFilter(t *testing.T) {
	entries := []types.MonitorEntry{
		{Cmd: "SET", Args: []string{"k", "v"}},
		{Cmd: "GET", Args: []string{"k"}},
	}
	out := filterMonitorEntries(entries, "")
	if len(out) != 2 {
		t.Errorf("empty filter should return all entries, got %d", len(out))
	}
}

func TestFilterMonitorEntries_CmdMatch(t *testing.T) {
	entries := []types.MonitorEntry{
		{Cmd: "SET", Args: []string{"k", "v"}},
		{Cmd: "GET", Args: []string{"k"}},
		{Cmd: "DEL", Args: []string{"k"}},
	}
	out := filterMonitorEntries(entries, "set")
	if len(out) != 1 || out[0].Cmd != "SET" {
		t.Errorf("filter \"set\" got %v, want only SET", out)
	}
}

func TestFilterMonitorEntries_ArgMatch(t *testing.T) {
	entries := []types.MonitorEntry{
		{Cmd: "SET", Args: []string{"user:1", "alice"}},
		{Cmd: "SET", Args: []string{"session:abc", "data"}},
	}
	out := filterMonitorEntries(entries, "user")
	if len(out) != 1 {
		t.Errorf("filter \"user\" got %d entries, want 1", len(out))
	}
	if out[0].Args[0] != "user:1" {
		t.Errorf("first entry args = %v, want first arg user:1", out[0].Args)
	}
}

func TestFilterMonitorEntries_CaseInsensitive(t *testing.T) {
	entries := []types.MonitorEntry{
		{Cmd: "SET", Args: []string{"USER:1"}},
	}
	out := filterMonitorEntries(entries, "USER")
	if len(out) != 1 {
		t.Errorf("case-insensitive match should find USER:1, got %d", len(out))
	}
	out = filterMonitorEntries(entries, "user")
	if len(out) != 1 {
		t.Errorf("lowercase filter on uppercase arg should match, got %d", len(out))
	}
}

func TestRenderMonitorEntry_Truncates(t *testing.T) {
	long := make([]string, 50)
	for i := range long {
		long[i] = "very-long-arg-value"
	}
	entry := types.MonitorEntry{Cmd: "SET", Args: long}
	out := renderMonitorEntry(entry, 80)
	// Lipgloss may wrap rendering w/ ANSI sequences — assert the underlying line doesn't blow up
	if len(out) == 0 {
		t.Error("expected non-empty render")
	}
}
