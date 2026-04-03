package redis

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
)

// ---------------------------------------------------------------------------
// GetValue
// ---------------------------------------------------------------------------

func TestGetValue(t *testing.T) {
	client, mr := setupTestClient(t)

	// Seed each key type via miniredis helpers.
	mr.Set("str", "hello")
	mr.RPush("list", "a", "b")
	mr.SAdd("set", "x", "y")
	mr.ZAdd("zset", 1.5, "m1")
	mr.HSet("hash", "f1", "v1")

	// Streams are not natively supported by miniredis helpers,
	// so we use the client wrapper.
	streamID, err := client.XAdd("stream", map[string]any{"field": "value"})
	if err != nil {
		t.Fatalf("XAdd failed: %v", err)
	}
	if streamID == "" {
		t.Fatal("expected non-empty stream ID")
	}

	tests := []struct {
		name     string
		key      string
		wantType types.KeyType
		check    func(t *testing.T, v types.RedisValue)
	}{
		{
			name:     "string key",
			key:      "str",
			wantType: types.KeyTypeString,
			check: func(t *testing.T, v types.RedisValue) {
				if v.StringValue != "hello" {
					t.Errorf("StringValue = %q, want %q", v.StringValue, "hello")
				}
			},
		},
		{
			name:     "list key",
			key:      "list",
			wantType: types.KeyTypeList,
			check: func(t *testing.T, v types.RedisValue) {
				if len(v.ListValue) != 2 {
					t.Fatalf("ListValue length = %d, want 2", len(v.ListValue))
				}
				if v.ListValue[0] != "a" || v.ListValue[1] != "b" {
					t.Errorf("ListValue = %v, want [a b]", v.ListValue)
				}
			},
		},
		{
			name:     "set key",
			key:      "set",
			wantType: types.KeyTypeSet,
			check: func(t *testing.T, v types.RedisValue) {
				if len(v.SetValue) != 2 {
					t.Fatalf("SetValue length = %d, want 2", len(v.SetValue))
				}
				sort.Strings(v.SetValue)
				if v.SetValue[0] != "x" || v.SetValue[1] != "y" {
					t.Errorf("SetValue = %v, want [x y]", v.SetValue)
				}
			},
		},
		{
			name:     "zset key",
			key:      "zset",
			wantType: types.KeyTypeZSet,
			check: func(t *testing.T, v types.RedisValue) {
				if len(v.ZSetValue) != 1 {
					t.Fatalf("ZSetValue length = %d, want 1", len(v.ZSetValue))
				}
				if v.ZSetValue[0].Member != "m1" {
					t.Errorf("ZSetValue[0].Member = %q, want %q", v.ZSetValue[0].Member, "m1")
				}
				if v.ZSetValue[0].Score != 1.5 {
					t.Errorf("ZSetValue[0].Score = %f, want 1.5", v.ZSetValue[0].Score)
				}
			},
		},
		{
			name:     "hash key",
			key:      "hash",
			wantType: types.KeyTypeHash,
			check: func(t *testing.T, v types.RedisValue) {
				if v.HashValue["f1"] != "v1" {
					t.Errorf("HashValue[f1] = %q, want %q", v.HashValue["f1"], "v1")
				}
			},
		},
		{
			name:     "stream key",
			key:      "stream",
			wantType: types.KeyTypeStream,
			check: func(t *testing.T, v types.RedisValue) {
				if len(v.StreamValue) != 1 {
					t.Fatalf("StreamValue length = %d, want 1", len(v.StreamValue))
				}
				if v.StreamValue[0].Fields["field"] != "value" {
					t.Errorf("StreamValue[0].Fields[field] = %v, want %q", v.StreamValue[0].Fields["field"], "value")
				}
			},
		},
		{
			name:     "missing key returns none type",
			key:      "nonexistent",
			wantType: types.KeyType("none"),
			check:    func(t *testing.T, v types.RedisValue) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := client.GetValue(tt.key)
			if err != nil {
				t.Fatalf("GetValue(%q) error: %v", tt.key, err)
			}
			if v.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", v.Type, tt.wantType)
			}
			tt.check(t, v)
		})
	}
}

// ---------------------------------------------------------------------------
// DeleteKey
// ---------------------------------------------------------------------------

func TestDeleteKey(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("to-delete", "bye")

	if err := client.DeleteKey("to-delete"); err != nil {
		t.Fatalf("DeleteKey error: %v", err)
	}

	if mr.Exists("to-delete") {
		t.Error("key should no longer exist after DeleteKey")
	}
}

// ---------------------------------------------------------------------------
// BulkDelete
// ---------------------------------------------------------------------------

