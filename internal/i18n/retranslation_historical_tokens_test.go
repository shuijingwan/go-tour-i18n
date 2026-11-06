package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Only representation behavior is synthetic; fixtures use the normal schema-v2
// exporter, processor, latest-result selector and glossary-stale reader.
type historicalTokensFixtureAdapter struct{ recoveryFixtureAdapter }

func (a *historicalTokensFixtureAdapter) ProtectedInput(*Glossary) (string, []string, error) {
	tokens := make([]string, 26)
	for i := range tokens {
		tokens[i] = fmt.Sprintf("⟦P%d⟧", i)
	}
	return strings.Join(tokens, "English prose") + "\n", tokens, nil
}

func historicalTokensFixture(t *testing.T) (string, *Catalog, *TranslationUnit, *latestRetranslationUnits, []byte) {
	t.Helper()
	root := t.TempDir()
	writeGlossaryStaleTestGlossary(t, root, "synthetic-review", false)
	a := &historicalTokensFixtureAdapter{}
	source := []byte("Frozen English source")
	c := &Catalog{}
	descriptor := TranslationPackageDescriptor{ID: "learn-docs-v1"}
	for index, id := range []string{"historical-page", "other-package-page"} {
		descriptor.Units = append(descriptor.Units, AdaptedTranslationUnit{
			Unit:    TranslationUnit{ID: id, Kind: UnitKindPage, SourcePath: "_content/doc/" + id + ".html", Source: source, SourceSHA256: sum(source)},
			Adapter: a, StableIndex: index + 1, Route: "/doc/" + id,
		})
	}
	if err := c.RegisterTranslationPackage(descriptor); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage("learn-docs-v1"); err != nil {
		t.Fatal(err)
	}
	exported, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitIDs: []string{"historical-page"}})
	if err != nil {
		t.Fatal(err)
	}
	processGlossaryStaleTestBatch(t, root, c, exported.BatchID)
	// Historical schema v2 predates the explicit EOF transport field. Preserve
	// its exact persisted bytes while exercising the supported legacy reader.
	manifest := readRetranslationManifest(t, root, exported.BatchID)
	if manifest.SchemaVersion != 2 || manifest.UnitKind != UnitKindPage || manifest.ProtectionMode != "default" {
		t.Fatalf("historical fixture manifest=%+v", manifest)
	}
	manifest.ArtifactEOF = ""
	if err := writeTranslationJSON(filepath.Join(root, exported.BatchPath, "manifest.json"), manifest); err != nil {
		t.Fatal(err)
	}
	latest, err := selectLatestRetranslationUnits(root, c, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	u, err := c.Unit("historical-page")
	if err != nil {
		t.Fatal(err)
	}
	input, _, err := a.ProtectedInput(nil)
	if err != nil {
		t.Fatal(err)
	}
	return root, c, u, latest, []byte(input)
}

func TestGlossaryStaleHistoricalTokenIntegrity(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"supported schema v2", ""},
		{"count 25", "protected token count 26 does not match manifest 25"},
		{"count 27", "protected token count 26 does not match manifest 27"},
		{"duplicate", "duplicate protected token ⟦P7⟧"},
		{"SHA mismatch", "saved input hash does not match manifest"},
		{"malformed", "unknown or malformed protected token"},
		{"unknown", "unknown or malformed protected token"},
		{"foreign representation", "unknown or malformed protected token"},
		{"canonical restore rejects", "obsolete adapter false positive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, _, unit, latest, saved := historicalTokensFixture(t)
			choice := latest.selectedByID[unit.ID]
			switch tc.name {
			case "count 25":
				choice.manifest.ProtectedTokenCount = 25
			case "count 27":
				choice.manifest.ProtectedTokenCount = 27
			case "duplicate":
				saved = append([]byte("⟦P7⟧"), saved...)
				choice.manifest.ProtectedTokenCount = 27
			case "SHA mismatch":
				choice.manifest.InputSHA256 = strings.Repeat("0", 64)
			case "malformed":
				saved = append([]byte("⟦P00⟧"), saved...)
			case "unknown":
				saved = append([]byte("⟦P999⟧"), saved...)
			case "foreign representation":
				saved = append([]byte("⟪GTI18N_deadbeef_000000⟫"), saved...)
			case "canonical restore rejects":
				unit.Adapter.(*historicalTokensFixtureAdapter).broken = true
			}
			if tc.name != "SHA mismatch" {
				choice.manifest.InputSHA256 = sum(saved)
			}
			path := filepath.Join(choice.batchDir, filepath.FromSlash(choice.manifest.InputPath))
			if err := os.WriteFile(path, saved, 0644); err != nil {
				t.Fatal(err)
			}
			latest.selectedByID[unit.ID] = choice
			before := snapshotDirectoryBytes(t, root)
			got, err := validateGlossaryStaleRetranslationUnit(root, "zh-CN", unit, append([]byte("Changed prose "), saved...), latest)
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("error=%v, want %q", err, tc.want)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if got.protectedTokenCount != 26 || got.inputSHA256 != sum(saved) {
					t.Fatalf("provenance=%+v", got)
				}
				count, err := validateSavedRetranslationInputTokens(unit, saved, 26)
				if err != nil || count != 26 {
					t.Fatalf("recognized=%d err=%v", count, err)
				}
			}
			if !reflect.DeepEqual(before, snapshotDirectoryBytes(t, root)) {
				t.Fatal("reader mutated evidence")
			}
		})
	}
}

