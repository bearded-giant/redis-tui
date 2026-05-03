package ui

import (
	"github.com/bearded-giant/redis-tui/internal/types"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleConnectionsScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.SelectedConnIdx > 0 {
			m.SelectedConnIdx--
			m.ConnectionError = "" // Clear error on navigation
		}
	case "down", "j":
		if m.SelectedConnIdx < len(m.Connections)-1 {
			m.SelectedConnIdx++
			m.ConnectionError = "" // Clear error on navigation
		}
	case "enter":
		if len(m.Connections) > 0 && m.SelectedConnIdx < len(m.Connections) {
			conn := m.Connections[m.SelectedConnIdx]
			m.CurrentConn = &conn
			m.Loading = true
			m.StatusMsg = "Connecting..."
			m.ConnectionError = "" // Clear any previous connection error
			return m, m.Cmds.Connect(conn)
		}
	case "a", "n":
		m.Screen = types.ScreenAddConnection
		m.resetConnInputs()
	case "e":
		if len(m.Connections) > 0 && m.SelectedConnIdx < len(m.Connections) {
			conn := m.Connections[m.SelectedConnIdx]
			m.EditingConnection = &conn
			m.populateConnInputs(conn)
			m.Screen = types.ScreenEditConnection
		}
	case "d", "delete", "backspace":
		if len(m.Connections) > 0 && m.SelectedConnIdx < len(m.Connections) {
			m.ConfirmType = "connection"
			m.ConfirmData = m.Connections[m.SelectedConnIdx]
			m.Screen = types.ScreenConfirmDelete
		}
	case "r":
		m.Loading = true
		return m, m.Cmds.LoadConnections()
	}
	return m, nil
}

func (m Model) handleAddConnectionScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	fieldCount := m.connFieldCount()

	switch msg.String() {
	case "tab", "down":
		m.blurConnField()
		m.ConnFocusIdx = (m.ConnFocusIdx + 1) % fieldCount
		m.focusConnField()
	case "shift+tab", "up":
		m.blurConnField()
		m.ConnFocusIdx--
		if m.ConnFocusIdx < 0 {
			m.ConnFocusIdx = fieldCount - 1
		}
		m.focusConnField()
	case " ":
		if m.ConnFocusIdx == 5 {
			m.ConnClusterMode = !m.ConnClusterMode
			return m, nil
		}
		return m.updateConnInputs(msg)
	case "enter":
		if m.ConnFocusIdx == 5 {
			m.ConnClusterMode = !m.ConnClusterMode
			return m, nil
		}
		if m.ConnInputs[0].Value() != "" && m.ConnInputs[1].Value() != "" {
			m.Loading = true
			conn := m.convertCurrentInputsToConnection(m.ConnInputs, "add")
			return m, m.Cmds.AddConnection(
				conn,
			)
		}
	case "ctrl+t":
		m.Loading = true
		m.Screen = types.ScreenTestConnection
		conn := m.convertCurrentInputsToConnection(m.ConnInputs, "test")
		return m, m.Cmds.TestConnection(
			conn,
		)
	case "ctrl+s":
		m.populateSSHInputs(m.PendingSSH)
		m.Screen = types.ScreenSSHTunnel
		return m, nil
	case "esc":
		m.Screen = types.ScreenConnections
		m.resetConnInputs()
	default:
		return m.updateConnInputs(msg)
	}
	return m, nil
}

// connInputIndex maps a ConnFocusIdx to the actual ConnInputs array index.
// Indices 0-4 map directly to ConnInputs[0-4], index 5 is the cluster toggle (no input),
// and index 6 maps to ConnInputs[5] (Database).
func connInputIndex(focusIdx int) int {
	if focusIdx <= 4 {
		return focusIdx
	}
	if focusIdx == 6 {
		return 5 // Database input
	}
	return -1 // cluster toggle, no text input
}

func (m *Model) blurConnField() {
	idx := connInputIndex(m.ConnFocusIdx)
	if idx >= 0 && idx < len(m.ConnInputs) {
		m.ConnInputs[idx].Blur()
	}
}

func (m *Model) focusConnField() {
	idx := connInputIndex(m.ConnFocusIdx)
	if idx >= 0 && idx < len(m.ConnInputs) {
		m.ConnInputs[idx].Focus()
	}
}

func (m Model) updateConnInputs(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Only update the focused text input, not the cluster toggle
	idx := connInputIndex(m.ConnFocusIdx)
	if idx >= 0 && idx < len(m.ConnInputs) {
		var inputCmd tea.Cmd
		m.ConnInputs[idx], inputCmd = m.ConnInputs[idx].Update(msg)
		return m, inputCmd
	}
	return m, nil
}

