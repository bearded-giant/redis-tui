package redis

import (
	"sort"
	"testing"

	goredis "github.com/redis/go-redis/v9"
)

// ---------------------------------------------------------------------------
// RPush
// ---------------------------------------------------------------------------

func TestRPush(t *testing.T) {
	client, mr := setupTestClient(t)

	if err := client.RPush("mylist", "a", "b", "c"); err != nil {
		t.Fatalf("RPush error: %v", err)
	}

	got, err := mr.List("mylist")
	if err != nil {
		t.Fatalf("miniredis List error: %v", err)
	}
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("list length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("list[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// ---------------------------------------------------------------------------
// SAdd
// ---------------------------------------------------------------------------

func TestSAdd(t *testing.T) {
	client, mr := setupTestClient(t)

	if err := client.SAdd("myset", "x", "y", "z"); err != nil {
		t.Fatalf("SAdd error: %v", err)
	}

	got, err := mr.Members("myset")
	if err != nil {
		t.Fatalf("miniredis Members error: %v", err)
	}
	sort.Strings(got)
	want := []string{"x", "y", "z"}
	if len(got) != len(want) {
		t.Fatalf("set size = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("member[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// ---------------------------------------------------------------------------
// ZAdd
// ---------------------------------------------------------------------------

func TestZAdd(t *testing.T) {
	client, mr := setupTestClient(t)

	if err := client.ZAdd("myzset", 2.5, "member1"); err != nil {
		t.Fatalf("ZAdd error: %v", err)
	}

	score, err := mr.ZScore("myzset", "member1")
	if err != nil {
		t.Fatalf("miniredis ZScore error: %v", err)
	}
	if score != 2.5 {
		t.Errorf("score = %f, want 2.5", score)
	}
}

// ---------------------------------------------------------------------------
// HSet
// ---------------------------------------------------------------------------

func TestHSet(t *testing.T) {
	client, mr := setupTestClient(t)

	if err := client.HSet("myhash", "field1", "value1"); err != nil {
		t.Fatalf("HSet error: %v", err)
	}

	got := mr.HGet("myhash", "field1")
	if got != "value1" {
		t.Errorf("HGet = %q, want %q", got, "value1")
	}
}

// ---------------------------------------------------------------------------
// XAdd
// ---------------------------------------------------------------------------

func TestXAdd(t *testing.T) {
	client, _ := setupTestClient(t)

	id, err := client.XAdd("mystream", map[string]any{
		"key1": "val1",
		"key2": "val2",
	})
	if err != nil {
		t.Fatalf("XAdd error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty stream entry ID")
	}
}

// ---------------------------------------------------------------------------
// LSet
// ---------------------------------------------------------------------------

func TestLSet(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.RPush("ls", "a", "b", "c")

	if err := client.LSet("ls", 1, "B"); err != nil {
		t.Fatalf("LSet error: %v", err)
	}

	got, err := mr.List("ls")
	if err != nil {
		t.Fatalf("miniredis List error: %v", err)
	}
	if got[1] != "B" {
		t.Errorf("list[1] = %q, want %q", got[1], "B")
	}
}

// ---------------------------------------------------------------------------
// LRem
// ---------------------------------------------------------------------------

func TestLRem(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.RPush("lr", "a", "b", "a", "c", "a")

	// Remove up to 2 occurrences of "a" from head.
	if err := client.LRem("lr", 2, "a"); err != nil {
		t.Fatalf("LRem error: %v", err)
	}

	got, err := mr.List("lr")
	if err != nil {
		t.Fatalf("miniredis List error: %v", err)
	}
	// After removing first 2 "a"s: ["b", "c", "a"]
	want := []string{"b", "c", "a"}
	if len(got) != len(want) {
		t.Fatalf("list length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("list[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// ---------------------------------------------------------------------------
// SRem
// ---------------------------------------------------------------------------

func TestSRem(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.SAdd("sr", "a", "b", "c")

	if err := client.SRem("sr", "b"); err != nil {
		t.Fatalf("SRem error: %v", err)
	}

	got, err := mr.Members("sr")
	if err != nil {
		t.Fatalf("miniredis Members error: %v", err)
	}
	sort.Strings(got)
	want := []string{"a", "c"}
	if len(got) != len(want) {
		t.Fatalf("set size = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("member[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// ---------------------------------------------------------------------------
// ZRem
// ---------------------------------------------------------------------------

func TestZRem(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.ZAdd("zr", 1, "a")
	mr.ZAdd("zr", 2, "b")
	mr.ZAdd("zr", 3, "c")

	if err := client.ZRem("zr", "b"); err != nil {
		t.Fatalf("ZRem error: %v", err)
	}

	members, err := mr.ZMembers("zr")
	if err != nil {
		t.Fatalf("miniredis ZMembers error: %v", err)
	}
	sort.Strings(members)
	want := []string{"a", "c"}
	if len(members) != len(want) {
		t.Fatalf("zset size = %d, want %d", len(members), len(want))
	}
	for i := range want {
		if members[i] != want[i] {
			t.Errorf("member[%d] = %q, want %q", i, members[i], want[i])
		}
	}
}

// ---------------------------------------------------------------------------
// HDel
// ---------------------------------------------------------------------------

func TestHDel(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.HSet("hd", "f1", "v1")
	mr.HSet("hd", "f2", "v2")
	mr.HSet("hd", "f3", "v3")

	if err := client.HDel("hd", "f2"); err != nil {
		t.Fatalf("HDel error: %v", err)
	}

	keys, err := mr.HKeys("hd")
	if err != nil {
		t.Fatalf("miniredis HKeys error: %v", err)
	}
	sort.Strings(keys)
	want := []string{"f1", "f3"}
	if len(keys) != len(want) {
		t.Fatalf("hash field count = %d, want %d", len(keys), len(want))
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Errorf("field[%d] = %q, want %q", i, keys[i], want[i])
		}
	}
}

// ---------------------------------------------------------------------------
// XDel
// ---------------------------------------------------------------------------

func TestXDel(t *testing.T) {
	client, _ := setupTestClient(t)

	id1, err := client.XAdd("xd", map[string]any{"k": "1"})
	if err != nil {
		t.Fatalf("XAdd error: %v", err)
	}
	id2, err := client.XAdd("xd", map[string]any{"k": "2"})
	if err != nil {
		t.Fatalf("XAdd error: %v", err)
	}

	if err := client.XDel("xd", id1); err != nil {
		t.Fatalf("XDel error: %v", err)
	}

	// Verify the remaining entry via GetValue.
	v, err := client.GetValue("xd")
	if err != nil {
		t.Fatalf("GetValue error: %v", err)
	}
	if len(v.StreamValue) != 1 {
		t.Fatalf("stream length = %d, want 1", len(v.StreamValue))
	}
	if v.StreamValue[0].ID != id2 {
		t.Errorf("remaining entry ID = %q, want %q", v.StreamValue[0].ID, id2)
	}
}

// ---------------------------------------------------------------------------
// ZAddBatch
// ---------------------------------------------------------------------------

func TestZAddBatch(t *testing.T) {
	client, mr := setupTestClient(t)

	members := []goredis.Z{
		{Score: 1.0, Member: "alpha"},
		{Score: 2.0, Member: "beta"},
		{Score: 3.0, Member: "gamma"},
	}
	if err := client.ZAddBatch("zbatch", members...); err != nil {
		t.Fatalf("ZAddBatch error: %v", err)
	}

	for _, m := range members {
		score, err := mr.ZScore("zbatch", m.Member.(string))
		if err != nil {
			t.Fatalf("ZScore(%q) error: %v", m.Member, err)
		}
		if score != m.Score {
			t.Errorf("ZScore(%q) = %f, want %f", m.Member, score, m.Score)
		}
	}
}

// ---------------------------------------------------------------------------
// HSetMap
// ---------------------------------------------------------------------------

func TestHSetMap(t *testing.T) {
	t.Run("set multiple fields", func(t *testing.T) {
		client, mr := setupTestClient(t)

		fields := map[string]string{
			"name": "alice",
			"age":  "30",
			"city": "paris",
		}
		if err := client.HSetMap("hmap", fields); err != nil {
			t.Fatalf("HSetMap error: %v", err)
		}

		for k, want := range fields {
			got := mr.HGet("hmap", k)
			if got != want {
				t.Errorf("HGet(%q) = %q, want %q", k, got, want)
			}
		}
	})

	t.Run("empty map returns nil", func(t *testing.T) {
		client, _ := setupTestClient(t)

		if err := client.HSetMap("hmap-empty", map[string]string{}); err != nil {
			t.Errorf("HSetMap with empty map should return nil, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// SetBit / GetBit / BitCount
// ---------------------------------------------------------------------------

func TestSetBit(t *testing.T) {
	t.Run("set bit at offset", func(t *testing.T) {
		client, _ := setupTestClient(t)

		if err := client.SetBit("bitmap", 7, 1); err != nil {
			t.Fatalf("SetBit error: %v", err)
		}

		val, err := client.GetBit("bitmap", 7)
		if err != nil {
			t.Fatalf("GetBit error: %v", err)
		}
		if val != 1 {
			t.Errorf("GetBit(7) = %d, want 1", val)
		}
	})

	t.Run("set bit to zero", func(t *testing.T) {
		client, _ := setupTestClient(t)

		if err := client.SetBit("bitmap2", 3, 1); err != nil {
			t.Fatalf("SetBit(3, 1) error: %v", err)
		}
		if err := client.SetBit("bitmap2", 3, 0); err != nil {
			t.Fatalf("SetBit(3, 0) error: %v", err)
		}

		val, err := client.GetBit("bitmap2", 3)
		if err != nil {
			t.Fatalf("GetBit error: %v", err)
		}
		if val != 0 {
			t.Errorf("GetBit(3) = %d, want 0", val)
		}
	})

}

func TestGetBit(t *testing.T) {
	t.Run("unset bit returns zero", func(t *testing.T) {
		client, _ := setupTestClient(t)

		val, err := client.GetBit("nokey", 10)
		if err != nil {
			t.Fatalf("GetBit error: %v", err)
		}
		if val != 0 {
			t.Errorf("GetBit on non-existent key = %d, want 0", val)
		}
	})

}

func TestBitCount(t *testing.T) {
	t.Run("counts set bits", func(t *testing.T) {
		client, _ := setupTestClient(t)

		// Set bits at offsets 0, 1, 7
		for _, offset := range []int64{0, 1, 7} {
			if err := client.SetBit("bc", offset, 1); err != nil {
				t.Fatalf("SetBit(%d) error: %v", offset, err)
			}
		}

		count, err := client.BitCount("bc")
		if err != nil {
			t.Fatalf("BitCount error: %v", err)
		}
		if count != 3 {
			t.Errorf("BitCount = %d, want 3", count)
		}
	})

	t.Run("empty key returns zero", func(t *testing.T) {
		client, _ := setupTestClient(t)

		count, err := client.BitCount("nokey")
		if err != nil {
			t.Fatalf("BitCount error: %v", err)
		}
		if count != 0 {
			t.Errorf("BitCount on non-existent key = %d, want 0", count)
		}
	})

}

// ---------------------------------------------------------------------------
// PFAdd / PFCount
// ---------------------------------------------------------------------------

func TestPFAdd(t *testing.T) {
	t.Run("add elements", func(t *testing.T) {
		client, _ := setupTestClient(t)

		if err := client.PFAdd("hll", "a", "b", "c"); err != nil {
			t.Fatalf("PFAdd error: %v", err)
		}

		count, err := client.PFCount("hll")
		if err != nil {
			t.Fatalf("PFCount error: %v", err)
		}
		if count != 3 {
			t.Errorf("PFCount = %d, want 3", count)
		}
	})

	t.Run("duplicate elements", func(t *testing.T) {
		client, _ := setupTestClient(t)

		if err := client.PFAdd("hll2", "x", "y"); err != nil {
			t.Fatalf("PFAdd error: %v", err)
		}
		if err := client.PFAdd("hll2", "x", "z"); err != nil {
			t.Fatalf("PFAdd second call error: %v", err)
		}

		count, err := client.PFCount("hll2")
		if err != nil {
			t.Fatalf("PFCount error: %v", err)
		}
		// x, y, z = 3 unique elements
		if count != 3 {
			t.Errorf("PFCount = %d, want 3", count)
		}
	})

}

func TestPFCount(t *testing.T) {
	t.Run("empty key returns zero", func(t *testing.T) {
		client, _ := setupTestClient(t)

		count, err := client.PFCount("nokey")
		if err != nil {
			t.Fatalf("PFCount error: %v", err)
		}
		if count != 0 {
			t.Errorf("PFCount on non-existent key = %d, want 0", count)
		}
	})

}
