package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestQualityCheckHeterogeneousJudgments(t *testing.T) {
	root, c, _ := makeRetranslationReviewBatchFixture(t, 4, "bulk-qc")
	judgments := []QualityCheckJudgment{{"lesson/4", "D", "specific D finding"}, {"lesson/2", "B", "specific B\nsecond line"}, {"lesson/1", "A", ""}, {"lesson/3", "C", "specific C finding"}}
	options := QualityCheckRecordJudgmentsOptions{Locale: "zh-CN", SnapshotID: "bulk-qc", ReviewerSession: "independent-bulk-reviewer", Judgments: judgments}
	stats := &qualityCheckRecordingStats{}
	result, err := recordQualityCheckJudgmentsBulk(root, c, options, stats)
	if err != nil {
		t.Fatal(err)
	}
	if result.RecordedCount != 4 || result.ResultCount != 4 || result.Rating != "" {
		t.Fatal(result)
	}
	if stats.SnapshotReads != 1 || stats.ContextBuilds != 1 || stats.RepositoryEvidenceReads != 4 || stats.WriteAttempts != 1 {
		t.Fatal("record rebuilt shared context or published more than once", stats)
	}
	path := qualityCheckResultsPath(root, options.Locale, options.SnapshotID)
	bulk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var recorded QualityCheckResults
	if err := json.Unmarshal(bulk, &recorded); err != nil {
		t.Fatal(err)
	}
	expected := map[string]QualityCheckJudgment{}
	for _, j := range judgments {
		expected[j.UnitID] = j
	}
	for i, r := range recorded.Results {
		if r.Index != i+1 || r.Rating != expected[r.UnitID].Rating || r.Finding != expected[r.UnitID].Finding {
			t.Fatal("finding/index/order changed", r)
		}
	}
	// Reuse exactly the same immutable Snapshot to compare final bytes with the
	// old single-rating API sequence, not a differently identified fixture.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	for _, j := range judgments {
		if _, err := RecordQualityCheckResults(root, c, QualityCheckRecordOptions{Locale: options.Locale, SnapshotID: options.SnapshotID, ReviewerSession: options.ReviewerSession, UnitIDs: []string{j.UnitID}, Rating: j.Rating, Finding: j.Finding}); err != nil {
			t.Fatal(err)
		}
	}
	sequential, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(bulk, sequential) {
		t.Fatal("heterogeneous results differ from legacy sequence", err)
	}
}

