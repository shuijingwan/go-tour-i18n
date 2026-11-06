package sitecontent

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

type cachedPageReplacement struct {
	input  *i18n.Catalog
	result *i18n.TranslationUnitReplacementResolution
}

// AttachPageReplacementInvocation gives one freshly built command Catalog the
// existing exact-input projection reuse. It caches no QC/currentness decision;
// every resolver call still reads and verifies the frozen representation file.
// Callers that retain a Catalog across commands must start a fresh invocation.
func AttachPageReplacementInvocation(root string, c *i18n.Catalog) {
	newClosureValidation(root).attachReplacementReuse(c)
}

func (v *closureValidation) attachReplacementReuse(c *i18n.Catalog) {
	if !v.disabled {
		c.ReplacementResolver = v.resolvePageReplacement
		c.BridgeResolver = v.resolveBridgeReconstruction
		c.BridgeConstructionContext = withBridgeConstructionEnvironment
	}
}

// The ordinary Page resolver has exactly two inputs: the frozen reference and
// the supplied inventory. Only its pure result is reused, with exact typed
// equality (including nil/empty slices). Shell foundations additionally read
// adoption/parser authority and always retain their original current reader.
func (v *closureValidation) resolvePageReplacement(root string, c *i18n.Catalog, ref i18n.GlossaryArchiveReference) (*i18n.TranslationUnitReplacementResolution, error) {
	// Representation reconciliation consumes the original owned source
	// adapters. It must not serialize or clone a readonly bridge proof owner.
	c = originalBridgeCatalog(c)
	v.count("replacement-resolution-call:" + ref.Path)
	if root != v.root {
		return nil, fmt.Errorf("validation root changed")
	}
	b, err := v.readRegular(root, ref.Path)
	if err != nil {
		return nil, err
	}
	if !validSHA(ref.SHA256) || v.deps[ref.Path] != ref.SHA256 {
		return nil, fmt.Errorf("historical Page representation hash mismatch")
	}
	var header struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(b, &header); err != nil || header.Schema == ShellSuccessorSchema {
		return resolvePageReplacements(root, c, ref)
	}
	// Admit only the repository-owned immutable adapters. Future adapters may
	// contain runtime state; they keep the canonical uncached resolver.
	for _, a := range c.Adapted {
		switch a.Adapter.(type) {
		case PageAdapter, ShellPageAdapter, adoptedPageAdapter:
		default:
			return resolvePageReplacements(root, c, ref)
		}
	}
	input := *c
	input.BridgeResolver = nil
	input.BridgeConstructionContext = nil
	input.ReplacementResolver, input.SurfaceAuthorityReader, input.HistoricalContextReader, input.PreviewFailureReader = nil, nil, nil, nil
	key := Reference{Path: ref.Path, SHA256: ref.SHA256}
	for _, cached := range v.reconciliations[key] {
		if reflect.DeepEqual(&input, cached.input) {
			v.count("replacement-resolution-reuse:" + ref.Path)
			return restoreReplacementReaders(cloneReplacementValue(reflect.ValueOf(cached.result)).Interface().(*i18n.TranslationUnitReplacementResolution), c), nil
		}
	}
	v.count("replacement-resolution-build:" + ref.Path)
	r, err := resolvePageReplacements(root, c, ref)
	if err != nil {
		return nil, err
	}
	if r.EventCatalog != nil {
		// A canonical successor bridge observes disk authority. It is not a
		// pure transform of the two ordinary Page reconciliation inputs.
		return r, nil
	}
	// Keep a small invocation-local set; no eviction, persistence or global
	// cache. A miss beyond this bound simply repeats the canonical calculation.
	count := 0
	for _, entries := range v.reconciliations {
		count += len(entries)
	}
	if count < 8 {
		if v.reconciliations == nil {
			v.reconciliations = map[Reference][]cachedPageReplacement{}
		}
		private := cloneReplacementValue(reflect.ValueOf(r)).Interface().(*i18n.TranslationUnitReplacementResolution)
		h := private.HistoricalCatalog
		h.BridgeResolver = nil
		h.BridgeConstructionContext = nil
		h.ReplacementResolver, h.SurfaceAuthorityReader, h.HistoricalContextReader, h.PreviewFailureReader = nil, nil, nil, nil
		v.reconciliations[key] = append(v.reconciliations[key], cachedPageReplacement{cloneReplacementValue(reflect.ValueOf(&input)).Interface().(*i18n.Catalog), private})
	}
	return r, nil
}

func restoreReplacementReaders(r *i18n.TranslationUnitReplacementResolution, c *i18n.Catalog) *i18n.TranslationUnitReplacementResolution {
	h := r.HistoricalCatalog
	h.BridgeResolver = c.BridgeResolver
	h.BridgeConstructionContext = c.BridgeConstructionContext
	h.ReplacementResolver, h.SurfaceAuthorityReader, h.HistoricalContextReader, h.PreviewFailureReader = c.ReplacementResolver, c.SurfaceAuthorityReader, c.HistoricalContextReader, c.PreviewFailureReader
	return r
}

// Private copies prevent caller mutations from changing either a memo key or
// result. This is limited above to plain Catalog data and the two owned adapter
// types; runtime reader functions are copied/restored separately.
func cloneReplacementValue(v reflect.Value) reflect.Value {
	out := reflect.New(v.Type()).Elem()
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			out.Set(reflect.New(v.Type().Elem()))
			out.Elem().Set(cloneReplacementValue(v.Elem()))
		}
	case reflect.Interface:
		if !v.IsNil() {
			out.Set(cloneReplacementValue(v.Elem()))
		}
	case reflect.Slice:
		if !v.IsNil() {
			out.Set(reflect.MakeSlice(v.Type(), v.Len(), v.Len()))
			// Source/archive buffers contain no mutable references. Bulk-copy
			// their bytes into private storage instead of reflecting each byte;
			// named byte slices and nil/empty distinctions remain unchanged.
			if v.Type().Elem().Kind() == reflect.Uint8 {
				reflect.Copy(out, v)
				break
			}
			for i := 0; i < v.Len(); i++ {
				out.Index(i).Set(cloneReplacementValue(v.Index(i)))
			}
		}
	case reflect.Struct:
		// This owned wrapper adds only an immutable root string. Its current
		// corpus gate lives in ProtectedInput, never in pure reconciliation.
		// Copy the private field explicitly and retain it in the typed key.
		if v.Type() == reflect.TypeOf(adoptedPageAdapter{}) {
			a := v.Interface().(adoptedPageAdapter)
			a.PageAdapter = cloneReplacementValue(reflect.ValueOf(a.PageAdapter)).Interface().(PageAdapter)
			out.Set(reflect.ValueOf(a))
			break
		}
		for i := 0; i < v.NumField(); i++ {
			out.Field(i).Set(cloneReplacementValue(v.Field(i)))
		}
	case reflect.Map:
		if !v.IsNil() {
			out.Set(reflect.MakeMapWithSize(v.Type(), v.Len()))
			it := v.MapRange()
			for it.Next() {
				out.SetMapIndex(it.Key(), cloneReplacementValue(it.Value()))
			}
		}
	default:
		out.Set(v)
	}
	return out
}
