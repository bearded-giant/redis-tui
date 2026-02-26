package redis

import (
	"context"
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
	c.mu.RLock()
	isCluster := c.isCluster
	cluster := c.cluster
	client := c.client
	ctx := c.ctx
	c.mu.RUnlock()

	if isCluster {
		return cluster.Subscribe(ctx, channel)
	}
	return client.Subscribe(ctx, channel)
}

// PubSubChannels lists active channels
func (c *Client) PubSubChannels(pattern string) ([]string, error) {
	return c.cmdable().PubSubChannels(c.ctx, pattern).Result()
}

// SubscribeKeyspace subscribes to keyspace notifications
func (c *Client) SubscribeKeyspace(pattern string, handler func(types.KeyspaceEvent)) error {
	// Enable keyspace notifications (may fail on managed Redis, but we try)
	c.mu.RLock()
	isCluster := c.isCluster
	cluster := c.cluster
	client := c.client
	ctx := c.ctx
	c.mu.RUnlock()

	if isCluster {
		_ = cluster.ConfigSet(ctx, "notify-keyspace-events", "KEA").Err()
	} else {
		_ = client.ConfigSet(ctx, "notify-keyspace-events", "KEA").Err()
	}

	c.mu.Lock()
	// Cancel previous goroutine if any
	if c.cancelKeyspace != nil {
		c.cancelKeyspace()
		c.cancelKeyspace = nil
	}

	// Close existing subscription if any to prevent leaks
	if c.keyspacePS != nil {
		_ = c.keyspacePS.Close()
		c.keyspacePS = nil
	}

	// Clear old handlers to prevent memory leak and duplicate events
	c.eventHandlers = []func(types.KeyspaceEvent){handler}

	// Snapshot db for goroutine
	db := c.db
	handlers := c.eventHandlers

	channel := "__keyspace@" + strconv.Itoa(db) + "__:" + pattern
	if isCluster {
		c.keyspacePS = cluster.PSubscribe(ctx, channel)
	} else {
		c.keyspacePS = client.PSubscribe(ctx, channel)
	}

	kctx, cancel := context.WithCancel(ctx)
	c.cancelKeyspace = cancel
	ps := c.keyspacePS
	c.mu.Unlock()

	go func() {
		ch := ps.Channel()
		for {
			select {
			case <-kctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				event := types.KeyspaceEvent{
					Timestamp: time.Now(),
					DB:        db,
					Event:     msg.Payload,
					Key:       strings.TrimPrefix(msg.Channel, "__keyspace@"+strconv.Itoa(db)+"__:"),
				}
				for _, h := range handlers {
					h(event)
				}
			}
		}
	}()

	return nil
}

// UnsubscribeKeyspace unsubscribes from keyspace notifications
func (c *Client) UnsubscribeKeyspace() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancelKeyspace != nil {
		c.cancelKeyspace()
		c.cancelKeyspace = nil
	}
	if c.keyspacePS != nil {
		err := c.keyspacePS.Close()
		c.keyspacePS = nil
		return err
	}
	return nil
}
