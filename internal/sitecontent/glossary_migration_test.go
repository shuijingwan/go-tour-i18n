package sitecontent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// All campaign receipts and mutations are confined to the isolated fixture.
func migrationTourFixture(t *testing.T) string {
	t.Helper()
	root := existingTourFixture(t)
	if err := os.RemoveAll(filepath.Join(root, "data/unified-glossary-reviews")); err != nil {
		t.Fatal(err)
	}
	if err := i18n.RequireCurrentGlossaryReview(root, "zh-CN"); err != nil {
		t.Fatal("old Tour authority not current", err)
	}
	return root
}

func migrationReview(t *testing.T, root, id, decision string) *i18n.UnifiedGlossaryReview {
	t.Helper()
	b, _, err := i18n.ExportUnifiedGlossaryBundle(root, "zh-CN", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	findings := []string{}
	if decision == "failed" {
		findings = append(findings, "Synthetic full-site terminology finding")
	}
	r, _, err := i18n.RecordUnifiedGlossaryReview(root, b, id, "synthetic-independent-reviewer", "synthetic-generation", decision, findings)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func assertMigrationNewWorkBlocked(t *testing.T, root string) {
	t.Helper()
	if _, err := i18n.RequireUnifiedGlossaryReview(root, "zh-CN"); err == nil {
		t.Fatal("strict unified authority accepted")
	}
	if err := i18n.RequireNewLanguageGlossaryReview(root, "zh-CN"); err == nil {
		t.Fatal("new language authority accepted")
	}
	if _, err := GenerationPlan(root, "zh-CN", "learn-docs-v1", "initial", nil); err == nil {
		t.Fatal("Site Page generation accepted")
	}
	if _, err := PlanStructuredAssets(root, "zh-CN"); err == nil {
		t.Fatal("structured generation accepted")
	}
}

func TestVersionlessGlossaryMigrationCoexistence(t *testing.T) {
	for _, decision := range []string{"failed", "conflicting"} {
		t.Run(decision, func(t *testing.T) {
			root := migrationTourFixture(t)
			if decision == "failed" {
				migrationReview(t, root, "migration-failed", "failed")
			} else {
				migrationReview(t, root, "migration-pass-one", "passed")
				migrationReview(t, root, "migration-pass-two", "passed")
			}
			assertMigrationNewWorkBlocked(t, root)
			if err := i18n.RequireCurrentGlossaryReview(root, "zh-CN"); err != nil {
				t.Fatal("campaign evidence shadowed exact old Tour proof", err)
			}
			g, err := LoadCurrent(root)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := CheckLocale(root, g, "zh-CN"); err != nil {
				t.Fatal("unchanged historical Tour completion became stale", err)
			}
		})
	}
}

func TestVersionlessGlossaryMigrationChangedGlossary(t *testing.T) {
	root := migrationTourFixture(t)
	b, err := readRegular(root, "locales/zh-CN/glossary.yaml")
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "locales/zh-CN/glossary.yaml", append(b, []byte("\n# Synthetic glossary byte change\n")...))
	migrationReview(t, root, "migration-changed-failed", "failed")
	assertMigrationNewWorkBlocked(t, root)
	if err := i18n.RequireCurrentGlossaryReview(root, "zh-CN"); err == nil {
		t.Fatal("stale old glossary authority carried")
	}
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CheckLocale(root, g, "zh-CN"); err == nil {
		t.Fatal("changed glossary Tour completion accepted")
	}
}

func TestVersionlessGlossaryMigrationCorruption(t *testing.T) {
	for _, defect := range []string{"json", "schema", "identity", "unsafe-path", "missing-bundle", "bundle-hash", "symlink"} {
		t.Run(defect, func(t *testing.T) {
			root := siteFixture(t)
			r := migrationReview(t, root, "migration-corrupt", "failed")
			p := i18n.UnifiedGlossaryReviewPath("zh-CN", r.ReviewID)
			archive := "data/unified-glossary-reviews/zh-CN-bundles/" + r.BundleSHA + ".zip"
			switch defect {
			case "json":
				writeFixture(t, root, p, []byte("{broken JSON"))
			case "schema", "identity", "unsafe-path":
				if defect == "schema" {
					r.Schema = "unknown"
				}
				if defect == "identity" {
					r.Identity = strings.Repeat("0", 64)
				}
				if defect == "unsafe-path" {
					r.GlossaryPath = "../glossary.yaml"
				}
				b, _ := Encode(r)
				writeFixture(t, root, p, b)
			case "missing-bundle":
				if err := os.Remove(filepath.Join(root, archive)); err != nil {
					t.Fatal(err)
				}
			case "bundle-hash":
				writeFixture(t, root, archive, []byte("corrupted archive"))
			case "symlink":
				if err := os.Remove(filepath.Join(root, p)); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(root, archive), filepath.Join(root, p)); err != nil {
					t.Fatal(err)
				}
			}
			assertMigrationNewWorkBlocked(t, root)
			if err := i18n.RequireCurrentGlossaryReview(root, "zh-CN"); err == nil {
				t.Fatal("legacy fallback masked corrupt unified artifact")
			}
		})
	}
}
