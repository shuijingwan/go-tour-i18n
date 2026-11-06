package i18n

import (
	"fmt"
	"reflect"
)

// A command-owned extension of the existing Catalog projection reader. Only
// immutable reconstruction is reused; this layer always observes disk anew.
// Adapters must be privately owned by the resolver, never a cached PASS.
type SameUnitBridgeResolution struct {
	Receipt  *QualityCheckSurfaceReopen
	Adapters map[string]TranslationUnitAdapter
}
type SameUnitBridgeResolver func(string, *Catalog, string, GlossaryArchiveReference, func() (*SameUnitBridgeResolution, error)) (*SameUnitBridgeResolution, error)

// Supplies a private construction Catalog, not an authorization verdict. The
// source provider may share pure representation derivations and an explicit
// readonly proof lifetime; all current/frozen readers still run inside work.
type SameUnitBridgeConstructionContext func(string, *Catalog, func(*Catalog) (*SameUnitBridgeResolution, error)) (*SameUnitBridgeResolution, error)

func readSameUnitBridgeForInvocation(root string, c *Catalog, locale string, ref GlossaryArchiveReference) (*QualityCheckSurfaceReopen, map[string]protectedTranslation, error) {
	if c == nil || c.BridgeResolver == nil {
		return readSameUnitBridge(root, c, locale, ref)
	}
	r, err := readFrozenSameUnitBridge(root, locale, ref)
	if err != nil {
		return nil, nil, err
	}
	// These are fresh observations, including Review selection (new files can
	// change it), full package membership, and executable adapter/policy identity.
	check := func() error {
		if err := verifySameUnitBridgeDependencies(root, r); err != nil {
			return err
		}
		scoped, err := SelectCurrentValidationReopenPackage(c, r.SameUnitBridge.Options.ContentPackage)
		if err != nil {
			return err
		}
		_, identity, _, units, err := translationAuthorizationScope(scoped)
		if err != nil || identity != r.SameUnitBridge.PackageIdentity || len(units) != len(r.SameUnitBridge.OrderedUnits) {
			return fmt.Errorf("bridge current package drift")
		}
		for i, u := range units {
			if u.ID != r.SameUnitBridge.OrderedUnits[i] {
				return fmt.Errorf("bridge current ordered scope drift")
			}
			disk, err := readTranslationUnitSource(root, u)
			if err != nil || validateSnapshotSourceIdentity(scoped, u, disk) != nil {
				return fmt.Errorf("bridge current package source drift: %s", u.ID)
			}
		}
		if err := RequireCurrentGlossaryReview(root, locale); err != nil {
			return err
		}
		review, err := RequireUnifiedGlossaryReview(root, locale)
		if err != nil {
			return err
		}
		found := false
		for _, d := range r.SameUnitBridge.Dependencies {
			if d == review {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("bridge current Review selection drift")
		}
		for _, b := range r.SameUnitBridge.Bindings {
			u, err := scoped.Unit(b.UnitID)
			if err != nil {
				return err
			}
			got, err := currentTranslationAuthority(root, u, locale)
			if err != nil || got != b.CurrentAuthority {
				return fmt.Errorf("bridge current adapter/validator drift")
			}
		}
		return nil
	}
	if err := check(); err != nil {
		return nil, nil, err
	}
	result, err := c.BridgeResolver(root, c, "current", ref, func() (*SameUnitBridgeResolution, error) {
		work := func(construction *Catalog) (*SameUnitBridgeResolution, error) {
			receipt, outputs, err := readSameUnitBridge(root, construction, locale, ref)
			if err != nil {
				return nil, err
			}
			result := &SameUnitBridgeResolution{Receipt: receipt, Adapters: map[string]TranslationUnitAdapter{}}
			for id, p := range outputs {
				result.Adapters[id] = p.Adapter
			}
			return result, nil
		}
		if c.BridgeConstructionContext != nil {
			return c.BridgeConstructionContext(root, c, work)
		}
		return work(c)
	})
	if err != nil {
		return nil, nil, err
	}
	if result == nil || !reflect.DeepEqual(result.Receipt, r) || len(result.Adapters) != len(r.Units) {
		return nil, nil, fmt.Errorf("bridge reconstruction scope differs")
	}
	g, err := LoadGlossary(root, locale)
	if err != nil {
		return nil, nil, err
	}
	outputs := map[string]protectedTranslation{}
	for _, u := range r.Units {
		a := result.Adapters[u.UnitID]
		if a == nil {
			return nil, nil, fmt.Errorf("bridge decoder missing")
		}
		text, tokens, err := a.ProtectedInput(g)
		if err != nil {
			return nil, nil, err
		}
		outputs[u.UnitID] = protectedTranslation{Text: text, Tokens: tokens, Adapter: a}
	}
	if err := check(); err != nil {
		return nil, nil, err
	}
	// Receipt bytes themselves are not among their own dependency refs.
	fresh, err := readFrozenSameUnitBridge(root, locale, ref)
	if err != nil || !reflect.DeepEqual(fresh, r) {
		return nil, nil, fmt.Errorf("bridge receipt changed during reconstruction")
	}
	return r, outputs, nil
}

func bridgeOutputForInvocation(root string, c *Catalog, g *Glossary, locale string, record RetranslationBatchUnit) (protectedTranslation, error) {
	if record.RepresentationBridge == nil {
		return protectedTranslation{}, fmt.Errorf("explicit bridge required")
	}
	r, outputs, err := readSameUnitBridgeForInvocation(root, c, locale, *record.RepresentationBridge)
	if err != nil {
		return protectedTranslation{}, err
	}
	for _, q := range r.Units {
		if q.UnitID == record.UnitID && record.RevisionFeedbackSource == "quality_check" && record.PreviousSnapshotID == r.PreviousSnapshotID && record.PreviousRating == q.PreviousRating && record.PreviousFinding == q.Finding {
			return outputs[q.UnitID], nil
		}
	}
	return protectedTranslation{}, fmt.Errorf("bridge per-Unit QC feedback mismatch")
}
func prepareInvocationRetranslationOutput(root string, c *Catalog, g *Glossary, locale string, selected []RetranslationBatchUnit, record RetranslationBatchUnit) (protectedTranslation, error) {
	if p, lineage, err := surfaceTransportOutput(root, c, locale, record); err != nil || lineage != nil {
		return p, err
	}
	if record.RepresentationBridge != nil {
		return bridgeOutputForInvocation(root, c, g, locale, record)
	}
	p, lineage, err := inheritedBridgeOutput(root, c, g, locale, record)
	if err != nil || lineage != nil {
		return p, err
	}
	return prepareRetranslationOutput(root, c, g, locale, selected, record)
}
