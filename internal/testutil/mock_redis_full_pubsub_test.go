package testutil

import (
	"errors"
	"testing"

	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestFullMockRedisClient_Publish(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.PublishResult = 3
		n, err := m.Publish("channel", "msg")
		AssertNoError(t, err, "Publish")
		AssertEqual(t, n, int64(3), "Publish result")
		AssertEqual(t, m.Calls[0], "Publish", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.PublishError = errTest
		_, err := m.Publish("channel", "msg")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_Subscribe(t *testing.T) {
	m := NewFullMockRedisClient()
	result := m.Subscribe("channel")
	if result != nil {
		t.Error("Subscribe should return nil")
	}
	AssertEqual(t, m.Calls[0], "Subscribe", "call name")
}

func TestFullMockRedisClient_PubSubChannels(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.PubSubChannelsResult = []string{"ch1", "ch2"}
		got, err := m.PubSubChannels("*")
		AssertNoError(t, err, "PubSubChannels")
		AssertSliceLen(t, got, 2, "PubSubChannels result")
		AssertEqual(t, got[0], "ch1", "channel 0")
		AssertEqual(t, m.Calls[0], "PubSubChannels", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.PubSubChannelsError = errTest
		_, err := m.PubSubChannels("*")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_SubscribeKeyspace(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.SubscribeKeyspace("*", func(_ types.KeyspaceEvent) {})
		AssertNoError(t, err, "SubscribeKeyspace")
		AssertEqual(t, m.Calls[0], "SubscribeKeyspace", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.SubscribeKeyspaceError = errTest
		err := m.SubscribeKeyspace("*", func(_ types.KeyspaceEvent) {})
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})

	t.Run("invokes handler for each configured event in order", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.SubscribeKeyspaceEvents = []types.KeyspaceEvent{
			{Key: "k1", Event: "set"},
			{Key: "k2", Event: "del"},
			{Key: "k3", Event: "expire"},
		}
		var received []types.KeyspaceEvent
		err := m.SubscribeKeyspace("*", func(e types.KeyspaceEvent) {
			received = append(received, e)
		})
		AssertNoError(t, err, "SubscribeKeyspace")
		AssertSliceLen(t, received, 3, "received events")
		AssertEqual(t, received[0].Key, "k1", "event 0 key")
		AssertEqual(t, received[0].Event, "set", "event 0 event")
		AssertEqual(t, received[1].Key, "k2", "event 1 key")
		AssertEqual(t, received[2].Key, "k3", "event 2 key")
	})

	t.Run("returns configured error after invoking handler", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.SubscribeKeyspaceEvents = []types.KeyspaceEvent{{Key: "k", Event: "set"}}
		m.SubscribeKeyspaceError = errTest
		var calls int
		err := m.SubscribeKeyspace("*", func(_ types.KeyspaceEvent) {
			calls++
		})
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
		AssertEqual(t, calls, 1, "handler invocations")
	})

	t.Run("nil handler is tolerated", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.SubscribeKeyspaceEvents = []types.KeyspaceEvent{{Key: "k", Event: "set"}}
		err := m.SubscribeKeyspace("*", nil)
		AssertNoError(t, err, "SubscribeKeyspace nil handler")
	})
}

func TestFullMockRedisClient_UnsubscribeKeyspace(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.UnsubscribeKeyspace()
		AssertNoError(t, err, "UnsubscribeKeyspace")
		AssertEqual(t, m.Calls[0], "UnsubscribeKeyspace", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.UnsubscribeKSError = errTest
		err := m.UnsubscribeKeyspace()
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_ConfigGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ConfigGetResult = map[string]string{"maxmemory": "100mb"}
		got, err := m.ConfigGet("maxmemory")
		AssertNoError(t, err, "ConfigGet")
		AssertEqual(t, got["maxmemory"], "100mb", "config value")
		AssertEqual(t, m.Calls[0], "ConfigGet", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ConfigGetError = errTest
		_, err := m.ConfigGet("maxmemory")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_ConfigSet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		err := m.ConfigSet("maxmemory", "100mb")
		AssertNoError(t, err, "ConfigSet")
		AssertEqual(t, m.Calls[0], "ConfigSet", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ConfigSetError = errTest
		err := m.ConfigSet("maxmemory", "100mb")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_ScanKeysWithRegex(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.RegexSearchResult = []types.RedisKey{{Key: "user:1"}}
		got, err := m.ScanKeysWithRegex("user:.*", 100)
		AssertNoError(t, err, "ScanKeysWithRegex")
		AssertSliceLen(t, got, 1, "ScanKeysWithRegex result")
		AssertEqual(t, got[0].Key, "user:1", "key name")
		AssertEqual(t, m.Calls[0], "ScanKeysWithRegex", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.RegexSearchError = errTest
		_, err := m.ScanKeysWithRegex("user:.*", 100)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_FuzzySearchKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.FuzzySearchResult = []types.RedisKey{{Key: "session:abc"}}
		got, err := m.FuzzySearchKeys("sess", 50)
		AssertNoError(t, err, "FuzzySearchKeys")
		AssertSliceLen(t, got, 1, "FuzzySearchKeys result")
		AssertEqual(t, m.Calls[0], "FuzzySearchKeys", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.FuzzySearchError = errTest
		_, err := m.FuzzySearchKeys("sess", 50)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_SearchByValue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.SearchByValueResult = []types.RedisKey{{Key: "k1"}, {Key: "k2"}}
		got, err := m.SearchByValue("*", "needle", 100)
		AssertNoError(t, err, "SearchByValue")
		AssertSliceLen(t, got, 2, "SearchByValue result")
		AssertEqual(t, m.Calls[0], "SearchByValue", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.SearchByValueError = errTest
		_, err := m.SearchByValue("*", "needle", 100)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_CompareKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.CompareValue1 = types.RedisValue{StringValue: "a"}
		m.CompareValue2 = types.RedisValue{StringValue: "b"}
		v1, v2, err := m.CompareKeys("key1", "key2")
		AssertNoError(t, err, "CompareKeys")
		AssertEqual(t, v1.StringValue, "a", "CompareValue1")
		AssertEqual(t, v2.StringValue, "b", "CompareValue2")
		AssertEqual(t, m.Calls[0], "CompareKeys", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.CompareKeysError = errTest
		_, _, err := m.CompareKeys("key1", "key2")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_GetKeyPrefixes(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.KeyPrefixesResult = []string{"user:", "session:"}
		got, err := m.GetKeyPrefixes(":", 10)
		AssertNoError(t, err, "GetKeyPrefixes")
		AssertSliceLen(t, got, 2, "GetKeyPrefixes result")
		AssertEqual(t, m.Calls[0], "GetKeyPrefixes", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.KeyPrefixesError = errTest
		_, err := m.GetKeyPrefixes(":", 10)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_ExportKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ExportResult = map[string]any{"key1": "val1"}
		got, err := m.ExportKeys("*")
		AssertNoError(t, err, "ExportKeys")
		if got["key1"] != "val1" {
			t.Errorf("expected key1=val1, got %v", got["key1"])
		}
		AssertEqual(t, m.Calls[0], "ExportKeys", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ExportError = errTest
		_, err := m.ExportKeys("*")
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}

func TestFullMockRedisClient_ImportKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ImportResult = 3
		n, err := m.ImportKeys(map[string]any{"a": "1", "b": "2", "c": "3"})
		AssertNoError(t, err, "ImportKeys")
		AssertEqual(t, n, 3, "ImportKeys result")
		AssertEqual(t, m.Calls[0], "ImportKeys", "call name")
	})

	t.Run("error", func(t *testing.T) {
		m := NewFullMockRedisClient()
		m.ImportError = errTest
		_, err := m.ImportKeys(map[string]any{})
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest, got %v", err)
		}
	})
}
