package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// A complete registered package with carried A, a new glossary endpoint, and
// no legacy compatibility replay. The CLI's inventory also contains Tour but
// generation-bundle export need not receive an explicit --content-package.
func previewRevisionPackageFixture(t *testing.T) (string, *Catalog, GenerationBundleOptions, QualityCheckSnapshotUnit) {
	t.Helper()
	root, c, ids, previous := checkpointFixture(t)
	if _, _, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", previous); err != nil {
		t.Fatal(err)
	}
	checkpointGlossary(t, root, "# reviewed current endpoint\n", "preview-endpoint-review")
	const snapshot = "preview-carried-a"
	s := futureSnapshot(t, root, c, snapshot)
	if _, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: snapshot, PreviousSnapshotID: previous}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot); err != nil {
		t.Fatal(err)
	}
	p, err := PreparePreviewTUFailure(root, c, "zh-CN", snapshot, ids[0])
	if err != nil {
		t.Fatal(err)
	}
	p.FailureID, p.Package, p.PackageSHA256 = "preview-carried-failure", c.TranslationPackages[0].ID, sum([]byte("fixture-package"))
	p.Route, p.PreviewURL = "/fixture/cards/1", "http://127.0.0.1:3999/"
	p.ContextSHA256, p.ResponseHTML = sum([]byte("fixture-build")), "<main>exact machine failure fixture</main>"
	p.ResponseSHA256, p.MissingLinks = sum([]byte(p.ResponseHTML)), []string{"https://example.invalid/required"}
	c.PreviewFailureReader = func(_ string, _ *Catalog, _ *PreviewTUFailure, _ bool) error { return nil }
	path, err := RecordPreviewTUFailure(root, c, *p)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = RecordQualityCheckSurfaceReopen(root, c, QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: "preview-carried-reopen", PreviousSnapshotID: snapshot, UnitIDs: ids[:1], PreviewFailure: GlossaryArchiveReference{Path: path, SHA256: sum(b)}})
	if err != nil {
		t.Fatal(err)
	}
	e, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitIDs: ids[:1], AllowReexport: true, PreviousSnapshotID: snapshot, SurfaceReopenID: "preview-carried-reopen"})
	if err != nil {
		t.Fatal(err)
	}
	copyBundleAuthority(t, root, []string{"docs/PREVIEW_TU_FAILURE.md"})
	c.Pages = retranslationTestCatalog(1).Pages
	return root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID}, s.Units[0]
}

func TestPreviewRevisionGenerationBundleInfersFrozenPackageScope(t *testing.T) {
	root, c, options, predecessor := previewRevisionPackageFixture(t)
	manifestPath := filepath.Join(root, "data", "retranslation-runs", options.Locale, options.BatchID, "manifest.json")
	before, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, memo := range []bool{false, true} {
		t.Run(fmt.Sprintf("memo=%v", memo), func(t *testing.T) {
			if memo {
				done := contentidentity.BeginMemoization()
				defer done()
			}
			inventory := *c
			inventory.TranslationScope = nil
			bundle, m, err := ExportTranslationUnitGenerationBundle(root, &inventory, options)
			if err != nil {
				t.Fatal(err)
			}
			if len(inventory.TranslationScope) != 0 || len(m.ExpectedOutputs) != 1 || m.ExpectedOutputs[0].UnitID != predecessor.UnitID {
				t.Fatal("scope inference mutated caller or expanded revision")
			}
			_, files, err := readGenerationBundle(bundle)
			if err != nil {
				t.Fatal(err)
			}
			candidatePath := "formal/candidates/" + filepath.Base(predecessor.CandidatePath)
			if sum(files[candidatePath]) != predecessor.CandidateSHA256 {
				t.Fatal("frozen candidate was replaced")
			}
			found := false
			for _, f := range m.Files {
				if f.BundlePath == candidatePath {
					found = f.RepositoryPath == predecessor.CandidatePath && f.SHA256 == predecessor.CandidateSHA256
				}
			}
			if !found {
				t.Fatal("transport lacks exact predecessor binding")
			}
			// Reusing a scope must never turn a candidate mismatch into an A.
			candidate, err := os.ReadFile(filepath.Join(root, predecessor.CandidatePath))
			if err != nil {
				t.Fatal(err)
			}
			writeSnapshotSource(t, root, predecessor.CandidatePath, append(bytes.Clone(candidate), '\n'))
			_, _, err = ExportTranslationUnitGenerationBundle(root, &inventory, options)
			writeSnapshotSource(t, root, predecessor.CandidatePath, candidate)
			if err == nil {
				t.Fatal("candidate mismatch authorized revision")
			}
			wrong := *c
			wrong.TranslationScope = []string{predecessor.UnitID}
			if _, _, err := ExportTranslationUnitGenerationBundle(root, &wrong, options); err == nil {
				t.Fatal("explicit partial scope was inferred away")
			}
		})
	}
	after, err := os.ReadFile(manifestPath)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("saved batch was rewritten", err)
	}
}

func TestRevisionGenerationBundleOrdinaryRatings(t *testing.T) {
	for _, rating := range []string{"B", "C", "D"} {
		t.Run(rating, func(t *testing.T) {
			root, c, _ := makeRetranslationReviewBatchFixture(t, 1, "qc-001")
			copyBundleAuthority(t, root, translationUnitGenerationAuthorityPaths)
			recordQualityCheckRatings(t, root, c, "qc-001", "", rating, []string{"lesson/1"})
			e, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", UnitIDs: []string{"lesson/1"}, AllowReexport: true, PreviousSnapshotID: "qc-001"})
			if err != nil {
				t.Fatal(err)
			}
			opts := GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID}
			b, m, err := ExportTranslationUnitGenerationBundle(root, c, opts)
			if err != nil || m.TaskKind != "translation-unit-revision" {
				t.Fatalf("ordinary %s revision: %v", rating, err)
			}
			assertRevisionCandidates(t, root, c, opts, b)
		})
	}
}
