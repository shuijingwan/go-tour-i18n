package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestTourCheckpointNonTUSurfaceAuthorityBoundary(t *testing.T) {
	root, g, cat, l := currentTourCheckpointFixture(t)
	uiPath := "internal/tour/ui/zh-CN.json"
	ui, err := readRegular(root, uiPath)
	if err != nil {
		t.Fatal(err)
	}
	oldSHA := ""
	for _, closure := range l.Packages {
		for _, ref := range closure.Evidence {
			if ref.Path == uiPath {
				oldSHA = ref.SHA256
			}
		}
	}
	if oldSHA == "" || oldSHA == digest(ui) {
		t.Fatal("fixture lacks historical/current UI difference")
	}
	gate, err := i18n.CurrentExactLocaleSurfaceReviewAGate(root, l.Locale, cat)
	if err != nil {
		t.Fatal(err)
	}
	gatePath := "data/locale-surface-reviews/" + l.Locale + "/" + gate.ReviewID + ".a-gate.json"
	if _, current, err := currentTourCheckpoint(root, g, cat, l); err != nil || !current {
		t.Fatalf("exact-current Surface checkpoint: current=%v %v", current, err)
	}
	plan, err := PlanLocaleLanguage(root, l.Locale)
	if err != nil {
		t.Fatal(err)
	}
	for _, work := range plan.Work {
		if work.Package == "tour-v1" && work.State != "carry" {
			t.Fatalf("exact-current Tour not carried: %+v", work)
		}
	}
	if err := AttachRegisteredTranslationPackages(root, cat); err != nil {
		t.Fatal(err)
	}
	if err := i18n.AuthorizeHistoricalCanonicalReference(root, l.Locale, uiPath, oldSHA, cat); err == nil {
		t.Fatal("Surface PASS became TU authorization for UI")
	}
	for _, kind := range []string{"stale", "missing"} {
		t.Run(kind, func(t *testing.T) {
			if kind == "stale" {
				writeFixture(t, root, uiPath, append(bytes.Clone(ui), '\n'))
				defer writeFixture(t, root, uiPath, ui)
			} else {
				path := filepath.Join(root, gatePath)
				if err := os.Rename(path, path+".fixture-backup"); err != nil {
					t.Fatal(err)
				}
				defer func() {
					if err := os.Rename(path+".fixture-backup", path); err != nil {
						t.Fatal(err)
					}
				}()
			}
			if _, _, err := i18n.RequireCurrentCanonicalQualityCheck(root, l.Locale, cat); err != nil {
				t.Fatalf("Surface state revoked independent TU authority: %v", err)
			}
			if _, err := PlanLocaleLanguage(root, l.Locale); err == nil {
				t.Fatal("missing/stale Surface accepted by campaign")
			} else if !strings.Contains(err.Error(), "exact-current Tour Surface checkpoint required") {
				t.Fatalf("wrong Surface authority boundary: %v", err)
			}
		})
	}
}

func currentTourCheckpointFixture(t *testing.T) (string, *Global, *i18n.Catalog, Locale) {
	t.Helper()
	// Match the CLI's existing exact-byte transformation scope for this complete
	// current input copy. Filesystem observations and currentness stay live.
	t.Cleanup(contentidentity.BeginMemoization())
	// The current corpus binds physical Docs and adopted shell sources. Use the
	// shared complete input copy rather than the sparse legacy ZIP fixture.
	root := copiedCanonicalLocaleFixture(t)
	path := "data/quality-check-snapshots/zh-CN/qc-008/finalization.json"
	b, err := readRegular(repositoryRoot(t), path)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, path, b)
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	cat, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	l, err := readLocaleAuthority(root, g, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	// Return the Tour projection for this fixture's explicit carry checks. Keep
	// the full on-disk locale authority: other packages' successor readers need
	// their installed predecessors, even when those old completions are stale.
	tour := Locale{Schema: l.Schema, Locale: l.Locale, Packages: []Completion{}}
	for _, completion := range l.Packages {
		if completion.Package == "tour-v1" {
			tour.Packages = append(tour.Packages, completion)
		}
	}
	if len(tour.Packages) != 1 {
		t.Fatal("fixture requires one historical Tour completion")
	}
	return root, g, cat, tour
}

