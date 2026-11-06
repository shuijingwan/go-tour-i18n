package sitecontent

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestRepoLocalFrozenGitArchiveMetadata(t *testing.T) {
	h := &tar.Header{Typeflag: tar.TypeXGlobalHeader, Name: "pax_global_header", PAXRecords: map[string]string{"comment": UpstreamCommit}}
	if err := verifyFrozenGitArchiveMetadata(h); err != nil {
		t.Fatal(err)
	}
	h.PAXRecords["comment"] = strings.Repeat("0", 40)
	if err := verifyFrozenGitArchiveMetadata(h); err == nil {
		t.Fatal("archive from another commit accepted")
	}
}

func sourceAuthorityFixture(t *testing.T) (string, FrozenSourceManifest, []byte) {
	t.Helper()
	root := t.TempDir()
	m := FrozenSourceManifest{Schema: sourceManifestSchema, Repository: "https://github.com/golang/website.git", Commit: UpstreamCommit, Files: []FrozenSourceFile{}, Legacy: LegacySourceSubset{Path: SnapshotPath, Encoding: "go-zip-deflate-0644-v1", Paths: []string{"_content/doc/tutorial/index.html", "internal/web/page.go"}}}
	payloads := map[string][]byte{"_content/doc/tutorial/index.html": []byte("<h1>Frozen source</h1>\n"), "internal/web/page.go": []byte("package web\n"), "_content/images/unregistered.bin": {0, 1, 255}}
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	for _, p := range m.Legacy.Paths {
		h := &zip.FileHeader{Name: p, Method: zip.Deflate}
		h.SetMode(0644)
		w, _ := z.CreateHeader(h)
		w.Write(payloads[p])
	}
	z.Close()
	m.Legacy.SHA256 = digest(out.Bytes())
	for p, b := range payloads {
		name := adoptedSourcePath(p)
		writeFixture(t, root, name, b)
		m.Files = append(m.Files, FrozenSourceFile{Path: name, UpstreamPath: p, GitMode: "100644", LocalMode: 0644, Classification: "upstream-exact", Role: sourceRole(p), Size: int64(len(b)), SHA256: digest(b), UpstreamSHA256: digest(b), UpstreamBlob: strings.Repeat("1", 40), Provenance: "synthetic test Git object"})
	}
	sealSourceManifest(&m)
	b, _ := Encode(m)
	writeFixture(t, root, SourceManifestPath, b)
	return root, m, out.Bytes()
}

func TestRepoLocalFrozenSourceSubsetAndIntegrity(t *testing.T) {
	root, m, legacy := sourceAuthorityFixture(t)
	got, err := ReadSourceSnapshot(root)
	if err != nil || !bytes.Equal(got, legacy) {
		t.Fatal("no physical ZIP should be required", err)
	}
	if _, err := CheckFrozenSources(root); err != nil {
		t.Fatal(err)
	}
	// Package currentness must not depend on an unrelated mirror asset. The full
	// adoption integrity checker still catches its corruption.
	writeFixture(t, root, "_content/images/unregistered.bin", []byte("changed unrelated asset"))
	if got, err := ReadSourceSnapshot(root); err != nil || !bytes.Equal(got, legacy) {
		t.Fatal("unrelated asset made registered package stale", err)
	}
	if _, err := CheckFrozenSources(root); err == nil {
		t.Fatal("complete mirror integrity accepted changed asset")
	}
	writeFixture(t, root, "_content/doc/tutorial/index.html", []byte("changed registered source"))
	if _, err := ReadSourceSnapshot(root); err == nil {
		t.Fatal("registered source drift accepted")
	}
	writeFixture(t, root, "_content/doc/tutorial/index.html", []byte("<h1>Frozen source</h1>\n"))
	m.Legacy.SHA256 = digest([]byte("different legacy identity"))
	sealSourceManifest(&m)
	b, _ := Encode(m)
	writeFixture(t, root, SourceManifestPath, b)
	if _, err := ReadSourceSnapshot(root); err == nil {
		t.Fatal("unproven legacy digest accepted")
	}
}

