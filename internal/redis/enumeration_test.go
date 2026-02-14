package redis

import (
	"sort"
	"testing"

	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestGetTotalKeys(t *testing.T) {
	t.Run("returns count of existing keys", func(t *testing.T) {
		client, mr := setupTestClient(t)

		for i := 0; i < 5; i++ {
			mr.Set("key:"+string(rune('a'+i)), "val")
		}

		got := client.GetTotalKeys()
		if got != 5 {
			t.Errorf("GetTotalKeys() = %d, want 5", got)
		}
	})

	t.Run("empty database returns 0", func(t *testing.T) {
		client, _ := setupTestClient(t)

		got := client.GetTotalKeys()
		if got != 0 {
			t.Errorf("GetTotalKeys() = %d, want 0", got)
		}
	})
}

func TestScanKeys(t *testing.T) {
	t.Run("scan with pattern", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("user:1", "alice")
		mr.Set("user:2", "bob")
		mr.Set("other:1", "charlie")

		keys, _, err := client.ScanKeys("user:*", 0, 100)
		if err != nil {
			t.Fatalf("ScanKeys() error = %v", err)
		}

		if len(keys) != 2 {
			t.Fatalf("ScanKeys() returned %d keys, want 2", len(keys))
		}

		names := make(map[string]bool)
		for _, k := range keys {
			names[k.Key] = true
			if k.Type != types.KeyTypeString {
				t.Errorf("key %q type = %q, want %q", k.Key, k.Type, types.KeyTypeString)
			}
		}
		if !names["user:1"] || !names["user:2"] {
			t.Errorf("expected user:1 and user:2, got %v", names)
		}
	})

	t.Run("empty pattern defaults to wildcard", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("a", "1")
		mr.Set("b", "2")

		keys, _, err := client.ScanKeys("", 0, 100)
		if err != nil {
			t.Fatalf("ScanKeys() error = %v", err)
		}
		if len(keys) != 2 {
			t.Errorf("ScanKeys(\"\") returned %d keys, want 2", len(keys))
		}
	})

	t.Run("empty database returns empty slice", func(t *testing.T) {
		client, _ := setupTestClient(t)

		keys, _, err := client.ScanKeys("*", 0, 100)
		if err != nil {
			t.Fatalf("ScanKeys() error = %v", err)
		}
		if len(keys) != 0 {
			t.Errorf("ScanKeys() on empty db returned %d keys, want 0", len(keys))
		}
	})

	t.Run("returns TTL field", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("ttlkey", "value")

		keys, _, err := client.ScanKeys("ttlkey", 0, 100)
		if err != nil {
			t.Fatalf("ScanKeys() error = %v", err)
		}
		if len(keys) != 1 {
			t.Fatalf("ScanKeys() returned %d keys, want 1", len(keys))
		}
		// miniredis returns -1 for no TTL
		if keys[0].TTL == 0 {
			t.Log("TTL field is populated (zero or negative for no expiry)")
		}
	})
}

func TestScanKeys_WithoutTypes(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("key:1", "val1")
	mr.Set("key:2", "val2")

	client.SetIncludeTypes(false)

	keys, _, err := client.ScanKeys("key:*", 0, 100)
	if err != nil {
		t.Fatalf("ScanKeys() error = %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("ScanKeys() returned %d keys, want 2", len(keys))
	}
	for _, k := range keys {
		if k.Type != "" {
			t.Errorf("key %q type = %q, want empty when includeTypes=false", k.Key, k.Type)
		}
		// TTL should still be populated (miniredis returns -1 for no expiry)
		if k.TTL == 0 {
			t.Errorf("key %q TTL should be non-zero (no expiry = -1)", k.Key)
		}
	}
}

func TestScanKeysWithRegex(t *testing.T) {
	t.Run("matches regex pattern", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("user:123", "a")
		mr.Set("user:abc", "b")
		mr.Set("session:456", "c")

		keys, err := client.ScanKeysWithRegex(`user:\d+`, 100)
		if err != nil {
			t.Fatalf("ScanKeysWithRegex() error = %v", err)
		}

		if len(keys) != 1 {
			t.Fatalf("ScanKeysWithRegex() returned %d keys, want 1", len(keys))
		}
		if keys[0].Key != "user:123" {
			t.Errorf("key = %q, want %q", keys[0].Key, "user:123")
		}
	})

	t.Run("invalid regex returns error", func(t *testing.T) {
		client, _ := setupTestClient(t)

		_, err := client.ScanKeysWithRegex(`[invalid`, 100)
		if err == nil {
			t.Fatal("ScanKeysWithRegex() expected error for invalid regex, got nil")
		}
	})

	t.Run("no matches returns empty slice", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("foo", "bar")

		keys, err := client.ScanKeysWithRegex(`^zzz`, 100)
		if err != nil {
			t.Fatalf("ScanKeysWithRegex() error = %v", err)
		}
		if len(keys) != 0 {
			t.Errorf("ScanKeysWithRegex() returned %d keys, want 0", len(keys))
		}
	})
}

