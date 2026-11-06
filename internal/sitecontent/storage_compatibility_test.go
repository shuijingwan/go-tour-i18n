package sitecontent

import (
	"archive/zip"
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sourcestorage"
)

func storageProofFixture(t *testing.T) (string, sourcestorage.Compatibility, []byte) {
	t.Helper()
	root, m, old := sourceAuthorityFixture(t)
	corpus := []byte("immutable historical corpus bytes\n")
	writeFixture(t, root, i18n.GlossaryCorpusPath, corpus)
	manifest, _ := readRegular(root, SourceManifestPath)
	name := "_content/doc/tutorial/index.html"
	b, _ := readRegular(root, name)
	p := sourcestorage.Compatibility{
		Schema:           sourcestorage.CompatibilitySchema,
		OldStorage:       contentidentity.Reference{Path: SnapshotPath, SHA256: digest(old)},
		Manifest:         contentidentity.Reference{Path: SourceManifestPath, SHA256: digest(manifest)},
		ManifestIdentity: m.Identity, Equality: "byte-exact",
		Corpora: []sourcestorage.CorpusBinding{{Reference: contentidentity.Reference{Path: i18n.GlossaryCorpusPath, SHA256: digest(corpus)}, SemanticIdentity: digest([]byte("unchanged language contexts")), RequiredEntries: []string{name}}},
		Entries: []sourcestorage.EquivalentEntry{{OldPath: name, OldSHA: digest(b), NewPath: name, NewSHA: digest(b), Role: "website-source"}},
	}
	sourcestorage.SealCompatibility(&p)
	return root, p, old
}

func writeProof(t *testing.T, root string, p sourcestorage.Compatibility) {
	t.Helper()
	b, err := Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, sourcestorage.CompatibilityPath(p), b)
}

func resolveProof(root string, p sourcestorage.Compatibility) error {
	c := p.Corpora[0]
	return sourcestorage.CheckCorpusReference(root, c.Reference, c.SemanticIdentity, p.OldStorage, c.RequiredEntries)
}

func TestStorageCompatibilityPhysicalAndRetired(t *testing.T) {
	root, p, old := storageProofFixture(t)
	writeFixture(t, root, SnapshotPath, old)
	if err := resolveProof(root, p); err != nil {
		t.Fatal("original physical validation", err)
	}
	if err := sourcestorage.HasCurrentProof(root, p.Corpora[0].Reference, p.Corpora[0].SemanticIdentity, p.OldStorage, p.Corpora[0].RequiredEntries); err == nil {
		t.Fatal("retirement without proof accepted")
	}
	writeProof(t, root, p)
	if err := resolveProof(root, p); err != nil {
		t.Fatal("ZIP-present proof", err)
	}
	writeFixture(t, root, SnapshotPath, append(bytes.Clone(old), 0))
	if err := resolveProof(root, p); err == nil {
		t.Fatal("present corrupt ZIP replaced by proof")
	}
	if err := os.Remove(pathJoin(root, SnapshotPath)); err != nil {
		t.Fatal(err)
	}
	if err := resolveProof(root, p); err != nil {
		t.Fatal("ZIP-absent proof", err)
	}
	// These bytes belong to the old transport, but are not consumed by this
	// corpus. Their current storage must not become a glossary dependency.
	writeFixture(t, root, adoptedSourcePath("internal/web/page.go"), []byte("unconsumed machine contract changed"))
	if err := resolveProof(root, p); err != nil {
		t.Fatal("unused ZIP entry became glossary dependency", err)
	}
	writeFixture(t, root, p.Entries[0].NewPath, []byte("changed consumed bytes"))
	if err := resolveProof(root, p); err == nil {
		t.Fatal("changed consumed bytes accepted")
	}
}

