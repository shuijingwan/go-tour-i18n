package sitecontent

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

type assessmentFieldDifference struct {
	Field   string          `json:"field"`
	Frozen  json.RawMessage `json:"frozen_value"`
	Current json.RawMessage `json:"current_recomputed_value"`
}

// Struct declaration order is deterministic, and Reference values retain both
// path and SHA. This diagnostic never normalizes or changes assessment fields.
func assessmentFieldDifferences(frozen, current StructuredAssessment) []assessmentFieldDifference {
	before, after := reflect.ValueOf(frozen), reflect.ValueOf(current)
	out := []assessmentFieldDifference{}
	for i := 0; i < before.NumField(); i++ {
		if reflect.DeepEqual(before.Field(i).Interface(), after.Field(i).Interface()) {
			continue
		}
		a, _ := json.Marshal(before.Field(i).Interface())
		b, _ := json.Marshal(after.Field(i).Interface())
		out = append(out, assessmentFieldDifference{strings.Split(before.Type().Field(i).Tag.Get("json"), ",")[0], a, b})
	}
	return out
}

type structuredAssessmentDiagnostic struct {
	Asset               string                      `json:"asset"`
	Package             string                      `json:"package"`
	State               string                      `json:"frozen_state"`
	Frozen              StructuredAssessment        `json:"frozen_assessment"`
	Current             StructuredAssessment        `json:"current_recomputed_assessment"`
	Differences         []assessmentFieldDifference `json:"field_differences"`
	FrozenSourceSHA     string                      `json:"frozen_contributor_identity"`
	CurrentSourceSHA    string                      `json:"current_contributor_identity"`
	FrozenContributors  []i18n.LanguageContributor  `json:"frozen_contributors"`
	CurrentContributors []i18n.LanguageContributor  `json:"current_contributors"`
	FrozenDocument      *Document                   `json:"frozen_document"`
	CurrentDocument     *Document                   `json:"current_document"`
	FrozenRawSHA        string                      `json:"frozen_raw_source_sha256"`
	CurrentRawSHA       string                      `json:"current_raw_source_sha256"`
}

