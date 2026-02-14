package redis

import (
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/redis/go-redis/v9"
)

// GetValue retrieves the value for a key
func (c *Client) GetValue(key string) (types.RedisValue, error) {
	keyType, err := c.cmdable().Type(c.ctx, key).Result()
	if err != nil {
		return types.RedisValue{}, err
	}

	var value types.RedisValue
	value.Type = types.KeyType(keyType)

	switch keyType {
	case "string":
		val, err := c.cmdable().Get(c.ctx, key).Result()
		if err != nil {
			return value, err
		}
		value.StringValue = val

	case "list":
		vals, err := c.cmdable().LRange(c.ctx, key, 0, -1).Result()
		if err != nil {
			return value, err
		}
		value.ListValue = vals

	case "set":
		vals, err := c.cmdable().SMembers(c.ctx, key).Result()
		if err != nil {
			return value, err
		}
		value.SetValue = vals

	case "zset":
		vals, err := c.cmdable().ZRangeWithScores(c.ctx, key, 0, -1).Result()
		if err != nil {
			return value, err
		}
		for _, z := range vals {
			value.ZSetValue = append(value.ZSetValue, types.ZSetMember{
				Member: z.Member.(string),
				Score:  z.Score,
			})
		}

	case "hash":
		vals, err := c.cmdable().HGetAll(c.ctx, key).Result()
		if err != nil {
			return value, err
		}
		value.HashValue = vals

	case "stream":
		entries, err := c.cmdable().XRange(c.ctx, key, "-", "+").Result()
		if err != nil {
			return value, err
		}
		for _, entry := range entries {
			value.StreamValue = append(value.StreamValue, types.StreamEntry{
				ID:     entry.ID,
				Fields: entry.Values,
			})
		}
	}

	return value, nil
}

// DeleteKey deletes a single key
func (c *Client) DeleteKey(key string) error {
	return c.cmdable().Del(c.ctx, key).Err()
}

// DeleteKeys deletes multiple keys
func (c *Client) DeleteKeys(keys ...string) (int64, error) {
	return c.cmdable().Del(c.ctx, keys...).Result()
}

// BulkDelete deletes all keys matching a pattern
func (c *Client) BulkDelete(pattern string) (int, error) {
	allKeys, err := c.scanAll(pattern, 100)
	if err != nil {
		return 0, err
	}

	var deleted int
	// Delete in chunks to avoid huge DEL commands
	chunkSize := 100
	for i := 0; i < len(allKeys); i += chunkSize {
		end := min(i+chunkSize, len(allKeys))
		count, err := c.cmdable().Del(c.ctx, allKeys[i:end]...).Result()
		if err != nil {
			return deleted, err
		}
		deleted += int(count)
	}

	return deleted, nil
}

// SetString sets a string value
func (c *Client) SetString(key, value string, ttl time.Duration) error {
	return c.cmdable().Set(c.ctx, key, value, ttl).Err()
}

// SetTTL sets or removes TTL on a key
func (c *Client) SetTTL(key string, ttl time.Duration) error {
	if ttl <= 0 {
		return c.cmdable().Persist(c.ctx, key).Err()
	}
	return c.cmdable().Expire(c.ctx, key, ttl).Err()
}

// BatchSetTTL sets TTL on all keys matching a pattern
func (c *Client) BatchSetTTL(pattern string, ttl time.Duration) (int, error) {
	allKeys, err := c.scanAll(pattern, 100)
	if err != nil {
		return 0, err
	}

	var count int
	// Process in chunks to keep pipeline sizes reasonable
	chunkSize := 100
	for i := 0; i < len(allKeys); i += chunkSize {
		end := min(i+chunkSize, len(allKeys))
		keys := allKeys[i:end]

		pipe := c.pipeline()
		cmds := make([]*redis.BoolCmd, len(keys))

		for j, key := range keys {
			if ttl <= 0 {
				cmds[j] = pipe.Persist(c.ctx, key)
			} else {
				cmds[j] = pipe.Expire(c.ctx, key, ttl)
			}
		}

		_, _ = pipe.Exec(c.ctx)

		for _, cmd := range cmds {
			if cmd.Err() == nil {
				count++
			}
		}
	}

	return count, nil
}

// MemoryUsage returns memory usage for a key
func (c *Client) MemoryUsage(key string) (int64, error) {
	return c.cmdable().MemoryUsage(c.ctx, key).Result()
}

// Rename renames a key
func (c *Client) Rename(oldKey, newKey string) error {
	return c.cmdable().Rename(c.ctx, oldKey, newKey).Err()
}

// Copy copies a key
func (c *Client) Copy(src, dst string, replace bool) error {
	return c.cmdable().Copy(c.ctx, src, dst, 0, replace).Err()
}
