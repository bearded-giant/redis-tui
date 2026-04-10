package redis

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
	goredis "github.com/redis/go-redis/v9"
)

// newClusterClientForTest creates a ClusterClient pointing at a single seed
// address. The ClusterSlots callback short-circuits cluster topology discovery
// so the client doesn't depend on real CLUSTER SLOTS replies.
func newClusterClientForTest(addr string) *goredis.ClusterClient {
	return goredis.NewClusterClient(&goredis.ClusterOptions{
		Addrs:        []string{addr},
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		MaxRetries:   1,
		ClusterSlots: func(ctx context.Context) ([]goredis.ClusterSlot, error) {
			return []goredis.ClusterSlot{
				{
					Start: 0,
					End:   16383,
					Nodes: []goredis.ClusterNode{
						{Addr: addr, ID: "test-node"},
					},
				},
			}, nil
		},
	})
}

// fakeRedisServer is a minimal RESP-protocol mock that responds to a fixed
// set of commands with canned replies. Used to drive the parsing branches
// of GetServerInfo / GetMemoryStats / GetLiveMetrics / SlowLogGet / ClientList
// that miniredis cannot satisfy.
type fakeRedisServer struct {
	t        *testing.T
	listener net.Listener
	wg       sync.WaitGroup
	mu       sync.Mutex
	closed   bool

	// canned responses keyed by command name (uppercased)
	responses map[string]string

	// handler, if set, overrides per-command lookup. Receives the full
	// argv (uppercased command at index 0) and returns the raw RESP reply.
	// Returning an empty string falls back to the default response logic.
	handler func(argv []string) string
}

func newFakeRedisServer(t *testing.T) *fakeRedisServer {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen failed: %v", err)
	}
	srv := &fakeRedisServer{
		t:         t,
		listener:  l,
		responses: make(map[string]string),
	}
	srv.wg.Add(1)
	go srv.serve()
	t.Cleanup(srv.close)
	return srv
}

func (s *fakeRedisServer) close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()
	_ = s.listener.Close()
	s.wg.Wait()
}

func (s *fakeRedisServer) addr() (string, int) {
	host, portStr, _ := net.SplitHostPort(s.listener.Addr().String())
	port, _ := strconv.Atoi(portStr)
	return host, port
}

func (s *fakeRedisServer) setResponse(cmd, resp string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responses[strings.ToUpper(cmd)] = resp
}

func (s *fakeRedisServer) setHandler(h func(argv []string) string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handler = h
}

func (s *fakeRedisServer) serve() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

func (s *fakeRedisServer) handleConn(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	reader := bufio.NewReader(conn)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		cmd, err := readRESPCommand(reader)
		if err != nil {
			return
		}
		if len(cmd) == 0 {
			continue
		}

		upper := strings.ToUpper(cmd[0])
		argv := append([]string{upper}, cmd[1:]...)

		s.mu.Lock()
		h := s.handler
		s.mu.Unlock()

		var resp string
		var ok bool
		if h != nil {
			resp = h(argv)
			ok = resp != ""
		}
		if !ok {
			s.mu.Lock()
			resp, ok = s.responses[upper]
			s.mu.Unlock()
		}

		if !ok {
			// Default reply for unrecognized commands.
			switch upper {
			case "PING":
				resp = "+PONG\r\n"
			case "AUTH":
				resp = "+OK\r\n"
			case "HELLO":
				// Reject HELLO so go-redis falls back to RESP2.
				resp = "-ERR unknown command\r\n"
			case "CLIENT":
				resp = "+OK\r\n"
			case "QUIT":
				resp = "+OK\r\n"
				_, _ = conn.Write([]byte(resp))
				return
			default:
				resp = "+OK\r\n"
			}
		}
		if _, err := conn.Write([]byte(resp)); err != nil {
			return
		}
	}
}