func TestStructuredAssessmentReadonlyFieldDiagnostic(t *testing.T) {
	if os.Getenv("GO_TOUR_STRUCTURED_ASSESSMENT_DIAGNOSTIC") != "1" {
		t.Skip("explicit read-only assessment diagnosis")
	}
	root := repositoryRoot(t)
	fingerprint := func() map[string]string {
		t.Helper()
		out := map[string]string{}
		for _, name := range []string{"data/site-shell", "data/glossary-source-corpus.json", "data/quality-check-snapshots", "data/quality-check-reopens", "data/translation-unit-reconciliations", "data/translation-authorizations", "data/locale-language", "data/site-workflow", "data/unified-glossary-reviews", "locales/zh-CN/glossary.yaml"} {
			if err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(name)), func(path string, e fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if e.IsDir() {
					return nil
				}
				b, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				out[path] = digest(b)
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		}
		return out
	}
	before := fingerprint()
	defer func() {
		if !reflect.DeepEqual(before, fingerprint()) {
			t.Error("FORMAL_ARTIFACT_MUTATION detected")
		}
	}()
	done := contentidentity.BeginMemoization()
	defer done()
	v := newClosureValidation(root)
	defer func() {
		if err := v.verify(); err != nil {
			t.Error(err)
		}
	}()
	if err := v.currentInputs("zh-CN"); err != nil {
		t.Fatal(err)
	}
	ref := Reference{"data/locale-language/zh-CN/structured-generation/7caebba33580782e5837719efa23a8d1cf4d60f51de84fa3e5c56c8816d25cb5.json", "3bdea5f48d19a48c69964d515d87117a095015d9aa6a33e2ad4837b59a10c8c3"}
	r, _, _, err := v.historicalStructuredGeneration(root, ref)
	if err != nil {
		t.Fatal("historical delta integrity", err)
	}
	var currentCorpus i18n.GlossarySourceCorpus
	if err := v.readReference(root, Reference{i18n.GlossaryCorpusPath, digest(mustDiagnosticRead(t, root, i18n.GlossaryCorpusPath))}, &currentCorpus); err != nil {
		t.Fatal(err)
	}
	currentGroups, err := structuredContributorGroups(currentCorpus.Contributors)
	if err != nil {
		t.Fatal(err)
	}
	glob, err := v.loadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	docs := map[string]Document{}
	raw := map[string][]byte{}
	packages := map[string]bool{}
	baselines := map[string]*structuredBaseline{}
	results := []structuredAssessmentDiagnostic{}
	first := ""
	for _, a := range r.Plan.Assets {
		if a.Assessment == nil {
			continue
		}
		x := a.Assessment
		b := baselines[x.Closure.Path]
		if b == nil {
			b, err = v.historicalStructuredClosure(root, x.Closure, map[string]bool{})
			if err != nil {
				t.Fatal(err)
			}
			if err := v.validateHistoricalIntegratedSurface(root, r.Plan.Locale, x.Surface, x.Closure); err != nil {
				t.Fatal(err)
			}
			b.Surface = x.Surface
			baselines[x.Closure.Path] = b
		}
		want, err := v.assessStructuredAsset(root, r.Plan, a, b)
		if err != nil {
			t.Fatal(a.ID, err)
		}
		if !packages[a.Package] {
			ds, bs, err := CurrentPackageDocuments(root, glob, a.Package)
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range ds {
				docs[d.Path] = d
			}
			for p, b := range bs {
				raw[p] = b
			}
			packages[a.Package] = true
		}
		var oldCorpus i18n.GlossarySourceCorpus
		if err := StrictJSON(b.Inputs[x.Generation.Path]["inputs/"+i18n.GlossaryCorpusPath], &oldCorpus); err != nil {
			t.Fatal(err)
		}
		oldGroups, err := structuredContributorGroups(oldCorpus.Contributors)
		if err != nil {
			t.Fatal(err)
		}
		item := structuredAssessmentDiagnostic{Asset: a.ID, Package: a.Package, State: a.State, Frozen: *x, Current: *want, Differences: assessmentFieldDifferences(*x, *want), FrozenSourceSHA: a.SourceSHA, FrozenContributors: oldGroups[a.ID], CurrentContributors: currentGroups[a.ID], FrozenDocument: a.Document}
		if cs := currentGroups[a.ID]; len(cs) > 0 {
			item.CurrentSourceSHA = identity(cs)
		}
		if a.Document != nil {
			item.FrozenRawSHA = digest(b.Inputs[x.Generation.Path]["source/"+a.Document.Path])
			if d, ok := docs[a.Document.Path]; ok {
				item.CurrentDocument = &d
				item.CurrentRawSHA = digest(raw[d.Path])
			}
		}
		if len(item.Differences) > 0 {
			diff, _ := Encode(item.Differences)
			t.Logf("ASSESSMENT_DIFF asset=%s fields=%s", a.ID, diff)
			if first == "" {
				first = a.ID
			}
		}
		results = append(results, item)
	}
	report, err := Encode(results)
	if err != nil {
		t.Fatal(err)
	}
	expectMatch := os.Getenv("GO_TOUR_STRUCTURED_ASSESSMENT_EXPECT_MATCH") == "1"
	output := "/tmp/go-tour-structured-assessment-diff.json"
	if expectMatch {
		output = "/tmp/go-tour-structured-assessment-current.json"
	}
	if err := os.WriteFile(output, report, 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("FIRST_MISMATCH_ASSET=%s ASSETS=%d REPORT=%s", first, len(results), output)
	if first == "" && !expectMatch {
		t.Fatal("reported failure was not reproduced")
	}
	if first != "" && expectMatch {
		t.Fatal("assessment mismatch remains", first)
	}
	if expectMatch {
		if _, _, err := v.checkStructuredGeneration(root, ref, "learn-docs-v1"); err != nil {
			t.Fatal("CURRENT_DELTA_GENERATION_VERIFIED=FAILED NEXT_BLOCKER=", err)
		}
		t.Log("CURRENT_DELTA_GENERATION_VERIFIED=PASS")
		page := Reference{"data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-015/finalization.json", "530374a7aa563c103f975940b480f998fbd315541c56438e21d2557e22275b1d"}
		baseline := Reference{"data/locale-language/zh-CN/structured-generation/3aa00f3907d55604a78a411e88fe0fb478e232dd5f80cbfea66ca9e80c53bc78.json", "e4c7a8b4e9fa94cf06422d9c1a376777faf3a9e43a03f0694a90c358cebd0413"}
		c, files, err := v.buildPackageClosure(root, "zh-CN", "learn-docs-v1", &page, []Reference{baseline, ref})
		if err != nil {
			t.Fatal("PACKAGE_CLOSURE_READONLY_EQUIVALENT=FAILED NEXT_BLOCKER=", err)
		}
		if len(c.Files) != 56 || len(files) != 56 {
			t.Fatal("non-exact Learn/Docs closure")
		}
		t.Log("PACKAGE_CLOSURE_READONLY_EQUIVALENT=PASS files=56; no artifact saved")
	}
}

func mustDiagnosticRead(t *testing.T, root, path string) []byte {
	t.Helper()
	b, err := readRegular(root, path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
