package decoder

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/vmihailenco/msgpack/v5"
)

func TestDetect_Empty(t *testing.T) {
	if got := Detect(nil); got != FormatRaw {
		t.Errorf("Detect(nil) = %v, want raw", got)
	}
}

func TestDetect_PlainText(t *testing.T) {
	if got := Detect([]byte("hello world")); got != FormatRaw {
		t.Errorf("Detect plain = %v, want raw", got)
	}
}

func TestDetect_JSON(t *testing.T) {
	if got := Detect([]byte(`{"k":"v"}`)); got != FormatJSON {
		t.Errorf("Detect json = %v, want json", got)
	}
	if got := Detect([]byte(`[1,2,3]`)); got != FormatJSON {
		t.Errorf("Detect array = %v, want json", got)
	}
}

func TestDetect_JsonPlus(t *testing.T) {
	envelope := `{"checkpoint":{"type":"msgpack","data":"gA=="},"v":1}`
	if got := Detect([]byte(envelope)); got != FormatJsonPlus {
		t.Errorf("Detect envelope = %v, want jsonplus", got)
	}
}

func TestDetect_Base64(t *testing.T) {
	// 24 chars of clean base64 with no special prefix.
	encoded := base64.StdEncoding.EncodeToString([]byte("hello world"))
	if got := Detect([]byte(encoded)); got != FormatBase64 {
		t.Errorf("Detect base64 = %v, want base64", got)
	}
}

func TestDetect_Msgpack(t *testing.T) {
	packed, err := msgpack.Marshal(map[string]any{"a": 1, "b": "two"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := Detect(packed); got != FormatMsgpack {
		t.Errorf("Detect msgpack = %v, want msgpack", got)
	}
}

func TestDecode_Raw_Printable(t *testing.T) {
	got, err := Decode([]byte("hello"), FormatRaw)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Pretty != "hello" {
		t.Errorf("Pretty = %q", got.Pretty)
	}
}

func TestDecode_Raw_Binary(t *testing.T) {
	got, err := Decode([]byte{0x00, 0x01, 0x02, 0xff}, FormatRaw)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !strings.Contains(got.Pretty, "000102ff") {
		t.Errorf("expected hex dump, got %q", got.Pretty)
	}
}

func TestDecode_Base64_Printable(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("hello world"))
	got, err := Decode([]byte(encoded), FormatBase64)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Pretty != "hello world" {
		t.Errorf("Pretty = %q", got.Pretty)
	}
}

func TestDecode_Base64_Binary(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte{0xde, 0xad, 0xbe, 0xef})
	got, err := Decode([]byte(encoded), FormatBase64)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !strings.Contains(got.Pretty, "deadbeef") {
		t.Errorf("expected hex, got %q", got.Pretty)
	}
}

func TestDecode_Base64_URLSafe(t *testing.T) {
	encoded := base64.URLEncoding.EncodeToString([]byte("ok"))
	got, err := Decode([]byte(encoded), FormatBase64)
	if err != nil {
		t.Fatalf("Decode urlsafe: %v", err)
	}
	if got.Pretty != "ok" {
		t.Errorf("Pretty = %q", got.Pretty)
	}
}

func TestDecode_Base64_Invalid(t *testing.T) {
	_, err := Decode([]byte("!!!not-base64!!!"), FormatBase64)
	if err == nil {
		t.Error("expected base64 decode error")
	}
}

func TestDecode_JSON(t *testing.T) {
	got, err := Decode([]byte(`{"a":1,"b":[2,3]}`), FormatJSON)
	if err != nil {
		t.Fatalf("Decode json: %v", err)
	}
	if !strings.Contains(got.Pretty, `"a": 1`) {
		t.Errorf("expected pretty json, got %q", got.Pretty)
	}
}

func TestDecode_JSON_Invalid(t *testing.T) {
	_, err := Decode([]byte(`{not json`), FormatJSON)
	if err == nil {
		t.Error("expected json decode error")
	}
}

func TestDecode_Msgpack(t *testing.T) {
	packed, err := msgpack.Marshal(map[string]any{"x": 42})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := Decode(packed, FormatMsgpack)
	if err != nil {
		t.Fatalf("Decode msgpack: %v", err)
	}
	if !strings.Contains(got.Pretty, `"x": 42`) {
		t.Errorf("expected x:42 in %q", got.Pretty)
	}
}

func TestDecode_Msgpack_Invalid(t *testing.T) {
	_, err := Decode([]byte{0xc1}, FormatMsgpack) // 0xc1 is reserved/invalid
	if err == nil {
		t.Error("expected msgpack decode error")
	}
}

