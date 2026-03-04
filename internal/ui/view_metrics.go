package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	metricsHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	metricsSectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245"))
	metricsCardBorder  = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1).
				Width(22)
	metricsCardLabel  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	metricsCardValue  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	metricsGreenValue = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("35"))
	metricsYellowValue = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33"))
	metricsChartBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("237")).
				Padding(0, 1)
	metricsClusterBadge = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("208")).
				Padding(0, 1)
)

// viewLiveMetrics renders the live metrics dashboard with ASCII charts
func (m Model) viewLiveMetrics() string {
	var b strings.Builder

	separatorWidth := m.Width - 10
	if separatorWidth < 20 {
		separatorWidth = 20
	}
	if separatorWidth > 80 {
		separatorWidth = 80
	}

	// Header box
	b.WriteString(metricsHeaderStyle.Render("Live Metrics Dashboard"))
	b.WriteString("\n")

	// Connection info line
	connInfo := ""
	if m.CurrentConn != nil {
		connInfo = fmt.Sprintf("%s (%s:%d)", m.CurrentConn.Name, m.CurrentConn.Host, m.CurrentConn.Port)
	}

	if m.LiveMetrics != nil && len(m.LiveMetrics.DataPoints) > 0 {
		connInfo += fmt.Sprintf("  data points: %d/%d", len(m.LiveMetrics.DataPoints), m.LiveMetrics.MaxDataPoints)
	}

	b.WriteString(dimStyle.Render(connInfo))

	// Cluster badge — based on connection config, not server detection
	if m.CurrentConn != nil && m.CurrentConn.UseCluster {
		clusterBadge := metricsClusterBadge.Render(fmt.Sprintf("CLUSTER  %d nodes", len(m.ClusterNodes)))
		b.WriteString("  ")
		b.WriteString(clusterBadge)
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", separatorWidth)))
	b.WriteString("\n\n")

	if m.LiveMetrics == nil || len(m.LiveMetrics.DataPoints) == 0 {
		b.WriteString(dimStyle.Render("Collecting metrics..."))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Auto-refreshing (1s) | c:clear | q/esc:back"))
		return b.String()
	}

	// Chart dimensions
	chartWidth := m.Width - 20
	if chartWidth < 30 {
		chartWidth = 30
	}
	if chartWidth > 100 {
		chartWidth = 100
	}

	latest := m.LiveMetrics.DataPoints[len(m.LiveMetrics.DataPoints)-1]

	// Calculate derived stats
	hitRate := float64(0)
	if latest.KeyspaceHits+latest.KeyspaceMisses > 0 {
		hitRate = float64(latest.KeyspaceHits) / float64(latest.KeyspaceHits+latest.KeyspaceMisses) * 100
	}
	cpuTotal := latest.UsedCPUSys + latest.UsedCPUUser

	// Build stat cards — Performance row
	perfOps := metricsCardBorder.Render(
		metricsCardLabel.Render("Ops/sec") + "\n" +
			metricsCardValue.Render(fmt.Sprintf("%.0f", latest.OpsPerSec)),
	)
	perfHit := metricsCardBorder.Render(
		metricsCardLabel.Render("Hit Rate") + "\n" +
			metricsGreenValue.Render(fmt.Sprintf("%.1f%%", hitRate)),
	)
	perfCPU := metricsCardBorder.Render(
		metricsCardLabel.Render("CPU (sys+user)") + "\n" +
			metricsYellowValue.Render(fmt.Sprintf("%.2fs", cpuTotal)),
	)

	b.WriteString(metricsSectionStyle.Render("  Performance"))
	b.WriteString("\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, perfOps, perfHit, perfCPU))
	b.WriteString("\n")

	// Resources row
	resMem := metricsCardBorder.Render(
		metricsCardLabel.Render("Memory") + "\n" +
			metricsCardValue.Render(formatBytes(latest.UsedMemoryBytes)),
	)
	resClients := metricsCardBorder.Render(
		metricsCardLabel.Render("Connected Clients") + "\n" +
			metricsGreenValue.Render(fmt.Sprintf("%d", latest.ConnectedClients)),
	)
	resBlocked := metricsCardBorder.Render(
		metricsCardLabel.Render("Blocked Clients") + "\n" +
			metricsYellowValue.Render(fmt.Sprintf("%d", latest.BlockedClients)),
	)

	b.WriteString(metricsSectionStyle.Render("  Resources"))
	b.WriteString("\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, resMem, resClients, resBlocked))
	b.WriteString("\n")

	// Network row
	netIn := metricsCardBorder.Render(
		metricsCardLabel.Render("Input KB/s") + "\n" +
			metricsCardValue.Render(fmt.Sprintf("%.2f", latest.InputKbps)),
	)
	netOut := metricsCardBorder.Render(
		metricsCardLabel.Render("Output KB/s") + "\n" +
			metricsCardValue.Render(fmt.Sprintf("%.2f", latest.OutputKbps)),
	)
	netTotal := metricsCardBorder.Render(
		metricsCardLabel.Render("Total Connections") + "\n" +
			metricsGreenValue.Render(fmt.Sprintf("%d", latest.TotalConnections)),
	)

	b.WriteString(metricsSectionStyle.Render("  Network"))
	b.WriteString("\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, netIn, netOut, netTotal))
	b.WriteString("\n\n")

	// Charts section
	b.WriteString(dimStyle.Render(strings.Repeat("─", separatorWidth)))
	b.WriteString("\n\n")

	// Ops/sec chart
	opsData := make([]float64, len(m.LiveMetrics.DataPoints))
	for i, dp := range m.LiveMetrics.DataPoints {
		opsData[i] = dp.OpsPerSec
	}
	b.WriteString(metricsChartBorder.Render(renderLineChart("Ops/sec", opsData, chartWidth, 6, lipgloss.Color("39"))))
	b.WriteString("\n")

	// Memory chart
	memData := make([]float64, len(m.LiveMetrics.DataPoints))
	for i, dp := range m.LiveMetrics.DataPoints {
		memData[i] = float64(dp.UsedMemoryBytes) / 1024 / 1024
	}
	b.WriteString(metricsChartBorder.Render(renderLineChart("Memory (MB)", memData, chartWidth, 6, lipgloss.Color("35"))))
	b.WriteString("\n")

	// Network chart
	netData := make([]float64, len(m.LiveMetrics.DataPoints))
	for i, dp := range m.LiveMetrics.DataPoints {
		netData[i] = dp.InputKbps + dp.OutputKbps
	}
	b.WriteString(metricsChartBorder.Render(renderLineChart("Network KB/s", netData, chartWidth, 5, lipgloss.Color("33"))))
	b.WriteString("\n")

	// Clients chart
	clientsData := make([]float64, len(m.LiveMetrics.DataPoints))
	for i, dp := range m.LiveMetrics.DataPoints {
		clientsData[i] = float64(dp.ConnectedClients)
	}
	b.WriteString(metricsChartBorder.Render(renderLineChart("Clients", clientsData, chartWidth, 5, lipgloss.Color("32"))))
	b.WriteString("\n")

	// CPU chart
	cpuData := make([]float64, len(m.LiveMetrics.DataPoints))
	for i, dp := range m.LiveMetrics.DataPoints {
		cpuData[i] = dp.UsedCPUSys + dp.UsedCPUUser
	}
	b.WriteString(metricsChartBorder.Render(renderLineChart("CPU (seconds)", cpuData, chartWidth, 5, lipgloss.Color("208"))))
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("Auto-refreshing (1s) | c:clear | q/esc:back"))

	return b.String()
}

// renderLineChart creates a bar chart using block characters
func renderLineChart(title string, data []float64, width, height int, color lipgloss.Color) string {
	if len(data) == 0 {
		return ""
	}

	var b strings.Builder

	// Find min/max for scaling
	minVal, maxVal := data[0], data[0]
	for _, v := range data {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	// Ensure we have a range
	if maxVal == minVal {
		maxVal = minVal + 1
	}
	rangeVal := maxVal - minVal

	// Current value
	current := data[len(data)-1]

	// Title with current/max values
	chartTitleStyle := lipgloss.NewStyle().Bold(true).Foreground(color)

	b.WriteString(chartTitleStyle.Render(title))
	b.WriteString(metricsCardLabel.Render(fmt.Sprintf("  %.1f", current)))
	b.WriteString(dimStyle.Render(fmt.Sprintf(" (max: %.1f)", maxVal)))
	b.WriteString("\n")

	// Resample data to fit width
	chartData := resampleData(data, width)

	// Block characters for bar heights (from empty to full)
	blocks := []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	chartStyle := lipgloss.NewStyle().Foreground(color)

	// Y-axis max label
	maxLabel := fmt.Sprintf("%6.1f ", maxVal)
	padLabel := strings.Repeat(" ", len(maxLabel))

	// Render the chart row by row from top to bottom
	for row := height - 1; row >= 0; row-- {
		if row == height-1 {
			b.WriteString(dimStyle.Render(maxLabel))
		} else {
			b.WriteString(padLabel)
		}
		for _, val := range chartData {
			// Normalize value to 0-1
			normalized := (val - minVal) / rangeVal
			// Total height in "sub-rows" (each character has 8 levels)
			totalSubRows := normalized * float64(height) * 8.0
			// How many full rows below this row
			fullRowsBelow := float64(row) * 8.0

			if totalSubRows >= fullRowsBelow+8 {
				// This row is fully filled
				b.WriteString(chartStyle.Render("█"))
			} else if totalSubRows > fullRowsBelow {
				// This row is partially filled
				partialFill := int(totalSubRows - fullRowsBelow)
				if partialFill > 7 {
					partialFill = 7
				}
				b.WriteString(chartStyle.Render(string(blocks[partialFill])))
			} else {
				// This row is empty
				b.WriteString(" ")
			}
		}
		b.WriteString("\n")
	}

	// Bottom axis
	b.WriteString(padLabel)
	b.WriteString(dimStyle.Render(strings.Repeat("─", width)))
	b.WriteString("\n")

	return b.String()
}

// resampleData resamples data to fit the target width
func resampleData(data []float64, targetWidth int) []float64 {
	if len(data) == 0 {
		return data
	}
	if len(data) <= targetWidth {
		// Pad with the same values
		result := make([]float64, targetWidth)
		for i := range result {
			idx := i * len(data) / targetWidth
			if idx >= len(data) {
				idx = len(data) - 1
			}
			result[i] = data[idx]
		}
		return result
	}

	// Downsample
	result := make([]float64, targetWidth)
	for i := range result {
		startIdx := i * len(data) / targetWidth
		endIdx := (i + 1) * len(data) / targetWidth
		if endIdx > len(data) {
			endIdx = len(data)
		}
		if startIdx >= endIdx {
			startIdx = endIdx - 1
		}
		if startIdx < 0 {
			startIdx = 0
		}

		// Average the values in this bucket
		sum := 0.0
		count := 0
		for j := startIdx; j < endIdx; j++ {
			sum += data[j]
			count++
		}
		if count > 0 {
			result[i] = sum / float64(count)
		}
	}
	return result
}
