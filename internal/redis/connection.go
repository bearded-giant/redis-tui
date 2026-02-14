package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// cleanup closes existing connections before establishing a new one
func (c *Client) cleanup() {
	if c.keyspacePS != nil {
		_ = c.keyspacePS.Close()
		c.keyspacePS = nil
	}
	if c.pubsub != nil {
		_ = c.pubsub.Close()
		c.pubsub = nil
	}
	if c.cluster != nil {
		_ = c.cluster.Close()
		c.cluster = nil
	}
	if c.client != nil {
		_ = c.client.Close()
		c.client = nil
	}
	c.isCluster = false
}

// Connect establishes a connection to Redis
func (c *Client) Connect(host string, port int, password string, db int) error {
	c.cleanup()

	c.host = host
	c.port = port
	c.password = password
	c.db = db

	c.client = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	_, err := c.client.Ping(ctx).Result()
	return err
}

// ConnectWithTLS establishes a TLS connection to Redis
func (c *Client) ConnectWithTLS(host string, port int, password string, db int, tlsConfig *tls.Config) error {
	c.cleanup()

	c.host = host
	c.port = port
	c.password = password
	c.db = db

	c.client = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		TLSConfig:    tlsConfig,
	})

	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	_, err := c.client.Ping(ctx).Result()
	return err
}

// ConnectCluster establishes a connection to a Redis cluster
func (c *Client) ConnectCluster(addrs []string, password string) error {
	c.cleanup()

	c.isCluster = true
	c.password = password

	// Parse first address for display purposes
	seedHost := "127.0.0.1"
	if len(addrs) > 0 {
		host, port := parseAddr(addrs[0])
		c.host = host
		c.port = port
		seedHost = host
	}

	c.cluster = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        addrs,
		Password:     password,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		// Remap discovered cluster node addresses to the seed host.
		// Cluster nodes advertise internal IPs (e.g. Docker bridge IPs)
		// that may be unreachable from the client. This dialer keeps the
		// port but replaces the host with the original seed host.
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			_, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			return net.DialTimeout(network, net.JoinHostPort(seedHost, port), 5*time.Second)
		},
	})

	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	_, err := c.cluster.Ping(ctx).Result()
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
	if c.pubsub != nil {
		_ = c.pubsub.Close()
	}
	if c.keyspacePS != nil {
		_ = c.keyspacePS.Close()
	}
	var err error
	if c.cluster != nil {
		err = c.cluster.Close()
	}
	if c.client != nil {
		err = c.client.Close()
	}
	c.isCluster = false
	return err
}

// IsCluster returns whether connected to a cluster
func (c *Client) IsCluster() bool {
	return c.isCluster
}

// SelectDB switches the database
func (c *Client) SelectDB(db int) error {
	if c.isCluster {
		return fmt.Errorf("database selection not supported in cluster mode")
	}
	if c.client == nil {
		return fmt.Errorf("not connected")
	}
	if err := c.client.Do(c.ctx, "SELECT", db).Err(); err != nil {
		return err
	}
	c.db = db
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
