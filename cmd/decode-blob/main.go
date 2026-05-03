// decode-blob is a small CLI to exercise internal/decoder against a file.
//
//	go run ./cmd/decode-blob /path/to/value
package main

import (
	"fmt"
	"os"

	"github.com/bearded-giant/redis-tui/internal/decoder"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: decode-blob <file>")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	format := decoder.Detect(data)
	fmt.Fprintf(os.Stderr, "detected: %s\nbytes:    %d\n\n", format, len(data))

	out, err := decoder.Decode(data, format)
	if err != nil {
		fmt.Fprintln(os.Stderr, "decode error:", err)
		os.Exit(1)
	}
	if out.Note != "" {
		fmt.Fprintf(os.Stderr, "note: %s\n\n", out.Note)
	}
	fmt.Println(out.Pretty)
}
