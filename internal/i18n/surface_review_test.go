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
	if gate.SchemaVersion != localeSurfaceReviewASchemaVersion || gate.Inputs.LanguageRegistryBaseline == nil || len(gate.Inputs.LanguageRegistryBaseline.Entries) != 3 || len(gate.Inputs.LanguageRegistryBaseline.Profiles) != 2 || gate.Stage != localeSurfaceReviewAStage || gate.Decision != "passed" || gate.Inputs.CourseMetadataSHA256 == "" || gate.Inputs.ProductionPublicIdentitySHA256 == "" || gate.Inputs.ProductionIdentitySHA256 != "" || !strings.HasSuffix(filepath.ToSlash(path), "review-1.a-gate.json") {
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

func TestLocaleSurfaceReviewAGateUniqueAndNamedCurrentQueries(t *testing.T) {
	root, catalog := surfaceReviewTestRoot(t)
	if _, _, err := RecordLocaleSurfaceReviewA(root, "zz-ZZ", "review-1", "reviewer", catalog); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireUniqueCurrentLocaleSurfaceReviewAGate(root, "zz-ZZ", catalog); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireCurrentLocaleSurfaceReviewAByReviewID(root, "zz-ZZ", "review-1", catalog); err != nil {
		t.Fatal(err)
	}
	if _, _, err := RecordLocaleSurfaceReviewA(root, "zz-ZZ", "review-2", "reviewer", catalog); err != nil {
		t.Fatal(err)
	}
	if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err != nil {
		t.Fatalf("ordinary RequireCurrentLocaleSurfaceReviewA rejected multiple current gates: %v", err)
	}
	if _, err := RequireUniqueCurrentLocaleSurfaceReviewAGate(root, "zz-ZZ", catalog); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("multiple current gates were not rejected as ambiguous: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "tour", "seo.go"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireCurrentLocaleSurfaceReviewAByReviewID(root, "zz-ZZ", "review-1", catalog); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("named stale gate accepted: %v", err)
	}
}

func TestLocaleSurfaceReviewAGateV3ProductionIdentityScope(t *testing.T) {
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
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(identityPath, []byte(mutate(string(original))), 0644); err != nil {
				t.Fatal(err)
			}
			if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err != nil {
				t.Fatalf("v3 gate became stale after %s: %v", name, err)
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
				t.Fatalf("v3 gate did not stale after %s: %v", name, err)
			}
			if err := os.WriteFile(identityPath, original, 0644); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLocaleSurfaceReviewAGateV3RejectsInvalidTargetProductionProfile(t *testing.T) {
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

func TestLocaleSurfaceReviewV1DoesNotBindCourseSourceDescriptionAuthority(t *testing.T) {
	root, catalog := surfaceReviewTestRoot(t)
	if _, _, err := RecordLocaleSurfaceReviewA(root, "zz-ZZ", "review-1", "reviewer", catalog); err != nil {
		t.Fatal(err)
	}
	writeCourseSourceDescriptionAsset(t, root, catalog, nil)
	writeCourseSourceDescriptionReview(t, root, catalog, "source-review-1")
	if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err != nil {
		t.Fatalf("schema v1 Surface Review gate was staled by new global source-description authority: %v", err)
	}
	writeCourseSourceDescriptionAsset(t, root, catalog, map[string]string{catalog.Pages[0].ID: "changed"})
	if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err != nil {
		t.Fatalf("schema v1 Surface Review gate was staled by changed global source-description asset: %v", err)
	}
}

func TestLocaleSurfaceReviewV2BindsCourseSourceDescriptionAuthority(t *testing.T) {
	root, catalog := surfaceReviewTestRoot(t)
	if err := os.WriteFile(filepath.Join(root, "locales", "zz-ZZ", "course-metadata.json"), []byte(`{"schema_version":2}`), 0644); err != nil {
		t.Fatal(err)
	}
	writeCourseSourceDescriptionAsset(t, root, catalog, nil)
	writeCourseSourceDescriptionReview(t, root, catalog, "source-review-1")
	if _, _, err := RecordLocaleSurfaceReviewA(root, "zz-ZZ", "review-1", "reviewer", catalog); err != nil {
		t.Fatal(err)
	}
	if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err != nil {
		t.Fatalf("current schema v2 Surface Review gate rejected: %v", err)
	}
	writeCourseSourceDescriptionAsset(t, root, catalog, map[string]string{catalog.Pages[0].ID: "changed"})
	writeCourseSourceDescriptionReview(t, root, catalog, "source-review-2")
	if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("schema v2 Surface Review gate did not stale after source-description authority changed: %v", err)
	}
}

func surfaceReviewTestRoot(t *testing.T) (string, *Catalog) {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"internal/tour/ui/en.json": "en", "internal/tour/ui/zz-ZZ.json": "target", "locales/zz-ZZ/glossary.yaml": "glossary", "locales/zz-ZZ/article-metadata.json": "article", "locales/zz-ZZ/course-metadata.json": `{"schema_version":1}`, "internal/tour/languages.go": `package tour
type LanguageLink struct { Locale, EnglishName, Autonym, URL string; Official bool }
type localeProfile struct { TimeLabel string }
var languageRegistry = []LanguageLink{
{Locale:"en", EnglishName:"English", Autonym:"English", URL:"https://go.dev/tour/", Official:true},
{Locale:"other-AA", EnglishName:"Other", Autonym:"Other", URL:"https://other.example/"},
{Locale:"zz-ZZ", EnglishName:"Test", Autonym:"Test", URL:"https://zz.example/"},
}
var localeProfiles = map[string]localeProfile{"other-AA":{TimeLabel:"Other"}, "zz-ZZ":{TimeLabel:"Test"}}
func languagesFor(locale string) []LanguageLink { return languageRegistry }
`, "internal/tour/project.go": "project", "internal/tour/seo.go": "seo", "production/identity.json": `{"locales":[{"locale":"other-AA","production_hostname":"other.example","production_public_url":"https://other.example/","production_state":"live","loopback_port":4200},{"locale":"zz-ZZ","production_hostname":"zz.example","production_public_url":"https://zz.example/","production_state":"first-production","loopback_port":4100,"systemd_service":"go-tour-zz.service","cdn":"cloudflare"}]}`,
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

func TestLocaleSurfaceReviewAGateV3AllowsOnlyCompatibleRegistryAddition(t *testing.T) {
	root, catalog := surfaceReviewTestRoot(t)
	gate, _, err := RecordLocaleSurfaceReviewA(root, "zz-ZZ", "review-1", "reviewer", catalog)
	if err != nil {
		t.Fatal(err)
	}
	languagesPath := filepath.Join(root, "internal", "tour", "languages.go")
	languages, _ := os.ReadFile(languagesPath)
	added := strings.Replace(string(languages),
		`{Locale:"other-AA", EnglishName:"Other", Autonym:"Other", URL:"https://other.example/"},`,
		`{Locale:"new-AA", EnglishName:"New", Autonym:"New", URL:"https://new.example/"},
{Locale:"other-AA", EnglishName:"Other", Autonym:"Other", URL:"https://other.example/"},`, 1)
	added = strings.Replace(added,
		`var localeProfiles = map[string]localeProfile{`,
		`var newTimeLabel = "New"
var localeProfiles = map[string]localeProfile{"new-AA":{TimeLabel:newTimeLabel},`, 1)
	if err := os.WriteFile(languagesPath, []byte(added), 0644); err != nil {
		t.Fatal(err)
	}
	identityPath := filepath.Join(root, "production", "identity.json")
	identity, _ := os.ReadFile(identityPath)
	updatedIdentity := strings.Replace(string(identity), `{"locales":[`, `{"locales":[{"locale":"new-AA","production_hostname":"new.example","production_public_url":"https://new.example/"},`, 1)
	if err := os.WriteFile(identityPath, []byte(updatedIdentity), 0644); err != nil {
		t.Fatal(err)
	}
	current, err := currentLocaleSurfaceReviewAInputs(root, "zz-ZZ", catalog, localeSurfaceReviewASchemaVersion)
	if err != nil {
		t.Fatal(err)
	}
	if current.LanguagesConfigSHA256 == gate.Inputs.LanguagesConfigSHA256 {
		t.Fatal("registry addition did not change whole-file hash")
	}
	if current.LanguagesReviewProjectionSHA256 != gate.Inputs.LanguagesReviewProjectionSHA256 {
		t.Fatal("unrelated registry addition changed target review projection")
	}
	if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err != nil {
		t.Fatalf("compatible registry addition staled v3 language evidence: %v", err)
	}
}

func TestLocaleSurfaceReviewAGateV3StalesTargetLanguageProjection(t *testing.T) {
	for name, values := range map[string][2]string{
		"display name":      {`Autonym:"Test"`, `Autonym:"Changed"`},
		"runtime profile":   {`TimeLabel:"Test"`, `TimeLabel:"Changed"`},
		"runtime semantics": {`return languageRegistry`, `return append([]LanguageLink(nil), languageRegistry...)`},
	} {
		t.Run(name, func(t *testing.T) {
			root, catalog := surfaceReviewTestRoot(t)
			if _, _, err := RecordLocaleSurfaceReviewA(root, "zz-ZZ", "review-1", "reviewer", catalog); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "internal", "tour", "languages.go")
			data, _ := os.ReadFile(path)
			if err := os.WriteFile(path, []byte(strings.Replace(string(data), values[0], values[1], 1)), 0644); err != nil {
				t.Fatal(err)
			}
			if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err == nil || !strings.Contains(err.Error(), "stale") {
				t.Fatalf("target language change was accepted: %v", err)
			}
		})
	}
}

func TestLanguageRegistryCompatibilityFailsClosed(t *testing.T) {
	for name, mutations := range map[string]struct {
		identity  func(string) string
		languages func(string) string
	}{
		"duplicate registry locale": {languages: func(s string) string {
			return strings.Replace(s, `{Locale:"zz-ZZ"`, `{Locale:"other-AA"`, 1)
		}},
		"missing production identity": {identity: func(s string) string {
			return strings.Replace(s, `{"locale":"other-AA","production_hostname":"other.example","production_public_url":"https://other.example/","production_state":"live","loopback_port":4200},`, "", 1)
		}},
		"wrong registry URL": {languages: func(s string) string {
			return strings.Replace(s, `https://zz.example/`, `https://wrong.example/`, 1)
		}},
		"orphan runtime profile": {languages: func(s string) string {
			return strings.Replace(s, `var localeProfiles = map[string]localeProfile{`, `var localeProfiles = map[string]localeProfile{"orphan-AA":{TimeLabel:"Orphan"},`, 1)
		}},
		"untrusted identity URL": {identity: func(s string) string {
			return strings.Replace(s, `"production_public_url":"https://zz.example/"`, `"production_public_url":"http://zz.example/"`, 1)
		}},
	} {
		t.Run(name, func(t *testing.T) {
			root, _ := surfaceReviewTestRoot(t)
			if mutations.identity != nil {
				path := filepath.Join(root, "production", "identity.json")
				data, _ := os.ReadFile(path)
				if err := os.WriteFile(path, []byte(mutations.identity(string(data))), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if mutations.languages != nil {
				path := filepath.Join(root, "internal", "tour", "languages.go")
				data, _ := os.ReadFile(path)
				if err := os.WriteFile(path, []byte(mutations.languages(string(data))), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := validateCurrentLanguageRegistryCompatibility(root); err == nil {
				t.Fatal("invalid registry/identity accepted")
			}
		})
	}
}

func TestLocaleSurfaceReviewAGateV2DoesNotGainV3RegistryCompatibility(t *testing.T) {
	root, catalog := surfaceReviewTestRoot(t)
	inputs, err := currentLocaleSurfaceReviewAInputs(root, "zz-ZZ", catalog, localeSurfaceReviewASchemaVersionV2)
	if err != nil {
		t.Fatal(err)
	}
	gate := LocaleSurfaceReviewAGate{SchemaVersion: localeSurfaceReviewASchemaVersionV2, Locale: "zz-ZZ", ReviewID: "legacy-v2", Stage: localeSurfaceReviewAStage, Decision: "passed", Reviewer: "reviewer", Inputs: inputs}
	path, _ := LocaleSurfaceReviewAGatePath(root, "zz-ZZ", gate.ReviewID)
	data, _ := json.Marshal(gate)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	languagesPath := filepath.Join(root, "internal", "tour", "languages.go")
	languages, _ := os.ReadFile(languagesPath)
	if err := os.WriteFile(languagesPath, append(languages, []byte("\n// unrelated registry-era source change\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireCurrentLocaleSurfaceReviewAByReviewID(root, "zz-ZZ", gate.ReviewID, catalog); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("historical v2 receipt bypassed exact languages.go identity: %v", err)
	}
}

func TestCurrentRepositoryLanguageRegistryCompatibility(t *testing.T) {
	if err := validateCurrentLanguageRegistryCompatibility(repoRoot(t)); err != nil {
		t.Fatal(err)
	}
}

func TestLocaleSurfaceReviewAGateV2RegistryBaselineRequiresExplicitImmutableEvidence(t *testing.T) {
	root, catalog := surfaceReviewTestRoot(t)
	inputs, err := currentLocaleSurfaceReviewAInputs(root, "zz-ZZ", catalog, localeSurfaceReviewASchemaVersionV2)
	if err != nil {
		t.Fatal(err)
	}
	gate := LocaleSurfaceReviewAGate{
		SchemaVersion: localeSurfaceReviewASchemaVersionV2,
		Locale:        "zz-ZZ",
		ReviewID:      "legacy-v2-baseline",
		Stage:         localeSurfaceReviewAStage,
		Decision:      "passed",
		Reviewer:      "reviewer",
		Inputs:        inputs,
	}
	gatePath, _ := LocaleSurfaceReviewAGatePath(root, gate.Locale, gate.ReviewID)
	gateData, _ := json.Marshal(gate)
	if err := os.MkdirAll(filepath.Dir(gatePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(gatePath, gateData, 0644); err != nil {
		t.Fatal(err)
	}
	baseline, baselinePath, err := RecordLocaleSurfaceReviewRegistryBaseline(root, gate.Locale, gate.ReviewID, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if baseline.GateSHA256 != hashBytes(gateData) || baseline.LanguagesReviewProjectionSHA256 == "" {
		t.Fatalf("incomplete registry baseline: %+v", baseline)
	}
	if _, _, err := RecordLocaleSurfaceReviewRegistryBaseline(root, gate.Locale, gate.ReviewID, catalog); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("registry baseline overwrite accepted: %v", err)
	}

	languagesPath := filepath.Join(root, "internal", "tour", "languages.go")
	languages, _ := os.ReadFile(languagesPath)
	added := strings.Replace(string(languages),
		`{Locale:"other-AA", EnglishName:"Other", Autonym:"Other", URL:"https://other.example/"},`,
		`{Locale:"new-AA", EnglishName:"New", Autonym:"New", URL:"https://new.example/"},
{Locale:"other-AA", EnglishName:"Other", Autonym:"Other", URL:"https://other.example/"},`, 1)
	added = strings.Replace(added,
		`var localeProfiles = map[string]localeProfile{`,
		`var newTimeLabel = "New"
var localeProfiles = map[string]localeProfile{"new-AA":{TimeLabel:newTimeLabel},`, 1)
	if err := os.WriteFile(languagesPath, []byte(added), 0644); err != nil {
		t.Fatal(err)
	}
	identityPath := filepath.Join(root, "production", "identity.json")
	identity, _ := os.ReadFile(identityPath)
	updatedIdentity := strings.Replace(string(identity), `{"locales":[`, `{"locales":[{"locale":"new-AA","production_hostname":"new.example","production_public_url":"https://new.example/"},`, 1)
	if err := os.WriteFile(identityPath, []byte(updatedIdentity), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireCurrentLocaleSurfaceReviewAByReviewID(root, gate.Locale, gate.ReviewID, catalog); err != nil {
		t.Fatalf("explicit baseline did not preserve compatible v2 gate: %v", err)
	}

	baselineData, _ := os.ReadFile(baselinePath)
	var damaged LocaleSurfaceReviewRegistryBaseline
	if err := json.Unmarshal(baselineData, &damaged); err != nil {
		t.Fatal(err)
	}
	damaged.GateSHA256 = strings.Repeat("0", 64)
	damagedData, _ := json.Marshal(damaged)
	if err := os.WriteFile(baselinePath, damagedData, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireCurrentLocaleSurfaceReviewAByReviewID(root, gate.Locale, gate.ReviewID, catalog); err == nil || !strings.Contains(err.Error(), "invalid registry baseline") {
		t.Fatalf("damaged registry baseline accepted: %v", err)
	}
	if err := os.WriteFile(baselinePath, baselineData, 0644); err != nil {
		t.Fatal(err)
	}

	targetChanged := strings.Replace(added, `Autonym:"Test"`, `Autonym:"Changed"`, 1)
	if err := os.WriteFile(languagesPath, []byte(targetChanged), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireCurrentLocaleSurfaceReviewAByReviewID(root, gate.Locale, gate.ReviewID, catalog); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("target projection change was accepted through v2 baseline: %v", err)
	}
}

func TestCurrentRepositoryLanguageReviewProjections(t *testing.T) {
	root := repoRoot(t)
	parsed, err := parseSurfaceReviewLanguages(filepath.Join(root, "internal", "tour", "languages.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range parsed.entries {
		if entry.Locale == "en" {
			continue
		}
		if projection, err := currentLanguageReviewProjectionSHA256(root, entry.Locale); err != nil || projection == "" {
			t.Fatalf("locale %s projection=%q err=%v", entry.Locale, projection, err)
		}
	}
}

func TestLocaleSurfaceReviewAGateV3StalesLegalChangesToAnyExistingLocale(t *testing.T) {
	tests := map[string]struct {
		languages func(string) string
		identity  func(string) string
	}{
		"autonym": {
			languages: func(s string) string {
				return strings.Replace(s, `EnglishName:"Other", Autonym:"Other"`, `EnglishName:"Other", Autonym:"Changed"`, 1)
			},
		},
		"english name": {
			languages: func(s string) string {
				return strings.Replace(s, `EnglishName:"Other", Autonym:"Other"`, `EnglishName:"Other Language", Autonym:"Other"`, 1)
			},
		},
		"runtime profile": {
			languages: func(s string) string {
				return strings.Replace(s, `"other-AA":{TimeLabel:"Other"}`, `"other-AA":{TimeLabel:"Changed"}`, 1)
			},
		},
		"unrelated variable": {
			languages: func(s string) string {
				return strings.Replace(s, "var localeProfiles = map[string]localeProfile{", "var unrelated = \"not a locale profile dependency\"\nvar localeProfiles = map[string]localeProfile{", 1)
			},
		},
		"URL with synchronized production identity": {
			languages: func(s string) string {
				return strings.Replace(s, "https://other.example/", "https://other-new.example/", 1)
			},
			identity: func(s string) string {
				return strings.ReplaceAll(s, "other.example", "other-new.example")
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			root, catalog := surfaceReviewTestRoot(t)
			if _, _, err := RecordLocaleSurfaceReviewA(root, "zz-ZZ", "review-1", "reviewer", catalog); err != nil {
				t.Fatal(err)
			}
			languagesPath := filepath.Join(root, "internal", "tour", "languages.go")
			languages, _ := os.ReadFile(languagesPath)
			if err := os.WriteFile(languagesPath, []byte(test.languages(string(languages))), 0644); err != nil {
				t.Fatal(err)
			}
			if test.identity != nil {
				identityPath := filepath.Join(root, "production", "identity.json")
				identity, _ := os.ReadFile(identityPath)
				if err := os.WriteFile(identityPath, []byte(test.identity(string(identity))), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := validateCurrentLanguageRegistryCompatibility(root); err != nil {
				t.Fatalf("mutation should remain a legal current registry: %v", err)
			}
			if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err == nil || !strings.Contains(err.Error(), "stale") {
				t.Fatalf("legal change to an existing locale was accepted: %v", err)
			}
		})
	}
}

func recordSchemaV2GateForRegistryTest(t *testing.T, root string, catalog *Catalog, reviewID string) LocaleSurfaceReviewAGate {
	t.Helper()
	inputs, err := currentLocaleSurfaceReviewAInputs(root, "zz-ZZ", catalog, localeSurfaceReviewASchemaVersionV2)
	if err != nil {
		t.Fatal(err)
	}
	gate := LocaleSurfaceReviewAGate{
		SchemaVersion: localeSurfaceReviewASchemaVersionV2,
		Locale:        "zz-ZZ",
		ReviewID:      reviewID,
		Stage:         localeSurfaceReviewAStage,
		Decision:      "passed",
		Reviewer:      "reviewer",
		Inputs:        inputs,
	}
	path, _ := LocaleSurfaceReviewAGatePath(root, gate.Locale, gate.ReviewID)
	data, _ := json.Marshal(gate)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return gate
}

func TestLocaleSurfaceReviewAGateV2RegistryBaselineStalesExistingLocaleChanges(t *testing.T) {
	tests := map[string]struct {
		languages func(string) string
		identity  func(string) string
	}{
		"display name": {
			languages: func(s string) string {
				return strings.Replace(s, `EnglishName:"Other", Autonym:"Other"`, `EnglishName:"Other Language", Autonym:"Other"`, 1)
			},
		},
		"URL with synchronized production identity": {
			languages: func(s string) string {
				return strings.Replace(s, "https://other.example/", "https://other-new.example/", 1)
			},
			identity: func(s string) string {
				return strings.ReplaceAll(s, "other.example", "other-new.example")
			},
		},
		"runtime profile": {
			languages: func(s string) string {
				return strings.Replace(s, `"other-AA":{TimeLabel:"Other"}`, `"other-AA":{TimeLabel:"Changed"}`, 1)
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			root, catalog := surfaceReviewTestRoot(t)
			gate := recordSchemaV2GateForRegistryTest(t, root, catalog, "legacy-v2")
			baseline, _, err := RecordLocaleSurfaceReviewRegistryBaseline(root, gate.Locale, gate.ReviewID, catalog)
			if err != nil {
				t.Fatal(err)
			}
			if len(baseline.LanguageRegistryBaseline.Entries) != 3 || len(baseline.LanguageRegistryBaseline.Profiles) != 2 {
				t.Fatalf("incomplete structured baseline: %+v", baseline.LanguageRegistryBaseline)
			}
			languagesPath := filepath.Join(root, "internal", "tour", "languages.go")
			languages, _ := os.ReadFile(languagesPath)
			if err := os.WriteFile(languagesPath, []byte(test.languages(string(languages))), 0644); err != nil {
				t.Fatal(err)
			}
			if test.identity != nil {
				identityPath := filepath.Join(root, "production", "identity.json")
				identity, _ := os.ReadFile(identityPath)
				if err := os.WriteFile(identityPath, []byte(test.identity(string(identity))), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := validateCurrentLanguageRegistryCompatibility(root); err != nil {
				t.Fatalf("mutation should remain a legal current registry: %v", err)
			}
			if _, err := RequireCurrentLocaleSurfaceReviewAByReviewID(root, gate.Locale, gate.ReviewID, catalog); err == nil || !strings.Contains(err.Error(), "stale") {
				t.Fatalf("v2 baseline accepted change to an existing locale: %v", err)
			}
		})
	}
}
