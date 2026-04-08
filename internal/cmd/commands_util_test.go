package cmd

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/davidbudnick/redis-tui/internal/types"
)

// roundTripFunc lets us plug in a fake transport for the version HTTP client.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func withVersionTransport(t *testing.T, rt http.RoundTripper) {
	t.Helper()
	orig := versionHTTPClient.Transport
	versionHTTPClient.Transport = rt
	t.Cleanup(func() { versionHTTPClient.Transport = orig })
}

func fakeResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

func TestCheckVersion(t *testing.T) {
	t.Run("empty version returns empty msg", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.CheckVersion("")()
		result := msg.(types.UpdateAvailableMsg)
		if result.LatestVersion != "" {
			t.Errorf("LatestVersion = %q, want empty", result.LatestVersion)
		}
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("dev version returns empty msg", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.CheckVersion("dev")()
		result := msg.(types.UpdateAvailableMsg)
		if result.LatestVersion != "" {
			t.Errorf("LatestVersion = %q, want empty", result.LatestVersion)
		}
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("same version returns empty latest", func(t *testing.T) {
		withVersionTransport(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
			return fakeResponse(http.StatusOK, `{"tag_name":"v1.2.3"}`), nil
		}))
		cmds := NewCommands(nil, nil)
		msg := cmds.CheckVersion("v1.2.3")()
		result := msg.(types.UpdateAvailableMsg)
		if result.LatestVersion != "" {
			t.Errorf("LatestVersion = %q, want empty on match", result.LatestVersion)
		}
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("newer version populates latest", func(t *testing.T) {
		withVersionTransport(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
			return fakeResponse(http.StatusOK, `{"tag_name":"v2.0.0"}`), nil
		}))
		cmds := NewCommands(nil, nil)
		msg := cmds.CheckVersion("v1.0.0")()
		result := msg.(types.UpdateAvailableMsg)
		if result.LatestVersion != "v2.0.0" {
			t.Errorf("LatestVersion = %q, want v2.0.0", result.LatestVersion)
		}
		if result.UpgradeCmd == "" {
			t.Error("expected non-empty upgrade cmd")
		}
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("fetch error surfaces", func(t *testing.T) {
		withVersionTransport(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
			return fakeResponse(http.StatusInternalServerError, ""), nil
		}))
		cmds := NewCommands(nil, nil)
		msg := cmds.CheckVersion("v1.0.0")()
		result := msg.(types.UpdateAvailableMsg)
		if result.Err == nil {
			t.Error("expected error on non-200 response")
		}
	})
}

func TestFetchLatestTag(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		withVersionTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if !strings.Contains(r.URL.String(), "/releases/latest") {
				t.Errorf("unexpected URL: %s", r.URL)
			}
			return fakeResponse(http.StatusOK, `{"tag_name":"v9.9.9"}`), nil
		}))
		tag, err := fetchLatestTag()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tag != "v9.9.9" {
			t.Errorf("tag = %q, want v9.9.9", tag)
		}
	})

	t.Run("http error", func(t *testing.T) {
		withVersionTransport(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, io.ErrUnexpectedEOF
		}))
		if _, err := fetchLatestTag(); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("non-200", func(t *testing.T) {
		withVersionTransport(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
			return fakeResponse(http.StatusNotFound, ""), nil
		}))
		if _, err := fetchLatestTag(); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("bad json", func(t *testing.T) {
		withVersionTransport(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
			return fakeResponse(http.StatusOK, "not json"), nil
		}))
		if _, err := fetchLatestTag(); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("empty tag", func(t *testing.T) {
		withVersionTransport(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
			return fakeResponse(http.StatusOK, `{"tag_name":""}`), nil
		}))
		if _, err := fetchLatestTag(); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("NewRequest error", func(t *testing.T) {
		origURL := githubReleaseURL
		t.Cleanup(func() { githubReleaseURL = origURL })
		// Invalid control character in URL forces http.NewRequest to fail.
		githubReleaseURL = "http://\x00invalid"
		if _, err := fetchLatestTag(); err == nil {
			t.Error("expected error from invalid URL")
		}
	})
}

func TestDetectUpgradeCmd(t *testing.T) {
	origExec := osExecutable
	t.Cleanup(func() { osExecutable = origExec })

	cases := []struct {
		name     string
		path     string
		err      error
		expected string
	}{
		{"executable error fallback", "", errFake, "redis-tui --update"},
		{"homebrew Cellar path", "/usr/local/Cellar/redis-tui/1.0/bin/redis-tui", nil, "brew upgrade redis-tui"},
		{"homebrew path", "/opt/homebrew/bin/redis-tui", nil, "brew upgrade redis-tui"},
		{"go bin path", "/Users/me/go/bin/redis-tui", nil, "go install github.com/davidbudnick/redis-tui@latest"},
		{"generic path", "/usr/local/bin/redis-tui", nil, "redis-tui --update"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			osExecutable = func() (string, error) { return tc.path, tc.err }
			if got := detectUpgradeCmd(); got != tc.expected {
				t.Errorf("detectUpgradeCmd() = %q, want %q", got, tc.expected)
			}
		})
	}
}

var errFake = errTestFake("boom")

type errTestFake string

func (e errTestFake) Error() string { return string(e) }

func TestWatchKeyTick(t *testing.T) {
	t.Run("returns non-nil cmd", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		cmd := cmds.WatchKeyTick()
		if cmd == nil {
			t.Error("expected non-nil cmd from WatchKeyTick")
		}
	})

	t.Run("tick fires and returns WatchTickMsg", func(t *testing.T) {
		// tea.Tick waits ~1s; this test validates the inner callback runs.
		cmds := NewCommands(nil, nil)
		msg := cmds.WatchKeyTick()()
		if _, ok := msg.(types.WatchTickMsg); !ok {
			t.Errorf("expected WatchTickMsg, got %T", msg)
		}
	})
}

func TestCopyToClipboard(t *testing.T) {
	t.Run("returns cmd", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		cmd := cmds.CopyToClipboard("test content")
		if cmd == nil {
			t.Fatal("expected non-nil cmd from CopyToClipboard")
		}
		// Execute the command - it may fail in CI if pbcopy is not available
		msg := cmd()
		result := msg.(types.ClipboardCopiedMsg)
		if result.Content != "test content" {
			t.Errorf("Content = %q, want %q", result.Content, "test content")
		}
		// Note: result.Err may be non-nil if pbcopy is unavailable (e.g. in CI)
	})
}
