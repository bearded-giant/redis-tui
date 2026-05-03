package redis

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/bearded-giant/redis-tui/internal/types"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

const (
	defaultSSHPort        = 22
	defaultSSHTimeout     = 10 * time.Second
	defaultTunnelLoopback = "127.0.0.1"
)

// sshDialFunc is the seam for ssh.Dial, swapped in tests.
var sshDialFunc = ssh.Dial

// agentDialFunc returns an SSH agent client. Returns nil agent + nil error
// when SSH_AUTH_SOCK is unset (treated as "agent unavailable").
var agentDialFunc = func() (agent.ExtendedAgent, io.Closer, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, nil, nil
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, nil, fmt.Errorf("ssh agent dial: %w", err)
	}
	return agent.NewClient(conn), conn, nil
}

// knownHostsPath returns the path to the user's known_hosts file.
var knownHostsPath = func() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home dir: %w", err)
	}
	return filepath.Join(home, ".ssh", "known_hosts"), nil
}

// dialSSH establishes an SSH connection using auth precedence:
// private key (with optional passphrase) → password → SSH agent.
func dialSSH(cfg *types.SSHConfig) (*ssh.Client, error) {
	if cfg == nil {
		return nil, errors.New("ssh config is nil")
	}
	if cfg.Host == "" {
		return nil, errors.New("ssh host is required")
	}
	if cfg.User == "" {
		return nil, errors.New("ssh user is required")
	}

	port := cfg.Port
	if port == 0 {
		port = defaultSSHPort
	}

	auth, agentCloser, err := buildAuthMethods(cfg)
	if err != nil {
		return nil, err
	}
	if agentCloser != nil {
		defer agentCloser.Close()
	}
	if len(auth) == 0 {
		return nil, errors.New("no SSH auth method available: provide private key, password, or run ssh-agent")
	}

	hostKeyCallback, err := buildHostKeyCallback()
	if err != nil {
		return nil, err
	}

	clientCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            auth,
		HostKeyCallback: hostKeyCallback,
		Timeout:         defaultSSHTimeout,
	}

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))
	client, err := sshDialFunc("tcp", addr, clientCfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", addr, err)
	}
	return client, nil
}

// buildAuthMethods applies precedence: private key → password → agent.
// Returns (methods, agentCloser, error). agentCloser is non-nil only when
// the agent connection was opened and must be closed by the caller.
func buildAuthMethods(cfg *types.SSHConfig) ([]ssh.AuthMethod, io.Closer, error) {
	var methods []ssh.AuthMethod

	if cfg.PrivateKeyPath != "" {
		signer, err := loadPrivateKey(cfg.PrivateKeyPath, cfg.Passphrase)
		if err != nil {
			return nil, nil, err
		}
		methods = append(methods, ssh.PublicKeys(signer))
		return methods, nil, nil
	}

	if cfg.Password != "" {
		methods = append(methods, ssh.Password(cfg.Password))
		return methods, nil, nil
	}

	ag, closer, err := agentDialFunc()
	if err != nil {
		return nil, nil, err
	}
	if ag == nil {
		return methods, nil, nil
	}
	signers, err := ag.Signers()
	if err != nil {
		_ = closer.Close()
		return nil, nil, fmt.Errorf("ssh agent signers: %w", err)
	}
	if len(signers) == 0 {
		_ = closer.Close()
		return methods, nil, nil
	}
	methods = append(methods, ssh.PublicKeys(signers...))
	return methods, closer, nil
}

func loadPrivateKey(path, passphrase string) (ssh.Signer, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is user-configured
	if err != nil {
		return nil, fmt.Errorf("read private key %s: %w", path, err)
	}
	if passphrase != "" {
		signer, err := ssh.ParsePrivateKeyWithPassphrase(data, []byte(passphrase))
		if err != nil {
			return nil, fmt.Errorf("parse private key with passphrase: %w", err)
		}
		return signer, nil
	}
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return signer, nil
}

func buildHostKeyCallback() (ssh.HostKeyCallback, error) {
	path, err := knownHostsPath()
	if err != nil {
		return nil, err
	}
	cb, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("load known_hosts %s: %w", path, err)
	}
	return cb, nil
}

// Tunnel is a local-listener SSH port forward.
// It accepts connections on a loopback port and forwards each through an
// SSH client to a remote address (the redis target).
type Tunnel struct {
	listener   net.Listener
	sshClient  *ssh.Client
	remoteAddr string
	cancel     context.CancelFunc
	wg         sync.WaitGroup

	mu     sync.Mutex
	closed bool
}

// LocalAddr returns the address the listener is bound to (host:port).
func (t *Tunnel) LocalAddr() string {
	return t.listener.Addr().String()
}

// LocalPort returns just the assigned port.
func (t *Tunnel) LocalPort() int {
	_, p, err := net.SplitHostPort(t.listener.Addr().String())
	if err != nil {
		return 0
	}
	port, _ := strconv.Atoi(p)
	return port
}

// Close stops the accept loop, closes the listener, and tears down the SSH client.
// Idempotent.
func (t *Tunnel) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	t.mu.Unlock()

	t.cancel()
	var errs []error
	if err := t.listener.Close(); err != nil {
		errs = append(errs, err)
	}
	t.wg.Wait()
	if t.sshClient != nil {
		if err := t.sshClient.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// startTunnel binds a loopback listener on localPort (0 = ephemeral) and
// starts an accept loop forwarding each conn through sshClient to remoteAddr.
// Caller owns the returned Tunnel and must Close it.
func startTunnel(parentCtx context.Context, sshClient *ssh.Client, remoteAddr string, localPort int) (*Tunnel, error) {
	if sshClient == nil {
		return nil, errors.New("ssh client is nil")
	}
	bind := net.JoinHostPort(defaultTunnelLoopback, strconv.Itoa(localPort))
	listener, err := net.Listen("tcp", bind)
	if err != nil {
		return nil, fmt.Errorf("tunnel listen %s: %w", bind, err)
	}

	ctx, cancel := context.WithCancel(parentCtx)
	t := &Tunnel{
		listener:   listener,
		sshClient:  sshClient,
		remoteAddr: remoteAddr,
		cancel:     cancel,
	}

	t.wg.Add(1)
	go t.acceptLoop(ctx)

	return t, nil
}

func (t *Tunnel) acceptLoop(ctx context.Context) {
	defer t.wg.Done()
	for {
		local, err := t.listener.Accept()
		if err != nil {
			// Listener closed or fatal accept error — exit loop.
			return
		}
		t.wg.Add(1)
		go t.handleConn(ctx, local)
	}
}

func (t *Tunnel) handleConn(ctx context.Context, local net.Conn) {
	defer t.wg.Done()
	defer local.Close()

	remote, err := t.sshClient.Dial("tcp", t.remoteAddr)
	if err != nil {
		return
	}
	defer remote.Close()

	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(remote, local)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(local, remote)
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}
}
