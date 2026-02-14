package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/davidbudnick/redis-tui/internal/types"
)

const githubRepo = "davidbudnick/redis-tui"

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func CheckVersionCmd(currentVersion string) tea.Cmd {
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

func fetchLatestTag() (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req) // #nosec G107 - URL built from hardcoded GitHub API base
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

func detectUpgradeCmd() string {
	execPath, err := os.Executable()
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
