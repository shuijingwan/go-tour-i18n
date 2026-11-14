package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func qualityCheckCorrectionFixture(t *testing.T, count int) (string, *Catalog, *QualityCheckSnapshotManifest, QualityCheckCorrectionOptions) {
	t.Helper()
	root, c, _ := makeRetranslationReviewBatchFixture(t, count, "correction-qc")
	s, err := readQualityCheckSnapshot(root, "zh-CN", "correction-qc", false)
	if err != nil {
		t.Fatal(err)
	}
	id := s.Units[0].UnitID
	if _, err := RecordQualityCheckResults(root, c, QualityCheckRecordOptions{Locale: s.Locale, SnapshotID: s.SnapshotID, ReviewerSession: "independent-correction-reviewer", UnitIDs: []string{id}, Rating: "B", Finding: "Incorrect source literal finding"}); err != nil {
		t.Fatal(err)
	}
	o := correctionEvidenceFor(t, root, s, id)
	return root, c, s, o
}

func correctionEvidenceFor(t *testing.T, root string, s *QualityCheckSnapshotManifest, id string) QualityCheckCorrectionOptions {
	t.Helper()
	r, err := readOriginalQualityCheckResults(root, s.Locale, s)
	if err != nil || r == nil {
		t.Fatal(err)
	}
	for _, j := range r.Results {
		if j.UnitID != id {
			continue
		}
		path, _ := repositoryRelativePath(root, qualityCheckResultsPath(root, s.Locale, s.SnapshotID))
		b, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		no := false
		e := QualityCheckCorrectionEvidence{Locale: s.Locale, SnapshotID: s.SnapshotID, OriginalResults: GlossaryArchiveReference{Path: path, SHA256: sum(b)},
			ReviewerSession: r.ReviewerSession, UnitID: id, Index: j.Index, OriginalRating: j.Rating, OriginalFinding: j.Finding, CorrectedRating: "A",
			FindingDisposition: "withdrawn-source-mismatch", Reason: "Independent Reviewer verified the exact source occurrence; the finding is withdrawn.",
			CorrectionRequired: true, ReplacementTextGenerated: &no, RepositoryMutation: "none"}
		data := mustJSON(e)
		return QualityCheckCorrectionOptions{Locale: s.Locale, SnapshotID: s.SnapshotID, ReviewerSession: e.ReviewerSession, Evidence: data, EvidenceSHA256: sum(data)}
	}
	t.Fatal("fixture judgment missing")
	return QualityCheckCorrectionOptions{}
}

