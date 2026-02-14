package redis

import (
	"regexp"
	"sort"
	"strings"

	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/redis/go-redis/v9"
)

// GetTotalKeys returns the total number of keys in the current database
func (c *Client) GetTotalKeys() int64 {
	count, err := c.cmdable().DBSize(c.ctx).Result()
	if err != nil {
		return 0
	}
	return count
}

// ScanKeys scans keys matching a pattern
func (c *Client) ScanKeys(pattern string, cursor uint64, count int64) ([]types.RedisKey, uint64, error) {
	if pattern == "" {
		pattern = "*"
	}

	var keys []string
	var nextCursor uint64
	var err error

	if c.isCluster {
		// In cluster mode, scan all masters to get keys from every shard
		keys, err = c.scanAll(pattern, count)
		nextCursor = 0
	} else {
		keys, nextCursor, err = c.client.Scan(c.ctx, cursor, pattern, count).Result()
	}
	if err != nil {
		return nil, 0, err
	}

	if len(keys) == 0 {
		return []types.RedisKey{}, nextCursor, nil
	}

	// Use pipeline to batch TTL (and optionally TYPE) calls
	pipe := c.pipeline()
	var typeCmds []*redis.StatusCmd
	ttlCmds := make([]*redis.DurationCmd, len(keys))

	if c.includeTypes {
		typeCmds = make([]*redis.StatusCmd, len(keys))
		for i, key := range keys {
			typeCmds[i] = pipe.Type(c.ctx, key)
			ttlCmds[i] = pipe.TTL(c.ctx, key)
		}
	} else {
		for i, key := range keys {
			ttlCmds[i] = pipe.TTL(c.ctx, key)
		}
	}

	_, err = pipe.Exec(c.ctx)
	if err != nil && err != redis.Nil {
		return nil, 0, err
	}

	result := make([]types.RedisKey, len(keys))
	for i, key := range keys {
		var keyType string
		if c.includeTypes && typeCmds != nil {
			keyType, _ = typeCmds[i].Result()
		}
		ttl, _ := ttlCmds[i].Result()
		result[i] = types.RedisKey{
			Key:  key,
			Type: types.KeyType(keyType),
			TTL:  ttl,
		}
	}

	return result, nextCursor, nil
}

// ScanKeysWithRegex scans keys using regex pattern
func (c *Client) ScanKeysWithRegex(regexPattern string, maxKeys int) ([]types.RedisKey, error) {
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, errInvalidRegex(err)
	}

	allKeys, err := c.scanAll("*", 100)
	if err != nil {
		return nil, err
	}

	// Filter by regex
	matchingKeys := make([]string, 0, maxKeys)
	for _, key := range allKeys {
		if re.MatchString(key) {
			matchingKeys = append(matchingKeys, key)
			if len(matchingKeys) >= maxKeys {
				break
			}
		}
	}

	if len(matchingKeys) == 0 {
		return []types.RedisKey{}, nil
	}

	// Use pipeline to batch Type and TTL calls
	pipe := c.pipeline()
	typeCmds := make([]*redis.StatusCmd, len(matchingKeys))
	ttlCmds := make([]*redis.DurationCmd, len(matchingKeys))

	for i, key := range matchingKeys {
		typeCmds[i] = pipe.Type(c.ctx, key)
		ttlCmds[i] = pipe.TTL(c.ctx, key)
	}

	_, _ = pipe.Exec(c.ctx)

	result := make([]types.RedisKey, len(matchingKeys))
	for i, key := range matchingKeys {
		keyType, _ := typeCmds[i].Result()
		ttl, _ := ttlCmds[i].Result()
		result[i] = types.RedisKey{
			Key:  key,
			Type: types.KeyType(keyType),
			TTL:  ttl,
		}
	}

	return result, nil
}

// FuzzySearchKeys performs fuzzy matching on key names
func (c *Client) FuzzySearchKeys(searchTerm string, maxKeys int) ([]types.RedisKey, error) {
	searchLower := strings.ToLower(searchTerm)

	allKeys, err := c.scanAll("*", 100)
	if err != nil {
		return nil, err
	}

	type scoredKey struct {
		key   string
		score int
	}
	scoredKeys := make([]scoredKey, 0, maxKeys*2)

	for _, key := range allKeys {
		keyLower := strings.ToLower(key)
		score := fuzzyScore(keyLower, searchLower)
		if score > 0 {
			scoredKeys = append(scoredKeys, scoredKey{key: key, score: score})
		}
	}

	// Sort by score descending
	sort.Slice(scoredKeys, func(i, j int) bool {
		return scoredKeys[i].score > scoredKeys[j].score
	})

	// Limit to maxKeys
	limit := min(maxKeys, len(scoredKeys))
	scoredKeys = scoredKeys[:limit]

	if len(scoredKeys) == 0 {
		return []types.RedisKey{}, nil
	}

	// Use pipeline to batch Type and TTL calls for top results only
	pipe := c.pipeline()
	typeCmds := make([]*redis.StatusCmd, len(scoredKeys))
	ttlCmds := make([]*redis.DurationCmd, len(scoredKeys))

	for i, sk := range scoredKeys {
		typeCmds[i] = pipe.Type(c.ctx, sk.key)
		ttlCmds[i] = pipe.TTL(c.ctx, sk.key)
	}

	_, _ = pipe.Exec(c.ctx)

	result := make([]types.RedisKey, len(scoredKeys))
	for i, sk := range scoredKeys {
		keyType, _ := typeCmds[i].Result()
		ttl, _ := ttlCmds[i].Result()
		result[i] = types.RedisKey{
			Key:  sk.key,
			Type: types.KeyType(keyType),
			TTL:  ttl,
		}
	}

	return result, nil
}

