package ui

import (
	"strings"
	"testing"
)

func TestPreviewDecodeString_RawText(t *testing.T) {
	rendered, badge := previewDecodeString("just plain text")
	if badge != "" {
		t.Errorf("badge = %q, want empty for raw text", badge)
	}
	if !strings.Contains(rendered, "just plain text") {
		t.Errorf("rendered = %q, want it to contain original text", rendered)
	}
}

func TestPreviewDecodeString_Base64(t *testing.T) {
	// "hello world" base64-encoded
	rendered, badge := previewDecodeString("aGVsbG8gd29ybGQ=")
	if badge != "base64" {
		t.Errorf("badge = %q, want base64", badge)
	}
	if !strings.Contains(rendered, "hello world") {
		t.Errorf("rendered = %q, want decoded \"hello world\"", rendered)
	}
}

func TestPreviewDecodeString_JSON(t *testing.T) {
	rendered, badge := previewDecodeString(`{"name":"alice","role":"admin"}`)
	if badge != "json" {
		t.Errorf("badge = %q, want json", badge)
	}
	if !strings.Contains(rendered, "alice") || !strings.Contains(rendered, "admin") {
		t.Errorf("rendered missing decoded content: %q", rendered)
	}
}

func TestPreviewDecodeString_BytecapsHugeBlob(t *testing.T) {
	// Build a 10KB raw string — must be truncated to ~previewByteCap.
	huge := strings.Repeat("x", 10*1024)
	rendered, _ := previewDecodeString(huge)
	if len(rendered) > previewByteCap+100 {
		t.Errorf("rendered length = %d, want <= %d (cap + small overhead for truncation marker)", len(rendered), previewByteCap+100)
	}
}

func TestCapPreview(t *testing.T) {
	t.Run("under cap unchanged", func(t *testing.T) {
		got := capPreview("short")
		if got != "short" {
			t.Errorf("capPreview(short) = %q, want unchanged", got)
		}
	})
	t.Run("over cap truncated", func(t *testing.T) {
		input := strings.Repeat("a", previewByteCap+500)
		got := capPreview(input)
		if !strings.HasSuffix(got, "bytes)") {
			t.Errorf("capPreview should end with truncation marker, got tail: %q", got[len(got)-30:])
		}
		if len(got) > previewByteCap+100 {
			t.Errorf("length = %d, want <= %d", len(got), previewByteCap+100)
		}
	})
}