func TestQualityCheckCorrectionImmutableAndEffective(t *testing.T) {
	root, c, s, o := qualityCheckCorrectionFixture(t, 1)
	// A zero-output exported revision is retained, but is not a newer passed
	// candidate and must not block correction of the reviewed predecessor.
	export, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: s.Locale, Generator: RetranslationGeneratorCodex, UnitIDs: []string{s.Units[0].UnitID}, PreviousSnapshotID: s.SnapshotID, AllowReexport: true})
	if err != nil {
		t.Fatal(err)
	}
	zeroManifest := filepath.Join(root, export.BatchPath, "manifest.json")
	zeroBytes, err := os.ReadFile(zeroManifest)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(qualityCheckResultsPath(root, s.Locale, s.SnapshotID))
	// Ordinary readers work without any implicit correction/migration.
	legacy, err := readQualityCheckResults(root, s.Locale, s)
	if err != nil || legacy.Results[0].Rating != "B" {
		t.Fatal(legacy, err)
	}
	got, err := RecordQualityCheckCorrection(root, c, o)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(qualityCheckResultsPath(root, s.Locale, s.SnapshotID))
	if !bytes.Equal(before, after) {
		t.Fatal("original evidence rewritten")
	}
	if b, err := os.ReadFile(zeroManifest); err != nil || !bytes.Equal(b, zeroBytes) {
		t.Fatal("zero-output revision changed", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, got.Path))
	if got.SHA256 != sum(data) {
		t.Fatal("incorrect publication SHA")
	}
	effective, err := readQualityCheckResults(root, s.Locale, s)
	if err != nil || effective.Results[0].Rating != "A" || effective.Results[0].Finding != "" {
		t.Fatal(effective, err)
	}
	if _, err := RecordQualityCheckCorrection(root, c, o); err == nil {
		t.Fatal("duplicate replay accepted")
	}
	var e QualityCheckCorrectionEvidence
	_ = json.Unmarshal(o.Evidence, &e)
	e.Reason = "different correction"
	o.Evidence, o.EvidenceSHA256 = mustJSON(e), sum(mustJSON(e))
	if _, err := RecordQualityCheckCorrection(root, c, o); err == nil {
		t.Fatal("conflicting correction accepted")
	}
	if _, err := RecordQualityCheckResults(root, c, QualityCheckRecordOptions{Locale: s.Locale, SnapshotID: s.SnapshotID, ReviewerSession: o.ReviewerSession, UnitIDs: []string{s.Units[0].UnitID}, Rating: "A"}); err == nil {
		t.Fatal("record may rewrite frozen original")
	}
	if _, err := BackfillQualityCheckFinding(root, c, QualityCheckFindingBackfillOptions{Locale: s.Locale, SnapshotID: s.SnapshotID, UnitID: s.Units[0].UnitID, Finding: "changed"}); err == nil {
		t.Fatal("backfill may rewrite frozen original")
	}
	if b, _ := os.ReadFile(filepath.Join(root, got.Path)); !bytes.Equal(b, data) {
		t.Fatal("no-replace violated")
	}
	// The current finalization and historical proof both bind original + correction.
	f, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: s.Locale, SnapshotID: s.SnapshotID})
	if err != nil || len(f.QCResults) != 2 || f.QCResults[1].Path != got.Path {
		t.Fatal(f, err)
	}
	proof, err := VerifyHistoricalQualityCheckFinalization(root, s.Locale, s.SnapshotID, c)
	if err != nil || proof.Finalization.Units[0].Rating != "A" {
		t.Fatal(proof, err)
	}
	// A finalization cannot silently acquire an unbound correction overlay.
	finalPath := qualityCheckFinalizationPath(root, s.Locale, s.SnapshotID)
	finalBytes, _ := os.ReadFile(finalPath)
	missing := *f
	missing.QCResults = missing.QCResults[:1]
	writeSnapshotSource(t, root, "data/quality-check-snapshots/zh-CN/"+s.SnapshotID+"/finalization.json", mustJSON(missing))
	if _, err := VerifyHistoricalQualityCheckFinalization(root, s.Locale, s.SnapshotID, c); err == nil {
		t.Fatal("unbound historical correction accepted")
	}
	if err := os.WriteFile(finalPath, finalBytes, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, got.Path), append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readQualityCheckResults(root, s.Locale, s); err == nil {
		t.Fatal("artifact byte tamper accepted")
	}
	if _, err := VerifyHistoricalQualityCheckFinalization(root, s.Locale, s.SnapshotID, c); err == nil {
		t.Fatal("historical correction SHA tamper accepted")
	}
}