func TestStorageCompatibilityFailClosed(t *testing.T) {
	for _, kind := range []string{"missing-proof", "missing-entry", "extra-entry", "duplicate-entry", "traversal", "wrong-role", "wrong-byte-sha", "wrong-old-sha", "wrong-old-path", "self-identity", "stale-manifest", "changed-corpus", "wrong-corpus-membership", "wrong-semantic-identity", "unknown-field", "duplicate-field", "noncanonical", "source-symlink", "proof-symlink"} {
		t.Run(kind, func(t *testing.T) {
			root, p, _ := storageProofFixture(t)
			original := p
			original.Corpora = append([]sourcestorage.CorpusBinding{}, p.Corpora...)
			original.Corpora[0].RequiredEntries = append([]string{}, p.Corpora[0].RequiredEntries...)
			switch kind {
			case "missing-proof":
				if err := resolveProof(root, p); err == nil {
					t.Fatal("missing proof accepted")
				}
				return
			case "missing-entry":
				p.Entries = nil
			case "extra-entry", "duplicate-entry":
				p.Entries = append(p.Entries, p.Entries[0])
			case "traversal":
				p.Entries[0].OldPath = "_content/../outside"
			case "wrong-role":
				p.Entries[0].Role = "asset"
			case "wrong-byte-sha":
				p.Entries[0].OldSHA = digest([]byte("different"))
				p.Entries[0].NewSHA = p.Entries[0].OldSHA
			case "wrong-old-sha":
				p.OldStorage.SHA256 = digest([]byte("different zip"))
			case "wrong-old-path":
				p.OldStorage.Path = "data/other.zip"
			case "stale-manifest":
				p.ManifestIdentity = digest([]byte("older manifest"))
			case "changed-corpus":
				writeFixture(t, root, i18n.GlossaryCorpusPath, []byte("changed corpus"))
			case "wrong-corpus-membership":
				p.Corpora[0].RequiredEntries = []string{"_content/doc/another.html"}
				p.Entries[0].OldPath = p.Corpora[0].RequiredEntries[0]
				p.Entries[0].NewPath = p.Entries[0].OldPath
			case "wrong-semantic-identity":
				p.Corpora[0].SemanticIdentity = digest([]byte("different contexts"))
			case "source-symlink":
				name := pathJoin(root, p.Entries[0].NewPath)
				if err := os.Rename(name, name+".saved"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(name+".saved", name); err != nil {
					t.Fatal(err)
				}
			}
			sourcestorage.SealCompatibility(&p)
			if kind == "self-identity" {
				p.Identity = digest([]byte("forged"))
			}
			writeProof(t, root, p)
			path := sourcestorage.CompatibilityPath(p)
			raw, _ := readRegular(root, path)
			switch kind {
			case "unknown-field":
				raw = bytes.Replace(raw, []byte(`"schema":`), []byte(`"extra":true,"schema":`), 1)
			case "duplicate-field":
				raw = bytes.Replace(raw, []byte(`"schema":`), []byte(`"schema":"duplicate","schema":`), 1)
			case "noncanonical":
				raw = append(raw, '\n')
			case "proof-symlink":
				name := pathJoin(root, path)
				if err := os.Rename(name, name+".saved"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(name+".saved", name); err != nil {
					t.Fatal(err)
				}
			}
			if kind != "proof-symlink" {
				writeFixture(t, root, path, raw)
			}
			if err := resolveProof(root, original); err == nil {
				t.Fatal("invalid storage resolution accepted", kind)
			}
		})
	}
}

func TestStorageCompatibilityPresentZIPRejectsFalseHistoricalProof(t *testing.T) {
	root, p, old := storageProofFixture(t)
	writeFixture(t, root, SnapshotPath, old)
	p.ManifestIdentity = digest([]byte("older manifest"))
	p.Entries[0].OldSHA = digest([]byte("false old bytes"))
	p.Entries[0].NewSHA = p.Entries[0].OldSHA
	sourcestorage.SealCompatibility(&p)
	writeProof(t, root, p)
	if err := resolveProof(root, p); err == nil {
		t.Fatal("non-current proof disagreed with present ZIP but was skipped")
	}
}

func TestStorageCompatibilityHistoricalProofWithPresentZIP(t *testing.T) {
	root, p, old := storageProofFixture(t)
	writeFixture(t, root, SnapshotPath, old)
	p.ManifestIdentity = digest([]byte("older manifest"))
	sourcestorage.SealCompatibility(&p)
	writeProof(t, root, p)
	if err := resolveProof(root, p); err != nil {
		t.Fatal("exact physical validation rejected harmless historical proof", err)
	}
	c := p.Corpora[0]
	if err := sourcestorage.HasCurrentProof(root, c.Reference, c.SemanticIdentity, p.OldStorage, c.RequiredEntries); err == nil {
		t.Fatal("historical proof authorized retirement")
	}
	if err := os.Remove(pathJoin(root, SnapshotPath)); err != nil {
		t.Fatal(err)
	}
	if err := resolveProof(root, p); err == nil {
		t.Fatal("retired ZIP accepted stale manifest proof")
	}
}

func TestStorageCompatibilityZIPRejectsUnsafeEntries(t *testing.T) {
	for _, kind := range []string{"duplicate", "traversal", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			var buf bytes.Buffer
			z := zip.NewWriter(&buf)
			name := "_content/doc/tutorial/index.html"
			mode := fs.FileMode(0644)
			if kind == "traversal" {
				name = "_content/../escape"
			}
			if kind == "symlink" {
				mode |= os.ModeSymlink
			}
			count := 1
			if kind == "duplicate" {
				count = 2
			}
			for i := 0; i < count; i++ {
				h := &zip.FileHeader{Name: name, Method: zip.Deflate}
				h.SetMode(mode)
				w, err := z.CreateHeader(h)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := w.Write([]byte("source")); err != nil {
					t.Fatal(err)
				}
			}
			if err := z.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := sourcestorage.ZIPEntries(buf.Bytes()); err == nil {
				t.Fatal("unsafe old ZIP accepted")
			}
		})
	}
}

