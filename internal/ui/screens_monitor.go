package ui

import (
	"fmt"
	"strings"

	"github.com/bearded-giant/redis-tui/internal/types"
	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
)

// handleMonitorScreen drives the MONITOR live-stream view.
//
// Keys:
//   space         pause/resume buffering
//   c             clear buffer
//   /             focus filter input
//   esc           close session + back to keys screen
func (m Model) handleMonitorScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.MonitorFilter.Focused() {
		switch msg.String() {
		case "enter", "esc":
			m.MonitorFilter.Blur()
		default:
			var inputCmd tea.Cmd
			m.MonitorFilter, inputCmd = m.MonitorFilter.Update(msg)
			return m, inputCmd
		}
		return m, nil
	}

	switch msg.String() {
	case "esc":
		if m.MonitorSession != nil {
			m.MonitorSession.Close()
			m.MonitorSession = nil
		}
		m.Screen = types.ScreenKeys
		m.StatusMsg = "MONITOR stopped"
	case " ", "space":
		m.MonitorPaused = !m.MonitorPaused
		if m.MonitorPaused {
			m.StatusMsg = "MONITOR paused"
		} else {
			m.StatusMsg = "MONITOR resumed"
		}
	case "c":
		m.MonitorEntries = nil
		m.StatusMsg = "MONITOR buffer cleared"
	case "/":
		m.MonitorFilter.Focus()
	}
	return m, nil
}

func (m Model) viewMonitor() string {
	var b strings.Builder
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)

	b.WriteString(titleStyle.Render("MONITOR Live Stream"))
	b.WriteString("\n")
	b.WriteString(warningStyle.Render("⚠ MONITOR may impact server performance"))
	b.WriteString("\n\n")

	if m.MonitorErr != nil {
		b.WriteString(errorStyle.Render("Error: " + m.MonitorErr.Error()))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("esc:back"))
		return b.String()
	}

	b.WriteString(keyStyle.Render("Filter: "))
	if m.MonitorFilter.Focused() {
		b.WriteString(m.MonitorFilter.View())
	} else {
		pattern := m.MonitorFilter.Value()
		if pattern == "" {
			pattern = "(none)"
		}
		b.WriteString(dimStyle.Render(pattern))
	}
	b.WriteString("\n\n")

	visible := filterMonitorEntries(m.MonitorEntries, m.MonitorFilter.Value())

	status := fmt.Sprintf("Entries: %d (buffer %d/%d)", len(visible), len(m.MonitorEntries), m.MonitorBufferCap)
	if m.MonitorPaused {
		status += " — PAUSED"
	}
	b.WriteString(dimStyle.Render(status))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", 80)))
	b.WriteString("\n")

	maxLines := m.Height - 12
	if maxLines < 5 {
		maxLines = 5
	}
	start := 0
	if len(visible) > maxLines {
		start = len(visible) - maxLines
	}
	for i := start; i < len(visible); i++ {
		b.WriteString(renderMonitorEntry(visible[i], m.Width-4))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("/:filter  space:pause/resume  c:clear  esc:stop+back"))
	return b.String()
}

func filterMonitorEntries(entries []types.MonitorEntry, filter string) []types.MonitorEntry {
	if filter == "" {
		return entries
	}
	filter = strings.ToLower(filter)
	out := make([]types.MonitorEntry, 0, len(entries))
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Cmd), filter) {
			out = append(out, e)
			continue
		}
		matched := false
		for _, a := range e.Args {
			if strings.Contains(strings.ToLower(a), filter) {
				matched = true
				break
			}
		}
		if matched {
			out = append(out, e)
		}
	}
	return out
}

func renderMonitorEntry(e types.MonitorEntry, maxWidth int) string {
	ts := e.Time.Format("15:04:05.000")
	cmd := e.Cmd
	if cmd == "" {
		cmd = "(?)"
	}
	args := strings.Join(e.Args, " ")
	line := fmt.Sprintf("%s [%d %s] %s %s", ts, e.DB, e.Client, cmd, args)
	if maxWidth > 0 && len(line) > maxWidth {
		line = line[:maxWidth-3] + "..."
	}
	return normalStyle.Render(line)
}
