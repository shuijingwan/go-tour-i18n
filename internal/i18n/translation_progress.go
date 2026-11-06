package i18n

import (
	"fmt"
	"path/filepath"
)

// TranslationUnitProgress is a read-only projection of existing lifecycle
// evidence. It is not persisted and does not add another state machine.
type TranslationUnitProgress struct {
	Total            int                       `json:"unit_count"`
	Passed           int                       `json:"passed"`
	RestoreFailed    int                       `json:"restore_failed"`
	ValidationFailed int                       `json:"validation_failed"`
	Missing          int                       `json:"missing"`
	Stale            int                       `json:"stale"`
	Units            []RetranslationUnitResult `json:"units"`
}

func BuildTranslationUnitProgress(root string, catalog *Catalog, locale string) (*TranslationUnitProgress, error) {
	if err := RequireNewLanguageGlossaryReview(root, locale); err != nil {
		return nil, err
	}
	latest, err := selectLatestRetranslationUnits(root, catalog, locale)
	if err != nil {
		return nil, err
	}
	glossary, err := LoadGlossary(root, locale)
	if err != nil {
		return nil, err
	}
	compatibility := PrepareGlossaryCompatibility(root, locale, catalog)
	r := &TranslationUnitProgress{Total: len(latest.ordered), Units: []RetranslationUnitResult{}}
	for _, unit := range latest.ordered {
		choice, ok := latest.selectedByID[unit.ID]
		row := RetranslationUnitResult{UnitID: unit.ID, UnitKind: unit.Kind, Status: "missing"}
		if !ok {
			r.Missing++
			r.Units = append(r.Units, row)
			continue
		}
		row = choice.result
		base := filepath.ToSlash(filepath.Join("data/retranslation-runs", locale, choice.batchID))
		if choice.machine != nil {
			base = ""
		}
		if base != "" {
			row.ValidationPath = base + "/" + row.ValidationPath
			if row.CandidatePath != "" {
				row.CandidatePath = base + "/" + row.CandidatePath
			}
		}
		switch row.Status {
		case "restore_failed":
			r.RestoreFailed++
		case "validation_failed":
			r.ValidationFailed++
		case "passed":
			if _, err := validateSelectedRetranslationCandidate(root, catalog, locale, glossary, unit, choice, compatibility); err != nil {
				row.Status = "stale"
				row.Error = err.Error()
				r.Stale++
			} else {
				r.Passed++
			}
		default:
			return nil, fmt.Errorf("unknown shared lifecycle status %q", row.Status)
		}
		r.Units = append(r.Units, row)
	}
	return r, nil
}
