package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type recoveryFixtureAdapter struct {
	broken  bool
	invalid bool
}

func (a *recoveryFixtureAdapter) ProtectedInput(*Glossary) (string, []string, error) {
	return "Frozen English source", nil, nil
}
func (a *recoveryFixtureAdapter) Restore(raw string) ([]byte, error) {
	if a.broken {
		return nil, fmt.Errorf("obsolete adapter false positive")
	}
	return []byte(raw), nil
}
func (a *recoveryFixtureAdapter) Validate([]byte, *Glossary) error {
	if a.invalid {
		return fmt.Errorf("real candidate defect")
	}
	return nil
}

// This fixture's physical format is one prose field, with no code regions.
// Implement the active shared comment authority rather than bypassing its gate.
func (*recoveryFixtureAdapter) CommentSourceBlocks() ([]CommentCodeBlock, string, error) {
	return nil, "", nil
}
func (*recoveryFixtureAdapter) CommentTargetBlocks([]byte) ([]CommentCodeBlock, error) {
	return nil, nil
}

func restoreRecoveryFixture(t *testing.T, locale string) (string, *Catalog, string, *recoveryFixtureAdapter) {
	t.Helper()
	root := t.TempDir()
	copyBundleAuthority(t, root, translationUnitGenerationAuthorityPaths)
	writeRetranslationTestGlossaryForLocale(t, root, locale)
	recordSyntheticUnifiedReview(t, root, "synthetic-generation-review", locale)
	a := &recoveryFixtureAdapter{}
	source := []byte("Frozen English source")
	c := &Catalog{}
	if err := c.RegisterTranslationPackage(TranslationPackageDescriptor{ID: "generic-fixture", Units: []AdaptedTranslationUnit{{Unit: TranslationUnit{ID: "fixture-home", Kind: UnitKindPage, Source: source, SourceSHA256: sum(source), SourcePath: "_content/fixture.md"}, Adapter: a, Route: "/fixture/", StableIndex: 1}}}); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage("generic-fixture"); err != nil {
		t.Fatal(err)
	}
	exported, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: locale, Generator: RetranslationGeneratorChatGPT, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	bundle, gm, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: locale, BatchID: exported.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	outputDir := t.TempDir()
	writeGenerationFixtureOutputs(t, root, exported.BatchID, outputDir, gm)
	resultBundle, _, err := PackGenerationResultBundle(root, c, bundle, "chatgpt", "gpt-5.6-sol-high", outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ImportGenerationResultBundle(root, c, bundle, resultBundle); err != nil {
		t.Fatal(err)
	}
	a.broken = true
	result, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: locale, BatchID: exported.BatchID})
	if err != nil || result.RestoreFailed != 1 {
		t.Fatalf("process=%+v error=%v", result, err)
	}
	a.broken = false
	return root, c, exported.BatchID, a
}

func TestRetranslationRestoreRecoveryPreservesHistoryAndAttempt(t *testing.T) {
	for _, locale := range []string{"zh-CN", "ja-JP"} {
		t.Run(locale, func(t *testing.T) {
			root, c, batch, a := restoreRecoveryFixture(t, locale)
			dir := filepath.Join(root, "data/retranslation-runs", locale, batch)
			read := func(p string) []byte {
				t.Helper()
				b, err := os.ReadFile(filepath.Join(dir, p))
				if err != nil {
					t.Fatal(err)
				}
				return b
			}
			oldValidation, oldResult, oldRaw := read("validation/fixture-home.json"), read("result.json"), read("raw-responses/fixture-home.article")
			options := RetranslationRevalidateOptions{Locale: locale, BatchID: batch, UnitID: "fixture-home"}
			// A still-broken adapter leaves all current evidence untouched.
			a.broken = true
			if _, err := RevalidateRetranslationCandidate(root, c, options); err == nil || !strings.Contains(err.Error(), "current restore failed") {
				t.Fatalf("error=%v", err)
			}
			if !bytes.Equal(oldValidation, read("validation/fixture-home.json")) || !bytes.Equal(oldResult, read("result.json")) {
				t.Fatal("failed recovery mutated evidence")
			}
			a.broken = false
			got, err := RevalidateRetranslationCandidate(root, c, options)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != "passed" || got.PreviousStatus != "restore_failed" || got.Attempt != 1 || got.RestorePassed != 1 || got.RestoreFailed != 0 || got.ValidationPassed != 1 || got.ValidationFailed != 0 {
				t.Fatalf("result=%+v", got)
			}
			if !bytes.Equal(oldValidation, read(got.HistoryPath)) || !bytes.Equal(oldResult, read(got.ResultHistoryPath)) || !bytes.Equal(oldRaw, read("raw-responses/fixture-home.article")) || !bytes.Equal(oldRaw, read("candidates/fixture-home.article")) {
				t.Fatal("history/raw/candidate mismatch")
			}
			if _, err := os.Stat(filepath.Join(dir, "retries")); !os.IsNotExist(err) {
				t.Fatalf("fake retry: %v", err)
			}
			second, err := RevalidateRetranslationCandidate(root, c, options)
			if err != nil || second.Revalidation != 2 || second.Attempt != 1 {
				t.Fatalf("second=%+v error=%v", second, err)
			}
		})
	}
}

