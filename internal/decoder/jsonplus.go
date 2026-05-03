package decoder

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// jsonPlusField is the LangGraph JsonPlus envelope shape:
//
//	{"type": "msgpack" | "json" | "bytes" | ..., "data": "<base64>"}
type jsonPlusField struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

// isJsonPlusEnvelope returns true if data is a JSON object whose top-level
// values include at least one {type, data} field shape.
func isJsonPlusEnvelope(data []byte) bool {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return false
	}
	for _, raw := range obj {
		var f jsonPlusField
		if err := json.Unmarshal(raw, &f); err != nil {
			continue
		}
		if f.Type != "" && f.Data != "" {
			return true
		}
	}
	return false
}

// decodeJsonPlus walks an envelope, decoding each {type, data} field and
// rebuilding the object as a regular JSON document.
func decodeJsonPlus(data []byte) (Decoded, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return Decoded{}, fmt.Errorf("jsonplus parse: %w", err)
	}

	var pickleNote string
	out := make(map[string]any, len(obj))
	for key, raw := range obj {
		var f jsonPlusField
		if err := json.Unmarshal(raw, &f); err != nil || f.Type == "" || f.Data == "" {
			// Not an envelope field — pass through verbatim as a JSON value.
			var v any
			if err := json.Unmarshal(raw, &v); err == nil {
				out[key] = v
			} else {
				out[key] = string(raw)
			}
			continue
		}

		decoded, note, err := decodeJsonPlusField(f)
		if err != nil {
			out[key] = fmt.Sprintf("[decode error: %v]", err)
			continue
		}
		if note != "" {
			pickleNote = note
		}
		out[key] = decoded
	}

	pretty, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return Decoded{}, fmt.Errorf("jsonplus marshal: %w", err)
	}
	return Decoded{
		Format: FormatJsonPlus,
		Pretty: string(pretty),
		Note:   pickleNote,
	}, nil
}

// decodeJsonPlusField decodes a single {type, data} field. Returns
// (value, note, err). value is what should be placed in the output JSON.
// note is set when an unsupported (e.g. pickle) blob was placeholdered.
func decodeJsonPlusField(f jsonPlusField) (any, string, error) {
	raw, err := base64.StdEncoding.DecodeString(f.Data)
	if err != nil {
		raw, err = base64.URLEncoding.DecodeString(f.Data)
		if err != nil {
			return nil, "", fmt.Errorf("base64: %w", err)
		}
	}

	t := strings.ToLower(f.Type)
	if isPickleType(t) {
		return fmt.Sprintf("[unsupported: %s blob, %d bytes]", t, len(raw)), "envelope contains pickle data — values rendered as placeholders", nil
	}

	switch {
	case t == "json":
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, "", fmt.Errorf("json: %w", err)
		}
		return v, "", nil
	case t == "bytes":
		return renderBytes(raw), "", nil
	case t == "msgpack" || t == "default" || t == "":
		v, err := unmarshalMsgpack(raw)
		if err != nil {
			// LangGraph and similar libs encode custom Python types as
			// msgpack ext (id 0..15). Surface a useful placeholder rather
			// than a hard error so the rest of the envelope still renders.
			if extID, ok := extractExtID(err); ok {
				return map[string]any{
					"_unsupported_msgpack_ext": map[string]any{
						"ext_id":     extID,
						"raw_size":   len(raw),
						"raw_prefix": hexDump(raw, 64),
					},
				}, "value contains custom msgpack ext types (LangGraph python objects)", nil
			}
			return nil, "", fmt.Errorf("msgpack: %w", err)
		}
		return v, "", nil
	default:
		// Unknown type — render as bytes. Don't guess msgpack: leading byte
		// 0x00-0x7f is positive fixint to msgpack but plain text to bytes.
		return renderBytes(raw), "", nil
	}
}

// extractExtID parses an error message like
//
//	"msgpack: unknown ext id=5"
//
// to surface the ext id without depending on internal msgpack types.
func extractExtID(err error) (int, bool) {
	if err == nil {
		return 0, false
	}
	msg := err.Error()
	const marker = "ext id="
	idx := strings.Index(msg, marker)
	if idx < 0 {
		return 0, false
	}
	rest := msg[idx+len(marker):]
	id := 0
	consumed := 0
	for _, r := range rest {
		if r < '0' || r > '9' {
			break
		}
		id = id*10 + int(r-'0')
		consumed++
	}
	if consumed == 0 {
		return 0, false
	}
	return id, true
}

func isPickleType(t string) bool {
	if t == "pickle" {
		return true
	}
	// LangGraph pickle types tend to be qualified Python class names like
	// "pickle.numpy.ndarray" or include "pickle" in the prefix.
	if strings.HasPrefix(t, "pickle.") || strings.Contains(t, ".pickle.") {
		return true
	}
	return false
}

// renderBytes returns a printable representation: utf-8 string when valid,
// hex dump otherwise.
func renderBytes(raw []byte) any {
	if isMostlyPrintable(raw) {
		return string(raw)
	}
	return map[string]any{
		"_bytes_hex": hexDump(raw, 256),
		"_size":      len(raw),
	}
}
