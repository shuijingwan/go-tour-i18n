package i18n

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var ErrRevisionTemplateIncapable = errors.New("ordinary QC protected predecessor is not executable by its response decoder; representation recovery authority required")

// Test the editing template against the exact decoder used for this new
// response. Historical restore alone is insufficient: a predecessor may have
// used a restricted, finding-authorized decoder which ordinary QC cannot use.
// This checks no finding language and never re-protects the reviewed candidate.
func validateOrdinaryQCRevisionTemplates(root string, c *Catalog, g *Glossary, m *RetranslationBatchManifest, inputs []generationBundleInput) error {
	byPath := map[string][]byte{}
	for _, input := range inputs {
		byPath[input.path] = input.data
	}
	for _, r := range m.Units {
		if r.RevisionFeedbackSource != "quality_check" {
			continue
		}
		u, err := c.Unit(r.UnitID)
		if err != nil {
			return err
		}
		decoder, err := prepareTranslationUnitInput(u, g)
		if err != nil {
			return err
		}
		if r.RepresentationBridge != nil || byPath["formal/representation-lineage/"+r.UnitID+".json"] != nil {
			decoder, err = prepareInvocationRetranslationOutput(root, c, g, m.Locale, m.Units, r)
			if err != nil {
				return err
			}
		}
		if response, lineage, err := inheritedBridgeOutput(root, c, g, m.Locale, r); err != nil {
			return err
		} else if lineage != nil && strings.HasPrefix(lineage.Bridge.Path, "data/surface-revision-transports/") {
			unrestricted, ok := response.Adapter.(interface{ UnrestrictedRevisionTransport() bool })
			if !ok || !unrestricted.UnrestrictedRevisionTransport() {
				return fmt.Errorf("%s: %w: Surface-bounded transport is not generic QC edit authority", r.UnitID, ErrRevisionTemplateIncapable)
			}
		}
		name := filepath.Base(r.InputPath)
		baseline, ok := byPath["formal/protected-candidates/"+name]
		candidate, present := byPath["formal/candidates/"+name]
		if !ok || !present {
			return fmt.Errorf("%s: missing QC predecessor representation", r.UnitID)
		}
		restored, failures := decoder.restore(string(baseline))
		b := []byte(restored)
		want := candidate
		if m.ArtifactEOF == retranslationArtifactEOFSingleLF {
			b = canonicalizeRetranslationArtifactEOF(b)
			want = canonicalizeRetranslationArtifactEOF(want)
		}
		if len(failures) != 0 || !bytes.Equal(b, want) {
			return fmt.Errorf("%s: %w", r.UnitID, ErrRevisionTemplateIncapable)
		}
	}
	return nil
}
