package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeRetranslationProcessBatch(t *testing.T, count int) (string, *Catalog, string) {
	return makeRetranslationProcessBatchForLocale(t, "zh-CN", count)
}

func makeRetranslationProcessBatchForLocale(t *testing.T, locale string, count int) (string, *Catalog, string) {
	t.Helper()
	root := t.TempDir()
	writeRetranslationTestGlossaryForLocale(t, root, locale)
	catalog := retranslationTestCatalog(count)
	result, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: locale, Limit: count})
	if err != nil {
		t.Fatal(err)
	}
	batchDir := filepath.Join(root, result.BatchPath)
	if err := os.Mkdir(filepath.Join(batchDir, "raw-responses"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := readRetranslationManifestForLocale(t, root, locale, result.BatchID)
	for _, page := range manifest.Units {
		input, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(page.InputPath)))
		if err != nil {
			t.Fatal(err)
		}
		raw := strings.ReplaceAll(string(input), "* Page", "* 页面")
		raw = strings.ReplaceAll(raw, "Use ", "在此页面使用 ")
		raw = strings.ReplaceAll(raw, " on this page.", "。")
		if err := os.WriteFile(filepath.Join(batchDir, "raw-responses", retranslationUnitInputName(&TranslationUnit{ID: page.UnitID, Kind: page.UnitKind})), []byte(raw), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root, catalog, result.BatchID
}

func TestRetranslationProcessRejectsManifestLocaleMismatch(t *testing.T) {
	const locale = "ja-JP"
	root, catalog, batchID := makeRetranslationProcessBatchForLocale(t, locale, 1)
	manifest := readRetranslationManifestForLocale(t, root, locale, batchID)
	manifest.Locale = "zh-CN"
	if err := writeTranslationJSON(filepath.Join(root, "data", "retranslation-runs", locale, batchID, "manifest.json"), manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: locale, BatchID: batchID}); err == nil || !strings.Contains(err.Error(), "incompatible manifest metadata") {
		t.Fatalf("process error = %v, want locale mismatch", err)
	}
}

func TestDecodeLegacyRetranslationValidationDerivesAttemptFromProvenance(t *testing.T) {
	unit := &TranslationUnit{ID: "moretypes/1", Kind: UnitKindPage, SourceSHA256: strings.Repeat("a", 64)}
	tests := []struct {
		name    string
		rawPath string
		want    int
		wantErr string
	}{
		{name: "initial", rawPath: "raw-responses/moretypes-1.article", want: 1},
		{name: "retry", rawPath: "retries/moretypes-1/attempt-002.article", want: 2},
		{name: "invalid", rawPath: "retries/other/attempt-002.article", wantErr: "not a recognized attempt"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(legacyRetranslationValidation{
				SchemaVersion: 1, BatchID: "chatgpt-zh-CN-004", Locale: "zh-CN", PageID: unit.ID,
				Status: "passed", InputPath: "inputs/moretypes-1.article", RawResponsePath: test.rawPath,
				CandidatePath: "candidates/moretypes-1.article",
			})
			if err != nil {
				t.Fatal(err)
			}
			validation, err := decodeRetranslationValidation(data, unit)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want %q", err, test.wantErr)
				}
				return
			}
			if err != nil || validation.Attempt != test.want {
				t.Fatalf("validation=%+v error=%v, want attempt=%d", validation, err, test.want)
			}
		})
	}
}

