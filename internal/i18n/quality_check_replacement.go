package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
)

const QCRepresentationFeedback = "quality_check_representation"

// Additive authority on the existing replacement reopen. It binds the original
// non-A QC judgment without inventing a Surface finding or inherited rating.
type QualityCheckReplacementAuthority struct {
	Snapshot         GlossaryArchiveReference `json:"snapshot"`
	Results          GlossaryArchiveReference `json:"results"`
	EngineeringProof GlossaryArchiveReference `json:"engineering_proof"`
}
type RepresentationNecessityProof struct {
	Reason string                        `json:"reason"`
	Units  []RepresentationNecessityUnit `json:"units"`
}
type RepresentationNecessityUnit struct {
	Replacement   TranslationUnitReplacement `json:"replacement"`
	Rating        string                     `json:"rating"`
	FindingSHA256 string                     `json:"finding_sha256"`
}

func qualityCheckReplacementExpected(root string, c *Catalog, locale, id, snapshot string, context *TranslationUnitReplacementContext, currentIDs []string) (*QualityCheckSurfaceReopen, error) {
	if context == nil || context.QualityCheck == nil || context.Machine != nil || context.PredecessorReopen.Path != "" || context.PredecessorReopen.SHA256 != "" {
		return nil, fmt.Errorf("QC replacement requires exclusive authority")
	}
	resolution, err := replacementResolution(root, c, context)
	if err != nil {
		return nil, err
	}
	a := context.QualityCheck
	base := filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, snapshot))
	for _, ref := range []GlossaryArchiveReference{a.Snapshot, a.Results, a.EngineeringProof} {
		b, err := readSnapshotReferencedFile(root, ref.Path)
		if err != nil || !validSHA256(ref.SHA256) || sum(b) != ref.SHA256 {
			return nil, fmt.Errorf("QC replacement authority drift: %s", ref.Path)
		}
	}
	if a.Snapshot.Path != base+"/manifest.json" || a.Results.Path != base+"/quality-check-results.json" {
		return nil, fmt.Errorf("QC replacement Snapshot/result authority mismatch")
	}
	b, err := readSnapshotReferencedFile(root, a.EngineeringProof.Path)
	if err != nil {
		return nil, err
	}
	var proof RepresentationNecessityProof
	if err = decodeStrictCourseSourceDescriptionReviewJSON(b, &proof); err != nil {
		return nil, err
	}
	if proof.Reason != "protected-natural-language-unrepresentable" || len(proof.Units) != len(currentIDs) {
		return nil, fmt.Errorf("exact engineering necessity proof required")
	}
	eval := newQualityCheckEvaluation(root, locale, resolution.HistoricalCatalog)
	s, results, err := eval.loadEffectiveQualityCheckResults(snapshot, map[string]bool{})
	if err != nil {
		return nil, err
	}
	if len(results) != s.UnitCount {
		return nil, fmt.Errorf("complete initial independent QC before concentrated reconciliation: effective=%d expected=%d historical_scope=%d", len(results), s.UnitCount, len(resolution.HistoricalCatalog.TranslationScope))
	}
	byID := map[string]QualityCheckSnapshotUnit{}
	for _, u := range s.Units {
		byID[u.UnitID] = u
	}
	selected := map[string]bool{}
	for _, id := range currentIDs {
		if selected[id] {
			return nil, fmt.Errorf("duplicate replacement Unit")
		}
		selected[id] = true
	}
	r := &QualityCheckSurfaceReopen{AuthoritySource: "quality-check", SchemaVersion: qualityCheckSurfaceReopenSchemaVersion, EvidenceType: qualityCheckSurfaceReopenEvidenceType, Locale: locale, ReopenID: id, PreviousSnapshotID: snapshot, ReplacementContext: context, Units: []QualityCheckSurfaceReopenUnit{}}
	for _, pu := range proof.Units {
		p, ok := resolution.Units[pu.Replacement.OldUnitID]
		if !ok || !reflect.DeepEqual(p, pu.Replacement) || p.OldUnitID == p.CurrentUnitID || !selected[p.CurrentUnitID] {
			return nil, fmt.Errorf("engineering proof/replacement exact set mismatch")
		}
		delete(selected, p.CurrentUnitID)
		u, ok := byID[p.OldUnitID]
		result, found := results[p.OldUnitID]
		if !ok || !found || result.rubric != TranslationQualityRubric || !qualityCheckSnapshotIdentityMatches(u, result.unit) || (result.rating != "B" && result.rating != "C" && result.rating != "D") || result.finding == "" || pu.Rating != result.rating || pu.FindingSHA256 != sum([]byte(result.finding)) {
			return nil, fmt.Errorf("exact QC B/C/D finding required")
		}
		e, err := eval.readSnapshotUnitRepositoryEvidence(snapshot, u)
		if err != nil || e.record.InputSHA256 != p.OldInputSHA256 {
			return nil, fmt.Errorf("historical QC protected input mismatch: %v", err)
		}
		r.Units = append(r.Units, QualityCheckSurfaceReopenUnit{PreviousRating: result.rating, Replacement: &p, Index: u.Index, UnitID: p.CurrentUnitID, Finding: result.finding, Snapshot: u})
	}
	if len(selected) > 0 || len(r.Units) == 0 || len(r.Units) > MaxRetranslationExportLimit {
		return nil, fmt.Errorf("exact bounded QC replacement scope required")
	}
	sort.Slice(r.Units, func(i, j int) bool { return r.Units[i].Index < r.Units[j].Index })
	r.UnitCount = len(r.Units)
	return r, nil
}