func TestRetranslationRestoreRecoveryRejectsStaleInputsAndProvenance(t *testing.T) {
	for _, change := range []string{"source", "input", "glossary", "provider", "raw", "attempt", "missing-provenance", "existing-candidate", "pending-import"} {
		t.Run(change, func(t *testing.T) {
			root, c, batch, _ := restoreRecoveryFixture(t, "ja-JP")
			dir := filepath.Join(root, "data/retranslation-runs/ja-JP", batch)
			before, _ := os.ReadFile(filepath.Join(dir, "result.json"))
			path := ""
			switch change {
			case "source":
				c.Adapted[0].Unit.SourceSHA256 = strings.Repeat("a", 64)
			case "input":
				path = filepath.Join(dir, "inputs/fixture-home.article")
			case "glossary":
				path = filepath.Join(root, "locales/ja-JP/glossary.yaml")
			case "raw":
				path = filepath.Join(dir, "raw-responses/fixture-home.article")
			case "provider":
				path = filepath.Join(dir, "generation-imports/initial-manifest.json")
				b, _ := os.ReadFile(path)
				if err := os.WriteFile(path, bytes.Replace(b, []byte(`"provider": "chatgpt"`), []byte(`"provider": "invalid"`), 1), 0644); err != nil {
					t.Fatal(err)
				}
				path = ""
			case "attempt":
				path = filepath.Join(dir, "validation/fixture-home.json")
				b, _ := os.ReadFile(path)
				if err := os.WriteFile(path, bytes.Replace(b, []byte(`"attempt": 1`), []byte(`"attempt": 2`), 1), 0644); err != nil {
					t.Fatal(err)
				}
				path = ""
			case "missing-provenance":
				if err := os.Remove(filepath.Join(dir, "generation-imports/initial-manifest.json")); err != nil {
					t.Fatal(err)
				}
			case "existing-candidate":
				path = filepath.Join(dir, "candidates/fixture-home.article")
			case "pending-import":
				if err := os.WriteFile(filepath.Join(dir, "generation-imports/import.pending"), []byte("pending\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if path != "" {
				if err := os.WriteFile(path, []byte("stale bytes\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := RevalidateRetranslationCandidate(root, c, RetranslationRevalidateOptions{Locale: "ja-JP", BatchID: batch, UnitID: "fixture-home"}); err == nil {
				t.Fatal("stale recovery accepted")
			}
			after, _ := os.ReadFile(filepath.Join(dir, "result.json"))
			if !bytes.Equal(before, after) {
				t.Fatal("failed preflight mutated result")
			}
			if _, err := os.Stat(filepath.Join(dir, "revalidation-history")); !os.IsNotExist(err) {
				t.Fatal("failed preflight created history")
			}
		})
	}
}

func TestRetranslationRestoreRecoveryRecordsRealValidationDefect(t *testing.T) {
	root, c, batch, a := restoreRecoveryFixture(t, "ja-JP")
	a.invalid = true
	got, err := RevalidateRetranslationCandidate(root, c, RetranslationRevalidateOptions{Locale: "ja-JP", BatchID: batch, UnitID: "fixture-home"})
	if err != nil || got.Status != "validation_failed" || got.ValidationFailed != 1 || got.Attempt != 1 || got.RestorePassed != 1 {
		t.Fatalf("result=%+v error=%v", got, err)
	}
}
