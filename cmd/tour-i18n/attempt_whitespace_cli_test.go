package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestProtectedWhitespaceRetryCLIContract(t *testing.T) {
	doc := string(courseContextRead(t, "../..", "docs/RETRANSLATION_RUNBOOK.md"))
	for _, s := range []string{"--protected-whitespace-repair", "protected-response representation cannot express this validation repair", "go-tour-i18n/protected-structural-whitespace-attempt/v1", "Formal Command Contract v3"} {
		if !strings.Contains(doc, s) {
			t.Fatal("missing canonical whitespace command contract", s)
		}
	}
	root := t.TempDir()
	output := filepath.Join(root, "attempt.zip")
	base := []string{"--locale", "zh-CN", "--batch-id", "chatgpt-zh-CN-001", "--output", output}
	for _, extra := range [][]string{{"--protected-whitespace-repair"}, {"--unit-id", "u", "--unit-id", "v", "--protected-whitespace-repair"}, {"--unit-id", "u", "--protected-whitespace-repair=invalid"}, {"--unit-id", "u", "--protected-whitespace-repair", "true"}, {"--unit-id", "u", "--protected-whitespace-repair", "--protected-whitespace-repair=false"}, {"--unit-id", "u", "--protected-whitespace-repair", "--approved-interval-selection", "x"}} {
		if err := exportGenerationBundleCommand(root, nil, append(append([]string(nil), base...), extra...), "learn-docs-v1"); err == nil {
			t.Fatal("invalid contract accepted", extra)
		}
		if _, err := os.Lstat(output); !os.IsNotExist(err) {
			t.Fatal("failed contract published ZIP", err)
		}
	}
	metadata := i18n.Locale{Locale: "zh-CN", HTMLLang: "zh-CN", LanguageName: "fixture", EnglishName: "fixture", Phase: "scaffold", TranslationUnit: "present.Section", LanguageAudit: testLocaleInitLanguagePolicy("zh-CN")}
	courseContextWrite(t, root, "locales/zh-CN/locale.json", courseContextJSON(t, metadata))
	if err := exportGenerationBundleCommand(root, nil, append(base, "--unit-id", "u", "--protected-whitespace-repair"), "learn-docs-v1"); err == nil || !strings.Contains(err.Error(), "catalog is required") {
		t.Fatal("flag did not reach canonical exporter", err)
	}
	if err := checkTranslationUnitGenerationBundleCommand(root, nil, []string{"--bundle", "x", "--protected-whitespace-repair"}, "learn-docs-v1"); err == nil {
		t.Fatal("current-check accepted representation override")
	}
	// Existing retry + explicit mode on the wrong failure class stops before publication.
	f := makeNativeGenerationFixture(t, true)
	var formal i18n.RetranslationBatchManifest
	if err := json.Unmarshal(courseContextRead(t, f.f.root, f.batch.BatchPath+"/manifest.json"), &formal); err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(formal.Units[0].InputPath)
	courseContextWrite(t, f.f.root, f.batch.BatchPath+"/raw-responses/"+name, []byte("broken response\n"))
	if r, err := i18n.ProcessRetranslationBatch(f.f.root, f.f.catalog, i18n.RetranslationProcessOptions{Locale: f.f.locale, BatchID: f.batch.BatchID}); err != nil || r.RestoreFailed+r.ValidationFailed != 1 {
		t.Fatal(r, err)
	}
	rejected := filepath.Join(t.TempDir(), "wrong-class.zip")
	args := append(generationExportArgs(f, rejected), "--unit-id", formal.Units[0].UnitID, "--protected-whitespace-repair")
	if _, err := invokeGenerationCLI(t, f.f.root, f.f.catalog, args); err == nil || !strings.Contains(err.Error(), "exact protected structural-whitespace failure") {
		t.Fatal("wrong failure class gained codec", err)
	}
	if _, err := os.Lstat(rejected); !os.IsNotExist(err) {
		t.Fatal("failure published ZIP")
	}
}
