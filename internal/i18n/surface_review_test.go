package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocaleSurfaceReviewAGateFailsClosedAndStales(t *testing.T) {
	root, catalog := surfaceReviewTestRoot(t)
	if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err == nil || !strings.Contains(err.Error(), "gate missing") {
		t.Fatalf("missing gate error=%v", err)
	}
	gate, path, err := RecordLocaleSurfaceReviewA(root, "zz-ZZ", "review-1", "reviewer", catalog)
	if err != nil {
		t.Fatal(err)
	}
	if gate.SchemaVersion != localeSurfaceReviewASchemaVersion || gate.Stage != localeSurfaceReviewAStage || gate.Decision != "passed" || gate.Inputs.CourseMetadataSHA256 == "" || gate.Inputs.ProductionPublicIdentitySHA256 == "" || gate.Inputs.ProductionIdentitySHA256 != "" || !strings.HasSuffix(filepath.ToSlash(path), "review-1.a-gate.json") {
		t.Fatalf("recorded gate is incomplete: %+v path=%s", gate, path)
	}
	if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err != nil {
		t.Fatalf("current gate rejected: %v", err)
	}
	for _, path := range []string{
		"internal/tour/ui/en.json", "internal/tour/ui/zz-ZZ.json", "locales/zz-ZZ/glossary.yaml", "locales/zz-ZZ/article-metadata.json", "locales/zz-ZZ/course-metadata.json",
		"internal/tour/languages.go", "internal/tour/project.go", "internal/tour/seo.go",
	} {
		original, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), append(original, 'x'), 0644); err != nil {
			t.Fatal(err)
		}
		if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err == nil || !strings.Contains(err.Error(), "stale") {
			t.Fatalf("%s did not stale gate: %v", path, err)
		}
		if err := os.WriteFile(filepath.Join(root, path), original, 0644); err != nil {
			t.Fatal(err)
		}
	}
	changed := *catalog
	changed.Pages = append([]Page(nil), catalog.Pages...)
	changed.Pages[0].Source = []byte("changed source")
	if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", &changed); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("catalog/source change did not stale gate: %v", err)
	}
}

