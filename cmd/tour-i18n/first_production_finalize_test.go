package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func finalizeFixture(t *testing.T) (string, *i18n.Catalog, string, string, string) {
	t.Helper()
	root := t.TempDir()
	for path, body := range map[string]string{
		"internal/tour/ui/en.json": "en", "internal/tour/ui/zz-ZZ.json": "target", "locales/zz-ZZ/glossary.yaml": "g", "locales/zz-ZZ/article-metadata.json": "a", "locales/zz-ZZ/course-metadata.json": "c", "internal/tour/languages.go": "l", "internal/tour/project.go": "p", "internal/tour/seo.go": "s",
		"production/identity.json": "{\n  \"locales\": [\n    {\"locale\": \"other-AA\", \"production_hostname\": \"other.example\", \"production_state\": \"live\"},\n    {\n      \"locale\": \"zz-ZZ\",\n      \"production_hostname\": \"zz.example\",\n      \"production_state\": \"first-production\"\n    }\n  ]\n}\n",
	} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	catalog := &i18n.Catalog{Pages: []i18n.Page{{ID: "lesson/1", Source: []byte("source")}}}
	if _, _, err := i18n.RecordLocaleSurfaceReviewA(root, "zz-ZZ", "review-1", "reviewer", catalog); err != nil {
		t.Fatal(err)
	}
	evidence := filepath.Join(root, "data", "locale-surface-reviews", "zz-ZZ", "review-1.md")
	if err := os.WriteFile(evidence, []byte("# Evidence\n\n"+finalizationPlaceholder+"\n"), 0644); err != nil {
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
	if err := finalizeFirstProduction(root, catalog, release, "review-1", strings.NewReader("wrong\n"), ioDiscard{}, true, noIdentityValidation); err == nil {
		t.Fatal("wrong token accepted")
	}
	if err := finalizeFirstProduction(root, catalog, release, "review-1", strings.NewReader("VISUAL-PASS\n"), ioDiscard{}, false, noIdentityValidation); err == nil {
		t.Fatal("non-TTY accepted")
	}
	if err := finalizeFirstProduction(root, catalog, release, "review-1", strings.NewReader("VISUAL-PASS\n"), ioDiscard{}, true, noIdentityValidation); err != nil {
		t.Fatal(err)
	}
	identity, _ := os.ReadFile(filepath.Join(root, "production", "identity.json"))
	if !strings.Contains(string(identity), `"production_state": "live"`) || !strings.Contains(string(identity), `"locale": "other-AA", "production_hostname": "other.example", "production_state": "live"`) {
		t.Fatal("identity transition was not exact")
	}
	result, _ := os.ReadFile(evidence)
	if !strings.Contains(string(result), "maintainer confirmation") || strings.Contains(string(result), "`PENDING`") {
		t.Fatal("evidence was not finalized")
	}
	if err := finalizeFirstProduction(root, catalog, release, "review-1", strings.NewReader("VISUAL-PASS\n"), ioDiscard{}, true, noIdentityValidation); err == nil {
		t.Fatal("live locale finalized twice")
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
	if !bytes.Equal(oldEvidence, newEvidence) || !bytes.Equal(oldIdentity, newIdentity) {
		t.Fatal("validation failure left partial finalization")
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