func (m Model) handleEditConnectionScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	fieldCount := m.connFieldCount()

	switch msg.String() {
	case "tab", "down":
		m.blurConnField()
		m.ConnFocusIdx = (m.ConnFocusIdx + 1) % fieldCount
		m.focusConnField()
	case "shift+tab", "up":
		m.blurConnField()
		m.ConnFocusIdx--
		if m.ConnFocusIdx < 0 {
			m.ConnFocusIdx = fieldCount - 1
		}
		m.focusConnField()
	case " ":
		if m.ConnFocusIdx == 5 {
			m.ConnClusterMode = !m.ConnClusterMode
			return m, nil
		}
		return m.updateConnInputs(msg)
	case "enter":
		if m.ConnFocusIdx == 5 {
			m.ConnClusterMode = !m.ConnClusterMode
			return m, nil
		}
		if m.EditingConnection != nil && m.ConnInputs[0].Value() != "" && m.ConnInputs[1].Value() != "" {
			m.Loading = true
			conn := m.convertCurrentInputsToConnection(m.ConnInputs, "edit")
			return m, m.Cmds.UpdateConnection(
				conn,
			)
		}
	case "ctrl+s":
		m.populateSSHInputs(m.PendingSSH)
		m.Screen = types.ScreenSSHTunnel
		return m, nil
	case "esc":
		m.Screen = types.ScreenConnections
		m.EditingConnection = nil
		m.resetConnInputs()
	default:
		return m.updateConnInputs(msg)
	}
	return m, nil
}

func (m Model) handleTestConnectionScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.Screen = types.ScreenAddConnection
	}
	return m, nil
}

// SSH tunnel sub-screen. Reached via Ctrl+S from Add/Edit connection.
// Form fields:
//
//	0 host, 1 port, 2 user, 3 key path, 4 passphrase, 5 password, 6 local port
//
// Plus a focusable "SSH enabled" toggle at index 7.
const sshFieldCount = 8

func (m Model) handleSSHTunnelScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "down":
		m.blurSSHField()
		m.SSHFocusIdx = (m.SSHFocusIdx + 1) % sshFieldCount
		m.focusSSHField()
	case "shift+tab", "up":
		m.blurSSHField()
		m.SSHFocusIdx--
		if m.SSHFocusIdx < 0 {
			m.SSHFocusIdx = sshFieldCount - 1
		}
		m.focusSSHField()
	case " ":
		if m.SSHFocusIdx == 7 {
			m.SSHEnabled = !m.SSHEnabled
			return m, nil
		}
		return m.updateSSHInputs(msg)
	case "ctrl+t":
		cfg := m.convertSSHInputs()
		if cfg == nil {
			m.SSHTunnelStatus = "host required"
			return m, nil
		}
		m.SSHTunnelStatus = "testing..."
		return m, m.Cmds.TestSSHConnection(cfg)
	case "enter":
		if m.SSHFocusIdx == 7 {
			m.SSHEnabled = !m.SSHEnabled
			return m, nil
		}
		// Save: stash config to pending buffer, return to caller screen.
		m.PendingSSH = m.convertSSHInputs()
		if m.PendingSSH == nil {
			m.SSHEnabled = false
		}
		if m.EditingConnection != nil {
			m.Screen = types.ScreenEditConnection
		} else {
			m.Screen = types.ScreenAddConnection
		}
		return m, nil
	case "esc":
		// Cancel: discard input changes, keep prior PendingSSH.
		if m.EditingConnection != nil {
			m.Screen = types.ScreenEditConnection
		} else {
			m.Screen = types.ScreenAddConnection
		}
		return m, nil
	default:
		return m.updateSSHInputs(msg)
	}
	return m, nil
}

func sshInputIndex(focusIdx int) int {
	if focusIdx >= 0 && focusIdx <= 6 {
		return focusIdx
	}
	return -1
}

func (m *Model) blurSSHField() {
	idx := sshInputIndex(m.SSHFocusIdx)
	if idx >= 0 && idx < len(m.SSHInputs) {
		m.SSHInputs[idx].Blur()
	}
}

func (m *Model) focusSSHField() {
	idx := sshInputIndex(m.SSHFocusIdx)
	if idx >= 0 && idx < len(m.SSHInputs) {
		m.SSHInputs[idx].Focus()
	}
}

func (m Model) updateSSHInputs(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	idx := sshInputIndex(m.SSHFocusIdx)
	if idx >= 0 && idx < len(m.SSHInputs) {
		var inputCmd tea.Cmd
		m.SSHInputs[idx], inputCmd = m.SSHInputs[idx].Update(msg)
		return m, inputCmd
	}
	return m, nil
}