func TestQualityCheckCorrectionReaderTamperAndLegacy(t *testing.T) {
	root, c, s, o := qualityCheckCorrectionFixture(t, 1)
	path := qualityCheckResultsPath(root, s.Locale, s.SnapshotID)
	prior, _ := os.ReadFile(path)
	var legacy QualityCheckResults
	_ = json.Unmarshal(prior, &legacy)
	legacy.SchemaVersion = legacyQualityCheckResultsSchemaVersion
	legacy.ReviewerSession = ""
	if err := os.WriteFile(path, mustJSON(legacy), 0644); err != nil {
		t.Fatal(err)
	}
	if got, err := readQualityCheckResults(root, s.Locale, s); err != nil || got.SchemaVersion != 1 || got.Results[0].Rating != "B" {
		t.Fatal(got, err)
	}
	if _, err := RecordQualityCheckCorrection(root, c, o); err == nil {
		t.Fatal("legacy evidence acquired Reviewer authority")
	}
	if _, err := os.Stat(filepath.Join(root, qualityCheckCorrectionDir(s.Locale, s.SnapshotID))); !os.IsNotExist(err) {
		t.Fatal("legacy read migrated evidence")
	}
	if err := os.WriteFile(path, prior, 0644); err != nil {
		t.Fatal(err)
	}
	got, err := RecordQualityCheckCorrection(root, c, o)
	if err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(root, got.Path)
	data, _ := os.ReadFile(artifact)
	for _, mode := range []string{"original-results", "rebound-unit", "unknown-field", "duplicate-artifact", "missing-original"} {
		t.Run(mode, func(t *testing.T) {
			defer os.WriteFile(path, prior, 0644)
			defer os.WriteFile(artifact, data, 0644)
			switch mode {
			case "original-results":
				if err := os.WriteFile(path, append(bytes.Clone(prior), '\n'), 0644); err != nil {
					t.Fatal(err)
				}
			case "rebound-unit":
				var bad QualityCheckCorrection
				_ = json.Unmarshal(data, &bad)
				bad.Unit.CandidateSHA256 = sum([]byte("wrong candidate"))
				bad.IdentitySHA256 = qualityCheckCorrectionIdentity(bad)
				if err := os.WriteFile(artifact, append(mustJSON(bad), '\n'), 0644); err != nil {
					t.Fatal(err)
				}
			case "unknown-field":
				if err := os.WriteFile(artifact, []byte(strings.TrimSpace(string(data[:len(data)-2]))+`,"unknown":true}`), 0644); err != nil {
					t.Fatal(err)
				}
			case "duplicate-artifact":
				duplicate := filepath.Join(filepath.Dir(artifact), "duplicate.json")
				if err := os.WriteFile(duplicate, data, 0644); err != nil {
					t.Fatal(err)
				}
				defer os.Remove(duplicate)
			case "missing-original":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := readQualityCheckResults(root, s.Locale, s); err == nil {
				t.Fatal("invalid correction transport accepted", mode)
			}
		})
	}
}

func TestQualityCheckCorrectionScopeCounts(t *testing.T) {
	root, c, _ := makeRetranslationReviewBatchFixture(t, 51, "correction-prev")
	previous, _ := readQualityCheckSnapshot(root, "zh-CN", "correction-prev", false)
	carry := []string{}
	for _, u := range previous.Units[23:] {
		carry = append(carry, u.UnitID)
	}
	if _, err := RecordQualityCheckResults(root, c, QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: previous.SnapshotID, ReviewerSession: "independent-correction-reviewer", UnitIDs: carry, Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	s, _, err := CreateQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "correction-current"})
	if err != nil {
		t.Fatal(err)
	}
	judgments := []QualityCheckJudgment{}
	for i, u := range s.Units[:23] {
		rating, finding := "A", ""
		if i == 6 || (i >= 10 && i < 17) {
			rating, finding = "B", "Exact source-mismatch finding"
		} else if i >= 17 {
			rating, finding = "C", "Exact C finding"
		}
		judgments = append(judgments, QualityCheckJudgment{UnitID: u.UnitID, Rating: rating, Finding: finding})
	}
	if _, err := RecordQualityCheckJudgments(root, c, QualityCheckRecordJudgmentsOptions{Locale: s.Locale, SnapshotID: s.SnapshotID, PreviousSnapshotID: previous.SnapshotID, ReviewerSession: "independent-correction-reviewer", Judgments: judgments}); err != nil {
		t.Fatal(err)
	}
	before, err := BuildQualityCheckScope(root, c, QualityCheckScopeOptions{Locale: s.Locale, SnapshotID: s.SnapshotID})
	if err != nil || before.ACount != 37 || before.BCount != 8 || before.CCount != 6 || before.PendingCount != 14 || before.CarryForwardCount != 28 {
		t.Fatal(before, err)
	}
	original, _ := readOriginalQualityCheckResults(root, s.Locale, s)
	o := correctionEvidenceFor(t, root, s, s.Units[6].UnitID)
	if _, err := RecordQualityCheckCorrection(root, c, o); err != nil {
		t.Fatal(err)
	}
	after, err := BuildQualityCheckScope(root, c, QualityCheckScopeOptions{Locale: s.Locale, SnapshotID: s.SnapshotID})
	if err != nil || after.ACount != 38 || after.BCount != 7 || after.CCount != 6 || after.DCount != 0 || after.PendingCount != 13 || after.CurrentResultCount != 23 || !reflect.DeepEqual(before.CarryForward, after.CarryForward) {
		t.Fatal(after, err)
	}
	effective, _ := readQualityCheckResults(root, s.Locale, s)
	unchanged := 0
	for i, j := range effective.Results {
		if j.UnitID != s.Units[6].UnitID {
			if !reflect.DeepEqual(j, original.Results[i]) {
				t.Fatal("unrelated judgment changed")
			}
			unchanged++
		}
	}
	if unchanged != 22 {
		t.Fatal("wrong unrelated result count", unchanged)
	}
}

