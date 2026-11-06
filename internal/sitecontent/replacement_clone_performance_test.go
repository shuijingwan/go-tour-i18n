package sitecontent

import (
	"bytes"
	"reflect"
	"testing"
)

// Byte buffers are private observations, not cached validation decisions.
func TestReplacementByteCloneIsolation(t *testing.T) {
	type octet uint8
	type octets []octet
	type node struct {
		Source []byte
		Named  octets
		Nil    []byte
		Empty  []byte
		Nested map[string][]byte
	}
	input := node{Source: []byte("source"), Named: octets{109, 97, 112, 112, 105, 110, 103}, Empty: []byte{}, Nested: map[string][]byte{"raw": []byte("raw")}}
	got := cloneReplacementValue(reflect.ValueOf(input)).Interface().(node)
	if !reflect.DeepEqual(input, got) {
		t.Fatal("clone changed typed value or nil/empty distinction")
	}
	got.Source[0] = 'X'
	got.Named[0] = 'X'
	got.Nested["raw"][0] = 'X'
	if !bytes.Equal(input.Source, []byte("source")) || !reflect.DeepEqual(input.Named, octets{109, 97, 112, 112, 105, 110, 103}) || string(input.Nested["raw"]) != "raw" {
		t.Fatal("private clone aliases input")
	}
	input.Source[1] = 'Y'
	if got.Source[1] == 'Y' {
		t.Fatal("input mutation escaped into clone")
	}
}

func BenchmarkReplacementSourceByteClone(b *testing.B) {
	source := bytes.Repeat([]byte("source-owned immutable observation\n"), 16384)
	value := reflect.ValueOf(source)
	b.ReportAllocs()
	b.SetBytes(int64(len(source)))
	for i := 0; i < b.N; i++ {
		copy := cloneReplacementValue(value).Interface().([]byte)
		if len(copy) != len(source) {
			b.Fatal("clone length")
		}
	}
}
