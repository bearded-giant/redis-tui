package redis

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// cleanup closes existing connections before establishing a new one
func (c *Client) cleanup() {
	c.mu.Lock()
	_ = c.disconnectLocked()
	c.mu.Unlock()
}

// Connect establishes a connection to Redis
func (c *Client) Connect(host string, port int, password string, db int) error {
	c.cleanup()

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 3,
		MaxRetries:   3,
	})

	c.mu.Lock()
	c.host = host
	c.port = port
	c.password = password
	c.db = db
	c.client = client
	ctx := c.ctx
	c.mu.Unlock()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := client.Ping(pingCtx).Result()
	return err
}

// ConnectWithTLS establishes a TLS connection to Redis
func (c *Client) ConnectWithTLS(host string, port int, password string, db int, tlsConfig *tls.Config) error {
	c.cleanup()

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 3,
		MaxRetries:   3,
		TLSConfig:    tlsConfig,
	})

	c.mu.Lock()
	c.host = host
	c.port = port
	c.password = password
	c.db = db
	c.client = client
	ctx := c.ctx
	c.mu.Unlock()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := client.Ping(pingCtx).Result()
	return err
}

// ConnectCluster establishes a connection to a Redis cluster
func (c *Client) ConnectCluster(addrs []string, password string) error {
	c.cleanup()

	// Parse first address for display purposes
	seedHost := "127.0.0.1"
	host := seedHost
	port := 6379
	if len(addrs) > 0 {
		host, port = parseAddr(addrs[0])
		seedHost = host
	}

	cluster := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        addrs,
		Password:     password,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 3,
		MaxRetries:   3,
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			_, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			return net.DialTimeout(network, net.JoinHostPort(seedHost, port), 5*time.Second)
		},
	})

	c.mu.Lock()
	c.isCluster = true
	c.password = password
	c.host = host
	c.port = port
	c.cluster = cluster
	ctx := c.ctx
	c.mu.Unlock()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
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
func (c *Client) TestConnection(host string, port int, password string, db int) (time.Duration, error) {
	testClient := redis.NewClient(&redis.Options{
		Addr:        fmt.Sprintf("%s:%d", host, port),
		Password:    password,
		DB:          db,
		DialTimeout: 5 * time.Second,
	})
	defer testClient.Close()

	start := time.Now()
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	_, err := testClient.Ping(ctx).Result()
	return time.Since(start), err
}
