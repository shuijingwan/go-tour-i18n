package i18n

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// ParsedGlossaryDelta exposes the V2-B normalization algorithm for formal
// surface parsers. It does not create a new glossary or a Review decision.
func ParsedGlossaryDelta(locale string, oldBytes, newBytes []byte) ([]GlossarySemanticDelta, error) {
	return glossarySemanticDelta(locale, oldBytes, newBytes)
}

// ParseGlossaryBytes uses the same strict semantic normalization as impact
// assessment, so frozen structured evidence need not read today's glossary.
func ParseGlossaryBytes(locale string, data []byte) (*Glossary, error) {
	entries, err := normalizeCompatibilityGlossary(locale, data)
	if err != nil {
		return nil, err
	}
	g := &Glossary{Locale: locale, Mandatory: map[string]string{}, Preferred: map[string]string{}}
	for _, e := range entries {
		switch e.Category {
		case "mandatory":
			g.Mandatory[e.Term] = e.Value
		case "preferred":
			g.Preferred[e.Term] = e.Value
		case "forbidden":
			g.Forbidden = append(g.Forbidden, e.Term)
		case "keep":
			g.Keep = append(g.Keep, e.Term)
		}
	}
	return g, nil
}

// ParsedGlossaryImpact accepts only text already selected by a formal parser.
// Unknown parsers remain affected; source/target protection belongs to that
// caller's versioned parser and its exact context evidence.
func ParsedGlossaryImpact(source, target string, delta []GlossarySemanticDelta) (bool, []string) {
	return classifyGlossaryContext(GlossaryCompatibilityContext{Parser: "plain-visible/v1", Source: source, Target: target}, delta)
}

func ValidateParsedGlossary(source, target string, g *Glossary) error {
	for _, term := range g.Forbidden {
		if compatibilityForbiddenTermPresent(target, term) {
			return fmt.Errorf("forbidden glossary text %q", term)
		}
	}
	for term, want := range g.Mandatory {
		if source == term && target != want {
			return fmt.Errorf("mandatory exact label %q requires %q", term, want)
		}
	}
	for _, term := range g.Keep {
		// Keep is an exact technical identity, not compatibility impact's
		// case-insensitive English term/inflection discovery. Reuse the shared
		// protector's spelling, word boundaries, target-language adjacency and
		// explicit ordinary-language exceptions on already-visible parser text.
		keep := &Glossary{Keep: []string{term}}
		if len(translationKeepProtectionSpans(source, keep, nil, nil)) != 0 && len(translationKeepProtectionSpans(target, keep, nil, nil)) == 0 {
			return fmt.Errorf("keep term %q changed", term)
		}
	}
	return nil
}

// ValidateParsedGlossaryReview reuses the existing full independent Review gate
// for an archived glossary edge; it never accepts a subset Review.
func ValidateParsedGlossaryReview(root, locale, sha string, ref GlossaryArchiveReference) error {
	return validateCompatibilityReview(root, locale, sha, ref)
}
func CurrentParsedGlossaryReview(root, locale, sha string) (GlossaryArchiveReference, error) {
	return currentGlossaryReviewReference(root, locale, sha)
}

// CheckHistoricalUnifiedGlossaryReview proves the original complete review
// against its immutable archive, without requiring today's corpus/glossary.
func CheckHistoricalUnifiedGlossaryReview(root, locale, glossarySHA, corpusSHA string, ref GlossaryArchiveReference) error {
	b, err := readCompatibilityFile(root, ref.Path)
	if err != nil || sum(b) != ref.SHA256 {
		return fmt.Errorf("historical Unified Review hash mismatch")
	}
	var r UnifiedGlossaryReview
	if err := contentidentity.StrictJSON(b, &r); err != nil {
		return err
	}
	if err := validateUnifiedReview(r, locale); err != nil {
		return err
	}
	if r.Decision != "passed" || r.GlossarySHA != glossarySHA || r.CorpusSHA != corpusSHA || ref.Path != UnifiedGlossaryReviewPath(locale, r.ReviewID) {
		return fmt.Errorf("historical full Unified Review binding mismatch")
	}
	return validateUnifiedReviewBundle(root, r)
}

// LocaleLanguageContexts exposes the established Tour compatibility inventory
// to the versionless locale work planner, without creating translation units.
func LocaleLanguageContexts(root, locale string, catalog *Catalog) ([]GlossaryCompatibilityContext, error) {
	return currentGlossaryCompatibilityContexts(root, locale, catalog)
}
