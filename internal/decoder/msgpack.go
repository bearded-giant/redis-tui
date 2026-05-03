package decoder

import (
	"encoding/json"
	"fmt"
)

// unmarshalMsgpack decodes data into a generic Go value. Uses the custom
// walker so msgpack ext types (LangGraph python objects, etc.) render as
// opaque tagged structures instead of erroring out the whole decode.
func unmarshalMsgpack(data []byte) (any, error) {
	v, err := walkMsgpack(data)
	if err != nil {
		return nil, err
	}
	return normalizeMsgpackValue(v), nil
}

// normalizeMsgpackValue walks a decoded msgpack value and replaces types
// that don't survive json.Marshal (e.g. map[any]any, []byte that isn't UTF-8).
func normalizeMsgpackValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			out[k] = normalizeMsgpackValue(vv)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			out[fmt.Sprint(k)] = normalizeMsgpackValue(vv)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, vv := range val {
			out[i] = normalizeMsgpackValue(vv)
		}
		return out
	case []byte:
		return renderBytes(val)
	}
	return v
}

func decodeMsgpack(data []byte) (Decoded, error) {
	v, err := unmarshalMsgpack(data)
	if err != nil {
		return Decoded{}, fmt.Errorf("msgpack: %w", err)
	}
	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return Decoded{}, fmt.Errorf("msgpack marshal: %w", err)
	}
	return Decoded{Format: FormatMsgpack, Pretty: string(pretty)}, nil
}
