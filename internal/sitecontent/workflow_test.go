package sitecontent

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// All language/evidence fixtures are synthetic, isolated in t.TempDir. No test
// writes a repository locale, historical artifact or completion record.
func siteFixture(t *testing.T, extraAuthority ...string) string {
	t.Helper()
	src := repositoryRoot(t)
	root := t.TempDir()
	copyFile := func(p string) {
		if p == PageRegistryPath {
			// These fixtures model the immutable pre-adoption 49-Page world.
			// Current 51-Page scheduling is exercised by membership fixtures.
			m, err := readPackageMembership(src)
			if err != nil {
				t.Fatal(err)
			}
			if m != nil {
				b, err := readRegular(src, m.PreviousRegistry.Path)
				if err != nil {
					t.Fatal(err)
				}
				writeFixture(t, root, p, b)
				return
			}
		}
		if p == SnapshotPath {
			// Isolated legacy fixtures keep their original ZIP-backed reader.
			// The repository supplies the unchanged encoding from adopted bytes.
			b, err := ReadSourceSnapshot(src)
			if err != nil {
				t.Fatal(err)
			}
			writeFixture(t, root, p, b)
			return
		}
		// Declared authority paths are repository-relative regular files; reject
		// symlinks/unsafe paths instead of widening the list to ad-hoc directories.
		b, err := readRegular(src, p)
		if err != nil {
			t.Fatal(err)
		}
		st, err := os.Stat(filepath.Join(src, p))
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, p, b)
		if err := os.Chmod(filepath.Join(root, p), st.Mode().Perm()); err != nil {
			t.Fatal(err)
		}
	}
	freshCurrent := false
	for _, p := range extraAuthority {
		freshCurrent = freshCurrent || p == "current-authorization-fresh-fixture"
	}
	if freshCurrent {
		// A fresh synthetic campaign has its own current corpus/Review, not
		// the real locale's applied successor/mutable completion history.
		for _, dir := range []string{"internal/i18n", "internal/sitecontent", "internal/languageaudit", "internal/languageview"} {
			if err := filepath.WalkDir(filepath.Join(src, dir), func(p string, d fs.DirEntry, e error) error {
				if e != nil {
					return e
				}
				if d.IsDir() {
					return nil
				}
				if strings.HasSuffix(p, "_test.go") {
					return nil
				}
				rel, err := filepath.Rel(src, p)
				if err != nil {
					return err
				}
				copyFile(filepath.ToSlash(rel))
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		}
		copyFile("go.mod")
		copyFile("go.sum")
	}
	if err := filepath.WalkDir(filepath.Join(src, "_content/tour"), func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !d.IsDir() {
			rel, e := filepath.Rel(src, p)
			if e != nil {
				return e
			}
			copyFile(filepath.ToSlash(rel))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, p := range append(append(append([]string{}, PackageAuthorities...), StructuredSemanticAuthorities...), structuredTransportContract, GlobalPath, SnapshotPath, "UPSTREAM_MANIFEST.tsv", "data/tour-pages.tsv", "data/tour-conditional-pages.tsv", "data/tour-examples.tsv", "production/identity.json", "locales/zh-CN/locale.json", "docs/LOCALE_SURFACE_REVIEW.md", "data/site-shell/home.html", "data/site-shell/translation.html") {
		copyFile(p)
	}
	for _, p := range []string{PageRegistryPath, i18n.GlossaryCorpusPath, "internal/tour/ui/en.json", "data/course-seo/source-descriptions.json", "docs/NEW_LOCALE_RUNBOOK.md", "docs/TERMINOLOGY_GUIDE.md", "docs/TRANSLATION_TERMINOLOGY.md", "internal/tour/project.go", "internal/tour/languages.go", "internal/tourpolicy/policy.go"} {
		if freshCurrent && p == i18n.GlossaryCorpusPath {
			continue
		}
		copyFile(p)
	}
	for _, p := range []string{"data/terminology-discovery-sources.json", "data/terminology-discovery-sources.zip"} {
		if freshCurrent {
			continue
		}
		if _, err := os.Stat(filepath.Join(src, p)); err == nil {
			copyFile(p)
		}
	}
	for _, p := range extraAuthority {
		if p == "current-authorization-fresh-fixture" {
			continue
		}
		copyFile(p)
	}
	copyFile("docs/SHARED_TRANSLATION_UNIT_LIFECYCLE.md")
	copyFile("docs/SURFACE_REVISION_TRANSPORT.md")
	if freshCurrent {
		g, err := LoadCurrent(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, pkg := range []string{"site-v2-shell", "learn-docs-v1"} {
			_, raw, err := PackageDocuments(root, g, pkg)
			if err != nil {
				t.Fatal(err)
			}
			for p, b := range raw {
				writeFixture(t, root, p, b)
			}
		}
		corpus, err := BuildUnifiedCorpus(root)
		if err != nil {
			t.Fatal(err)
		}
		b, err := Encode(corpus)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, i18n.GlossaryCorpusPath, b)
	}
	writeFixture(t, root, "locales/zh-CN/glossary.yaml", []byte("locale: zh-CN\nmandatory:\n  synthetic-test-term: 测试术语\n"))
	if _, _, err := i18n.RecordGlossaryReview(root, "zh-CN", "fixture-glossary-1", "independent-fixture-reviewer", "passed", nil); err != nil {
		t.Fatal(err)
	}
	bundle, _, err := i18n.ExportUnifiedGlossaryBundle(root, "zh-CN", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.RecordUnifiedGlossaryReview(root, bundle, "synthetic-unified-review", "synthetic-independent-reviewer", "synthetic-generation", "passed", nil); err != nil {
		t.Fatal(err)
	}
	b, _ := Encode(Locale{Schema: LocaleSchema, Locale: "zh-CN", Packages: []Completion{}})
	writeFixture(t, root, LocalePath("zh-CN"), b)
	return root
}
func writeFixture(t *testing.T, root, p string, b []byte) {
	t.Helper()
	name := filepath.Join(root, p)
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, b, 0644); err != nil {
		t.Fatal(err)
	}
}
func importFixture(t *testing.T, root, pkg, batch string, selected []string) (*Generation, Reference, []byte) {
	t.Helper()
	b, m, err := ExportPackageGeneration(root, "zh-CN", pkg, "initial", batch, selected, nil)
	if err != nil {
		t.Fatal(err)
	}
	out := Output{Schema: WorkflowSchema, BatchSHA: m.Identity, Targets: []Target{}}
	units := unitsByID(m.Documents)
	for _, id := range m.Selected {
		out.Targets = append(out.Targets, Target{id, units[id].Source})
	}
	raw, _ := Encode(out)
	g, name, err := ImportPackageGeneration(root, b, raw, Provenance{Provider: "codex", Model: "gpt-5.6-sol-high", Session: "synthetic-generation", GeneratedAt: "2026-10-06T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := ReferenceFile(root, name)
	if err != nil {
		t.Fatal(err)
	}
	return g, ref, b
}
func reviewFixture(t *testing.T, root, pkg string, gr Reference, ids []string, rating string) []Reference {
	t.Helper()
	refs := []Reference{}
	for start := 0; start < len(ids); start += MaxReviewUnits {
		end := start + MaxReviewUnits
		if end > len(ids) {
			end = len(ids)
		}
		r, err := ReviewScope(root, "zh-CN", pkg, []Reference{gr}, ids[start:end])
		if err != nil {
			t.Fatal(err)
		}
		r.ReviewID = "fixture-review-" + gr.SHA256[:8] + "-" + ids[start][:8]
		r.Invocation = r.ReviewID
		r.Session = "synthetic-independent-reviewer"
		r.Model = "gpt-5.6-sol-high"
		r.ReviewedAt = "2026-10-06T00:01:00Z"
		g, err := loadGeneration(root, gr)
		if err != nil {
			t.Fatal(err)
		}
		values := map[string]string{}
		for _, v := range g.Output.Targets {
			values[v.ID] = v.Text
		}
		for _, id := range r.Selected {
			findings := []string{}
			if rating != "A" {
				findings = append(findings, "synthetic finding")
			}
			r.Ratings = append(r.Ratings, Rating{id, digest([]byte(values[id])), rating, findings})
		}
		raw, _ := Encode(r)
		_, name, err := RecordPackageReview(root, raw)
		if err != nil {
			t.Fatal(err)
		}
		ref, err := ReferenceFile(root, name)
		if err != nil {
			t.Fatal(err)
		}
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Path < refs[j].Path })
	return refs
}

func TestSiteFoundationStrictEvidence(t *testing.T) {
	root := t.TempDir()
	var out Output
	for _, raw := range []string{`{"schema":"x","unknown":1}`, `{"Schema":"x"}`, `{"schema":null}`, `{"schema":"x","schema":"y"}`, `{"targets":[{"id":"a","id":"b"}]}`, `{} {}`} {
		if StrictJSON([]byte(raw), &out) == nil {
			t.Fatal("accepted strict violation")
		}
	}
	if err := SaveArtifact(root, "../escape", []byte("x")); err == nil {
		t.Fatal("traversal")
	}
	writeFixture(t, root, "regular/a", []byte("x"))
	if err := os.Symlink(filepath.Join(root, "regular"), filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadArtifact(root, "link/a"); err == nil {
		t.Fatal("symlink read")
	}
	if err := SaveArtifact(root, "link/b", []byte("x")); err == nil {
		t.Fatal("symlink write")
	}
	if err := readReference(root, Reference{"regular/a", strings.Repeat("0", 64)}, &out); err == nil {
		t.Fatal("hash mismatch")
	}
	if err := SaveArtifact(root, "regular/a", []byte("different")); err == nil {
		t.Fatal("overwrite")
	}
	if err := SaveArtifact(root, "regular/a", []byte("x")); err != nil {
		t.Fatal("idempotence", err)
	}
}
