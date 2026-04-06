package testutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestTempConfigPath(t *testing.T) {
	path := TempConfigPath(t)
	if path == "" {
		t.Fatal("TempConfigPath returned empty string")
	}
	if filepath.Base(path) != "config.json" {
		t.Errorf("expected filename config.json, got %q", filepath.Base(path))
	}
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("temp directory does not exist: %s", dir)
	}
}

func TestNewTestConfig(t *testing.T) {
	cfg := NewTestConfig(t)
	if cfg == nil {
		t.Fatal("NewTestConfig returned nil")
	}
	connections, err := cfg.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections failed: %v", err)
	}
	if len(connections) != 0 {
		t.Errorf("expected 0 connections, got %d", len(connections))
	}
}

func TestMustAddConnection(t *testing.T) {
	cfg := NewTestConfig(t)
	conn := MustAddConnection(t, cfg, "test", "localhost", 6379, "", 0)
	if conn.Name != "test" {
		t.Errorf("Name = %q, want %q", conn.Name, "test")
	}
	if conn.Host != "localhost" {
		t.Errorf("Host = %q, want %q", conn.Host, "localhost")
	}
	if conn.ID == 0 {
		t.Error("ID should not be 0")
	}
}

func TestAssertConnectionExists(t *testing.T) {
	cfg := NewTestConfig(t)
	conn := MustAddConnection(t, cfg, "test", "localhost", 6379, "", 0)
	got := AssertConnectionExists(t, cfg, conn.ID)
	if got.Name != "test" {
		t.Errorf("Name = %q, want %q", got.Name, "test")
	}
}

func TestAssertConnectionNotExists(t *testing.T) {
	cfg := NewTestConfig(t)
	AssertConnectionNotExists(t, cfg, 999)
}

func TestAssertEqual(t *testing.T) {
	// Should not fail
	AssertEqual(t, 42, 42, "integers")
	AssertEqual(t, "hello", "hello", "strings")
	AssertEqual(t, true, true, "booleans")
}

func TestAssertNoError(t *testing.T) {
	AssertNoError(t, nil, "nil error")
}

func TestAssertError(t *testing.T) {
	AssertError(t, errors.New("something broke"), "expected error")
}

func TestAssertSliceLen(t *testing.T) {
	AssertSliceLen(t, []int{1, 2, 3}, 3, "int slice")
	AssertSliceLen(t, []string{}, 0, "empty slice")
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "testfile.txt")

	if FileExists(path) {
		t.Error("FileExists should return false for non-existent file")
	}

	err := os.WriteFile(path, []byte("test"), 0600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if !FileExists(path) {
		t.Error("FileExists should return true for existing file")
	}
}

func TestSampleConnection(t *testing.T) {
	conn := SampleConnection()
	if conn.ID != 1 {
		t.Errorf("ID = %d, want 1", conn.ID)
	}
	if conn.Host != "localhost" {
		t.Errorf("Host = %q, want %q", conn.Host, "localhost")
	}
	if conn.Port != 6379 {
		t.Errorf("Port = %d, want 6379", conn.Port)
	}
}

func TestSampleRedisKey(t *testing.T) {
	key := SampleRedisKey("user:1", types.KeyTypeHash)
	if key.Key != "user:1" {
		t.Errorf("Key = %q, want %q", key.Key, "user:1")
	}
	if key.Type != types.KeyTypeHash {
		t.Errorf("Type = %q, want %q", key.Type, types.KeyTypeHash)
	}
	if key.TTL != -1 {
		t.Errorf("TTL = %d, want -1", key.TTL)
	}
}

func TestSampleFavorite(t *testing.T) {
	fav := SampleFavorite(1, "cache:main")
	if fav.ConnectionID != 1 {
		t.Errorf("ConnectionID = %d, want 1", fav.ConnectionID)
	}
	if fav.Key != "cache:main" {
		t.Errorf("Key = %q, want %q", fav.Key, "cache:main")
	}
	if fav.Label != "Test Favorite" {
		t.Errorf("Label = %q, want %q", fav.Label, "Test Favorite")
	}
}