func fuzzyScore(str, pattern string) int {
	if strings.Contains(str, pattern) {
		return 100 + (len(str) - len(pattern))
	}

	score := 0
	patternIdx := 0

	for i := 0; i < len(str) && patternIdx < len(pattern); i++ {
		if str[i] == pattern[patternIdx] {
			score += 10
			if i > 0 && (str[i-1] == ':' || str[i-1] == '_' || str[i-1] == '-') {
				score += 5
			}
			patternIdx++
		}
	}

	if patternIdx == len(pattern) {
		return score
	}
	return 0
}

// SearchByValue searches for keys containing a value
func (c *Client) SearchByValue(pattern string, valueSearch string, maxKeys int) ([]types.RedisKey, error) {
	allKeys, err := c.scanAll(pattern, 100)
	if err != nil {
		return nil, err
	}

	result := make([]types.RedisKey, 0, maxKeys)

	// Process in chunks to keep pipeline sizes reasonable
	chunkSize := 100
	for i := 0; i < len(allKeys) && len(result) < maxKeys; i += chunkSize {
		end := min(i+chunkSize, len(allKeys))
		keys := allKeys[i:end]

		// First pipeline: get types for all keys
		typePipe := c.pipeline()
		typeCmds := make([]*redis.StatusCmd, len(keys))
		for j, key := range keys {
			typeCmds[j] = typePipe.Type(c.ctx, key)
		}
		_, _ = typePipe.Exec(c.ctx)

		keyTypes := make([]string, len(keys))
		for j := range keys {
			keyTypes[j], _ = typeCmds[j].Result()
		}

		// Second pipeline: get values based on type
		valuePipe := c.pipeline()
		type valueCmd struct {
			idx     int
			keyType string
			strCmd  *redis.StringCmd
			hashCmd *redis.MapStringStringCmd
			listCmd *redis.StringSliceCmd
			setCmd  *redis.StringSliceCmd
		}
		valueCmds := make([]valueCmd, 0, len(keys))

		for j, key := range keys {
			kt := keyTypes[j]
			vc := valueCmd{idx: j, keyType: kt}
			switch kt {
			case "string":
				vc.strCmd = valuePipe.Get(c.ctx, key)
			case "hash":
				vc.hashCmd = valuePipe.HGetAll(c.ctx, key)
			case "list":
				vc.listCmd = valuePipe.LRange(c.ctx, key, 0, -1)
			case "set":
				vc.setCmd = valuePipe.SMembers(c.ctx, key)
			default:
				continue
			}
			valueCmds = append(valueCmds, vc)
		}
		_, _ = valuePipe.Exec(c.ctx)

		// Find matching keys
		matchingIndices := make([]int, 0)
		for _, vc := range valueCmds {
			found := false
			switch vc.keyType {
			case "string":
				val, _ := vc.strCmd.Result()
				found = strings.Contains(val, valueSearch)
			case "hash":
				vals, _ := vc.hashCmd.Result()
				for _, v := range vals {
					if strings.Contains(v, valueSearch) {
						found = true
						break
					}
				}
			case "list":
				vals, _ := vc.listCmd.Result()
				for _, v := range vals {
					if strings.Contains(v, valueSearch) {
						found = true
						break
					}
				}
			case "set":
				vals, _ := vc.setCmd.Result()
				for _, v := range vals {
					if strings.Contains(v, valueSearch) {
						found = true
						break
					}
				}
			}
			if found {
				matchingIndices = append(matchingIndices, vc.idx)
			}
		}

		if len(matchingIndices) > 0 {
			ttlPipe := c.pipeline()
			ttlCmds := make([]*redis.DurationCmd, len(matchingIndices))
			for j, idx := range matchingIndices {
				ttlCmds[j] = ttlPipe.TTL(c.ctx, keys[idx])
			}
			_, _ = ttlPipe.Exec(c.ctx)

			for j, idx := range matchingIndices {
				if len(result) >= maxKeys {
					break
				}
				ttl, _ := ttlCmds[j].Result()
				result = append(result, types.RedisKey{
					Key:  keys[idx],
					Type: types.KeyType(keyTypes[idx]),
					TTL:  ttl,
				})
			}
		}
	}

	return result, nil
}

// GetKeyPrefixes returns all unique key prefixes (for tree view)
func (c *Client) GetKeyPrefixes(separator string, maxDepth int) ([]string, error) {
	allKeys, err := c.scanAll("*", 500)
	if err != nil {
		return nil, err
	}

	prefixes := make(map[string]bool)
	for _, key := range allKeys {
		parts := strings.Split(key, separator)
		for i := 1; i <= len(parts) && i <= maxDepth; i++ {
			prefix := strings.Join(parts[:i], separator)
			prefixes[prefix] = true
		}
	}

	result := make([]string, 0, len(prefixes))
	for p := range prefixes {
		result = append(result, p)
	}
	sort.Strings(result)

	return result, nil
}
