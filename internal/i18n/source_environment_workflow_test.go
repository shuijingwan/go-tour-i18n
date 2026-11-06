package i18n

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// This is an actual registered future package attached to the full source
// Catalog, as in the CLI. The package has three complete synthetic Units; it
// never presents a partial historical Snapshot as a complete real package.
func sourceEnvironmentWorkflowCatalog(t *testing.T, root string, derive sourceCatalogDeriver) *Catalog {
	t.Helper()
	source, err := observedSourceCatalog(root, false, true, derive)
	if err != nil {
		t.Fatal(err)
	}
	c, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := HydrateCatalogSources(c, source); err != nil {
		t.Fatal(err)
	}
	d := TranslationPackageDescriptor{ID: "test-future-cards"}
	for i := 1; i <= 3; i++ {
		path := fmt.Sprintf("fixtures/future/card-%d.xml", i)
		b, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		var card futureCard
		if err := xml.Unmarshal(b, &card); err != nil {
			t.Fatal(err)
		}
		d.Units = append(d.Units, AdaptedTranslationUnit{Unit: TranslationUnit{ID: fmt.Sprintf("future/card-%d", i), Kind: UnitKindPage, SourcePath: path, Source: b, SourceSHA256: sum(b)}, Adapter: futureCardAdapter{card}, StableIndex: i, Route: fmt.Sprintf("/fixture/cards/%d", i), WorkingSet: fmt.Sprintf("initial-%d", i), WorkingSetLimit: 1})
	}
	if err := c.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestSourceAuthorityWorkflowProcesses(t *testing.T) {
	if root := os.Getenv("GO_TOUR_SOURCE_WORKFLOW_ROOT"); root != "" {
		stage := os.Getenv("GO_TOUR_SOURCE_WORKFLOW_STAGE")
		count := 0
		done := contentidentity.BeginMemoization()
		defer done()
		start := time.Now()
		c := sourceEnvironmentWorkflowCatalog(t, root, func(o sourceCatalogObservation) (*Catalog, error) { count++; return deriveSourceCatalog(o) })
		var payload []byte
		var identity string
		var err error
		switch stage {
		case "revision":
			var e *RetranslationExportResult
			e, err = ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, BatchID: "chatgpt-zh-CN-004", PreviousSnapshotID: "source-environment-reviewed", AllowReexport: true, UnitKind: UnitKindPage, UnitIDs: []string{"future/card-1"}})
			if err == nil {
				payload, err = os.ReadFile(filepath.Join(root, e.BatchPath, "manifest.json"))
			}
		case "bundle":
			var m GenerationBundleManifest
			payload, m, err = ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: "chatgpt-zh-CN-004"})
			identity = m.InputIdentitySHA256
			if err == nil {
				err = os.WriteFile(filepath.Join(root, "engineering/source-environment.zip"), payload, 0600)
			}
		case "current-check":
			payload, err = os.ReadFile(filepath.Join(root, "engineering/source-environment.zip"))
			if err == nil {
				var m GenerationBundleManifest
				m, err = VerifyCurrentTranslationUnitGenerationBundle(root, c, payload)
				identity = m.InputIdentitySHA256
			}
		}
		if err != nil {
			t.Fatal(stage, err)
		}
		fmt.Printf("SOURCE_WORKFLOW=%s\n", mustJSON(map[string]any{"stage": stage, "builds": count, "seconds": time.Since(start).Seconds(), "sha256": sum(payload), "identity": identity}))
		return
	}
	root, _, ids := futureContentFixture(t)
	source := sourceEnvironmentFixture(t)
	if err := filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(source, path)
		dest := filepath.Join(root, relative)
		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, b, 0644)
	}); err != nil {
		t.Fatal(err)
	}
	copyBundleAuthority(t, root, []string{"data/tour-pages.tsv", "data/tour-conditional-pages.tsv", "data/tour-examples.tsv"})
	os.MkdirAll(filepath.Join(root, "engineering"), 0755)
	c := sourceEnvironmentWorkflowCatalog(t, root, deriveSourceCatalog)
	for _, id := range ids {
		e, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitIDs: []string{id}})
		if err != nil {
			t.Fatal(err)
		}
		b, m, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
		if err != nil {
			t.Fatal(err)
		}
		_, files, err := readGenerationBundle(b)
		if err != nil {
			t.Fatal(err)
		}
		raw := files["formal/inputs/"+filepath.Base(m.ExpectedOutputs[0].BundlePath)]
		futureImport(t, root, c, b, map[string][]byte{id: raw})
		p, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID})
		if err != nil || p.ValidationPassed != 1 {
			t.Fatal(p, err)
		}
	}
	futureSnapshot(t, root, c, "source-environment-reviewed")
	futureRecord(t, root, c, "source-environment-reviewed", "", "B", "Synthetic independent QC revision finding.", ids[:1])
	for _, id := range ids[1:] {
		futureRecord(t, root, c, "source-environment-reviewed", "", "A", "", []string{id})
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var firstManifest, bundleDigest, bundleIdentity string
	for i, stage := range []string{"revision", "revision", "bundle", "current-check"} {
		if i == 1 {
			// Reset only the disposable, ungenerated fixture batch for the exact
			// same export comparison. The real batch062 is never accessed.
			if err := os.RemoveAll(filepath.Join(root, "data/retranslation-runs/zh-CN/chatgpt-zh-CN-004")); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command(os.Args[0], "-test.run=^TestSourceAuthorityWorkflowProcesses$", "-test.count=1")
		cmd.Env = append(os.Environ(), "GO_TOUR_SOURCE_WORKFLOW_ROOT="+root, "GO_TOUR_SOURCE_WORKFLOW_STAGE="+stage)
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(stage, string(b), err)
		}
		var row struct {
			Stage            string
			Builds           int
			Seconds          float64
			SHA256, Identity string
		}
		found := false
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "SOURCE_WORKFLOW=") {
				if json.Unmarshal([]byte(strings.TrimPrefix(line, "SOURCE_WORKFLOW=")), &row) != nil {
					t.Fatal(line)
				}
				found = true
			}
		}
		if !found {
			t.Fatal(string(b))
		}
		expected := 0
		if i == 0 {
			expected = 1
			firstManifest = row.SHA256
		}
		if row.Builds != expected {
			t.Fatal("source environment reconstructed across workflow commands", row)
		}
		if i == 1 && row.SHA256 != firstManifest {
			t.Fatal("warm export manifest identity changed")
		}
		if stage == "bundle" {
			bundleDigest, bundleIdentity = row.SHA256, row.Identity
		}
		if stage == "current-check" && (bundleDigest != row.SHA256 || bundleIdentity != row.Identity) {
			t.Fatal("independent Bundle current-check differs")
		}
		t.Logf("WORKFLOW phase=%d stage=%s source_builds=%d seconds=%.6f sha=%s", i, stage, row.Builds, row.Seconds, row.SHA256)
	}
	paths := map[string]string{
		"glossary":          "locales/zh-CN/glossary.yaml",
		"snapshot":          "data/quality-check-snapshots/zh-CN/source-environment-reviewed/manifest.json",
		"QC":                "data/quality-check-snapshots/zh-CN/source-environment-reviewed/quality-check-results.json",
		"registered-source": "fixtures/future/card-1.xml",
	}
	bundle, _ := os.ReadFile(filepath.Join(root, "engineering/source-environment.zip"))
	for name, path := range paths {
		t.Run("warm-authority-drift-"+name, func(t *testing.T) {
			absolute := filepath.Join(root, path)
			original, err := os.ReadFile(absolute)
			if err != nil {
				t.Fatal(err)
			}
			defer os.WriteFile(absolute, original, 0644)
			changed := append(bytes.Clone(original), ' ')
			if name == "QC" {
				var results QualityCheckResults
				if err := json.Unmarshal(original, &results); err != nil {
					t.Fatal(err)
				}
				results.Results[0].Rating = "A"
				changed, _ = json.Marshal(results)
			}
			if err := os.WriteFile(absolute, changed, 0644); err != nil {
				t.Fatal(err)
			}
			done := contentidentity.BeginMemoization()
			defer done()
			count := 0
			fresh := sourceEnvironmentWorkflowCatalog(t, root, func(o sourceCatalogObservation) (*Catalog, error) { count++; return deriveSourceCatalog(o) })
			if count != 0 {
				t.Fatal("unrelated authority entered source data key")
			}
			if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, fresh, bundle); err == nil {
				t.Fatal("source graph replaced current authority", name)
			}
		})
	}
	t.Run("warm-package-scope", func(t *testing.T) {
		done := contentidentity.BeginMemoization()
		defer done()
		fresh := sourceEnvironmentWorkflowCatalog(t, root, deriveSourceCatalog)
		d := fresh.TranslationPackages[0]
		// Remove the selected formal Unit from the registered inventory. A
		// smaller inventory retaining that Unit is not necessarily a stale
		// single-Unit Bundle under the existing generic adapter contract.
		d.Units = append([]AdaptedTranslationUnit(nil), d.Units[1:]...)
		d.AuthoritySHA256 = ""
		fresh.Adapted = nil
		fresh.TranslationPackages = nil
		fresh.TranslationScope = nil
		if err := fresh.RegisterTranslationPackage(d); err != nil {
			t.Fatal(err)
		}
		if err := fresh.UseTranslationPackage(d.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, fresh, bundle); err == nil {
			t.Fatal("changed registered package membership accepted")
		}
	})
}
