package sitecontent

import (
	"bytes"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestSharedPageFinalizationWithoutReplacement(t *testing.T) {
	root := siteFixture(t, "docs/RETRANSLATION_RUNBOOK.md", "docs/TRANSLATION_QUALITY_REVIEW.md", "current-authorization-fresh-fixture")
	c, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := AttachPageTranslationUnits(root, c)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage("learn-docs-v1"); err != nil {
		t.Fatal(err)
	}
	for _, set := range [][]string{ids[:30], ids[30:]} {
		batch, _ := importSyntheticSharedPages(t, root, c, set)
		if r, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch}); err != nil || r.ValidationPassed != len(set) {
			t.Fatal(r, err)
		}
	}
	const snapshot = "synthetic-package-closure"
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: snapshot}); err != nil {
		t.Fatal(err)
	}
	for _, set := range [][]string{ids[:30], ids[30:]} {
		if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: snapshot, ReviewerSession: "synthetic-independent-reviewer", UnitIDs: set, Rating: "A"}); err != nil {
			t.Fatal(err)
		}
	}
	_, finalization, err := i18n.FinalizeQualityCheck(root, c, i18n.QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := ReferenceFile(root, finalization)
	if err != nil {
		t.Fatal(err)
	}
	_, checkpoint, err := i18n.EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot)
	if err != nil {
		t.Fatal("shared current checkpoint", err)
	}
	adapterFile := "internal/sitecontent/translation_adapter.go"
	adapterBytes, err := readRegular(root, adapterFile)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, adapterFile, append(adapterBytes, []byte("\nfunc unrelatedFutureLoaderFixture() {}\n")...))
	if _, err := i18n.CheckCurrentTranslationAuthorization(root, c, checkpoint); err != nil {
		t.Fatal("unrelated adapter-file growth invalidated checkpoint", err)
	}
	_, structured := importStructuredFixture(t, root)
	assertSharedPageClosure(t, root, "zh-CN", ref, []Reference{structured})
	// Steady-state consumers must ignore unusable pre-checkpoint compatibility
	// logs while preserving original Reviewer/candidate/validation hashes.
	writeFixture(t, root, "data/glossary-compatibility/zh-CN/unused-pre-checkpoint.json", []byte("{bad replay log}\n"))
	for i := 0; i < 2; i++ {
		if _, err := i18n.CheckCurrentTranslationAuthorization(root, c, checkpoint); err != nil {
			t.Fatal(err)
		}
		assertSharedPageClosure(t, root, "zh-CN", ref, []Reference{structured})
	}
	// Legacy Tour glossary coverage must not authorize a new package closure.
	if err := os.Remove(path.Join(root, "data/unified-glossary-reviews/zh-CN/synthetic-unified-review.review.json")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := BuildPackageClosure(root, "zh-CN", "learn-docs-v1", &ref, []Reference{structured}); err == nil {
		t.Fatal("package closure accepted missing current unified Review")
	}
}

