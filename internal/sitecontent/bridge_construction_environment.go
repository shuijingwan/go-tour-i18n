package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// This private decorator preserves every original adapter gate/identity. Only
// the two filesystem-free source-owned representation factories are reused.
// It is never serialized as a runtime reader or authorization verdict.
type observedBridgePage struct {
	PageAdapter
	original i18n.TranslationUnitAdapter
	proof    *structuredSuccessorValidation
	root     string
}

func (a observedBridgePage) ReattestationAuthority(root string) (string, error) {
	return a.original.(i18n.TranslationReattestationAdapter).ReattestationAuthority(root)
}
func (a observedBridgePage) BridgeContextReferences(root string) ([]i18n.GlossaryArchiveReference, error) {
	return a.original.(i18n.SameUnitBridgeAdapter).BridgeContextReferences(root)
}
func (a observedBridgePage) ProtectedInput(g *i18n.Glossary) (string, []string, error) {
	if adopted, ok := a.original.(adoptedPageAdapter); ok && a.proof != nil && adopted.root == a.root {
		// Same canonical current-corpus comparison, with one explicit source
		// proof owner for this readonly construction. Each comparison remains
		// fresh and installedCorpusObservation closes its complete guard.
		if err := CheckUnifiedCorpus(adopted.root, a.proof); err != nil {
			return "", nil, err
		}
		return adopted.PageAdapter.ProtectedInput(g)
	}
	return a.original.ProtectedInput(g)
}

// BridgeRepresentationCatalog is a private data view, not a new eligibility
// scope. Record keeps its original two fresh builds/publication boundaries;
// this form has no deferred verification after a write.
func BridgeRepresentationCatalog(root string, c *i18n.Catalog) *i18n.Catalog {
	// Record/public construction keeps eager per-call observation boundaries.
	// Only deterministic historical/source proof nodes live across its two
	// fresh builds; no verification is deferred until after publication.
	v := successorValidation(root, nil)
	v.currentSourcePhase = true
	return bridgeRepresentationCatalog(c, v, root)
}
func bridgeRepresentationCatalog(c *i18n.Catalog, v *structuredSuccessorValidation, root string) *i18n.Catalog {
	return mapBridgeCatalog(c, func(a i18n.TranslationUnitAdapter) i18n.TranslationUnitAdapter {
		switch p := a.(type) {
		case observedBridgePage:
			p.proof, p.root = v, root
			return p
		case PageAdapter:
			return observedBridgePage{p, a, v, root}
		case adoptedPageAdapter:
			return observedBridgePage{p.PageAdapter, a, v, root}
		default:
			return a
		}
	})
}
func originalBridgeCatalog(c *i18n.Catalog) *i18n.Catalog {
	return mapBridgeCatalog(c, func(a i18n.TranslationUnitAdapter) i18n.TranslationUnitAdapter {
		if p, ok := a.(observedBridgePage); ok {
			return p.original
		}
		return a
	})
}
func mapBridgeCatalog(c *i18n.Catalog, wrap func(i18n.TranslationUnitAdapter) i18n.TranslationUnitAdapter) *i18n.Catalog {
	out := *c
	out.Adapted = append([]i18n.AdaptedTranslationUnit(nil), c.Adapted...)
	for i := range out.Adapted {
		out.Adapted[i].Adapter = wrap(out.Adapted[i].Adapter)
	}
	copyPackages := func(in []i18n.TranslationPackageDescriptor) []i18n.TranslationPackageDescriptor {
		if in == nil {
			return nil
		}
		ps := append([]i18n.TranslationPackageDescriptor{}, in...)
		for i := range ps {
			ps[i].Units = append([]i18n.AdaptedTranslationUnit(nil), ps[i].Units...)
			for j := range ps[i].Units {
				ps[i].Units[j].Adapter = wrap(ps[i].Units[j].Adapter)
			}
		}
		return ps
	}
	out.TranslationPackages = copyPackages(c.TranslationPackages)
	out.HistoricalTranslationPackages = copyPackages(c.HistoricalTranslationPackages)
	return &out
}

func withBridgeConstructionEnvironment(root string, c *i18n.Catalog, work func(*i18n.Catalog) (*i18n.SameUnitBridgeResolution, error)) (*i18n.SameUnitBridgeResolution, error) {
	var result *i18n.SameUnitBridgeResolution
	err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		v.sharedSourceObservation, v.currentSourcePhase = true, true
		defer func() { v.sharedSourceObservation, v.currentSourcePhase = false, false }()
		var err error
		result, err = work(bridgeRepresentationCatalog(c, v, root))
		return err
	})
	return result, err
}

type bridgeRepresentationData struct {
	Page               PageAdapter
	Template, Ancestor []byte
}

