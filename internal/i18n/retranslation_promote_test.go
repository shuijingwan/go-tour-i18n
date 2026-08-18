package i18n

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func processedPromotionFixture(t *testing.T, count int) (string, *Catalog, string) {
	t.Helper()
	root, catalog, batchID := makeRetranslationProcessBatch(t, count)
	if _, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batchID}); err != nil {
		t.Fatal(err)
	}
	return root, catalog, batchID
}

func addProcessedPromotionBatch(t *testing.T, root string, catalog *Catalog, batchID string, ids []string) {
	t.Helper()
	result, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", BatchID: batchID, PageIDs: ids, Limit: len(ids), AllowReexport: true})
	if err != nil {
		t.Fatal(err)
	}
	batchDir := filepath.Join(root, result.BatchPath)
	if err := os.Mkdir(filepath.Join(batchDir, "raw-responses"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := readRetranslationManifest(t, root, batchID)
	for _, record := range manifest.Pages {
		input, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(record.InputPath)))
		if err != nil {
			t.Fatal(err)
		}
		translated := strings.ReplaceAll(string(input), "* Page", "* 新页面")
		translated = strings.ReplaceAll(translated, "Use ", "新批次使用 ")
		translated = strings.ReplaceAll(translated, " on this page.", "。")
		if err := os.WriteFile(filepath.Join(batchDir, "raw-responses", flattenedPageArticleName(record.PageID)), []byte(translated), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batchID}); err != nil {
		t.Fatal(err)
	}
}

func writePromotionStatus(t *testing.T, root string, catalog *Catalog, canonical string) {
	t.Helper()
	localeDir := filepath.Join(root, "locales", "zh-CN")
	if err := os.MkdirAll(filepath.Join(localeDir, "candidates"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "locale.json"), []byte(`{"locale":"zh-CN","phase":"scaffold","translation_unit":"present.Section"}`), 0644); err != nil {
		t.Fatal(err)
	}
	statuses := make([]Status, 0, len(catalog.Pages))
	for _, page := range catalog.Pages {
		path := ""
		state := "pending"
		if canonical != "" {
			path = canonicalCandidatePath("zh-CN", page.ID)
			state = "ready"
			if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(path)), []byte(canonical), 0644); err != nil {
				t.Fatal(err)
			}
		}
		statuses = append(statuses, Status{PageID: page.ID, State: state, Attempts: 7, SourceSHA256: page.SourceSHA256, CandidatePath: path})
	}
	if err := writeStatuses(filepath.Join(localeDir, "status.tsv"), statuses); err != nil {
		t.Fatal(err)
	}
}

func TestRetranslationPromoteDryRunDoesNotModifyCanonicalAndLatestWins(t *testing.T) {
	root, catalog, _ := processedPromotionFixture(t, 2)
	addProcessedPromotionBatch(t, root, catalog, "chatgpt-zh-CN-002", []string{"lesson/1"})
	writePromotionStatus(t, root, catalog, "old canonical\n")
	statusPath := filepath.Join(root, "locales", "zh-CN", "status.tsv")
	statusBefore, _ := os.ReadFile(statusPath)
	canonicalPath := filepath.Join(root, filepath.FromSlash(canonicalCandidatePath("zh-CN", "lesson/1")))
	canonicalBefore, _ := os.ReadFile(canonicalPath)
	plan, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.PageCount != 2 || plan.ChangedCount != 2 || plan.UnchangedCount != 0 || !plan.CanApply || plan.Pages[0].BatchID != "chatgpt-zh-CN-002" {
		t.Fatalf("plan = %+v", plan)
	}
	statusAfter, _ := os.ReadFile(statusPath)
	canonicalAfter, _ := os.ReadFile(canonicalPath)
	if !bytes.Equal(statusBefore, statusAfter) || !bytes.Equal(canonicalBefore, canonicalAfter) {
		t.Fatal("dry-run modified canonical data")
	}
}

func TestRetranslationPromoteLatestFailureDoesNotFallback(t *testing.T) {
	root, catalog, _ := processedPromotionFixture(t, 1)
	addProcessedPromotionBatch(t, root, catalog, "chatgpt-zh-CN-002", []string{"lesson/1"})
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", "chatgpt-zh-CN-002")
	var result RetranslationProcessResult
	b, _ := os.ReadFile(filepath.Join(batchDir, "result.json"))
	_ = json.Unmarshal(b, &result)
	result.Pages[0].Status = "validation_failed"
	if err := writeTranslationJSON(filepath.Join(batchDir, "result.json"), result); err != nil {
		t.Fatal(err)
	}
	if _, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"}); err == nil || !strings.Contains(err.Error(), "refusing fallback") {
		t.Fatalf("error = %v", err)
	}
}

