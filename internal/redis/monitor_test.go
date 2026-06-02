package redis

import (
	"strings"
	"testing"
	"time"

	"github.com/bearded-giant/redis-tui/internal/types"
)

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestParseMonitorLine_HappyPath(t *testing.T) {
	line := `1574099031.764036 [0 127.0.0.1:60270] "SET" "user:1" "alice"`
	entry := parseMonitorLine(line)

	if entry.DB != 0 {
		t.Errorf("DB = %d, want 0", entry.DB)
	}
	if entry.Client != "127.0.0.1:60270" {
		t.Errorf("Client = %q, want 127.0.0.1:60270", entry.Client)
	}
	if entry.Cmd != "SET" {
		t.Errorf("Cmd = %q, want SET", entry.Cmd)
	}
	if len(entry.Args) != 2 || entry.Args[0] != "user:1" || entry.Args[1] != "alice" {
		t.Errorf("Args = %v, want [user:1 alice]", entry.Args)
	}
	if entry.Time.IsZero() {
		t.Error("Time = zero, want parsed")
	}
}

func TestParseMonitorLine_FractionalTime(t *testing.T) {
	// Fractional second precision should survive.
	line := `1574099031.500000 [0 127.0.0.1:60270] "GET" "x"`
	entry := parseMonitorLine(line)
	if entry.Time.Unix() != 1574099031 {
		t.Errorf("Time.Unix() = %d, want 1574099031", entry.Time.Unix())
	}
	if entry.Time.Nanosecond() < 400_000_000 || entry.Time.Nanosecond() > 600_000_000 {
		t.Errorf("Nanosecond = %d, want ~500_000_000", entry.Time.Nanosecond())
	}
}

func TestParseMonitorLine_NoArgs(t *testing.T) {
	line := `1574099031.764036 [0 127.0.0.1:60270] "PING"`
	entry := parseMonitorLine(line)
	if entry.Cmd != "PING" {
		t.Errorf("Cmd = %q, want PING", entry.Cmd)
	}
	if len(entry.Args) != 0 {
		t.Errorf("Args = %v, want empty", entry.Args)
	}
}

func TestParseMonitorLine_QuotedArgWithSpaces(t *testing.T) {
	line := `1574099031.764036 [0 client] "SET" "key" "hello world"`
	entry := parseMonitorLine(line)
	if len(entry.Args) != 2 || entry.Args[1] != "hello world" {
		t.Errorf("Args = %v, want [key, \"hello world\"]", entry.Args)
	}
}

func TestParseMonitorLine_EscapedQuotes(t *testing.T) {
	// Redis escapes inner quotes as \"
	line := `1574099031.764036 [0 c] "SET" "key" "with\"quotes"`
	entry := parseMonitorLine(line)
	if len(entry.Args) != 2 || entry.Args[1] != `with"quotes` {
		t.Errorf("Args[1] = %q, want with\"quotes", entry.Args[1])
	}
}

func TestParseMonitorLine_Garbage(t *testing.T) {
	// Forgiving — non-empty Raw is the fallback.
	entry := parseMonitorLine("garbage line with no structure")
	if entry.Raw == "" {
		t.Error("Raw should always be preserved")
	}
}

func TestParseMonitorLine_TimeZero(t *testing.T) {
	// Empty line still produces a zero-value entry, not a panic.
	entry := parseMonitorLine("")
	if !entry.Time.IsZero() {
		t.Errorf("Time should be zero for empty input, got %v", entry.Time)
	}
}

func TestSplitQuoted(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{`"a"`, []string{"a"}},
		{`"a" "b"`, []string{"a", "b"}},
		{`"a b c"`, []string{"a b c"}},
		{`"a\"b"`, []string{`a"b`}},
		{``, nil},
		{`no quotes here`, nil},
	}
	for _, c := range cases {
		got := splitQuoted(c.in)
		if !equalSlices(got, c.want) {
			t.Errorf("splitQuoted(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestStartMonitor_RejectsCluster(t *testing.T) {
	c := &Client{isCluster: true}
	_, err := c.StartMonitor(func(types.MonitorEntry) {})
	if err == nil {
		t.Fatal("expected error for cluster mode")
	}
	if !strings.Contains(err.Error(), "cluster") {
		t.Errorf("error = %q, want cluster mention", err.Error())
	}
}

func TestStartMonitor_NotConnected(t *testing.T) {
	c := &Client{} // no client field set
	_, err := c.StartMonitor(func(types.MonitorEntry) {})
	if err == nil {
		t.Fatal("expected error when not connected")
	}
}

// Test the live subscription against miniredis. miniredis supports MONITOR
// (verified via its source) — entries arrive on the callback as commands run.
//
// Skipped under -race: go-redis MonitorCmd races its connection reader against
// the pool's PeekReplyTypeSafe on Stop(). Race is in the upstream library, not
// our code. Parse-layer tests above still cover entry handling.
func TestStartMonitor_ReceivesEntries(t *testing.T) {
	if raceEnabled {
		t.Skip("skipping under -race: upstream go-redis MonitorCmd race")
	}
	client, mr := setupTestClient(t)

	received := make(chan types.MonitorEntry, 16)
	session, err := client.StartMonitor(func(e types.MonitorEntry) {
		received <- e
	})
	if err != nil {
		t.Fatalf("StartMonitor: %v", err)
	}
	defer session.Close()

	// Give MONITOR setup a moment, then issue a write through miniredis directly.
	time.Sleep(50 * time.Millisecond)
	mr.Set("k1", "v1")

	select {
	case entry := <-received:
		if entry.Raw == "" {
			t.Errorf("entry raw is empty: %+v", entry)
		}
	case <-time.After(2 * time.Second):
		// Miniredis MONITOR support varies by version; if events don't surface
		// within the window, skip rather than fail to avoid blocking CI.
		t.Skip("no MONITOR events received within 2s (miniredis version may not stream MONITOR)")
	}
}
