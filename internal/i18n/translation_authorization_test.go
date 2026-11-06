package i18n

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func checkpointFixture(t *testing.T) (string, *Catalog, []string, string) {
	t.Helper()
	root, c, ids := futureContentFixture(t)
	// Generic XML, explicit source registry and representation-owned authority.
	for i := range c.Adapted {
		card := c.Adapted[i].Adapter.(futureCardAdapter).source
		card.Message = fmt.Sprintf("Read message number %d.", i+1)
		if i == 0 {
			card.Message = "Read the unique apple message."
		}
		b, _ := xml.Marshal(card)
		b = append(b, '\n')
		c.Adapted[i].Unit.Source = b
		c.Adapted[i].Unit.SourceSHA256 = sum(b)
		c.Adapted[i].Adapter = futureCardAdapter{card}
		c.TranslationPackages[0].Units[i] = c.Adapted[i]
		writeSnapshotSource(t, root, c.Adapted[i].Unit.SourcePath, b)
	}
	checkpointGenerate(t, root, c, ids)
	snapshot := "checkpoint-initial"
	futureSnapshot(t, root, c, snapshot)
	for _, id := range ids {
		futureRecord(t, root, c, snapshot, "", "A", "", []string{id})
	}
	if _, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: snapshot}); err != nil {
		t.Fatal(err)
	}
	return root, c, ids, snapshot
}

func checkpointGenerate(t *testing.T, root string, c *Catalog, ids []string) {
	t.Helper()
	for _, id := range ids {
		e, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitIDs: []string{id}})
		if err != nil {
			t.Fatal(err)
		}
		b, m, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
		if err != nil {
			t.Fatal(err)
		}
		_, files, err := readGenerationBundle(b)
		if err != nil {
			t.Fatal(err)
		}
		futureImport(t, root, c, b, map[string][]byte{id: files["formal/inputs/"+filepath.Base(m.ExpectedOutputs[0].BundlePath)]})
		r, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID})
		if err != nil || r.ValidationPassed != 1 {
			t.Fatalf("fresh process: %+v %v", r, err)
		}
	}
}

func checkpointGlossary(t *testing.T, root, extra, review string) {
	t.Helper()
	p := "locales/zh-CN/glossary.yaml"
	b, err := os.ReadFile(filepath.Join(root, p))
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotSource(t, root, p, append(b, []byte(extra)...))
	recordSyntheticUnifiedReview(t, root, review, "zh-CN")
}

func TestCurrentTranslationAuthorizationFreshAndUnrelatedPackage(t *testing.T) {
	root, c, ids, snapshot := checkpointFixture(t)
	a, path, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot)
	if err != nil || !a.Ready() || a.Compatible != 3 {
		t.Fatalf("fresh: %+v %v", a, err)
	}
	// Deliberately unusable intermediate compatibility log: a sealed proof
	// and direct endpoint assessment cannot consult that historical replay log.
	writeSnapshotSource(t, root, "data/glossary-compatibility/zh-CN/unrelated-intermediate.json", []byte("{broken\n"))
	for i := 0; i < 2; i++ {
		if _, err := CheckCurrentTranslationAuthorization(root, c, path); err != nil {
			t.Fatal("steady-state replayed old log", err)
		}
	}
	d := TranslationPackageDescriptor{ID: "second-unrelated-package", AuthoritySHA256: sum([]byte("second-source-authority"))}
	secondIDs := []string{}
	for i, old := range c.TranslationPackages[0].Units {
		v := old
		v.Unit.ID = fmt.Sprintf("second/card-%d", i+1)
		v.Unit.SourcePath = fmt.Sprintf("fixtures/second/card-%d.xml", i+1)
		v.Route = fmt.Sprintf("/second/%d", i+1)
		writeSnapshotSource(t, root, v.Unit.SourcePath, v.Unit.Source)
		d.Units = append(d.Units, v)
		secondIDs = append(secondIDs, v.Unit.ID)
	}
	if err := c.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckCurrentTranslationAuthorization(root, c, path); err != nil {
		t.Fatal("package append invalidated old scope", err)
	}
	if err := c.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	checkpointGenerate(t, root, c, secondIDs)
	futureSnapshot(t, root, c, "checkpoint-second")
	for _, id := range secondIDs {
		futureRecord(t, root, c, "checkpoint-second", "", "A", "", []string{id})
	}
	if _, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "checkpoint-second"}); err != nil {
		t.Fatal(err)
	}
	second, _, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", "checkpoint-second")
	if err != nil || second.Package != d.ID || second.Compatible != len(ids) {
		t.Fatalf("second primitive: %+v %v", second, err)
	}
}

