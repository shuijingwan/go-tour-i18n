package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestQualityCheckSurfaceReopenHistoricalCLIContract(t *testing.T) {
	root, c, snapshot := authorizationDiagnosisFixture(t, 2)
	if err := c.UseTranslationPackage("diagnostic-fixture"); err != nil {
		t.Fatal(err)
	}
	var manifest i18n.QualityCheckSnapshotManifest
	if err := json.Unmarshal(courseContextRead(t, root, "data/quality-check-snapshots/zh-CN/"+snapshot+"/manifest.json"), &manifest); err != nil {
		t.Fatal(err)
	}
	const review = "historical-cli-failed"
	const receipt = "data/locale-language/zh-CN/surface/" + review + ".json"
	const evidence = "data/locale-surface-reviews/zh-CN/" + review + ".md"
	for _, p := range []string{receipt, evidence} {
		courseContextWrite(t, root, p, []byte("synthetic immutable Surface authority\n"))
	}
	a := &i18n.SurfaceReviewAuthority{Locale: "zh-CN", ReviewID: review, Stage: "failed", ReceiptIdentity: "synthetic-receipt", ScopeIdentity: "synthetic-current-scope", Receipt: i18n.GlossaryArchiveReference{Path: receipt, SHA256: courseContextSHA(courseContextRead(t, root, receipt))}, Evidence: i18n.GlossaryArchiveReference{Path: evidence, SHA256: courseContextSHA(courseContextRead(t, root, evidence))}}
	base := []string{"--locale", "zh-CN", "--reopen-id", "cli-historical", "--previous-snapshot-id", snapshot, "--surface-review-id", review, "--integrated-surface"}
	for i, u := range manifest.Units {
		finding := "Correct reviewed exact defect " + u.UnitID
		a.Findings = append(a.Findings, i18n.SurfaceLanguageFinding{ID: u.SourcePath, Finding: finding, Class: i18n.SurfaceTranslationUnitLanguage, Artifacts: []i18n.SurfaceCandidateArtifact{{SnapshotID: snapshot, Unit: u}}})
		base = append(base, "--finding", finding, "--unit-id", manifest.Units[len(manifest.Units)-1-i].UnitID)
	}
	c.SurfaceAuthorityReader = func(_, _, _ string, current bool, _ *i18n.Catalog) (*i18n.SurfaceReviewAuthority, error) {
		if !current {
			t.Fatal("record did not check current Surface")
		}
		return a, nil
	}
	glossary := courseContextRead(t, root, "locales/zh-CN/glossary.yaml")
	courseContextWrite(t, root, "locales/zh-CN/glossary.yaml", append(glossary, []byte("preferred:\n  unused-cli-term: 无关术语\n")...))
	b, _, err := i18n.ExportUnifiedGlossaryBundle(root, "zh-CN", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.RecordUnifiedGlossaryReview(root, b, "cli-current-review", "synthetic-reviewer", "synthetic-generator", "passed", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.VerifyQualityCheckFinalization(root, c, "zh-CN", snapshot); err == nil {
		t.Fatal("ordinary verifier no longer rejects incompatible historical replay")
	}
	before := authorizationDiagnosisInventory(t, root)
	for _, classification := range []string{"affected", "stale", "unknown", "snapshot-file-path"} {
		t.Run(classification, func(t *testing.T) {
			adapter := c.Adapted[0].Adapter.(*authorizationDiagnosisAdapter)
			source := courseContextRead(t, root, c.Adapted[0].Unit.SourcePath)
			defer func() {
				adapter.validationFail, adapter.unknown = false, false
				courseContextWrite(t, root, c.Adapted[0].Unit.SourcePath, source)
			}()
			args := append([]string(nil), base...)
			switch classification {
			case "affected":
				adapter.validationFail = true
			case "stale":
				courseContextWrite(t, root, c.Adapted[0].Unit.SourcePath, []byte("changed source"))
			case "unknown":
				adapter.unknown = true
			case "snapshot-file-path":
				args[5] = "/tmp/finalization.json"
			}
			out, err := authorizationDiagnosisOutput(t, func() error { return qualityCheckSurfaceReopenCommand(root, c, args) })
			if err == nil || len(out) != 0 {
				t.Fatal("ineligible predecessor reported success", string(out), err)
			}
			if _, err := os.Stat(filepath.Join(root, "data/quality-check-reopens/zh-CN/cli-historical.json")); !os.IsNotExist(err) {
				t.Fatal("rejection published reopen", err)
			}
		})
	}
	out, err := authorizationDiagnosisOutput(t, func() error { return qualityCheckSurfaceReopenCommand(root, c, base) })
	if err != nil || !strings.Contains(string(out), "predecessor="+snapshot+" reopen_id=cli-historical units=2") {
		t.Fatal("direct CLI historical reopen", string(out), err)
	}
	after := authorizationDiagnosisInventory(t, root)
	created := filepath.Join(root, "data/quality-check-reopens/zh-CN/cli-historical.json")
	if len(after) != len(before)+1 || after[created] == "" {
		t.Fatal("CLI did not publish exactly one reopen")
	}
	delete(after, created)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("CLI modified prior authority")
	}
	for _, args := range [][]string{base, append(append([]string(nil), base...), "--unit-id", "wrong-unit")} {
		out, err := authorizationDiagnosisOutput(t, func() error { return qualityCheckSurfaceReopenCommand(root, c, args) })
		if err == nil || len(out) != 0 {
			t.Fatal("failure reported success", string(out), err)
		}
	}
}

func TestParseQualityCheckSurfaceReopenOptions(t *testing.T) {
	base := []string{"--locale", "zh-CN", "--reopen-id", "aggregate", "--previous-snapshot-id", "qc-009", "--surface-review-id", "failed"}
	t.Run("legacy-single", func(t *testing.T) {
		o, err := parseQualityCheckSurfaceReopenOptions(append(base, "--finding", "Exact finding.", "--unit-id", "page/1"))
		if err != nil || !reflect.DeepEqual(o.Findings, []string{"Exact finding."}) || o.IntegratedSurface {
			t.Fatal(o, err)
		}
	})
	t.Run("independently-repeated-flags", func(t *testing.T) {
		o, err := parseQualityCheckSurfaceReopenOptions(append(base, "--integrated-surface", "--unit-id", "page/2", "--finding", "First exact finding.", "--finding", "Second exact finding.", "--unit-id", "page/1"))
		if err != nil || !o.IntegratedSurface || !reflect.DeepEqual(o.Findings, []string{"First exact finding.", "Second exact finding."}) || !reflect.DeepEqual(o.UnitIDs, []string{"page/2", "page/1"}) {
			t.Fatal(o, err)
		}
	})
	for _, flags := range [][]string{{"--unit-id", "page/1"}, {"--finding", "Exact finding."}, {"--finding", "Exact finding.", "--unit-id", "page/1", "extra"}} {
		if _, err := parseQualityCheckSurfaceReopenOptions(append(base, flags...)); err == nil {
			t.Fatal("incomplete or extra argument accepted", flags)
		}
	}
	for _, flags := range [][]string{
		{"--previous-reopen-id", "old"},
		{"--historical-documents", "data/frozen.json"},
		{"--previous-reopen-id", "old", "--historical-documents", "data/frozen.json"},
		{"--integrated-surface", "--previous-reopen-id", "old", "--historical-documents", "/tmp/frozen.json"},
		{"--integrated-surface", "--previous-reopen-id", "old", "--historical-documents", "../frozen.json"},
	} {
		args := append(append([]string(nil), base...), "--finding", "Exact finding.", "--unit-id", "page/1")
		if _, err := parseQualityCheckSurfaceReopenOptions(append(args, flags...)); err == nil {
			t.Fatal("unsafe/incomplete replacement flags accepted", flags)
		}
	}
}

func TestParsePreviewFailureReopenOptions(t *testing.T) {
	f, err := os.CreateTemp(".", ".preview-authority-test-*")
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(f.Name())
	defer os.Remove(name)
	if _, err := f.WriteString("synthetic immutable receipt bytes\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	base := []string{"--locale", "fr-FR", "--reopen-id", "preview-fix", "--previous-snapshot-id", "qc-001", "--unit-id", "unit/1", "--preview-failure", name}
	o, err := parseQualityCheckSurfaceReopenOptions(base)
	if err != nil || o.PreviewFailure.Path != name || o.PreviewFailure.SHA256 == "" || o.SurfaceReviewID != "" {
		t.Fatal(o, err)
	}
	for _, extra := range [][]string{
		{"--finding", "pretend finding"}, {"--surface-review-id", "old-pass"}, {"--integrated-surface"},
		{"--historical-documents", "frozen.json"}, {"--previous-reopen-id", "old"}, {"--unit-id", "unit/2"}, {"extra"},
	} {
		if _, err := parseQualityCheckSurfaceReopenOptions(append(append([]string{}, base...), extra...)); err == nil {
			t.Fatal("mixed/inexact preview authority flags accepted", extra)
		}
	}
	for _, unsafe := range []string{"../receipt.json", "/tmp/receipt.json"} {
		args := append([]string{}, base...)
		args[len(args)-1] = unsafe
		if _, err := parseQualityCheckSurfaceReopenOptions(args); err == nil {
			t.Fatal("unsafe path", unsafe)
		}
	}
}
