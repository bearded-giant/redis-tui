package decoder

import (
	"bytes"
	"fmt"
	"io"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

// walkMsgpack decodes a msgpack stream into a generic Go value, handling
// custom ext types as opaque structures rather than failing the whole decode.
//
// LangGraph (and any library that uses msgpack ext types) embeds python-typed
// values as ext (id 0..15). The standard msgpack.Unmarshal aborts with
// "unknown ext id=N" when it hits one. This walker keeps going.
//
// Ext values are returned as:
//
//	map[string]any{
//	    "_ext_id":  N,
//	    "_payload": <recursive decode of the ext payload>,
//	}
//
// If the ext payload is itself msgpack (typical for LangGraph),
// the inner structure renders inline. If it isn't decodable as msgpack,
// the payload is returned as a hex dump.
func walkMsgpack(data []byte) (any, error) {
	d := msgpack.NewDecoder(bytes.NewReader(data))
	return walkValue(d)
}

func walkValue(d *msgpack.Decoder) (any, error) {
	code, err := d.PeekCode()
	if err != nil {
		return nil, err
	}

	switch {
	case code == msgpcode.Nil:
		return nil, d.DecodeNil()
	case code == msgpcode.False, code == msgpcode.True:
		return d.DecodeBool()
	case msgpcode.IsString(code):
		return d.DecodeString()
	case msgpcode.IsBin(code):
		b, err := d.DecodeBytes()
		if err != nil {
			return nil, err
		}
		return renderBytes(b), nil
	case msgpcode.IsFixedMap(code), code == msgpcode.Map16, code == msgpcode.Map32:
		n, err := d.DecodeMapLen()
		if err != nil {
			return nil, err
		}
		out := make(map[string]any, n)
		for i := 0; i < n; i++ {
			key, err := walkValue(d)
			if err != nil {
				return nil, fmt.Errorf("map key #%d: %w", i, err)
			}
			val, err := walkValue(d)
			if err != nil {
				return nil, fmt.Errorf("map value for key %v: %w", key, err)
			}
			out[fmt.Sprint(key)] = val
		}
		return out, nil
	case msgpcode.IsFixedArray(code), code == msgpcode.Array16, code == msgpcode.Array32:
		n, err := d.DecodeArrayLen()
		if err != nil {
			return nil, err
		}
		out := make([]any, n)
		for i := 0; i < n; i++ {
			v, err := walkValue(d)
			if err != nil {
				return nil, fmt.Errorf("array #%d: %w", i, err)
			}
			out[i] = v
		}
		return out, nil
	case msgpcode.IsExt(code):
		return walkExt(d)
	case msgpcode.IsFixedNum(code),
		code == msgpcode.Uint8, code == msgpcode.Uint16, code == msgpcode.Uint32, code == msgpcode.Uint64,
		code == msgpcode.Int8, code == msgpcode.Int16, code == msgpcode.Int32, code == msgpcode.Int64:
		return d.DecodeInt64()
	case code == msgpcode.Float:
		return d.DecodeFloat32()
	case code == msgpcode.Double:
		return d.DecodeFloat64()
	}

	// Fallback: let msgpack handle anything we missed.
	return d.DecodeInterface()
}

// walkExt handles a msgpack ext value. The vmihailenco decoder exposes
// DecodeExtHeader which returns id+length, leaving the payload bytes in the
// underlying buffer. We read those bytes via Buffered() and try to recursively
// decode them as msgpack (LangGraph stores typed-python args as msgpack).
func walkExt(d *msgpack.Decoder) (any, error) {
	id, length, err := d.DecodeExtHeader()
	if err != nil {
		return nil, err
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(d.Buffered(), payload); err != nil {
		return nil, fmt.Errorf("ext id=%d payload read: %w", id, err)
	}

	// Try recursive msgpack decode. If it works, return the structured form.
	if inner, err := walkMsgpack(payload); err == nil {
		return map[string]any{
			"_ext_id":  int(id),
			"_payload": inner,
		}, nil
	}

	// Otherwise: opaque hex dump.
	return map[string]any{
		"_ext_id":   int(id),
		"_size":     length,
		"_data_hex": hexDump(payload, 256),
	}, nil
}
