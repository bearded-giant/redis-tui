package redis

import (
	"context"
	"sync"

	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/redis/go-redis/v9"
)

// silentLogger discards all log output from the Redis client
type silentLogger struct{}

func (l *silentLogger) Printf(ctx context.Context, format string, v ...interface{}) {}

func init() {
	// Disable go-redis internal logging to prevent noisy connection pool messages
	redis.SetLogger(&silentLogger{})
}

// Client wraps the Redis client with additional functionality
type Client struct {
	client  *redis.Client
	cluster *redis.ClusterClient
	ctx     context.Context

	host     string
	port     int
	password string
	db       int

	isCluster     bool
	pubsub        *redis.PubSub
	keyspacePS    *redis.PubSub
	eventHandlers []func(types.KeyspaceEvent)
}

func (c *Client) cmdable() redis.Cmdable {
	if c.isCluster {
		return c.cluster
	}
	return c.client
}

func (c *Client) pipeline() redis.Pipeliner {
	if c.isCluster {
		return c.cluster.Pipeline()
	}
	return c.client.Pipeline()
}

// scanAll scans all keys matching pattern. In cluster mode, scans all master
// nodes via ForEachMaster so keys from every shard are returned.
func (c *Client) scanAll(pattern string, batchSize int64) ([]string, error) {
	if c.isCluster {
		var mu sync.Mutex
		var allKeys []string
		err := c.cluster.ForEachMaster(c.ctx, func(ctx context.Context, client *redis.Client) error {
			var cursor uint64
			for {
				keys, nextCursor, err := client.Scan(ctx, cursor, pattern, batchSize).Result()
				if err != nil {
					return err
				}
				if len(keys) > 0 {
					mu.Lock()
					allKeys = append(allKeys, keys...)
					mu.Unlock()
				}
				cursor = nextCursor
				if cursor == 0 {
					break
				}
			}
			return nil
		})
		return allKeys, err
	}

	var allKeys []string
	var cursor uint64
	for {
		keys, nextCursor, err := c.cmdable().Scan(c.ctx, cursor, pattern, batchSize).Result()
		if err != nil {
			return allKeys, err
		}
		allKeys = append(allKeys, keys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return allKeys, nil
}

// NewClient creates a new Redis client wrapper
func NewClient() *Client {
	return &Client{
		ctx:           context.Background(),
		eventHandlers: []func(types.KeyspaceEvent){},
	}
}
