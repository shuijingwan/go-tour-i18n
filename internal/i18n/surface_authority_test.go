package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSurfaceAuthorityReaderIsNotCatalogIdentity(t *testing.T) {
	c := &Catalog{}
	before, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	c.SurfaceAuthorityReader = func(string, string, string, bool, *Catalog) (*SurfaceReviewAuthority, error) {
		t.Fatal("identity encoding invoked authority reader")
		return nil, nil
	}
	c.HistoricalContextReader = func(string, string, GlossaryArchiveReference) ([]byte, error) {
		t.Fatal("identity encoding invoked context verifier")
		return nil, nil
	}
	after, err := json.Marshal(c)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("runtime reader changed canonical catalog/source identity", err)
	}
}

func TestSurfaceAuthorityContextTransitionIsHistoricalOnly(t *testing.T) {
	root := t.TempDir()
	write := func(name, value string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("receipt.json", "immutable receipt")
	write("evidence.md", "immutable evidence")
	write("context.json", "baseline context")
	a := &SurfaceReviewAuthority{Locale: "zh-CN", Receipt: GlossaryArchiveReference{Path: "receipt.json", SHA256: sum([]byte("immutable receipt"))}, Evidence: GlossaryArchiveReference{Path: "evidence.md", SHA256: sum([]byte("immutable evidence"))}, Context: []GlossaryArchiveReference{{Path: "context.json", SHA256: sum([]byte("baseline context"))}}}
	c := &Catalog{}
	if err := verifySurfaceAuthorityFiles(root, c, a, false); err != nil {
		t.Fatal(err)
	}
	write("context.json", "expected successor context")
	if err := verifySurfaceAuthorityFiles(root, c, a, false); err == nil {
		t.Fatal("historical context drift accepted without adapter proof")
	}
	calls := 0
	c.HistoricalContextReader = func(_ string, locale string, ref GlossaryArchiveReference) ([]byte, error) {
		calls++
		if locale != a.Locale || ref != a.Context[0] {
			t.Fatal("adapter received a different frozen context")
		}
		return []byte("baseline context"), nil // Sitecontent tests verify the lifecycle proof.
	}
	if err := verifySurfaceAuthorityFiles(root, c, a, false); err != nil || calls != 1 {
		t.Fatal("historical adapter proof not consumed", err)
	}
	if _, err := resolveSurfaceReopenAuthority(root, c, QualityCheckSurfaceReopenOptions{Locale: a.Locale, Finding: "synthetic finding"}, false); err == nil || err.Error() != "Integrated Surface authority reader required" {
		t.Fatal("context verifier replaced the required independent authority reader", err)
	}
	if err := verifySurfaceAuthorityFiles(root, c, a, true); err == nil || calls != 1 {
		t.Fatal("current reopen consumed a historical transition exception")
	}
	for _, file := range []string{"receipt.json", "evidence.md"} {
		before, _ := os.ReadFile(filepath.Join(root, file))
		write(file, "rewritten immutable evidence")
		if err := verifySurfaceAuthorityFiles(root, c, a, false); err == nil || calls != 1 {
			t.Fatal("immutable evidence drift delegated to context adapter")
		}
		if err := os.WriteFile(filepath.Join(root, file), before, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSurfaceCandidateArtifactLookupIsInventoryWide(t *testing.T) {
	root, c, ids := futureContentFixture(t)
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
		if result, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID}); err != nil || result.ValidationPassed != 1 {
			t.Fatal(result, err)
		}
	}
	snapshot := futureSnapshot(t, root, c, "surface-artifacts")
	artifact := SurfaceCandidateArtifact{SnapshotID: snapshot.SnapshotID, Unit: snapshot.Units[2]}
	if err := c.UseTranslationUnitScope(ids[:1]); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSurfaceCandidateArtifact(root, c, "zh-CN", artifact); err != nil {
		t.Fatal("selected scheduling scope hid integrated inventory", err)
	}
	if len(c.TranslationScope) != 1 || c.TranslationScope[0] != ids[0] {
		t.Fatal("read-only artifact lookup changed selected scope")
	}
	// This identity lookup does not authorize a partial lifecycle Snapshot/QC.
	if _, err := readQualityCheckSnapshotForReview(root, "zh-CN", snapshot.SnapshotID, c); err == nil {
		t.Fatal("artifact lookup weakened complete lifecycle scope")
	}
	artifact.Unit.UnitID = ids[0]
	if err := ValidateSurfaceCandidateArtifact(root, c, "zh-CN", artifact); err == nil {
		t.Fatal("claimed ID overrode exact source mapping")
	}
}
