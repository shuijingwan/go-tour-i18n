package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestGlossaryCompatibilityCLICurrentCatalog(t *testing.T) {
	root, expected := generationRunSourceFixture(t)
	catalog, err := glossaryCompatibilityCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Pages) != len(expected.Pages) || len(catalog.Examples) != len(expected.Examples) || len(catalog.Adapted) == 0 || len(catalog.Adapted) != len(expected.Adapted) || len(catalog.TranslationPackages) != len(expected.TranslationPackages) || len(catalog.TranslationScope) != 0 {
		t.Fatal("compatibility Catalog lost complete native/registered inventory")
	}
	for _, p := range expected.Pages {
		u, err := catalog.Unit(p.ID)
		if err != nil || !bytes.Equal(u.Source, p.Source) || u.SourceSHA256 != p.SourceSHA256 {
			t.Fatal("native source not hydrated", p.ID, err)
		}
	}
	for _, a := range expected.Adapted {
		u, err := catalog.Unit(a.Unit.ID)
		if err != nil || u.Adapter == nil || !bytes.Equal(u.Source, a.Unit.Source) || u.SourceSHA256 != a.Unit.SourceSHA256 {
			t.Fatal("registered source/adapter missing", a.Unit.ID, err)
		}
	}
	installCurrentValidationExportAuthority(t, currentValidationExportFixture{root: root, locale: "en-GB"})
	t.Chdir(root)
	before := authorizationDiagnosisInventory(t, root)
	currentSHA := courseContextSHA(courseContextRead(t, root, "locales/en-GB/glossary.yaml"))
	for _, old := range []string{currentSHA, courseContextSHA([]byte("unavailable historical glossary"))} {
		out, err := authorizationDiagnosisOutput(t, func() error {
			return run([]string{"glossary-compatibility", "status", "--locale", "en-GB", "--old-sha256", old, "--scope", "tu:welcome/1"})
		})
		if err != nil {
			t.Fatal("status dispatch", err)
		}
		var result i18n.GlossaryCompatibilityResolution
		if err := json.Unmarshal(out, &result); err != nil {
			t.Fatal("status JSON contract", err)
		}
		switch result.Status {
		case "exact", "compatible", "affected", "missing evidence", "stale evidence":
		default:
			t.Fatal("unknown machine status", result)
		}
		if old == currentSHA && result.Status != "exact" {
			t.Fatal("exact glossary identity changed semantics", result)
		}
		if old != currentSHA && (result.Status == "exact" || result.Status == "compatible") {
			t.Fatal("missing historical evidence accepted", result)
		}
	}
	if after := authorizationDiagnosisInventory(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("readonly status mutated fixture")
	}
	// A stale committed source lock must be rejected, not replaced by a nil
	// Catalog or a fabricated compatibility enum.
	p := "_content/tour/basics.article"
	courseContextWrite(t, root, p, append(courseContextRead(t, root, p), []byte("\nChanged source.\n")...))
	if _, err := glossaryCompatibilityCatalog(root); err == nil {
		t.Fatal("source/catalog mismatch accepted")
	}
}

func TestGlossaryCompatibilityCLIRejectsInvalidArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"unknown"}, {"archive"}, {"archive", "--all", "--locale", "zh-CN"}, {"assess", "--all"}, {"check", "--locale", "zh-CN", "--evidence", "../outside"}, {"check", "--locale", "zh-CN", "--evidence", "/tmp/outside"}, {"status", "--locale", "zh-CN", "--review-id", "other"}, {"archive", "--locale", "zh-CN", "--scope", "tu:one"}, {"config-baseline", "--all", "--review-id", "one"}, {"archive", "--locale", "../escape"}, {"archive", "--locale", "zh-CN", "trailing"}} {
		if glossaryCompatibilityCommand(t.TempDir(), args) == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestGlossaryCompatibilityCLIArchiveIsImmutableAndReviewed(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "locales/zh-CN/glossary.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	original := []byte("locale: zh-CN\nmandatory:\n  Go: Go\n")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	args := []string{"archive", "--locale", "zh-CN"}
	if glossaryCompatibilityCommand(root, args) == nil {
		t.Fatal("unreviewed glossary archived")
	}
	if _, _, err := i18n.RecordGlossaryReview(root, "zh-CN", "original", "independent-reviewer", "passed", nil); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := glossaryCompatibilityCommand(root, args); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(path, append(original, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	if glossaryCompatibilityCommand(root, args) == nil {
		t.Fatal("byte-stale Review allowed archive/current")
	}
	entries, err := os.ReadDir(filepath.Join(root, "data/glossary-history/zh-CN"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("archive=%v err=%v", entries, err)
	}
}