func TestRetranslationPromoteRejectsIncompleteAndInvalidEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, string, *Catalog, string)
		want   string
	}{
		{"missing page", func(t *testing.T, root string, catalog *Catalog, batch string) {
			catalog.Pages = append(catalog.Pages, Page{ID: "missing/1", Article: "missing.article", Source: []byte("* Missing\n"), SourceSHA256: sum([]byte("* Missing\n"))})
		}, "covers 1 of 2"},
		{"illegal batch", func(t *testing.T, root string, catalog *Catalog, batch string) {
			if err := os.MkdirAll(filepath.Join(root, "data", "retranslation-runs", "zh-CN", "bad"), 0755); err != nil {
				t.Fatal(err)
			}
		}, "illegal retranslation batch"},
		{"ambiguous batch number", func(t *testing.T, root string, catalog *Catalog, batch string) {
			source := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batch)
			target := filepath.Join(root, "data", "retranslation-runs", "zh-CN", "chatgpt-zh-CN-01")
			if err := copyTestTree(source, target); err != nil {
				t.Fatal(err)
			}
			manifest := readRetranslationManifestAt(t, filepath.Join(target, "manifest.json"))
			manifest.BatchID = "chatgpt-zh-CN-01"
			if err := writeTranslationJSON(filepath.Join(target, "manifest.json"), manifest); err != nil {
				t.Fatal(err)
			}
		}, "ambiguous retranslation batch number"},
		{"source hash", func(t *testing.T, root string, catalog *Catalog, batch string) {
			rewriteProcessManifest(t, root, batch, func(m *RetranslationBatchManifest) { m.Pages[0].SourceSHA256 = strings.Repeat("0", 64) })
		}, "source metadata"},
		{"result validation mismatch", func(t *testing.T, root string, catalog *Catalog, batch string) {
			path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batch, "validation", "lesson-1.json")
			var v RetranslationValidation
			b, _ := os.ReadFile(path)
			_ = json.Unmarshal(b, &v)
			v.Status = "validation_failed"
			if err := writeTranslationJSON(path, v); err != nil {
				t.Fatal(err)
			}
		}, "validation does not match"},
		{"validator failure", func(t *testing.T, root string, catalog *Catalog, batch string) {
			path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batch, "candidates", "lesson-1.article")
			b, _ := os.ReadFile(path)
			b = append(b, []byte(" `unexpected`\n")...)
			if err := os.WriteFile(path, b, 0644); err != nil {
				t.Fatal(err)
			}
		}, "canonical candidate validator"},
		{"GTI18N token", func(t *testing.T, root string, catalog *Catalog, batch string) {
			path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batch, "candidates", "lesson-1.article")
			b, _ := os.ReadFile(path)
			b = append(b, []byte(" GTI18N_leftover\n")...)
			if err := os.WriteFile(path, b, 0644); err != nil {
				t.Fatal(err)
			}
		}, "contains GTI18N token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, catalog, batch := processedPromotionFixture(t, 1)
			tt.mutate(t, root, catalog, batch)
			if _, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"}); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func readRetranslationManifestAt(t *testing.T, path string) RetranslationBatchManifest {
	t.Helper()
	var manifest RetranslationBatchManifest
	b, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(b, &manifest) != nil {
		t.Fatalf("read manifest %s: %v", path, err)
	}
	return manifest
}

func copyTestTree(source, target string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(target, rel)
		if info.IsDir() {
			return os.MkdirAll(dst, info.Mode())
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, info.Mode())
	})
}

func TestRetranslationPromoteSelectsRetryFinalCandidate(t *testing.T) {
	root, catalog, batch := processRetryFixture(t, 1, func(raw string) string { return raw + " `bad`" })
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batch)
	valid, _ := os.ReadFile(filepath.Join(batchDir, "raw-responses", "lesson-1.article"))
	valid = []byte(strings.TrimSuffix(string(valid), " `bad`"))
	writeRetryRaw(t, root, batch, "lesson/1", 2, string(valid))
	if _, err := ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batch, PageID: "lesson/1"}); err != nil {
		t.Fatal(err)
	}
	plan, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"})
	if err != nil || plan.PageCount != 1 || plan.Pages[0].BatchID != batch {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}

func TestRetranslationPromoteApplyUpdatesCanonicalAndPreservesAttempts(t *testing.T) {
	root, catalog, batch := processedPromotionFixture(t, 2)
	writePromotionStatus(t, root, catalog, "old\n")
	now := time.Date(2026, 8, 18, 3, 4, 5, 0, time.FixedZone("test", 8*60*60))
	plan, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN", Apply: true, Now: func() time.Time { return now }})
	if err != nil || plan.PageCount != 2 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	statuses, err := ReadStatuses(filepath.Join(root, "locales", "zh-CN", "status.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range statuses {
		if status.State != "ready" || status.Attempts != 7 || status.SourceSHA256 == "" || status.CandidatePath != canonicalCandidatePath("zh-CN", status.PageID) || status.UpdatedAt != "2026-08-17T19:04:05Z" || status.Note != "ChatGPT retranslation promoted from "+batch+"; passed canonical validator" {
			t.Fatalf("status = %+v", status)
		}
		candidate, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(status.CandidatePath)))
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateCandidateForLocale(root, catalog, status.PageID, "zh-CN", candidate); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRetranslationPromoteApplyRollbackOnCommitFailure(t *testing.T) {
	root, catalog, _ := processedPromotionFixture(t, 1)
	writePromotionStatus(t, root, catalog, "old\n")
	statusPath := filepath.Join(root, "locales", "zh-CN", "status.tsv")
	candidatePath := filepath.Join(root, filepath.FromSlash(canonicalCandidatePath("zh-CN", "lesson/1")))
	statusBefore, _ := os.ReadFile(statusPath)
	candidateBefore, _ := os.ReadFile(candidatePath)
	calls := 0
	rename := func(old, new string) error {
		calls++
		if calls == 4 {
			return errors.New("injected failure")
		}
		return os.Rename(old, new)
	}
	if _, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN", Apply: true, rename: rename}); err == nil || !strings.Contains(err.Error(), "injected failure") {
		t.Fatalf("error = %v", err)
	}
	statusAfter, _ := os.ReadFile(statusPath)
	candidateAfter, _ := os.ReadFile(candidatePath)
	if !bytes.Equal(statusBefore, statusAfter) || !bytes.Equal(candidateBefore, candidateAfter) {
		t.Fatal("failed apply left partial canonical state")
	}
}