func TestRepoLocalFrozenSourceManifestFailClosed(t *testing.T) {
	root, original, _ := sourceAuthorityFixture(t)
	for _, kind := range []string{"duplicate", "traversal", "classification", "role", "commit", "legacy-duplicate", "unproven-override", "mode"} {
		t.Run(kind, func(t *testing.T) {
			b, _ := Encode(original)
			var m FrozenSourceManifest
			if err := StrictJSON(b, &m); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "duplicate":
				m.Files = append(m.Files, m.Files[0])
			case "traversal":
				m.Files[0].Path = "_content/../outside"
			case "classification":
				m.Files[0].Classification = "unknown"
			case "role":
				m.Files[0].Role = "unknown"
			case "commit":
				m.Commit = strings.Repeat("2", 40)
			case "legacy-duplicate":
				m.Legacy.Paths = append(m.Legacy.Paths, m.Legacy.Paths[0])
			case "unproven-override":
				m.Files[0].Classification = "project-override"
			case "mode":
				m.Files[0].GitMode = "120000"
			}
			sealSourceManifest(&m)
			if err := validateSourceManifest(m); err == nil {
				t.Fatal("invalid manifest accepted")
			}
		})
	}
	b, _ := Encode(original)
	for _, raw := range [][]byte{
		bytes.Replace(b, []byte(`"schema":`), []byte(`"unknown":true,"schema":`), 1),
		bytes.Replace(b, []byte(`"schema":`), []byte(`"schema":"duplicate","schema":`), 1),
		append(bytes.Clone(b), []byte("{}")...),
	} {
		writeFixture(t, root, SourceManifestPath, raw)
		if _, err := loadSourceManifest(root); err == nil {
			t.Fatal("non-strict JSON accepted")
		}
	}
}

func TestRepoLocalFrozenSourceSymlinkAndPendingMutation(t *testing.T) {
	root, _, _ := sourceAuthorityFixture(t)
	p := pathJoin(root, "_content/doc/tutorial/index.html")
	if err := os.Rename(p, p+".saved"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(p+".saved", p); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSourceSnapshot(root); err == nil {
		t.Fatal("symlink accepted")
	}
	writeFixture(t, root, sourceMigrationLock, []byte("pending"))
	if _, err := ReadSourceSnapshot(root); err == nil || !strings.Contains(err.Error(), "mutation=unknown") {
		t.Fatal("pending migration did not fail closed", err)
	}
}

func TestRepoLocalSourceInstallNeverOverwrites(t *testing.T) {
	root := t.TempDir()
	p := "_content/js/playground.js"
	before := []byte("intentional project override")
	writeFixture(t, root, p, before)
	if err := installSourceFile(root, p, []byte("upstream"), 0644); err == nil {
		t.Fatal("collision overwritten")
	}
	got, _ := readRegular(root, p)
	if !bytes.Equal(before, got) {
		t.Fatal("existing source changed")
	}
	if err := installSourceFile(root, "_content/new-script", []byte("binary\x00"), 0755); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(pathJoin(root, "_content/new-script"))
	if info.Mode().Perm() != 0755 {
		t.Fatal("Git executable mode lost")
	}
	if err := installSourceFile(root, "_content/../outside", nil, 0644); err == nil {
		t.Fatal("traversal accepted")
	}
}

func TestRepoLocalSourcePreservesExistingPermissionProvenance(t *testing.T) {
	root, m, _ := sourceAuthorityFixture(t)
	m.Files[0].LocalMode = 0775
	if err := os.Chmod(pathJoin(root, m.Files[0].Path), 0775); err != nil {
		t.Fatal(err)
	}
	sealSourceManifest(&m)
	b, _ := Encode(m)
	writeFixture(t, root, SourceManifestPath, b)
	if _, err := ReadSourceSnapshot(root); err != nil {
		t.Fatal("existing project permissions must be preserved independently of Git mode", err)
	}
}

func TestRepoLocalHistoricalCorpusRequiresPhysicalZIPOrExactProof(t *testing.T) {
	repo := repositoryRoot(t)
	m, err := loadSourceManifest(repo)
	if err != nil {
		t.Fatal(err)
	}
	corpus, err := i18n.LoadUnifiedGlossaryCorpus(repo)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	copyFile := func(p string) {
		t.Helper()
		b, err := readRegular(repo, p)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, p, b)
		info, _ := os.Stat(pathJoin(repo, p))
		if err := os.Chmod(pathJoin(root, p), info.Mode().Perm()); err != nil {
			t.Fatal(err)
		}
	}
	copyFile(SourceManifestPath)
	copyFile(i18n.GlossaryCorpusPath)
	for _, p := range m.Legacy.Paths {
		copyFile(adoptedSourcePath(p))
	}
	for _, ref := range corpus.Sources {
		if ref.Path != SnapshotPath {
			copyFile(ref.Path)
		}
	}
	legacy, err := ReadSourceSnapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, SnapshotPath, legacy)
	if _, err := i18n.LoadUnifiedGlossaryCorpus(root); err != nil {
		t.Fatal("unchanged historical/current corpus lost its source bindings", err)
	}
	if err := os.Remove(pathJoin(root, SnapshotPath)); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSourceSnapshot(root); err != nil {
		t.Fatal("migrated source reader still needs physical ZIP", err)
	}
	if _, err := i18n.LoadUnifiedGlossaryCorpus(root); err == nil {
		t.Fatal("physical historical source reference was silently weakened")
	}
}