func verifyQualityCheckReplacement(root string, c *Catalog, r *QualityCheckSurfaceReopen) error {
	if r.AuthoritySource == RepresentationMachineAuthority {
		return verifyRepresentationRevision(root, c, r)
	}
	ids := []string{}
	for _, u := range r.Units {
		ids = append(ids, u.UnitID)
	}
	want, err := qualityCheckReplacementExpected(root, c, r.Locale, r.ReopenID, r.PreviousSnapshotID, r.ReplacementContext, ids)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(want, r) {
		return fmt.Errorf("QC replacement differs from exact authority")
	}
	return nil
}

// InspectQualityCheckRepresentationReopen runs the same exact QC replacement
// authorization as record, without creating a receipt or language artifact.
func InspectQualityCheckRepresentationReopen(root string, c *Catalog, o QualityCheckSurfaceReopenOptions) (*QualityCheckSurfaceReopen, string, error) {
	if o.AuthoritySource != "quality-check" || o.CurrentValidation != nil {
		return nil, "", fmt.Errorf("exclusive quality-check representation authority required")
	}
	if err := ValidateLocaleName(o.Locale); err != nil {
		return nil, "", err
	}
	if !reviewIDPattern.MatchString(o.ReopenID) || o.PreviousReopenID != "" || o.SurfaceReviewID != "" || o.IntegratedSurface || o.PreviewFailure.Path != "" || len(o.UnitIDs) == 0 || len(o.UnitIDs) > MaxRetranslationExportLimit {
		return nil, "", fmt.Errorf("exclusive exact QC reconciliation authority required")
	}
	if err := validateSnapshotID(o.PreviousSnapshotID); err != nil {
		return nil, "", err
	}
	base := filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", o.Locale, o.PreviousSnapshotID))
	a := &QualityCheckReplacementAuthority{EngineeringProof: o.RepresentationProof}
	for _, entry := range []struct {
		path string
		ref  *GlossaryArchiveReference
	}{{base + "/manifest.json", &a.Snapshot}, {base + "/quality-check-results.json", &a.Results}} {
		b, err := readSnapshotReferencedFile(root, entry.path)
		if err != nil {
			return nil, "", err
		}
		*entry.ref = GlossaryArchiveReference{Path: entry.path, SHA256: sum(b)}
	}
	context := &TranslationUnitReplacementContext{QualityCheck: a, HistoricalDocuments: o.HistoricalDocuments}
	r, err := qualityCheckReplacementExpected(root, c, o.Locale, o.ReopenID, o.PreviousSnapshotID, context, o.UnitIDs)
	if err != nil {
		return nil, "", err
	}
	if len(o.Findings) != 0 && len(o.Findings) != len(r.Units) {
		return nil, "", fmt.Errorf("exact ordered QC findings required")
	}
	for i, u := range r.Units {
		finding := o.Finding
		if len(o.Findings) > 0 {
			finding = o.Findings[i]
		}
		if u.Finding != finding || u.PreviousRating != o.PreviousRating {
			return nil, "", fmt.Errorf("requested QC rating/finding mismatch")
		}
	}
	if err = qualityCheckReplacementUnconsumed(root, c, r); err != nil {
		return nil, "", err
	}
	path := qualityCheckSurfaceReopenPath(root, o.Locale, o.ReopenID)
	if _, err := os.Lstat(path); err == nil || !os.IsNotExist(err) {
		return nil, "", fmt.Errorf("QC representation receipt already exists or cannot be inspected")
	}
	rel, err := repositoryRelativePath(root, path)
	return r, rel, err
}

func recordQualityCheckReplacement(root string, c *Catalog, o QualityCheckSurfaceReopenOptions) (*QualityCheckSurfaceReopen, string, error) {
	r, rel, err := InspectQualityCheckRepresentationReopen(root, c, o)
	if err != nil {
		return nil, "", err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, "", err
	}
	path := qualityCheckSurfaceReopenPath(root, o.Locale, o.ReopenID)
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, "", err
	}
	if err = writeNewGenerationFile(path, append(data, '\n')); err != nil {
		return nil, "", err
	}
	return r, rel, nil
}

func qualityCheckReplacementUnconsumed(root string, c *Catalog, r *QualityCheckSurfaceReopen) error {
	if err := verifyQualityCheckReplacement(root, c, r); err != nil {
		return err
	}
	base := filepath.Join(root, "data", "retranslation-runs", r.Locale)
	entries, err := os.ReadDir(base)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name()[0] == '.' {
			continue
		}
		dir := filepath.Join(base, entry.Name())
		m, err := readRetranslationProcessManifest(dir, r.Locale, entry.Name())
		if err != nil {
			return err
		}
		bound := false
		for _, record := range m.Units {
			for _, u := range r.Units {
				if record.UnitID == u.UnitID || (record.UnitID == u.Snapshot.UnitID && record.PreviousSnapshotID == r.PreviousSnapshotID) {
					bound = true
				}
			}
		}
		if !bound {
			continue
		}
		if err = ensureNoGenerationImportPending(dir); err != nil {
			return err
		}
		for _, name := range []string{"raw-responses", "generation-imports", "result.json"} {
			if _, err = os.Lstat(filepath.Join(dir, name)); err == nil {
				return fmt.Errorf("QC reconciliation already has formal outputs; continue existing lifecycle")
			} else if !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}
