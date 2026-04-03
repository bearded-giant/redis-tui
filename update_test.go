package main

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFetchLatestVersion(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v1.2.3"}`)
		}))
		defer srv.Close()

		old := githubAPIBase
		githubAPIBase = srv.URL
		defer func() { githubAPIBase = old }()

		ver, err := fetchLatestVersion()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ver != "v1.2.3" {
			t.Errorf("version = %q, want %q", ver, "v1.2.3")
		}
	})

	t.Run("non-200 status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		old := githubAPIBase
		githubAPIBase = srv.URL
		defer func() { githubAPIBase = old }()

		_, err := fetchLatestVersion()
		if err == nil {
			t.Fatal("expected error for non-200 status")
		}
	})

	t.Run("bad JSON", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `not json`)
		}))
		defer srv.Close()

		old := githubAPIBase
		githubAPIBase = srv.URL
		defer func() { githubAPIBase = old }()

		_, err := fetchLatestVersion()
		if err == nil {
			t.Fatal("expected error for bad JSON")
		}
	})

	t.Run("empty tag", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":""}`)
		}))
		defer srv.Close()

		old := githubAPIBase
		githubAPIBase = srv.URL
		defer func() { githubAPIBase = old }()

		_, err := fetchLatestVersion()
		if err == nil {
			t.Fatal("expected error for empty tag")
		}
	})
}

func TestArchiveName(t *testing.T) {
	tests := []struct {
		ver, goos, goarch string
		want              string
	}{
		{"1.0.0", "darwin", "arm64", "redis-tui_1.0.0_Darwin_arm64.tar.gz"},
		{"1.0.0", "linux", "amd64", "redis-tui_1.0.0_Linux_x86_64.tar.gz"},
		{"2.1.0", "windows", "amd64", "redis-tui_2.1.0_Windows_x86_64.zip"},
		{"1.0.0", "darwin", "amd64", "redis-tui_1.0.0_Darwin_x86_64.tar.gz"},
		{"1.0.0", "linux", "arm64", "redis-tui_1.0.0_Linux_arm64.tar.gz"},
	}
	for _, tt := range tests {
		t.Run(tt.goos+"/"+tt.goarch, func(t *testing.T) {
			got := archiveName(tt.ver, tt.goos, tt.goarch)
			if got != tt.want {
				t.Errorf("archiveName(%q, %q, %q) = %q, want %q", tt.ver, tt.goos, tt.goarch, got, tt.want)
			}
		})
	}
}

func TestVerifyChecksum(t *testing.T) {
	t.Run("matching checksum", func(t *testing.T) {
		tmpDir := t.TempDir()

		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		if err := os.WriteFile(archivePath, []byte("test archive content"), 0644); err != nil {
			t.Fatal(err)
		}

		hash := sha256.Sum256([]byte("test archive content"))
		checksumContent := fmt.Sprintf("%x  test.tar.gz\n", hash)
		checksumPath := filepath.Join(tmpDir, "checksums.txt")
		if err := os.WriteFile(checksumPath, []byte(checksumContent), 0644); err != nil {
			t.Fatal(err)
		}

		if err := verifyChecksum(archivePath, checksumPath, "test.tar.gz"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("mismatching checksum", func(t *testing.T) {
		tmpDir := t.TempDir()

		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		if err := os.WriteFile(archivePath, []byte("test archive content"), 0644); err != nil {
			t.Fatal(err)
		}

		checksumContent := "0000000000000000000000000000000000000000000000000000000000000000  test.tar.gz\n"
		checksumPath := filepath.Join(tmpDir, "checksums.txt")
		if err := os.WriteFile(checksumPath, []byte(checksumContent), 0644); err != nil {
			t.Fatal(err)
		}

		if err := verifyChecksum(archivePath, checksumPath, "test.tar.gz"); err == nil {
			t.Error("expected error for mismatching checksum")
		}
	})

	t.Run("missing archive in checksums", func(t *testing.T) {
		tmpDir := t.TempDir()

		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		if err := os.WriteFile(archivePath, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}

		checksumContent := "abcdef1234567890  other_file.tar.gz\n"
		checksumPath := filepath.Join(tmpDir, "checksums.txt")
		if err := os.WriteFile(checksumPath, []byte(checksumContent), 0644); err != nil {
			t.Fatal(err)
		}

		if err := verifyChecksum(archivePath, checksumPath, "test.tar.gz"); err == nil {
			t.Error("expected error for missing archive in checksums")
		}
	})
}

func TestRunUpdateDevVersion(t *testing.T) {
	err := runUpdate("dev")
	if err == nil {
		t.Fatal("expected error for dev version")
	}
	if got := err.Error(); !contains(got, "development build") {
		t.Errorf("error = %q, want it to mention 'development build'", got)
	}
}

func TestRunUpdateNonSemver(t *testing.T) {
	err := runUpdate("abc123")
	if err == nil {
		t.Fatal("expected error for non-semver version")
	}
	if got := err.Error(); !contains(got, "development build") {
		t.Errorf("error = %q, want it to mention 'development build'", got)
	}
}

func TestIsSemver(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"1.0.0", true},
		{"v1.0.0", true},
		{"0.1.2", true},
		{"dev", false},
		{"abc", false},
		{"", false},
		{"1.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isSemver(tt.input); got != tt.want {
				t.Errorf("isSemver(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsHomebrew(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/usr/local/Cellar/redis-tui/1.0.0/bin/redis-tui", true},
		{"/opt/homebrew/bin/redis-tui", true},
		{"/usr/local/bin/redis-tui", false},
		{"/home/user/go/bin/redis-tui", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isHomebrew(tt.path); got != tt.want {
				t.Errorf("isHomebrew(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
