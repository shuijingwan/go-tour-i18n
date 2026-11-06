package sitecontent

import (
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

type historicalTranslationContextKey struct {
	locale    string
	reference i18n.GlossaryArchiveReference
}

type historicalTranslationContext struct {
	catalog   *i18n.Catalog
	proof     *i18n.HistoricalFinalizationIntegrity
	reference i18n.GlossaryArchiveReference
}

// The caller has verified the complete original A proof. Keep its exact frozen
// reference separate from the physical observation used by the invocation guard.
// Only the canonical historical reader can explain a SHA difference; ordinary
// bind/readReference retain their strict current-physical contract.
func (v *closureValidation) bindHistoricalTranslationReference(c *i18n.Catalog, proof *i18n.HistoricalFinalizationIntegrity, ref i18n.GlossaryArchiveReference) error {
	if _, err := v.readRegular(v.root, ref.Path); err != nil {
		return err
	}
	frozen, err := i18n.ReadHistoricalTranslationReference(v.root, c, proof, ref)
	if err != nil {
		return err
	}
	if digest(frozen) != ref.SHA256 {
		return fmt.Errorf("historical translation dependency hash mismatch: %s", ref.Path)
	}
	if v.deps[ref.Path] == ref.SHA256 {
		return nil
	}
	// Never put the historical SHA in deps/files: those maps describe actual
	// physical bytes/modes and must detect even a lawful head change mid-proof.
	key := historicalTranslationContextKey{proof.Finalization.Locale, ref}
	context := historicalTranslationContext{c, proof, ref}
	if v.historicalContexts == nil {
		v.historicalContexts = map[historicalTranslationContextKey]historicalTranslationContext{}
	}
	v.historicalContexts[key] = context
	for _, frame := range v.frames {
		if frame.historicalContexts == nil {
			frame.historicalContexts = map[historicalTranslationContextKey]historicalTranslationContext{}
		}
		frame.historicalContexts[key] = context
	}
	return nil
}
