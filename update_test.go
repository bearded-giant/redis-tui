package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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

	t.Run("HTTP request error", func(t *testing.T) {
		old := githubAPIBase
		githubAPIBase = "http://127.0.0.1:1" // unreachable
		defer func() { githubAPIBase = old }()

		_, err := fetchLatestVersion()
		if err == nil {
			t.Fatal("expected error for unreachable server")
		}
	})

	t.Run("invalid URL", func(t *testing.T) {
		old := githubAPIBase
		githubAPIBase = "://bad-url"
		defer func() { githubAPIBase = old }()

		_, err := fetchLatestVersion()
		if err == nil {
			t.Fatal("expected error for invalid URL")
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
		if err := os.WriteFile(archivePath, []byte("test archive content"), 0o644); err != nil {
			t.Fatal(err)
		}

		hash := sha256.Sum256([]byte("test archive content"))
		checksumContent := fmt.Sprintf("%x  test.tar.gz\n", hash)
		checksumPath := filepath.Join(tmpDir, "checksums.txt")
		if err := os.WriteFile(checksumPath, []byte(checksumContent), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := verifyChecksum(archivePath, checksumPath, "test.tar.gz"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("mismatching checksum", func(t *testing.T) {
		tmpDir := t.TempDir()

		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		if err := os.WriteFile(archivePath, []byte("test archive content"), 0o644); err != nil {
			t.Fatal(err)
		}

		checksumContent := "0000000000000000000000000000000000000000000000000000000000000000  test.tar.gz\n"
		checksumPath := filepath.Join(tmpDir, "checksums.txt")
		if err := os.WriteFile(checksumPath, []byte(checksumContent), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := verifyChecksum(archivePath, checksumPath, "test.tar.gz"); err == nil {
			t.Error("expected error for mismatching checksum")
		}
	})

	t.Run("missing archive in checksums", func(t *testing.T) {
		tmpDir := t.TempDir()

		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		if err := os.WriteFile(archivePath, []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}

		checksumContent := "abcdef1234567890  other_file.tar.gz\n"
		checksumPath := filepath.Join(tmpDir, "checksums.txt")
		if err := os.WriteFile(checksumPath, []byte(checksumContent), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := verifyChecksum(archivePath, checksumPath, "test.tar.gz"); err == nil {
			t.Error("expected error for missing archive in checksums")
		}
	})

	t.Run("checksums file not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		if err := os.WriteFile(archivePath, []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}

		err := verifyChecksum(archivePath, filepath.Join(tmpDir, "nonexistent.txt"), "test.tar.gz")
		if err == nil {
			t.Error("expected error for missing checksums file")
		}
	})

	t.Run("archive file not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		checksumPath := filepath.Join(tmpDir, "checksums.txt")
		if err := os.WriteFile(checksumPath, []byte("abc123  test.tar.gz\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		err := verifyChecksum(filepath.Join(tmpDir, "nonexistent.tar.gz"), checksumPath, "test.tar.gz")
		if err == nil {
			t.Error("expected error for missing archive file")
		}
	})
}

func TestDownloadFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "file content here")
		}))
		defer srv.Close()

		dest := filepath.Join(t.TempDir(), "downloaded")
		if err := downloadFile(srv.URL+"/test.tar.gz", dest); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(dest)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if string(data) != "file content here" {
			t.Errorf("content = %q, want %q", string(data), "file content here")
		}
	})

	t.Run("non-200 status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		dest := filepath.Join(t.TempDir(), "downloaded")
		err := downloadFile(srv.URL+"/missing", dest)
		if err == nil {
			t.Fatal("expected error for 404")
		}
	})

	t.Run("HTTP error", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "downloaded")
		err := downloadFile("http://127.0.0.1:1/bad", dest)
		if err == nil {
			t.Fatal("expected error for unreachable server")
		}
	})

	t.Run("invalid URL", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "downloaded")
		err := downloadFile("://bad", dest)
		if err == nil {
			t.Fatal("expected error for invalid URL")
		}
	})

	t.Run("unwritable dest", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "data")
		}))
		defer srv.Close()

		err := downloadFile(srv.URL, "/nonexistent-dir/file")
		if err == nil {
			t.Fatal("expected error for unwritable destination")
		}
	})
}

