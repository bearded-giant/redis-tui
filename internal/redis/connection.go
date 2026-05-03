package redis

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/bearded-giant/redis-tui/internal/types"
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
		Username:     conn.Username,
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

	var tunnel *Tunnel
	if conn.UseSSH {
		t, err := openTunnel(c.ctx, conn.SSHConfig, fmt.Sprintf("%s:%d", conn.Host, conn.Port))
		if err != nil {
			return err
		}
		tunnel = t
		opts.Addr = t.LocalAddr()
	}

	client := redis.NewClient(opts)

	c.mu.Lock()
	c.host = conn.Host
	c.port = conn.Port
	c.username = conn.Username
	c.password = conn.Password
	c.db = conn.DB
	c.client = client
	c.tunnel = tunnel
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

	// Parse first address for display purposes and for the Dialer below.
	seedHost := "127.0.0.1"
	host := seedHost
	port := 6379
	if len(addrs) > 0 {
		host, port = parseAddr(addrs[0])
		seedHost = host
	}

	var tunnel *Tunnel
	dialerHost := seedHost
	dialerPort := 0
	if conn.UseSSH {
		// Tunnel forwards a single local port to seed addr. Cluster Dialer
		// remaps every node's advertised addr to the seed host, then dials
		// the local listener instead of the seed. Bastion must reach all
		// cluster nodes for this to work.
		seedAddr := net.JoinHostPort(seedHost, strconv.Itoa(port))
		t, err := openTunnel(c.ctx, conn.SSHConfig, seedAddr)
		if err != nil {
			return err
		}
		tunnel = t
		dialerHost = defaultTunnelLoopback
		dialerPort = t.LocalPort()
	}

	opts := &redis.ClusterOptions{
		Addrs:        addrs,
		Username:     conn.Username,
		Password:     conn.Password,
		DialTimeout:  defaultDialTimeout,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		PoolSize:     defaultPoolSize,
		MinIdleConns: defaultMinIdleConns,
		MaxRetries:   defaultMaxRetries,
		// Remap cluster node addresses to the seed host. Cluster nodes
		// (especially in Docker) advertise internal IPs that may not be
		// reachable from the client. Keep the port from each node but
		// route through the original host. When SSH is on, route through
		// the local tunnel listener instead.
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			_, p, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			targetPort := p
			if dialerPort != 0 {
				targetPort = strconv.Itoa(dialerPort)
			}
			return net.DialTimeout(network, net.JoinHostPort(dialerHost, targetPort), defaultDialTimeout)
		},
	}

	if conn.UseTLS {
		if conn.TLSConfig == nil {
			if tunnel != nil {
				_ = tunnel.Close()
			}
			return fmt.Errorf("TLS requested but TLS configuration is missing")
		}
		tlsCfg, err := conn.TLSConfig.BuildTLSConfig()
		if err != nil {
			if tunnel != nil {
				_ = tunnel.Close()
			}
			return fmt.Errorf("failed to build TLS config: %w", err)
		}
		opts.TLSConfig = tlsCfg
	}

	cluster := redis.NewClusterClient(opts)

	c.mu.Lock()
	c.isCluster = true
	c.username = conn.Username
	c.password = conn.Password
	c.host = host
	c.port = port
	c.cluster = cluster
	c.tunnel = tunnel
	ctx := c.ctx
	c.mu.Unlock()

	pingCtx, cancel := context.WithTimeout(ctx, defaultPingTimeout)
	defer cancel()

	_, err := cluster.Ping(pingCtx).Result()
	return err
}

func parseAddr(addr string) (string, int) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		// No port separator or invalid format — treat whole string as host.
		return strings.TrimSpace(addr), 6379
	}
	p, err := strconv.Atoi(portStr)
	if err != nil {
		return host, 6379
	}
	return host, p
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
	// Close tunnel after redis clients so in-flight writes can drain.
	// Tunnel.Close() also closes the underlying SSH client.
	if c.tunnel != nil {
		errs = append(errs, c.tunnel.Close())
		c.tunnel = nil
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

	var tunnel *Tunnel
	if conn.UseSSH {
		t, err := openTunnel(c.ctx, conn.SSHConfig, fmt.Sprintf("%s:%d", conn.Host, conn.Port))
		if err != nil {
			return 0, err
		}
		tunnel = t
		opts.Addr = t.LocalAddr()
		defer tunnel.Close()
	}

	testClient := redis.NewClient(opts)
	defer testClient.Close()

	start := time.Now()
	ctx, cancel := context.WithTimeout(c.ctx, defaultPingTimeout)
	defer cancel()

	_, err := testClient.Ping(ctx).Result()
	return time.Since(start), err
}

// TestSSHConnection verifies SSH connectivity standalone — dials the bastion
// and tears down. Does not connect to redis. Useful as a UI test step before
// committing a connection record.
func (c *Client) TestSSHConnection(sshCfg *types.SSHConfig) (time.Duration, error) {
	if sshCfg == nil {
		return 0, fmt.Errorf("SSH configuration is missing")
	}
	start := time.Now()
	client, err := dialSSH(sshCfg)
	if err != nil {
		return 0, err
	}
	_ = client.Close()
	return time.Since(start), nil
}

// openTunnel dials SSH and starts a local-listener tunnel to remoteAddr.
// Returns the tunnel; caller owns its lifecycle.
func openTunnel(ctx context.Context, sshCfg *types.SSHConfig, remoteAddr string) (*Tunnel, error) {
	if sshCfg == nil {
		return nil, fmt.Errorf("SSH requested but SSH configuration is missing")
	}
	sshClient, err := dialSSH(sshCfg)
	if err != nil {
		return nil, err
	}
	tunnel, err := startTunnel(ctx, sshClient, remoteAddr, sshCfg.LocalPort)
	if err != nil {
		_ = sshClient.Close()
		return nil, err
	}
	return tunnel, nil
}