func TestGlossaryStaleHistoricalTokensStillRequireRealDrift(t *testing.T) {
	root, c, unit, latest, saved := historicalTokensFixture(t)
	if !reflect.DeepEqual(c.TranslationScope, []string{"historical-page", "other-package-page"}) {
		t.Fatal("explicit Unit export narrowed complete package scope", c.TranslationScope)
	}
	before := snapshotDirectoryBytes(t, root)
	if _, err := validateGlossaryStaleRetranslationUnit(root, "zh-CN", unit, bytes.Clone(saved), latest); err == nil || !strings.Contains(err.Error(), "no glossary-induced drift") {
		t.Fatalf("no-drift reader error=%v", err)
	}
	// Complete package scope and explicit Unit selection follow the real export
	// preflight. No synthetic batch may be created when protected bytes match.
	if _, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, AllowReexport: true, GlossaryStale: true, UnitIDs: []string{unit.ID}}); err == nil || !strings.Contains(err.Error(), "no glossary-induced drift") {
		t.Fatalf("no-drift export error=%v", err)
	}
	if !reflect.DeepEqual(before, snapshotDirectoryBytes(t, root)) {
		t.Fatal("blocked export mutated evidence")
	}
	if !reflect.DeepEqual(c.TranslationScope, []string{"historical-page", "other-package-page"}) {
		t.Fatal("glossary-stale preflight narrowed complete package scope", c.TranslationScope)
	}
}

func TestGlossaryStaleCurrentTokenIntegrity(t *testing.T) {
	unit := &TranslationUnit{ID: "tour", Kind: UnitKindPage}
	const token = "⟪GTI18N_deadbeef_000000⟫"
	for _, tc := range []struct {
		name, input, want string
		count             int
	}{
		{"current Tour token", token + " prose", "", 1},
		{"count mismatch", token, "protected token count 1 does not match manifest 2", 2},
		{"duplicate", token + token, "duplicate protected token", 2},
		{"malformed", token + "⟪GTI18N_bad⟫", "unknown or malformed protected token", 1},
		{"truncated", token + "⟪GTI18N_deadbeef_000001", "unknown or malformed protected token", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			count, err := validateSavedRetranslationInputTokens(unit, []byte(tc.input), tc.count)
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("error=%v, want %q", err, tc.want)
				}
			} else if err != nil || count != tc.count {
				t.Fatalf("recognized=%d err=%v", count, err)
			}
		})
	}
}

func TestGlossaryStaleTokenRepresentationIsNotGuessed(t *testing.T) {
	unit := &TranslationUnit{ID: "tour", Kind: UnitKindPage}
	if _, err := validateSavedRetranslationInputTokens(unit, []byte("⟦P0⟧"), 1); err == nil {
		t.Fatal("Tour accepted an adapted token without an adapter")
	}
	if _, err := validateSavedRetranslationInputTokens(unit, []byte("⟪GTI18N_deadbeef_000000⟫⟪GTI18N_bad⟫"), 1); err == nil {
		t.Fatal("accepted malformed Tour token")
	}
}
