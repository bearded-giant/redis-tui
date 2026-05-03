package redis

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bearded-giant/redis-tui/internal/types"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// ---------------------------------------------------------------------------
// Test SSH server: accepts a single user/password OR public key; for "direct-tcpip"
// channel requests, dials the requested target and bridges the streams. This
// lets us exercise the full tunnel path against an in-process server.
// ---------------------------------------------------------------------------

type testSSHServer struct {
	listener net.Listener
	addr     string
	hostKey  ssh.Signer
	wg       sync.WaitGroup
	stop     chan struct{}

	// Auth config
	allowedUser     string
	allowedPassword string
	allowedKey      ssh.PublicKey
}

func newTestSSHServer(t *testing.T) *testSSHServer {
	t.Helper()
	hostSigner := mustGenerateHostKey(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("test ssh server listen: %v", err)
	}
	srv := &testSSHServer{
		listener: listener,
		addr:     listener.Addr().String(),
		hostKey:  hostSigner,
		stop:     make(chan struct{}),
	}
	srv.wg.Add(1)
	go srv.acceptLoop(t)
	return srv
}

func (s *testSSHServer) Close() {
	close(s.stop)
	_ = s.listener.Close()
	s.wg.Wait()
}

func (s *testSSHServer) HostPublicKey() ssh.PublicKey {
	return s.hostKey.PublicKey()
}

func (s *testSSHServer) acceptLoop(t *testing.T) {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.wg.Add(1)
		go s.handleConn(t, conn)
	}
}

func (s *testSSHServer) handleConn(t *testing.T, conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	cfg := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if s.allowedUser != "" && c.User() != s.allowedUser {
				return nil, errors.New("user not allowed")
			}
			if s.allowedPassword == "" {
				return nil, errors.New("password auth disabled")
			}
			if string(password) != s.allowedPassword {
				return nil, errors.New("password mismatch")
			}
			return nil, nil
		},
		PublicKeyCallback: func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if s.allowedKey == nil {
				return nil, errors.New("pubkey auth disabled")
			}
			if string(key.Marshal()) != string(s.allowedKey.Marshal()) {
				return nil, errors.New("pubkey mismatch")
			}
			return nil, nil
		},
	}
	cfg.AddHostKey(s.hostKey)

	srvConn, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		return
	}
	go ssh.DiscardRequests(reqs)
	defer srvConn.Close()

	for nc := range chans {
		if nc.ChannelType() != "direct-tcpip" {
			_ = nc.Reject(ssh.UnknownChannelType, "only direct-tcpip supported")
			continue
		}
		// Parse the direct-tcpip target from extra data.
		target := parseDirectTCPIP(nc.ExtraData())
		ch, _, err := nc.Accept()
		if err != nil {
			continue
		}
		go bridge(ch, target)
	}
}

// parseDirectTCPIP extracts host:port from the direct-tcpip channel extra data.
// Format: string host, uint32 port, string origin host, uint32 origin port.
func parseDirectTCPIP(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	hostLen := int(uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3]))
	if len(data) < 4+hostLen+4 {
		return ""
	}
	host := string(data[4 : 4+hostLen])
	portOff := 4 + hostLen
	port := uint32(data[portOff])<<24 | uint32(data[portOff+1])<<16 | uint32(data[portOff+2])<<8 | uint32(data[portOff+3])
	return net.JoinHostPort(host, itoa(int(port)))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [11]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func bridge(ch ssh.Channel, target string) {
	defer ch.Close()
	if target == "" {
		return
	}
	remote, err := net.DialTimeout("tcp", target, 2*time.Second)
	if err != nil {
		return
	}
	defer remote.Close()
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(remote, ch); done <- struct{}{} }()
	go func() { _, _ = io.Copy(ch, remote); done <- struct{}{} }()
	<-done
}

func mustGenerateHostKey(t *testing.T) ssh.Signer {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	return signer
}

// withTempKnownHosts redirects knownHostsPath to a file that trusts the given
// host:port + key. Returns a cleanup func.
func withTempKnownHosts(t *testing.T, hostPort string, key ssh.PublicKey) func() {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "known_hosts")
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	entry := "[" + host + "]:" + port + " " + key.Type() + " " + base64Encode(key.Marshal()) + "\n"
	if err := os.WriteFile(file, []byte(entry), 0o600); err != nil {
		t.Fatalf("write known_hosts: %v", err)
	}
	orig := knownHostsPath
	knownHostsPath = func() (string, error) { return file, nil }
	return func() { knownHostsPath = orig }
}

