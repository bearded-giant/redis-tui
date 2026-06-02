package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewBulkDelete() string {
	var b strings.Builder

	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)

	b.WriteString(warningStyle.Render("Bulk Delete Keys"))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("Delete all keys matching a pattern"))
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Pattern:"))
	b.WriteString("\n")
	b.WriteString(m.Inputs.BulkDeleteInput.View())
	b.WriteString("\n\n")

	if len(m.BulkDeletePreview) > 0 {
		b.WriteString(keyStyle.Render(fmt.Sprintf("Will delete %d keys:", len(m.BulkDeletePreview))))
		b.WriteString("\n")
		for i, k := range m.BulkDeletePreview {
			if i >= 5 {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  ... and %d more", len(m.BulkDeletePreview)-5)))
				break
			}
			b.WriteString(normalStyle.Render("  • " + k))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("enter:delete  esc:cancel"))

	return m.renderModal(b.String())
}

func (m Model) viewBatchTTL() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Batch Set TTL"))
	b.WriteString("\n\n")

	previewLoaded := m.BatchTTLPendingPattern != ""

	if previewLoaded {
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
		b.WriteString(warningStyle.Render(fmt.Sprintf(
			"Will set TTL=%ds on %d keys matching %q",
			int(m.BatchTTLPendingTTL.Seconds()),
			m.BatchTTLMatched,
			m.BatchTTLPendingPattern,
		)))
		b.WriteString("\n\n")

		if m.BatchTTLPendingTTL <= 0 {
			b.WriteString(dimStyle.Render("(TTL <= 0 → PERSIST: removes any existing TTL)"))
			b.WriteString("\n\n")
		}

		if len(m.BatchTTLPreview) > 0 {
			b.WriteString(keyStyle.Render("Sample keys:"))
			b.WriteString("\n")
			for _, k := range m.BatchTTLPreview {
				b.WriteString(normalStyle.Render("  • " + k))
				b.WriteString("\n")
			}
			if m.BatchTTLMatched > len(m.BatchTTLPreview) {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  ... and %d more", m.BatchTTLMatched-len(m.BatchTTLPreview))))
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}

		if m.BatchTTLMatched > 0 {
			b.WriteString(helpStyle.Render("a:apply  esc:cancel"))
		} else {
			b.WriteString(helpStyle.Render("esc:cancel (no matching keys)"))
		}
		return m.renderModal(b.String())
	}

	b.WriteString(dimStyle.Render("Set TTL on all keys matching a pattern. Enter previews matched count first."))
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("TTL (seconds, <=0 to remove TTL):"))
	b.WriteString("\n")
	b.WriteString(m.Inputs.BatchTTLInput.View())
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Pattern:"))
	b.WriteString("\n")
	b.WriteString(m.Inputs.BatchTTLPattern.View())
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("tab:next  enter:preview  esc:cancel"))

	return m.renderModal(b.String())
}
