package redis

import "github.com/redis/go-redis/v9"

// RPush appends values to a list
func (c *Client) RPush(key string, values ...string) error {
	args := make([]interface{}, len(values))
	for i, v := range values {
		args[i] = v
	}
	return c.cmdable().RPush(c.ctx, key, args...).Err()
}

// SAdd adds members to a set
func (c *Client) SAdd(key string, members ...string) error {
	args := make([]interface{}, len(members))
	for i, v := range members {
		args[i] = v
	}
	return c.cmdable().SAdd(c.ctx, key, args...).Err()
}

// ZAdd adds a member to a sorted set
func (c *Client) ZAdd(key string, score float64, member string) error {
	return c.cmdable().ZAdd(c.ctx, key, redis.Z{Score: score, Member: member}).Err()
}

// HSet sets a hash field
func (c *Client) HSet(key, field, value string) error {
	return c.cmdable().HSet(c.ctx, key, field, value).Err()
}

// XAdd adds an entry to a stream
func (c *Client) XAdd(key string, fields map[string]interface{}) (string, error) {
	return c.cmdable().XAdd(c.ctx, &redis.XAddArgs{
		Stream: key,
		Values: fields,
	}).Result()
}

// LSet sets a list element by index
func (c *Client) LSet(key string, index int64, value string) error {
	return c.cmdable().LSet(c.ctx, key, index, value).Err()
}

// LRem removes list elements
func (c *Client) LRem(key string, count int64, value string) error {
	return c.cmdable().LRem(c.ctx, key, count, value).Err()
}

// SRem removes set members
func (c *Client) SRem(key string, members ...string) error {
	args := make([]interface{}, len(members))
	for i, v := range members {
		args[i] = v
	}
	return c.cmdable().SRem(c.ctx, key, args...).Err()
}

// ZRem removes sorted set members
func (c *Client) ZRem(key string, members ...string) error {
	args := make([]interface{}, len(members))
	for i, v := range members {
		args[i] = v
	}
	return c.cmdable().ZRem(c.ctx, key, args...).Err()
}

// HDel removes hash fields
func (c *Client) HDel(key string, fields ...string) error {
	return c.cmdable().HDel(c.ctx, key, fields...).Err()
}

// XDel removes stream entries
func (c *Client) XDel(key string, ids ...string) error {
	return c.cmdable().XDel(c.ctx, key, ids...).Err()
}

// ZAddBatch adds multiple members to a sorted set in one call
func (c *Client) ZAddBatch(key string, members ...redis.Z) error {
	return c.cmdable().ZAdd(c.ctx, key, members...).Err()
}

// HSetMap sets multiple hash fields in one call
func (c *Client) HSetMap(key string, fields map[string]string) error {
	if len(fields) == 0 {
		return nil
	}
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return c.cmdable().HSet(c.ctx, key, args...).Err()
}

// PFAdd adds elements to a HyperLogLog
func (c *Client) PFAdd(key string, elements ...string) error {
	args := make([]interface{}, len(elements))
	for i, v := range elements {
		args[i] = v
	}
	return c.cmdable().PFAdd(c.ctx, key, args...).Err()
}

// PFCount returns the approximate cardinality of a HyperLogLog
func (c *Client) PFCount(key string) (int64, error) {
	return c.cmdable().PFCount(c.ctx, key).Result()
}

// GeoAdd adds members with coordinates to a geospatial index
func (c *Client) GeoAdd(key string, members ...*redis.GeoLocation) error {
	return c.cmdable().GeoAdd(c.ctx, key, members...).Err()
}

// GeoPos returns the positions of members in a geospatial index
func (c *Client) GeoPos(key string, members ...string) ([]*redis.GeoPos, error) {
	return c.cmdable().GeoPos(c.ctx, key, members...).Result()
}

// SetBit sets a bit at the given offset
func (c *Client) SetBit(key string, offset int64, value int) error {
	return c.cmdable().SetBit(c.ctx, key, offset, value).Err()
}

// GetBit returns the bit value at the given offset
func (c *Client) GetBit(key string, offset int64) (int64, error) {
	return c.cmdable().GetBit(c.ctx, key, offset).Result()
}

// BitCount returns the number of set bits in a string
func (c *Client) BitCount(key string) (int64, error) {
	return c.cmdable().BitCount(c.ctx, key, nil).Result()
}