// This fixture uses only existing authority/evidence bytes. It creates no
// language or lifecycle evidence and removes only its own temporary ZIP.
func TestStorageRetirementRealCorpusAndHistory(t *testing.T) {
	repo := repositoryRoot(t)
	m, err := loadSourceManifest(repo)
	if err != nil {
		t.Fatal(err)
	}
	proof, proofPath, err := CreateSourceStorageCompatibility(repo, false)
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatal(err)
		}
		// After authorized retirement, reuse the immutable proof instead of
		// reconstructing a new historical trust root from current sources.
		paths, e := filepath.Glob(pathJoin(repo, sourcestorage.CompatibilityRoot+"/"+m.Legacy.SHA256+"/*.json"))
		if e != nil || len(paths) != 1 {
			t.Fatal(err, e)
		}
		rel, _ := filepath.Rel(repo, paths[0])
		proofPath = filepath.ToSlash(rel)
		b, e := readRegular(repo, proofPath)
		if e != nil {
			t.Fatal(e)
		}
		proof = &sourcestorage.Compatibility{}
		if e := StrictJSON(b, proof); e != nil {
			t.Fatal(e)
		}
		if e := sourcestorage.VerifyCompatibility(repo, *proof, nil); e != nil {
			t.Fatal(e)
		}
	}
	root := t.TempDir()
	copyFile := func(p string) {
		t.Helper()
		b, err := readRegular(repo, p)
		if err != nil {
			t.Fatal(p, err)
		}
		writeFixture(t, root, p, b)
		info, err := os.Stat(pathJoin(repo, p))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(pathJoin(root, p), info.Mode().Perm()); err != nil {
			t.Fatal(err)
		}
	}
	copyTree := func(dir string) {
		t.Helper()
		if err := filepath.WalkDir(pathJoin(repo, dir), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, err := filepath.Rel(repo, p)
			if err != nil {
				return err
			}
			copyFile(filepath.ToSlash(rel))
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range m.Legacy.Paths {
		copyFile(adoptedSourcePath(p))
	}
	for _, p := range []string{SourceManifestPath, GlobalPath, "UPSTREAM_MANIFEST.tsv", "data/tour-pages.tsv", "data/tour-conditional-pages.tsv", "data/tour-examples.tsv", "data/site-shell/home.html", "data/site-shell/translation.html", "locales/zh-CN/glossary.yaml"} {
		copyFile(p)
	}
	copyTree("_content/tour")
	copyTree("data/unified-glossary-reviews/zh-CN")
	copyTree("data/unified-glossary-reviews/zh-CN-bundles")
	paths, err := CorpusAuthorityPaths(repo)
	if err != nil {
		t.Fatal(err)
	}
	before := map[string]*i18n.GlossarySourceCorpus{}
	for _, p := range paths {
		c, err := i18n.LoadGlossaryCorpusAt(repo, p)
		if err != nil {
			t.Fatal(p, err)
		}
		before[p] = c
		copyFile(p)
		for _, ref := range c.Sources {
			if ref.Path != SnapshotPath {
				copyFile(ref.Path)
			}
		}
	}
	legacy, err := ReadSourceSnapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, SnapshotPath, legacy)
	writeProof(t, root, *proof)
	if sourcestorage.CompatibilityPath(*proof) != proofPath {
		t.Fatal("proof path changed")
	}
	wantGlobal, err := LoadCurrent(repo)
	if err != nil {
		t.Fatal(err)
	}
	wantReview, err := i18n.RequireUnifiedGlossaryReview(repo, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"present", "absent"} {
		if state == "absent" {
			if err := os.Remove(pathJoin(root, SnapshotPath)); err != nil {
				t.Fatal(err)
			}
		}
		for _, p := range paths {
			got, err := i18n.LoadGlossaryCorpusAt(root, p)
			if err != nil || !reflect.DeepEqual(before[p], got) {
				t.Fatal(state, "corpus/history identity or contexts changed", p, err)
			}
		}
		if err := CheckUnifiedCorpus(root); err != nil {
			t.Fatal(state, "projected current corpus changed", err)
		}
		got, err := LoadCurrent(root)
		if err != nil || !reflect.DeepEqual(wantGlobal, got) {
			t.Fatal(state, "package/global identity changed", err)
		}
		if gotReview, err := i18n.RequireUnifiedGlossaryReview(root, "zh-CN"); err != nil || gotReview != wantReview {
			t.Fatal(state, "Unified Glossary Review stale", err)
		}
	}
	// Historical verification has its own corpus binding. It cannot depend on
	// the physical bytes of a later current corpus, even in the same proof.
	currentBytes, err := readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, i18n.GlossaryCorpusPath, append(bytes.Clone(currentBytes), '\n'))
	for _, p := range paths {
		if p == i18n.GlossaryCorpusPath {
			continue
		}
		if _, err := i18n.LoadGlossaryCorpusAt(root, p); err != nil {
			t.Fatal("archive depends on later current corpus bytes", err)
		}
	}
	if _, err := i18n.LoadUnifiedGlossaryCorpus(root); err == nil {
		t.Fatal("proof accepted changed current corpus binding")
	}
	writeFixture(t, root, i18n.GlossaryCorpusPath, currentBytes)
	// The no-proof case must fail even though all equivalent current bytes exist.
	if err := os.Remove(pathJoin(root, proofPath)); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.LoadUnifiedGlossaryCorpus(root); err == nil || !strings.Contains(err.Error(), "proof") {
		t.Fatal("missing ZIP silently fell back to current source", err)
	}
}
