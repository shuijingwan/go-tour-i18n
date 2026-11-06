package sitecontent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestUnifiedLearnDocsMembership(t *testing.T) {
	root := repositoryRoot(t)
	done := contentidentity.BeginMemoization()
	defer done()
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	c, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackages(root, c); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage("docs-entry-v1"); err == nil {
		t.Fatal("historical package scheduled")
	}
	if err := c.UseTranslationPackage("learn-docs-v1"); err != nil {
		t.Fatal(err)
	}
	if len(c.TranslationScope) != 51 {
		t.Fatal(len(c.TranslationScope))
	}
	if total, pages, examples, err := i18n.LocaleWorkflowUnitCounts(c); err != nil || total != 51 || pages != 51 || examples != 0 {
		t.Fatal("scoped status inventory", total, pages, examples, err)
	}
	native := *c
	native.TranslationScope = nil
	if total, pages, examples, err := i18n.LocaleWorkflowUnitCounts(&native); err != nil || total != 122 || pages != 103 || examples != 19 {
		t.Fatal("Tour status inventory changed", total, pages, examples, err)
	}
	docs, raw, err := WorkflowDocuments(root, g, "learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 51 || docs[49].Route != "/doc/" || docs[50].Route != "/doc/install" {
		t.Fatal("append-only order")
	}
	for i, d := range docs {
		if d.StableIndex != i+1 || c.TranslationScope[i] != d.Units[0].ID {
			t.Fatal("ordered identity", i)
		}
	}
	m, err := readPackageMembership(root)
	if err != nil {
		t.Fatal(err)
	}
	for i, ref := range m.Predecessors[0].Sources {
		if docs[i].Path != ref.Path {
			t.Fatal("original relative order")
		}
	}
	ids := append([]string{}, c.TranslationScope...)
	if err := c.RequireInitialTranslationWorkingSet(ids[:30]); err != nil {
		t.Fatal(err)
	}
	if err := c.RequireInitialTranslationWorkingSet(ids[30:]); err != nil {
		t.Fatal(err)
	}
	if err := c.RequireInitialTranslationWorkingSet(ids); err == nil {
		t.Fatal("initial Generation/QC combined two invocations")
	}
	sort.Strings(ids)
	plan, err := PlanPageWorkingSets(docs, raw, ids, nil)
	if err != nil || len(plan.Sets) != 2 || len(plan.Sets[0].Selected) != 30 || len(plan.Sets[1].Selected) != 21 {
		t.Fatal("30+21", err)
	}
	if _, _, err := WorkflowDocuments(root, g, "docs-entry-v1"); err == nil {
		t.Fatal("retired package planned")
	}
	if err := CheckUnifiedCorpus(root); err != nil {
		t.Fatal("membership changed terminology authority", err)
	}
	// Native Tour status/SEO scope stays separate; registry membership only
	// becomes lifecycle inventory on explicit complete package selection.
	if len(c.Pages) != 103 {
		t.Fatal("Tour inventory changed")
	}
}

