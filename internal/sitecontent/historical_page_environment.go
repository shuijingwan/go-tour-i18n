package sitecontent

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Only the representation-owned affected package data is persisted. Other
// packages, readers, roots, current authority and QC judgments come from the
// freshly supplied Catalog in resolvePageReplacements on every invocation.
type historicalPageProjection struct {
	Descriptor i18n.TranslationPackageDescriptor
	Units      map[string]i18n.TranslationUnitReplacement
}

type historicalPageBinding struct {
	Inventory i18n.AdaptedTranslationUnit
	Page      PageAdapter
}

type historicalPageEnvironmentInput struct {
	Frozen             HistoricalPageRepresentations
	Package, Authority string
	Current            []historicalPageBinding
}

type historicalPageEnvironmentData struct {
	Package, Authority string
	Bindings           []historicalPageBinding
	Units              map[string]i18n.TranslationUnitReplacement
}

func sharedHistoricalPageProjection(frozen *HistoricalPageRepresentations, descriptor *i18n.TranslationPackageDescriptor, next []PageAdapter) (*historicalPageProjection, error) {
	r, _, err := historicalPageProjectionWithCounter(frozen, descriptor, next)
	return r, err
}

// The counter reports actual complete projection calculations, not cache hits
// or acceptance decisions. It is returned to focused tests, never logged by CLI.
func historicalPageProjectionWithCounter(frozen *HistoricalPageRepresentations, descriptor *i18n.TranslationPackageDescriptor, next []PageAdapter) (*historicalPageProjection, bool, error) {
	canonical := func() (*historicalPageProjection, error) {
		return deriveHistoricalPageProjection(frozen, descriptor, next)
	}
	input := historicalPageEnvironmentInput{Frozen: *frozen, Package: descriptor.ID, Authority: descriptor.AuthoritySHA256}
	for i, u := range descriptor.Units {
		if u.Unit.Adapter != nil || i >= len(next) {
			r, e := canonical()
			return r, true, e
		}
		switch u.Adapter.(type) {
		case PageAdapter, adoptedPageAdapter:
		default:
			r, e := canonical()
			return r, true, e
		}
		u.Adapter = nil
		input.Current = append(input.Current, historicalPageBinding{Inventory: u, Page: next[i]})
	}
	key, err := json.Marshal(input)
	if err != nil {
		r, e := canonical()
		return r, true, e
	}
	var copied historicalPageEnvironmentInput
	if json.Unmarshal(key, &copied) != nil || !reflect.DeepEqual(input, copied) {
		r, e := canonical()
		return r, true, e
	}
	built := false
	compute := func() ([]byte, error) {
		built = true
		r, err := canonical()
		if err != nil {
			return nil, err
		}
		wire := historicalPageEnvironmentData{Package: r.Descriptor.ID, Authority: r.Descriptor.AuthoritySHA256, Units: r.Units}
		for _, u := range r.Descriptor.Units {
			page, ok := u.Adapter.(PageAdapter)
			if !ok {
				return nil, fmt.Errorf("unpersistable historical adapter")
			}
			u.Adapter = nil
			wire.Bindings = append(wire.Bindings, historicalPageBinding{Inventory: u, Page: page})
		}
		encoded, err := json.Marshal(wire)
		if err != nil {
			return nil, err
		}
		var copy historicalPageEnvironmentData
		if json.Unmarshal(encoded, &copy) != nil || !reflect.DeepEqual(wire, copy) {
			return nil, fmt.Errorf("historical data codec is not lossless")
		}
		return encoded, nil
	}
	b, err := contentidentity.Transform("sitecontent/historical-page-environment/v1", key, compute)
	if err != nil {
		r, e := canonical()
		return r, true, e
	}
	var wire historicalPageEnvironmentData
	if json.Unmarshal(b, &wire) != nil || wire.Package != descriptor.ID || wire.Authority != descriptor.AuthoritySHA256 || len(wire.Bindings) != len(frozen.Documents) || len(wire.Units) != len(frozen.Documents) {
		r, e := canonical()
		return r, true, e
	}
	out := &historicalPageProjection{Descriptor: i18n.TranslationPackageDescriptor{ID: wire.Package, AuthoritySHA256: wire.Authority}, Units: wire.Units}
	for _, binding := range wire.Bindings {
		u := binding.Inventory
		u.Adapter = binding.Page
		out.Descriptor.Units = append(out.Descriptor.Units, u)
	}
	return out, built, nil
}
