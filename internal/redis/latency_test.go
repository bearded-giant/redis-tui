package redis

import (
	"testing"
)

func TestAsString(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{"plain", "plain"},
		{[]byte("bytes"), "bytes"},
		{42, "42"},
		{nil, "<nil>"},
	}
	for _, c := range cases {
		got := asString(c.in)
		if got != c.want {
			t.Errorf("asString(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestAsInt64(t *testing.T) {
	cases := []struct {
		in   any
		want int64
	}{
		{int64(42), 42},
		{int(7), 7},
		{"100", 100},
		{[]byte("250"), 250},
		{"not a number", 0},
		{nil, 0},
	}
	for _, c := range cases {
		got := asInt64(c.in)
		if got != c.want {
			t.Errorf("asInt64(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}
