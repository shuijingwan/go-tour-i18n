package i18n

import (
	"bytes"
	"fmt"
)

// Reproduce the original language authorization on its frozen A baseline.
// This intentionally does not call the consumed/latest export-right builder.
// The new recovery receipt, not this assessment, renews invocation rights.
func assessRecoveryLanguageAuthority(root string, c *Catalog, m *RetranslationBatchManifest, reopen *QualityCheckSurfaceReopen, g *Glossary) (*CurrentTranslationAuthorization, []RecoveryDependency, error) {
	frozen, err := decodeCurrentValidationAssessment(reopen.CurrentValidation)
	if err != nil {
		return nil, nil, err
	}
	fresh, err := AssessCurrentTranslation(root, c, m.Locale, m.Units[0].PreviousSnapshotID)
	if err != nil {
		return nil, nil, err
	}
	if fresh.Locale != frozen.Locale || fresh.Package != frozen.Package || fresh.PackageAuthoritySHA != frozen.PackageAuthoritySHA || fresh.ScopeIdentity != frozen.ScopeIdentity || fresh.Algorithm != frozen.Algorithm || fresh.Normalization != frozen.Normalization || fresh.Glossary != frozen.Glossary || fresh.Stale != 0 || fresh.Unknown != 0 || len(fresh.Units) != len(frozen.Units) || fresh.Affected < len(m.Units) {
		return nil, nil, fmt.Errorf("original language authority/source/glossary/scope is no longer current")
	}
	if !bytes.Equal(mustJSON(fresh.Units), mustJSON(frozen.Units)) {
		return nil, nil, fmt.Errorf("original typed language/protection/finding assessment changed")
	}
	selected := map[string]RetranslationBatchUnit{}
	for _, r := range m.Units {
		selected[r.UnitID] = r
	}
	pos, checked := 0, 0
	for _, u := range fresh.Units {
		if u.Classification != "affected" {
			continue
		}
		if pos >= len(reopen.CurrentValidation.Findings) {
			return nil, nil, fmt.Errorf("original finding set incomplete")
		}
		f := reopen.CurrentValidation.Findings[pos]
		pos++
		r, ok := selected[u.Snapshot.UnitID]
		if !ok {
			continue
		}
		if checked >= len(m.Units) || m.Units[checked].UnitID != r.UnitID {
			return nil, nil, fmt.Errorf("original batch finding order differs from reopen")
		}
		if u.Snapshot.UnitID != r.UnitID || f.UnitID != r.UnitID || f.ValidationIdentity != u.ValidationIdentity || r.PreviousFinding != currentValidationFinding(f) {
			return nil, nil, fmt.Errorf("original ordered finding authority mismatch")
		}
		unit, err := c.Unit(r.UnitID)
		if err != nil {
			return nil, nil, err
		}
		candidate, err := readCompatibilityReference(root, GlossaryArchiveReference{u.Snapshot.CandidatePath, u.Snapshot.CandidateSHA256})
		if err != nil {
			return nil, nil, err
		}
		validation := validateTranslationUnitWithGlossary(root, c, unit, m.Locale, candidate, g)
		if validation == nil || validation.Error() != f.Reason {
			return nil, nil, fmt.Errorf("original machine finding no longer exactly reproducible")
		}
		checked++
	}
	if checked != len(m.Units) || pos != len(reopen.CurrentValidation.Findings) {
		return nil, nil, fmt.Errorf("recovery does not exactly cover original finding scope")
	}
	var refs []RecoveryDependency
	for _, old := range frozen.Historical.References {
		r, _, err := recoveryDiskReference(root, old.Path)
		if err != nil || r.SHA256 != old.SHA256 {
			return nil, nil, fmt.Errorf("original A/finalization authority reference drift")
		}
		refs = append(refs, RecoveryDependency{"original-language-authority", r})
	}
	return fresh, refs, nil
}
