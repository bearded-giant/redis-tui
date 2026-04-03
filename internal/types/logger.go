package types

import (
	"io"
	"strings"
	"sync"
)

// LogWriter implements io.Writer for capturing logs using a ring buffer.
type LogWriter struct {
	mu    sync.Mutex
	buf   [MaxLogs]string
	head  int // index of oldest entry
	count int // number of entries stored
}

// MaxLogs is the maximum number of log entries to keep
const MaxLogs = 100

// NewLogWriter creates a new LogWriter
func NewLogWriter() *LogWriter {
	return &LogWriter{}
}

// Write implements io.Writer
func (w *LogWriter) Write(p []byte) (n int, err error) {
	logStr := string(p)
	if strings.Contains(logStr, `"level":"DEBUG"`) {
		return len(p), nil
	}
	w.mu.Lock()
	idx := (w.head + w.count) % MaxLogs
	if w.count == MaxLogs {
		// Buffer full — overwrite oldest entry and advance head
		w.buf[w.head] = logStr
		w.head = (w.head + 1) % MaxLogs
	} else {
		w.buf[idx] = logStr
		w.count++
	}
	w.mu.Unlock()
	return len(p), nil
}

// GetLogs returns a snapshot copy of all log entries in chronological order
func (w *LogWriter) GetLogs() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	result := make([]string, w.count)
	for i := range w.count {
		result[i] = w.buf[(w.head+i)%MaxLogs]
	}
	return result
}

// Len returns the number of log entries
func (w *LogWriter) Len() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.count
}

var _ io.Writer = &LogWriter{}
