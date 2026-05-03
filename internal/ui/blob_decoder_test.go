package ui

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bearded-giant/redis-tui/internal/decoder"
	"github.com/bearded-giant/redis-tui/internal/types"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vmihailenco/msgpack/v5"
)

func TestRenderDecodedString_AutoJSON(t *testing.T) {
	m, _, _ := newTestModel(t)
	body, badge := m.renderDecodedString(`{"a":1}`)
	if !strings.Contains(body, `"a": 1`) {
		t.Errorf("body = %q", body)
	}
	if badge != "json" {
		t.Errorf("badge = %q, want json", badge)
	}
}

func TestRenderDecodedString_AutoJsonPlus(t *testing.T) {
	m, _, _ := newTestModel(t)
	inner, _ := msgpack.Marshal(map[string]any{"k": "v"})
	envelope := map[string]any{
		"v":          1,
		"checkpoint": map[string]string{"type": "msgpack", "data": base64.StdEncoding.EncodeToString(inner)},
	}
	envBytes, _ := json.Marshal(envelope)
	body, badge := m.renderDecodedString(string(envBytes))
	if badge != "jsonplus" {
		t.Errorf("badge = %q, want jsonplus", badge)
	}
	if !strings.Contains(body, `"k": "v"`) {
		t.Errorf("body should contain decoded inner, got %q", body)
	}
}

func TestRenderDecodedString_PlainTextRaw(t *testing.T) {
	m, _, _ := newTestModel(t)
	body, badge := m.renderDecodedString("just plain text")
	if badge != "raw" {
		t.Errorf("badge = %q, want raw", badge)
	}
	if !strings.Contains(body, "just plain text") {
		t.Errorf("body = %q", body)
	}
}

func TestRenderDecodedString_OverrideForcesFormat(t *testing.T) {
	m, _, _ := newTestModel(t)
	encoded := base64.StdEncoding.EncodeToString([]byte("hello"))
	m.ValueDecodeOverride = decoder.FormatBase64
	body, badge := m.renderDecodedString(encoded)
	if badge != "base64" {
		t.Errorf("badge = %q, want base64", badge)
	}
	if !strings.Contains(body, "hello") {
		t.Errorf("body should contain decoded value, got %q", body)
	}
}

func TestRenderDecodedString_OverrideDecodeError(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.ValueDecodeOverride = decoder.FormatJSON
	body, badge := m.renderDecodedString("not json")
	if !strings.HasPrefix(badge, "raw (decode failed") {
		t.Errorf("badge = %q, want raw fallback with error", badge)
	}
	if body == "" {
		t.Error("body should still render fallback content")
	}
}

func TestRenderDecodedString_Empty(t *testing.T) {
	m, _, _ := newTestModel(t)
	body, badge := m.renderDecodedString("")
	if body != "" || badge != "" {
		t.Errorf("empty input → (%q, %q), want both empty", body, badge)
	}
}

func TestKeyDetail_BCyclesDecodeOverride(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Screen = types.ScreenKeyDetail
	m.CurrentKey = &types.RedisKey{Key: "x", Type: types.KeyTypeString}

	if m.ValueDecodeOverride != "" {
		t.Fatal("override should default to empty")
	}

	// Cycle from empty (auto) goes through: raw → base64 → json → ...
	wants := []decoder.Format{
		decoder.FormatRaw,
		decoder.FormatBase64,
		decoder.FormatJSON,
		decoder.FormatJsonPlus,
		decoder.FormatMsgpack,
		decoder.FormatRaw,
	}
	for i, want := range wants {
		updated, _ := m.handleKeyDetailScreen(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
		m = updated.(Model)
		if m.ValueDecodeOverride != want {
			t.Errorf("after press #%d: override = %v, want %v", i+1, m.ValueDecodeOverride, want)
		}
	}
}

func TestKeysList_EnterResetsDecodeOverride(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Screen = types.ScreenKeys
	m.Keys = []types.RedisKey{{Key: "x", Type: types.KeyTypeString}}
	m.SelectedKeyIdx = 0
	m.ValueDecodeOverride = decoder.FormatJSON

	updated, _ := m.handleKeysScreen(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)
	if got.ValueDecodeOverride != "" {
		t.Errorf("override should reset on key enter, got %v", got.ValueDecodeOverride)
	}
}
