// ssh-probe dials a bastion the exact same way redis-tui does and reports
// success/failure. Use to isolate redis-tui's SSH plumbing from form/UX bugs.
//
//	go run ./cmd/ssh-probe <host> <port> <user> <keypath>
package main

import (
	"fmt"
	"os"

	"github.com/bearded-giant/redis-tui/internal/redis"
	"github.com/bearded-giant/redis-tui/internal/types"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "usage: ssh-probe <host> <port> <user> <keypath>")
		os.Exit(2)
	}
	port := 0
	fmt.Sscanf(os.Args[2], "%d", &port)

	c := redis.NewClient()
	dur, err := c.TestSSHConnection(&types.SSHConfig{
		Host:           os.Args[1],
		Port:           port,
		User:           os.Args[3],
		PrivateKeyPath: os.Args[4],
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
	fmt.Println("OK in", dur)
}