func TestUnifiedLearnDocsCompositionProof(t *testing.T) {
	out := os.Getenv("GO_TOUR_UNIFIED_COMPOSITION_PROOF")
	if out == "" {
		t.Skip("explicit readonly real-authority proof required")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	c, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackages(root, c); err != nil {
		t.Fatal(err)
	}
	AttachIntegratedSurfaceAuthority(c)
	if err := c.UseTranslationPackage("learn-docs-v1"); err != nil {
		t.Fatal(err)
	}
	m, err := readPackageMembership(root)
	if err != nil {
		t.Fatal(err)
	}
	inputs := []i18n.TranslationFinalizationInput{}
	for i, snapshot := range []string{"learn-docs-v1-qc-021", "docs-entry-v1-qc-002"} {
		r, err := ReferenceFile(root, "data/quality-check-snapshots/zh-CN/"+snapshot+"/finalization.json")
		if err != nil {
			t.Fatal(err)
		}
		inputs = append(inputs, i18n.TranslationFinalizationInput{Package: m.Predecessors[i].Package, PackageAuthority: m.Predecessors[i].Authority, Finalization: i18n.GlossaryArchiveReference{Path: r.Path, SHA256: r.SHA256}})
	}
	p, files, err := i18n.ComposeFinalizedTranslationPackage(root, c, "zh-CN", inputs)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Units) != 51 || len(files) != 51 || p.Assessments[0].Compatible != 49 || p.Assessments[1].Compatible != 2 {
		t.Fatal("composition incomplete")
	}
	before, err := readRegular(root, PackageMembershipPath)
	if err != nil {
		t.Fatal(err)
	}
	again, err := readPackageMembership(root)
	if err != nil || !reflect.DeepEqual(m, again) {
		t.Fatal("membership changed")
	}
	b, _ := json.MarshalIndent(p, "", "  ")
	if err := os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "composition-preflight.json"), append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "membership-observed.json"), before, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestPackageMembershipRejectsDrift(t *testing.T) {
	root := repositoryRoot(t)
	done := contentidentity.BeginMemoization()
	defer done()
	base, err := loadSourceCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	m, err := readPackageMembership(root)
	if err != nil {
		t.Fatal(err)
	}
	baseline := t.TempDir()
	var current Reference
	b, err := readRegular(root, PackageMembershipPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := StrictJSON(b, &current); err != nil {
		t.Fatal(err)
	}
	paths := []string{PackageMembershipPath, current.Path, PageRegistryPath, m.Registry.Path, m.PreviousRegistry.Path}
	for _, p := range m.Predecessors {
		for _, r := range p.Sources {
			paths = append(paths, r.Path)
		}
	}
	for _, p := range paths {
		b, err := readRegular(root, p)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, baseline, p, b)
	}
	tests := []struct {
		name string
		edit func(*testing.T, string, *PackageMembership)
	}{
		{"wrong-descriptor", func(t *testing.T, r string, m *PackageMembership) {
			m.Predecessors[1].Authority = digest([]byte("wrong"))
		}},
		{"duplicate-predecessor", func(t *testing.T, r string, m *PackageMembership) { m.Predecessors[1] = m.Predecessors[0] }},
		{"duplicate-unit-source", func(t *testing.T, r string, m *PackageMembership) {
			m.Predecessors[0].Sources[1] = m.Predecessors[0].Sources[0]
		}},
		{"overlapping-source", func(t *testing.T, r string, m *PackageMembership) {
			m.Predecessors[1].Sources[0] = m.Predecessors[0].Sources[0]
		}},
		{"missing-50-of-51", func(t *testing.T, r string, m *PackageMembership) {
			m.Predecessors[1].Sources = m.Predecessors[1].Sources[:1]
		}},
		{"extra-52nd", func(t *testing.T, r string, m *PackageMembership) {
			m.Predecessors[1].Sources = append(m.Predecessors[1].Sources, m.Predecessors[0].Sources[0])
		}},
		{"source-sha", func(t *testing.T, r string, m *PackageMembership) {
			m.Predecessors[1].Sources[0].SHA256 = digest([]byte("wrong"))
		}},
		{"source-bytes", func(t *testing.T, r string, m *PackageMembership) {
			p := m.Predecessors[1].Sources[0].Path
			b, _ := readRegular(r, p)
			writeFixture(t, r, p, append(b, ' '))
		}},
		{"registry-order", func(t *testing.T, r string, m *PackageMembership) {
			var pages PageRegistry
			if err := readReference(r, m.Registry, &pages); err != nil {
				t.Fatal(err)
			}
			pages.Pages[49].Path, pages.Pages[50].Path = pages.Pages[50].Path, pages.Pages[49].Path
			sealPages(&pages)
			b, _ := Encode(pages)
			writeFixture(t, r, PageRegistryPath, b)
			writeFixture(t, r, m.Registry.Path, b)
			m.Registry.SHA256 = digest(b)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := t.TempDir()
			for _, p := range paths {
				b, _ := readRegular(baseline, p)
				writeFixture(t, r, p, b)
			}
			var copy PackageMembership
			b, _ := Encode(m)
			if err := StrictJSON(b, &copy); err != nil {
				t.Fatal(err)
			}
			tc.edit(t, r, &copy)
			copy.Identity = ""
			copy.Identity = identity(copy)
			b, _ = Encode(copy)
			name := "data/content-package-memberships/receipts/" + copy.Identity + ".json"
			writeFixture(t, r, name, b)
			pointer, _ := Encode(Reference{Path: name, SHA256: digest(b)})
			writeFixture(t, r, PackageMembershipPath, pointer)
			if _, err := applyPackageMembership(r, base); err == nil {
				t.Fatal("invalid membership accepted")
			}
		})
	}
}