func TestDecodeSchemaV2RetranslationValidationBehaviorUnchanged(t *testing.T) {
	unit := &TranslationUnit{ID: "moretypes/1", Kind: UnitKindPage}
	want := RetranslationValidation{
		SchemaVersion: retranslationProcessSchemaVersion, UnitID: unit.ID, UnitKind: unit.Kind,
		Attempt: 7, RawResponsePath: "preserved-by-v2-decoder",
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeRetranslationValidation(data, unit)
	if err != nil || got.Attempt != want.Attempt || got.RawResponsePath != want.RawResponsePath {
		t.Fatalf("validation=%+v error=%v", got, err)
	}
}

func rewriteProcessManifest(t *testing.T, root, batchID string, mutate func(*RetranslationBatchManifest)) {
	t.Helper()
	manifest := readRetranslationManifest(t, root, batchID)
	mutate(&manifest)
	if err := writeTranslationJSON(filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "manifest.json"), manifest); err != nil {
		t.Fatal(err)
	}
}

func makeRetranslationExampleProcessBatch(t *testing.T) (string, *Catalog, string, string) {
	t.Helper()
	root := t.TempDir()
	writeGoExampleValidationGlossary(t, root)
	source := []byte("package main\n\n// A goroutine sends the value through the channel.\nfunc main() { println(1) }\n")
	example := Example{
		ID: "example:basics/channel.go", SourcePath: "_content/tour/basics/channel.go",
		Source: source, SourceSHA256: sum(source),
	}
	catalog := &Catalog{Examples: []Example{example}}
	result, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{
		Locale: "zh-CN", UnitKind: UnitKindExample, UnitIDs: []string{example.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	batchDir := filepath.Join(root, result.BatchPath)
	if err := os.Mkdir(filepath.Join(batchDir, "raw-responses"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := readRetranslationManifest(t, root, result.BatchID)
	input, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(manifest.Units[0].InputPath)))
	if err != nil {
		t.Fatal(err)
	}
	raw := strings.Replace(string(input), "A ", "一个 ", 1)
	raw = strings.Replace(raw, " sends the value through the channel.", " 通过通道发送该值。", 1)
	name := filepath.Base(filepath.FromSlash(manifest.Units[0].InputPath))
	if err := os.WriteFile(filepath.Join(batchDir, "raw-responses", name), []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	return root, catalog, result.BatchID, name
}

func TestRetranslationProcessExampleRestoresValidatesAndPreservesFormalData(t *testing.T) {
	root, catalog, batchID, name := makeRetranslationExampleProcessBatch(t)
	statusPath := filepath.Join(root, "locales", "zh-CN", "status.tsv")
	canonicalPath := filepath.Join(root, "locales", "zh-CN", "candidates", "sentinel.go")
	if err := os.WriteFile(statusPath, []byte("status sentinel\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(canonicalPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(canonicalPath, []byte("candidate sentinel\n"), 0644); err != nil {
		t.Fatal(err)
	}
	result, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batchID})
	if err != nil {
		t.Fatal(err)
	}
	if result.RestorePassed != 1 || result.ValidationPassed != 1 || result.Units[0].UnitKind != UnitKindExample || result.Units[0].UnitID != catalog.Examples[0].ID {
		t.Fatalf("example process result=%+v", result)
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	candidateName := retranslationUnitCandidateName(&TranslationUnit{ID: catalog.Examples[0].ID, Kind: UnitKindExample})
	candidate, err := os.ReadFile(filepath.Join(batchDir, "candidates", candidateName))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(name) != ".txt" || filepath.Ext(candidateName) != ".go" || !strings.Contains(string(candidate), "// 一个 goroutine 通过通道发送该值。") {
		t.Fatalf("example input=%s candidate %s=%q", name, candidateName, candidate)
	}
	validationName := strings.TrimSuffix(name, filepath.Ext(name)) + ".json"
	var evidence RetranslationValidation
	data, err := os.ReadFile(filepath.Join(batchDir, "validation", validationName))
	if err != nil || json.Unmarshal(data, &evidence) != nil {
		t.Fatalf("validation evidence=%s err=%v", data, err)
	}
	if evidence.Status != "passed" || evidence.UnitID != catalog.Examples[0].ID || evidence.UnitKind != UnitKindExample || evidence.SourceSHA256 != catalog.Examples[0].SourceSHA256 || evidence.RawResponsePath != filepath.ToSlash(filepath.Join("raw-responses", name)) {
		t.Fatalf("example validation evidence=%+v", evidence)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for label, encoded := range map[string][]byte{"result": resultJSON, "validation": data} {
		for _, obsolete := range []string{`"page_id"`, `"page_count"`, `"pages"`} {
			if bytes.Contains(encoded, []byte(obsolete)) {
				t.Fatalf("new %s evidence contains obsolete field %s: %s", label, obsolete, encoded)
			}
		}
	}
	for path, want := range map[string]string{statusPath: "status sentinel\n", canonicalPath: "candidate sentinel\n"} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Fatalf("formal data %s changed: %q err=%v", path, got, err)
		}
	}
}

func TestRetranslationProcessExamplePreflightRejectsUnsafeBatch(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, string, *Catalog, string, string)
		want   string
	}{
		{"source hash", func(t *testing.T, root string, _ *Catalog, batch, _ string) {
			rewriteProcessManifest(t, root, batch, func(m *RetranslationBatchManifest) { m.Units[0].SourceSHA256 = strings.Repeat("0", 64) })
		}, "source metadata"},
		{"input hash", func(t *testing.T, root string, _ *Catalog, batch, _ string) {
			rewriteProcessManifest(t, root, batch, func(m *RetranslationBatchManifest) { m.Units[0].InputSHA256 = strings.Repeat("0", 64) })
		}, "input_sha256 mismatch"},
		{"saved input differs", func(t *testing.T, root string, _ *Catalog, batch, name string) {
			path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batch, "inputs", name)
			data, _ := os.ReadFile(path)
			data = append(data, 'x')
			if err := os.WriteFile(path, data, 0644); err != nil {
				t.Fatal(err)
			}
			rewriteProcessManifest(t, root, batch, func(m *RetranslationBatchManifest) { m.Units[0].InputSHA256 = sum(data) })
		}, "regenerated protected input differs"},
		{"unit kind", func(t *testing.T, root string, _ *Catalog, batch, _ string) {
			rewriteProcessManifest(t, root, batch, func(m *RetranslationBatchManifest) {
				m.UnitKind = UnitKindPage
				m.Units[0].UnitKind = UnitKindPage
			})
		}, "unit_kind"},
		{"mixed kind", func(t *testing.T, root string, catalog *Catalog, batch, name string) {
			pageSource := []byte("* Page\n")
			catalog.Pages = append(catalog.Pages, Page{ID: "lesson/1", Article: "lesson.article", SectionNumber: 1, Route: "/lesson/1", Source: pageSource, SourceSHA256: sum(pageSource)})
			rewriteProcessManifest(t, root, batch, func(m *RetranslationBatchManifest) {
				m.UnitCount++
				m.Units = append(m.Units, RetranslationBatchUnit{UnitID: "lesson/1", UnitKind: UnitKindPage, SourcePath: "_content/tour/lesson.article", SourceSHA256: sum(pageSource), InputPath: "inputs/lesson-1.article"})
			})
			_ = name
		}, "unit_kind"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, catalog, batchID, name := makeRetranslationExampleProcessBatch(t)
			tt.mutate(t, root, catalog, batchID, name)
			_, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batchID})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error=%v, want %q", err, tt.want)
			}
			batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
			for _, output := range []string{"candidates", "validation", "result.json"} {
				if _, statErr := os.Stat(filepath.Join(batchDir, output)); !os.IsNotExist(statErr) {
					t.Fatalf("unsafe output %s exists: %v", output, statErr)
				}
			}
		})
	}
}

