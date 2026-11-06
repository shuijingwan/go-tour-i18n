package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func TestRetranslationSurfaceComposedExportCLIContract(t *testing.T) {
	root, c, _ := authorizationDiagnosisFixture(t, 3)
	full := c.TranslationPackages[0]
	old, sibling := full, full
	old.AuthoritySHA256, old.Units = courseContextSHA([]byte("old-complete-two")), append([]i18n.AdaptedTranslationUnit(nil), full.Units[:2]...)
	sibling.ID, sibling.AuthoritySHA256, sibling.Units = "diagnostic-sibling", courseContextSHA([]byte("sibling-one")), append([]i18n.AdaptedTranslationUnit(nil), full.Units[2:]...)
	c.HistoricalTranslationPackages = []i18n.TranslationPackageDescriptor{old, sibling}
	if err := c.UseTranslationPackage(full.ID); err != nil {
		t.Fatal(err)
	}
	inputs := []i18n.TranslationFinalizationInput{}
	var predecessor, siblingFinal *i18n.QualityCheckFinalization
	for i, d := range []i18n.TranslationPackageDescriptor{old, sibling} {
		h, err := i18n.HistoricalTranslationPackageCatalog(c, d.ID, d.AuthoritySHA256)
		if err != nil {
			t.Fatal(err)
		}
		snapshot := fmt.Sprintf("cli-predecessor-%d", i)
		if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, h, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: snapshot}); err != nil {
			t.Fatal(err)
		}
		if _, err := i18n.RecordQualityCheckResults(root, h, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: snapshot, UnitIDs: h.TranslationScope, Rating: "A", ReviewerSession: "synthetic-independent-reviewer"}); err != nil {
			t.Fatal(err)
		}
		f, path, err := i18n.FinalizeQualityCheck(root, h, i18n.QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: snapshot})
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			predecessor = f
		} else {
			siblingFinal = f
		}
		inputs = append(inputs, i18n.TranslationFinalizationInput{Package: d.ID, PackageAuthority: d.AuthoritySHA256, Finalization: i18n.GlossaryArchiveReference{Path: path, SHA256: courseContextSHA(courseContextRead(t, root, path))}})
	}
	const review = "cli-composed-failed"
	receipt := "data/locale-language/zh-CN/surface/" + review + ".json"
	markdown := "data/locale-surface-reviews/zh-CN/" + review + ".md"
	for _, path := range []string{receipt, markdown} {
		courseContextWrite(t, root, path, []byte("synthetic immutable failed Surface evidence\n"))
	}
	u := predecessor.Units[0].Snapshot
	finding := "Correct this exact reviewed artifact."
	a := &i18n.SurfaceReviewAuthority{Locale: "zh-CN", ReviewID: review, Stage: "failed", ReceiptIdentity: "synthetic-receipt", ScopeIdentity: "synthetic-complete-composed-scope", Receipt: i18n.GlossaryArchiveReference{Path: receipt, SHA256: courseContextSHA(courseContextRead(t, root, receipt))}, Evidence: i18n.GlossaryArchiveReference{Path: markdown, SHA256: courseContextSHA(courseContextRead(t, root, markdown))}, Findings: []i18n.SurfaceLanguageFinding{{ID: u.SourcePath, Finding: finding, Class: i18n.SurfaceTranslationUnitLanguage, Artifacts: []i18n.SurfaceCandidateArtifact{{SnapshotID: predecessor.SnapshotID, Unit: u}}}}}
	siblingUnit := siblingFinal.Units[0].Snapshot
	const siblingFinding = "Correct the sibling historical artifact."
	a.Findings = append(a.Findings, i18n.SurfaceLanguageFinding{ID: siblingUnit.SourcePath, Finding: siblingFinding, Class: i18n.SurfaceTranslationUnitLanguage, Artifacts: []i18n.SurfaceCandidateArtifact{{SnapshotID: siblingFinal.SnapshotID, Unit: siblingUnit}}})
	closure := sitecontent.PackageClosure{Schema: sitecontent.ClosureSchema, Locale: "zh-CN", Package: full.ID, PackageSHA: full.AuthoritySHA256, PageComposition: inputs}
	encoded, _ := json.Marshal(closure)
	closure.Identity = courseContextSHA(encoded)
	closurePath := "data/site-workflow/zh-CN/" + full.ID + "/closure/" + closure.Identity + ".json"
	courseContextWrite(t, root, closurePath, courseContextJSON(t, closure))
	a.Context = []i18n.GlossaryArchiveReference{{Path: closurePath, SHA256: courseContextSHA(courseContextRead(t, root, closurePath))}}
	// This fixture's current Surface authority replays the actual shared whole
	// composition; the sitecontent regression covers formal membership routing.
	c.SurfaceAuthorityReader = func(root, locale, _ string, current bool, _ *i18n.Catalog) (*i18n.SurfaceReviewAuthority, error) {
		if current {
			if _, _, err := i18n.ComposeFinalizedTranslationPackage(root, c, locale, inputs); err != nil {
				return nil, err
			}
		}
		return a, nil
	}
	const reopenID = "cli-composed-reopen"
	if _, _, err := i18n.RecordQualityCheckSurfaceReopen(root, c, i18n.QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: reopenID, PreviousSnapshotID: predecessor.SnapshotID, SurfaceReviewID: review, Finding: finding, UnitIDs: []string{u.UnitID}, IntegratedSurface: true}); err != nil {
		t.Fatal(err)
	}
	args := []string{"retranslation", "export", "--locale", "zh-CN", "--content-package", full.ID, "--generator", "chatgpt", "--batch-id", "chatgpt-zh-CN-060", "--unit-kind", "page", "--previous-snapshot-id", predecessor.SnapshotID, "--surface-reopen-id", reopenID, "--allow-reexport", "--id", u.UnitID, "--json"}
	pkg, parsed, err := parseTranslationContentPackageArgs(args)
	if err != nil || pkg != full.ID {
		t.Fatal(pkg, err)
	}
	before := authorizationDiagnosisInventory(t, root)
	wrong := append([]string(nil), parsed[2:]...)
	for i := range wrong {
		if wrong[i] == predecessor.SnapshotID {
			wrong[i] = "data/quality-check-snapshots/old/finalization.json"
		}
	}
	out, err := authorizationDiagnosisOutput(t, func() error { return retranslationExportCommand(root, c, wrong) })
	if err == nil || len(out) != 0 || !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, root)) {
		t.Fatal("unsafe Snapshot path accepted/published", string(out), err)
	}
	out, err = authorizationDiagnosisOutput(t, func() error { return retranslationExportCommand(root, c, parsed[2:]) })
	var result i18n.RetranslationExportResult
	if err != nil || json.Unmarshal(out, &result) != nil || result.UnitCount != 1 || !reflect.DeepEqual(result.UnitIDs, []string{u.UnitID}) || len(c.TranslationScope) != 3 {
		t.Fatal("direct CLI composed export", string(out), err)
	}
	var manifest i18n.RetranslationBatchManifest
	if err := json.Unmarshal(courseContextRead(t, root, result.BatchPath+"/manifest.json"), &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Units) != 1 || manifest.Units[0].PreviousSnapshotID != predecessor.SnapshotID || manifest.Units[0].RevisionAuthorizationID != reopenID || manifest.Units[0].RevisionFeedbackSource != "surface_review" {
		t.Fatal("manifest lost exact reopen provenance", manifest)
	}
	after := authorizationDiagnosisInventory(t, root)
	for path := range after {
		if _, ok := before[path]; !ok && strings.HasPrefix(path, filepath.Join(root, result.BatchPath)+string(filepath.Separator)) {
			delete(after, path)
		}
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("export changed preexisting formal authority")
	}
	for _, batch := range []string{"chatgpt-zh-CN-060", "chatgpt-zh-CN-061"} {
		repeated := append([]string(nil), parsed[2:]...)
		for i := range repeated {
			if repeated[i] == "chatgpt-zh-CN-060" {
				repeated[i] = batch
			}
		}
		out, err := authorizationDiagnosisOutput(t, func() error { return retranslationExportCommand(root, c, repeated) })
		if err == nil || len(out) != 0 {
			t.Fatal("consumed reopen reported success", string(out), err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "data/retranslation-runs/zh-CN/chatgpt-zh-CN-061")); !os.IsNotExist(err) {
		t.Fatal("consumed reopen published sibling export", err)
	}
	// The sibling's historical package name differs from the explicit current
	// owner. It still needs the already checked current owner's closure and the
	// exact subset relation; an unrelated owner cannot be inferred from IDs.
	siblingReopenArgs := []string{"--locale", "zh-CN", "--reopen-id", "cli-sibling-reopen", "--previous-snapshot-id", siblingFinal.SnapshotID, "--surface-review-id", review, "--finding", siblingFinding, "--unit-id", siblingUnit.UnitID, "--integrated-surface"}
	context := a.Context
	a.Context = nil
	out, err = authorizationDiagnosisOutput(t, func() error { return qualityCheckSurfaceReopenCommand(root, c, siblingReopenArgs) })
	if err == nil || len(out) != 0 {
		t.Fatal("cross-package predecessor accepted without current owner closure", string(out), err)
	}
	a.Context = context
	out, err = authorizationDiagnosisOutput(t, func() error { return qualityCheckSurfaceReopenCommand(root, c, siblingReopenArgs) })
	if err != nil || !strings.Contains(string(out), "predecessor="+siblingFinal.SnapshotID+" reopen_id=cli-sibling-reopen units=1") {
		t.Fatal("direct CLI sibling reopen", string(out), err)
	}
	siblingExportArgs := append([]string(nil), parsed[2:]...)
	for i := range siblingExportArgs {
		switch siblingExportArgs[i] {
		case predecessor.SnapshotID:
			siblingExportArgs[i] = siblingFinal.SnapshotID
		case reopenID:
			siblingExportArgs[i] = "cli-sibling-reopen"
		case u.UnitID:
			siblingExportArgs[i] = siblingUnit.UnitID
		case "chatgpt-zh-CN-060":
			siblingExportArgs[i] = "chatgpt-zh-CN-061"
		}
	}
	out, err = authorizationDiagnosisOutput(t, func() error { return retranslationExportCommand(root, c, siblingExportArgs) })
	if err != nil || json.Unmarshal(out, &result) != nil || result.UnitCount != 1 || !reflect.DeepEqual(result.UnitIDs, []string{siblingUnit.UnitID}) || len(c.TranslationScope) != 3 {
		t.Fatal("direct CLI sibling export", string(out), err)
	}
}
