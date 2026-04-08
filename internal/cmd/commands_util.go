package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"

	tea "github.com/charmbracelet/bubbletea"
)

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

func (c *Commands) CopyToClipboard(content string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(content)
		err := cmd.Run()
		return types.ClipboardCopiedMsg{Content: content, Err: err}
	}
}

// Version check helpers

const githubRepo = "davidbudnick/redis-tui"

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
		return "go install github.com/davidbudnick/redis-tui@latest"
	}
	return "redis-tui --update"
}
