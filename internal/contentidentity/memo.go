package contentidentity

import (
	"bytes"
	"crypto/sha256"
	"encoding"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sync"
)

// Only deterministic byte transformations are memoized, never filesystem
// observations, currentness, authorization or PASS decisions. Each lookup
// compares the complete freshly supplied bytes; signatures are only indexes.
// In-memory objects never outlive the outermost CLI invocation. Pure Transform
// byte results additionally have a disposable, implementation-bound L2 store.
var memoScope struct {
	sync.Mutex
	users int
	cache *byteMemo
}

type byteSignature struct {
	size                int
	first, middle, last [16]byte
}
type byteMemoEntry struct {
	raw        []byte
	sha        string
	json       map[reflect.Type]reflect.Value
	transforms map[string][]byte
}
type byteMemo struct {
	sync.Mutex
	store                            *deterministicStore
	entries                          map[byteSignature][]*byteMemoEntry
	bytes, jsonBytes, transformBytes int
}

// BeginMemoization enables bounded, exact-byte transformation reuse until the
// returned function closes this invocation's scope. It never caches I/O results.
func BeginMemoization() func() {
	memoScope.Lock()
	if memoScope.users == 0 {
		memoScope.cache = &byteMemo{entries: map[byteSignature][]*byteMemoEntry{}, store: invocationStore()}
	}
	memoScope.users++
	memoScope.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			memoScope.Lock()
			defer memoScope.Unlock()
			memoScope.users--
			if memoScope.users == 0 {
				memoScope.cache = nil
			}
		})
	}
}
func activeMemo() *byteMemo { memoScope.Lock(); defer memoScope.Unlock(); return memoScope.cache }

func memoByteSignature(b []byte) (byteSignature, bool) {
	if len(b) < 128 || len(b) > 32<<20 {
		return byteSignature{}, false
	}
	k := byteSignature{size: len(b)}
	copy(k.first[:], b[:16])
	copy(k.middle[:], b[len(b)/2:len(b)/2+16])
	copy(k.last[:], b[len(b)-16:])
	return k, true
}

// Caller holds m's lock. A bounded miss always falls back to the original work.
func (m *byteMemo) entry(b []byte, create bool) *byteMemoEntry {
	k, ok := memoByteSignature(b)
	if !ok {
		return nil
	}
	for _, e := range m.entries[k] {
		if bytes.Equal(e.raw, b) {
			return e
		}
	}
	if !create || m.bytes+len(b) > 64<<20 {
		return nil
	}
	e := &byteMemoEntry{raw: bytes.Clone(b)}
	m.entries[k] = append(m.entries[k], e)
	m.bytes += len(b)
	return e
}

// The existing pure-transform budget includes its input descriptors too. A
// full SHA/JSON raw pool must not starve small transform keys. This uses the
// same exact-byte index/lifetime and existing 16 MiB limit, not another cache.
// Caller holds m's lock and has already looked for an existing entry.
func (m *byteMemo) transformInput(b []byte, outputCost int) *byteMemoEntry {
	k, ok := memoByteSignature(b)
	inputCost := len(b) + 64
	if !ok || m.transformBytes+inputCost+outputCost > 16<<20 {
		return nil
	}
	e := &byteMemoEntry{raw: bytes.Clone(b)}
	m.entries[k] = append(m.entries[k], e)
	m.transformBytes += inputCost
	return e
}

// SHA256Hex verifies b against any private cached observation in full before
// reusing its SHA-256. Uncached content always takes the normal hashing path.
func SHA256Hex(b []byte) string {
	compute := func() string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
	m := activeMemo()
	if m == nil {
		return compute()
	}
	m.Lock()
	defer m.Unlock()
	e := m.entry(b, true)
	if e == nil {
		return compute()
	}
	if e.sha == "" {
		e.sha = compute()
	}
	return e.sha
}

var jsonUnmarshalerType = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
var textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

// Custom unmarshalling and nonzero destinations retain the original decoder
// path, including side effects and merge semantics. Memo values never alias a
// caller's mutable maps, slices, pointers or interface payloads.
func plainJSONType(t reflect.Type, seen map[reflect.Type]bool) bool {
	if seen[t] {
		return true
	}
	seen[t] = true
	if t.Implements(jsonUnmarshalerType) || t.Implements(textUnmarshalerType) || reflect.PointerTo(t).Implements(jsonUnmarshalerType) || reflect.PointerTo(t).Implements(textUnmarshalerType) {
		return false
	}
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return plainJSONType(t.Elem(), seen)
	case reflect.Map:
		return t.Key().Kind() == reflect.String && plainJSONType(t.Key(), seen) && plainJSONType(t.Elem(), seen)
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" || !plainJSONType(f.Type, seen) {
				return false
			}
		}
		return true
	case reflect.Interface:
		return t.NumMethod() == 0
	case reflect.String, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return true
	}
	return false
}
func cloneJSONValue(v reflect.Value) reflect.Value {
	out := reflect.New(v.Type()).Elem()
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			out.Set(reflect.New(v.Type().Elem()))
			out.Elem().Set(cloneJSONValue(v.Elem()))
		}
	case reflect.Interface:
		if !v.IsNil() {
			out.Set(cloneJSONValue(v.Elem()))
		}
	case reflect.Slice:
		if !v.IsNil() {
			out.Set(reflect.MakeSlice(v.Type(), v.Len(), v.Len()))
			for i := 0; i < v.Len(); i++ {
				out.Index(i).Set(cloneJSONValue(v.Index(i)))
			}
		}
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			out.Index(i).Set(cloneJSONValue(v.Index(i)))
		}
	case reflect.Map:
		if !v.IsNil() {
			out.Set(reflect.MakeMapWithSize(v.Type(), v.Len()))
			it := v.MapRange()
			for it.Next() {
				out.SetMapIndex(it.Key(), cloneJSONValue(it.Value()))
			}
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			out.Field(i).Set(cloneJSONValue(v.Field(i)))
		}
	default:
		out.Set(v)
	}
	return out
}
func memoJSON(b []byte, v any, store bool) bool {
	m := activeMemo()
	if m == nil {
		return false
	}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Pointer || rv.IsNil() {
		return false
	}
	dst := rv.Elem()
	typ := dst.Type()
	if (!store && !dst.IsZero()) || !plainJSONType(typ, map[reflect.Type]bool{}) {
		return false
	}
	m.Lock()
	defer m.Unlock()
	e := m.entry(b, store)
	if e == nil {
		return false
	}
	if !store {
		if previous, ok := e.json[typ]; ok {
			dst.Set(cloneJSONValue(previous))
			return true
		}
		return false
	}
	// Bound decoded-value admission by raw input size. This is a cache budget,
	// not an exact Go heap-size estimate; misses retain the original validation.
	if _, ok := e.json[typ]; !ok && m.jsonBytes+4*len(b) <= 64<<20 {
		if e.json == nil {
			e.json = map[reflect.Type]reflect.Value{}
		}
		e.json[typ] = cloneJSONValue(dst)
		m.jsonBytes += 4 * len(b)
	}
	return false
}