func TestCurrentTranslationReattestationGlossaryByteOnlyAndMissingIntermediate(t *testing.T) {
	root, c, _, snapshot := checkpointFixture(t)
	_, oldPath, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot)
	if err != nil {
		t.Fatal(err)
	}
	checkpointGlossary(t, root, "# semantic byte-only change\n", "checkpoint-byte-review")
	if _, err := CheckCurrentTranslationAuthorization(root, c, oldPath); err == nil {
		t.Fatal("old glossary checkpoint remained byte-current")
	}
	writeSnapshotSource(t, root, "data/glossary-compatibility/zh-CN/broken-history.json", []byte("not a compatibility proof\n"))
	a, path, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot)
	if err != nil || !a.Ready() || a.Compatible != 3 || a.Affected != 0 {
		t.Fatalf("direct endpoint carry: %+v %v", a, err)
	}
	if _, err := CheckCurrentTranslationAuthorization(root, c, path); err != nil {
		t.Fatal(err)
	}
	if len(a.Historical.Finalization.QCResults) != 1 {
		t.Fatal("re-attestation fabricated Reviewer results")
	}
}

func TestCurrentTranslationReattestationAffectedOnlyAndSuccessorQC(t *testing.T) {
	testCurrentTranslationAffectedOnly(t, true)
}

func TestCurrentTranslationFirstMigrationAffectedOnly(t *testing.T) {
	testCurrentTranslationAffectedOnly(t, false)
}

// Build the intended lexical revision inside the fixture's editable prose
// field. Source English, code, attributes, URLs and envelope tokens stay exact.
func localizedAppleCardFixtureRevision(t *testing.T, raw []byte) []byte {
	t.Helper()
	prefix, rest, ok := strings.Cut(string(raw), futureCardTokens[1])
	if !ok {
		t.Fatal("fixture message boundary missing")
	}
	message, suffix, ok := strings.Cut(rest, futureCardTokens[2])
	if !ok || !strings.Contains(message, "apple") || !strings.HasSuffix(message, ".") {
		t.Fatal("expected ordinary apple message fixture")
	}
	message = strings.TrimSuffix(strings.ReplaceAll(message, "apple", "苹果"), ".") + "。"
	return []byte(prefix + futureCardTokens[1] + message + futureCardTokens[2] + suffix)
}

func testCurrentTranslationAffectedOnly(t *testing.T, sealed bool) {
	t.Helper()
	root, c, ids, snapshot := checkpointFixture(t)
	if sealed {
		if _, _, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot); err != nil {
			t.Fatal(err)
		}
	}
	checkpointGlossary(t, root, "preferred:\n  apple: 苹果\n", "checkpoint-affected-review")
	a, err := AssessCurrentTranslation(root, c, "zh-CN", snapshot)
	if err != nil || a.Affected != 1 || a.Compatible != 2 || a.Units[0].Classification != "affected" {
		t.Fatalf("affected exact-set: %+v %v", a, err)
	}
	if _, _, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot); err == nil {
		t.Fatal("affected complete scope authorized")
	}
	futureSnapshot(t, root, c, "checkpoint-current")
	scope, err := BuildQualityCheckScope(root, c, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "checkpoint-current", PreviousSnapshotID: snapshot})
	if err != nil || scope.CarryForwardCount != 2 || scope.PendingCount != 1 || scope.Pending[0].UnitID != ids[0] {
		t.Fatalf("affected-only QC planning: %+v %v", scope, err)
	}
	// Only this fixture's independent Reviewer records a B finding. The same
	// existing exact revision/re-QC path handles the replacement.
	futureRecord(t, root, c, "checkpoint-current", snapshot, "B", "Replace the affected apple wording.", []string{ids[0]})
	e, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitIDs: []string{ids[0]}, AllowReexport: true, PreviousSnapshotID: "checkpoint-current"})
	if err != nil {
		t.Fatal(err)
	}
	b, m, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	_, files, err := readGenerationBundle(b)
	if err != nil {
		t.Fatal(err)
	}
	raw := localizedAppleCardFixtureRevision(t, files["formal/inputs/"+filepath.Base(m.ExpectedOutputs[0].BundlePath)])
	futureImport(t, root, c, b, map[string][]byte{ids[0]: raw})
	if result, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID}); err != nil || result.ValidationPassed != 1 {
		t.Fatalf("exact revision: %+v %v", result, err)
	}
	futureSnapshot(t, root, c, "checkpoint-successor")
	futureRecord(t, root, c, "checkpoint-successor", "checkpoint-current", "A", "", []string{ids[0]})
	if _, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "checkpoint-successor", PreviousSnapshotID: "checkpoint-current"}); err != nil {
		t.Fatal(err)
	}
	a, _, err = EstablishCurrentTranslationAuthorization(root, c, "zh-CN", "checkpoint-successor")
	if err != nil || !a.Ready() {
		t.Fatalf("successor A-only: %+v %v", a, err)
	}
}