func TestTourCurrentCheckpointCarriesLegacyWithoutHistoricalBytes(t *testing.T) {
	root, g, cat, l := currentTourCheckpointFixture(t)
	closure, _ := readRegular(root, LocalePath(l.Locale))
	gate, err := i18n.CurrentExactLocaleSurfaceReviewAGate(root, l.Locale, cat)
	if err != nil {
		t.Fatal(err)
	}
	base := "data/locale-surface-reviews/" + l.Locale + "/" + gate.ReviewID
	md, _ := readRegular(root, base+".md")
	gateBytes, _ := readRegular(root, base+".a-gate.json")
	// The historical registry/UI/SEO world really differs from this already
	// reviewed current world; no new drift is introduced after assessment.
	languages, _ := readRegular(root, "internal/tour/languages.go")
	for _, c := range l.Packages {
		for _, ref := range c.Evidence {
			if !bytes.HasSuffix([]byte(ref.Path), []byte(".a-gate.json")) {
				continue
			}
			old, err := readRegular(root, ref.Path)
			if err != nil {
				t.Fatal(err)
			}
			var previous i18n.LocaleSurfaceReviewAGate
			if err := StrictJSON(old, &previous); err != nil {
				t.Fatal(err)
			}
			if previous.Inputs.LanguagesConfigSHA256 == digest(languages) {
				t.Fatal("fixture lacks historical registry drift")
			}
		}
	}
	plan, err := PlanLocaleLanguage(root, l.Locale)
	if err != nil {
		t.Fatal(err)
	}
	contexts, err := i18n.LocaleLanguageContexts(root, l.Locale, cat)
	if err != nil {
		t.Fatal(err)
	}
	tour, pending := 0, 0
	for _, work := range plan.Work {
		if work.Package == "tour-v1" {
			tour++
			if work.State != "carry" {
				t.Fatalf("Tour reopened: %+v", work)
			}
		} else if work.State == "generation-required" {
			pending++
		}
	}
	if tour != len(contexts) || pending == 0 {
		t.Fatalf("tour=%d contexts=%d pending=%d", tour, len(contexts), pending)
	}
	if err := verifyCarriedLocale(root, g, l); err != nil {
		t.Fatal(err)
	}
	structured, err := PlanStructuredAssets(root, l.Locale)
	if err != nil {
		t.Fatal(err)
	}
	for _, asset := range structured.Assets {
		if asset.Package == "tour-v1" && asset.State != "carry" {
			t.Fatalf("structured planner reopened current Tour: %+v", asset)
		}
	}
	for _, path := range structured.Expected {
		if path == "internal/tour/ui/zh-CN.json" || path == "locales/zh-CN/article-metadata.json" {
			t.Fatal("checkpoint carry leaked into generation exact-set")
		}
	}
	for _, pair := range []struct {
		path   string
		before []byte
	}{{LocalePath(l.Locale), closure}, {base + ".md", md}, {base + ".a-gate.json", gateBytes}} {
		after, _ := readRegular(root, pair.path)
		if !bytes.Equal(pair.before, after) {
			t.Fatalf("evidence changed: %s", pair.path)
		}
	}
	t.Logf("Tour %d scope carry; genuine pending=%d", tour, pending)
}

// Rebind only a synthetic copied gate, so failures exercise independent machine
// authority rather than merely making Surface input hashes stale. Never record
// or alter any real review evidence.
func rebindCheckpointFixtureGate(t *testing.T, root string, cat *i18n.Catalog, path string) {
	t.Helper()
	b, err := readRegular(root, path)
	if err != nil {
		t.Fatal(err)
	}
	var gate i18n.LocaleSurfaceReviewAGate
	if err := StrictJSON(b, &gate); err != nil {
		t.Fatal(err)
	}
	gate.Inputs, err = i18n.CurrentLocaleSurfaceReviewAInputs(root, "zh-CN", cat)
	if err != nil {
		t.Fatal(err)
	}
	b, _ = json.Marshal(gate)
	writeFixture(t, root, path, b)
}