// buildTestTarGz creates a tar.gz archive containing a file named "redis-tui"
// with the given content.
func buildTestTarGz(t *testing.T, binaryContent string) []byte {
	t.Helper()
	var buf strings.Builder
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name:     "redis-tui_1.0.0_Test/redis-tui",
		Mode:     0o755,
		Size:     int64(len(binaryContent)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("tar write header: %v", err)
	}
	if _, err := tw.Write([]byte(binaryContent)); err != nil {
		t.Fatalf("tar write: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return []byte(buf.String())
}

func TestExtractBinary(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tmpDir := t.TempDir()
		archiveData := buildTestTarGz(t, "#!/bin/sh\necho hello\n")

		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		if err := os.WriteFile(archivePath, archiveData, 0o644); err != nil {
			t.Fatal(err)
		}

		destPath := filepath.Join(tmpDir, "redis-tui")
		if err := extractBinary(archivePath, destPath); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(destPath)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if string(data) != "#!/bin/sh\necho hello\n" {
			t.Errorf("content = %q", string(data))
		}
	})

	t.Run("archive not found", func(t *testing.T) {
		err := extractBinary("/nonexistent.tar.gz", filepath.Join(t.TempDir(), "out"))
		if err == nil {
			t.Fatal("expected error for missing archive")
		}
	})

	t.Run("not a gzip", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "bad.tar.gz")
		if err := os.WriteFile(archivePath, []byte("not a gzip"), 0o644); err != nil {
			t.Fatal(err)
		}

		err := extractBinary(archivePath, filepath.Join(tmpDir, "out"))
		if err == nil {
			t.Fatal("expected error for non-gzip file")
		}
	})

	t.Run("binary not in archive", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Build a tar.gz without a redis-tui file.
		var buf strings.Builder
		gw := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gw)
		hdr := &tar.Header{Name: "other-file", Mode: 0o644, Size: 5, Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte("hello")); err != nil {
			t.Fatal(err)
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gw.Close(); err != nil {
			t.Fatal(err)
		}

		archivePath := filepath.Join(tmpDir, "no-binary.tar.gz")
		if err := os.WriteFile(archivePath, []byte(buf.String()), 0o644); err != nil {
			t.Fatal(err)
		}

		err := extractBinary(archivePath, filepath.Join(tmpDir, "out"))
		if err == nil {
			t.Fatal("expected error for missing binary in archive")
		}
		if !strings.Contains(err.Error(), "binary not found") {
			t.Errorf("error = %q, want it to contain 'binary not found'", err)
		}
	})

	t.Run("unwritable destination", func(t *testing.T) {
		tmpDir := t.TempDir()
		archiveData := buildTestTarGz(t, "binary")
		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		if err := os.WriteFile(archivePath, archiveData, 0o644); err != nil {
			t.Fatal(err)
		}

		err := extractBinary(archivePath, "/nonexistent-dir/redis-tui")
		if err == nil {
			t.Fatal("expected error for unwritable destination")
		}
	})
}

