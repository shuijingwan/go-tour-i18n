package contentidentity

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

// Persist parsed data, never a validation/currentness verdict. Reference hashes,
// package scope, sources, Reviews, candidates and all other semantic comparisons
// remain in callers. Custom decoders, nonzero targets and lossy JSON round trips
// always use the original decoder. Large strict JSON walks are a measured shared
// cost; small documents retain the cheaper existing L1/eager path.
func persistentJSON(b []byte, target any) (bool, error) {
	m := activeMemo()
	if m == nil || m.store == nil || len(b) < 4096 || len(b) > persistentResultLimit {
		return false, nil
	}
	v := reflect.ValueOf(target).Elem()
	typ := v.Type()
	if !plainJSONType(typ, map[reflect.Type]bool{}) {
		return false, nil
	}
	key, _ := json.Marshal(struct {
		Type, SHA256 string
		Bytes        int
	}{jsonTypeIdentity(typ), exactSHA256(b), len(b)})
	computed := false
	lossy := errors.New("JSON value is not round-trip exact")
	encoded, err := Transform("strict-json/parsed-value/v1", key, func() ([]byte, error) {
		computed = true
		if err := strictJSONDecode(b, target); err != nil {
			return nil, err
		}
		out, err := json.Marshal(target)
		if err != nil {
			return nil, lossy
		}
		copy := reflect.New(typ)
		if json.Unmarshal(out, copy.Interface()) != nil || !reflect.DeepEqual(v.Interface(), copy.Elem().Interface()) {
			return nil, lossy
		}
		return out, nil
	})
	if computed {
		if err == lossy {
			return true, nil // original successful value; never admitted to either cache
		}
		return true, err
	}
	if err == nil {
		// Decode into a private zero target. A malformed derived result is a
		// cache miss, never an error propagated from the acceleration layer.
		copy := reflect.New(typ)
		if json.Unmarshal(encoded, copy.Interface()) == nil {
			v.Set(copy.Elem())
			return true, nil
		}
	}
	return false, nil
}

func jsonTypeIdentity(typ reflect.Type) string {
	var b strings.Builder
	seen := map[reflect.Type]bool{}
	var walk func(reflect.Type)
	walk = func(t reflect.Type) {
		b.WriteString(t.PkgPath() + ":" + t.String() + ";")
		if seen[t] {
			return
		}
		seen[t] = true
		switch t.Kind() {
		case reflect.Pointer, reflect.Array, reflect.Slice:
			walk(t.Elem())
		case reflect.Map:
			walk(t.Key())
			walk(t.Elem())
		case reflect.Struct:
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				b.WriteString(f.Name + ":" + string(f.Tag) + ";")
				walk(f.Type)
			}
		}
	}
	walk(typ)
	return b.String()
}