func TestQualityCheckCorrectionRejectsWrongEvidence(t *testing.T) {
	for _, mode := range []string{"sha", "path", "rating", "finding", "Unit", "index", "Snapshot", "locale", "reviewer", "generation-role", "disposition", "reason", "replacement", "missing-replacement", "mutation", "required", "corrected-rating", "evidence-sha", "source", "candidate", "validation", "manifest", "missing-results", "finalized", "unknown-json", "duplicate-json", "trailing-json"} {
		t.Run(mode, func(t *testing.T) {
			root, c, s, o := qualityCheckCorrectionFixture(t, 1)
			var e QualityCheckCorrectionEvidence
			_ = json.Unmarshal(o.Evidence, &e)
			switch mode {
			case "sha":
				e.OriginalResults.SHA256 = sum([]byte("wrong"))
			case "path":
				e.OriginalResults.Path = "data/wrong.json"
			case "rating":
				e.OriginalRating = "C"
			case "finding":
				e.OriginalFinding = "wrong finding"
			case "Unit":
				e.UnitID = "another Unit"
			case "index":
				e.Index++
			case "Snapshot":
				e.SnapshotID = "another-snapshot"
			case "locale":
				e.Locale = "en-GB"
			case "reviewer":
				e.ReviewerSession = "self-declared-reviewer"
				o.ReviewerSession = e.ReviewerSession
			case "generation-role":
				entries, err := os.ReadDir(filepath.Join(root, "data/unified-glossary-reviews", s.Locale))
				if err != nil || len(entries) == 0 {
					t.Fatal(err)
				}
				b, err := os.ReadFile(filepath.Join(root, "data/unified-glossary-reviews", s.Locale, entries[0].Name()))
				if err != nil {
					t.Fatal(err)
				}
				var role UnifiedGlossaryReview
				if err := json.Unmarshal(b, &role); err != nil || role.GenerationSession == "" {
					t.Fatal(err)
				}
				original, err := readOriginalQualityCheckResults(root, s.Locale, s)
				if err != nil {
					t.Fatal(err)
				}
				original.ReviewerSession = role.GenerationSession
				if err := writeQualityCheckResults(qualityCheckResultsPath(root, s.Locale, s.SnapshotID), original); err != nil {
					t.Fatal(err)
				}
				e.ReviewerSession, o.ReviewerSession = role.GenerationSession, role.GenerationSession
				b, _ = os.ReadFile(qualityCheckResultsPath(root, s.Locale, s.SnapshotID))
				e.OriginalResults.SHA256 = sum(b)
			case "disposition":
				e.FindingDisposition = "ignore"
			case "reason":
				e.Reason = ""
			case "replacement":
				*e.ReplacementTextGenerated = true
			case "missing-replacement":
				e.ReplacementTextGenerated = nil
			case "mutation":
				e.RepositoryMutation = "known"
			case "required":
				e.CorrectionRequired = false
			case "corrected-rating":
				e.CorrectedRating = "B"
			case "source":
				writeSnapshotSource(t, root, s.Units[0].SourcePath, []byte("changed source"))
			case "candidate":
				writeSnapshotSource(t, root, s.Units[0].CandidatePath, []byte("changed candidate"))
			case "validation":
				writeSnapshotSource(t, root, s.Units[0].ValidationPath, []byte("{}"))
			case "manifest":
				writeSnapshotSource(t, root, "data/quality-check-snapshots/zh-CN/"+s.SnapshotID+"/manifest.json", []byte("{}"))
			case "missing-results":
				if err := os.Remove(qualityCheckResultsPath(root, s.Locale, s.SnapshotID)); err != nil {
					t.Fatal(err)
				}
			case "finalized":
				writeSnapshotSource(t, root, "data/quality-check-snapshots/zh-CN/"+s.SnapshotID+"/finalization.json", []byte("{}"))
			}
			o.Evidence, o.EvidenceSHA256 = mustJSON(e), sum(mustJSON(e))
			switch mode {
			case "unknown-json":
				o.Evidence = []byte(strings.TrimSuffix(string(o.Evidence), "}") + `,"unknown":true}`)
			case "duplicate-json":
				o.Evidence = []byte(strings.TrimSuffix(string(o.Evidence), "}") + `,"unit_id":"duplicate"}`)
			case "trailing-json":
				o.Evidence = append(o.Evidence, []byte(" null")...)
			}
			o.EvidenceSHA256 = sum(o.Evidence)
			if mode == "evidence-sha" {
				o.EvidenceSHA256 = sum([]byte("wrong evidence"))
			}
			if _, err := RecordQualityCheckCorrection(root, c, o); err == nil {
				t.Fatal("invalid correction accepted", mode)
			}
			if entries, _ := os.ReadDir(filepath.Join(root, qualityCheckCorrectionDir(s.Locale, s.SnapshotID))); len(entries) != 0 {
				t.Fatal("failed correction published evidence")
			}
		})
	}
}