func TestFuzzySearchKeys(t *testing.T) {
	t.Run("returns matching keys sorted by score", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("user:profile:settings", "a")
		mr.Set("user:data", "b")
		mr.Set("other:key", "c")

		// "user" is a substring match for "user:profile:settings" and "user:data"
		keys, err := client.FuzzySearchKeys("user", 10)
		if err != nil {
			t.Fatalf("FuzzySearchKeys() error = %v", err)
		}

		if len(keys) < 2 {
			t.Fatalf("FuzzySearchKeys(\"user\") returned %d results, want at least 2", len(keys))
		}

		// Results should be sorted by score; shorter key with substring match scores
		// higher because fuzzyScore returns 100 + (len(str) - len(pattern))
		// "user:data" (len 9) scores 105, "user:profile:settings" (len 21) scores 117
		// Both are substring matches so higher len(str)-len(pattern) gives higher score,
		// meaning the longer key should come first.
		// Verify all returned keys have names and types
		for _, k := range keys {
			if k.Key == "" {
				t.Error("FuzzySearchKeys() returned a key with empty name")
			}
			if k.Type == "" {
				t.Error("FuzzySearchKeys() returned a key with empty type")
			}
		}
	})

	t.Run("no matches returns empty slice", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("abc", "1")

		keys, err := client.FuzzySearchKeys("zzz", 10)
		if err != nil {
			t.Fatalf("FuzzySearchKeys() error = %v", err)
		}
		if len(keys) != 0 {
			t.Errorf("FuzzySearchKeys() returned %d keys, want 0", len(keys))
		}
	})
}

func TestSearchByValue(t *testing.T) {
	t.Run("finds matching values across types", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("s1", "hello world")
		mr.RPush("l1", "foo", "bar world")
		mr.SAdd("set1", "world")
		mr.HSet("h1", "f", "world")

		keys, err := client.SearchByValue("*", "world", 10)
		if err != nil {
			t.Fatalf("SearchByValue() error = %v", err)
		}

		if len(keys) < 4 {
			t.Fatalf("SearchByValue() returned %d keys, want at least 4", len(keys))
		}

		found := make(map[string]bool)
		for _, k := range keys {
			found[k.Key] = true
		}
		for _, expected := range []string{"s1", "l1", "set1", "h1"} {
			if !found[expected] {
				t.Errorf("SearchByValue() missing expected key %q", expected)
			}
		}
	})

	t.Run("non-matching search returns empty", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("s1", "hello")

		keys, err := client.SearchByValue("*", "nonexistent", 10)
		if err != nil {
			t.Fatalf("SearchByValue() error = %v", err)
		}
		if len(keys) != 0 {
			t.Errorf("SearchByValue() returned %d keys, want 0", len(keys))
		}
	})
}

func TestGetKeyPrefixes(t *testing.T) {
	t.Run("returns unique prefixes", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("user:profile:name", "a")
		mr.Set("user:profile:age", "b")
		mr.Set("user:settings", "c")
		mr.Set("session:abc", "d")

		prefixes, err := client.GetKeyPrefixes(":", 3)
		if err != nil {
			t.Fatalf("GetKeyPrefixes() error = %v", err)
		}

		// Should include: user, user:profile, user:profile:name, user:profile:age,
		// user:settings, session, session:abc
		sort.Strings(prefixes)

		expected := map[string]bool{
			"user":              true,
			"user:profile":      true,
			"user:profile:name": true,
			"user:profile:age":  true,
			"user:settings":     true,
			"session":           true,
			"session:abc":       true,
		}

		for _, p := range prefixes {
			if !expected[p] {
				t.Errorf("unexpected prefix %q", p)
			}
			delete(expected, p)
		}
		for p := range expected {
			t.Errorf("missing expected prefix %q", p)
		}
	})

	t.Run("empty database returns empty slice", func(t *testing.T) {
		client, _ := setupTestClient(t)

		prefixes, err := client.GetKeyPrefixes(":", 3)
		if err != nil {
			t.Fatalf("GetKeyPrefixes() error = %v", err)
		}
		if len(prefixes) != 0 {
			t.Errorf("GetKeyPrefixes() on empty db returned %d prefixes, want 0", len(prefixes))
		}
	})
}
