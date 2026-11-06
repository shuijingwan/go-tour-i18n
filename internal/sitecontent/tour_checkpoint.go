package sitecontent

import (
	"fmt"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const tourCheckpointReason = "exact-current Surface A checkpoint with independent TU/glossary/canonical/SEO authority"

// A one-time modern baseline is a composition of existing independent gates,
// not a new closure or permission to carry affected derived consumers via TU QC.
func currentTourCheckpoint(root string, g *Global, catalog *i18n.Catalog, l Locale, validation ...*closureValidation) ([]Reference, bool, error) {
	if err := ValidateLocale(g, l); err != nil {
		return nil, false, err
	}
	legacy := false
	for _, c := range l.Packages {
		if c.Package == "tour-v1" && c.State == "complete" && c.EvidenceKind == "legacy-tour-closure/v1" {
			legacy = true
		}
	}
	if !legacy {
		return nil, false, nil
	}
	// The caller may retain a decoded locale across a mutation checkpoint, or
	// supply only the Tour projection. Re-read the complete authority and prove
	// every supplied completion still belongs to those exact current bytes.
	closurePath := LocalePath(l.Locale)
	closureBytes, err := readRegular(root, closurePath)
	if err != nil {
		return nil, false, err
	}
	closureRef := Reference{Path: closurePath, SHA256: digest(closureBytes)}
	var persisted Locale
	if err := StrictJSON(closureBytes, &persisted); err != nil {
		return nil, false, fmt.Errorf("Tour checkpoint locale closure: %w", err)
	}
	if persisted.Locale != l.Locale {
		return nil, false, fmt.Errorf("Tour checkpoint locale closure mismatch")
	}
	if err := ValidateLocale(g, persisted); err != nil {
		return nil, false, err
	}
	for _, expected := range l.Packages {
		matched := false
		for _, actual := range persisted.Packages {
			if reflect.DeepEqual(expected, actual) {
				matched = true
				break
			}
		}
		if !matched {
			return nil, false, fmt.Errorf("Tour checkpoint locale completion changed: %s", expected.Package)
		}
	}
	if len(validation) > 0 {
		if err := validation[0].bind(closureRef); err != nil {
			return nil, false, err
		}
	}
	gate, err := i18n.CurrentExactLocaleSurfaceReviewAGate(root, l.Locale, catalog)
	if err != nil {
		return nil, false, nil
	} // No checkpoint: existing fail-closed path.
	// Discovery supplies package identity without selecting its workflow scope
	// or changing the caller's Tour catalog.
	copy := *catalog
	catalog = &copy
	if len(catalog.TranslationPackages) == 0 {
		var successors *structuredSuccessorValidation
		if len(validation) > 0 && validation[0] != nil {
			successors = validation[0].readonlySuccessors
		}
		if err := AttachRegisteredTranslationPackages(root, catalog, successors); err != nil {
			return nil, false, err
		}
	}
	AttachIntegratedSurfaceAuthority(catalog)
	if len(validation) > 0 {
		validation[0].attachReplacementReuse(catalog)
	}
	review, err := i18n.RequireUnifiedGlossaryReview(root, l.Locale)
	if err != nil {
		return nil, false, err
	}
	_, tuRefs, err := i18n.RequireCurrentCanonicalQualityCheck(root, l.Locale, catalog)
	if err != nil {
		return nil, false, err
	}
	if _, err := i18n.LoadCourseMetadata(root, l.Locale, catalog); err != nil {
		return nil, false, err
	}
	base := "data/locale-surface-reviews/" + l.Locale + "/" + gate.ReviewID
	refs, err := sortedReferences(root, []string{LocalePath(l.Locale), base + ".a-gate.json", base + ".md"})
	if err != nil {
		return nil, false, err
	}
	// sortedReferences reads the files again. Never bind a successful check to
	// different closure bytes observed after its independent validations.
	for _, ref := range refs {
		if ref.Path == closureRef.Path && ref != closureRef {
			return nil, false, fmt.Errorf("Tour checkpoint locale closure changed during validation")
		}
	}
	refs = append(refs, Reference{Path: review.Path, SHA256: review.SHA256})
	for _, r := range tuRefs {
		refs = append(refs, Reference{Path: r.Path, SHA256: r.SHA256})
	}
	return refs, true, nil
}
