package sitecontent

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func pagePackageAdoptionFixture(t *testing.T) (string, Reference) {
	t.Helper()
	root := siteFixture(t, "current-authorization-fresh-fixture")
	real := repositoryRoot(t)
	m, err := loadSourceManifest(real)
	if err != nil {
		t.Fatal(err)
	}
	keep := map[string]bool{}
	for _, name := range m.Legacy.Paths {
		keep[adoptedSourcePath(name)] = true
	}
	for _, name := range []string{"_content/doc/index.html", "_content/doc/install.html", "_content/site.tmpl", "_content/article.tmpl", "_content/doc/default.tmpl"} {
		keep[name] = true
	}
	for _, f := range m.Files {
		if !keep[f.Path] {
			continue
		}
		b, err := readAdoptedSource(real, f)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, f.Path, b)
	}
	b, err := readRegular(real, SourceManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, SourceManifestPath, b)
	p := PagePackageProposal{Schema: PagePackageAdoptionSchema, Package: "future-entry-pages", Surface: "entry-pages", Sources: []Reference{}}
	for _, name := range []string{"_content/doc/index.html", "_content/doc/install.html"} {
		ref, err := ReferenceFile(root, name)
		if err != nil {
			t.Fatal(err)
		}
		p.Sources = append(p.Sources, ref)
	}
	b, _ = Encode(p)
	writeFixture(t, root, "data/page-proposal.json", b)
	ref, _ := ReferenceFile(root, "data/page-proposal.json")
	return root, ref
}

func TestPagePackageAdoptionSourceAuthority(t *testing.T) {
	root, ref := pagePackageAdoptionFixture(t)
	before, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	a, err := BuildPagePackageAdoption(root, ref)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, PagePackageAdoptionRoot)); !os.IsNotExist(err) {
		t.Fatal("preflight mutated repository")
	}
	if len(a.Documents) != 2 || a.Documents[0].Route != "/doc/" || a.Documents[1].Route != "/doc/install" {
		t.Fatal("exact source-owned canonical order", a.Documents)
	}
	if _, err := AdoptPagePackage(root, ref, strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong predecessor CAS accepted")
	}
	if _, err := AdoptPagePackage(root, ref, a.BaseIdentity); err != nil {
		t.Fatal(err)
	}
	if _, err := AdoptPagePackage(root, ref, a.BaseIdentity); err != nil {
		t.Fatal("identical replay", err)
	}
	current, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before.Packages, current.Packages[:len(before.Packages)]) || !reflect.DeepEqual(before.Sources, current.Sources[:len(before.Sources)]) {
		t.Fatal("historical package/source authority changed")
	}
	aliases, err := RedirectContracts(root, current)
	if err != nil || aliases["/doc"] != "/doc/" {
		t.Fatal("canonical index alias not source-bound", err)
	}
	cat := &i18n.Catalog{}
	ids, err := AttachAdoptedPageTranslationUnits(root, cat)
	if err != nil || len(ids) != 2 {
		t.Fatal("shared registry", ids, err)
	}
	if err := cat.UseTranslationPackage(a.Package.ID); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cat.TranslationScope, ids) {
		t.Fatal("registered package exact ordered set")
	}
	if _, err := cat.Adapted[0].Adapter.(adoptedPageAdapter).ReattestationAuthority(root); err != nil {
		t.Fatal("adopted Page current adapter authority", err)
	}
	docs, raw, err := WorkflowDocuments(root, current, a.Package.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range docs {
		restored, err := (PageAdapter{d, raw[d.Path]}).Restore(d.Units[0].Source)
		if err != nil || !bytes.Equal(restored, raw[d.Path]) {
			t.Fatal("Page shared restore", err)
		}
	}
	if _, _, err := cat.Adapted[0].Adapter.ProtectedInput(nil); err == nil {
		t.Fatal("pre-adoption corpus/Review authorized new content")
	}
	corpus, err := BuildUnifiedCorpus(root)
	if err != nil {
		t.Fatal("registered future package corpus projection", err)
	}
	if len(corpus.PackageAuthorities) != 1 || len(corpus.PackageAuthorities[0].SourcePaths) != 2 {
		t.Fatal("new package source authority not carried into corpus")
	}
	if !reflect.DeepEqual(packageRouteFamilies(a.Package), a.Package.Routes) {
		t.Fatal("future package inherited unrelated wildcard coverage")
	}
	priorCorpus, _ := readRegular(root, i18n.GlossaryCorpusPath)
	if _, err := AdoptCorpusProjection(root, digest(priorCorpus), true); err != nil {
		t.Fatal("adopted frozen-source corpus publication", err)
	}
	if err := CheckUnifiedCorpus(root); err != nil {
		t.Fatal("adopted frozen-source corpus currentness", err)
	}
	if _, err := i18n.RequireUnifiedGlossaryReview(root, "zh-CN"); err == nil {
		t.Fatal("old full Review authorized new package")
	}
	if _, err := CheckPagePackageAdoption(root, Reference{PagePackageAdoptionPath(a), "wrong"}); err == nil {
		t.Fatal("wrong receipt hash accepted")
	}
	b, _ := readRegular(root, ref.Path)
	writeFixture(t, root, ref.Path, append(b, ' '))
	if _, err := LoadCurrent(root); err == nil {
		t.Fatal("proposal byte drift accepted")
	}
	writeFixture(t, root, ref.Path, b)
	source, _ := readRegular(root, a.Sources[0].Path)
	writeFixture(t, root, a.Sources[0].Path, append(source, ' '))
	if _, err := LoadCurrent(root); err == nil {
		t.Fatal("source byte drift accepted")
	}
	writeFixture(t, root, a.Sources[0].Path, source)
	if _, err := LoadCurrent(root); err != nil {
		t.Fatal(err)
	}
}

func TestPagePackageAdoptionRejectsInvalidScope(t *testing.T) {
	root, ref := pagePackageAdoptionFixture(t)
	b, _ := readRegular(root, ref.Path)
	var p PagePackageProposal
	if err := StrictJSON(b, &p); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*PagePackageProposal)
	}{
		{"wrong-hash", func(p *PagePackageProposal) { p.Sources[0].SHA256 = strings.Repeat("0", 64) }},
		{"duplicate", func(p *PagePackageProposal) { p.Sources = append(p.Sources, p.Sources[0]) }},
		{"existing-package", func(p *PagePackageProposal) { p.Package = "learn-docs-v1" }},
		{"existing-surface", func(p *PagePackageProposal) { p.Surface = "tutorial" }},
		{"empty", func(p *PagePackageProposal) { p.Sources = nil }},
		{"unsafe", func(p *PagePackageProposal) { p.Package = "../escape" }},
		{"unknown-source", func(p *PagePackageProposal) { p.Sources[0].Path = "_content/unknown.html" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var next PagePackageProposal
			_ = StrictJSON(b, &next)
			tc.change(&next)
			data, _ := Encode(next)
			writeFixture(t, root, "data/invalid-page-proposal.json", data)
			bad, _ := ReferenceFile(root, "data/invalid-page-proposal.json")
			if _, err := BuildPagePackageAdoption(root, bad); err == nil {
				t.Fatal("invalid scope accepted")
			}
		})
	}
}