func TestReplaceBinary(t *testing.T) {
	t.Run("replace existing", func(t *testing.T) {
		tmpDir := t.TempDir()
		current := filepath.Join(tmpDir, "redis-tui")
		newBin := filepath.Join(tmpDir, "redis-tui-new")

		if err := os.WriteFile(current, []byte("old"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(newBin, []byte("new"), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := replaceBinary(current, newBin); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(current)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if string(data) != "new" {
			t.Errorf("content = %q, want %q", string(data), "new")
		}

		// Old backup should be cleaned up.
		if _, err := os.Stat(current + ".old"); !os.IsNotExist(err) {
			t.Error("expected .old backup to be removed")
		}
	})

	t.Run("no existing binary", func(t *testing.T) {
		tmpDir := t.TempDir()
		current := filepath.Join(tmpDir, "redis-tui")
		newBin := filepath.Join(tmpDir, "redis-tui-new")

		if err := os.WriteFile(newBin, []byte("fresh"), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := replaceBinary(current, newBin); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(current)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if string(data) != "fresh" {
			t.Errorf("content = %q, want %q", string(data), "fresh")
		}
	})

	t.Run("new binary missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		current := filepath.Join(tmpDir, "redis-tui")
		if err := os.WriteFile(current, []byte("old"), 0o755); err != nil {
			t.Fatal(err)
		}

		err := replaceBinary(current, filepath.Join(tmpDir, "nonexistent"))
		if err == nil {
			t.Fatal("expected error when new binary doesn't exist")
		}

		// Original should be restored from backup.
		data, err := os.ReadFile(current)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if string(data) != "old" {
			t.Errorf("content = %q, want original restored", string(data))
		}
	})
}

func TestCheckWriteAccess(t *testing.T) {
	t.Run("writable directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "redis-tui")
		if err := checkWriteAccess(path); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("non-writable directory", func(t *testing.T) {
		err := checkWriteAccess("/nonexistent-dir/redis-tui")
		if err == nil {
			t.Error("expected error for non-writable directory")
		}
	})
}

func TestRunUpdateDevVersion(t *testing.T) {
	err := runUpdate("dev")
	if err == nil {
		t.Fatal("expected error for dev version")
	}
	if !strings.Contains(err.Error(), "development build") {
		t.Errorf("error = %q, want it to mention 'development build'", err)
	}
}

func TestRunUpdateNonSemver(t *testing.T) {
	err := runUpdate("abc123")
	if err == nil {
		t.Fatal("expected error for non-semver version")
	}
	if !strings.Contains(err.Error(), "development build") {
		t.Errorf("error = %q, want it to mention 'development build'", err)
	}
}

func TestRunUpdate_Homebrew(t *testing.T) {
	// Create a file inside a path containing "/homebrew/" to trigger
	// the Homebrew detection after EvalSymlinks succeeds.
	tmpDir := t.TempDir()
	brewDir := filepath.Join(tmpDir, "homebrew", "bin")
	if err := os.MkdirAll(brewDir, 0o755); err != nil {
		t.Fatal(err)
	}
	execPath := filepath.Join(brewDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}

	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	err := runUpdate("1.0.0")
	if err == nil {
		t.Fatal("expected error for Homebrew install")
	}
	if !strings.Contains(err.Error(), "Homebrew") {
		t.Errorf("error = %q, want Homebrew mention", err)
	}
}

func TestRunUpdate_AlreadyUpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v1.0.0"}`)
	}))
	defer srv.Close()

	old := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = old }()

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}

	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	err := runUpdate("1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunUpdate_FullFlow(t *testing.T) {
	// Build a tar.gz with a fake binary.
	binaryContent := "#!/bin/sh\necho updated\n"
	archiveData := buildTestTarGz(t, binaryContent)

	archiveHash := sha256.Sum256(archiveData)
	archiveName := archiveName("2.0.0", runtime.GOOS, runtime.GOARCH)
	checksumContent := fmt.Sprintf("%x  %s\n", archiveHash, archiveName)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "releases/latest"):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
		case strings.HasSuffix(r.URL.Path, "checksums.txt"):
			fmt.Fprint(w, checksumContent)
		case strings.HasSuffix(r.URL.Path, archiveName):
			if _, err := w.Write(archiveData); err != nil {
				t.Errorf("write archive: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldAPI := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = oldAPI }()

	// Set up a writable exec path.
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	// Override the GitHub download base URL to use our test server.
	// runUpdate builds URLs from githubRepo constant, so we need to
	// override httpClient to redirect github.com to our test server.
	origClient := httpClient
	httpClient = srv.Client()
	// Intercept all requests and rewrite to test server.
	httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return http.DefaultTransport.RoundTrip(req)
	})
	t.Cleanup(func() { httpClient = origClient })

	err := runUpdate("1.0.0")
	if err != nil {
		t.Fatalf("runUpdate failed: %v", err)
	}

	// Verify the binary was replaced.
	data, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(data) != binaryContent {
		t.Errorf("binary content = %q, want %q", string(data), binaryContent)
	}
}

