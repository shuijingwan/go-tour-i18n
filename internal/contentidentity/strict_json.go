package contentidentity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"
)

func StrictJSON(b []byte, v any) error {
	if !utf8.Valid(b) {
		return fmt.Errorf("invalid JSON UTF-8")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	var walk func(reflect.Type) error
	walk = func(typ reflect.Type) error {
		for typ != nil && typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		if typ == reflect.TypeOf(json.RawMessage{}) {
			typ = nil
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		if t == json.Delim('{') {
			seen := map[string]bool{}
			fields := map[string]reflect.Type{}
			if typ != nil && typ.Kind() == reflect.Struct {
				for i := 0; i < typ.NumField(); i++ {
					f := typ.Field(i)
					if f.PkgPath != "" {
						continue
					}
					name := strings.Split(f.Tag.Get("json"), ",")[0]
					if name == "-" {
						continue
					}
					if name == "" {
						name = f.Name
					}
					fields[name] = f.Type
				}
			}
			for d.More() {
				k, err := d.Token()
				if err != nil {
					return err
				}
				s, ok := k.(string)
				if !ok || seen[s] {
					return fmt.Errorf("duplicate/invalid JSON member %v", k)
				}
				seen[s] = true
				var child reflect.Type
				if typ != nil && typ.Kind() == reflect.Struct {
					var exists bool
					child, exists = fields[s]
					if !exists {
						return fmt.Errorf("unknown JSON member %s", s)
					}
				} else if typ != nil && typ.Kind() == reflect.Map {
					child = typ.Elem()
				}
				if err := walk(child); err != nil {
					return err
				}
			}
			_, err = d.Token()
			return err
		}
		if t == json.Delim('[') {
			var child reflect.Type
			if typ != nil && (typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array) {
				child = typ.Elem()
			}
			for d.More() {
				if err := walk(child); err != nil {
					return err
				}
			}
			_, err = d.Token()
			return err
		}
		if t == nil && typ != nil && (typ.Kind() == reflect.String || typ.Kind() == reflect.Bool || typ.Kind() == reflect.Int || typ.Kind() == reflect.Int64) {
			return fmt.Errorf("null scalar is not allowed")
		}
		return nil
	}
	if err := walk(reflect.TypeOf(v)); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON")
	}
	d = json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	return d.Decode(v)
}