func TestRetranslationProcessExampleRecordsRestoreAndValidationFailures(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(string) string
		status string
		want   string
	}{
		{"token deleted", func(raw string) string { return strings.Replace(raw, translationTokenRE.FindString(raw), "", 1) }, "restore_failed", "occurrence count"},
		{"token modified", func(raw string) string {
			return strings.Replace(raw, translationTokenRE.FindString(raw), "⟪GTI18N_deadbeef_999999⟫", 1)
		}, "restore_failed", "unknown protected token"},
		{"token duplicated", func(raw string) string { return raw + translationTokenRE.FindString(raw) }, "restore_failed", "occurrence count"},
		{"token reordered", func(raw string) string {
			tokens := translationTokenRE.FindAllString(raw, -1)
			if len(tokens) < 2 {
				return raw
			}
			raw = strings.Replace(raw, tokens[0], "SWAP_TOKEN", 1)
			raw = strings.Replace(raw, tokens[1], tokens[0], 1)
			return strings.Replace(raw, "SWAP_TOKEN", tokens[1], 1)
		}, "restore_failed", "order changed"},
		{"forbidden", func(raw string) string {
			return strings.Replace(raw, "通过通道发送该值。", "使用了幻灯片。", 1)
		}, "validation_failed", "禁止译法"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, catalog, batchID, name := makeRetranslationExampleProcessBatch(t)
			rawPath := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "raw-responses", name)
			raw, _ := os.ReadFile(rawPath)
			if err := os.WriteFile(rawPath, []byte(tt.mutate(string(raw))), 0644); err != nil {
				t.Fatal(err)
			}
			result, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batchID})
			if err != nil {
				t.Fatal(err)
			}
			if result.Units[0].Status != tt.status {
				t.Fatalf("result=%+v", result)
			}
			validationName := strings.TrimSuffix(name, filepath.Ext(name)) + ".json"
			data, _ := os.ReadFile(filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "validation", validationName))
			if !strings.Contains(string(data), tt.want) {
				t.Fatalf("validation=%s, want %q", data, tt.want)
			}
		})
	}
}

