package contentidentity

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestMemoSHA256ExactBytesAndLifetime(t *testing.T) {
	b := bytes.Repeat([]byte("evidence"), 1024)
	done := BeginMemoization()
	check := func() {
		h := sha256.Sum256(b)
		if got := SHA256Hex(b); got != hex.EncodeToString(h[:]) {
			t.Fatal("SHA changed", got)
		}
	}
	check()
	b[40] = 'X'
	check()
	b[40] = 'Y'
	check()
	m := activeMemo()
	if len(m.entries) != 1 {
		t.Fatal("test did not collide in signature index")
	}
	if len(m.entries[signatureForTest(b)]) != 3 {
		t.Fatal("signature was mistaken for content identity")
	}
	nested := BeginMemoization()
	done()
	if activeMemo() == nil {
		t.Fatal("overlapping invocation lost cache")
	}
	nested()
	if activeMemo() != nil {
		t.Fatal("cache survived invocation")
	}
	check()
}
func signatureForTest(b []byte) byteSignature {
	k := byteSignature{size: len(b)}
	copy(k.first[:], b[:16])
	copy(k.middle[:], b[len(b)/2:len(b)/2+16])
	copy(k.last[:], b[len(b)-16:])
	return k
}

type memoJSONFixture struct {
	Name   string           `json:"name"`
	Values []string         `json:"values"`
	Map    map[string][]int `json:"map"`
	Child  *memoJSONFixture `json:"child"`
	Any    any              `json:"any"`
}

func TestMemoStrictJSONDifferentialAndIsolation(t *testing.T) {
	pad := strings.Repeat("value", 40)
	valid := `{"name":"` + pad + `","values":["one"],"map":{"numbers":[1,2]},"child":{"name":"child"},"any":{"array":[{"value":"text"}]}}`
	bad := []string{
		strings.Replace(valid, `"name":`, `"extra":`, 1),
		strings.Replace(valid, `"values":["one"]`, `"values":["one"],"values":[]`, 1),
		strings.Replace(valid, `"numbers":[1,2]`, `"numbers":[1,2],"numbers":[]`, 1),
		strings.Replace(valid, `"value":"text"`, `"value":"text","value":"other"`, 1),
		strings.Replace(valid, `"name":"child"`, `"name":null`, 1),
		strings.Replace(valid, `"name":"child"`, `"Name":"child"`, 1),
		strings.Replace(valid, `"name":"child"`, `"name":"child","\u006eame":"other"`, 1),
		valid + ` {}`, valid[:len(valid)-1], strings.Replace(valid, "child", "\xff", 1),
	}
	for _, raw := range append([]string{valid}, bad...) {
		var original memoJSONFixture
		before := StrictJSON([]byte(raw), &original)
		done := BeginMemoization()
		for i := 0; i < 2; i++ {
			var got memoJSONFixture
			err := StrictJSON([]byte(raw), &got)
			if (err == nil) != (before == nil) || (err == nil && !reflect.DeepEqual(original, got)) {
				t.Fatalf("decoder semantics changed: before=%v after=%v", before, err)
			}
		}
		done()
	}
	done := BeginMemoization()
	defer done()
	var first memoJSONFixture
	if err := StrictJSON([]byte(valid), &first); err != nil {
		t.Fatal(err)
	}
	first.Values[0] = "mutated"
	first.Map["numbers"][0] = 99
	first.Child.Name = "mutated"
	first.Any.(map[string]any)["array"].([]any)[0].(map[string]any)["value"] = "mutated"
	var next memoJSONFixture
	if err := StrictJSON([]byte(valid), &next); err != nil {
		t.Fatal(err)
	}
	if next.Values[0] != "one" || next.Map["numbers"][0] != 1 || next.Child.Name != "child" || next.Any.(map[string]any)["array"].([]any)[0].(map[string]any)["value"] != "text" {
		t.Fatal("cached value aliases caller")
	}
	merge := memoJSONFixture{Name: "retained"}
	raw := `{"values":["` + pad + `"]}`
	if err := StrictJSON([]byte(raw), &merge); err != nil || merge.Name != "retained" {
		t.Fatal("nonzero merge changed", err)
	}
}

type customMemoJSON struct{ Calls int }

type customMemoKey string

var customMemoKeyCalls int

func (v *customMemoKey) UnmarshalText(b []byte) error {
	customMemoKeyCalls++
	*v = customMemoKey(b)
	return nil
}

func (v *customMemoJSON) UnmarshalJSON(b []byte) error {
	v.Calls++
	var ignored map[string]any
	return json.Unmarshal(b, &ignored)
}
func TestMemoCustomDecoderAndBoundedFallback(t *testing.T) {
	done := BeginMemoization()
	defer done()
	raw := []byte(`{"padding":"` + strings.Repeat("x", 256) + `"}`)
	var v customMemoJSON
	for i := 0; i < 2; i++ {
		if err := StrictJSON(raw, &v); err == nil {
			t.Fatal("unknown custom fields must retain original strict rejection")
		}
	}
	// A custom method on an allowed field must still run for every decode.
	var a, b struct {
		Custom  customMemoJSON `json:"custom"`
		Padding string         `json:"padding"`
	}
	raw = []byte(`{"custom":{"Calls":0},"padding":"` + strings.Repeat("x", 256) + `"}`)
	if err := StrictJSON(raw, &a); err != nil {
		t.Fatal(err)
	}
	if err := StrictJSON(raw, &b); err != nil {
		t.Fatal(err)
	}
	if a.Custom.Calls != 1 || b.Custom.Calls != 1 {
		t.Fatal("custom decoder skipped")
	}
	customMemoKeyCalls = 0
	for i := 0; i < 2; i++ {
		var keys map[customMemoKey]string
		if err := StrictJSON([]byte(`{"key":"`+strings.Repeat("x", 256)+`"}`), &keys); err != nil {
			t.Fatal(err)
		}
	}
	if customMemoKeyCalls != 2 {
		t.Fatal("custom map key decoder skipped")
	}
	m := activeMemo()
	m.bytes = 64 << 20
	input := bytes.Repeat([]byte("different"), 128)
	h := sha256.Sum256(input)
	if SHA256Hex(input) != hex.EncodeToString(h[:]) {
		t.Fatal("bounded cache miss changed SHA")
	}
}
