package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Supply the ordinary full-inventory fixture inputs, not a fabricated prepared
// resolver or a cached PASS. Every edge below is built by the native assessor.
func recordedScopeCompatibilityInputs(t *testing.T, root string) {
	t.Helper()
	for _, locale := range []string{"en", "zh-CN"} {
		writeSnapshotSource(t, root, "internal/tour/ui/"+locale+".json", []byte(`{"locale":"`+locale+`","html_lang":"`+locale+`","messages":{"test.one":{"kind":"plain","text":"Welcome"}}}`))
	}
	writeSnapshotSource(t, root, "locales/zh-CN/article-metadata.json", []byte(`{"locale":"zh-CN","articles":[]}`))
	writeCourseMetadataFixture(t, root, "zh-CN", &CourseMetadata{SchemaVersion: 1, Locale: "zh-CN", GeneratorContract: CourseMetadataGeneratorContract})
	for _, name := range []string{"internal/tour/languages.go", "internal/tour/tour.go", "internal/tour/production.go", "internal/tour/project.go", "internal/tour/seo.go", "_content/js/playground.js", "_content/tour/static/js/app.js", "_content/tour/template/index.tmpl", "_content/tour/static/partials/index.html"} {
		writeSnapshotSource(t, root, name, []byte("package tour\nconst Config = \"Tour\"\n"))
	}
}

func TestRecordedReconciliationGlossaryAuthorityScope(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, event, _, plan, ref, receiptBytes := authorizationMachineFixtureSize(t, 49)
	resolution, err := event.ReplacementResolver(root, event, plan.Receipt.Options.HistoricalDocuments)
	if err != nil {
		t.Fatal(err)
	}
	historical := resolution.HistoricalCatalog
	recordedScopeCompatibilityInputs(t, root)
	checkpointGlossary(t, root, "preferred:\n  AbsentFixtureGlossaryTerm: 固定术语\n", "recorded-scope-next-glossary")
	currentGlossary, err := readCompatibilityFile(root, "locales/zh-CN/glossary.yaml")
	if err != nil || sum(currentGlossary) == plan.Receipt.GlossarySHA256 {
		t.Fatal("old/new glossary must differ", err)
	}
	evidence, path, err := AssessGlossaryCompatibility(root, "zh-CN", plan.Receipt.GlossarySHA256, historical)
	if err != nil {
		t.Fatal("native historical-scope compatibility authority", err)
	}
	current, added := expandedMachinePackage(t, root, event, 2)
	addition, err := HistoricalTranslationPackageCatalog(current, added.ID, added.AuthoritySHA256)
	if err != nil {
		t.Fatal(err)
	}
	for _, unit := range added.Units {
		checkpointGenerate(t, root, addition, []string{unit.Unit.ID})
	}
	member := plan.Receipt.Units[0].Replacement
	scope := "tu:" + member.OldUnitID
	prepared := PrepareGlossaryCompatibility(root, "zh-CN", historical)
	got := prepared.Resolve(plan.Receipt.GlossarySHA256, sum(currentGlossary), scope)
	if got.Status != "compatible" || got.Reason != "verified_lineage" || len(got.Chain) != 1 {
		t.Fatal("exact historical authority was not consumed", got)
	}
	for _, unit := range added.Units {
		if r := prepared.Resolve(plan.Receipt.GlossarySHA256, sum(currentGlossary), "tu:"+unit.Unit.ID); r.Status == "compatible" || r.Status == "exact" {
			t.Fatal("49-Unit evidence retroactively authorized a new member", r)
		}
	}
	if _, _, err := NewRepresentationSuccessorVerifier(root, current, "zh-CN").Verify(member.OldUnitID, member.CurrentUnitID); err != nil {
		t.Fatal("49 historical / 51 current / different glossary with applicable proof", err)
	}
	if len(historical.TranslationScope) != 49 || len(current.TranslationScope) != 51 || evidence.OldGlossary.SHA256 != plan.Receipt.GlossarySHA256 || evidence.NewGlossary.SHA256 != sum(currentGlossary) {
		t.Fatal("fixture topology drift")
	}
	t.Run("context-stale", func(t *testing.T) {
		name := "internal/tour/ui/zh-CN.json"
		before, err := readCompatibilityFile(root, name)
		if err != nil {
			t.Fatal(err)
		}
		defer writeSnapshotSource(t, root, name, before)
		writeSnapshotSource(t, root, name, bytes.Replace(before, []byte("Welcome"), []byte("Changed context"), 1))
		if err := CheckGlossaryCompatibilityEvidence(root, "zh-CN", path, historical); err == nil {
			t.Fatal("changed exact context remained valid")
		}
		_, _, err = NewRepresentationSuccessorVerifier(root, current, "zh-CN").Verify(member.OldUnitID, member.CurrentUnitID)
		if err == nil || !strings.Contains(err.Error(), "reason=") {
			t.Fatal("true stale context accepted or Reason lost", err)
		}
	})
	t.Run("invalid-evidence-reason", func(t *testing.T) {
		before, err := readCompatibilityFile(root, path)
		if err != nil {
			t.Fatal(err)
		}
		defer writeSnapshotSource(t, root, path, before)
		writeSnapshotSource(t, root, path, []byte("{"))
		_, _, err = NewRepresentationSuccessorVerifier(root, current, "zh-CN").Verify(member.OldUnitID, member.CurrentUnitID)
		if err == nil || !strings.Contains(err.Error(), "status=stale evidence reason=invalid_evidence") {
			t.Fatal("invalid authority accepted or exact Reason lost", err)
		}
	})
	t.Run("valid-edge-foreign-scope", func(t *testing.T) {
		// Matching glossary endpoints do not authorize a different package's
		// TU inventory. Build a genuine edge for the two added Units only.
		before, err := readCompatibilityFile(root, path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(path))); err != nil {
			t.Fatal(err)
		}
		defer writeSnapshotSource(t, root, path, before)
		foreign, foreignPath, err := AssessGlossaryCompatibility(root, "zh-CN", plan.Receipt.GlossarySHA256, addition)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(filepath.Join(root, filepath.FromSlash(foreignPath)))
		if foreign.OldGlossary != evidence.OldGlossary || foreign.NewGlossary != evidence.NewGlossary {
			t.Fatal("must keep identical glossary endpoints")
		}
		for _, context := range foreign.Contexts {
			if context.Scope == scope {
				t.Fatal("foreign edge unexpectedly contains historical TU")
			}
		}
		_, _, err = NewRepresentationSuccessorVerifier(root, current, "zh-CN").Verify(member.OldUnitID, member.CurrentUnitID)
		if err == nil || !strings.Contains(err.Error(), "reason=invalid_evidence") {
			t.Fatal("foreign authority was widened to historical package", err)
		}
	})
	after, err := readSnapshotReferencedFile(root, ref.Path)
	if err != nil || !bytes.Equal(after, receiptBytes) {
		t.Fatal("historical receipt mutated", err)
	}
	t.Logf("historical=49 current=51 old_to_new=verified_lineage chain=1 applicable_contexts=%d; new_members_not_inherited; true_stale_rejected", len(evidence.Contexts))
}
