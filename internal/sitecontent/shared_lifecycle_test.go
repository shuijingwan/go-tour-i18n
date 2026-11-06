package sitecontent

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestCurrentPageBatchSharedInputSelection(t *testing.T) {
	if os.Getenv("GO_TOUR_CURRENT_PAGE_REGRESSION") != "1" {
		t.Skip("explicit real worktree regression")
	}
	root := repositoryRoot(t)
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	docs, raw, err := WorkflowDocuments(root, g, "learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	current := map[string]Document{}
	for _, d := range docs {
		if d.StableIndex <= 30 {
			current[d.Path] = d
		}
	}
	chosen := map[string]string{}
	for _, suffix := range []string{"", "-recovery-8", "-recovery-2", "-recovery-1"} {
		base := filepath.Join(root, ".site-v2-staging", "zh-CN-learn-docs-page-batch-1"+suffix+"-generation")
		b, err := os.ReadFile(base + ".zip")
		if err != nil {
			t.Fatal(err)
		}
		files, err := i18n.ReadTransportBundle(b, 256, 32<<20)
		if err != nil {
			t.Fatal(err)
		}
		var m Batch
		if err := StrictJSON(files["manifest.json"], &m); err != nil {
			t.Fatal(err)
		}
		if err := i18n.ValidateTransportBundleInventory(files, m.Files, true); err != nil {
			t.Fatal(err)
		}
		ob, err := os.ReadFile(base + "-result.json")
		if err != nil {
			t.Fatal(err)
		}
		var o Output
		if err := StrictJSON(ob, &o); err != nil {
			t.Fatal(err)
		}
		if o.BatchSHA != m.Identity || len(o.Targets) != len(m.Selected) {
			t.Fatal("original invocation identity")
		}
		for _, d := range m.Documents {
			now, ok := current[d.Path]
			if !ok {
				t.Fatal("scope")
			}
			if !bytes.Equal(files["source/"+d.Path], raw[d.Path]) {
				t.Fatal("source changed")
			}
			if !reflect.DeepEqual(now.Units, d.Units) {
				continue
			}
			for _, target := range o.Targets {
				if target.ID != d.Units[0].ID {
					continue
				}
				a := PageAdapter{now, raw[d.Path]}
				candidate, err := a.Restore(target.Text)
				if err != nil {
					t.Log(d.Path, err)
					continue
				}
				glossary, err := i18n.LoadGlossary(root, "zh-CN")
				if err != nil {
					t.Fatal(err)
				}
				if err := a.Validate(candidate, glossary); err != nil {
					t.Log(d.Path, err)
					continue
				}
				if prior, exists := chosen[target.ID]; exists && prior != target.Text {
					t.Fatal("conflicting valid candidate", d.Path)
				}
				chosen[target.ID] = target.Text
			}
		}
	}
	if len(chosen) != 30 {
		t.Fatalf("current exact-input candidates: %d/30", len(chosen))
	}
	t.Log("current exact formal inputs and original candidate bytes: 30/30 PASS")
}

func TestCurrentPageBatchSharedProcessing(t *testing.T) {
	if os.Getenv("GO_TOUR_CURRENT_PAGE_REGRESSION") != "1" {
		t.Skip("explicit real worktree regression")
	}
	actual := repositoryRoot(t)
	root := siteFixture(t, "docs/RETRANSLATION_RUNBOOK.md")
	copyFile := func(p string) {
		b, err := os.ReadFile(filepath.Join(actual, p))
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, p, b)
	}
	copyFile("locales/zh-CN/glossary.yaml")
	for _, dir := range []string{"data/unified-glossary-reviews/zh-CN", "data/unified-glossary-reviews/zh-CN-bundles"} {
		if err := filepath.WalkDir(filepath.Join(actual, dir), func(p string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if !d.IsDir() {
				rel, err := filepath.Rel(actual, p)
				if err != nil {
					return err
				}
				copyFile(filepath.ToSlash(rel))
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	sources := []i18n.TranslationUnitGenerationOrigin{}
	for _, suffix := range []string{"", "-recovery-8", "-recovery-2", "-recovery-1"} {
		base := ".site-v2-staging/zh-CN-learn-docs-page-batch-1" + suffix + "-generation"
		copyFile(base + ".zip")
		copyFile(base + "-result.json")
		b, _ := ReferenceFile(root, base+".zip")
		r, _ := ReferenceFile(root, base+"-result.json")
		sources = append(sources, i18n.TranslationUnitGenerationOrigin{Bundle: i18n.GlossaryArchiveReference{Path: b.Path, SHA256: b.SHA256}, Result: i18n.GlossaryArchiveReference{Path: r.Path, SHA256: r.SHA256}})
	}
	r, err := LandArchivedPageGeneration(root, "zh-CN", 1, sources)
	if err != nil {
		t.Fatal(err)
	}
	if r.UnitCount != 30 || r.ValidationPassed != 30 || r.RestoreFailed+r.ValidationFailed != 0 {
		t.Fatal("real shared process", r)
	}
	if r.SchemaVersion != 2 {
		t.Fatal("created a parallel processing schema")
	}
	if _, err := os.Stat(filepath.Join(root, "data/site-workflow")); !os.IsNotExist(err) {
		t.Fatal("shared landing created Site-specific lifecycle evidence", err)
	}
	c, err := i18n.ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	current, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := i18n.HydrateCatalogSources(c, current); err != nil {
		t.Fatal(err)
	}
	ids, err := AttachPageTranslationUnits(root, c)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationUnitScope(ids); err != nil {
		t.Fatal(err)
	}
	var manifest i18n.RetranslationBatchManifest
	batchDir := filepath.Join(root, "data/retranslation-runs/zh-CN", r.BatchID)
	mb, err := os.ReadFile(filepath.Join(batchDir, "manifest.json"))
	if err != nil || StrictJSON(mb, &manifest) != nil {
		t.Fatal("shared manifest", err)
	}
	first := manifest.Units[0]
	if first.GenerationOrigin == nil {
		t.Fatal("missing original complete invocation binding")
	}
	u, err := c.Unit(first.UnitID)
	if err != nil {
		t.Fatal(err)
	}
	a := u.Adapter.(PageAdapter)
	rawResponse, err := os.ReadFile(filepath.Join(batchDir, "raw-responses", filepath.Base(first.InputPath)))
	if err != nil || a.VerifyGenerationOrigin(root, "zh-CN", *first.GenerationOrigin, rawResponse) != nil {
		t.Fatal("original exact-input binding", err)
	}
	a.Document.Units = append([]Unit(nil), a.Document.Units...)
	a.Document.Units[0].Source += " changed input"
	if err := a.VerifyGenerationOrigin(root, "zh-CN", *first.GenerationOrigin, rawResponse); err == nil {
		t.Fatal("changed formal input carried")
	}
	progress, err := i18n.BuildTranslationUnitProgress(root, c, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if progress.Passed != 30 || progress.Missing != 19 || progress.Stale != 0 {
		t.Fatal("real evidence verification", progress)
	}
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "synthetic-real-before-batch2"}); err == nil {
		t.Fatal("real batch1 prematurely entered QC")
	}
	t.Logf("真实结果通过共享 process：%d/30；第二批19页尚未生成；未执行正式质量审核", r.ValidationPassed)
}
