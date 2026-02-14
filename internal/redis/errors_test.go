package redis

import (
	"errors"
	"testing"
)

func TestErrInvalidRegex(t *testing.T) {
	t.Run("wraps error", func(t *testing.T) {
		original := errors.New("bad pattern")
		wrapped := errInvalidRegex(original)
		if wrapped == nil {
			t.Fatal("expected non-nil error")
		}
		if wrapped.Error() != "invalid regex: bad pattern" {
			t.Errorf("Error() = %q, want %q", wrapped.Error(), "invalid regex: bad pattern")
		}
	})

	t.Run("unwrappable via errors.Is", func(t *testing.T) {
		original := errors.New("bad pattern")
		wrapped := errInvalidRegex(original)
		if !errors.Is(wrapped, original) {
			t.Error("expected errors.Is to return true for wrapped error")
		}
	})
}