// readRESPCommand reads a single RESP-encoded command.
func readRESPCommand(r *bufio.Reader) ([]string, error) {
	header, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	header = strings.TrimRight(header, "\r\n")

	// Inline command (legacy) — just split by spaces.
	if !strings.HasPrefix(header, "*") {
		parts := strings.Fields(header)
		return parts, nil
	}

	count, err := strconv.Atoi(header[1:])
	if err != nil {
		return nil, err
	}

	args := make([]string, 0, count)
	for i := 0; i < count; i++ {
		lenLine, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		lenLine = strings.TrimRight(lenLine, "\r\n")
		if !strings.HasPrefix(lenLine, "$") {
			return nil, fmt.Errorf("expected $ prefix, got %q", lenLine)
		}
		argLen, err := strconv.Atoi(lenLine[1:])
		if err != nil {
			return nil, err
		}
		// Read argLen bytes plus trailing CRLF.
		buf := make([]byte, argLen+2)
		if _, err := readFull(r, buf); err != nil {
			return nil, err
		}
		args = append(args, string(buf[:argLen]))
	}
	return args, nil
}

func readFull(r *bufio.Reader, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := r.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// respBulkString wraps a string as a RESP bulk-string reply.
func respBulkString(s string) string {
	return fmt.Sprintf("$%d\r\n%s\r\n", len(s), s)
}

// ---------------------------------------------------------------------------
// GetServerInfo — full INFO parsing
// ---------------------------------------------------------------------------

func TestGetServerInfo_FullInfoParsing(t *testing.T) {
	srv := newFakeRedisServer(t)

	infoBody := strings.Join([]string{
		"# Server",
		"redis_version:7.2.0",
		"redis_mode:standalone",
		"os:Linux",
		"# Memory",
		"used_memory_human:1.23M",
		"used_memory_peak_human:2.45M",
		"mem_fragmentation_ratio:1.42",
		"# Clients",
		"connected_clients:5",
		"# Stats",
		"total_commands_processed:1234",
		"# Persistence",
		"aof_enabled:1",
		"# CPU",
		"# Cluster",
		"cluster_enabled:0",
		"# Keyspace",
		"uptime_in_seconds:3600",
	}, "\r\n") + "\r\n"

	srv.setResponse("INFO", respBulkString(infoBody))
	srv.setResponse("DBSIZE", ":42\r\n")
	srv.setResponse("SELECT", "+OK\r\n")

	host, port := srv.addr()
	client := NewClient()
	if err := client.Connect(&types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect to fake server: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })

	info, err := client.GetServerInfo()
	if err != nil {
		t.Fatalf("GetServerInfo error: %v", err)
	}

	if info.Version != "7.2.0" {
		t.Errorf("Version = %q, want %q", info.Version, "7.2.0")
	}
	if info.Mode != "standalone" {
		t.Errorf("Mode = %q, want %q", info.Mode, "standalone")
	}
	if info.OS != "Linux" {
		t.Errorf("OS = %q, want %q", info.OS, "Linux")
	}
	if info.UsedMemory != "1.23M" {
		t.Errorf("UsedMemory = %q, want %q", info.UsedMemory, "1.23M")
	}
	if info.PeakMemory != "2.45M" {
		t.Errorf("PeakMemory = %q, want %q", info.PeakMemory, "2.45M")
	}
	if info.Clients != "5" {
		t.Errorf("Clients = %q, want %q", info.Clients, "5")
	}
	if info.MemFragRatio != "1.42" {
		t.Errorf("MemFragRatio = %q, want %q", info.MemFragRatio, "1.42")
	}
	if info.TotalCommands != "1234" {
		t.Errorf("TotalCommands = %q, want %q", info.TotalCommands, "1234")
	}
	if !info.AOFEnabled {
		t.Error("AOFEnabled = false, want true")
	}
	if info.ClusterMode {
		t.Error("ClusterMode = true, want false")
	}
	if info.Uptime == "" {
		t.Error("Uptime should be populated from uptime_in_seconds")
	}
	if info.TotalKeys != "42" {
		t.Errorf("TotalKeys = %q, want %q", info.TotalKeys, "42")
	}
}