// Reuse the shared replacement lifecycle fixture. All mutations are confined
// to its t.TempDir; neither a new lifecycle nor repository evidence is created.
func assertReplacementPackageClosure(t *testing.T, root string, c *i18n.Catalog, pkg, snapshot string, oldReview, surface Reference, predecessor, successor, representations string) {
	t.Helper()
	withoutReader := *c
	withoutReader.SurfaceAuthorityReader = nil
	if _, err := i18n.FinalizedTranslationPackageCandidates(root, &withoutReader, "zh-CN", snapshot); err == nil || !strings.Contains(err.Error(), "Integrated Surface authority reader required") {
		t.Fatal("shared replacement boundary must still require reader", err)
	}
	AttachIntegratedSurfaceAuthority(c)
	check := func(t *testing.T) {
		t.Helper()
		closure, files, err := BuildTranslationPackageClosure(root, c, "zh-CN", pkg, snapshot)
		if err != nil || len(files) != 6 || len(closure.Files) != 6 {
			t.Fatal("replacement descriptor closure", err)
		}
	}
	check(t)

	// A later corpus projection changes current Review coverage, not the bytes
	// or validity of the already frozen historical Surface/reopen authority.
	var review i18n.UnifiedGlossaryReview
	if err := readReference(root, oldReview, &review); err != nil {
		t.Fatal(err)
	}
	corpus, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		t.Fatal(err)
	}
	corpus.Contributors[0].Contexts[0].Text += " Additional projected terminology context."
	corpus.Contributors[0].SourceSHA = i18n.ContributorIdentity(corpus.Contributors[0])
	corpus.Identity = i18n.CorpusIdentity(*corpus)
	if corpus.Identity == review.CorpusSHA {
		t.Fatal("fixture corpus did not stale predecessor Review")
	}
	corpusBytes, err := Encode(corpus)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, i18n.GlossaryCorpusPath, corpusBytes)
	if _, err := i18n.RequireUnifiedGlossaryReview(root, "zh-CN"); err == nil {
		t.Fatal("old Review remained current after corpus drift")
	}
	bundle, _, err := i18n.ExportUnifiedGlossaryBundle(root, "zh-CN", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.RecordUnifiedGlossaryReview(root, bundle, "synthetic-successor-unified-review", "synthetic-independent-reviewer", "synthetic-generation", "passed", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.RequireUnifiedGlossaryReview(root, "zh-CN"); err != nil {
		t.Fatal("current successor unified Review required", err)
	}
	// Historical replacement validation must never demand equality to a rebuilt
	// current Surface scope. It still validates all frozen authority hashes.
	attachIntegratedSurfaceAuthority(c, func(string, string, []Reference) (*IntegratedSurfaceScope, error) {
		t.Fatal("historical replacement proof attempted current Surface rebuilding")
		return nil, nil
	})
	check(t)

	finalPath := path.Join("data/quality-check-snapshots/zh-CN", snapshot, "finalization.json")
	finalRef, err := ReferenceFile(root, finalPath)
	if err != nil {
		t.Fatal(err)
	}
	var final i18n.QualityCheckFinalization
	if err := readReference(root, finalRef, &final); err != nil {
		t.Fatal(err)
	}
	unit := final.Units[2].Snapshot
	var receipt IntegratedSurfaceReceipt
	if err := readReference(root, surface, &receipt); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, file string
		missing    bool
	}{
		{"missing-Surface-authority", surface.Path, true},
		{"fake-Surface-authority", surface.Path, false},
		{"stale-Surface-evidence", receipt.Evidence.Path, false},
		{"missing-predecessor-reopen", predecessor, true},
		{"fake-successor-reopen", successor, false},
		{"stale-historical-representations", representations, false},
		{"stale-historical-glossary-Review-bytes", oldReview.Path, false},
		{"changed-current-candidate", unit.CandidatePath, false},
		{"changed-current-validation", unit.ValidationPath, false},
		{"changed-current-finalization", finalPath, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before, err := readRegular(root, tc.file)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, tc.file, before)
			if tc.missing {
				if err := os.Remove(path.Join(root, tc.file)); err != nil {
					t.Fatal(err)
				}
			} else {
				writeFixture(t, root, tc.file, append(bytes.Clone(before), []byte("forged mutation\n")...))
			}
			if _, _, err := BuildTranslationPackageClosure(root, c, "zh-CN", pkg, snapshot); err == nil {
				t.Fatal("package closure accepted missing/fake/stale authority or mutated candidate evidence")
			}
		})
	}
	check(t)
}

// Opt-in frozen evidence regression is entirely read-only. BuildPackageClosure
// projects in memory; FinalizePackageClosure must never be called here.
func TestSharedPageFinalizationReplacementFrozenSnapshot(t *testing.T) {
	finalization := os.Getenv("GO_TOUR_PACKAGE_CLOSURE_FINALIZATION")
	structured := os.Getenv("GO_TOUR_PACKAGE_CLOSURE_STRUCTURED")
	if finalization == "" || structured == "" {
		t.Skip("explicit read-only finalized Page replacement fixture")
	}
	root := repositoryRoot(t)
	ref, err := ReferenceFile(root, finalization)
	if err != nil {
		t.Fatal(err)
	}
	var f i18n.QualityCheckFinalization
	if err := readReference(root, ref, &f); err != nil {
		t.Fatal(err)
	}
	generation, err := ReferenceFile(root, structured)
	if err != nil {
		t.Fatal(err)
	}
	assertSharedPageClosure(t, root, f.Locale, ref, []Reference{generation})
}

func assertSharedPageClosure(t *testing.T, root, locale string, finalization Reference, structured []Reference) {
	t.Helper()
	before, err := readRegular(root, finalization.Path)
	if err != nil {
		t.Fatal(err)
	}
	var f i18n.QualityCheckFinalization
	if err := StrictJSON(before, &f); err != nil {
		t.Fatal(err)
	}
	out := map[string][]byte{}
	if err := projectSharedPageFinalization(root, locale, finalization, out); err != nil {
		t.Fatal("shared Page projection", err)
	}
	if len(out) != len(f.Units) {
		t.Fatal("projection lost finalized Pages")
	}
	for _, u := range f.Units {
		candidate, err := readRegular(root, u.Snapshot.CandidatePath)
		if err != nil || !bytes.Equal(candidate, out[u.Snapshot.SourcePath]) {
			t.Fatal("projection did not use exact finalized candidate", u.UnitID, err)
		}
	}
	closure, files, err := BuildPackageClosure(root, locale, "learn-docs-v1", &finalization, structured)
	if err != nil {
		t.Fatal("package closure", err)
	}
	if len(files) != 56 || len(closure.Files) != 56 {
		t.Fatal("incomplete Page/structured closure")
	}
	for p, b := range out {
		if !bytes.Equal(files[p], b) {
			t.Fatal("closure changed finalized Page bytes", p)
		}
	}
	if _, err := os.Stat(path.Join(root, workflowPath(locale, "learn-docs-v1", "closure", closure.Identity))); !os.IsNotExist(err) {
		t.Fatal("test must not create successor closure", err)
	}
	after, err := readRegular(root, finalization.Path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("projection modified finalization", err)
	}
}
