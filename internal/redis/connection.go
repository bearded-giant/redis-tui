package redis

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/redis/go-redis/v9"
)

const (
	defaultDialTimeout  = 5 * time.Second
	defaultReadTimeout  = 3 * time.Second
	defaultWriteTimeout = 3 * time.Second
	defaultPoolSize     = 10
	defaultMinIdleConns = 3
	defaultMaxRetries   = 3
	defaultPingTimeout  = 5 * time.Second
)

func defaultOptions(conn types.Connection) (*redis.Options, error) {
	if conn.Host == "" {
		return nil, errors.New("host is required")
	}
	if conn.Port == 0 {
		return nil, errors.New("port is required")
	}

	return &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", conn.Host, conn.Port),
		Password:     conn.Password,
		DB:           conn.DB,
		DialTimeout:  defaultDialTimeout,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		PoolSize:     defaultPoolSize,
		MinIdleConns: defaultMinIdleConns,
		MaxRetries:   defaultMaxRetries,
	}, nil
}

// cleanup closes existing connections before establishing a new one
func (c *Client) cleanup() {
	c.mu.Lock()
	_ = c.disconnectLocked()
	c.mu.Unlock()
}

// Connect establishes a connection to Redis
func (c *Client) Connect(conn types.Connection) error {
	c.cleanup()

	opts, optErr := defaultOptions(conn)
	if optErr != nil {
		return optErr
	}

	if conn.UseTLS {
		if conn.TLSConfig == nil {
			return fmt.Errorf("TLS requested but TLS configuration is missing")
		}
		tlsCfg, err := conn.TLSConfig.BuildTLSConfig()
		if err != nil {
			return err
		}
		opts.TLSConfig = tlsCfg
	}
	client := redis.NewClient(opts)

	c.mu.Lock()
	c.host = conn.Host
	c.port = conn.Port
	c.password = conn.Password
	c.db = conn.DB
	c.client = client
	ctx := c.ctx
	c.mu.Unlock()

	pingCtx, cancel := context.WithTimeout(ctx, defaultPingTimeout)
	defer cancel()

	_, err := client.Ping(pingCtx).Result()
	return err
}

// ConnectCluster establishes a connection to a Redis cluster
func (c *Client) ConnectCluster(addrs []string, conn types.Connection) error {
	c.cleanup()

	// Parse first address for display purposes
	seedHost := "127.0.0.1"
	host := seedHost
	port := 6379
	if len(addrs) > 0 {
		host, port = parseAddr(addrs[0])
		seedHost = host
	}

	opts := &redis.ClusterOptions{
		Addrs:        addrs,
		Password:     conn.Password,
		DialTimeout:  defaultDialTimeout,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		PoolSize:     defaultPoolSize,
		MinIdleConns: defaultMinIdleConns,
		MaxRetries:   defaultMaxRetries,
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			_, p, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			return net.DialTimeout(network, net.JoinHostPort(seedHost, p), defaultDialTimeout)
		},
	}

	if conn.UseTLS {
		if conn.TLSConfig == nil {
			return fmt.Errorf("TLS requested but TLS configuration is missing")
		}
		tlsCfg, err := conn.TLSConfig.BuildTLSConfig()
		if err != nil {
			return fmt.Errorf("failed to build TLS config: %w", err)
		}
		opts.TLSConfig = tlsCfg
	}

	cluster := redis.NewClusterClient(opts)

	c.mu.Lock()
	c.isCluster = true
	c.password = conn.Password
	c.host = host
	c.port = port
	c.cluster = cluster
	ctx := c.ctx
	c.mu.Unlock()

	pingCtx, cancel := context.WithTimeout(ctx, defaultPingTimeout)
	defer cancel()

	_, err := cluster.Ping(pingCtx).Result()
	return err
}

func parseAddr(addr string) (string, int) {
	host := addr
	port := 6379
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		host = addr[:i]
		if p, err := strconv.Atoi(addr[i+1:]); err == nil {
			port = p
		}
	}
	return host, port
}

// Disconnect closes the Redis connection
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.disconnectLocked()
}

func (c *Client) disconnectLocked() error {
	var errs []error
	if c.cancelKeyspace != nil {
		c.cancelKeyspace()
		c.cancelKeyspace = nil
	}
	if c.keyspacePS != nil {
		errs = append(errs, c.keyspacePS.Close())
		c.keyspacePS = nil
	}
	if c.pubsub != nil {
		errs = append(errs, c.pubsub.Close())
		c.pubsub = nil
	}
	if c.cluster != nil {
		errs = append(errs, c.cluster.Close())
		c.cluster = nil
	}
	if c.client != nil {
		errs = append(errs, c.client.Close())
		c.client = nil
	}
	c.isCluster = false
	c.eventHandlers = nil
	return errors.Join(errs...)
}

// IsCluster returns whether connected to a cluster
func (c *Client) IsCluster() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isCluster
}

// SelectDB switches the database
func (c *Client) SelectDB(db int) error {
	c.mu.RLock()
	isCluster := c.isCluster
	client := c.client
	ctx := c.ctx
	c.mu.RUnlock()

	if isCluster {
		return fmt.Errorf("database selection not supported in cluster mode")
	}
	if client == nil {
		return fmt.Errorf("not connected")
	}
	if err := client.Do(ctx, "SELECT", db).Err(); err != nil {
		return err
	}
	c.mu.Lock()
	c.db = db
	c.mu.Unlock()
	return nil
}

// TestConnection tests a connection
func (c *Client) TestConnection(conn types.Connection) (time.Duration, error) {
	opts, optErr := defaultOptions(conn)
	if optErr != nil {
		return 0, optErr
	}

	if conn.UseTLS {
		if conn.TLSConfig == nil {
			return 0, fmt.Errorf("TLS requested but TLS configuration is missing")
		}
		tlsCfg, err := conn.TLSConfig.BuildTLSConfig()
		if err != nil {
			return 0, err
		}
		opts.TLSConfig = tlsCfg
	}
	testClient := redis.NewClient(opts)
	defer testClient.Close()

	start := time.Now()
	ctx, cancel := context.WithTimeout(c.ctx, defaultPingTimeout)
	defer cancel()

	_, err := testClient.Ping(ctx).Result()
	return time.Since(start), err
}
