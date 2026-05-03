// Package decoder turns Redis values that are not plain text into something
// human-readable. It auto-detects common envelope formats (base64, JSON,
// LangGraph JsonPlus, raw msgpack) and renders them as pretty-printed JSON.
package decoder

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Format identifies how a value was decoded.
type Format string

const (
	FormatRaw      Format = "raw"
	FormatBase64   Format = "base64"
	FormatJSON     Format = "json"
	FormatJsonPlus Format = "jsonplus"
	FormatMsgpack  Format = "msgpack"
)

// Decoded is a successfully-rendered value.
type Decoded struct {
	Format Format
	Pretty string // human-readable rendering (typically pretty JSON)
	Note   string // optional caveat ("contains pickle blob", etc.)
}

// Detect picks the best format for the input. It does not decode.
// Order: JsonPlus envelope (JSON containing {type, data} fields) → JSON →
// msgpack (binary) → base64 → raw.
func Detect(data []byte) Format {
	if len(data) == 0 {
		return FormatRaw
	}

	if looksLikeJSON(data) {
		if isJsonPlusEnvelope(data) {
			return FormatJsonPlus
		}
		return FormatJSON
	}

	if looksLikeMsgpack(data) {
		return FormatMsgpack
	}

	if looksLikeBase64(data) {
		return FormatBase64
	}

	return FormatRaw
}

// Decode renders data according to format. If format is FormatRaw, data is
// returned as a UTF-8 string (or hex dump if non-printable).
func Decode(data []byte, format Format) (Decoded, error) {
	switch format {
	case FormatRaw:
		return decodeRaw(data), nil
	case FormatBase64:
		return decodeBase64(data)
	case FormatJSON:
		return decodeJSON(data)
	case FormatJsonPlus:
		return decodeJsonPlus(data)
	case FormatMsgpack:
		return decodeMsgpack(data)
	}
	return Decoded{}, fmt.Errorf("unknown format: %s", format)
}

// CycleFormat returns the next format when the user presses 'd' to override.
// Order: raw → base64 → json → jsonplus → msgpack → raw.
func CycleFormat(current Format) Format {
	switch current {
	case FormatRaw:
		return FormatBase64
	case FormatBase64:
		return FormatJSON
	case FormatJSON:
		return FormatJsonPlus
	case FormatJsonPlus:
		return FormatMsgpack
	case FormatMsgpack:
		return FormatRaw
	}
	return FormatRaw
}

func decodeRaw(data []byte) Decoded {
	if utf8.Valid(data) && isMostlyPrintable(data) {
		return Decoded{Format: FormatRaw, Pretty: string(data)}
	}
	return Decoded{Format: FormatRaw, Pretty: hexDump(data, 1024)}
}

func decodeBase64(data []byte) (Decoded, error) {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		// Try URL-safe variant.
		decoded, err = base64.URLEncoding.DecodeString(strings.TrimSpace(string(data)))
		if err != nil {
			return Decoded{}, fmt.Errorf("base64 decode: %w", err)
		}
	}
	// If decoded looks printable, surface as text. Otherwise hex dump.
	if utf8.Valid(decoded) && isMostlyPrintable(decoded) {
		return Decoded{Format: FormatBase64, Pretty: string(decoded)}, nil
	}
	return Decoded{Format: FormatBase64, Pretty: hexDump(decoded, 1024)}, nil
}

func decodeJSON(data []byte) (Decoded, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return Decoded{}, fmt.Errorf("json decode: %w", err)
	}
	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return Decoded{}, fmt.Errorf("json marshal: %w", err)
	}
	return Decoded{Format: FormatJSON, Pretty: string(pretty)}, nil
}

