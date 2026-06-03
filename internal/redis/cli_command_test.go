package redis

import (
	"strings"
	"testing"

	"github.com/bearded-giant/redis-tui/internal/types"
)

func TestBuildCLICommand_StringKeyDefaults(t *testing.T) {
	conn := types.Connection{Host: "redis.example", Port: 6379}
	key := types.RedisKey{Key: "user:42", Type: types.KeyTypeString}
	got := BuildCLICommand(conn, key)
	want := "redis-cli -h 'redis.example' GET 'user:42'"
	if got != want {
		t.Errorf("got\n  %q\nwant\n  %q", got, want)
	}
}

func TestBuildCLICommand_NonDefaultPortAndDB(t *testing.T) {
	conn := types.Connection{Host: "h", Port: 6380, DB: 3}
	got := BuildCLICommand(conn, types.RedisKey{Key: "k", Type: types.KeyTypeString})
	if !strings.Contains(got, "-p 6380") {
		t.Errorf("expected -p 6380 in %q", got)
	}
	if !strings.Contains(got, "-n 3") {
		t.Errorf("expected -n 3 in %q", got)
	}
}

func TestBuildCLICommand_AuthFlags(t *testing.T) {
	conn := types.Connection{Host: "h", Port: 6379, Username: "alice", Password: "s3cr3t"}
	got := BuildCLICommand(conn, types.RedisKey{Key: "k", Type: types.KeyTypeString})
	if !strings.Contains(got, "--user 'alice'") {
		t.Errorf("expected --user 'alice' in %q", got)
	}
	if !strings.Contains(got, "-a 's3cr3t'") {
		t.Errorf("expected -a 's3cr3t' in %q", got)
	}
}

func TestBuildCLICommand_Cluster(t *testing.T) {
	conn := types.Connection{Host: "h", Port: 6379, UseCluster: true}
	got := BuildCLICommand(conn, types.RedisKey{Key: "k", Type: types.KeyTypeString})
	if !strings.Contains(got, " -c ") {
		t.Errorf("expected -c flag in %q", got)
	}
}

func TestBuildCLICommand_TLSWithFiles(t *testing.T) {
	conn := types.Connection{
		Host: "h", Port: 6379, UseTLS: true,
		TLSConfig: &types.TLSConfig{
			CertFile:           "/tls/client.crt",
			KeyFile:            "/tls/client.key",
			CAFile:             "/tls/ca.crt",
			InsecureSkipVerify: true,
			ServerName:         "redis.example",
		},
	}
	got := BuildCLICommand(conn, types.RedisKey{Key: "k", Type: types.KeyTypeString})
	for _, want := range []string{"--tls", "--cert '/tls/client.crt'", "--key '/tls/client.key'", "--cacert '/tls/ca.crt'", "--insecure", "--sni 'redis.example'"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in %q", want, got)
		}
	}
}

func TestBuildCLICommand_SSHCommentPrepended(t *testing.T) {
	conn := types.Connection{
		Host: "redis.internal", Port: 6379, UseSSH: true,
		SSHConfig: &types.SSHConfig{Host: "bastion.example", Port: 22, User: "deploy"},
	}
	got := BuildCLICommand(conn, types.RedisKey{Key: "k", Type: types.KeyTypeString})
	lines := strings.SplitN(got, "\n", 2)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d in %q", len(lines), got)
	}
	if !strings.HasPrefix(lines[0], "# requires SSH tunnel:") {
		t.Errorf("expected SSH comment first line, got %q", lines[0])
	}
	if !strings.Contains(lines[0], "deploy@bastion.example") {
		t.Errorf("expected user@host in comment, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "redis-cli -h 'redis.internal'") {
		t.Errorf("expected command targeting Redis host, got %q", lines[1])
	}
}

func TestCliOpFor_PerType(t *testing.T) {
	cases := map[types.KeyType]string{
		types.KeyTypeString:      "GET 'k'",
		types.KeyTypeList:        "LRANGE 'k' 0 -1",
		types.KeyTypeSet:         "SMEMBERS 'k'",
		types.KeyTypeZSet:        "ZRANGE 'k' 0 -1 WITHSCORES",
		types.KeyTypeHash:        "HGETALL 'k'",
		types.KeyTypeStream:      "XRANGE 'k' - +",
		types.KeyTypeJSON:        "JSON.GET 'k'",
		types.KeyTypeHyperLogLog: "PFCOUNT 'k'",
		types.KeyTypeBitmap:      "BITCOUNT 'k'",
	}
	for typ, want := range cases {
		got := strings.Join(cliOpFor(typ, "k"), " ")
		if got != want {
			t.Errorf("type %s: got %q want %q", typ, got, want)
		}
	}
}

func TestShellQuote_EscapesSingleQuote(t *testing.T) {
	got := shellQuote("it's")
	want := `'it'\''s'`
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