type checkpointRejectingAdapter struct{ futureCardAdapter }

func (a checkpointRejectingAdapter) ReattestationAuthority(string) (string, error) {
	return sum([]byte("rejecting-parser-contract/v2")), nil
}
func (a checkpointRejectingAdapter) Validate([]byte, *Glossary) error {
	return fmt.Errorf("new current parser rejects accepted content")
}

func TestCurrentTranslationReattestationFailClosedMatrix(t *testing.T) {
	for _, kind := range []string{"source", "candidate", "parser", "reviewer", "finalization", "checkpoint", "unknown"} {
		t.Run(kind, func(t *testing.T) {
			root, c, _, snapshot := checkpointFixture(t)
			a, path, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "source":
				v := &c.Adapted[0]
				v.Unit.Source = append(v.Unit.Source, []byte("\n")...)
				v.Unit.SourceSHA256 = sum(v.Unit.Source)
				c.TranslationPackages[0].Units[0] = *v
				writeSnapshotSource(t, root, v.Unit.SourcePath, v.Unit.Source)
				assessment, err := AssessCurrentTranslation(root, c, "zh-CN", snapshot)
				if err != nil || assessment.Stale != 1 || assessment.Compatible != 2 {
					t.Fatalf("source exact stale: %+v %v", assessment, err)
				}
			case "candidate":
				writeSnapshotSource(t, root, a.Units[0].Snapshot.CandidatePath, []byte("corrupt accepted bytes\n"))
			case "parser":
				c.Adapted[0].Adapter = checkpointRejectingAdapter{c.Adapted[0].Adapter.(futureCardAdapter)}
				c.TranslationPackages[0].Units[0] = c.Adapted[0]
				assessment, err := AssessCurrentTranslation(root, c, "zh-CN", snapshot)
				if err != nil || assessment.Affected != 1 || assessment.Units[0].Validation != "failed" {
					t.Fatalf("parser fail closed: %+v %v", assessment, err)
				}
			case "reviewer":
				writeSnapshotSource(t, root, a.Historical.Finalization.QCResults[0].Path, []byte("{missing original Reviewer A}\n"))
			case "finalization":
				writeSnapshotSource(t, root, "data/quality-check-snapshots/zh-CN/"+snapshot+"/finalization.json", []byte("{bad}\n"))
			case "checkpoint":
				writeSnapshotSource(t, root, path, []byte("{bad}\n"))
			case "unknown":
				c.Adapted[0].Adapter = projectionTestAdapter{"unknown representation"}
				c.TranslationPackages[0].Units[0] = c.Adapted[0]
			}
			if _, err := CheckCurrentTranslationAuthorization(root, c, path); err == nil {
				t.Fatalf("%s drift accepted", kind)
			}
			if kind == "candidate" || kind == "reviewer" || kind == "finalization" || kind == "checkpoint" {
				if _, err := AssessCurrentTranslation(root, c, "zh-CN", snapshot); err == nil {
					t.Fatal("corrupt original proof re-attested")
				}
			}
		})
	}
}

func TestHistoricalFinalizationIntegrityIndependentOfCurrentGlossary(t *testing.T) {
	root, _, _, snapshot := checkpointFixture(t)
	writeSnapshotSource(t, root, "locales/zh-CN/glossary.yaml", []byte("current world deliberately unreadable\n"))
	p, err := VerifyHistoricalQualityCheckFinalization(root, "zh-CN", snapshot)
	if err != nil || len(p.Finalization.Units) != 3 {
		t.Fatalf("historical/current still coupled: %+v %v", p, err)
	}
	if strings.Contains(fmt.Sprint(p.References), "glossary-compatibility/") {
		t.Fatal("endpoint integrity retained replay requirement")
	}
}