// looksLikeJSON returns true if data appears to be a JSON object or array.
// Conservative — only checks first/last non-whitespace chars and validates
// with a stdlib decode.
func looksLikeJSON(data []byte) bool {
	trimmed := bytes_trim_space(data)
	if len(trimmed) < 2 {
		return false
	}
	first := trimmed[0]
	last := trimmed[len(trimmed)-1]
	if !((first == '{' && last == '}') || (first == '[' && last == ']')) {
		return false
	}
	var v any
	return json.Unmarshal(trimmed, &v) == nil
}

// looksLikeBase64 returns true if data is plausibly base64.
// Heuristic: alnum/+/=/- and padding-aware length, plus a successful decode.
func looksLikeBase64(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	trimmed := bytes_trim_space(data)
	for _, c := range trimmed {
		if !isBase64Byte(c) {
			return false
		}
	}
	// Valid base64 length is multiple of 4 (with padding).
	if len(trimmed)%4 != 0 {
		// URL-safe and unpadded variants are common; accept them but require decode.
	}
	if _, err := base64.StdEncoding.DecodeString(string(trimmed)); err == nil {
		return true
	}
	if _, err := base64.URLEncoding.DecodeString(string(trimmed)); err == nil {
		return true
	}
	return false
}

func isBase64Byte(b byte) bool {
	return (b >= 'A' && b <= 'Z') ||
		(b >= 'a' && b <= 'z') ||
		(b >= '0' && b <= '9') ||
		b == '+' || b == '/' || b == '=' || b == '-' || b == '_'
}

// looksLikeMsgpack does a quick first-byte check. msgpack starts with type
// indicators in known ranges. False positives are possible — Decode will
// confirm by attempting unmarshal.
func looksLikeMsgpack(data []byte) bool {
	if len(data) < 1 {
		return false
	}
	b := data[0]
	// fixmap, fixarray, fixstr, nil, false, true, ints, floats, etc.
	// Rough check: msgpack uses 0x80-0xff and some specific low bytes.
	// Reject obvious text starts.
	if b == '{' || b == '[' || b == '"' || b == ' ' || b == '\n' || b == '\t' {
		return false
	}
	// printable ASCII other than known msgpack low-byte type indicators?
	// Be conservative: only accept the typical msgpack ranges.
	switch {
	case b == 0xc0, b == 0xc2, b == 0xc3:
		return true
	case b >= 0x80 && b <= 0x9f: // fixmap, fixarray, fixstr
		return true
	case b >= 0xa0 && b <= 0xbf: // fixstr
		return true
	case b >= 0xc4 && b <= 0xd9: // bin, ext, float, int, str variants
		return true
	case b >= 0xda && b <= 0xdf: // str/array/map 16/32
		return true
	case b >= 0xe0: // negative fixint
		return true
	}
	return false
}

func isMostlyPrintable(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	printable := 0
	for _, b := range data {
		if b == '\n' || b == '\t' || b == '\r' || (b >= 0x20 && b < 0x7f) {
			printable++
		}
	}
	return printable*10 > len(data)*9 // ≥ 90%
}

func hexDump(data []byte, max int) string {
	if len(data) > max {
		data = data[:max]
	}
	var b strings.Builder
	enc := hex.EncodeToString(data)
	for i := 0; i < len(enc); i += 32 {
		end := i + 32
		if end > len(enc) {
			end = len(enc)
		}
		b.WriteString(enc[i:end])
		b.WriteByte('\n')
	}
	if len(data) == max {
		b.WriteString(fmt.Sprintf("... (truncated at %d bytes)\n", max))
	}
	return b.String()
}

// bytes_trim_space is a small helper to avoid importing "bytes" twice.
func bytes_trim_space(b []byte) []byte {
	start, end := 0, len(b)
	for start < end && isSpace(b[start]) {
		start++
	}
	for end > start && isSpace(b[end-1]) {
		end--
	}
	return b[start:end]
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// errInvalidEnvelope is returned when JsonPlus detection passes but decode
// fails (mismatched structure).
var errInvalidEnvelope = errors.New("invalid jsonplus envelope")
