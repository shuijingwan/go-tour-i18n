package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestQualityCheckCorrectCLIContract(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "reviewer-correction.json")
	if err := os.WriteFile(input, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	base := []string{"--locale", "zh-CN", "--snapshot-id", "qc", "--reviewer-session", "independent", "--correction-file", input, "--correction-sha256", courseContextSHA([]byte("{}"))}
	for _, extra := range [][]string{{"--apply"}, {"--record"}, {"--rating", "A"}, {"--unit-id", "u"}, {"extra"}, {"--correction-file", input}, {"-reviewer-session", "another"}} {
		if err := qualityCheckCorrectCommand(root, nil, append(append([]string(nil), base...), extra...), &bytes.Buffer{}); err == nil {
			t.Fatal("invalid contract accepted", extra)
		}
	}
	for i := 0; i < len(base); i += 2 {
		args := append(append([]string(nil), base[:i]...), base[i+2:]...)
		if err := qualityCheckCorrectCommand(root, nil, args, &bytes.Buffer{}); err == nil {
			t.Fatal("missing required flag", base[i])
		}
	}
	pkg, args, err := parseTranslationContentPackageArgs(append([]string{"quality-check", "correct", "--content-package", "learn-docs-v1"}, base...))
	if err != nil || pkg != "learn-docs-v1" || len(args) != len(base)+2 {
		t.Fatal(pkg, args, err)
	}
	doc, err := os.ReadFile("../../docs/TRANSLATION_QUALITY_REVIEW.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"quality-check correct", "correction-sha256", "COMMAND", "OPERATION", "INPUT_PATH_CONTRACT", "STDOUT", "MUTATION_EXPECTATION", "IDEMPOTENCE", "CURRENTNESS_CHECK", "PRIOR_OR_CAS", "SUCCESS_STOP", "FAILURE_STOP"} {
		if !strings.Contains(string(doc), field) {
			t.Error("canonical contract missing", field)
		}
	}
}

func TestQualityCheckCorrectRegisteredPackageCLI(t *testing.T) {
	root := authorizationDiagnosisMachineRoot(t)
	f := currentValidationExportFixture{root: root, locale: "en-GB", catalog: &i18n.Catalog{}}
	installCurrentValidationExportAuthority(t, f)
	text, id, path := "Read this message.\n", "correction/page", "fixtures/correction.txt"
	courseContextWrite(t, root, path, []byte(text))
	d := i18n.TranslationPackageDescriptor{ID: "correction-package", AuthoritySHA256: courseContextSHA([]byte("correction source authority")), Units: []i18n.AdaptedTranslationUnit{{Unit: i18n.TranslationUnit{ID: id, Kind: i18n.UnitKindPage, SourcePath: path, Source: []byte(text), SourceSHA256: courseContextSHA([]byte(text))}, Adapter: &authorizationDiagnosisAdapter{source: text}, StableIndex: 1, Route: "/correction"}}}
	if err := f.catalog.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if err := f.catalog.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	e, err := i18n.ExportRetranslationBatch(root, f.catalog, i18n.RetranslationExportOptions{Locale: f.locale, Generator: i18n.RetranslationGeneratorCodex, UnitIDs: []string{id}})
	if err != nil {
		t.Fatal(err)
	}
	var m i18n.RetranslationBatchManifest
	if err := json.Unmarshal(courseContextRead(t, root, e.BatchPath+"/manifest.json"), &m); err != nil {
		t.Fatal(err)
	}
	u := m.Units[0]
	courseContextWrite(t, root, e.BatchPath+"/raw-responses/"+filepath.Base(u.InputPath), courseContextRead(t, root, e.BatchPath+"/"+u.InputPath))
	if p, err := i18n.ProcessRetranslationBatch(root, f.catalog, i18n.RetranslationProcessOptions{Locale: f.locale, BatchID: e.BatchID}); err != nil || p.ValidationPassed != 1 {
		t.Fatal(p, err)
	}
	const snapshot, reviewer = "correction-cli", "independent-correction-reviewer"
	s, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, f.catalog, i18n.QualityCheckSnapshotOptions{Locale: f.locale, SnapshotID: snapshot, ContentPackage: d.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.RecordQualityCheckResults(root, f.catalog, i18n.QualityCheckRecordOptions{Locale: f.locale, SnapshotID: snapshot, ReviewerSession: reviewer, UnitIDs: []string{id}, Rating: "B", Finding: "Incorrect source literal"}); err != nil {
		t.Fatal(err)
	}
	resultPath := "data/quality-check-snapshots/" + f.locale + "/" + snapshot + "/quality-check-results.json"
	prior := courseContextRead(t, root, resultPath)
	no := false
	input := i18n.QualityCheckCorrectionEvidence{Locale: f.locale, SnapshotID: snapshot, OriginalResults: i18n.GlossaryArchiveReference{Path: resultPath, SHA256: courseContextSHA(prior)}, ReviewerSession: reviewer, UnitID: id, Index: s.Units[0].Index, OriginalRating: "B", OriginalFinding: "Incorrect source literal", CorrectedRating: "A", FindingDisposition: "withdrawn-source-mismatch", Reason: "Reviewer verified the exact canonical source", CorrectionRequired: true, ReplacementTextGenerated: &no, RepositoryMutation: "none"}
	b, _ := json.Marshal(input)
	file := filepath.Join(t.TempDir(), "correction.json")
	if err := os.WriteFile(file, b, 0600); err != nil {
		t.Fatal(err)
	}
	command := []string{"quality-check", "correct", "--content-package", d.ID, "--locale", f.locale, "--snapshot-id", snapshot, "--reviewer-session", reviewer, "--correction-file", file, "--correction-sha256", courseContextSHA(b)}
	pkg, args, err := parseTranslationContentPackageArgs(command)
	if err != nil || pkg != d.ID {
		t.Fatal(pkg, err)
	}
	missing := *f.catalog
	missing.TranslationScope = nil
	if err := qualityCheckCorrectCommand(root, &missing, args[2:], &bytes.Buffer{}); err == nil {
		t.Fatal("missing explicit registered package scope accepted")
	}
	bad := *f.catalog
	bad.TranslationPackages = append([]i18n.TranslationPackageDescriptor(nil), f.catalog.TranslationPackages...)
	bad.TranslationPackages[0].AuthoritySHA256 = courseContextSHA([]byte("wrong package"))
	if err := qualityCheckCorrectCommand(root, &bad, args[2:], &bytes.Buffer{}); err == nil {
		t.Fatal("wrong registered authority accepted")
	}
	var out bytes.Buffer
	if err := qualityCheckCorrectCommand(root, f.catalog, args[2:], &out); err != nil {
		t.Fatal(err)
	}
	var got i18n.QualityCheckCorrectionRecordResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || got.UnitID != id || got.Index != 1 || got.Path == "" || got.SHA256 != courseContextSHA(courseContextRead(t, root, got.Path)) {
		t.Fatal(out.String(), err)
	}
	if !bytes.Equal(prior, courseContextRead(t, root, resultPath)) {
		t.Fatal("original results changed")
	}
	if err := qualityCheckCorrectCommand(root, f.catalog, args[2:], &bytes.Buffer{}); err == nil {
		t.Fatal("duplicate invocation accepted")
	}
	if scope, err := i18n.BuildQualityCheckScope(root, f.catalog, i18n.QualityCheckScopeOptions{Locale: f.locale, SnapshotID: snapshot}); err != nil || scope.ACount != 1 || scope.PendingCount != 0 {
		t.Fatal(scope, err)
	}
}