func TestQualityCheckHeterogeneousJudgmentsAtomicAppend(t *testing.T) {
	root, c, _ := makeRetranslationReviewBatchFixture(t, 4, "append-qc")
	base := QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: "append-qc", ReviewerSession: "independent-bulk-reviewer", UnitIDs: []string{"lesson/1"}, Rating: "A"}
	if _, err := RecordQualityCheckResults(root, c, base); err != nil {
		t.Fatal(err)
	}
	resultPath := qualityCheckResultsPath(root, base.Locale, base.SnapshotID)
	before, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	var s QualityCheckSnapshotManifest
	mb, err := os.ReadFile(qualityCheckSnapshotManifestPath(root, base.Locale, base.SnapshotID))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(mb, &s); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"missing-finding", "invalid-rating", "duplicate", "empty-id", "unknown-id", "already-recorded-last", "missing-session", "blank-session", "wrong-role", "changed-session", "wrong-predecessor", "full-rereview", "predecessor-full-conflict", "candidate-drift", "validation-drift", "source-drift", "snapshot-tamper"} {
		t.Run(mode, func(t *testing.T) {
			o := QualityCheckRecordJudgmentsOptions{Locale: base.Locale, SnapshotID: base.SnapshotID, ReviewerSession: base.ReviewerSession, Judgments: []QualityCheckJudgment{{"lesson/2", "B", "second"}, {"lesson/3", "C", "third"}}}
			localCatalog := c
			var restore func()
			switch mode {
			case "missing-finding":
				o.Judgments[1].Finding = " \n "
			case "invalid-rating":
				o.Judgments[1].Rating = "PASS"
			case "duplicate":
				o.Judgments[1].UnitID = o.Judgments[0].UnitID
			case "empty-id":
				o.Judgments[1].UnitID = ""
			case "unknown-id":
				o.Judgments[1].UnitID = "not-in-snapshot"
			case "already-recorded-last":
				o.Judgments[1].UnitID = "lesson/1"
			case "missing-session":
				o.ReviewerSession = ""
			case "blank-session":
				o.ReviewerSession = "  "
			case "wrong-role":
				entries, err := os.ReadDir(filepath.Join(root, "data/unified-glossary-reviews", base.Locale))
				if err != nil || len(entries) == 0 {
					t.Fatal(err)
				}
				b, err := os.ReadFile(filepath.Join(root, "data/unified-glossary-reviews", base.Locale, entries[0].Name()))
				if err != nil {
					t.Fatal(err)
				}
				var review UnifiedGlossaryReview
				if err := json.Unmarshal(b, &review); err != nil {
					t.Fatal(err)
				}
				o.ReviewerSession = review.GenerationSession
				if o.ReviewerSession == "" {
					t.Fatal("fixture lacks generation session")
				}
			case "changed-session":
				o.ReviewerSession = "different-independent-reviewer"
			case "wrong-predecessor":
				o.PreviousSnapshotID = "wrong-previous"
			case "full-rereview":
				o.FullRereview = true
			case "predecessor-full-conflict":
				o.FullRereview = true
				o.PreviousSnapshotID = "wrong-previous"
			case "candidate-drift", "validation-drift", "source-drift", "snapshot-tamper":
				path := s.Units[2].CandidatePath
				if mode == "validation-drift" {
					path = s.Units[2].ValidationPath
				}
				if mode == "source-drift" {
					path = s.Units[2].SourcePath
					// Match the real CLI boundary: rebuild current catalog identity after a
					// semantic source change, rather than append outside a native section.
					copyCatalog := *c
					copyCatalog.Pages = append([]Page{}, c.Pages...)
					copyCatalog.Pages[2].Source = append(bytes.Clone(c.Pages[2].Source), []byte("changed\n")...)
					copyCatalog.Pages[2].SourceSHA256 = sum(copyCatalog.Pages[2].Source)
					localCatalog = &copyCatalog
				}
				if mode == "snapshot-tamper" {
					path = filepath.ToSlash(filepath.Join("data/quality-check-snapshots", base.Locale, base.SnapshotID, "manifest.json"))
				}
				p := filepath.Join(root, path)
				b, err := os.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, append(bytes.Clone(b), []byte("changed")...), 0600); err != nil {
					t.Fatal(err)
				}
				restore = func() {
					if err := os.WriteFile(p, b, 0600); err != nil {
						t.Fatal(err)
					}
				}

			}
			if restore != nil {
				defer restore()
			}
			stats := &qualityCheckRecordingStats{}
			if _, err := recordQualityCheckJudgmentsBulk(root, localCatalog, o, stats); err == nil {
				t.Fatal("invalid set accepted", mode)
			}
			after, err := os.ReadFile(resultPath)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("failed request partially mutated results", mode, err)
			}
			if stats.WriteAttempts != 0 {
				t.Fatal("validation failure reached results writer", stats)
			}
		})
	}
	o := QualityCheckRecordJudgmentsOptions{Locale: base.Locale, SnapshotID: base.SnapshotID, ReviewerSession: base.ReviewerSession, Judgments: []QualityCheckJudgment{{"lesson/4", "D", "fourth"}, {"lesson/2", "B", "second"}, {"lesson/3", "C", "third"}}}
	if r, err := RecordQualityCheckJudgments(root, c, o); err != nil || r.ResultCount != 4 || r.RecordedCount != 3 {
		t.Fatal(r, err)
	}
}

