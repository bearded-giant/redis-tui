package cmd

import (
	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/redis/go-redis/v9"

	tea "github.com/charmbracelet/bubbletea"
)

// Add to collection commands

func AddToListCmd(key string, values ...string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.ItemAddedToCollectionMsg{Key: key, Err: nil}
		}
		err := rc.RPush(key, values...)
		return types.ItemAddedToCollectionMsg{Key: key, Err: err}
	}
}

func AddToSetCmd(key string, members ...string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.ItemAddedToCollectionMsg{Key: key, Err: nil}
		}
		err := rc.SAdd(key, members...)
		return types.ItemAddedToCollectionMsg{Key: key, Err: err}
	}
}

func AddToZSetCmd(key string, score float64, member string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.ItemAddedToCollectionMsg{Key: key, Err: nil}
		}
		err := rc.ZAdd(key, score, member)
		return types.ItemAddedToCollectionMsg{Key: key, Err: err}
	}
}

func AddToHashCmd(key, field, value string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.ItemAddedToCollectionMsg{Key: key, Err: nil}
		}
		err := rc.HSet(key, field, value)
		return types.ItemAddedToCollectionMsg{Key: key, Err: err}
	}
}

func AddToStreamCmd(key string, fields map[string]interface{}) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.ItemAddedToCollectionMsg{Key: key, Err: nil}
		}
		_, err := rc.XAdd(key, fields)
		return types.ItemAddedToCollectionMsg{Key: key, Err: err}
	}
}

// Remove from collection commands

func RemoveFromListCmd(key string, value string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.ItemRemovedFromCollectionMsg{Key: key, Err: nil}
		}
		err := rc.LRem(key, 1, value)
		return types.ItemRemovedFromCollectionMsg{Key: key, Err: err}
	}
}

func RemoveFromSetCmd(key string, members ...string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.ItemRemovedFromCollectionMsg{Key: key, Err: nil}
		}
		err := rc.SRem(key, members...)
		return types.ItemRemovedFromCollectionMsg{Key: key, Err: err}
	}
}

func RemoveFromZSetCmd(key string, members ...string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.ItemRemovedFromCollectionMsg{Key: key, Err: nil}
		}
		err := rc.ZRem(key, members...)
		return types.ItemRemovedFromCollectionMsg{Key: key, Err: err}
	}
}

func RemoveFromHashCmd(key string, fields ...string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.ItemRemovedFromCollectionMsg{Key: key, Err: nil}
		}
		err := rc.HDel(key, fields...)
		return types.ItemRemovedFromCollectionMsg{Key: key, Err: err}
	}
}

func RemoveFromStreamCmd(key string, ids ...string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.ItemRemovedFromCollectionMsg{Key: key, Err: nil}
		}
		err := rc.XDel(key, ids...)
		return types.ItemRemovedFromCollectionMsg{Key: key, Err: err}
	}
}

func AddToHLLCmd(key string, elements ...string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.ItemAddedToCollectionMsg{Key: key, Err: nil}
		}
		err := rc.PFAdd(key, elements...)
		return types.ItemAddedToCollectionMsg{Key: key, Err: err}
	}
}

func AddToGeoCmd(key string, lon, lat float64, member string) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.ItemAddedToCollectionMsg{Key: key, Err: nil}
		}
		err := rc.GeoAdd(key, &redis.GeoLocation{Name: member, Longitude: lon, Latitude: lat})
		return types.ItemAddedToCollectionMsg{Key: key, Err: err}
	}
}

func SetBitCmd(key string, offset int64, value int) tea.Cmd {
	return func() tea.Msg {
		rc := getRedisClient()
		if rc == nil {
			return types.ItemAddedToCollectionMsg{Key: key, Err: nil}
		}
		err := rc.SetBit(key, offset, value)
		return types.ItemAddedToCollectionMsg{Key: key, Err: err}
	}
}
