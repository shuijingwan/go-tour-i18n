package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/tour"
)

func finalizeFixture(t *testing.T) (string, *i18n.Catalog, string, string, string) {
	t.Helper()
	root := t.TempDir()
	for path, body := range map[string]string{
		"internal/tour/ui/en.json": "en", "internal/tour/ui/zz-ZZ.json": "target", "locales/zz-ZZ/glossary.yaml": "g", "locales/zz-ZZ/article-metadata.json": "a", "locales/zz-ZZ/course-metadata.json": "c", "internal/tour/languages.go": "l", "internal/tour/project.go": "p", "internal/tour/seo.go": "s",
		"production/identity.json":       "{\n  \"locales\": [\n    {\"locale\": \"other-AA\", \"production_hostname\": \"other.example\", \"production_public_url\": \"https://other.example/\", \"production_state\": \"first-production\"},\n    {\n      \"locale\": \"zz-ZZ\",\n      \"production_hostname\": \"zz.example\",\n      \"production_public_url\": \"https://zz.example/\",\n      \"production_state\": \"first-production\"\n    }\n  ]\n}\n",
		"scripts/production-identity.py": "#!/usr/bin/env python3\n",
		"README.md":                      "# README\n\n<!-- live-locales:start -->\nold\n<!-- live-locales:end -->\n",
	} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	originalProjector := projectFinalizationREADME
	projectFinalizationREADME = func(projectRoot string, identity []byte) ([]byte, error) {
		readme, err := os.ReadFile(filepath.Join(projectRoot, "README.md"))
		if err != nil {
			return nil, err
		}
		return projectLiveLocales(readme, identity, []tour.LanguageLink{{Locale: "zz-ZZ", EnglishName: "Test", Autonym: "Test language", URL: "https://zz.example/"}})
	}
	t.Cleanup(func() { projectFinalizationREADME = originalProjector })
	catalog := &i18n.Catalog{Pages: []i18n.Page{{ID: "lesson/1", Source: []byte("source")}}}
	evidence := filepath.Join(root, "data", "locale-surface-reviews", "zz-ZZ", "review-1.md")
	if err := os.MkdirAll(filepath.Dir(evidence), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(evidence, []byte("# Evidence\n\n"+finalizationPlaceholder+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.RecordLocaleSurfaceReviewA(root, "zz-ZZ", "review-1", "reviewer", catalog); err != nil {
		t.Fatal(err)
	}
	release := filepath.Join(root, "release-parent", "go-tour-release-20260905-zz-ZZ-a1b2c3d4")
	if err := os.MkdirAll(release, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(release, "release.json"), []byte(`{"locale":"zz-ZZ"}`), 0644); err != nil {
		t.Fatal(err)
	}
	receipt := filepath.Join(filepath.Dir(release), filepath.Base(release)+".first-production-receipt.json")
	body := `{"schema":"go-tour-i18n/first-production-receipt/v1","locale":"zz-ZZ","hostname":"zz.example","release":"20260905-zz-ZZ-a1b2c3d4","result":"passed","stages":{"public-machine":{"result":"PASS"},"browser":{"result":"PASS"}}}`
	if err := os.WriteFile(receipt, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return root, catalog, release, receipt, evidence
}

func noIdentityValidation(string, string) error { return nil }

func recordCurrentGate(t *testing.T, root string, catalog *i18n.Catalog, reviewID string) string {
	t.Helper()
	evidence := filepath.Join(root, "data", "locale-surface-reviews", "zz-ZZ", reviewID+".md")
	if err := os.WriteFile(evidence, []byte("# Evidence\n\n"+finalizationPlaceholder+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.RecordLocaleSurfaceReviewA(root, "zz-ZZ", reviewID, "reviewer", catalog); err != nil {
		t.Fatal(err)
	}
	return evidence
}

func TestFinalizationPlaceholderValidationFailsClosed(t *testing.T) {
	for name, evidence := range map[string]string{
		"valid":              finalizationPlaceholder,
		"missing":            "# Evidence\n",
		"duplicate complete": finalizationPlaceholder + "\n" + finalizationPlaceholder,
		"duplicate start":    finalizationPlaceholder + "\n<!-- first-production-finalization:start -->",
		"duplicate end":      finalizationPlaceholder + "\n<!-- first-production-finalization:end -->",
		"modified":           strings.Replace(finalizationPlaceholder, "`PENDING`", "`pending`", 1),
		"finalized":          renderFinalization(firstProductionReceipt{Locale: "zz-ZZ", Hostname: "zz.example", Release: "r"}),
	} {
		t.Run(name, func(t *testing.T) {
			err := validateFinalizationPlaceholder([]byte(evidence))
			if name == "valid" && err != nil {
				t.Fatalf("valid placeholder rejected: %v", err)
			}
			if name != "valid" && err == nil {
				t.Fatal("invalid placeholder accepted")
			}
		})
	}
}

func TestFirstProductionEvidencePreflight(t *testing.T) {
	root, catalog, release, _, evidence := finalizeFixture(t)
	if err := firstProductionEvidencePreflightCommand(root, catalog, []string{"--release-dir", release}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(evidence, []byte("# missing\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := firstProductionEvidencePreflightCommand(root, catalog, []string{"--release-dir", release}); err == nil {
		t.Fatal("invalid evidence preflight accepted")
	}
}

func TestFirstProductionEvidencePreflightRejectsAmbiguousCurrentGates(t *testing.T) {
	root, catalog, release, _, _ := finalizeFixture(t)
	recordCurrentGate(t, root, catalog, "review-2")
	if err := i18n.RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err != nil {
		t.Fatalf("ordinary current-gate requirement changed: %v", err)
	}
	if err := firstProductionEvidencePreflightCommand(root, catalog, []string{"--release-dir", release}); err == nil {
		t.Fatal("ambiguous current A gates were accepted")
	}
}

func TestFirstProductionEvidencePreflightSelectsOnlyCurrentGate(t *testing.T) {
	root, catalog, release, _, _ := finalizeFixture(t)
	if err := os.WriteFile(filepath.Join(root, "internal", "tour", "seo.go"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	recordCurrentGate(t, root, catalog, "review-2")
	if err := firstProductionEvidencePreflightCommand(root, catalog, []string{"--release-dir", release}); err != nil {
		t.Fatalf("one current gate after stale historical gate was rejected: %v", err)
	}
}

func TestRecordARequiresPlaceholderOnlyForFirstProduction(t *testing.T) {
	root, catalog, _, _, evidence := finalizeFixture(t)
	if err := os.Remove(filepath.Join(root, "data", "locale-surface-reviews", "zz-ZZ", "review-1.a-gate.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(evidence, []byte("# missing\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := recordLocaleSurfaceReviewACommand(root, catalog, []string{"--locale", "zz-ZZ", "--review-id", "review-1", "--reviewer", "reviewer"}); err == nil {
		t.Fatal("record-a accepted missing first-production placeholder")
	}
	identityPath := filepath.Join(root, "production", "identity.json")
	identity, err := os.ReadFile(identityPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(identityPath, []byte(strings.Replace(string(identity), `"production_state": "first-production"`, `"production_state": "live"`, 1)), 0644); err != nil {
		t.Fatal(err)
	}
	if err := recordLocaleSurfaceReviewACommand(root, catalog, []string{"--locale", "zz-ZZ", "--review-id", "review-1", "--reviewer", "reviewer"}); err != nil {
		t.Fatalf("live record-a was incorrectly blocked: %v", err)
	}
}

func TestFirstProductionFinalizeFailsClosedBeforeHumanGate(t *testing.T) {
	for name, mutate := range map[string]func(*testing.T, string, string, string){
		"missing receipt":   func(t *testing.T, _, receipt, _ string) { os.Remove(receipt) },
		"malformed receipt": func(t *testing.T, _, receipt, _ string) { os.WriteFile(receipt, []byte("{"), 0644) },
		"receipt mismatch": func(t *testing.T, _, receipt, _ string) {
			os.WriteFile(receipt, []byte(`{"schema":"go-tour-i18n/first-production-receipt/v1","locale":"other-AA","hostname":"zz.example","release":"20260905-zz-ZZ-a1b2c3d4","result":"passed","stages":{"public-machine":{"result":"PASS"},"browser":{"result":"PASS"}}}`), 0644)
		},
		"receipt hostname mismatch": func(t *testing.T, _, receipt, _ string) {
			os.WriteFile(receipt, []byte(`{"schema":"go-tour-i18n/first-production-receipt/v1","locale":"zz-ZZ","hostname":"wrong.example","release":"20260905-zz-ZZ-a1b2c3d4","result":"passed","stages":{"public-machine":{"result":"PASS"},"browser":{"result":"PASS"}}}`), 0644)
		},
		"missing public": func(t *testing.T, _, receipt, _ string) {
			os.WriteFile(receipt, []byte(`{"schema":"go-tour-i18n/first-production-receipt/v1","locale":"zz-ZZ","hostname":"zz.example","release":"20260905-zz-ZZ-a1b2c3d4","result":"passed","stages":{"browser":{"result":"PASS"}}}`), 0644)
		},
		"missing browser": func(t *testing.T, _, receipt, _ string) {
			os.WriteFile(receipt, []byte(`{"schema":"go-tour-i18n/first-production-receipt/v1","locale":"zz-ZZ","hostname":"zz.example","release":"20260905-zz-ZZ-a1b2c3d4","result":"passed","stages":{"public-machine":{"result":"PASS"}}}`), 0644)
		},
		"bad marker": func(t *testing.T, _, _, evidence string) { os.WriteFile(evidence, []byte("# Evidence\n"), 0644) },
		"duplicate marker": func(t *testing.T, _, _, evidence string) {
			os.WriteFile(evidence, []byte(finalizationPlaceholder+"\n"+finalizationPlaceholder), 0644)
		},
	} {
		t.Run(name, func(t *testing.T) {
			root, catalog, release, receipt, evidence := finalizeFixture(t)
			before, _ := os.ReadFile(filepath.Join(root, "production", "identity.json"))
			mutate(t, root, receipt, evidence)
			if err := finalizeFirstProduction(root, catalog, release, "review-1", strings.NewReader("VISUAL-PASS\n"), ioDiscard{}, true, noIdentityValidation); err == nil {
				t.Fatal("finalize unexpectedly passed")
			}
			after, _ := os.ReadFile(filepath.Join(root, "production", "identity.json"))
			if !bytes.Equal(before, after) {
				t.Fatal("identity mutated on failed preflight")
			}
		})
	}
}

func TestFirstProductionFinalizeHumanGateAndAtomicTransition(t *testing.T) {
	root, catalog, release, _, evidence := finalizeFixture(t)
	var output bytes.Buffer
	if err := finalizeFirstProduction(root, catalog, release, "review-1", strings.NewReader("wrong\n"), ioDiscard{}, true, noIdentityValidation); err == nil {
		t.Fatal("wrong token accepted")
	}
	if err := finalizeFirstProduction(root, catalog, release, "review-1", strings.NewReader("VISUAL-PASS\n"), ioDiscard{}, false, noIdentityValidation); err == nil {
		t.Fatal("non-TTY accepted")
	}
	if err := finalizeFirstProduction(root, catalog, release, "review-1", strings.NewReader("VISUAL-PASS\n"), &output, true, noIdentityValidation); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "FIRST PRODUCTION FINALIZATION: PASS (locale=zz-ZZ review_id=review-1 production_state=live)") {
		t.Fatal("missing finalization PASS summary")
	}
	identity, _ := os.ReadFile(filepath.Join(root, "production", "identity.json"))
	if !strings.Contains(string(identity), `"locale": "zz-ZZ"`) || !strings.Contains(string(identity), `"production_state": "live"`) || !strings.Contains(string(identity), `"locale": "other-AA", "production_hostname": "other.example", "production_public_url": "https://other.example/", "production_state": "first-production"`) {
		t.Fatal("identity transition was not exact")
	}
	result, _ := os.ReadFile(evidence)
	if !strings.Contains(string(result), "maintainer confirmation") || strings.Contains(string(result), "`PENDING`") {
		t.Fatal("evidence was not finalized")
	}
	readme, _ := os.ReadFile(filepath.Join(root, "README.md"))
	if !strings.Contains(string(readme), "[Test — Test language](https://zz.example/)") {
		t.Fatal("README was not projected from candidate live identity")
	}
	if err := i18n.RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err != nil {
		t.Fatalf("v2 A gate became stale after lifecycle-only finalization: %v", err)
	}
	if err := finalizeFirstProduction(root, catalog, release, "review-1", strings.NewReader("VISUAL-PASS\n"), ioDiscard{}, true, noIdentityValidation); err == nil {
		t.Fatal("live locale finalized twice")
	}
}

func TestProjectLiveLocales(t *testing.T) {
	const readme = "before\n<!-- live-locales:start -->\nold\n<!-- live-locales:end -->\nafter\n"
	registry := []tour.LanguageLink{
		{Locale: "pt-BR", EnglishName: "Brazilian Portuguese", Autonym: "Português (Brasil)", URL: "https://pt.example/"},
		{Locale: "en", EnglishName: "English", Autonym: "English", URL: "https://go.dev/tour/", Official: true},
		{Locale: "ja-JP", EnglishName: "Japanese", Autonym: "日本語", URL: "https://ja.example/"},
	}
	identity := []byte(`{"locales":[{"locale":"ja-JP","production_state":"live","production_public_url":"https://ja.example/"},{"locale":"pt-BR","production_state":"first-production","production_public_url":"https://pt.example/"}]}`)
	got, err := projectLiveLocales([]byte(readme), identity, registry)
	if err != nil {
		t.Fatal(err)
	}
	want := "before\n<!-- live-locales:start -->\n- [Japanese — 日本語](https://ja.example/)\n<!-- live-locales:end -->\nafter\n"
	if string(got) != want {
		t.Fatalf("projection = %q, want %q", got, want)
	}
	again, err := projectLiveLocales(got, identity, registry)
	if err != nil || !bytes.Equal(got, again) {
		t.Fatalf("projection is not idempotent: %v", err)
	}
}

func TestProjectLiveLocalesFailsClosed(t *testing.T) {
	registry := []tour.LanguageLink{{Locale: "ja-JP", EnglishName: "Japanese", Autonym: "日本語", URL: "https://ja.example/"}}
	identity := []byte(`{"locales":[{"locale":"ja-JP","production_state":"live","production_public_url":"https://wrong.example/"}]}`)
	for name, readme := range map[string]string{
		"missing marker":   "# README\n",
		"duplicate marker": "<!-- live-locales:start --><!-- live-locales:start --><!-- live-locales:end -->",
		"marker order":     "<!-- live-locales:end --><!-- live-locales:start -->",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := projectLiveLocales([]byte(readme), identity, registry); err == nil {
				t.Fatal("malformed README accepted")
			}
		})
	}
	valid := []byte("<!-- live-locales:start -->\n<!-- live-locales:end -->")
	if _, err := projectLiveLocales(valid, identity, registry); err == nil {
		t.Fatal("URL drift accepted")
	}
	missingRegistry := []byte(`{"locales":[{"locale":"ko-KR","production_state":"live","production_public_url":"https://ko.example/"}]}`)
	if _, err := projectLiveLocales(valid, missingRegistry, registry); err == nil {
		t.Fatal("missing registry entry accepted")
	}
	duplicate := []byte(`{"locales":[{"locale":"ja-JP","production_state":"live","production_public_url":"https://ja.example/"},{"locale":"ja-JP","production_state":"live","production_public_url":"https://ja.example/"}]}`)
	if _, err := projectLiveLocales(valid, duplicate, registry); err == nil {
		t.Fatal("duplicate identity accepted")
	}
	if _, err := projectLiveLocales(valid, []byte(`{`), registry); err == nil {
		t.Fatal("malformed identity accepted")
	}
	official := []tour.LanguageLink{{Locale: "ja-JP", EnglishName: "Japanese", Autonym: "日本語", URL: "https://ja.example/", Official: true}}
	if _, err := projectLiveLocales(valid, []byte(`{"locales":[{"locale":"ja-JP","production_state":"live","production_public_url":"https://ja.example/"}]}`), official); err == nil {
		t.Fatal("Official community locale accepted")
	}
}

func TestProjectRootREADMEMatchesCheckedInProjection(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	identity, err := os.ReadFile(filepath.Join(root, "production", "identity.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := projectRootREADME(root, identity)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("checked-in README live locale projection is stale")
	}
}

func TestFirstProductionFinalizeRejectsStaleGateAndValidationRollback(t *testing.T) {
	root, catalog, release, _, evidence := finalizeFixture(t)
	if err := os.WriteFile(filepath.Join(root, "internal", "tour", "seo.go"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := finalizeFirstProduction(root, catalog, release, "review-1", strings.NewReader("VISUAL-PASS\n"), ioDiscard{}, true, noIdentityValidation); err == nil {
		t.Fatal("stale A gate accepted")
	}
	root, catalog, release, _, evidence = finalizeFixture(t)
	oldEvidence, _ := os.ReadFile(evidence)
	oldIdentity, _ := os.ReadFile(filepath.Join(root, "production", "identity.json"))
	oldREADME, _ := os.ReadFile(filepath.Join(root, "README.md"))
	calls := 0
	validator := func(string, string) error {
		calls++
		if calls == 3 {
			return fmt.Errorf("forced validation failure")
		}
		return nil
	}
	if err := finalizeFirstProduction(root, catalog, release, "review-1", strings.NewReader("VISUAL-PASS\n"), ioDiscard{}, true, validator); err == nil {
		t.Fatal("post-write validation failure accepted")
	}
	newEvidence, _ := os.ReadFile(evidence)
	newIdentity, _ := os.ReadFile(filepath.Join(root, "production", "identity.json"))
	newREADME, _ := os.ReadFile(filepath.Join(root, "README.md"))
	if !bytes.Equal(oldEvidence, newEvidence) || !bytes.Equal(oldIdentity, newIdentity) || !bytes.Equal(oldREADME, newREADME) {
		t.Fatal("validation failure left partial finalization")
	}
}

func TestFirstProductionFinalizeRequiresNamedCurrentGateBeforeHumanPrompt(t *testing.T) {
	root, catalog, release, _, _ := finalizeFixture(t)
	if err := os.WriteFile(filepath.Join(root, "internal", "tour", "seo.go"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	recordCurrentGate(t, root, catalog, "review-2")
	var output bytes.Buffer
	if err := finalizeFirstProduction(root, catalog, release, "review-1", strings.NewReader("VISUAL-PASS\n"), &output, true, noIdentityValidation); err == nil {
		t.Fatal("finalizer accepted a stale named gate because another gate was current")
	}
	if output.Len() != 0 {
		t.Fatal("stale named gate reached the HUMAN prompt")
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
