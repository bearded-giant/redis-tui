package ui

import (
	"strings"
	"testing"

	"github.com/bearded-giant/redis-tui/internal/types"
)

func TestRenderHistogram_Empty(t *testing.T) {
	got := renderHistogram(nil, 80)
	if !strings.Contains(got, "no samples") {
		t.Errorf("expected empty marker, got %q", got)
	}
}

func TestRenderHistogram_TooNarrow(t *testing.T) {
	samples := []types.LatencySample{{LatencyMs: 100}}
	got := renderHistogram(samples, 10)
	if !strings.Contains(got, "too narrow") {
		t.Errorf("expected narrow marker, got %q", got)
	}
}

func TestRenderHistogram_RendersBars(t *testing.T) {
	samples := []types.LatencySample{
		{Time: 1, LatencyMs: 10},
		{Time: 2, LatencyMs: 50},
		{Time: 3, LatencyMs: 100},
	}
	got := renderHistogram(samples, 60)
	if !strings.Contains(got, "10ms") {
		t.Errorf("expected 10ms label, got %q", got)
	}
	if !strings.Contains(got, "100ms") {
		t.Errorf("expected 100ms label, got %q", got)
	}
	if !strings.Contains(got, "█") {
		t.Errorf("expected bar glyph, got %q", got)
	}
}

func TestRenderHistogram_CapsRows(t *testing.T) {
	// 30 samples — only last 15 should render.
	samples := make([]types.LatencySample, 30)
	for i := range samples {
		samples[i] = types.LatencySample{Time: int64(i), LatencyMs: int64(i + 1)}
	}
	got := renderHistogram(samples, 60)
	lines := strings.Count(got, "\n")
	if lines > 16 {
		t.Errorf("expected <= 16 lines (15 rows + trailing), got %d", lines)
	}
}

func TestMsStyle_Bucketing(t *testing.T) {
	// Just exercise the function for each bucket — actual ANSI codes are styled
	// per terminal so we only check the ms suffix renders.
	cases := []int64{1, 50, 100, 500, 1000, 5000}
	for _, ms := range cases {
		out := msStyle(ms)
		if !strings.Contains(out, "ms") {
			t.Errorf("msStyle(%d) = %q, missing ms suffix", ms, out)
		}
	}
}
