package decoder

import (
	"strings"
	"testing"
)

func TestApplyJqPath(t *testing.T) {
	dec := Decoded{
		Format: FormatJSON,
		Pretty: `{
  "job_id": "abc",
  "metadata": {
    "thread_id": "tid-9",
    "ts": 12345
  },
  "items": [1, 2, 3]
}`,
	}

	t.Run("empty expr returns Pretty unchanged", func(t *testing.T) {
		out, err := ApplyJqPath(dec, "")
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if out != dec.Pretty {
			t.Errorf("out = %q, want unchanged", out)
		}
	})

	t.Run("scalar path", func(t *testing.T) {
		out, err := ApplyJqPath(dec, ".job_id")
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if !strings.Contains(out, `"abc"`) {
			t.Errorf("out = %q, want abc", out)
		}
	})

	t.Run("nested path", func(t *testing.T) {
		out, err := ApplyJqPath(dec, ".metadata.thread_id")
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if !strings.Contains(out, `"tid-9"`) {
			t.Errorf("out = %q, want tid-9", out)
		}
	})

	t.Run("multi-result returns array", func(t *testing.T) {
		out, err := ApplyJqPath(dec, ".items[]")
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if !strings.Contains(out, "[") || !strings.Contains(out, "1") || !strings.Contains(out, "3") {
			t.Errorf("out = %q, want array w/ 1,2,3", out)
		}
	})

	t.Run("missing path returns null", func(t *testing.T) {
		out, err := ApplyJqPath(dec, ".nope")
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if strings.TrimSpace(out) != "null" {
			t.Errorf("out = %q, want null", out)
		}
	})

	t.Run("invalid expr returns error", func(t *testing.T) {
		_, err := ApplyJqPath(dec, "...not valid")
		if err == nil {
			t.Errorf("err = nil, want parse error")
		}
	})

	t.Run("non-json Pretty returns error", func(t *testing.T) {
		raw := Decoded{Format: FormatRaw, Pretty: "hello world"}
		_, err := ApplyJqPath(raw, ".foo")
		if err == nil {
			t.Errorf("err = nil, want unmarshal error")
		}
	})

	t.Run("empty Pretty returns error", func(t *testing.T) {
		empty := Decoded{Format: FormatJSON, Pretty: ""}
		_, err := ApplyJqPath(empty, ".foo")
		if err == nil {
			t.Errorf("err = nil, want error")
		}
	})
}