func TestDecode_JsonPlus_Msgpack(t *testing.T) {
	inner, err := msgpack.Marshal(map[string]any{"hello": "world"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	envelope := map[string]any{
		"v":          1,
		"checkpoint": map[string]string{"type": "msgpack", "data": base64.StdEncoding.EncodeToString(inner)},
	}
	envBytes, _ := json.Marshal(envelope)

	got, err := Decode(envBytes, FormatJsonPlus)
	if err != nil {
		t.Fatalf("Decode jsonplus: %v", err)
	}
	if !strings.Contains(got.Pretty, `"hello": "world"`) {
		t.Errorf("expected decoded msgpack inside envelope, got %q", got.Pretty)
	}
	if !strings.Contains(got.Pretty, `"v": 1`) {
		t.Errorf("expected envelope passthrough, got %q", got.Pretty)
	}
}

func TestDecode_JsonPlus_JSONField(t *testing.T) {
	innerJSON, _ := json.Marshal(map[string]string{"foo": "bar"})
	envelope := map[string]any{
		"meta": map[string]string{"type": "json", "data": base64.StdEncoding.EncodeToString(innerJSON)},
	}
	envBytes, _ := json.Marshal(envelope)

	got, err := Decode(envBytes, FormatJsonPlus)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !strings.Contains(got.Pretty, `"foo": "bar"`) {
		t.Errorf("expected json field decoded, got %q", got.Pretty)
	}
}

func TestDecode_JsonPlus_BytesField(t *testing.T) {
	envelope := map[string]any{
		"raw": map[string]string{"type": "bytes", "data": base64.StdEncoding.EncodeToString([]byte("plain text"))},
	}
	envBytes, _ := json.Marshal(envelope)
	got, err := Decode(envBytes, FormatJsonPlus)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !strings.Contains(got.Pretty, "plain text") {
		t.Errorf("expected bytes field rendered as text, got %q", got.Pretty)
	}
}

func TestDecode_JsonPlus_Pickle(t *testing.T) {
	envelope := map[string]any{
		"weights": map[string]string{"type": "pickle.numpy.ndarray", "data": base64.StdEncoding.EncodeToString([]byte{0x80, 0x04, 0x95})},
	}
	envBytes, _ := json.Marshal(envelope)
	got, err := Decode(envBytes, FormatJsonPlus)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !strings.Contains(got.Pretty, "[unsupported: pickle.numpy.ndarray blob") {
		t.Errorf("expected pickle placeholder, got %q", got.Pretty)
	}
	if !strings.Contains(got.Note, "pickle") {
		t.Errorf("expected pickle note, got %q", got.Note)
	}
}

func TestDecode_JsonPlus_PassthroughField(t *testing.T) {
	// Non-envelope field passes through verbatim.
	envelope := `{"v":1,"thread":"abc","ck":{"type":"json","data":"e30="}}`
	got, err := Decode([]byte(envelope), FormatJsonPlus)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !strings.Contains(got.Pretty, `"v": 1`) {
		t.Errorf("v not passed through")
	}
	if !strings.Contains(got.Pretty, `"thread": "abc"`) {
		t.Errorf("thread not passed through")
	}
}

func TestDecode_JsonPlus_BadOuter(t *testing.T) {
	_, err := Decode([]byte("{bad"), FormatJsonPlus)
	if err == nil {
		t.Error("expected outer parse error")
	}
}

func TestDecode_JsonPlus_BadBase64(t *testing.T) {
	envelope := `{"x":{"type":"msgpack","data":"!!!!"}}`
	got, err := Decode([]byte(envelope), FormatJsonPlus)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !strings.Contains(got.Pretty, "decode error") {
		t.Errorf("expected decode error placeholder, got %q", got.Pretty)
	}
}

func TestDecode_JsonPlus_DefaultType(t *testing.T) {
	// "default" type → treat as msgpack
	inner, _ := msgpack.Marshal([]any{"a", 1})
	envelope := map[string]any{
		"x": map[string]string{"type": "default", "data": base64.StdEncoding.EncodeToString(inner)},
	}
	envBytes, _ := json.Marshal(envelope)
	got, err := Decode(envBytes, FormatJsonPlus)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !strings.Contains(got.Pretty, `"a"`) {
		t.Errorf("default type should decode as msgpack, got %q", got.Pretty)
	}
}

func TestDecode_JsonPlus_UnknownTypeBytesFallback(t *testing.T) {
	// Unknown type with non-msgpack data → bytes rendering.
	envelope := map[string]any{
		"x": map[string]string{"type": "weird", "data": base64.StdEncoding.EncodeToString([]byte("plaintext"))},
	}
	envBytes, _ := json.Marshal(envelope)
	got, err := Decode(envBytes, FormatJsonPlus)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !strings.Contains(got.Pretty, "plaintext") {
		t.Errorf("expected bytes fallback, got %q", got.Pretty)
	}
}

func TestDecode_UnknownFormat(t *testing.T) {
	_, err := Decode([]byte("x"), Format("nope"))
	if err == nil {
		t.Error("expected unknown format error")
	}
}

func TestCycleFormat(t *testing.T) {
	tests := []struct{ in, want Format }{
		{FormatRaw, FormatBase64},
		{FormatBase64, FormatJSON},
		{FormatJSON, FormatJsonPlus},
		{FormatJsonPlus, FormatMsgpack},
		{FormatMsgpack, FormatRaw},
		{Format("garbage"), FormatRaw},
	}
	for _, tt := range tests {
		if got := CycleFormat(tt.in); got != tt.want {
			t.Errorf("CycleFormat(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeMsgpackValue_AnyMap(t *testing.T) {
	// Encode a map with non-string keys — msgpack decodes it as map[any]any.
	type tup struct{ A, B int }
	packed, err := msgpack.Marshal(tup{A: 1, B: 2})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	v, err := unmarshalMsgpack(packed)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Should be JSON-marshalable.
	if _, err := json.Marshal(v); err != nil {
		t.Errorf("normalized value should marshal as JSON: %v", err)
	}
}

func TestNormalizeMsgpackValue_BytesField(t *testing.T) {
	// Pack a struct with a binary field. msgpack will emit it as []byte.
	packed, err := msgpack.Marshal(map[string]any{"blob": []byte{0xde, 0xad}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	v, err := unmarshalMsgpack(packed)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, err := json.Marshal(v); err != nil {
		t.Errorf("expected json-marshalable, got %v", err)
	}
}

func TestWalkMsgpack_ExtType(t *testing.T) {
	// Build msgpack: fixmap{1} key="x" value=fixext1 with id=5 payload=0x42
	data := []byte{0x81, 0xa1, 'x', 0xd4, 0x05, 0x42}
	v, err := walkMsgpack(data)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", v)
	}
	ext, ok := m["x"].(map[string]any)
	if !ok {
		t.Fatalf("expected ext map at x, got %T", m["x"])
	}
	if ext["_ext_id"] != 5 {
		t.Errorf("_ext_id = %v, want 5", ext["_ext_id"])
	}
}

func TestWalkMsgpack_NestedExt(t *testing.T) {
	// Inner ext payload IS msgpack: fixarray{2} ["a", 1]
	innerArr := []byte{0x92, 0xa1, 'a', 0x01}
	// Wrap in fixmap{1} k="m" v=ext8(id=3, payload=innerArr)
	data := []byte{0x81, 0xa1, 'm', 0xc7, byte(len(innerArr)), 0x03}
	data = append(data, innerArr...)

	v, err := walkMsgpack(data)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	m := v.(map[string]any)
	ext := m["m"].(map[string]any)
	if ext["_ext_id"] != 3 {
		t.Errorf("ext_id = %v, want 3", ext["_ext_id"])
	}
	payload, ok := ext["_payload"].([]any)
	if !ok {
		t.Fatalf("payload should decode as array, got %T", ext["_payload"])
	}
	if len(payload) != 2 || payload[0] != "a" {
		t.Errorf("payload = %v", payload)
	}
}

func TestWalkMsgpack_ExtUndecodablePayload(t *testing.T) {
	// Ext payload is reserved msgpack code 0xc1 — non-decodable; walker
	// falls back to hex dump.
	data := []byte{0xd4, 0x07, 0xc1}
	v, err := walkMsgpack(data)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	ext := v.(map[string]any)
	if ext["_ext_id"] != 7 {
		t.Errorf("ext_id = %v", ext["_ext_id"])
	}
	if _, ok := ext["_data_hex"]; !ok {
		t.Errorf("expected _data_hex fallback, got %v", ext)
	}
}

func TestNormalizeMsgpackValue_NestedArray(t *testing.T) {
	packed, err := msgpack.Marshal([]any{1, []any{2, 3}, map[string]any{"k": "v"}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	v, err := unmarshalMsgpack(packed)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, err := json.Marshal(v); err != nil {
		t.Errorf("expected json-marshalable, got %v", err)
	}
}

// regression: hex dump truncation marker fires
func TestHexDump_Truncation(t *testing.T) {
	big := make([]byte, 2048)
	for i := range big {
		big[i] = byte(i)
	}
	out := hexDump(big, 1024)
	if !strings.Contains(out, "truncated") {
		t.Error("expected truncation note")
	}
}

func TestIsPickleType(t *testing.T) {
	cases := map[string]bool{
		"pickle":               true,
		"pickle.numpy.ndarray": true,
		"foo.pickle.bar":       true,
		"msgpack":              false,
		"json":                 false,
	}
	for in, want := range cases {
		if got := isPickleType(in); got != want {
			t.Errorf("isPickleType(%q) = %v, want %v", in, got, want)
		}
	}
}