func TestHistoricalFinalizationIntegrityLegacyGlossaryRequiresOriginalGate(t *testing.T) {
	root := t.TempDir()
	writeSnapshotSource(t, root, glossaryReviewGlossaryPath("zh-CN"), []byte("mandatory:\n  Go: Go\n"))
	writeLegacyGlossaryReviewTestCoverage(t, root, "zh-CN")
	b, err := readCompatibilityFile(root, legacyGlossaryReviewCoveragePath)
	if err != nil {
		t.Fatal(err)
	}
	var coverage legacyGlossaryReviewCoverage
	if err := json.Unmarshal(b, &coverage); err != nil {
		t.Fatal(err)
	}
	e := coverage.Entries[0]
	refs := map[string]string{}
	bind := func(path, expected string) ([]byte, error) {
		b, err := readCompatibilityFile(root, path)
		if err != nil || sum(b) != expected {
			return nil, fmt.Errorf("original hash missing/changed: %s", path)
		}
		refs[path] = expected
		return b, nil
	}
	writeSnapshotSource(t, root, glossaryReviewGlossaryPath("zh-CN"), []byte("current glossary is independently checked\n"))
	if err := bindHistoricalGlossaryReview(root, "zh-CN", e.GlossarySHA256, "", bind); err != nil {
		t.Fatal("legacy proof coupled to current glossary", err)
	}
	if refs[e.SurfaceReviewGatePath] != e.SurfaceReviewGateSHA256 {
		t.Fatal("original legacy judgment not sealed")
	}
	if err := os.Remove(filepath.Join(root, e.SurfaceReviewGatePath)); err != nil {
		t.Fatal(err)
	}
	if err := bindHistoricalGlossaryReview(root, "zh-CN", e.GlossarySHA256, "", bind); err == nil {
		t.Fatal("missing original legacy judgment accepted")
	}
}

func TestCurrentTranslationCanonicalCheckpointAvoidsSuccessorReplay(t *testing.T) {
	root, c, _, snapshot := checkpointFixture(t)
	proof, err := VerifyHistoricalQualityCheckFinalization(root, "zh-CN", snapshot, c)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range proof.Finalization.Units {
		u, err := c.Unit(x.UnitID)
		if err != nil {
			t.Fatal(err)
		}
		path, err := canonicalTranslationUnitCandidatePath("zh-CN", u)
		if err != nil {
			t.Fatal(err)
		}
		b, err := readCompatibilityFile(root, x.Snapshot.CandidatePath)
		if err != nil {
			t.Fatal(err)
		}
		writeSnapshotSource(t, root, path, canonicalizeCandidateEOF(b))
	}
	a, path, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot, true)
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotSource(t, root, "data/promotion-recoveries/zh-CN/broken-pre-checkpoint.json", []byte("{unavailable old replay}\n"))
	for i := 0; i < 2; i++ {
		f, refs, err := RequireCurrentCanonicalQualityCheck(root, "zh-CN", c)
		if err != nil || f.SnapshotID != snapshot || len(refs) != 4 {
			t.Fatalf("canonical steady-state: %+v %v", f, err)
		}
	}
	writeSnapshotSource(t, root, a.Canonical[0].Path, []byte("unaccepted changed canonical\n"))
	if _, err := CheckCurrentTranslationAuthorization(root, c, path); err == nil {
		t.Fatal("canonical drift carried old A")
	}
}

func TestCurrentTranslationMissingHistoricalIntermediateCompatibility(t *testing.T) {
	root, c, _, snapshot := checkpointFixture(t)
	// A synthetic legacy encoding declares an intermediate compatibility ref.
	// Original exact Reviewer A remains real; no language judgment is invented.
	path := qualityCheckFinalizationPath(root, "zh-CN", snapshot)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f QualityCheckFinalization
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	for i := range f.Units {
		f.Units[i].CompatibilityChain = []GlossaryArchiveReference{{Path: "data/glossary-compatibility/zh-CN/missing-intermediate.json", SHA256: strings.Repeat("1", 64)}}
	}
	b, err = json.MarshalIndent(f, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotSource(t, root, filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator))), append(b, '\n'))
	checkpointGlossary(t, root, "# new endpoint full Review\n", "checkpoint-missing-middle-review")
	a, _, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot)
	if err != nil || a.Compatible != 3 {
		t.Fatalf("original A + exact current endpoint was insufficient: %+v %v", a, err)
	}
}