func TestQualityCheckHeterogeneousJudgmentsPackageAndLineage(t *testing.T) {
	root, source, ids := futureContentFixture(t)
	d := source.TranslationPackages[0]
	d.AuthoritySHA256 = sum([]byte("bulk package authority"))
	d.Units = append([]AdaptedTranslationUnit{}, d.Units...)
	// This synthetic registered package declares one legal initial working set.
	for i := range d.Units {
		d.Units[i].WorkingSet = "bulk-fixture"
		d.Units[i].WorkingSetLimit = DefaultQualityCheckBatchLimit
	}
	c := &Catalog{}
	if err := c.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	checkpointGenerate(t, root, c, ids)
	const snapshot = "registered-bulk-qc"
	s, _, err := CreateQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: snapshot, ContentPackage: c.TranslationPackages[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	o := QualityCheckRecordJudgmentsOptions{Locale: "zh-CN", SnapshotID: snapshot, ReviewerSession: "registered-independent-reviewer"}
	for i, u := range s.Units {
		rating, finding := "A", ""
		if i > 0 {
			rating, finding = "B", fmt.Sprintf("finding %d", i)
		}
		o.Judgments = append(o.Judgments, QualityCheckJudgment{u.UnitID, rating, finding})
	}
	wrong := *c
	wrong.TranslationScope = append([]string{}, c.TranslationScope...)
	wrong.TranslationPackages = append([]TranslationPackageDescriptor{}, c.TranslationPackages...)
	wrong.TranslationPackages[0].AuthoritySHA256 = sum([]byte("wrong package authority"))
	if _, err := RecordQualityCheckJudgments(root, &wrong, o); err == nil {
		t.Fatal("wrong selected package accepted")
	}
	path := qualityCheckResultsPath(root, o.Locale, snapshot)
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatal("failed first write published results", err)
	}
	if _, err := RecordQualityCheckJudgments(root, c, o); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: o.Locale, SnapshotID: "registered-next", ContentPackage: c.TranslationPackages[0].ID}); err != nil {
		t.Fatal(err)
	}
	o.SnapshotID = "registered-next"
	o.PreviousSnapshotID = snapshot
	o.Judgments = o.Judgments[1:]
	if _, err := RecordQualityCheckJudgments(root, c, o); err != nil {
		t.Fatal(err)
	}
	// A later heterogeneous append must confirm, never retrofit, predecessor.
	o.Judgments = []QualityCheckJudgment{{s.Units[0].UnitID, "A", ""}}
	o.PreviousSnapshotID = "different-previous"
	if _, err := RecordQualityCheckJudgments(root, c, o); err == nil || !strings.Contains(err.Error(), "persisted previous_snapshot_id") {
		t.Fatal("persisted lineage changed", err)
	}
}

func TestQualityCheckHeterogeneousInitialPartitionAndLegacyBatch(t *testing.T) {
	t.Run("initial-partition-preserved", func(t *testing.T) {
		root, c, ids := futureContentFixture(t)
		checkpointGenerate(t, root, c, ids)
		if _, _, err := CreateQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "partition-qc"}); err != nil {
			t.Fatal(err)
		}
		o := QualityCheckRecordJudgmentsOptions{Locale: "zh-CN", SnapshotID: "partition-qc", ReviewerSession: "independent-reviewer", Judgments: []QualityCheckJudgment{{ids[0], "A", ""}, {ids[1], "B", "finding"}}}
		if _, err := RecordQualityCheckJudgments(root, c, o); err == nil || !strings.Contains(err.Error(), "固定页面批次") {
			t.Fatal("initial partition gate weakened", err)
		}
		if _, err := os.Stat(qualityCheckResultsPath(root, o.Locale, o.SnapshotID)); !os.IsNotExist(err) {
			t.Fatal("invalid initial working set published results", err)
		}

	})
	t.Run("legacy-record-batch", func(t *testing.T) {
		root, c, _ := makeRetranslationReviewBatchFixture(t, 4, "legacy-batch-qc")
		o := QualityCheckRecordBatchOptions{Locale: "zh-CN", SnapshotID: "legacy-batch-qc", ReviewerSession: "independent-reviewer", StartIndex: 2, Limit: 61, Rating: "B", Finding: "shared finding"}
		if _, err := RecordQualityCheckResultBatch(root, c, o); err == nil {
			t.Fatal("legacy batch limit weakened")
		}
		o.Limit = 2
		r, err := RecordQualityCheckResultBatch(root, c, o)
		if err != nil || r.RecordedCount != 2 || r.Rating != "B" || len(r.UnitIDs) != 2 || r.UnitIDs[0] != "lesson/2" || r.UnitIDs[1] != "lesson/3" {
			t.Fatal("legacy range/rating semantics changed", r, err)
		}
		b, err := os.ReadFile(qualityCheckResultsPath(root, o.Locale, o.SnapshotID))
		if err != nil {
			t.Fatal(err)
		}
		var results QualityCheckResults
		if err := json.Unmarshal(b, &results); err != nil {
			t.Fatal(err)
		}
		for _, j := range results.Results {
			if j.Finding != o.Finding || j.Rating != o.Rating {
				t.Fatal(j)
			}
		}
	})
}