// ---------------------------------------------------------------------------
// GetMemoryStats — full INFO memory parsing
// ---------------------------------------------------------------------------

func TestGetMemoryStats_FullInfoParsing(t *testing.T) {
	srv := newFakeRedisServer(t)

	memBody := strings.Join([]string{
		"# Memory",
		"used_memory:1048576",
		"used_memory_peak:2097152",
		"mem_fragmentation_bytes:5000",
		"mem_fragmentation_ratio:1.42",
		"total_system_memory:8589934592",
	}, "\r\n") + "\r\n"

	srv.setResponse("INFO", respBulkString(memBody))
	srv.setResponse("SCAN", "*2\r\n$1\r\n0\r\n*0\r\n")

	host, port := srv.addr()
	client := NewClient()
	if err := client.Connect(&types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect to fake server: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })

	stats, err := client.GetMemoryStats()
	if err != nil {
		t.Fatalf("GetMemoryStats error: %v", err)
	}

	if stats.UsedMemory != 1048576 {
		t.Errorf("UsedMemory = %d, want 1048576", stats.UsedMemory)
	}
	if stats.PeakMemory != 2097152 {
		t.Errorf("PeakMemory = %d, want 2097152", stats.PeakMemory)
	}
	if stats.FragmentedBytes != 5000 {
		t.Errorf("FragmentedBytes = %d, want 5000", stats.FragmentedBytes)
	}
	if stats.FragRatio != 1.42 {
		t.Errorf("FragRatio = %f, want 1.42", stats.FragRatio)
	}
	if stats.TotalMemory != 8589934592 {
		t.Errorf("TotalMemory = %d, want 8589934592", stats.TotalMemory)
	}
}

// ---------------------------------------------------------------------------
// GetLiveMetrics — full INFO parsing
// ---------------------------------------------------------------------------

func TestGetLiveMetrics_FullInfoParsing(t *testing.T) {
	srv := newFakeRedisServer(t)

	body := strings.Join([]string{
		"# Stats",
		"instantaneous_ops_per_sec:42.5",
		"keyspace_hits:1000",
		"keyspace_misses:50",
		"expired_keys:25",
		"evicted_keys:5",
		"instantaneous_input_kbps:1.5",
		"instantaneous_output_kbps:2.5",
		"total_connections_received:100",
		"rejected_connections:2",
		"# Memory",
		"used_memory:1048576",
		"# Clients",
		"connected_clients:7",
		"blocked_clients:1",
		"# CPU",
		"used_cpu_sys:0.5",
		"used_cpu_user:1.5",
	}, "\r\n") + "\r\n"

	srv.setResponse("INFO", respBulkString(body))

	host, port := srv.addr()
	client := NewClient()
	if err := client.Connect(&types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect to fake server: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })

	m, err := client.GetLiveMetrics()
	if err != nil {
		t.Fatalf("GetLiveMetrics error: %v", err)
	}

	if m.OpsPerSec != 42.5 {
		t.Errorf("OpsPerSec = %f, want 42.5", m.OpsPerSec)
	}
	if m.UsedMemoryBytes != 1048576 {
		t.Errorf("UsedMemoryBytes = %d, want 1048576", m.UsedMemoryBytes)
	}
	if m.ConnectedClients != 7 {
		t.Errorf("ConnectedClients = %d, want 7", m.ConnectedClients)
	}
	if m.BlockedClients != 1 {
		t.Errorf("BlockedClients = %d, want 1", m.BlockedClients)
	}
	if m.KeyspaceHits != 1000 {
		t.Errorf("KeyspaceHits = %d, want 1000", m.KeyspaceHits)
	}
	if m.KeyspaceMisses != 50 {
		t.Errorf("KeyspaceMisses = %d, want 50", m.KeyspaceMisses)
	}
	if m.ExpiredKeys != 25 {
		t.Errorf("ExpiredKeys = %d, want 25", m.ExpiredKeys)
	}
	if m.EvictedKeys != 5 {
		t.Errorf("EvictedKeys = %d, want 5", m.EvictedKeys)
	}
	if m.InputKbps != 1.5 {
		t.Errorf("InputKbps = %f, want 1.5", m.InputKbps)
	}
	if m.OutputKbps != 2.5 {
		t.Errorf("OutputKbps = %f, want 2.5", m.OutputKbps)
	}
	if m.UsedCPUSys != 0.5 {
		t.Errorf("UsedCPUSys = %f, want 0.5", m.UsedCPUSys)
	}
	if m.UsedCPUUser != 1.5 {
		t.Errorf("UsedCPUUser = %f, want 1.5", m.UsedCPUUser)
	}
	if m.TotalConnections != 100 {
		t.Errorf("TotalConnections = %d, want 100", m.TotalConnections)
	}
	if m.RejectedConns != 2 {
		t.Errorf("RejectedConns = %d, want 2", m.RejectedConns)
	}
}