// Input and output codecs must be lossless. The original pure factory is the
// only computation; invalid/unavailable derived cache always recomputes it.
func deriveBridgeRepresentation(input any, compute func() (*bridgeRepresentationData, error)) (*bridgeRepresentationData, bool, error) {
	canonical := func() (*bridgeRepresentationData, bool, error) { r, e := compute(); return r, true, e }
	key, err := json.Marshal(input)
	if err != nil {
		return canonical()
	}
	typ := reflect.TypeOf(input)
	if typ == nil || typ.Kind() != reflect.Struct {
		return canonical()
	}
	copyInput := reflect.New(typ)
	if json.Unmarshal(key, copyInput.Interface()) != nil || !reflect.DeepEqual(input, copyInput.Elem().Interface()) {
		return canonical()
	}
	built := false
	b, err := contentidentity.Transform("sitecontent/ordinary-bridge-representation/v1", key, func() ([]byte, error) {
		built = true
		r, err := compute()
		if err != nil {
			return nil, err
		}
		b, err := json.Marshal(r)
		if err != nil {
			return nil, err
		}
		var copy bridgeRepresentationData
		if json.Unmarshal(b, &copy) != nil || !reflect.DeepEqual(r, &copy) {
			return nil, fmt.Errorf("bridge representation codec is not lossless")
		}
		return b, nil
	})
	if err != nil {
		return canonical()
	}
	var r bridgeRepresentationData
	if json.Unmarshal(b, &r) != nil {
		return canonical()
	}
	// A parseable but unusable derived payload is also a miss. Re-prove the
	// exact restore on each use; storage never supplies a raw/candidate PASS.
	target := reflect.ValueOf(input).FieldByName("Reviewed")
	if !target.IsValid() || !target.CanInterface() {
		return canonical()
	}
	reviewed, ok := target.Interface().([]byte)
	if !ok || len(r.Page.Document.Units) != 1 || r.Page.Document.SourceSHA != digest(r.Page.Source) || len(r.Template) == 0 {
		return canonical()
	}
	restored, restoreErr := r.Page.Restore(string(r.Template))
	if restoreErr != nil || !bytes.Equal(bridgeEOF(restored), bridgeEOF(reviewed)) {
		return canonical()
	}
	return &r, built, nil
}

func (a observedBridgePage) BridgeSurfaceRevision(decoder i18n.TranslationUnitAdapter, raw, reviewed []byte) (i18n.TranslationUnitAdapter, []byte, []byte, error) {
	p, ok := decoder.(*surfacePageTransport)
	if !ok {
		return a.PageAdapter.BridgeSurfaceRevision(decoder, raw, reviewed)
	}
	input := struct {
		Kind          string
		Page          PageAdapter
		Transport     surfacePageTransport
		Raw, Reviewed []byte
	}{"surface-to-ordinary", a.PageAdapter, *p, raw, reviewed}
	r, _, err := deriveBridgeRepresentation(input, func() (*bridgeRepresentationData, error) {
		adapter, template, ancestor, err := a.PageAdapter.BridgeSurfaceRevision(decoder, raw, reviewed)
		if err != nil {
			return nil, err
		}
		page, ok := adapter.(PageAdapter)
		if !ok {
			return nil, fmt.Errorf("unknown ordinary bridge adapter")
		}
		return &bridgeRepresentationData{page, template, ancestor}, nil
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return r.Page, r.Template, r.Ancestor, nil
}

func (a observedBridgePage) BridgeRestrictedRevision(ancestor, old, reviewed, raw []byte, p *i18n.CommentRevisionTransport) (i18n.TranslationUnitAdapter, []byte, error) {
	if p == nil {
		return a.PageAdapter.BridgeRestrictedRevision(ancestor, old, reviewed, raw, p)
	}
	// The canonical source-owned factory reads exported mapping coordinates,
	// never the private bound decoder inside a transport slot. Capture only
	// this public data; do not persist runtime decoder bodies or QC authority.
	b, err := json.Marshal(p)
	if err != nil {
		return a.PageAdapter.BridgeRestrictedRevision(ancestor, old, reviewed, raw, p)
	}
	var public i18n.CommentRevisionTransport
	if json.Unmarshal(b, &public) != nil {
		return a.PageAdapter.BridgeRestrictedRevision(ancestor, old, reviewed, raw, p)
	}
	input := struct {
		Kind                         string
		Page                         PageAdapter
		Transport                    i18n.CommentRevisionTransport
		Ancestor, Old, Reviewed, Raw []byte
	}{"comment-to-ordinary", a.PageAdapter, public, ancestor, old, reviewed, raw}
	r, _, err := deriveBridgeRepresentation(input, func() (*bridgeRepresentationData, error) {
		adapter, template, err := a.PageAdapter.BridgeRestrictedRevision(ancestor, old, reviewed, raw, &public)
		if err != nil {
			return nil, err
		}
		page, ok := adapter.(PageAdapter)
		if !ok {
			return nil, fmt.Errorf("unknown ordinary bridge adapter")
		}
		return &bridgeRepresentationData{Page: page, Template: template}, nil
	})
	if err != nil {
		return a.PageAdapter.BridgeRestrictedRevision(ancestor, old, reviewed, raw, p)
	}
	return r.Page, r.Template, nil
}
