package redis

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bearded-giant/redis-tui/internal/types"
)

// BuildCLICommand returns a shell-quoted `redis-cli` command line that
// connects with the same parameters as conn and reads the given key.
//
// Notes on what is and isn't included:
//   - Password, if set on conn, is rendered as `-a 'literal'`. The caller is
//     responsible for redacting before sharing.
//   - SSH tunnel config is not translated. When conn.UseSSH is true we prepend
//     a `# requires SSH tunnel` comment so the recipient knows the redis host
//     is not directly reachable. The command itself still targets conn.Host so
//     the user can swap in their own tunnel/localhost as they see fit.
//   - TLS cert/key/CA paths are rendered as-is. Paths are local to the user
//     who copied the command.
func BuildCLICommand(conn types.Connection, key types.RedisKey) string {
	parts := connectionFlags(conn)
	parts = append([]string{"redis-cli"}, parts...)
	op := cliOpFor(key.Type, key.Key)
	parts = append(parts, op...)

	line := strings.Join(parts, " ")
	if conn.UseSSH && conn.SSHConfig != nil {
		return sshComment(conn) + "\n" + line
	}
	return line
}

// BuildScanCLICommand returns a `redis-cli --scan` shell line that reproduces
// the current pattern filter from the key list. Pattern may be empty for a full
// scan; quoted via shellQuote so glob chars survive shell expansion.
func BuildScanCLICommand(conn types.Connection, pattern string) string {
	parts := connectionFlags(conn)
	parts = append([]string{"redis-cli"}, parts...)
	parts = append(parts, "--scan")
	if pattern != "" {
		parts = append(parts, "--pattern", shellQuote(pattern))
	}
	parts = append(parts, "--count", "5000")

	line := strings.Join(parts, " ")
	if conn.UseSSH && conn.SSHConfig != nil {
		return sshComment(conn) + "\n" + line
	}
	return line
}

// connectionFlags returns the host/port/db/auth/tls/cluster flags for conn,
// shared between GET-style and SCAN-style cli builders.
func connectionFlags(conn types.Connection) []string {
	var parts []string
	parts = append(parts, "-h", shellQuote(conn.Host))
	if conn.Port != 0 && conn.Port != 6379 {
		parts = append(parts, "-p", strconv.Itoa(conn.Port))
	}
	if conn.DB != 0 {
		parts = append(parts, "-n", strconv.Itoa(conn.DB))
	}
	if conn.Username != "" {
		parts = append(parts, "--user", shellQuote(conn.Username))
	}
	if conn.Password != "" {
		parts = append(parts, "-a", shellQuote(conn.Password))
	}
	if conn.UseCluster {
		parts = append(parts, "-c")
	}
	if conn.UseTLS {
		parts = append(parts, "--tls")
		if conn.TLSConfig != nil {
			if conn.TLSConfig.CertFile != "" {
				parts = append(parts, "--cert", shellQuote(conn.TLSConfig.CertFile))
			}
			if conn.TLSConfig.KeyFile != "" {
				parts = append(parts, "--key", shellQuote(conn.TLSConfig.KeyFile))
			}
			if conn.TLSConfig.CAFile != "" {
				parts = append(parts, "--cacert", shellQuote(conn.TLSConfig.CAFile))
			}
			if conn.TLSConfig.InsecureSkipVerify {
				parts = append(parts, "--insecure")
			}
			if conn.TLSConfig.ServerName != "" {
				parts = append(parts, "--sni", shellQuote(conn.TLSConfig.ServerName))
			}
		}
	}
	return parts
}

func cliOpFor(t types.KeyType, key string) []string {
	q := shellQuote(key)
	switch t {
	case types.KeyTypeString:
		return []string{"GET", q}
	case types.KeyTypeList:
		return []string{"LRANGE", q, "0", "-1"}
	case types.KeyTypeSet:
		return []string{"SMEMBERS", q}
	case types.KeyTypeZSet:
		return []string{"ZRANGE", q, "0", "-1", "WITHSCORES"}
	case types.KeyTypeHash:
		return []string{"HGETALL", q}
	case types.KeyTypeStream:
		return []string{"XRANGE", q, "-", "+"}
	case types.KeyTypeJSON:
		return []string{"JSON.GET", q}
	case types.KeyTypeHyperLogLog:
		return []string{"PFCOUNT", q}
	case types.KeyTypeBitmap:
		return []string{"BITCOUNT", q}
	case types.KeyTypeGeo:
		return []string{"GEOSEARCH", q, "FROMLONLAT", "0", "0", "BYRADIUS", "20037509", "m", "ASC", "WITHCOORD"}
	default:
		return []string{"TYPE", q}
	}
}

func sshComment(conn types.Connection) string {
	cfg := conn.SSHConfig
	user := cfg.User
	if user == "" {
		user = "<user>"
	}
	port := cfg.Port
	if port == 0 {
		port = 22
	}
	return fmt.Sprintf("# requires SSH tunnel: ssh -L <local>:%s:%d %s@%s -p %d",
		conn.Host, conn.Port, user, cfg.Host, port)
}

// shellQuote wraps s in single quotes for POSIX shells, escaping any embedded
// single quotes. Always quotes, so output is unambiguous even for benign input.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
