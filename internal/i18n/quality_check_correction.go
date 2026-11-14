package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

const qualityCheckCorrectionSchema = "go-learning/quality-check-judgment-correction/v1"

// QualityCheckCorrectionEvidence is an explicit independent Reviewer response,
// not a machine-generated judgment. Only withdrawal of a source-mismatch
// finding to A is supported; translation/revision recovery is out of scope.
type QualityCheckCorrectionEvidence struct {
	Locale                   string                   `json:"locale"`
	SnapshotID               string                   `json:"snapshot_id"`
	OriginalResults          GlossaryArchiveReference `json:"original_results"`
	ReviewerSession          string                   `json:"reviewer_session"`
	UnitID                   string                   `json:"unit_id"`
	Index                    int                      `json:"index"`
	OriginalRating           string                   `json:"original_rating"`
	OriginalFinding          string                   `json:"original_finding"`
	CorrectedRating          string                   `json:"corrected_rating"`
	FindingDisposition       string                   `json:"finding_disposition"`
	Reason                   string                   `json:"reason"`
	CorrectionRequired       bool                     `json:"correction_required"`
	ReplacementTextGenerated *bool                    `json:"replacement_text_generated"`
	RepositoryMutation       string                   `json:"repository_mutation"`
}

// The exact response bytes are embedded, so an external /tmp input need not
// survive. Original results and Snapshot remain separate immutable evidence.
type QualityCheckCorrection struct {
	Schema                 string                   `json:"schema"`
	Locale                 string                   `json:"locale"`
	SnapshotID             string                   `json:"snapshot_id"`
	SnapshotManifestSHA256 string                   `json:"snapshot_manifest_sha256"`
	OriginalResults        GlossaryArchiveReference `json:"original_results"`
	Unit                   QualityCheckSnapshotUnit `json:"unit"`
	Original               QualityCheckResult       `json:"original"`
	Corrected              QualityCheckResult       `json:"corrected"`
	ReviewerSession        string                   `json:"reviewer_session"`
	FindingDisposition     string                   `json:"finding_disposition"`
	Reason                 string                   `json:"reason"`
	ReviewerEvidence       []byte                   `json:"reviewer_evidence"`
	ReviewerEvidenceSHA256 string                   `json:"reviewer_evidence_sha256"`
	IdentitySHA256         string                   `json:"identity_sha256"`
}

type QualityCheckCorrectionOptions struct {
	Locale, SnapshotID, ReviewerSession, EvidenceSHA256 string
	Evidence                                            []byte
}

type QualityCheckCorrectionRecordResult struct {
	Locale     string `json:"locale"`
	SnapshotID string `json:"snapshot_id"`
	UnitID     string `json:"unit_id"`
	Index      int    `json:"index"`
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
}

func qualityCheckCorrectionDir(locale, snapshot string) string {
	return "data/quality-check-snapshots/" + locale + "/" + snapshot + "/judgment-corrections"
}

func qualityCheckCorrectionPath(locale, snapshot, unit string) string {
	return qualityCheckCorrectionDir(locale, snapshot) + "/" + sum([]byte(unit)) + ".json"
}

func qualityCheckCorrectionIdentity(c QualityCheckCorrection) string {
	c.IdentitySHA256 = ""
	b, _ := json.Marshal(c)
	return sum(b)
}