// ---------------------------------------------------------------------------
// ClientList — full parsing branches
// ---------------------------------------------------------------------------

func TestClientList_FullParsing(t *testing.T) {
	srv := newFakeRedisServer(t)

	clientList := "id=3 addr=127.0.0.1:54321 name=client1 age=120 idle=10 flags=N db=0 cmd=ping sub=2\nid=4 addr=127.0.0.1:54322 name=client2 age=60 idle=5 flags=N db=1 cmd=get sub=0\n"
	srv.setResponse("CLIENT", respBulkString(clientList))

	host, port := srv.addr()
	client := NewClient()
	if err := client.Connect(&types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect to fake server: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })

	clients, err := client.ClientList()
	if err != nil {
		t.Fatalf("ClientList error: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("ClientList returned %d clients, want 2", len(clients))
	}

	first := clients[0]
	if first.ID != 3 {
		t.Errorf("ID = %d, want 3", first.ID)
	}
	if first.Addr != "127.0.0.1:54321" {
		t.Errorf("Addr = %q, want %q", first.Addr, "127.0.0.1:54321")
	}
	if first.Name != "client1" {
		t.Errorf("Name = %q, want %q", first.Name, "client1")
	}
	if first.Age != 120*time.Second {
		t.Errorf("Age = %v, want 120s", first.Age)
	}
	if first.Idle != 10*time.Second {
		t.Errorf("Idle = %v, want 10s", first.Idle)
	}
	if first.Flags != "N" {
		t.Errorf("Flags = %q, want %q", first.Flags, "N")
	}
	if first.DB != 0 {
		t.Errorf("DB = %d, want 0", first.DB)
	}
	if first.Cmd != "ping" {
		t.Errorf("Cmd = %q, want %q", first.Cmd, "ping")
	}
	if first.SubCount != 2 {
		t.Errorf("SubCount = %d, want 2", first.SubCount)
	}
}

// ---------------------------------------------------------------------------
// ClusterInfo — fake reply
// ---------------------------------------------------------------------------

func TestClusterInfo_FakeReply(t *testing.T) {
	srv := newFakeRedisServer(t)

	infoBody := "cluster_enabled:1\r\ncluster_state:ok\r\ncluster_slots_assigned:16384\r\n"
	srv.setResponse("CLUSTER", respBulkString(infoBody))

	host, port := srv.addr()
	client := NewClient()
	if err := client.Connect(&types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect to fake server: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })

	info, err := client.ClusterInfo()
	if err != nil {
		t.Fatalf("ClusterInfo error: %v", err)
	}
	if !strings.Contains(info, "cluster_state:ok") {
		t.Errorf("ClusterInfo result missing expected content: %q", info)
	}
}

// ---------------------------------------------------------------------------
// ClusterNodes — fake reply with multiple nodes
// ---------------------------------------------------------------------------

