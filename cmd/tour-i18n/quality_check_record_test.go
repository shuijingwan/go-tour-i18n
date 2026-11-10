package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestQualityCheckRecordJudgmentsCLIParsing(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "judgments.json")
	if err := os.WriteFile(path, []byte(`[{"unit_id":"u","rating":"A"}]`), 0600); err != nil {
		t.Fatal(err)
	}
	base := []string{"--locale", "zh-CN", "--snapshot-id", "new-qc", "--reviewer-session", "independent-reviewer", "--judgments-file", path}
	for _, extra := range [][]string{{"--unit-id", "u"}, {"--rating", "A"}, {"--finding", ""}, {"--judgments-file", path}, {"unexpected"}, {"--ignore"}, {"--apply"}} {
		if err := qualityCheckRecordCommand(root, nil, append(append([]string{}, base...), extra...), &bytes.Buffer{}); err == nil {
			t.Fatal("invalid CLI accepted", extra)
		}
	}
	for _, input := range []string{`{}`, `[{"unit_id":"u","rating":"A","index":1}]`, `[{"unit_id":"u","rating":"A","rating":"B"}]`, `[] true`, `not JSON`, `null`, `[]`} {
		if err := os.WriteFile(path, []byte(input), 0600); err != nil {
			t.Fatal(err)
		}
		err := qualityCheckRecordCommand(root, nil, base, &bytes.Buffer{})
		if err == nil {
			t.Fatal("invalid document accepted", input)
		}
		if input != "null" && input != "[]" && !strings.Contains(err.Error(), "quality-check judgments input") {
			t.Fatal("document was not rejected by the strict transport reader", input, err)
		}
	}
	if err := qualityCheckRecordCommand(root, nil, []string{"--locale", "zh-CN", "--snapshot-id", "new-qc", "--judgments-file", path}, &bytes.Buffer{}); err == nil {
		t.Fatal("missing Reviewer session accepted")
	}
	if err := qualityCheckRecordCommand(root, nil, []string{"--locale", "zh-CN", "--snapshot-id", "new-qc", "--judgments-file", ""}, &bytes.Buffer{}); err == nil {
		t.Fatal("empty file flag accepted")
	}
}

func TestQualityCheckRecordJudgmentsPackageCLI(t *testing.T) {
	root := authorizationDiagnosisMachineRoot(t)
	f := currentValidationExportFixture{root: root, locale: "en-GB", catalog: &i18n.Catalog{}}
	installCurrentValidationExportAuthority(t, f)
	d := i18n.TranslationPackageDescriptor{ID: "record-registered", AuthoritySHA256: courseContextSHA([]byte("registered QC source"))}
	for i, text := range []string{"Read the first message.\n", "Read the second message.\n"} {
		id := []string{"record/first", "record/second"}[i]
		path := []string{"fixtures/first.txt", "fixtures/second.txt"}[i]
		courseContextWrite(t, root, path, []byte(text))
		d.Units = append(d.Units, i18n.AdaptedTranslationUnit{Unit: i18n.TranslationUnit{ID: id, Kind: i18n.UnitKindPage, SourcePath: path, Source: []byte(text), SourceSHA256: courseContextSHA([]byte(text))}, Adapter: &authorizationDiagnosisAdapter{source: text}, StableIndex: i + 1, Route: "/" + id})
	}
	if err := f.catalog.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if err := f.catalog.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	ids := append([]string{}, f.catalog.TranslationScope...)
	exported, err := i18n.ExportRetranslationBatch(root, f.catalog, i18n.RetranslationExportOptions{Locale: f.locale, Generator: i18n.RetranslationGeneratorCodex, UnitIDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	var m i18n.RetranslationBatchManifest
	if err := json.Unmarshal(courseContextRead(t, root, exported.BatchPath+"/manifest.json"), &m); err != nil {
		t.Fatal(err)
	}
	for _, u := range m.Units {
		courseContextWrite(t, root, exported.BatchPath+"/raw-responses/"+filepath.Base(u.InputPath), courseContextRead(t, root, exported.BatchPath+"/"+u.InputPath))
	}
	if p, err := i18n.ProcessRetranslationBatch(root, f.catalog, i18n.RetranslationProcessOptions{Locale: f.locale, BatchID: exported.BatchID}); err != nil || p.ValidationPassed != 2 {
		t.Fatal(p, err)
	}
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, f.catalog, i18n.QualityCheckSnapshotOptions{Locale: f.locale, SnapshotID: "bulk-cli", ContentPackage: d.ID}); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(t.TempDir(), "judgments.json")
	judgments := []i18n.QualityCheckJudgment{{UnitID: ids[1], Rating: "B", Finding: "Exact second Unit finding"}, {UnitID: ids[0], Rating: "A"}}
	b, err := json.Marshal(judgments)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, b, 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"quality-check", "record", "--locale", f.locale, "--snapshot-id", "bulk-cli", "--content-package", d.ID, "--reviewer-session", "independent-cli-reviewer", "--judgments-file", input}
	pkg, parsed, err := parseTranslationContentPackageArgs(args)
	if err != nil || pkg != d.ID {
		t.Fatal("generic package dispatch lost bulk mode", pkg, err)
	}
	scoped := *f.catalog
	scoped.TranslationScope = nil
	if err := scoped.UseTranslationPackage(pkg); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := qualityCheckRecordCommand(root, &scoped, parsed[2:], &out); err != nil {
		t.Fatal(err)
	}
	var result i18n.QualityCheckRecordResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil || result.RecordedCount != 2 || result.Rating != "" {
		t.Fatal(out.String(), err)
	}
	var results i18n.QualityCheckResults
	canonical := courseContextRead(t, root, result.Path)
	if err := json.Unmarshal(canonical, &results); err != nil {
		t.Fatal(err)
	}
	if results.Results[0].UnitID != ids[0] || results.Results[1].Finding != judgments[0].Finding {
		t.Fatal(results)
	}
	out.Reset()
	if err := qualityCheckRecordCommand(root, &scoped, parsed[2:], &out); err == nil {
		t.Fatal("record replay overwrote results")
	}
	if !reflect.DeepEqual(canonical, courseContextRead(t, root, result.Path)) {
		t.Fatal("failed replay partially wrote results")
	}
}