func prepareQualityCheckCorrection(root string, snapshot *QualityCheckSnapshotManifest, original *QualityCheckResults, evidence []byte) (*QualityCheckCorrection, error) {
	var e QualityCheckCorrectionEvidence
	if err := contentidentity.StrictJSON(evidence, &e); err != nil {
		return nil, fmt.Errorf("Reviewer correction evidence: %w", err)
	}
	if original == nil || original.Rubric != TranslationQualityRubric || e.Locale != snapshot.Locale || e.SnapshotID != snapshot.SnapshotID ||
		e.Index < 1 || e.Index > len(snapshot.Units) || e.UnitID != snapshot.Units[e.Index-1].UnitID ||
		e.ReviewerSession == "" || e.ReviewerSession != original.ReviewerSession {
		return nil, fmt.Errorf("correction original Snapshot/Unit/Reviewer identity mismatch")
	}
	if e.CorrectedRating != "A" || (e.OriginalRating != "B" && e.OriginalRating != "C" && e.OriginalRating != "D") ||
		e.FindingDisposition != "withdrawn-source-mismatch" || strings.TrimSpace(e.Reason) == "" || !e.CorrectionRequired ||
		e.ReplacementTextGenerated == nil || *e.ReplacementTextGenerated || e.RepositoryMutation != "none" {
		return nil, fmt.Errorf("correction requires explicit non-A to A source-mismatch withdrawal without replacement or mutation")
	}
	if err := CheckUnifiedLanguageRole(root, e.Locale, e.ReviewerSession, "reviewer"); err != nil {
		return nil, err
	}
	path := filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", e.Locale, e.SnapshotID, "quality-check-results.json"))
	b, err := readCompatibilityFile(root, path)
	if err != nil || e.OriginalResults.Path != path || !validSHA256(e.OriginalResults.SHA256) || sum(b) != e.OriginalResults.SHA256 {
		return nil, fmt.Errorf("correction original results path/SHA changed")
	}
	for _, r := range original.Results {
		if r.UnitID != e.UnitID {
			continue
		}
		if r.Index != e.Index || r.Rating != e.OriginalRating || r.Finding != e.OriginalFinding || strings.TrimSpace(r.Finding) == "" {
			return nil, fmt.Errorf("correction original rating/finding/index mismatch")
		}
		c := &QualityCheckCorrection{Schema: qualityCheckCorrectionSchema, Locale: e.Locale, SnapshotID: e.SnapshotID,
			SnapshotManifestSHA256: original.SnapshotManifestSHA256, OriginalResults: e.OriginalResults,
			Unit: snapshot.Units[e.Index-1], Original: r, Corrected: QualityCheckResult{Index: r.Index, UnitID: r.UnitID, Rating: "A"},
			ReviewerSession: e.ReviewerSession, FindingDisposition: e.FindingDisposition, Reason: e.Reason,
			ReviewerEvidence: append([]byte(nil), evidence...), ReviewerEvidenceSHA256: sum(evidence)}
		c.IdentitySHA256 = qualityCheckCorrectionIdentity(*c)
		return c, nil
	}
	return nil, fmt.Errorf("correction requires an original recorded judgment for the exact Unit")
}

// Apply only after reconstructing every exact original binding. The reader
// never manufactures Reviewer identity, changes the original, or migrates it.
func readQualityCheckCorrections(root string, snapshot *QualityCheckSnapshotManifest, original *QualityCheckResults) ([]QualityCheckCorrection, []GlossaryArchiveReference, error) {
	dir := qualityCheckCorrectionDir(snapshot.Locale, snapshot.SnapshotID)
	// Validate the directory as a parent; compatibilityPath's final component
	// deliberately requires a regular file. This probe is never created.
	probe, err := compatibilityPath(root, dir+"/.read", false)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	entries, err := os.ReadDir(filepath.Dir(probe))
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var corrections []QualityCheckCorrection
	var refs []GlossaryArchiveReference
	seen := map[string]bool{}
	for _, entry := range entries {
		rel := dir + "/" + entry.Name()
		b, err := readCompatibilityFile(root, rel)
		if err != nil {
			return nil, nil, err
		}
		var c QualityCheckCorrection
		if err := contentidentity.StrictJSON(b, &c); err != nil {
			return nil, nil, err
		}
		if !bytes.Equal(b, append(mustJSON(c), '\n')) || c.Schema != qualityCheckCorrectionSchema || !validSHA256(c.IdentitySHA256) || c.IdentitySHA256 != qualityCheckCorrectionIdentity(c) ||
			sum(c.ReviewerEvidence) != c.ReviewerEvidenceSHA256 || rel != qualityCheckCorrectionPath(snapshot.Locale, snapshot.SnapshotID, c.Unit.UnitID) || seen[c.Unit.UnitID] {
			return nil, nil, fmt.Errorf("invalid/duplicate Quality Check correction: %s", rel)
		}
		expected, err := prepareQualityCheckCorrection(root, snapshot, original, c.ReviewerEvidence)
		if err != nil {
			return nil, nil, err
		}
		if !reflect.DeepEqual(c, *expected) {
			return nil, nil, fmt.Errorf("Quality Check correction binding changed: %s", rel)
		}
		seen[c.Unit.UnitID] = true
		corrections = append(corrections, c)
		refs = append(refs, GlossaryArchiveReference{Path: rel, SHA256: sum(b)})
	}
	return corrections, refs, nil
}

func readQualityCheckResults(root, locale string, snapshot *QualityCheckSnapshotManifest) (*QualityCheckResults, error) {
	original, err := readOriginalQualityCheckResults(root, locale, snapshot)
	if err != nil {
		return nil, err
	}
	corrections, _, err := readQualityCheckCorrections(root, snapshot, original)
	if err != nil {
		return nil, err
	}
	if len(corrections) == 0 {
		return original, nil
	}
	effective := *original
	effective.Results = append([]QualityCheckResult(nil), original.Results...)
	for _, correction := range corrections {
		for i, r := range effective.Results {
			if r.UnitID == correction.Unit.UnitID {
				effective.Results[i] = correction.Corrected
			}
		}
	}
	return &effective, nil
}

