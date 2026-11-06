package i18n

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// A composition is an input to the existing package closure, not a Snapshot,
// rating, authorization checkpoint or second finalization state machine.
type TranslationFinalizationInput struct {
	Package          string                   `json:"package"`
	PackageAuthority string                   `json:"package_authority_sha256"`
	Finalization     GlossaryArchiveReference `json:"finalization"`
}
type TranslationPackageComposition struct {
	Schema           string                            `json:"schema"`
	Locale           string                            `json:"locale"`
	Package          string                            `json:"package"`
	PackageAuthority string                            `json:"package_authority_sha256"`
	Inputs           []TranslationFinalizationInput    `json:"predecessors"`
	Assessments      []CurrentTranslationAuthorization `json:"current_assessments"`
	Units            []QualityCheckSnapshotUnit        `json:"ordered_units"`
	Identity         string                            `json:"identity_sha256"`
}

func ComposeFinalizedTranslationPackage(root string, c *Catalog, locale string, inputs []TranslationFinalizationInput) (*TranslationPackageComposition, map[string][]byte, error) {
	pkg, authority, _, ordered, err := translationAuthorizationScope(c)
	if err != nil {
		return nil, nil, err
	}
	if len(c.TranslationScope) == 0 || len(inputs) < 2 {
		return nil, nil, fmt.Errorf("complete package and explicit predecessor finalizations required")
	}
	result := &TranslationPackageComposition{Schema: "go-learning/finalized-package-composition/v1", Locale: locale, Package: pkg, PackageAuthority: authority, Inputs: append([]TranslationFinalizationInput{}, inputs...), Assessments: []CurrentTranslationAuthorization{}, Units: []QualityCheckSnapshotUnit{}}
	seen, origins := map[string]QualityCheckSnapshotUnit{}, map[string]bool{}
	files := map[string][]byte{}
	for _, in := range inputs {
		if origins[in.Package] {
			return nil, nil, fmt.Errorf("duplicate predecessor package")
		}
		origins[in.Package] = true
		historical, err := HistoricalTranslationPackageCatalog(c, in.Package, in.PackageAuthority)
		if err != nil {
			return nil, nil, err
		}
		var f QualityCheckFinalization
		b, err := readCompatibilityFile(root, in.Finalization.Path)
		if err != nil || sum(b) != in.Finalization.SHA256 {
			return nil, nil, fmt.Errorf("composition finalization hash mismatch")
		}
		if err := contentidentity.StrictJSON(b, &f); err != nil {
			return nil, nil, err
		}
		if f.Locale != locale || in.Finalization.Path != "data/quality-check-snapshots/"+locale+"/"+f.SnapshotID+"/finalization.json" {
			return nil, nil, fmt.Errorf("composition finalization scope mismatch")
		}
		a, err := AssessCurrentTranslation(root, historical, locale, f.SnapshotID)
		if err != nil {
			return nil, nil, fmt.Errorf("composition %s assessment: %w", in.Package, err)
		}
		if !a.Ready() {
			blocked := []string{}
			for _, u := range a.Units {
				if u.Classification != "compatible" {
					blocked = append(blocked, u.Snapshot.UnitID+": "+fmt.Sprint(u.Reasons))
				}
			}
			return nil, nil, fmt.Errorf("composition predecessor not currently eligible: %s affected=%d stale=%d unknown=%d: %v", in.Package, a.Affected, a.Stale, a.Unknown, blocked)
		}
		if err := finalizationMatchesLatestPromotionCandidates(root, historical, locale, f.SnapshotID); err != nil {
			return nil, nil, err
		}
		for _, x := range a.Units {
			u := x.Snapshot
			if _, ok := seen[u.UnitID]; ok {
				return nil, nil, fmt.Errorf("overlapping predecessor Unit %s", u.UnitID)
			}
			if _, ok := files[u.SourcePath]; ok {
				return nil, nil, fmt.Errorf("overlapping predecessor output")
			}
			b, err := readCompatibilityFile(root, u.CandidatePath)
			if err != nil || sum(b) != u.CandidateSHA256 {
				return nil, nil, fmt.Errorf("composition candidate changed")
			}
			seen[u.UnitID] = u
			files[u.SourcePath] = b
		}
		result.Assessments = append(result.Assessments, *a)
	}
	if len(seen) != len(ordered) {
		return nil, nil, fmt.Errorf("composition exact-set count mismatch: got=%d expected=%d", len(seen), len(ordered))
	}
	for _, u := range ordered {
		old, ok := seen[u.ID]
		if !ok || old.UnitKind != u.Kind || old.SourcePath != u.SourcePath || old.SourceSHA256 != u.SourceSHA256 {
			return nil, nil, fmt.Errorf("composition current membership mismatch: %s", u.ID)
		}
		result.Units = append(result.Units, old)
	}
	// Return only after fresh transitive history/current-world checks. A decoded
	// historical object or previous A never exempts a mutable dependency.
	for i, a := range result.Assessments {
		h, err := HistoricalTranslationPackageCatalog(c, inputs[i].Package, inputs[i].PackageAuthority)
		if err != nil {
			return nil, nil, err
		}
		_, _, _, units, err := translationAuthorizationScope(h)
		if err != nil {
			return nil, nil, err
		}
		if err := verifyMachineAssessmentInputs(root, h, &a, units); err != nil {
			return nil, nil, err
		}
	}
	result.Identity = sum(mustJSON(*result))
	return result, files, nil
}
