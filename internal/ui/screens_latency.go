package ui

import (
	"fmt"
	"strings"

	"github.com/bearded-giant/redis-tui/internal/types"
	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
)

// handleLatencyScreen drives the latency dashboard.
//
// Keys:
//   j/k          move cursor between events
//   enter        load LATENCY HISTORY for the selected event (histogram)
//   d            toggle DOCTOR narrative
//   r            refresh snapshot (LATENCY LATEST + DOCTOR + threshold)
//   R            open reset-confirm sub-screen
//   t            CONFIG SET latency-monitor-threshold 100 (one-key enable when 0)
//   esc          back to keys screen
func (m Model) handleLatencyScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Screen = types.ScreenKeys
		m.LatencyHistory = nil
		m.LatencyHistoryEvent = ""
		m.LatencyShowDoctor = false
	case "j", "down":
		if m.LatencySelectedIdx < len(m.LatencyEvents)-1 {
			m.LatencySelectedIdx++
		}
	case "k", "up":
		if m.LatencySelectedIdx > 0 {
			m.LatencySelectedIdx--
		}
	case "enter":
		if m.LatencySelectedIdx < len(m.LatencyEvents) {
			event := m.LatencyEvents[m.LatencySelectedIdx].Event
			m.Loading = true
			return m, m.Cmds.LoadLatencyHistory(event)
		}
	case "d":
		m.LatencyShowDoctor = !m.LatencyShowDoctor
	case "r":
		m.Loading = true
		return m, m.Cmds.LoadLatencySnapshot()
	case "R":
		m.Screen = types.ScreenLatencyConfirmReset
	case "t":
		if m.LatencyThreshold == 0 {
			return m, m.Cmds.SetRedisConfig("latency-monitor-threshold", "100")
		}
	}
	return m, nil
}

// handleLatencyResetConfirmScreen — destructive op, gated behind explicit y/n.
func (m Model) handleLatencyResetConfirmScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.Screen = types.ScreenLatency
		m.Loading = true
		return m, m.Cmds.ResetLatency()
	case "n", "N", "esc":
		m.Screen = types.ScreenLatency
	}
	return m, nil
}

func (m Model) viewLatency() string {
	var b strings.Builder
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)

	b.WriteString(titleStyle.Render("Latency Doctor"))
	b.WriteString("\n\n")

	if m.LatencyErr != nil {
		b.WriteString(errorStyle.Render("Error: " + m.LatencyErr.Error()))
		b.WriteString("\n\n")
	}

	if m.LatencyThreshold == 0 {
		b.WriteString(warnStyle.Render(
			"Latency monitoring is OFF — server's latency-monitor-threshold = 0. Press 't' to set it to 100ms."))
		b.WriteString("\n\n")
	} else {
		b.WriteString(dimStyle.Render(fmt.Sprintf("Threshold: %dms (events >= this are tracked)", m.LatencyThreshold)))
		b.WriteString("\n\n")
	}

	if len(m.LatencyEvents) == 0 {
		b.WriteString(dimStyle.Render("No latency events recorded."))
	} else {
		b.WriteString(headerStyle.Render(fmt.Sprintf("%-30s %-10s %-10s", "Event", "Latest", "Max")))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(strings.Repeat("─", 60)))
		b.WriteString("\n")
		for i, e := range m.LatencyEvents {
			line := fmt.Sprintf("%-30s %-10s %-10s",
				e.Event, msStyle(e.LatestMs), msStyle(e.MaxMs))
			if i == m.LatencySelectedIdx {
				b.WriteString(selectedStyle.Render("▶ " + line))
			} else {
				b.WriteString(normalStyle.Render("  " + line))
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	if m.LatencyHistoryEvent != "" && len(m.LatencyHistory) > 0 {
		b.WriteString(keyStyle.Render(fmt.Sprintf("History: %s (%d samples)", m.LatencyHistoryEvent, len(m.LatencyHistory))))
		b.WriteString("\n")
		b.WriteString(renderHistogram(m.LatencyHistory, m.Width-8))
		b.WriteString("\n")
	}

	if m.LatencyShowDoctor && m.LatencyDoctor != "" {
		b.WriteString(keyStyle.Render("Doctor:"))
		b.WriteString("\n")
		b.WriteString(normalStyle.Render(m.LatencyDoctor))
		b.WriteString("\n\n")
	}

	b.WriteString(helpStyle.Render("j/k:nav  enter:history  d:doctor  r:refresh  R:reset  esc:back"))
	return b.String()
}

func (m Model) viewLatencyResetConfirm() string {
	var b strings.Builder
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)

	b.WriteString(warnStyle.Render("Reset all latency events?"))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render("This runs LATENCY RESET on the server, clearing recorded latency events."))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("y:reset  n/esc:cancel"))
	return m.renderModal(b.String())
}

// msStyle colors a millisecond value green/yellow/red by typical thresholds.
func msStyle(ms int64) string {
	style := normalStyle
	if ms >= 1000 {
		style = errorStyle
	} else if ms >= 100 {
		style = ttlWarningStyle
	} else {
		style = ttlGreenStyle
	}
	return style.Render(fmt.Sprintf("%dms", ms))
}

// renderHistogram draws a horizontal bar chart of latency samples.
// One row per sample, bar width proportional to value vs max in the set.
func renderHistogram(samples []types.LatencySample, maxWidth int) string {
	if len(samples) == 0 || maxWidth < 20 {
		return dimStyle.Render("(no samples or too narrow)")
	}

	var peak int64
	for _, s := range samples {
		if s.LatencyMs > peak {
			peak = s.LatencyMs
		}
	}
	if peak == 0 {
		peak = 1
	}

	// Cap rows so the chart doesn't push everything else off-screen.
	const maxRows = 15
	start := 0
	if len(samples) > maxRows {
		start = len(samples) - maxRows
	}

	var b strings.Builder
	barFieldWidth := maxWidth - 12 // leaves room for ms label
	if barFieldWidth < 5 {
		barFieldWidth = 5
	}
	for i := start; i < len(samples); i++ {
		s := samples[i]
		barLen := int(int64(barFieldWidth) * s.LatencyMs / peak)
		if barLen < 1 && s.LatencyMs > 0 {
			barLen = 1
		}
		var style lipgloss.Style
		if s.LatencyMs >= peak/2 {
			style = errorStyle
		} else if s.LatencyMs >= peak/4 {
			style = ttlWarningStyle
		} else {
			style = ttlGreenStyle
		}
		bar := strings.Repeat("█", barLen)
		b.WriteString(style.Render(bar))
		b.WriteString(strings.Repeat(" ", barFieldWidth-barLen))
		b.WriteString(" ")
		b.WriteString(normalStyle.Render(fmt.Sprintf("%dms", s.LatencyMs)))
		b.WriteString("\n")
	}
	return b.String()
}
