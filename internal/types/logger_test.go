package types

import "testing"

func TestLogWriter_Write(t *testing.T) {
	t.Run("appends log entry", func(t *testing.T) {
		logs := []string{}
		w := LogWriter{Logs: &logs}

		input := []byte(`{"level":"INFO","msg":"test"}`)
		n, err := w.Write(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != len(input) {
			t.Errorf("Write returned %d bytes, want %d", n, len(input))
		}
		if len(logs) != 1 {
			t.Fatalf("expected 1 log entry, got %d", len(logs))
		}
		if logs[0] != `{"level":"INFO","msg":"test"}` {
			t.Errorf("log entry = %q, want %q", logs[0], `{"level":"INFO","msg":"test"}`)
		}
	})

	t.Run("filters DEBUG messages", func(t *testing.T) {
		logs := []string{}
		w := LogWriter{Logs: &logs}

		input := []byte(`{"level":"DEBUG","msg":"debug info"}`)
		n, err := w.Write(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != len(input) {
			t.Errorf("Write returned %d bytes, want %d", n, len(input))
		}
		if len(logs) != 0 {
			t.Errorf("expected 0 log entries for DEBUG, got %d", len(logs))
		}
	})

	t.Run("trims at MaxLogs", func(t *testing.T) {
		logs := []string{}
		w := LogWriter{Logs: &logs}

		// Write MaxLogs + 5 entries
		for i := 0; i < MaxLogs+5; i++ {
			_, err := w.Write([]byte(`{"level":"INFO","msg":"entry"}`))
			if err != nil {
				t.Fatalf("unexpected error at entry %d: %v", i, err)
			}
		}
		if len(logs) != MaxLogs {
			t.Errorf("expected %d log entries, got %d", MaxLogs, len(logs))
		}
	})

	t.Run("returns correct byte count", func(t *testing.T) {
		logs := []string{}
		w := LogWriter{Logs: &logs}
		input := []byte("some log message")

		n, err := w.Write(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != len(input) {
			t.Errorf("Write returned %d, want %d", n, len(input))
		}
	})

	t.Run("returns correct byte count for filtered DEBUG", func(t *testing.T) {
		logs := []string{}
		w := LogWriter{Logs: &logs}
		input := []byte(`{"level":"DEBUG","msg":"filtered"}`)

		n, err := w.Write(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != len(input) {
			t.Errorf("Write returned %d, want %d", n, len(input))
		}
	})
}