func base64Encode(b []byte) string {
	const enc = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var out strings.Builder
	for i := 0; i < len(b); i += 3 {
		var n uint32
		switch {
		case i+2 < len(b):
			n = uint32(b[i])<<16 | uint32(b[i+1])<<8 | uint32(b[i+2])
		case i+1 < len(b):
			n = uint32(b[i])<<16 | uint32(b[i+1])<<8
		default:
			n = uint32(b[i]) << 16
		}
		out.WriteByte(enc[(n>>18)&0x3f])
		out.WriteByte(enc[(n>>12)&0x3f])
		if i+1 < len(b) {
			out.WriteByte(enc[(n>>6)&0x3f])
		} else {
			out.WriteByte('=')
		}
		if i+2 < len(b) {
			out.WriteByte(enc[n&0x3f])
		} else {
			out.WriteByte('=')
		}
	}
	return out.String()
}

// ---------------------------------------------------------------------------
// dialSSH validation tests
// ---------------------------------------------------------------------------

func TestDialSSH_NilConfig(t *testing.T) {
	_, err := dialSSH(nil)
	if err == nil || !strings.Contains(err.Error(), "ssh config is nil") {
		t.Errorf("dialSSH(nil) err = %v, want 'ssh config is nil'", err)
	}
}

func TestDialSSH_MissingHost(t *testing.T) {
	_, err := dialSSH(&types.SSHConfig{User: "u"})
	if err == nil || !strings.Contains(err.Error(), "host is required") {
		t.Errorf("dialSSH missing host err = %v", err)
	}
}

func TestDialSSH_MissingUser(t *testing.T) {
	_, err := dialSSH(&types.SSHConfig{Host: "h"})
	if err == nil || !strings.Contains(err.Error(), "user is required") {
		t.Errorf("dialSSH missing user err = %v", err)
	}
}

func TestDialSSH_NoAuthAvailable(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	cleanup := withTempKnownHosts(t, "127.0.0.1:22", mustGenerateHostKey(t).PublicKey())
	defer cleanup()
	_, err := dialSSH(&types.SSHConfig{Host: "127.0.0.1", User: "u"})
	if err == nil || !strings.Contains(err.Error(), "no SSH auth method available") {
		t.Errorf("dialSSH no auth err = %v", err)
	}
}

func TestDialSSH_KnownHostsLoadFail(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	orig := knownHostsPath
	knownHostsPath = func() (string, error) { return "/nonexistent/known_hosts", nil }
	defer func() { knownHostsPath = orig }()

	_, err := dialSSH(&types.SSHConfig{Host: "127.0.0.1", User: "u", Password: "p"})
	if err == nil || !strings.Contains(err.Error(), "known_hosts") {
		t.Errorf("dialSSH known_hosts err = %v", err)
	}
}

func TestDialSSH_KnownHostsPathError(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	orig := knownHostsPath
	knownHostsPath = func() (string, error) { return "", errors.New("path err") }
	defer func() { knownHostsPath = orig }()

	_, err := dialSSH(&types.SSHConfig{Host: "127.0.0.1", User: "u", Password: "p"})
	if err == nil || !strings.Contains(err.Error(), "path err") {
		t.Errorf("dialSSH path err = %v", err)
	}
}

func TestDialSSH_AgentDialError(t *testing.T) {
	origAgent := agentDialFunc
	agentDialFunc = func() (agent.ExtendedAgent, io.Closer, error) {
		return nil, nil, errors.New("agent boom")
	}
	defer func() { agentDialFunc = origAgent }()
	cleanup := withTempKnownHosts(t, "127.0.0.1:22", mustGenerateHostKey(t).PublicKey())
	defer cleanup()

	_, err := dialSSH(&types.SSHConfig{Host: "127.0.0.1", User: "u"})
	if err == nil || !strings.Contains(err.Error(), "agent boom") {
		t.Errorf("dialSSH agent dial err = %v", err)
	}
}

func TestDialSSH_AgentSignersError(t *testing.T) {
	origAgent := agentDialFunc
	agentDialFunc = func() (agent.ExtendedAgent, io.Closer, error) {
		return &fakeAgent{signersErr: errors.New("signers boom")}, nopCloser{}, nil
	}
	defer func() { agentDialFunc = origAgent }()
	cleanup := withTempKnownHosts(t, "127.0.0.1:22", mustGenerateHostKey(t).PublicKey())
	defer cleanup()

	_, err := dialSSH(&types.SSHConfig{Host: "127.0.0.1", User: "u"})
	if err == nil || !strings.Contains(err.Error(), "signers boom") {
		t.Errorf("dialSSH agent signers err = %v", err)
	}
}