// All 61 requested Units are real current members with independently checked
// repository evidence. An existing result belongs to a different 62nd Unit.
func TestQualityCheckHeterogeneousValidCountBoundary(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, source, ids := futureContentFixtureSize(t, 62)
	d := source.TranslationPackages[0]
	d.AuthoritySHA256 = sum([]byte("bulk-count-boundary-package"))
	d.Units = append([]AdaptedTranslationUnit{}, d.Units...)
	for i := range d.Units {
		d.Units[i].WorkingSet = ""
		d.Units[i].WorkingSetLimit = 0
	}
	c := &Catalog{}
	if err := c.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	checkpointGenerate(t, root, c, ids)
	const snapshot = "valid-count-boundary"
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: snapshot, ContentPackage: d.ID}); err != nil {
		t.Fatal(err)
	}
	s, err := readQualityCheckSnapshotForReview(root, "zh-CN", snapshot, c)
	if err != nil || len(s.Units) != 62 {
		t.Fatal("invalid full current Snapshot", err)
	}
	for _, u := range s.Units {
		member, err := c.Unit(u.UnitID)
		if err != nil || member.Kind != UnitKindPage || u.UnitKind != UnitKindPage {
			t.Fatal("invalid count fixture member", u, err)
		}
		if _, err := readSnapshotUnitRepositoryEvidence(root, c, "zh-CN", u); err != nil {
			t.Fatal("count fixture repository identity", u.UnitID, err)
		}
	}
	const reviewer = "independent-count-reviewer"
	if err := CheckUnifiedLanguageRole(root, "zh-CN", reviewer, "reviewer"); err != nil {
		t.Fatal(err)
	}
	o := QualityCheckRecordJudgmentsOptions{Locale: "zh-CN", SnapshotID: snapshot, ReviewerSession: reviewer}
	for _, id := range ids[:61] {
		o.Judgments = append(o.Judgments, QualityCheckJudgment{UnitID: id, Rating: "A"})
	}
	path := qualityCheckResultsPath(root, o.Locale, o.SnapshotID)
	reject := func(before []byte) {
		t.Helper()
		stats := &qualityCheckRecordingStats{}
		_, err := recordQualityCheckJudgmentsBulk(root, c, o, stats)
		if err == nil || !strings.Contains(err.Error(), "heterogeneous judgment limit") || !strings.Contains(err.Error(), "60") {
			t.Fatal("61 legal judgments did not hit explicit bulk max-60 gate", err)
		}
		if stats.WriteAttempts != 0 || stats.SnapshotReads != 0 {
			t.Fatal("count guard ran after expensive work or publication", stats)
		}
		after, err := os.ReadFile(path)
		if before == nil {
			if !os.IsNotExist(err) {
				t.Fatal("over-limit first write mutated results", err)
			}
		} else if err != nil || !bytes.Equal(before, after) {
			t.Fatal("over-limit append changed exact results bytes", err)
		}
	}
	reject(nil)
	// The one existing result is outside the 61 requested IDs, so neither
	// already-recorded IDs nor unknown membership can explain this rejection.
	if _, err := RecordQualityCheckResults(root, c, QualityCheckRecordOptions{Locale: o.Locale, SnapshotID: snapshot, ReviewerSession: reviewer, UnitIDs: ids[61:], Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	reject(before)
	o.Judgments = o.Judgments[:60]
	stats := &qualityCheckRecordingStats{}
	if r, err := recordQualityCheckJudgmentsBulk(root, c, o, stats); err != nil || r.RecordedCount != 60 || r.ResultCount != 61 {
		t.Fatal("60 legal same-kind judgments must pass", r, err)
	}
	if stats.SnapshotReads != 1 || stats.ContextBuilds != 1 || stats.RepositoryEvidenceReads != 60 || stats.WriteAttempts != 1 {
		t.Fatal("valid-60 shared invocation contract", stats)
	}
}

func TestQualityCheckHeterogeneousSuccessorMixedKind(t *testing.T) {
	root, c, _ := makeRetranslationReviewBatchFixture(t, 2, "pages-before-example")
	source := []byte("package main\n\n// A goroutine sends the value through the channel.\nfunc main() { println(1) }\n")
	example := Example{ID: "example:basics/record-mixed.go", SourcePath: "_content/tour/basics/record-mixed.go", Source: source, SourceSHA256: sum(source)}
	c.Examples = append(c.Examples, example)
	writeSnapshotSource(t, root, example.SourcePath, source)
	exported, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", UnitKind: UnitKindExample, UnitIDs: []string{example.ID}})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, exported.BatchPath)
	m, err := readRetranslationProcessManifest(dir, "zh-CN", exported.BatchID)
	if err != nil {
		t.Fatal(err)
	}
	input, err := os.ReadFile(filepath.Join(dir, m.Units[0].InputPath))
	if err != nil {
		t.Fatal(err)
	}
	raw := strings.Replace(string(input), "A ", "一个 ", 1)
	raw = strings.Replace(raw, " sends the value through the channel.", " 通过通道发送该值。", 1)
	if err := os.MkdirAll(filepath.Join(dir, "raw-responses"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "raw-responses", filepath.Base(m.Units[0].InputPath)), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	if r, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: exported.BatchID}); err != nil || r.ValidationPassed != 1 {
		t.Fatal("real Example fixture validation", r, err)
	}
	const previous = "mixed-predecessor"
	const successor = "mixed-successor"
	for _, id := range []string{previous, successor} {
		if _, _, err := CreateQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: id}); err != nil {
			t.Fatal(err)
		}
	}
	const reviewer = "independent-mixed-reviewer"
	// Complete the predecessor using legal separate-kind invocations.
	for _, ids := range [][]string{{"lesson/1", "lesson/2"}, {example.ID}} {
		if _, err := RecordQualityCheckResults(root, c, QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: previous, ReviewerSession: reviewer, UnitIDs: ids, Rating: "A"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := RecordQualityCheckResults(root, c, QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: successor, PreviousSnapshotID: previous, ReviewerSession: reviewer, UnitIDs: []string{"lesson/2"}, Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	s, err := readQualityCheckSnapshotForReview(root, "zh-CN", successor, c)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range s.Units {
		if _, err := readSnapshotUnitRepositoryEvidence(root, c, "zh-CN", u); err != nil {
			t.Fatal("mixed fixture repository identity invalid", err)
		}
	}
	existing, err := readQualityCheckResults(root, "zh-CN", s)
	if err != nil || existing.PreviousSnapshotID != previous || existing.ReviewerSession != reviewer {
		t.Fatal("invalid existing successor lineage", err)
	}
	if err := CheckUnifiedLanguageRole(root, "zh-CN", reviewer, "reviewer"); err != nil {
		t.Fatal(err)
	}
	o := QualityCheckRecordJudgmentsOptions{Locale: "zh-CN", SnapshotID: successor, PreviousSnapshotID: previous, ReviewerSession: reviewer, Judgments: []QualityCheckJudgment{{"lesson/1", "B", "Page finding"}, {example.ID, "C", "Example finding"}}}
	path := qualityCheckResultsPath(root, o.Locale, o.SnapshotID)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stats := &qualityCheckRecordingStats{}
	_, err = recordQualityCheckJudgmentsBulk(root, c, o, stats)
	if err == nil || !strings.Contains(err.Error(), "same UnitKind") || !strings.Contains(err.Error(), "mixed page and example") {
		t.Fatal("successor mixed set must hit explicit bulk same-kind guard", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) || stats.WriteAttempts != 0 {
		t.Fatal("mixed successor set partially mutated results", stats, err)
	}
	if stats.SnapshotReads != 1 || stats.RepositoryEvidenceReads != 0 {
		t.Fatal("same-kind guard must use Snapshot resolution before per-Unit evidence work", stats)
	}
	// Each constituent request is otherwise legal in this persisted lineage.
	for _, j := range o.Judgments {
		single := o
		single.Judgments = []QualityCheckJudgment{j}
		if _, err := RecordQualityCheckJudgments(root, c, single); err != nil {
			t.Fatal("valid single-kind append rejected", err)
		}
	}
}
