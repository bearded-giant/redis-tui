package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/bearded-giant/redis-tui/internal/types"

	tea "github.com/charmbracelet/bubbletea"
)

// detectedOS is the runtime OS, overridable in tests.
var (
	detectedOS = runtime.GOOS
)

// clipboardCmd returns the command name and args for the platform's clipboard
// utility. Returns ("", nil) if none is available.
func clipboardCmd() (string, []string) {
	switch detectedOS {
	case "darwin":
		return "pbcopy", nil
	case "windows":
		return "clip", nil
	default: // linux, freebsd, etc.
		if path, err := exec.LookPath("xclip"); err == nil {
			return path, []string{"-selection", "clipboard"}
		}
		if path, err := exec.LookPath("xsel"); err == nil {
			return path, []string{"--clipboard", "--input"}
		}
		return "", nil
	}
}

func (c *Commands) CheckVersion(currentVersion string) tea.Cmd {
	return func() tea.Msg {
		if currentVersion == "" || currentVersion == "dev" {
			return types.UpdateAvailableMsg{}
		}

		latest, err := fetchLatestTag()
		if err != nil {
			return types.UpdateAvailableMsg{Err: err}
		}

		if strings.TrimPrefix(latest, "v") == strings.TrimPrefix(currentVersion, "v") {
			return types.UpdateAvailableMsg{}
		}

		upgradeCmd := detectUpgradeCmd()

		return types.UpdateAvailableMsg{
			LatestVersion: latest,
			UpgradeCmd:    upgradeCmd,
		}
	}
}

func (c *Commands) WatchKeyTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return types.WatchTickMsg{}
	})
}

// CopyToClipboard writes content to the system clipboard using two strategies
// in parallel to maximize reliability across terminal/multiplexer combos:
//   1. OSC 52 escape — works in modern terminals (wezterm, iTerm2, kitty,
//      alacritty, ghostty) and in tmux when `set-clipboard on` is configured.
//      No subprocess, no hang risk.
//   2. Native subprocess (pbcopy / xclip / xsel / clip) with a 2s timeout so
//      a hung pbcopy (known issue inside some altscreen TUIs) can't wedge
//      the UI status forever.
//
// Either path landing the content counts as success.
func (c *Commands) CopyToClipboard(content string) tea.Cmd {
	return func() tea.Msg {
		writeOSC52(content)

		name, args := clipboardCmd()
		if name == "" {
			// OSC 52 was best-effort; surface as success since most modern
			// terminals will have caught it.
			return types.ClipboardCopiedMsg{Content: content, Err: nil}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, name, args...) // #nosec G204 -- name/args from clipboardCmd() are hardcoded values
		cmd.Stdin = strings.NewReader(content)
		err := cmd.Run()
		if ctx.Err() == context.DeadlineExceeded {
			// pbcopy hung; OSC 52 already attempted, treat as soft success.
			return types.ClipboardCopiedMsg{Content: content, Err: nil}
		}
		return types.ClipboardCopiedMsg{Content: content, Err: err}
	}
}

// writeOSC52 emits an OSC 52 clipboard-set escape directly to the terminal.
// Writing to /dev/tty avoids racing with bubbletea's stdout renderer; falls
// back to stdout for environments without a tty (CI, pipes).
// Inside tmux/screen the sequence is wrapped via DCS passthrough.
func writeOSC52(content string) {
	b64 := base64.StdEncoding.EncodeToString([]byte(content))
	seq := "\x1b]52;c;" + b64 + "\x07"
	if term := os.Getenv("TERM"); strings.HasPrefix(term, "screen") || strings.HasPrefix(term, "tmux") {
		seq = "\x1bPtmux;\x1b" + seq + "\x1b\\"
	}
	if tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0); err == nil {
		_, _ = tty.WriteString(seq)
		_ = tty.Close()
		return
	}
	_, _ = os.Stdout.WriteString(seq)
}

// Version check helpers

const githubRepo = "bearded-giant/redis-tui"

var (
	versionHTTPClient = &http.Client{Timeout: 10 * time.Second}
	// githubReleaseURL is overridable in tests to cover the NewRequest error path.
	githubReleaseURL = fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)
)

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func fetchLatestTag() (string, error) {
	req, err := http.NewRequest(http.MethodGet, githubReleaseURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := versionHTTPClient.Do(req) // #nosec G107 G704 - URL built from hardcoded GitHub API base
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	if release.TagName == "" {
		return "", fmt.Errorf("empty tag_name in response")
	}

	return release.TagName, nil
}

// osExecutable indirection lets tests override executable path detection.
var osExecutable = os.Executable

func detectUpgradeCmd() string {
	execPath, err := osExecutable()
	if err != nil {
		return "redis-tui --update"
	}

	if strings.Contains(execPath, "/Cellar/") || strings.Contains(execPath, "/homebrew/") {
		return "brew upgrade redis-tui"
	}
	if strings.Contains(execPath, "/go/bin/") {
		return "go install github.com/bearded-giant/redis-tui@latest"
	}
	return "redis-tui --update"
}
