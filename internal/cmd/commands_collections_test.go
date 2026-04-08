package cmd

import (
	"errors"
	"testing"

	"github.com/davidbudnick/redis-tui/internal/testutil"
	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestAddToList(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.AddToList("list", "item1", "item2")()
		result := msg.(types.ItemAddedToCollectionMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.AddToList("list", "item")()
		result := msg.(types.ItemAddedToCollectionMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestAddToSet(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.AddToSet("set", "member1", "member2")()
	result := msg.(types.ItemAddedToCollectionMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestAddToZSet(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.AddToZSet("zset", 1.5, "member")()
	result := msg.(types.ItemAddedToCollectionMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestAddToHash(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.AddToHash("hash", "field", "value")()
	result := msg.(types.ItemAddedToCollectionMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestAddToStream(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.AddToStream("stream", map[string]any{"key": "val"})()
	result := msg.(types.ItemAddedToCollectionMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestRemoveFromList(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.RemoveFromList("list", "item")()
	result := msg.(types.ItemRemovedFromCollectionMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestRemoveFromSet(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.RemoveFromSet("set", "member")()
	result := msg.(types.ItemRemovedFromCollectionMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestRemoveFromZSet(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.RemoveFromZSet("zset", "member")()
	result := msg.(types.ItemRemovedFromCollectionMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestRemoveFromHash(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.RemoveFromHash("hash", "field")()
	result := msg.(types.ItemRemovedFromCollectionMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestRemoveFromStream(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.RemoveFromStream("stream", "1-0")()
	result := msg.(types.ItemRemovedFromCollectionMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestCollectionErrors(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*testutil.FullMockRedisClient)
		execute func(*Commands) any
		wantErr bool
	}{
		{
			"AddToList error",
			func(m *testutil.FullMockRedisClient) { m.RPushError = errors.New("err") },
			func(c *Commands) any { return c.AddToList("k", "v")() },
			true,
		},
		{
			"AddToSet error",
			func(m *testutil.FullMockRedisClient) { m.SAddError = errors.New("err") },
			func(c *Commands) any { return c.AddToSet("k", "v")() },
			true,
		},
		{
			"AddToZSet error",
			func(m *testutil.FullMockRedisClient) { m.ZAddError = errors.New("err") },
			func(c *Commands) any { return c.AddToZSet("k", 1.0, "v")() },
			true,
		},
		{
			"AddToHash error",
			func(m *testutil.FullMockRedisClient) { m.HSetError = errors.New("err") },
			func(c *Commands) any { return c.AddToHash("k", "f", "v")() },
			true,
		},
		{
			"AddToStream error",
			func(m *testutil.FullMockRedisClient) { m.XAddError = errors.New("err") },
			func(c *Commands) any {
				return c.AddToStream("k", map[string]any{"f": "v"})()
			},
			true,
		},
		{
			"RemoveFromList error",
			func(m *testutil.FullMockRedisClient) { m.LRemError = errors.New("err") },
			func(c *Commands) any { return c.RemoveFromList("k", "v")() },
			true,
		},
		{
			"RemoveFromSet error",
			func(m *testutil.FullMockRedisClient) { m.SRemError = errors.New("err") },
			func(c *Commands) any { return c.RemoveFromSet("k", "v")() },
			true,
		},
		{
			"RemoveFromZSet error",
			func(m *testutil.FullMockRedisClient) { m.ZRemError = errors.New("err") },
			func(c *Commands) any { return c.RemoveFromZSet("k", "v")() },
			true,
		},
		{
			"RemoveFromHash error",
			func(m *testutil.FullMockRedisClient) { m.HDelError = errors.New("err") },
			func(c *Commands) any { return c.RemoveFromHash("k", "f")() },
			true,
		},
		{
			"RemoveFromStream error",
			func(m *testutil.FullMockRedisClient) { m.XDelError = errors.New("err") },
			func(c *Commands) any { return c.RemoveFromStream("k", "1-0")() },
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := testutil.NewFullMockRedisClient()
			tt.setup(mock)
			cmds := NewCommands(nil, mock)
			result := tt.execute(cmds)

			// All collection messages have an Err field
			switch msg := result.(type) {
			case types.ItemAddedToCollectionMsg:
				if tt.wantErr && msg.Err == nil {
					t.Error("expected error")
				}
			case types.ItemRemovedFromCollectionMsg:
				if tt.wantErr && msg.Err == nil {
					t.Error("expected error")
				}
			default:
				t.Errorf("unexpected message type: %T", result)
			}
		})
	}
}

func TestAddToHLL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.AddToHLL("hll", "elem1", "elem2")()
		result := msg.(types.ItemAddedToCollectionMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Key != "hll" {
			t.Errorf("Key = %q, want %q", result.Key, "hll")
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.PFAddError = errors.New("pfadd error")
		msg := cmds.AddToHLL("hll", "elem")()
		result := msg.(types.ItemAddedToCollectionMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.AddToHLL("hll", "elem")()
		result := msg.(types.ItemAddedToCollectionMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestAddToGeo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.AddToGeo("geo", -122.4194, 37.7749, "San Francisco")()
		result := msg.(types.ItemAddedToCollectionMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Key != "geo" {
			t.Errorf("Key = %q, want %q", result.Key, "geo")
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.GeoAddError = errors.New("geoadd error")
		msg := cmds.AddToGeo("geo", 0, 0, "origin")()
		result := msg.(types.ItemAddedToCollectionMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.AddToGeo("geo", 0, 0, "origin")()
		result := msg.(types.ItemAddedToCollectionMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestCollectionNilRedisBranches(t *testing.T) {
	cmds := NewCommands(nil, nil)
	cases := []struct {
		name string
		fn   func() any
	}{
		{"AddToSet", func() any { return cmds.AddToSet("k", "m")() }},
		{"AddToZSet", func() any { return cmds.AddToZSet("k", 1, "m")() }},
		{"AddToHash", func() any { return cmds.AddToHash("k", "f", "v")() }},
		{"AddToStream", func() any { return cmds.AddToStream("k", map[string]any{"f": "v"})() }},
		{"RemoveFromList", func() any { return cmds.RemoveFromList("k", "v")() }},
		{"RemoveFromSet", func() any { return cmds.RemoveFromSet("k", "m")() }},
		{"RemoveFromZSet", func() any { return cmds.RemoveFromZSet("k", "m")() }},
		{"RemoveFromHash", func() any { return cmds.RemoveFromHash("k", "f")() }},
		{"RemoveFromStream", func() any { return cmds.RemoveFromStream("k", "1-0")() }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := c.fn()
			switch m := res.(type) {
			case types.ItemAddedToCollectionMsg:
				if m.Err != nil {
					t.Errorf("nil redis should not error: %v", m.Err)
				}
			case types.ItemRemovedFromCollectionMsg:
				if m.Err != nil {
					t.Errorf("nil redis should not error: %v", m.Err)
				}
			default:
				t.Errorf("unexpected msg type: %T", res)
			}
		})
	}
}

func TestSetBit(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.SetBit("bitmap", 7, 1)()
		result := msg.(types.ItemAddedToCollectionMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Key != "bitmap" {
			t.Errorf("Key = %q, want %q", result.Key, "bitmap")
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.SetBitError = errors.New("setbit error")
		msg := cmds.SetBit("bitmap", 0, 1)()
		result := msg.(types.ItemAddedToCollectionMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.SetBit("bitmap", 0, 1)()
		result := msg.(types.ItemAddedToCollectionMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}