func TestRetranslationProcessEquivalentExampleCorpusBaseline(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	glossary, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	processed := 0
	for i := range catalog.Examples {
		unit, err := catalog.Unit(catalog.Examples[i].ID)
		if err != nil {
			t.Fatal(err)
		}
		hasContent, err := hasTranslatableGoExampleComment(unit.Source)
		if err != nil {
			t.Fatal(err)
		}
		if !hasContent {
			continue
		}
		protected, err := prepareTranslationUnitInput(unit, glossary)
		if err != nil {
			t.Fatal(err)
		}
		restored, failures := protected.restore(protected.Text)
		if len(failures) != 0 {
			t.Fatalf("%s restore baseline=%v", unit.ID, failures)
		}
		if err := ValidateTranslationUnitCandidate(root, catalog, unit.ID, "zh-CN", []byte(restored)); err != nil {
			t.Fatalf("%s validator baseline: %v", unit.ID, err)
		}
		processed++
	}
	if processed != 19 {
		t.Fatalf("process-equivalent example baseline=%d, want 19", processed)
	}
}

func TestRetranslationProcessRestoresValidatesAndPreservesFormalData(t *testing.T) {
	root, catalog, batchID := makeRetranslationProcessBatch(t, 1)
	canonical := filepath.Join(root, "locales", "zh-CN", "candidates", "lesson-1.article")
	status := filepath.Join(root, "locales", "zh-CN", "status.tsv")
	if err := os.MkdirAll(filepath.Dir(canonical), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(canonical, []byte("canonical\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(status, []byte("status evidence\n"), 0644); err != nil {
		t.Fatal(err)
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	rawPath := filepath.Join(batchDir, "raw-responses", "lesson-1.article")
	rawBefore, _ := os.ReadFile(rawPath)
	result, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	if result.BatchID != batchID || result.RestorePassed != 1 || result.ValidationPassed != 1 || result.RestoreFailed != 0 || result.ValidationFailed != 0 || result.Units[0].Status != "passed" {
		t.Fatalf("result = %+v", result)
	}
	candidate, err := os.ReadFile(filepath.Join(batchDir, "candidates", "lesson-1.article"))
	if err != nil {
		t.Fatal(err)
	}
	if string(candidate) != "* 页面\n\n在此页面使用 `Go`。\n" {
		t.Fatalf("candidate = %q", candidate)
	}
	for path, want := range map[string]string{canonical: "canonical\n", status: "status evidence\n"} {
		got, _ := os.ReadFile(path)
		if string(got) != want {
			t.Fatalf("formal file %s changed: %q", path, got)
		}
	}
	rawAfter, _ := os.ReadFile(rawPath)
	if string(rawAfter) != string(rawBefore) {
		t.Fatal("raw response changed")
	}
	var evidence RetranslationValidation
	data, err := os.ReadFile(filepath.Join(batchDir, "validation", "lesson-1.json"))
	if err != nil || json.Unmarshal(data, &evidence) != nil || evidence.Status != "passed" {
		t.Fatalf("validation evidence = %s, err=%v", data, err)
	}
	if _, err := os.Stat(filepath.Join(batchDir, "result.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batchID}); err == nil || !strings.Contains(err.Error(), "already processed") {
		t.Fatalf("repeat error = %v", err)
	}
}

func TestRetranslationProcessPreflightRejectsUnsafeBatch(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, string, string)
		want   string
	}{
		{"saved input differs", func(t *testing.T, root, batch string) {
			path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batch, "inputs", "lesson-1.article")
			data, _ := os.ReadFile(path)
			data = append(data, 'x')
			if err := os.WriteFile(path, data, 0644); err != nil {
				t.Fatal(err)
			}
			rewriteProcessManifest(t, root, batch, func(m *RetranslationBatchManifest) { m.Units[0].InputSHA256 = sum(data) })
		}, "regenerated Default protected input differs"},
		{"source hash", func(t *testing.T, root, batch string) {
			rewriteProcessManifest(t, root, batch, func(m *RetranslationBatchManifest) { m.Units[0].SourceSHA256 = strings.Repeat("0", 64) })
		}, "source metadata"},
		{"input hash", func(t *testing.T, root, batch string) {
			rewriteProcessManifest(t, root, batch, func(m *RetranslationBatchManifest) { m.Units[0].InputSHA256 = strings.Repeat("0", 64) })
		}, "input_sha256 mismatch"},
		{"token count", func(t *testing.T, root, batch string) {
			rewriteProcessManifest(t, root, batch, func(m *RetranslationBatchManifest) { m.Units[0].ProtectedTokenCount++ })
		}, "protected_token_count"},
		{"missing raw", func(t *testing.T, root, batch string) {
			if err := os.Remove(filepath.Join(root, "data", "retranslation-runs", "zh-CN", batch, "raw-responses", "lesson-1.article")); err != nil {
				t.Fatal(err)
			}
		}, "read raw response"},
		{"extra raw", func(t *testing.T, root, batch string) {
			if err := os.WriteFile(filepath.Join(root, "data", "retranslation-runs", "zh-CN", batch, "raw-responses", "extra.article"), []byte("extra"), 0644); err != nil {
				t.Fatal(err)
			}
		}, "unexpected raw response"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, catalog, batchID := makeRetranslationProcessBatch(t, 1)
			tt.mutate(t, root, batchID)
			_, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN"})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
			batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
			for _, name := range []string{"candidates", "validation", "result.json"} {
				if _, statErr := os.Stat(filepath.Join(batchDir, name)); !os.IsNotExist(statErr) {
					t.Fatalf("unsafe output %s exists: %v", name, statErr)
				}
			}
		})
	}
}

