package ui

import (
	"strings"
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero bytes", 0, "0 B"},
		{"small bytes", 500, "500 B"},
		{"exactly 1 KB", 1024, "1.0 KB"},
		{"1.5 KB", 1536, "1.5 KB"},
		{"exactly 1 MB", 1024 * 1024, "1.0 MB"},
		{"exactly 1 GB", 1024 * 1024 * 1024, "1.0 GB"},
		{"1023 bytes", 1023, "1023 B"},
		{"large MB", 5 * 1024 * 1024, "5.0 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.expected)
			}
		})
	}
}

func TestSanitizeBinaryString(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedBinary bool
		checkContains  string // substring the result should contain
	}{
		{"plain ASCII", "hello world", false, "hello world"},
		{"empty string", "", false, ""},
		{"HyperLogLog prefix", "HYLL\x00\x01\x02data", true, "HyperLogLog"},
		{"tabs and newlines preserved", "line1\nline2\ttab", false, "line1\nline2\ttab"},
		{"high non-printable ratio", string([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x10, 0x11}), true, "binary data"},
		{"low non-printable below threshold", "abcdefghijklmnopqrst\x01", false, "\\x01"},
		{"low non-printable above threshold", "abc\x01def", true, "binary data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, isBinary := sanitizeBinaryString(tt.input)
			if isBinary != tt.expectedBinary {
				t.Errorf("sanitizeBinaryString(%q) isBinary = %v, want %v", tt.input, isBinary, tt.expectedBinary)
			}
			if tt.checkContains != "" && !strings.Contains(result, tt.checkContains) {
				t.Errorf("sanitizeBinaryString(%q) = %q, want to contain %q", tt.input, result, tt.checkContains)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"shorter than max", "hello", 10, "hello"},
		{"equal to max", "hello", 5, "hello"},
		{"longer than max", "hello world", 8, "hello..."},
		{"exactly at boundary", "abcdef", 6, "abcdef"},
		{"min truncation", "abcdefgh", 4, "a..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestParseLogEntry(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedLevel string
		expectedMsg   string
		hasTime       bool
	}{
		{
			"valid JSON with all fields",
			`{"time":"2024-01-15T10:30:00.123456789Z","level":"INFO","msg":"server started"}`,
			"INFO",
			"server started",
			true,
		},
		{
			"missing time field",
			`{"level":"ERROR","msg":"connection failed"}`,
			"ERROR",
			"connection failed",
			false,
		},
		{
			"missing level field",
			`{"time":"2024-01-15T10:30:00Z","msg":"hello"}`,
			"",
			"hello",
			true,
		},
		{
			"non-JSON fallback",
			"plain text log line",
			"",
			"plain text log line",
			false,
		},
		{
			"empty JSON object",
			`{}`,
			"",
			"",
			false,
		},
		{
			"level lowercased in source",
			`{"level":"warn","msg":"low disk"}`,
			"WARN",
			"low disk",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := parseLogEntry(tt.input)
			if entry.Level != tt.expectedLevel {
				t.Errorf("Level = %q, want %q", entry.Level, tt.expectedLevel)
			}
			if entry.Msg != tt.expectedMsg {
				t.Errorf("Msg = %q, want %q", entry.Msg, tt.expectedMsg)
			}
			if tt.hasTime && entry.Time == "" {
				t.Error("expected Time to be set")
			}
			if !tt.hasTime && entry.Time != "" {
				t.Errorf("expected empty Time, got %q", entry.Time)
			}
		})
	}
}

func TestParseLogEntry_RFC3339Nano(t *testing.T) {
	input := `{"time":"2024-01-15T10:30:45.123456789Z","level":"INFO","msg":"test"}`
	entry := parseLogEntry(input)
	expected := time.Date(2024, 1, 15, 10, 30, 45, 123456789, time.UTC).Format("15:04:05")
	if entry.Time != expected {
		t.Errorf("Time = %q, want %q", entry.Time, expected)
	}
}

func TestFindStringEnd(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		start    int
		expected int
	}{
		{"simple string", `"hello"`, 1, 6},
		{"escaped quote", `"he\"llo"`, 1, 8},
		{"escaped backslash", `"he\\llo"`, 1, 8},
		{"no closing quote", `"hello`, 1, -1},
		{"empty string", `""`, 1, 1},
		{"escaped backslash before quote", `"he\\\\"`, 1, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findStringEnd(tt.input, tt.start)
			if got != tt.expected {
				t.Errorf("findStringEnd(%q, %d) = %d, want %d", tt.input, tt.start, got, tt.expected)
			}
		})
	}
}

func TestIsAfterArrayStart(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		pos      int
		expected bool
	}{
		{"after open bracket", `["hello"]`, 1, true},
		{"after comma in array", `["a", "b"]`, 5, true},
		{"after open brace", `{"key": "val"}`, 8, false},
		{"after colon", `{"key": "val"}`, 8, false},
		{"beginning of string", `"hello"`, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAfterArrayStart(tt.input, tt.pos)
			if got != tt.expected {
				t.Errorf("isAfterArrayStart(%q, %d) = %v, want %v", tt.input, tt.pos, got, tt.expected)
			}
		})
	}
}

func TestIsInArrayContext(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		pos      int
		expected bool
	}{
		{"inside array", `["a", "b"]`, 5, true},
		{"inside object", `{"a": "b"}`, 5, false},
		{"nested array in object", `{"k": ["a", "b"]}`, 10, true},
		{"nested object in array", `[{"a": "b"}]`, 5, false},
		{"at array start", `["a"]`, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInArrayContext(tt.input, tt.pos)
			if got != tt.expected {
				t.Errorf("isInArrayContext(%q, %d) = %v, want %v", tt.input, tt.pos, got, tt.expected)
			}
		})
	}
}

func TestFormatPossibleJSON(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantEmpty  bool
		wantSame   bool // output should match input (passthrough)
		wantBinary bool // expect binary data message
	}{
		{"empty string", "", true, false, false},
		{"whitespace only", "   ", true, false, false},
		{"plain text passthrough", "hello world", false, true, false},
		{"non-JSON passthrough", "not json at all", false, true, false},
		{"valid JSON object", `{"key":"value"}`, false, false, false},
		{"valid JSON array", `[1,2,3]`, false, false, false},
		{"binary data", "HYLL\x00\x01\x02data", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPossibleJSON(tt.input)
			if tt.wantEmpty && got != "" {
				t.Errorf("expected empty string, got %q", got)
			}
			if tt.wantSame && got != tt.input {
				t.Errorf("expected passthrough %q, got %q", tt.input, got)
			}
			if tt.wantBinary && !strings.Contains(got, "HyperLogLog") {
				t.Errorf("expected binary data message, got %q", got)
			}
			if !tt.wantEmpty && got == "" {
				t.Error("expected non-empty output")
			}
		})
	}
}

func TestFormatPossibleJSON_NoPanic(t *testing.T) {
	// Ensure no panic on various inputs
	inputs := []string{
		"",
		"plain",
		`{"valid": true}`,
		`[1, 2, 3]`,
		`{"nested": {"deep": [1, 2]}}`,
		`invalid json {`,
		`{`,
		`[`,
		string([]byte{0x00, 0x01, 0x02}),
	}

	for _, input := range inputs {
		t.Run("no panic", func(t *testing.T) {
			// Should not panic
			_ = formatPossibleJSON(input)
		})
	}
}