func requireUncorrectedQualityCheckResults(root string, snapshot *QualityCheckSnapshotManifest, results *QualityCheckResults) error {
	_, refs, err := readQualityCheckCorrections(root, snapshot, results)
	if err != nil {
		return err
	}
	if len(refs) != 0 {
		return fmt.Errorf("original Quality Check results are frozen by judgment correction")
	}
	return nil
}

func RecordQualityCheckCorrection(root string, catalog *Catalog, o QualityCheckCorrectionOptions) (*QualityCheckCorrectionRecordResult, error) {
	if catalog == nil {
		return nil, fmt.Errorf("quality-check catalog is required")
	}
	if err := ValidateLocaleName(o.Locale); err != nil {
		return nil, err
	}
	if err := validateSnapshotID(o.SnapshotID); err != nil {
		return nil, err
	}
	if o.ReviewerSession == "" || !validSHA256(o.EvidenceSHA256) || sum(o.Evidence) != o.EvidenceSHA256 {
		return nil, fmt.Errorf("explicit Reviewer session and exact correction evidence SHA required")
	}
	s, err := readQualityCheckSnapshotForReview(root, o.Locale, o.SnapshotID, catalog)
	if err != nil {
		return nil, err
	}
	if _, err := os.Lstat(qualityCheckFinalizationPath(root, o.Locale, o.SnapshotID)); !os.IsNotExist(err) {
		return nil, fmt.Errorf("correction requires an unfinalized Snapshot: %v", err)
	}
	original, err := readOriginalQualityCheckResults(root, o.Locale, s)
	if err != nil {
		return nil, err
	}
	c, err := prepareQualityCheckCorrection(root, s, original, o.Evidence)
	if err != nil {
		return nil, err
	}
	if c.ReviewerSession != o.ReviewerSession {
		return nil, fmt.Errorf("explicit correction Reviewer session mismatch")
	}
	latest, err := selectLatestRetranslationUnits(root, catalog, o.Locale)
	if err != nil {
		return nil, err
	}
	if choice, ok := latest.selectedByID[c.Unit.UnitID]; !ok || !choiceMatchesSnapshot(o.Locale, choice, c.Unit) {
		return nil, fmt.Errorf("correction selected candidate identity is stale")
	}
	u, err := catalog.Unit(c.Unit.UnitID)
	if err != nil {
		return nil, err
	}
	source, err := readSnapshotReferencedFile(root, u.SourcePath)
	if err != nil {
		return nil, err
	}
	if err := validateSnapshotSourceIdentity(catalog, u, source); err != nil {
		return nil, err
	}
	_, refs, err := readQualityCheckCorrections(root, s, original)
	if err != nil {
		return nil, err
	}
	rel := qualityCheckCorrectionPath(o.Locale, o.SnapshotID, c.Unit.UnitID)
	for _, ref := range refs {
		if ref.Path == rel {
			return nil, fmt.Errorf("Quality Check correction already exists for Unit %s", c.Unit.UnitID)
		}
	}
	if c.Unit.RepresentationReconciliation != nil {
		_, _, err = machineSnapshotEvidence(root, catalog, o.Locale, c.Unit, map[string]*machineSelection{})
	} else {
		_, err = readSnapshotUnitRepositoryEvidence(root, catalog, o.Locale, c.Unit)
	}
	if err != nil {
		return nil, err
	}
	data := append(mustJSON(c), '\n')
	path, err := compatibilityPath(root, rel, true)
	if err != nil {
		return nil, err
	}
	// Bind the publication to the exact prior bytes again after preparation.
	manifest, err := os.ReadFile(qualityCheckSnapshotManifestPath(root, o.Locale, o.SnapshotID))
	if err != nil || sum(manifest) != c.SnapshotManifestSHA256 {
		return nil, fmt.Errorf("correction Snapshot changed before publication")
	}
	prior, err := readCompatibilityFile(root, c.OriginalResults.Path)
	if err != nil || sum(prior) != c.OriginalResults.SHA256 {
		return nil, fmt.Errorf("correction original results changed before publication")
	}
	// No-replace publication. Maintainers serialize formal recording; this is
	// not a multi-file transaction or a lock against external repository edits.
	if err := writeNewGenerationFile(path, data); err != nil {
		return nil, err
	}
	return &QualityCheckCorrectionRecordResult{Locale: o.Locale, SnapshotID: o.SnapshotID, UnitID: c.Unit.UnitID, Index: c.Unit.Index, Path: rel, SHA256: sum(data)}, nil
}