func TestRetranslationProcessRecordsRestoreFailures(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(string) string
		want   string
	}{
		{"missing token", func(raw string) string { return strings.Replace(raw, translationTokenRE.FindString(raw), "", 1) }, "occurrence count"},
		{"duplicate token", func(raw string) string { token := translationTokenRE.FindString(raw); return raw + token }, "occurrence count"},
		{"unknown token", func(raw string) string { return raw + "⟪GTI18N_deadbeef_999999⟫" }, "unknown protected token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, catalog, batchID := makeRetranslationProcessBatch(t, 1)
			batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
			rawPath := filepath.Join(batchDir, "raw-responses", "lesson-1.article")
			raw, _ := os.ReadFile(rawPath)
			if err := os.WriteFile(rawPath, []byte(tt.mutate(string(raw))), 0644); err != nil {
				t.Fatal(err)
			}
			result, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN"})
			if err != nil {
				t.Fatal(err)
			}
			if result.RestoreFailed != 1 || result.Units[0].Status != "restore_failed" {
				t.Fatalf("result = %+v", result)
			}
			if _, err := os.Stat(filepath.Join(batchDir, "candidates", "lesson-1.article")); !os.IsNotExist(err) {
				t.Fatalf("candidate created: %v", err)
			}
			data, _ := os.ReadFile(filepath.Join(batchDir, "validation", "lesson-1.json"))
			if !strings.Contains(string(data), tt.want) {
				t.Fatalf("evidence = %s", data)
			}
		})
	}
}