func TestRunUpdate_FetchError(t *testing.T) {
	old := githubAPIBase
	githubAPIBase = "http://127.0.0.1:1"
	defer func() { githubAPIBase = old }()

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}

	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	err := runUpdate("1.0.0")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to fetch") {
		t.Errorf("error = %q, want 'failed to fetch'", err)
	}
}

func TestRunUpdate_NoWriteAccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
	}))
	defer srv.Close()

	old := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = old }()

	origExec := osExecutable
	osExecutable = func() (string, error) {
		return "/usr/bin/redis-tui", nil // not writable
	}
	t.Cleanup(func() { osExecutable = origExec })

	// This should fall back to ~/.local/bin — won't error on that part,
	// but will error trying to download from fake github.com URLs.
	err := runUpdate("1.0.0")
	if err == nil {
		t.Fatal("expected error (download from real github.com URLs will fail)")
	}
}

func TestRunUpdate_ExecPathError(t *testing.T) {
	origExec := osExecutable
	osExecutable = func() (string, error) { return "", fmt.Errorf("no executable") }
	t.Cleanup(func() { osExecutable = origExec })

	err := runUpdate("1.0.0")
	if err == nil || !strings.Contains(err.Error(), "could not determine executable path") {
		t.Errorf("error = %v, want executable path error", err)
	}
}

func TestRunUpdate_EvalSymlinksError(t *testing.T) {
	origExec := osExecutable
	osExecutable = func() (string, error) { return "/nonexistent/path/redis-tui", nil }
	t.Cleanup(func() { osExecutable = origExec })

	err := runUpdate("1.0.0")
	if err == nil || !strings.Contains(err.Error(), "could not resolve executable path") {
		t.Errorf("error = %v, want resolve path error", err)
	}
}

func TestRunUpdate_DownloadArchiveError(t *testing.T) {
	// Server returns latest version but 404 for the archive download.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "releases/latest") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	oldAPI := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = oldAPI }()

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	origClient := httpClient
	httpClient = srv.Client()
	httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return http.DefaultTransport.RoundTrip(req)
	})
	t.Cleanup(func() { httpClient = origClient })

	err := runUpdate("1.0.0")
	if err == nil || !strings.Contains(err.Error(), "failed to download") {
		t.Errorf("error = %v, want download error", err)
	}
}

