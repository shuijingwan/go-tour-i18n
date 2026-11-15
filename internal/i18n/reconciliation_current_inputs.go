package i18n

import (
	"bytes"
	"fmt"
)

// One readonly QC evaluation owns these common language inputs. No per-Unit
// classification/validation or currentness answer is stored. Every Unit still
// runs the canonical assessment, and the owner re-observes the full Review and
// glossary before any effective graph leaves the phase.
type reconciliationCurrentInputs struct {
	root, locale string
	current      []byte
	glossary     *Glossary
	review       GlossaryArchiveReference
}

func (p *reconciliationCurrentInputs) prepare(root, locale string) error {
	if p.current != nil {
		if root != p.root || locale != p.locale {
			return fmt.Errorf("reconciliation input scope changed")
		}
		return nil
	}
	review, err := RequireUnifiedGlossaryReview(root, locale)
	if err != nil {
		return err
	}
	current, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(locale))
	if err != nil {
		return err
	}
	g, err := parseGlossary(locale, bytes.NewReader(current))
	if err != nil {
		return err
	}
	p.root, p.locale, p.current, p.glossary, p.review = root, locale, current, g, review
	return nil
}

func (p *reconciliationCurrentInputs) check() error {
	if p.current == nil {
		return nil
	}
	current, err := readCompatibilityFile(p.root, glossaryReviewGlossaryPath(p.locale))
	if err != nil || !bytes.Equal(current, p.current) {
		return fmt.Errorf("reconciliation assessment glossary drift")
	}
	review, err := RequireUnifiedGlossaryReview(p.root, p.locale)
	if err != nil || review != p.review {
		return fmt.Errorf("reconciliation assessment Unified Review drift: %v", err)
	}
	return nil
}