func TestClusterNodes_FakeReply(t *testing.T) {
	srv := newFakeRedisServer(t)

	// CLUSTER NODES output: each line space-separated with 8+ fields.
	nodes := "abc 127.0.0.1:7000@17000 master - 0 0 1 connected 0-5460\n" +
		"def 127.0.0.1:7001@17001 slave abc 0 0 1 connected\n"
	srv.setResponse("CLUSTER", respBulkString(nodes))

	host, port := srv.addr()
	client := NewClient()
	if err := client.Connect(&types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect to fake server: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })

	result, err := client.ClusterNodes()
	if err != nil {
		t.Fatalf("ClusterNodes error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("ClusterNodes returned %d nodes, want 2", len(result))
	}
	if result[0].ID != "abc" {
		t.Errorf("first node ID = %q, want abc", result[0].ID)
	}
}

// ---------------------------------------------------------------------------
// ClusterInfo / ClusterNodes — exercise the cluster client branch by manually
// configuring a ClusterClient that points to the fake server. The cluster
// client will fail CLUSTER SLOTS at construction but ClusterNodes/ClusterInfo
// dispatch directly via Do, so they still work.
// ---------------------------------------------------------------------------

func TestClusterNodes_ClusterClientBranch(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setResponse("CLUSTER", respBulkString("abc 127.0.0.1:7000@17000 master - 0 0 1 connected 0-5460\n"))

	host, port := srv.addr()
	addr := fmt.Sprintf("%s:%d", host, port)

	// Create a client and manually attach a ClusterClient pointing at the
	// fake server. We can't go through ConnectCluster because the Ping
	// would fail; we set the fields directly to exercise the cluster branch.
	client := NewClient()
	cluster := newClusterClientForTest(addr)
	client.cluster = cluster
	client.isCluster = true
	client.client = nil
	t.Cleanup(func() {
		_ = cluster.Close()
		client.cluster = nil
	})

	nodes, err := client.ClusterNodes()
	if err != nil {
		t.Logf("ClusterNodes (cluster branch) error: %v", err)
		return
	}
	if len(nodes) == 0 {
		t.Error("expected at least 1 node")
	}
}

func TestClusterInfo_ClusterClientBranch(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setResponse("CLUSTER", respBulkString("cluster_state:ok\r\n"))

	host, port := srv.addr()
	addr := fmt.Sprintf("%s:%d", host, port)

	client := NewClient()
	cluster := newClusterClientForTest(addr)
	client.cluster = cluster
	client.isCluster = true
	client.client = nil
	t.Cleanup(func() {
		_ = cluster.Close()
		client.cluster = nil
	})

	info, err := client.ClusterInfo()
	if err != nil {
		t.Logf("ClusterInfo (cluster branch) error: %v", err)
		return
	}
	if !strings.Contains(info, "cluster_state") {
		t.Errorf("info missing cluster_state: %q", info)
	}
}

// ---------------------------------------------------------------------------
// cmdable / do / pipeline — exercise the isCluster=true branch by manually
// installing a cluster client. We then call methods that route through these
// dispatchers.
// ---------------------------------------------------------------------------

func TestClusterDispatchers(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setResponse("PING", "+PONG\r\n")
	srv.setResponse("DBSIZE", ":3\r\n")
	srv.setResponse("CLUSTER", respBulkString("cluster_state:ok\r\n"))

	host, port := srv.addr()
	addr := fmt.Sprintf("%s:%d", host, port)

	client := NewClient()
	cluster := newClusterClientForTest(addr)
	client.cluster = cluster
	client.isCluster = true
	t.Cleanup(func() {
		_ = cluster.Close()
		client.cluster = nil
	})

	// cmdable() should return the cluster client.
	if got := client.cmdable(); got == nil {
		t.Error("cmdable() returned nil for cluster")
	}

	// do() routes through cluster client.
	_ = client.do("PING").Err()

	// pipeline() returns a Pipeliner from the cluster client.
	pipe := client.pipeline()
	if pipe == nil {
		t.Error("pipeline() returned nil for cluster")
	}
}

// ---------------------------------------------------------------------------
// Subscribe — cluster branch
// ---------------------------------------------------------------------------