func TestTourCurrentCheckpointIndependentAuthoritiesFailClosed(t *testing.T) {
	root, g, cat, l := currentTourCheckpointFixture(t)
	// A missing fixture dependency must not make every rejection case pass.
	if _, current, err := currentTourCheckpoint(root, g, cat, l); err != nil || !current {
		t.Fatalf("complete fixture checkpoint: current=%v %v", current, err)
	}
	gate, err := i18n.CurrentExactLocaleSurfaceReviewAGate(root, l.Locale, cat)
	if err != nil {
		t.Fatal(err)
	}
	gatePath := "data/locale-surface-reviews/zh-CN/" + gate.ReviewID + ".a-gate.json"
	gateOriginal, _ := readRegular(root, gatePath)
	tests := []struct {
		name, path string
		rebind     bool
	}{
		{"stale-Surface", "internal/tour/ui/zh-CN.json", false},
		{"canonical", "locales/zh-CN/candidates/concurrency-1.article", true},
		{"TU-finalization", "data/quality-check-snapshots/zh-CN/qc-008/finalization.json", false},
		{"original-finalization", "data/quality-check-snapshots/zh-CN/qc-007/finalization.json", false},
		{"glossary-Review", "data/unified-glossary-reviews/zh-CN/20261009-zh-CN-glossary-review-006.review.json", false},
		{"Course-SEO", "locales/zh-CN/course-metadata.json", true},
		{"historical-closure", LocalePath(l.Locale), false},
		{"recovery", "data/promotion-recoveries/zh-CN/569329ec1bbaf1a68adf74e2a0924a3565c52a155f8fa6b514e54b297d5ac903.json", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			original, err := readRegular(root, test.path)
			if err != nil {
				t.Fatal(err)
			}
			changed := append(append([]byte(nil), original...), []byte("\n{}")...)
			if test.name == "canonical" {
				changed = append(append([]byte(nil), original...), []byte("\nmanual edit\n")...)
			}
			if test.name == "Course-SEO" {
				changed = bytes.Replace(original, []byte(`"source_sha256": "`), []byte(`"source_sha256": "0`), 1)
			}
			writeFixture(t, root, test.path, changed)
			defer writeFixture(t, root, test.path, original)
			defer writeFixture(t, root, gatePath, gateOriginal)
			if test.rebind {
				rebindCheckpointFixtureGate(t, root, cat, gatePath)
			}
			_, current, err := currentTourCheckpoint(root, g, cat, l)
			if err == nil && current {
				t.Fatal("invalid independent authority accepted")
			}
			if test.name == "historical-closure" || test.name == "stale-Surface" {
				t.Logf("tampered %s: current=%v error=%v", test.path, current, err)
			}
		})
	}
	t.Run("replaced-historical-closure", func(t *testing.T) {
		path := LocalePath(l.Locale)
		original, err := readRegular(root, path)
		if err != nil {
			t.Fatal(err)
		}
		defer writeFixture(t, root, path, original)
		// Valid JSON and valid sparse authority still cannot replace the caller's
		// historical completion while its decoded object remains in memory.
		b, err := Encode(Locale{Schema: l.Schema, Locale: l.Locale, Packages: []Completion{}})
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, path, b)
		if _, current, err := currentTourCheckpoint(root, g, cat, l); err == nil && current {
			t.Fatal("replaced historical closure accepted")
		}
	})
	for _, invalid := range []string{"failed", "wrong-locale", "wrong-schema", "duplicate", "missing"} {
		t.Run(invalid, func(t *testing.T) {
			if invalid == "missing" {
				path := filepath.Join(root, gatePath)
				if err := os.Rename(path, path+".isolated-backup"); err != nil {
					t.Fatal(err)
				}
				defer func() {
					if err := os.Rename(path+".isolated-backup", path); err != nil {
						t.Fatal(err)
					}
				}()
				if _, current, err := currentTourCheckpoint(root, g, cat, l); err == nil && current {
					t.Fatal("missing current Surface accepted")
				}
				return
			}
			var v i18n.LocaleSurfaceReviewAGate
			if err := StrictJSON(gateOriginal, &v); err != nil {
				t.Fatal(err)
			}
			switch invalid {
			case "failed":
				v.Decision = "failed"
			case "wrong-locale":
				v.Locale = "fr-FR"
			case "wrong-schema":
				v.SchemaVersion = 5
			}
			b, _ := json.Marshal(v)
			if invalid == "duplicate" {
				b = bytes.Replace(b, []byte(`"locale":"zh-CN"`), []byte(`"locale":"zh-CN","locale":"zh-CN"`), 1)
			}
			writeFixture(t, root, gatePath, b)
			defer writeFixture(t, root, gatePath, gateOriginal)
			if _, current, err := currentTourCheckpoint(root, g, cat, l); err == nil && current {
				t.Fatal("invalid Surface accepted")
			}
		})
	}
	t.Run("malformed-historical-root", func(t *testing.T) {
		bad := l
		bad.Packages = append([]Completion(nil), l.Packages...)
		bad.Packages[0].EvidenceIdentity = digest([]byte("tampered identity"))
		if _, current, err := currentTourCheckpoint(root, g, cat, bad); err == nil || current {
			t.Fatal("malformed historical completion accepted")
		}
	})
}