func TestRetranslationProcessKeepsCandidateWhenValidationFails(t *testing.T) {
	root, catalog, batchID := makeRetranslationProcessBatch(t, 1)
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	rawPath := filepath.Join(batchDir, "raw-responses", "lesson-1.article")
	raw, _ := os.ReadFile(rawPath)
	raw = []byte(strings.Replace(string(raw), "。", "，另见 `bad`。", 1))
	if err := os.WriteFile(rawPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	result, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	if result.RestorePassed != 1 || result.ValidationFailed != 1 || result.Units[0].Status != "validation_failed" {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(filepath.Join(batchDir, "candidates", "lesson-1.article")); err != nil {
		t.Fatal(err)
	}
}

func TestRetranslationProcessAutomaticBatchOrdering(t *testing.T) {
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	catalog := retranslationTestCatalog(2)
	first, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	second, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range []*RetranslationExportResult{first, second} {
		dir := filepath.Join(root, result.BatchPath, "raw-responses")
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	secondManifest := readRetranslationManifest(t, root, second.BatchID)
	secondDir := filepath.Join(root, second.BatchPath)
	input, _ := os.ReadFile(filepath.Join(secondDir, filepath.FromSlash(secondManifest.Units[0].InputPath)))
	if err := os.WriteFile(filepath.Join(secondDir, "raw-responses", retranslationUnitInputName(&TranslationUnit{ID: secondManifest.Units[0].UnitID, Kind: secondManifest.Units[0].UnitKind})), input, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN"}); err == nil || !strings.Contains(err.Error(), "read raw response") {
		t.Fatalf("ordering error = %v", err)
	}
	firstManifest := readRetranslationManifest(t, root, first.BatchID)
	firstDir := filepath.Join(root, first.BatchPath)
	input, _ = os.ReadFile(filepath.Join(firstDir, filepath.FromSlash(firstManifest.Units[0].InputPath)))
	if err := os.WriteFile(filepath.Join(firstDir, "raw-responses", retranslationUnitInputName(&TranslationUnit{ID: firstManifest.Units[0].UnitID, Kind: firstManifest.Units[0].UnitKind})), input, 0644); err != nil {
		t.Fatal(err)
	}
	processed, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN"})
	if err != nil || processed.BatchID != first.BatchID {
		t.Fatalf("first result=%+v err=%v", processed, err)
	}
	processed, err = ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN"})
	if err != nil || processed.BatchID != second.BatchID {
		t.Fatalf("second result=%+v err=%v", processed, err)
	}
	done, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN"})
	if err != nil || !done.NoPendingBatches {
		t.Fatalf("done result=%+v err=%v", done, err)
	}
}
