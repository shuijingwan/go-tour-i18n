package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

func compatibleHistoricalProtectedInput(root string, catalog *Catalog, locale string, unit *TranslationUnit, record RetranslationBatchUnit, input, candidate []byte) (protectedTranslation, error) {
	current, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(locale))
	if err != nil {
		return protectedTranslation{}, err
	}
	entries, err := os.ReadDir(filepath.Join(root, "data", "quality-check-snapshots", locale))
	if err != nil {
		return protectedTranslation{}, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		snapshot, err := readQualityCheckSnapshot(root, locale, entry.Name(), false)
		if err != nil {
			return protectedTranslation{}, err
		}
		for _, s := range snapshot.Units {
			if s.UnitID != unit.ID || s.SourceSHA256 != unit.SourceSHA256 || s.CandidateSHA256 != sum(candidate) {
				continue
			}
			// The actual repository evidence proves batch/source/input/validation
			// provenance; a merely matching candidate hash is insufficient.
			evidence, err := readSnapshotUnitRepositoryEvidence(root, catalog, locale, s)
			if err != nil || evidence.record.InputSHA256 != record.InputSHA256 || evidence.record.InputPath != record.InputPath {
				continue
			}
			if !glossaryScopeCompatible(root, locale, snapshot.GlossarySHA256, sum(current), "tu:"+unit.ID, catalog) {
				continue
			}
			data, err := readArchivedGlossary(root, locale, snapshot.GlossarySHA256)
			if err != nil {
				continue
			}
			normalized, err := normalizeCompatibilityGlossary(locale, data)
			if err != nil {
				continue
			}
			g := &Glossary{Locale: locale, Mandatory: map[string]string{}, Preferred: map[string]string{}}
			for _, e := range normalized {
				switch e.Category {
				case "mandatory":
					g.Mandatory[e.Term] = e.Value
				case "preferred":
					g.Preferred[e.Term] = e.Value
				case "keep":
					g.Keep = append(g.Keep, e.Term)
				case "forbidden":
					g.Forbidden = append(g.Forbidden, e.Term)
				}
			}
			protected, err := prepareTranslationUnitInput(unit, g)
			if err != nil {
				continue
			}
			if len(protected.Tokens) == record.ProtectedTokenCount && (bytes.Equal(input, []byte(protected.Text)) || bytes.Equal(input, canonicalizeRetranslationArtifactEOF([]byte(protected.Text)))) {
				return protected, nil
			}
		}
	}
	return protectedTranslation{}, fmt.Errorf("no exact historical protected-input compatibility proof")
}