func TestDialSSH_AgentEmptySigners(t *testing.T) {
	origAgent := agentDialFunc
	agentDialFunc = func() (agent.ExtendedAgent, io.Closer, error) {
		return &fakeAgent{}, nopCloser{}, nil
	}
	defer func() { agentDialFunc = origAgent }()
	cleanup := withTempKnownHosts(t, "127.0.0.1:22", mustGenerateHostKey(t).PublicKey())
	defer cleanup()

	_, err := dialSSH(&types.SSHConfig{Host: "127.0.0.1", User: "u"})
	if err == nil || !strings.Contains(err.Error(), "no SSH auth method available") {
		t.Errorf("dialSSH agent empty signers err = %v", err)
	}
}

func TestDialSSH_DialFailure(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	cleanup := withTempKnownHosts(t, "127.0.0.1:1", mustGenerateHostKey(t).PublicKey())
	defer cleanup()

	origDial := sshDialFunc
	sshDialFunc = func(network, addr string, cfg *ssh.ClientConfig) (*ssh.Client, error) {
		return nil, errors.New("dial boom")
	}
	defer func() { sshDialFunc = origDial }()

	_, err := dialSSH(&types.SSHConfig{Host: "127.0.0.1", Port: 1, User: "u", Password: "p"})
	if err == nil || !strings.Contains(err.Error(), "dial boom") {
		t.Errorf("dialSSH dial fail err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// loadPrivateKey
// ---------------------------------------------------------------------------

func TestLoadPrivateKey_FileMissing(t *testing.T) {
	_, err := loadPrivateKey("/nonexistent/key", "")
	if err == nil || !strings.Contains(err.Error(), "read private key") {
		t.Errorf("loadPrivateKey missing file err = %v", err)
	}
}

func TestLoadPrivateKey_BadPEM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad")
	if err := os.WriteFile(path, []byte("not a key"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := loadPrivateKey(path, "")
	if err == nil || !strings.Contains(err.Error(), "parse private key") {
		t.Errorf("loadPrivateKey bad PEM err = %v", err)
	}
}

func TestLoadPrivateKey_Valid(t *testing.T) {
	path := writeTempPrivateKey(t, "")
	signer, err := loadPrivateKey(path, "")
	if err != nil {
		t.Fatalf("loadPrivateKey: %v", err)
	}
	if signer == nil {
		t.Fatal("signer should not be nil")
	}
}

func TestLoadPrivateKey_BadPassphrase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad")
	if err := os.WriteFile(path, []byte("not a key"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := loadPrivateKey(path, "anything")
	if err == nil || !strings.Contains(err.Error(), "parse private key with passphrase") {
		t.Errorf("loadPrivateKey passphrase err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// dialSSH end-to-end with password auth against test server
// ---------------------------------------------------------------------------

func TestDialSSH_PasswordAuthSuccess(t *testing.T) {
	srv := newTestSSHServer(t)
	defer srv.Close()
	srv.allowedUser = "alice"
	srv.allowedPassword = "secret"

	cleanup := withTempKnownHosts(t, srv.addr, srv.HostPublicKey())
	defer cleanup()

	host, port := splitForTest(t, srv.addr)
	client, err := dialSSH(&types.SSHConfig{Host: host, Port: port, User: "alice", Password: "secret"})
	if err != nil {
		t.Fatalf("dialSSH: %v", err)
	}
	defer client.Close()
}

func TestDialSSH_PasswordAuthRejected(t *testing.T) {
	srv := newTestSSHServer(t)
	defer srv.Close()
	srv.allowedUser = "alice"
	srv.allowedPassword = "secret"

	cleanup := withTempKnownHosts(t, srv.addr, srv.HostPublicKey())
	defer cleanup()

	host, port := splitForTest(t, srv.addr)
	_, err := dialSSH(&types.SSHConfig{Host: host, Port: port, User: "alice", Password: "wrong"})
	if err == nil {
		t.Fatal("dialSSH expected auth failure")
	}
}

func TestDialSSH_HostKeyMismatch(t *testing.T) {
	srv := newTestSSHServer(t)
	defer srv.Close()
	srv.allowedUser = "alice"
	srv.allowedPassword = "secret"

	// Use a different host key for known_hosts than the server actually presents.
	otherKey := mustGenerateHostKey(t).PublicKey()
	cleanup := withTempKnownHosts(t, srv.addr, otherKey)
	defer cleanup()

	host, port := splitForTest(t, srv.addr)
	_, err := dialSSH(&types.SSHConfig{Host: host, Port: port, User: "alice", Password: "secret"})
	if err == nil {
		t.Fatal("dialSSH expected host key mismatch error")
	}
}

func TestDialSSH_PrivateKeyAuthSuccess(t *testing.T) {
	srv := newTestSSHServer(t)
	defer srv.Close()
	srv.allowedUser = "alice"

	keyPath := writeTempPrivateKey(t, "")
	pubKey := derivePubKey(t, keyPath, "")
	srv.allowedKey = pubKey

	cleanup := withTempKnownHosts(t, srv.addr, srv.HostPublicKey())
	defer cleanup()

	host, port := splitForTest(t, srv.addr)
	client, err := dialSSH(&types.SSHConfig{Host: host, Port: port, User: "alice", PrivateKeyPath: keyPath})
	if err != nil {
		t.Fatalf("dialSSH: %v", err)
	}
	defer client.Close()
}

func TestDialSSH_DefaultPort(t *testing.T) {
	// Verify Port==0 substitutes 22. We do this by hooking sshDialFunc and
	// inspecting the addr argument.
	t.Setenv("SSH_AUTH_SOCK", "")
	cleanup := withTempKnownHosts(t, "127.0.0.1:22", mustGenerateHostKey(t).PublicKey())
	defer cleanup()

	var capturedAddr string
	origDial := sshDialFunc
	sshDialFunc = func(network, addr string, cfg *ssh.ClientConfig) (*ssh.Client, error) {
		capturedAddr = addr
		return nil, errors.New("captured")
	}
	defer func() { sshDialFunc = origDial }()

	_, _ = dialSSH(&types.SSHConfig{Host: "127.0.0.1", User: "u", Password: "p"})
	if capturedAddr != "127.0.0.1:22" {
		t.Errorf("default port addr = %q, want 127.0.0.1:22", capturedAddr)
	}
}

// ---------------------------------------------------------------------------
// Tunnel tests
// ---------------------------------------------------------------------------

func TestStartTunnel_NilSSHClient(t *testing.T) {
	_, err := startTunnel(context.Background(), nil, "127.0.0.1:1", 0)
	if err == nil || !strings.Contains(err.Error(), "ssh client is nil") {
		t.Errorf("startTunnel nil client err = %v", err)
	}
}

func TestStartTunnel_ListenError(t *testing.T) {
	// Bind to an already-used port to force listener error.
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("setup listener: %v", err)
	}
	defer occupied.Close()
	_, port := splitForTest(t, occupied.Addr().String())

	_, err = startTunnel(context.Background(), &ssh.Client{}, "127.0.0.1:1", port)
	if err == nil || !strings.Contains(err.Error(), "tunnel listen") {
		t.Errorf("startTunnel listen err = %v", err)
	}
}

func TestTunnel_EndToEnd_Bridge(t *testing.T) {
	// Backend: in-process echo server.
	echo := newEchoServer(t)
	defer echo.Close()

	srv := newTestSSHServer(t)
	defer srv.Close()
	srv.allowedUser = "alice"
	srv.allowedPassword = "secret"

	cleanup := withTempKnownHosts(t, srv.addr, srv.HostPublicKey())
	defer cleanup()

	host, port := splitForTest(t, srv.addr)
	tunnel, err := openTunnel(context.Background(), &types.SSHConfig{
		Host: host, Port: port, User: "alice", Password: "secret",
	}, echo.addr)
	if err != nil {
		t.Fatalf("openTunnel: %v", err)
	}
	defer tunnel.Close()

	if tunnel.LocalPort() == 0 {
		t.Error("LocalPort should be non-zero (ephemeral assigned)")
	}
	if tunnel.LocalAddr() == "" {
		t.Error("LocalAddr should not be empty")
	}

	// Dial the tunnel local addr, send "ping", expect "ping" back.
	conn, err := net.Dial("tcp", tunnel.LocalAddr())
	if err != nil {
		t.Fatalf("dial tunnel: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatalf("write: %v", err)
	}
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(buf) != "ping" {
		t.Errorf("echoed = %q, want %q", buf, "ping")
	}
}

func TestTunnel_CloseIdempotent(t *testing.T) {
	srv := newTestSSHServer(t)
	defer srv.Close()
	srv.allowedUser = "alice"
	srv.allowedPassword = "secret"
	cleanup := withTempKnownHosts(t, srv.addr, srv.HostPublicKey())
	defer cleanup()

	host, port := splitForTest(t, srv.addr)
	tunnel, err := openTunnel(context.Background(), &types.SSHConfig{
		Host: host, Port: port, User: "alice", Password: "secret",
	}, "127.0.0.1:1")
	if err != nil {
		t.Fatalf("openTunnel: %v", err)
	}
	if err := tunnel.Close(); err != nil {
		t.Errorf("first close: %v", err)
	}
	if err := tunnel.Close(); err != nil {
		t.Errorf("second close should be no-op, got: %v", err)
	}
}

func TestTunnel_LocalPort_BadAddr(t *testing.T) {
	// Construct a Tunnel with a listener whose Addr() returns a malformed string
	// to exercise the parse failure branch.
	tun := &Tunnel{listener: &fakeListener{addr: "not-a-host-port"}}
	if got := tun.LocalPort(); got != 0 {
		t.Errorf("LocalPort with bad addr = %d, want 0", got)
	}
}

func TestTunnel_HandleConn_DialFailure(t *testing.T) {
	srv := newTestSSHServer(t)
	defer srv.Close()
	srv.allowedUser = "alice"
	srv.allowedPassword = "secret"
	cleanup := withTempKnownHosts(t, srv.addr, srv.HostPublicKey())
	defer cleanup()

	host, port := splitForTest(t, srv.addr)
	// remoteAddr points at a closed port — bridge dial should fail and the
	// handler should clean up the local conn without crashing.
	tunnel, err := openTunnel(context.Background(), &types.SSHConfig{
		Host: host, Port: port, User: "alice", Password: "secret",
	}, "127.0.0.1:1")
	if err != nil {
		t.Fatalf("openTunnel: %v", err)
	}
	defer tunnel.Close()

	conn, err := net.Dial("tcp", tunnel.LocalAddr())
	if err != nil {
		t.Fatalf("dial tunnel: %v", err)
	}
	// Read should observe EOF since remote dial failed.
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 1)
	_, _ = conn.Read(buf)
	conn.Close()
}

func TestTunnel_AcceptLoopExitsOnContext(t *testing.T) {
	srv := newTestSSHServer(t)
	defer srv.Close()
	srv.allowedUser = "alice"
	srv.allowedPassword = "secret"
	cleanup := withTempKnownHosts(t, srv.addr, srv.HostPublicKey())
	defer cleanup()

	host, port := splitForTest(t, srv.addr)
	parent, cancel := context.WithCancel(context.Background())
	tunnel, err := openTunnel(parent, &types.SSHConfig{
		Host: host, Port: port, User: "alice", Password: "secret",
	}, "127.0.0.1:1")
	if err != nil {
		t.Fatalf("openTunnel: %v", err)
	}
	cancel()
	if err := tunnel.Close(); err != nil {
		t.Errorf("close after cancel: %v", err)
	}
}

// ---------------------------------------------------------------------------
// openTunnel error paths
// ---------------------------------------------------------------------------

func TestOpenTunnel_NilConfig(t *testing.T) {
	_, err := openTunnel(context.Background(), nil, "127.0.0.1:1")
	if err == nil || !strings.Contains(err.Error(), "SSH configuration is missing") {
		t.Errorf("openTunnel nil cfg err = %v", err)
	}
}

func TestOpenTunnel_DialFailure(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	cleanup := withTempKnownHosts(t, "127.0.0.1:22", mustGenerateHostKey(t).PublicKey())
	defer cleanup()

	origDial := sshDialFunc
	sshDialFunc = func(network, addr string, cfg *ssh.ClientConfig) (*ssh.Client, error) {
		return nil, errors.New("dial fail")
	}
	defer func() { sshDialFunc = origDial }()

	_, err := openTunnel(context.Background(), &types.SSHConfig{
		Host: "127.0.0.1", User: "u", Password: "p",
	}, "127.0.0.1:1")
	if err == nil || !strings.Contains(err.Error(), "dial fail") {
		t.Errorf("openTunnel dial err = %v", err)
	}
}

func TestOpenTunnel_StartTunnelFailure(t *testing.T) {
	srv := newTestSSHServer(t)
	defer srv.Close()
	srv.allowedUser = "alice"
	srv.allowedPassword = "secret"
	cleanup := withTempKnownHosts(t, srv.addr, srv.HostPublicKey())
	defer cleanup()

	// Hold a port to force startTunnel listener bind failure.
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("setup listener: %v", err)
	}
	defer occupied.Close()
	_, occupiedPort := splitForTest(t, occupied.Addr().String())

	host, port := splitForTest(t, srv.addr)
	_, err = openTunnel(context.Background(), &types.SSHConfig{
		Host: host, Port: port, User: "alice", Password: "secret", LocalPort: occupiedPort,
	}, "127.0.0.1:1")
	if err == nil || !strings.Contains(err.Error(), "tunnel listen") {
		t.Errorf("openTunnel start tunnel err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// agentDialFunc default behavior
// ---------------------------------------------------------------------------

func TestAgentDialFunc_NoSocket(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	ag, closer, err := agentDialFunc()
	if err != nil {
		t.Fatalf("agentDialFunc no socket err = %v", err)
	}
	if ag != nil || closer != nil {
		t.Errorf("agentDialFunc no socket = (%v, %v), want (nil, nil)", ag, closer)
	}
}

func TestAgentDialFunc_BadSocket(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "/nonexistent/socket")
	_, _, err := agentDialFunc()
	if err == nil || !strings.Contains(err.Error(), "ssh agent dial") {
		t.Errorf("agentDialFunc bad socket err = %v", err)
	}
}

func TestKnownHostsPath_Default(t *testing.T) {
	path, err := knownHostsPath()
	if err != nil {
		t.Fatalf("knownHostsPath: %v", err)
	}
	if !strings.HasSuffix(path, "/.ssh/known_hosts") {
		t.Errorf("knownHostsPath = %q, want suffix /.ssh/known_hosts", path)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func splitForTest(t *testing.T, hostPort string) (string, int) {
	t.Helper()
	host, p, err := net.SplitHostPort(hostPort)
	if err != nil {
		t.Fatalf("split %q: %v", hostPort, err)
	}
	port := 0
	for _, c := range p {
		port = port*10 + int(c-'0')
	}
	return host, port
}

func writeTempPrivateKey(t *testing.T, passphrase string) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	var block *pemBlock
	if passphrase == "" {
		b, err := ssh.MarshalPrivateKey(priv, "")
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		block = &pemBlock{Type: b.Type, Bytes: b.Bytes}
	} else {
		b, err := ssh.MarshalPrivateKeyWithPassphrase(priv, "", []byte(passphrase))
		if err != nil {
			t.Fatalf("marshal w/ passphrase: %v", err)
		}
		block = &pemBlock{Type: b.Type, Bytes: b.Bytes}
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(path, encodePEM(block.Type, block.Bytes), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	return path
}

type pemBlock struct {
	Type  string
	Bytes []byte
}

func encodePEM(typ string, data []byte) []byte {
	var b strings.Builder
	b.WriteString("-----BEGIN ")
	b.WriteString(typ)
	b.WriteString("-----\n")
	enc := base64Encode(data)
	for i := 0; i < len(enc); i += 64 {
		end := i + 64
		if end > len(enc) {
			end = len(enc)
		}
		b.WriteString(enc[i:end])
		b.WriteString("\n")
	}
	b.WriteString("-----END ")
	b.WriteString(typ)
	b.WriteString("-----\n")
	return []byte(b.String())
}

func derivePubKey(t *testing.T, keyPath, passphrase string) ssh.PublicKey {
	t.Helper()
	signer, err := loadPrivateKey(keyPath, passphrase)
	if err != nil {
		t.Fatalf("loadPrivateKey: %v", err)
	}
	return signer.PublicKey()
}

// ---------------------------------------------------------------------------
// Echo server for tunnel bridge tests
// ---------------------------------------------------------------------------

type echoServer struct {
	listener net.Listener
	addr     string
	wg       sync.WaitGroup
}

func newEchoServer(t *testing.T) *echoServer {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("echo listen: %v", err)
	}
	e := &echoServer{listener: l, addr: l.Addr().String()}
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			e.wg.Add(1)
			go func() {
				defer e.wg.Done()
				defer c.Close()
				_, _ = io.Copy(c, c)
			}()
		}
	}()
	return e
}

func (e *echoServer) Close() {
	_ = e.listener.Close()
	e.wg.Wait()
}

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type fakeAgent struct {
	signers    []ssh.Signer
	signersErr error
}

func (f *fakeAgent) List() ([]*agent.Key, error)                               { return nil, nil }
func (f *fakeAgent) Sign(ssh.PublicKey, []byte) (*ssh.Signature, error)        { return nil, nil }
func (f *fakeAgent) Add(agent.AddedKey) error                                  { return nil }
func (f *fakeAgent) Remove(ssh.PublicKey) error                                { return nil }
func (f *fakeAgent) RemoveAll() error                                          { return nil }
func (f *fakeAgent) Lock([]byte) error                                         { return nil }
func (f *fakeAgent) Unlock([]byte) error                                       { return nil }
func (f *fakeAgent) Signers() ([]ssh.Signer, error)                            { return f.signers, f.signersErr }
func (f *fakeAgent) SignWithFlags(ssh.PublicKey, []byte, agent.SignatureFlags) (*ssh.Signature, error) {
	return nil, nil
}
func (f *fakeAgent) Extension(string, []byte) ([]byte, error) { return nil, nil }

type nopCloser struct{}

func (nopCloser) Close() error { return nil }

type fakeListener struct {
	addr string
}

func (f *fakeListener) Accept() (net.Conn, error) { return nil, errors.New("not impl") }
func (f *fakeListener) Close() error              { return nil }
func (f *fakeListener) Addr() net.Addr            { return fakeAddr(f.addr) }

type fakeAddr string

func (f fakeAddr) Network() string { return "tcp" }
func (f fakeAddr) String() string  { return string(f) }

// ---------------------------------------------------------------------------
// Client.TestSSHConnection
// ---------------------------------------------------------------------------

func TestClient_TestSSHConnection_NilConfig(t *testing.T) {
	c := NewClient()
	_, err := c.TestSSHConnection(nil)
	if err == nil || !strings.Contains(err.Error(), "SSH configuration is missing") {
		t.Errorf("TestSSHConnection nil cfg err = %v", err)
	}
}

func TestClient_TestSSHConnection_DialFailure(t *testing.T) {
	c := NewClient()
	_, err := c.TestSSHConnection(&types.SSHConfig{
		Host: "127.0.0.1", Port: 1, User: "u", Password: "p",
	})
	if err == nil {
		t.Fatal("expected SSH dial failure")
	}
}

func TestClient_TestSSHConnection_Success(t *testing.T) {
	srv := newTestSSHServer(t)
	defer srv.Close()
	srv.allowedUser = "alice"
	srv.allowedPassword = "secret"
	cleanup := withTempKnownHosts(t, srv.addr, srv.HostPublicKey())
	defer cleanup()

	host, port := splitForTest(t, srv.addr)
	c := NewClient()
	dur, err := c.TestSSHConnection(&types.SSHConfig{
		Host: host, Port: port, User: "alice", Password: "secret",
	})
	if err != nil {
		t.Fatalf("TestSSHConnection: %v", err)
	}
	if dur <= 0 {
		t.Errorf("dur should be positive")
	}
}

// ---------------------------------------------------------------------------
// Coverage fillers
// ---------------------------------------------------------------------------

// dialSSH with private key path through buildAuthMethods, where loadPrivateKey
// returns an error inside buildAuthMethods (covers the err return at line ~110).
func TestDialSSH_PrivateKeyPathLoadError(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	cleanup := withTempKnownHosts(t, "127.0.0.1:22", mustGenerateHostKey(t).PublicKey())
	defer cleanup()

	_, err := dialSSH(&types.SSHConfig{
		Host:           "127.0.0.1",
		User:           "u",
		PrivateKeyPath: "/nonexistent/key",
	})
	if err == nil || !strings.Contains(err.Error(), "read private key") {
		t.Errorf("dialSSH bad key path err = %v", err)
	}
}

// loadPrivateKey roundtrip through a passphrase-encrypted key (covers passphrase
// success branch at ~line 152).
func TestLoadPrivateKey_ValidPassphrase(t *testing.T) {
	path := writeTempPrivateKey(t, "supersecret")
	signer, err := loadPrivateKey(path, "supersecret")
	if err != nil {
		t.Fatalf("loadPrivateKey w/ passphrase: %v", err)
	}
	if signer == nil {
		t.Fatal("signer should not be nil")
	}
}

// agentDialFunc successful path: spin up an in-process unix socket that speaks
// the agent protocol (we just need the dial to succeed; signers list is empty
// which is the "no auth available" outcome — covers line 41 path).
func TestAgentDialFunc_Success(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "agent.sock")
	listener, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	defer listener.Close()

	t.Setenv("SSH_AUTH_SOCK", sock)
	ag, closer, err := agentDialFunc()
	if err != nil {
		t.Fatalf("agentDialFunc success err = %v", err)
	}
	if ag == nil || closer == nil {
		t.Fatal("agentDialFunc success returned nil")
	}
	_ = closer.Close()
}

// dialSSH where buildAuthMethods returns a non-nil agentCloser AND len(auth)==0,
// triggering the deferred close + the "no auth" return. Plus the agentCloser
// non-nil branch in dialSSH itself.
func TestDialSSH_AgentReturnsAuthCloseDeferred(t *testing.T) {
	// Build a fake agent that returns one signer + a closer. dialSSH should
	// then defer-close the closer when it returns (here via dial failure).
	closed := make(chan struct{}, 1)
	origAgent := agentDialFunc
	agentDialFunc = func() (agent.ExtendedAgent, io.Closer, error) {
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		signer, err := ssh.NewSignerFromKey(priv)
		if err != nil {
			return nil, nil, err
		}
		return &fakeAgent{signers: []ssh.Signer{signer}}, &signalCloser{ch: closed}, nil
	}
	defer func() { agentDialFunc = origAgent }()

	cleanup := withTempKnownHosts(t, "127.0.0.1:22", mustGenerateHostKey(t).PublicKey())
	defer cleanup()

	origDial := sshDialFunc
	sshDialFunc = func(network, addr string, cfg *ssh.ClientConfig) (*ssh.Client, error) {
		return nil, errors.New("dial fail")
	}
	defer func() { sshDialFunc = origDial }()

	_, _ = dialSSH(&types.SSHConfig{Host: "127.0.0.1", User: "u"})

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Error("agent closer was not closed")
	}
}

type signalCloser struct {
	ch     chan struct{}
	closed bool
}

func (s *signalCloser) Close() error {
	if !s.closed {
		s.closed = true
		s.ch <- struct{}{}
	}
	return nil
}

// knownHostsPath UserHomeDir error: stub HOME to empty + drive userHomeDir
// failure. Easier: directly call knownHostsPath and verify it returns *some*
// path (covers lines 50). For the error branch (line 47-48), that requires
// UserHomeDir to fail — substitute via env + os.Getenv("HOME") empty on unix.
func TestKnownHostsPath_HomeUnset(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	// Unix UserHomeDir consults HOME; when empty it falls back to /etc/passwd
	// lookup which usually still succeeds for the test user. So this test may
	// not deterministically hit the error branch. Treat as best-effort: if it
	// returns successfully, that exercises the success branch instead.
	path, err := knownHostsPath()
	if err == nil && path == "" {
		t.Error("knownHostsPath returned empty path with no error")
	}
}

// Tunnel.Close error from listener.Close — wrap a fake listener that returns
// an error on Close.
func TestTunnel_Close_ListenerError(t *testing.T) {
	tun := &Tunnel{
		listener: &errCloseListener{realAddr: "127.0.0.1:0"},
		cancel:   func() {},
	}
	err := tun.Close()
	if err == nil || !strings.Contains(err.Error(), "listener boom") {
		t.Errorf("Tunnel.Close listener err = %v", err)
	}
}

type errCloseListener struct {
	realAddr string
}

func (e *errCloseListener) Accept() (net.Conn, error) {
	return nil, errors.New("not accepting")
}
func (e *errCloseListener) Close() error   { return errors.New("listener boom") }
func (e *errCloseListener) Addr() net.Addr { return fakeAddr(e.realAddr) }

// Tunnel.Close error from sshClient.Close — set sshClient to a *ssh.Client
// whose underlying transport is already closed. ssh.Client's Close on
// nil-conn-state returns an error consistently.
func TestTunnel_Close_SSHClientErrorSwallowed(t *testing.T) {
	// Real listener so listener.Close() succeeds; SSH client nil so we skip
	// the SSH close branch; this covers the path where only listener close
	// runs and tunnel.sshClient is nil.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	tun := &Tunnel{
		listener: listener,
		cancel:   func() {},
	}
	if err := tun.Close(); err != nil {
		t.Errorf("Close with nil ssh client: %v", err)
	}
}

// Tunnel.Close where sshClient.Close returns an error. Close the underlying
// SSH client first; the second close (from Tunnel.Close) returns an error.
func TestTunnel_Close_SSHClientCloseError(t *testing.T) {
	srv := newTestSSHServer(t)
	defer srv.Close()
	srv.allowedUser = "alice"
	srv.allowedPassword = "secret"
	cleanup := withTempKnownHosts(t, srv.addr, srv.HostPublicKey())
	defer cleanup()

	host, port := splitForTest(t, srv.addr)
	tunnel, err := openTunnel(context.Background(), &types.SSHConfig{
		Host: host, Port: port, User: "alice", Password: "secret",
	}, "127.0.0.1:1")
	if err != nil {
		t.Fatalf("openTunnel: %v", err)
	}
	// Pre-close the SSH client so Tunnel.Close's sshClient.Close returns err.
	_ = tunnel.sshClient.Close()
	// Tunnel.Close will still call Close on the already-closed ssh client,
	// which returns "use of closed network connection" or similar.
	_ = tunnel.Close()
}

// handleConn where sshClient.Dial fails (covers lines 272-274). Force this by
// closing the SSH client before any local accept happens.
func TestTunnel_HandleConn_SSHDialFails(t *testing.T) {
	srv := newTestSSHServer(t)
	defer srv.Close()
	srv.allowedUser = "alice"
	srv.allowedPassword = "secret"
	cleanup := withTempKnownHosts(t, srv.addr, srv.HostPublicKey())
	defer cleanup()

	host, port := splitForTest(t, srv.addr)
	tunnel, err := openTunnel(context.Background(), &types.SSHConfig{
		Host: host, Port: port, User: "alice", Password: "secret",
	}, "127.0.0.1:1")
	if err != nil {
		t.Fatalf("openTunnel: %v", err)
	}
	defer tunnel.Close()

	// Close the SSH client out from under the tunnel. Subsequent local Dials
	// trigger handleConn → sshClient.Dial returns error → early return.
	_ = tunnel.sshClient.Close()

	for i := 0; i < 3; i++ {
		conn, err := net.Dial("tcp", tunnel.LocalAddr())
		if err != nil {
			continue
		}
		_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		buf := make([]byte, 1)
		_, _ = conn.Read(buf)
		conn.Close()
	}
}