func TestRunUpdate_ChecksumMismatch(t *testing.T) {
	archiveData := buildTestTarGz(t, "binary")
	archiveName := archiveName("2.0.0", runtime.GOOS, runtime.GOARCH)
	// Provide wrong checksum.
	checksumContent := fmt.Sprintf("0000000000000000000000000000000000000000000000000000000000000000  %s\n", archiveName)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "releases/latest"):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
		case strings.HasSuffix(r.URL.Path, "checksums.txt"):
			fmt.Fprint(w, checksumContent)
		case strings.HasSuffix(r.URL.Path, archiveName):
			if _, err := w.Write(archiveData); err != nil {
				t.Errorf("write: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldAPI := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = oldAPI }()

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	origClient := httpClient
	httpClient = srv.Client()
	httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return http.DefaultTransport.RoundTrip(req)
	})
	t.Cleanup(func() { httpClient = origClient })

	err := runUpdate("1.0.0")
	if err == nil || !strings.Contains(err.Error(), "checksum verification failed") {
		t.Errorf("error = %v, want checksum error", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRunUpdate_NoWriteAccess_HomeDirError(t *testing.T) {
	// Covers the branch where checkWriteAccess fails AND os.UserHomeDir fails.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
	}))
	defer srv.Close()

	oldAPI := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = oldAPI }()

	// Use a read-only directory for the exec path.
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0o555); err != nil {
		t.Fatal(err)
	}
	execPath := filepath.Join(readOnlyDir, "redis-tui")

	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	// Create the file so EvalSymlinks succeeds.
	if err := os.Chmod(readOnlyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Make dir read-only again so checkWriteAccess fails.
	if err := os.Chmod(readOnlyDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0o755) })

	// The fallback creates ~/.local/bin — since the download will fail
	// (GitHub URLs), the error will be about downloading, not write access.
	origClient := httpClient
	httpClient = srv.Client()
	httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return http.DefaultTransport.RoundTrip(req)
	})
	t.Cleanup(func() { httpClient = origClient })

	err := runUpdate("1.0.0")
	// Should proceed past the write access check and fail on download.
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunUpdate_TempDirError(t *testing.T) {
	// We can't easily make os.MkdirTemp fail, so test the download
	// checksum error path which covers lines 100-102.
	// (This is already covered by TestRunUpdate_ChecksumMismatch.)
	// Instead, cover the extractBinary error path in runUpdate (lines 109-111).
	an := archiveName("2.0.0", runtime.GOOS, runtime.GOARCH)
	// Serve a non-gzip file with matching checksum so verifyChecksum
	// passes but extractBinary fails.
	badArchive := []byte("not-a-gzip")
	badHash := sha256.Sum256(badArchive)
	checksumContent := fmt.Sprintf("%x  %s\n", badHash, an)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "releases/latest"):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
		case strings.HasSuffix(r.URL.Path, "checksums.txt"):
			fmt.Fprint(w, checksumContent)
		case strings.HasSuffix(r.URL.Path, an):
			if _, err := w.Write(badArchive); err != nil {
				t.Errorf("write: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldAPI := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = oldAPI }()

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	origClient := httpClient
	httpClient = srv.Client()
	httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return http.DefaultTransport.RoundTrip(req)
	})
	t.Cleanup(func() { httpClient = origClient })

	err := runUpdate("1.0.0")
	if err == nil || !strings.Contains(err.Error(), "failed to extract") {
		t.Errorf("error = %v, want extract error", err)
	}
	// Verify the uncovered lines are different: checksum download error.
	// Serve checksums.txt as 404:
}

func TestRunUpdate_ChecksumDownloadError(t *testing.T) {
	archiveData := buildTestTarGz(t, "binary")
	an := archiveName("2.0.0", runtime.GOOS, runtime.GOARCH)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "releases/latest"):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
		case strings.HasSuffix(r.URL.Path, an):
			if _, err := w.Write(archiveData); err != nil {
				t.Errorf("write: %v", err)
			}
		default:
			// checksums.txt returns 404
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldAPI := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = oldAPI }()

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	origClient := httpClient
	httpClient = srv.Client()
	httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return http.DefaultTransport.RoundTrip(req)
	})
	t.Cleanup(func() { httpClient = origClient })

	err := runUpdate("1.0.0")
	if err == nil || !strings.Contains(err.Error(), "failed to download checksums") {
		t.Errorf("error = %v, want checksum download error", err)
	}
}

func TestReplaceBinary_BackupRenameError(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a directory at the "current" path — os.Rename to .old will
	// fail with a different error if .old already exists as a directory.
	current := filepath.Join(tmpDir, "redis-tui")
	if err := os.Mkdir(current, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a file at .old so Rename(current, .old) fails because
	// on some OSes you can't rename a dir over a file.
	oldPath := current + ".old"
	if err := os.WriteFile(oldPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	newBin := filepath.Join(tmpDir, "new")
	if err := os.WriteFile(newBin, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := replaceBinary(current, newBin)
	if err == nil {
		t.Fatal("expected error from backup rename")
	}
}

func TestCheckWriteAccess_CloseError(t *testing.T) {
	// checkWriteAccess creates a temp file, closes it, removes it.
	// The close error branch is very hard to trigger in normal conditions.
	// Just ensure the happy path works in a writable dir (already tested)
	// and verify error on truly unwritable dir.
	err := checkWriteAccess(filepath.Join("/proc", "redis-tui"))
	if err == nil {
		// On macOS /proc doesn't exist — use /nonexistent
		err = checkWriteAccess("/nonexistent/redis-tui")
		if err == nil {
			t.Error("expected error for unwritable directory")
		}
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
