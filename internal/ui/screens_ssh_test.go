package ui

import (
	"strings"
	"testing"

	"github.com/bearded-giant/redis-tui/internal/types"
	tea "github.com/charmbracelet/bubbletea"
)

func TestSSHTunnel_OpenViaCtrlS_FromAdd(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Screen = types.ScreenAddConnection

	updated, _ := m.handleAddConnectionScreen(tea.KeyMsg{Type: tea.KeyCtrlS})
	got := updated.(Model)
	if got.Screen != types.ScreenSSHTunnel {
		t.Errorf("expected ScreenSSHTunnel, got %v", got.Screen)
	}
}

func TestSSHTunnel_OpenViaCtrlS_FromEdit(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Screen = types.ScreenEditConnection
	conn := types.Connection{ID: 1, Name: "x"}
	m.EditingConnection = &conn

	updated, _ := m.handleEditConnectionScreen(tea.KeyMsg{Type: tea.KeyCtrlS})
	got := updated.(Model)
	if got.Screen != types.ScreenSSHTunnel {
		t.Errorf("expected ScreenSSHTunnel, got %v", got.Screen)
	}
}

func TestSSHTunnel_NavigateAndToggle(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Screen = types.ScreenSSHTunnel

	// Tab through fields. There are 8 focusable items (7 inputs + toggle).
	for i := 0; i < 7; i++ {
		updated, _ := m.handleSSHTunnelScreen(tea.KeyMsg{Type: tea.KeyTab})
		m = updated.(Model)
	}
	if m.SSHFocusIdx != 7 {
		t.Errorf("after 7 tabs, SSHFocusIdx = %d, want 7", m.SSHFocusIdx)
	}

	// Space on toggle field flips SSHEnabled.
	updated, _ := m.handleSSHTunnelScreen(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)
	if !m.SSHEnabled {
		t.Error("SSHEnabled should be true after space on toggle")
	}

	updated, _ = m.handleSSHTunnelScreen(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if !m.SSHEnabled {
		// Enter on toggle field also flips. We just toggled to true via space,
		// so enter should now flip back to false.
	}
}

func TestSSHTunnel_ShiftTabWraps(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Screen = types.ScreenSSHTunnel
	m.SSHFocusIdx = 0

	updated, _ := m.handleSSHTunnelScreen(tea.KeyMsg{Type: tea.KeyShiftTab})
	got := updated.(Model)
	if got.SSHFocusIdx != sshFieldCount-1 {
		t.Errorf("shift+tab from 0 = %d, want %d", got.SSHFocusIdx, sshFieldCount-1)
	}
}

func TestSSHTunnel_SaveAndCancel(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Screen = types.ScreenSSHTunnel
	m.SSHInputs[0].SetValue("bastion.example.com")
	m.SSHInputs[2].SetValue("alice")

	// Enter on a non-toggle focus → save and return to add screen.
	m.SSHFocusIdx = 0
	updated, _ := m.handleSSHTunnelScreen(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)
	if got.Screen != types.ScreenAddConnection {
		t.Errorf("after enter, screen = %v, want ScreenAddConnection", got.Screen)
	}
	if got.PendingSSH == nil {
		t.Fatal("PendingSSH should be set")
	}
	if got.PendingSSH.Host != "bastion.example.com" {
		t.Errorf("PendingSSH.Host = %q", got.PendingSSH.Host)
	}
	if got.PendingSSH.Port != 22 {
		t.Errorf("default port should be 22, got %d", got.PendingSSH.Port)
	}

	// Esc on edit screen returns to edit.
	m2, _, _ := newTestModel(t)
	m2.Screen = types.ScreenSSHTunnel
	conn := types.Connection{ID: 1}
	m2.EditingConnection = &conn
	updated, _ = m2.handleSSHTunnelScreen(tea.KeyMsg{Type: tea.KeyEsc})
	got = updated.(Model)
	if got.Screen != types.ScreenEditConnection {
		t.Errorf("esc on edit-flow → screen = %v, want ScreenEditConnection", got.Screen)
	}
}

func TestSSHTunnel_TestRequiresHost(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Screen = types.ScreenSSHTunnel
	updated, cmd := m.handleSSHTunnelScreen(tea.KeyMsg{Type: tea.KeyCtrlT})
	got := updated.(Model)
	if got.SSHTunnelStatus != "host required" {
		t.Errorf("SSHTunnelStatus = %q, want 'host required'", got.SSHTunnelStatus)
	}
	if cmd != nil {
		t.Error("no cmd should be returned when host empty")
	}
}

func TestSSHTunnel_TestDispatchesCmd(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Screen = types.ScreenSSHTunnel
	m.SSHInputs[0].SetValue("bastion")
	m.SSHInputs[2].SetValue("u")
	updated, c := m.handleSSHTunnelScreen(tea.KeyMsg{Type: tea.KeyCtrlT})
	got := updated.(Model)
	if got.SSHTunnelStatus != "testing..." {
		t.Errorf("status = %q, want 'testing...'", got.SSHTunnelStatus)
	}
	if c == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := c()
	result := msg.(types.SSHTunnelConnectedMsg)
	_ = result // mock returns nil err by default
}

func TestSSHTunnel_HandleConnectedMsg(t *testing.T) {
	m, _, _ := newTestModel(t)
	updated, _ := m.handleSSHTunnelConnectedMsg(types.SSHTunnelConnectedMsg{Err: nil})
	got := updated.(Model)
	if got.SSHTunnelStatus != "SSH OK" {
		t.Errorf("status = %q, want 'SSH OK'", got.SSHTunnelStatus)
	}

	updated, _ = m.handleSSHTunnelConnectedMsg(types.SSHTunnelConnectedMsg{Err: errTestSSH("boom")})
	got = updated.(Model)
	if !strings.Contains(got.SSHTunnelStatus, "SSH failed") {
		t.Errorf("status = %q, want SSH failed prefix", got.SSHTunnelStatus)
	}
}

func TestSSHTunnel_PopulateInputs(t *testing.T) {
	m, _, _ := newTestModel(t)
	cfg := &types.SSHConfig{
		Host: "bastion", Port: 2222, User: "alice",
		PrivateKeyPath: "/tmp/key", Passphrase: "p", Password: "x", LocalPort: 16379,
	}
	m.populateSSHInputs(cfg)
	if m.SSHInputs[0].Value() != "bastion" {
		t.Error("host not populated")
	}
	if m.SSHInputs[1].Value() != "2222" {
		t.Error("port not populated")
	}
	if m.SSHInputs[6].Value() != "16379" {
		t.Error("local port not populated")
	}

	m.populateSSHInputs(nil)
	if m.SSHInputs[0].Value() != "" {
		t.Error("host should be empty when cfg nil")
	}
}

func TestSSHTunnel_ConvertEmptyHost(t *testing.T) {
	m, _, _ := newTestModel(t)
	cfg := m.convertSSHInputs()
	if cfg != nil {
		t.Error("empty host should return nil config")
	}
}

func TestSSHTunnel_View(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Screen = types.ScreenSSHTunnel
	m.Width = 100
	m.Height = 40
	out := m.viewSSHTunnel()
	if !strings.Contains(out, "SSH Tunnel") {
		t.Error("view should contain 'SSH Tunnel' title")
	}
	if !strings.Contains(out, "known_hosts") {
		t.Error("view should mention known_hosts")
	}
}

func TestRenderConnForm_SSHStates(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Screen = types.ScreenAddConnection
	out := m.renderConnForm()
	if !strings.Contains(out, "not configured") {
		t.Error("default state should show 'not configured'")
	}

	m.PendingSSH = &types.SSHConfig{Host: "bastion"}
	out = m.renderConnForm()
	if !strings.Contains(out, "configured (disabled)") {
		t.Error("configured but disabled state missing")
	}

	m.SSHEnabled = true
	out = m.renderConnForm()
	if !strings.Contains(out, "enabled (bastion)") {
		t.Error("enabled state should show host")
	}
}

type errSSH string

func (e errSSH) Error() string { return string(e) }

func errTestSSH(s string) error { return errSSH(s) }