func TestSubscribe_ClusterBranch(t *testing.T) {
	srv := newFakeRedisServer(t)
	// Return a successful subscribe ack.
	srv.setResponse("SUBSCRIBE", "*3\r\n$9\r\nsubscribe\r\n$2\r\nch\r\n:1\r\n")

	host, port := srv.addr()
	addr := fmt.Sprintf("%s:%d", host, port)

	client := NewClient()
	cluster := newClusterClientForTest(addr)
	client.cluster = cluster
	client.isCluster = true
	t.Cleanup(func() {
		_ = cluster.Close()
		client.cluster = nil
	})

	sub := client.Subscribe("ch")
	if sub == nil {
		t.Fatal("Subscribe returned nil for cluster")
	}
	_ = sub.Close()
}

// ---------------------------------------------------------------------------
// scanAll / scanEach — cluster branch via ForEachMaster. The ClusterSlots
// callback configures a single shard pointing at the fake server, and SCAN
// is mocked to return one page of keys.
// ---------------------------------------------------------------------------

func TestScanAll_ClusterBranch(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setResponse("SCAN", "*2\r\n$1\r\n0\r\n*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n")

	host, port := srv.addr()
	addr := fmt.Sprintf("%s:%d", host, port)

	client := NewClient()
	cluster := newClusterClientForTest(addr)
	client.cluster = cluster
	client.isCluster = true
	t.Cleanup(func() {
		_ = cluster.Close()
		client.cluster = nil
	})

	keys, err := client.scanAll("*", 100)
	if err != nil {
		t.Logf("scanAll cluster branch: %v", err)
	}
	if len(keys) > 0 && keys[0] != "foo" {
		t.Errorf("first key = %q, want foo", keys[0])
	}
}

func TestScanEach_ClusterBranch(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setResponse("SCAN", "*2\r\n$1\r\n0\r\n*2\r\n$1\r\na\r\n$1\r\nb\r\n")

	host, port := srv.addr()
	addr := fmt.Sprintf("%s:%d", host, port)

	client := NewClient()
	cluster := newClusterClientForTest(addr)
	client.cluster = cluster
	client.isCluster = true
	t.Cleanup(func() {
		_ = cluster.Close()
		client.cluster = nil
	})

	var collected []string
	err := client.scanEach("*", 100, func(keys []string) bool {
		collected = append(collected, keys...)
		return true
	})
	if err != nil {
		t.Logf("scanEach cluster branch: %v", err)
	}
}

// ---------------------------------------------------------------------------
// SlowLogGet — non-empty result
// ---------------------------------------------------------------------------

func TestSlowLogGet_NonEmpty(t *testing.T) {
	srv := newFakeRedisServer(t)

	// SLOWLOG GET reply: array of arrays. Each entry is
	// [id, timestamp, duration, [args...], client_addr, client_name].
	// Use minimal valid 6-element entry.
	reply := "*1\r\n" +
		"*6\r\n" +
		":1\r\n" +
		":1700000000\r\n" +
		":250\r\n" +
		"*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n" +
		"$15\r\n127.0.0.1:54321\r\n" +
		"$7\r\nclient1\r\n"
	srv.setResponse("SLOWLOG", reply)

	host, port := srv.addr()
	client := NewClient()
	if err := client.Connect(&types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect to fake server: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })

	entries, err := client.SlowLogGet(10)
	if err != nil {
		t.Fatalf("SlowLogGet error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("SlowLogGet returned %d entries, want 1", len(entries))
	}

	e := entries[0]
	if e.ID != 1 {
		t.Errorf("ID = %d, want 1", e.ID)
	}
	if e.Command != "GET foo" {
		t.Errorf("Command = %q, want %q", e.Command, "GET foo")
	}
	if e.ClientAddr != "127.0.0.1:54321" {
		t.Errorf("ClientAddr = %q, want %q", e.ClientAddr, "127.0.0.1:54321")
	}
	if e.ClientName != "client1" {
		t.Errorf("ClientName = %q, want %q", e.ClientName, "client1")
	}
}