func TestBulkDelete(t *testing.T) {
	t.Run("basic pattern delete", func(t *testing.T) {
		client, mr := setupTestClient(t)

		for i := range 5 {
			mr.Set(fmt.Sprintf("bulk:%d", i), "val")
		}
		// Key that should NOT be deleted.
		mr.Set("other:1", "keep")

		deleted, err := client.BulkDelete("bulk:*")
		if err != nil {
			t.Fatalf("BulkDelete error: %v", err)
		}
		if deleted != 5 {
			t.Errorf("deleted = %d, want 5", deleted)
		}

		for i := range 5 {
			if mr.Exists(fmt.Sprintf("bulk:%d", i)) {
				t.Errorf("key bulk:%d should be deleted", i)
			}
		}
		if !mr.Exists("other:1") {
			t.Error("other:1 should still exist")
		}
	})

	t.Run("chunked delete with 250+ keys", func(t *testing.T) {
		client, mr := setupTestClient(t)

		const keyCount = 260
		for i := range keyCount {
			mr.Set(fmt.Sprintf("chunk:%d", i), "v")
		}

		deleted, err := client.BulkDelete("chunk:*")
		if err != nil {
			t.Fatalf("BulkDelete error: %v", err)
		}
		if deleted != keyCount {
			t.Errorf("deleted = %d, want %d", deleted, keyCount)
		}

		for i := range keyCount {
			if mr.Exists(fmt.Sprintf("chunk:%d", i)) {
				t.Errorf("key chunk:%d should be deleted", i)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// SetString
// ---------------------------------------------------------------------------

func TestSetString(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		value      string
		ttl        time.Duration
		wantValue  string
		wantTTL    bool
		ttlSeconds int
	}{
		{
			name:      "basic set without TTL",
			key:       "s1",
			value:     "v1",
			ttl:       0,
			wantValue: "v1",
			wantTTL:   false,
		},
		{
			name:       "set with TTL",
			key:        "s2",
			value:      "v2",
			ttl:        10 * time.Second,
			wantValue:  "v2",
			wantTTL:    true,
			ttlSeconds: 10,
		},
		{
			name:      "overwrite existing key",
			key:       "s1",
			value:     "overwritten",
			ttl:       0,
			wantValue: "overwritten",
			wantTTL:   false,
		},
	}

	client, mr := setupTestClient(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := client.SetString(tt.key, tt.value, tt.ttl); err != nil {
				t.Fatalf("SetString error: %v", err)
			}

			got, err := mr.Get(tt.key)
			if err != nil {
				t.Fatalf("miniredis Get error: %v", err)
			}
			if got != tt.wantValue {
				t.Errorf("value = %q, want %q", got, tt.wantValue)
			}

			if tt.wantTTL {
				mrTTL := mr.TTL(tt.key)
				if mrTTL != time.Duration(tt.ttlSeconds)*time.Second {
					t.Errorf("TTL = %v, want %v", mrTTL, time.Duration(tt.ttlSeconds)*time.Second)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SetTTL
// ---------------------------------------------------------------------------

func TestSetTTL(t *testing.T) {
	t.Run("set TTL on key", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("ttlkey", "val")
		if err := client.SetTTL("ttlkey", 30*time.Second); err != nil {
			t.Fatalf("SetTTL error: %v", err)
		}

		ttl := mr.TTL("ttlkey")
		if ttl != 30*time.Second {
			t.Errorf("TTL = %v, want 30s", ttl)
		}
	})

	t.Run("remove TTL (persist)", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("ttlkey2", "val")
		mr.SetTTL("ttlkey2", 60*time.Second)

		if err := client.SetTTL("ttlkey2", 0); err != nil {
			t.Fatalf("SetTTL(0) error: %v", err)
		}

		ttl := mr.TTL("ttlkey2")
		if ttl != 0 {
			t.Errorf("TTL = %v, want 0 (persistent)", ttl)
		}
	})
}

// ---------------------------------------------------------------------------
// BatchSetTTL
// ---------------------------------------------------------------------------

func TestBatchSetTTL(t *testing.T) {
	client, mr := setupTestClient(t)

	for i := range 5 {
		mr.Set(fmt.Sprintf("bttl:%d", i), "val")
	}

	count, err := client.BatchSetTTL("bttl:*", 20*time.Second)
	if err != nil {
		t.Fatalf("BatchSetTTL error: %v", err)
	}
	if count != 5 {
		t.Errorf("count = %d, want 5", count)
	}

	for i := range 5 {
		ttl := mr.TTL(fmt.Sprintf("bttl:%d", i))
		if ttl != 20*time.Second {
			t.Errorf("bttl:%d TTL = %v, want 20s", i, ttl)
		}
	}
}

// ---------------------------------------------------------------------------
// Rename
// ---------------------------------------------------------------------------

func TestRename(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("oldname", "data")

	if err := client.Rename("oldname", "newname"); err != nil {
		t.Fatalf("Rename error: %v", err)
	}

	if mr.Exists("oldname") {
		t.Error("old key should not exist after rename")
	}

	got, err := mr.Get("newname")
	if err != nil {
		t.Fatalf("miniredis Get(newname) error: %v", err)
	}
	if got != "data" {
		t.Errorf("newname value = %q, want %q", got, "data")
	}
}

// ---------------------------------------------------------------------------
// Copy
// ---------------------------------------------------------------------------

func TestCopy(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("src", "original")

	if err := client.Copy("src", "dst", false); err != nil {
		t.Fatalf("Copy error: %v", err)
	}

	srcVal, err := mr.Get("src")
	if err != nil {
		t.Fatalf("miniredis Get(src) error: %v", err)
	}
	dstVal, err := mr.Get("dst")
	if err != nil {
		t.Fatalf("miniredis Get(dst) error: %v", err)
	}

	if srcVal != "original" {
		t.Errorf("src value = %q, want %q", srcVal, "original")
	}
	if dstVal != "original" {
		t.Errorf("dst value = %q, want %q", dstVal, "original")
	}
}

// ---------------------------------------------------------------------------
// MemoryUsage
// ---------------------------------------------------------------------------

func TestMemoryUsage(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("memkey", "some-value-to-measure")

	mem, err := client.MemoryUsage("memkey")
	if err != nil {
		// miniredis may not support MEMORY USAGE; skip if unsupported.
		t.Skipf("MemoryUsage not supported by miniredis: %v", err)
	}
	if mem <= 0 {
		t.Errorf("MemoryUsage = %d, want > 0", mem)
	}
}
