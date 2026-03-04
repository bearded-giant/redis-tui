package cmd

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/redis/go-redis/v9"

	tea "github.com/charmbracelet/bubbletea"
)

func LoadKeysCmd(pattern string, cursor uint64, count int64) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.KeysLoadedMsg{Err: nil}
		}
		keys, nextCursor, err := rc.ScanKeys(pattern, cursor, count)
		totalKeys := rc.GetTotalKeys()
		return types.KeysLoadedMsg{Keys: keys, Cursor: nextCursor, TotalKeys: totalKeys, Err: err}
	}
}

func LoadKeyValueCmd(key string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.KeyValueLoadedMsg{Err: nil}
		}
		value, err := rc.GetValue(key)
		return types.KeyValueLoadedMsg{Key: key, Value: value, Err: err}
	}
}

func LoadKeyPreviewCmd(key string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.KeyPreviewLoadedMsg{Err: nil}
		}
		value, err := rc.GetValue(key)
		return types.KeyPreviewLoadedMsg{Key: key, Value: value, Err: err}
	}
}

func DeleteKeyCmd(key string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.KeyDeletedMsg{Key: key, Err: nil}
		}
		err := rc.DeleteKey(key)
		return types.KeyDeletedMsg{Key: key, Err: err}
	}
}

func SetTTLCmd(key string, ttl time.Duration) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.TTLSetMsg{Key: key, Err: nil}
		}
		err := rc.SetTTL(key, ttl)
		return types.TTLSetMsg{Key: key, TTL: ttl, Err: err}
	}
}

func CreateKeyCmd(key string, keyType types.KeyType, value string, extra string, ttl time.Duration) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.KeySetMsg{Key: key, Err: nil}
		}

		// Delete existing key to prevent WRONGTYPE errors
		_ = rc.DeleteKey(key)

		var err error
		switch keyType {
		case types.KeyTypeString:
			err = rc.SetString(key, value, ttl)
		case types.KeyTypeList:
			err = rc.RPush(key, value)
		case types.KeyTypeSet:
			err = rc.SAdd(key, value)
		case types.KeyTypeZSet:
			score := 0.0
			if extra != "" {
				score, _ = strconv.ParseFloat(extra, 64)
			}
			err = rc.ZAdd(key, score, value)
		case types.KeyTypeHash:
			field := extra
			if field == "" {
				field = "field"
			}
			err = rc.HSet(key, field, value)
		case types.KeyTypeStream:
			field := extra
			if field == "" {
				field = "data"
			}
			fields := map[string]interface{}{field: value}
			_, err = rc.XAdd(key, fields)
		case types.KeyTypeJSON:
			err = rc.JSONSet(key, value)
		case types.KeyTypeHyperLogLog:
			err = rc.PFAdd(key, value)
		case types.KeyTypeBitmap:
			offset := int64(0)
			if value != "" {
				offset, _ = strconv.ParseInt(value, 10, 64)
			}
			err = rc.SetBit(key, offset, 1)
		case types.KeyTypeGeo:
			lon, lat := 0.0, 0.0
			if extra != "" {
				parts := strings.SplitN(extra, ",", 2)
				if len(parts) == 2 {
					lon, _ = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
					lat, _ = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
				}
			}
			err = rc.GeoAdd(key, &redis.GeoLocation{Name: value, Longitude: lon, Latitude: lat})
		}
		return types.KeySetMsg{Key: key, Err: err}
	}
}

func EditJSONValueCmd(key, value string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.ValueEditedMsg{Key: key, Err: nil}
		}
		err := rc.JSONSet(key, value)
		return types.ValueEditedMsg{Key: key, Err: err}
	}
}

func EditStringValueCmd(key, value string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.ValueEditedMsg{Key: key, Err: nil}
		}
		err := rc.SetString(key, value, 0)
		return types.ValueEditedMsg{Key: key, Err: err}
	}
}

func EditListElementCmd(key string, index int64, value string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.ValueEditedMsg{Key: key, Err: nil}
		}
		err := rc.LSet(key, index, value)
		return types.ValueEditedMsg{Key: key, Err: err}
	}
}

func EditHashFieldCmd(key, field, value string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.ValueEditedMsg{Key: key, Err: nil}
		}
		err := rc.HSet(key, field, value)
		return types.ValueEditedMsg{Key: key, Err: err}
	}
}

func RenameKeyCmd(oldKey, newKey string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.KeyRenamedMsg{OldKey: oldKey, NewKey: newKey, Err: nil}
		}
		err := rc.Rename(oldKey, newKey)
		return types.KeyRenamedMsg{OldKey: oldKey, NewKey: newKey, Err: err}
	}
}

func CopyKeyCmd(src, dst string, replace bool) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.KeyCopiedMsg{SourceKey: src, DestKey: dst, Err: nil}
		}
		err := rc.Copy(src, dst, replace)
		return types.KeyCopiedMsg{SourceKey: src, DestKey: dst, Err: err}
	}
}

func GetMemoryUsageCmd(key string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.MemoryUsageMsg{Key: key, Err: nil}
		}
		bytes, err := rc.MemoryUsage(key)
		return types.MemoryUsageMsg{Key: key, Bytes: bytes, Err: err}
	}
}

func BulkDeleteCmd(pattern string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.BulkDeleteMsg{Pattern: pattern, Err: nil}
		}
		deleted, err := rc.BulkDelete(pattern)
		return types.BulkDeleteMsg{Pattern: pattern, Deleted: deleted, Err: err}
	}
}

func BatchSetTTLCmd(pattern string, ttl time.Duration) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.BatchTTLSetMsg{Pattern: pattern, Err: nil}
		}
		count, err := rc.BatchSetTTL(pattern, ttl)
		return types.BatchTTLSetMsg{Pattern: pattern, Count: count, TTL: ttl, Err: err}
	}
}

func WatchKeyTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return types.WatchTickMsg{}
	})
}

func LoadValueHistoryCmd(key string) tea.Cmd {
	return func() tea.Msg {
		cfg := GetConfig()
		if cfg == nil {
			return types.ValueHistoryMsg{Err: nil}
		}
		history := cfg.GetValueHistory(key)
		return types.ValueHistoryMsg{History: history, Err: nil}
	}
}

func SaveValueHistoryCmd(key string, value types.RedisValue, action string) tea.Cmd {
	return func() tea.Msg {
		cfg := GetConfig()
		if cfg != nil {
			cfg.AddValueHistory(key, value, action)
		}
		return nil
	}
}

// Keyspace events

func SubscribeKeyspaceCmd(pattern string, sendFunc func(tea.Msg)) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.KeyspaceSubscribedMsg{Err: nil}
		}
		err := rc.SubscribeKeyspace(pattern, func(event types.KeyspaceEvent) {
			if sendFunc != nil {
				sendFunc(types.KeyspaceEventMsg{Event: event})
			}
		})
		return types.KeyspaceSubscribedMsg{Pattern: pattern, Err: err}
	}
}

func UnsubscribeKeyspaceCmd() tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc != nil {
			_ = rc.UnsubscribeKeyspace()
		}
		return nil
	}
}

func LoadKeyPrefixesCmd(separator string, maxDepth int) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.TreeNodeExpandedMsg{Err: nil}
		}
		prefixes, err := rc.GetKeyPrefixes(separator, maxDepth)
		return types.TreeNodeExpandedMsg{Children: prefixes, Err: err}
	}
}

// slog is used for error logging
var _ = slog.Error
