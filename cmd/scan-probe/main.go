// scan-probe runs ScanKeys via the same client redis-tui uses, against a
// tunnel that's already open at 127.0.0.1:<port>.
//
//	go run ./cmd/scan-probe <port> <pattern>
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/bearded-giant/redis-tui/internal/redis"
	"github.com/bearded-giant/redis-tui/internal/types"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: scan-probe <port> [pattern]")
		os.Exit(2)
	}
	port, _ := strconv.Atoi(os.Args[1])
	pattern := "*"
	if len(os.Args) >= 3 {
		pattern = os.Args[2]
	}

	c := redis.NewClient()
	if err := c.Connect(types.Connection{
		Name: "probe",
		Host: "127.0.0.1",
		Port: port,
		DB:   0,
	}); err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		os.Exit(1)
	}
	defer c.Disconnect()

	total := c.GetTotalKeys()
	fmt.Fprintln(os.Stderr, "DBSIZE:", total)

	keys, cursor, err := c.ScanKeys(pattern, 0, 100)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan err:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "returned: %d keys, next cursor: %d\n\n", len(keys), cursor)
	for _, k := range keys {
		fmt.Printf("%s  type=%s  ttl=%v\n", k.Key, k.Type, k.TTL)
	}
}
