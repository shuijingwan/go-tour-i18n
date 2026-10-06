package i18n

import "fmt"

// ParsedGlossaryDelta exposes the V2-B normalization algorithm for formal
// surface parsers. It does not create a new glossary or a Review decision.
func ParsedGlossaryDelta(locale string, oldBytes, newBytes []byte) ([]GlossarySemanticDelta, error) {
	return glossarySemanticDelta(locale, oldBytes, newBytes)
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
		if compatibilitySourceTermPresent(source, term) && !compatibilitySourceTermPresent(target, term) {
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

// LocaleLanguageContexts exposes the established Tour compatibility inventory
// to the versionless locale work planner, without creating translation units.
func LocaleLanguageContexts(root, locale string, catalog *Catalog) ([]GlossaryCompatibilityContext, error) {
	return currentGlossaryCompatibilityContexts(root, locale, catalog)
}