// Real storage migration proof in an isolated minimal source fixture, without
// any Generation/Reviewer/closure/activation lifecycle or external checkout.
func TestRepoLocalRegisteredGlobalIdentityWithoutPhysicalZIP(t *testing.T) {
	repo := repositoryRoot(t)
	m, err := loadSourceManifest(repo)
	if err != nil {
		t.Fatal("run after the authorized source migration", err)
	}
	legacy, err := ReadSourceSnapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	want, err := BuildGlobal(repo, legacy)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	copyFile := func(p string) {
		t.Helper()
		b, err := readRegular(repo, p)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, p, b)
	}
	for _, p := range m.Legacy.Paths {
		copyFile(adoptedSourcePath(p))
	}
	for _, p := range []string{SourceManifestPath, GlobalPath, "data/site-shell/home.html", "data/site-shell/translation.html"} {
		copyFile(p)
	}
	if err := filepath.WalkDir(pathJoin(repo, "_content/tour"), func(p string, d fs.DirEntry, err error) error {
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
	got, err := LoadCurrent(root)
	if err != nil || !reflect.DeepEqual(want, got) {
		t.Fatal("storage form changed global/package identity", err)
	}
	if _, err := os.Stat(pathJoin(root, SnapshotPath)); !os.IsNotExist(err) {
		t.Fatal("fixture unexpectedly has physical ZIP")
	}
	if _, err := RedirectContracts(root, got); err != nil {
		t.Fatal("redirect contract still requires physical ZIP", err)
	}
	docs, _, err := PackageDocuments(root, got, "learn-docs-v1")
	if err != nil || len(docs) != 56 {
		t.Fatal("49 Page + 7 data changed", len(docs), err)
	}
	// Completion schema uses identical package projections for every locale.
	locales, _ := filepath.Glob(pathJoin(repo, "locales/*/content-scope.json"))
	if len(locales) != 65 {
		t.Fatal("expected current 65 sparse locale authorities", len(locales))
	}
	for _, p := range locales {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var l Locale
		if err := StrictJSON(b, &l); err != nil {
			t.Fatal(err)
		}
		if err := ValidateLocale(got, l); err != nil {
			t.Fatal("storage migration made locale completion stale", l.Locale, err)
		}
	}
}