func TestLocaleSurfaceReviewAGateV2ProductionIdentityScope(t *testing.T) {
	root, catalog := surfaceReviewTestRoot(t)
	if _, _, err := RecordLocaleSurfaceReviewA(root, "zz-ZZ", "review-1", "reviewer", catalog); err != nil {
		t.Fatal(err)
	}
	identityPath := filepath.Join(root, "production", "identity.json")
	original, err := os.ReadFile(identityPath)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(string) string{
		"target lifecycle": func(s string) string {
			return strings.Replace(s, `"production_state":"first-production"`, `"production_state":"live"`, 1)
		},
		"target infra": func(s string) string { return strings.Replace(s, `"loopback_port":4100`, `"loopback_port":4101`, 1) },
		"target service": func(s string) string {
			return strings.Replace(s, `"systemd_service":"go-tour-zz.service"`, `"systemd_service":"go-tour-zz-next.service"`, 1)
		},
		"target CDN": func(s string) string { return strings.Replace(s, `"cdn":"cloudflare"`, `"cdn":"edgeone"`, 1) },
		"other lifecycle": func(s string) string {
			return strings.Replace(s, `"production_state":"live"`, `"production_state":"first-production"`, 1)
		},
		"other infra": func(s string) string { return strings.Replace(s, `"loopback_port":4200`, `"loopback_port":4201`, 1) },
		"unrelated new profile": func(s string) string {
			return strings.Replace(s, `]}`, `,{"locale":"new-AA","production_hostname":"new.example","production_public_url":"https://new.example/"}]}`, 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(identityPath, []byte(mutate(string(original))), 0644); err != nil {
				t.Fatal(err)
			}
			if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err != nil {
				t.Fatalf("v2 gate became stale after %s: %v", name, err)
			}
			if err := os.WriteFile(identityPath, original, 0644); err != nil {
				t.Fatal(err)
			}
		})
	}
	for name, mutate := range map[string]func(string) string{
		"target hostname": func(s string) string {
			return strings.Replace(s, `"production_hostname":"zz.example"`, `"production_hostname":"changed.example"`, 1)
		},
		"target public URL": func(s string) string {
			return strings.Replace(s, `"production_public_url":"https://zz.example/"`, `"production_public_url":"https://changed.example/"`, 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(identityPath, []byte(mutate(string(original))), 0644); err != nil {
				t.Fatal(err)
			}
			if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err == nil || !strings.Contains(err.Error(), "stale") {
				t.Fatalf("v2 gate did not stale after %s: %v", name, err)
			}
			if err := os.WriteFile(identityPath, original, 0644); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLocaleSurfaceReviewAGateV2RejectsInvalidTargetProductionProfile(t *testing.T) {
	for name, identity := range map[string]string{
		"malformed":     `{`,
		"missing":       `{"locales":[{"locale":"other-AA","production_hostname":"other.example","production_public_url":"https://other.example/"}]}`,
		"missing field": `{"locales":[{"locale":"zz-ZZ","production_hostname":"zz.example"}]}`,
		"duplicate":     `{"locales":[{"locale":"zz-ZZ","production_hostname":"zz.example","production_public_url":"https://zz.example/"},{"locale":"zz-ZZ","production_hostname":"two.example","production_public_url":"https://two.example/"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			root, catalog := surfaceReviewTestRoot(t)
			if _, _, err := RecordLocaleSurfaceReviewA(root, "zz-ZZ", "review-1", "reviewer", catalog); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "production", "identity.json"), []byte(identity), 0644); err != nil {
				t.Fatal(err)
			}
			if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err == nil {
				t.Fatal("invalid target production profile accepted")
			}
		})
	}
}

func TestLocaleSurfaceReviewAGateV1Compatibility(t *testing.T) {
	root, catalog := surfaceReviewTestRoot(t)
	inputs, err := currentLocaleSurfaceReviewAInputs(root, "zz-ZZ", catalog, localeSurfaceReviewASchemaVersionV1)
	if err != nil {
		t.Fatal(err)
	}
	gate := LocaleSurfaceReviewAGate{SchemaVersion: localeSurfaceReviewASchemaVersionV1, Locale: "zz-ZZ", ReviewID: "legacy", Stage: localeSurfaceReviewAStage, Decision: "passed", Reviewer: "reviewer", Inputs: inputs}
	path, err := LocaleSurfaceReviewAGatePath(root, "zz-ZZ", "legacy")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(gate)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err != nil {
		t.Fatalf("unchanged v1 gate rejected: %v", err)
	}
	identityPath := filepath.Join(root, "production", "identity.json")
	identity, _ := os.ReadFile(identityPath)
	if err := os.WriteFile(identityPath, append(identity, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("v1 gate did not retain whole-file identity semantics: %v", err)
	}
}

func TestLocaleSurfaceReviewAGateRejectsMalformedWrongLocaleAndDecision(t *testing.T) {
	for _, body := range []string{
		"{", `{"schema_version":1,"locale":"other","review_id":"r","stage":"locale-level-language-quality-review","decision":"passed","reviewer":"r"}`,
		`{"schema_version":2,"locale":"zz-ZZ","review_id":"r","stage":"locale-level-language-quality-review","decision":"failed","reviewer":"r"}`,
		`{"schema_version":99,"locale":"zz-ZZ","review_id":"r","stage":"locale-level-language-quality-review","decision":"passed","reviewer":"r"}`,
	} {
		root, catalog := surfaceReviewTestRoot(t)
		dir := filepath.Join(root, "data", "locale-surface-reviews", "zz-ZZ")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "r.a-gate.json"), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
		if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err == nil {
			t.Fatal("invalid gate accepted")
		}
	}
}

func surfaceReviewTestRoot(t *testing.T) (string, *Catalog) {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"internal/tour/ui/en.json": "en", "internal/tour/ui/zz-ZZ.json": "target", "locales/zz-ZZ/glossary.yaml": "glossary", "locales/zz-ZZ/article-metadata.json": "article", "locales/zz-ZZ/course-metadata.json": "course", "internal/tour/languages.go": "languages", "internal/tour/project.go": "project", "internal/tour/seo.go": "seo", "production/identity.json": `{"locales":[{"locale":"other-AA","production_hostname":"other.example","production_public_url":"https://other.example/","production_state":"live","loopback_port":4200},{"locale":"zz-ZZ","production_hostname":"zz.example","production_public_url":"https://zz.example/","production_state":"first-production","loopback_port":4100,"systemd_service":"go-tour-zz.service","cdn":"cloudflare"}]}`,
	}
	for path, text := range files {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root, &Catalog{Pages: []Page{{ID: "lesson/1", Source: []byte("source"), SourceSHA256: sum([]byte("source"))}}}
}
