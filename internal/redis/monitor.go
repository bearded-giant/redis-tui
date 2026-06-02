package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bearded-giant/redis-tui/internal/types"
)

// MonitorSession owns a dedicated redis connection that has been switched into
// MONITOR mode. Close() detaches from the redis connection (releases it back
// to the pool) and stops the underlying goroutine.
type MonitorSession struct {
	cancel context.CancelFunc
	ch     chan string
	done   chan struct{}
}

// Close signals the MONITOR goroutine to stop and waits for it to exit.
func (s *MonitorSession) Close() {
	if s == nil {
		return
	}
	s.cancel()
	<-s.done
}

// StartMonitor opens a MONITOR subscription on a dedicated connection and
// invokes onEvent for each parsed entry. Returns a session whose Close stops
// the stream. Errors at start (auth/ACL/conn refused) are returned synchronously;
// transport-level errors mid-stream surface via onEvent with Err set.
//
// MONITOR is per-node — for cluster mode this targets the active client only.
// Document upstream.
func (c *Client) StartMonitor(onEvent func(types.MonitorEntry)) (types.MonitorSessionHandle, error) {
	c.mu.RLock()
	client := c.client
	isCluster := c.isCluster
	c.mu.RUnlock()

	if isCluster {
		return nil, fmt.Errorf("MONITOR not supported in cluster mode (per-node only)")
	}
	if client == nil {
		return nil, fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithCancel(c.ctx)
	ch := make(chan string, 256)
	done := make(chan struct{})

	mon := client.Monitor(ctx, ch)
	mon.Start()

	go func() {
		defer close(done)
		defer mon.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case line, ok := <-ch:
				if !ok {
					return
				}
				entry := parseMonitorLine(line)
				onEvent(entry)
			}
		}
	}()

	return &MonitorSession{
		cancel: cancel,
		ch:     ch,
		done:   done,
	}, nil
}

// parseMonitorLine extracts the structured fields from a MONITOR line.
//
// Format: `<timestamp> [<db> <client_addr>] "<cmd>" "<arg1>" "<arg2>" ...`
// Example: `1574099031.764036 [0 127.0.0.1:60270] "SET" "user:1" "alice"`
//
// Parser is intentionally forgiving — Redis adds context fields (lua, replica)
// between versions; falls back to putting the unparseable tail into Raw so the
// UI can still render something useful.
func parseMonitorLine(line string) types.MonitorEntry {
	entry := types.MonitorEntry{Raw: line}

	// Timestamp: everything up to first space, parsed as float seconds.
	firstSpace := strings.IndexByte(line, ' ')
	if firstSpace < 0 {
		return entry
	}
	tsStr := line[:firstSpace]
	if tsFloat, err := strconv.ParseFloat(tsStr, 64); err == nil {
		secs := int64(tsFloat)
		nanos := int64((tsFloat - float64(secs)) * 1e9)
		entry.Time = time.Unix(secs, nanos)
	}
	rest := strings.TrimSpace(line[firstSpace:])

	// Context block: [<db> <client_addr>]
	if strings.HasPrefix(rest, "[") {
		end := strings.IndexByte(rest, ']')
		if end > 0 {
			ctxFields := strings.Fields(rest[1:end])
			if len(ctxFields) >= 1 {
				if db, err := strconv.Atoi(ctxFields[0]); err == nil {
					entry.DB = db
				}
			}
			if len(ctxFields) >= 2 {
				entry.Client = ctxFields[1]
			}
			rest = strings.TrimSpace(rest[end+1:])
		}
	}

	// Remaining: quoted args. First is cmd, rest are args.
	parts := splitQuoted(rest)
	if len(parts) > 0 {
		entry.Cmd = parts[0]
		if len(parts) > 1 {
			entry.Args = parts[1:]
		}
	}

	return entry
}

// splitQuoted pulls successive `"..."` tokens out of s.
// Handles backslash escapes inside the quotes.
func splitQuoted(s string) []string {
	var out []string
	var buf strings.Builder
	in := false
	esc := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if esc {
			buf.WriteByte(ch)
			esc = false
			continue
		}
		if ch == '\\' && in {
			esc = true
			continue
		}
		if ch == '"' {
			if in {
				out = append(out, buf.String())
				buf.Reset()
				in = false
			} else {
				in = true
			}
			continue
		}
		if in {
			buf.WriteByte(ch)
		}
	}
	return out
}
