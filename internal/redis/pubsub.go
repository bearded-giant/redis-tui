package redis

import (
	"strconv"
	"strings"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/redis/go-redis/v9"
)

// Publish publishes a message to a channel
func (c *Client) Publish(channel, message string) (int64, error) {
	return c.cmdable().Publish(c.ctx, channel, message).Result()
}

// Subscribe subscribes to channels
func (c *Client) Subscribe(channel string) *redis.PubSub {
	if c.isCluster {
		return c.cluster.Subscribe(c.ctx, channel)
	}
	return c.client.Subscribe(c.ctx, channel)
}

// PubSubChannels lists active channels
func (c *Client) PubSubChannels(pattern string) ([]string, error) {
	return c.cmdable().PubSubChannels(c.ctx, pattern).Result()
}

// SubscribeKeyspace subscribes to keyspace notifications
func (c *Client) SubscribeKeyspace(pattern string, handler func(types.KeyspaceEvent)) error {
	// Enable keyspace notifications (may fail on managed Redis, but we try)
	if c.isCluster {
		_ = c.cluster.ConfigSet(c.ctx, "notify-keyspace-events", "KEA").Err()
	} else {
		_ = c.client.ConfigSet(c.ctx, "notify-keyspace-events", "KEA").Err()
	}

	// Close existing subscription if any to prevent leaks
	if c.keyspacePS != nil {
		_ = c.keyspacePS.Close()
		c.keyspacePS = nil
	}

	// Clear old handlers to prevent memory leak and duplicate events
	c.eventHandlers = []func(types.KeyspaceEvent){handler}

	channel := "__keyspace@" + strconv.Itoa(c.db) + "__:" + pattern
	if c.isCluster {
		c.keyspacePS = c.cluster.PSubscribe(c.ctx, channel)
	} else {
		c.keyspacePS = c.client.PSubscribe(c.ctx, channel)
	}

	go func() {
		ch := c.keyspacePS.Channel()
		for msg := range ch {
			event := types.KeyspaceEvent{
				Timestamp: time.Now(),
				DB:        c.db,
				Event:     msg.Payload,
				Key:       strings.TrimPrefix(msg.Channel, "__keyspace@"+strconv.Itoa(c.db)+"__:"),
			}
			for _, h := range c.eventHandlers {
				h(event)
			}
		}
	}()

	return nil
}

// UnsubscribeKeyspace unsubscribes from keyspace notifications
func (c *Client) UnsubscribeKeyspace() error {
	if c.keyspacePS != nil {
		return c.keyspacePS.Close()
	}
	return nil
}
