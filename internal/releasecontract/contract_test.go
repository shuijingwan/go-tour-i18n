package releasecontract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func fixture() (Artifact, fstest.MapFS) {
	const plan = `{"schema":"go-learning/preview-browser-plan/v1","locale":"test","families":["/"],"canonical_routes":["/"],"pages":[{"route":"/","family":"/","format":"md","source":"_content/index.md","html":"home"}]}`
	files := map[string]string{"site/pages/home.html": Digest([]byte("home")), BrowserPlanPath: Digest([]byte(plan)), "css/site.css": Digest([]byte("css"))}
	a := Artifact{Schema: Schema, Locale: "test", Origin: "https://locale.example", CoverageIdentity: Digest([]byte("coverage")), Routes: []string{"/"}, Families: []string{"/"}, Aliases: map[string]string{}, Pages: map[string]string{"/": "site/pages/home.html"}, Inputs: map[string]string{"source": Digest([]byte("source"))}, Files: files, AssetPolicy: AssetPolicy, AssetOrigin: "https://assets.example", SharedAssets: map[string]string{"css/site.css": Digest([]byte("css"))}}
	a.SharedAssetsIdentity = Identity(a.SharedAssets)
	a.Identity = a.ID()
	return a, fstest.MapFS{"site/pages/home.html": &fstest.MapFile{Data: []byte("home")}, BrowserPlanPath: &fstest.MapFile{Data: []byte(plan)}, "css/site.css": &fstest.MapFile{Data: []byte("css")}}
}

func TestProductionBundleExactIdentityAndDependency(t *testing.T) {
	a, f := fixture()
	root := t.TempDir()
	content := filepath.Join(root, "_content")
	for p, v := range f {
		dest := filepath.Join(content, p)
		os.MkdirAll(filepath.Dir(dest), 0755)
		os.WriteFile(dest, v.Data, 0644)
	}
	if e := Write(content, &a); e != nil {
		t.Fatal(e)
	}
	os.MkdirAll(filepath.Join(root, "bin"), 0755)
	os.WriteFile(filepath.Join(root, "bin/tour"), []byte("fixture executable"), 0755)
	m := Manifest{SchemaVersion: 3, Locale: a.Locale, Pages: 1, Articles: 1, TranslationUnits: 1, ExecutionTransport: "http-playground-proxy", GOOS: "linux", GOARCH: "amd64", SiteIdentity: a.Identity, BuildIdentity: a.Identity, InputBuildSHA256: Digest([]byte("input build")), SharedAssetsIdentity: a.SharedAssetsIdentity}
	write := func() {
		b, _ := json.Marshal(m)
		os.WriteFile(filepath.Join(root, "release.json"), b, 0644)
		s, e := Checksums(root)
		if e != nil {
			t.Fatal(e)
		}
		os.WriteFile(filepath.Join(root, "SHA256SUMS"), s, 0644)
	}
	write()
	id, got, e := VerifyBundle(root)
	if e != nil || got.Identity != a.Identity || !ValidSHA(id) {
		t.Fatalf("current bundle: %v", e)
	}
	m.SharedAssetsIdentity = Digest([]byte("old assets"))
	write()
	if _, _, e := VerifyBundle(root); e == nil {
		t.Fatal("disconnected shared dependency accepted")
	}
	m.SharedAssetsIdentity = a.SharedAssetsIdentity
	write()
	os.WriteFile(filepath.Join(root, "bin/other"), []byte("unexpected binary"), 0755)
	write()
	if _, _, e := VerifyBundle(root); e == nil {
		t.Fatal("nonexact executable set accepted")
	}
}

func TestSealedReleaseRejectsStaleUnsafeAndRetired(t *testing.T) {
	a, _ := fixture()
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"../outside", "/absolute", "data/site-shell/home.html", "a\\b"} {
		b, _ := fixture()
		b.Inputs[p] = Digest([]byte("x"))
		b.Identity = b.ID()
		if b.Validate() == nil {
			t.Fatalf("unsafe/retired accepted: %s", p)
		}
	}
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "source"), []byte("source"), 0644)
	if err := Current(root, &a); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "source"), []byte("changed"), 0644)
	if Current(root, &a) == nil {
		t.Fatal("stale authority accepted")
	}
}

func TestExactBundleChecksumsRejectSubsetTraversalAndSymlink(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "one"), []byte("1"), 0644)
	os.WriteFile(filepath.Join(root, "two"), []byte("2"), 0644)
	sums, e := Checksums(root)
	if e != nil {
		t.Fatal(e)
	}
	os.WriteFile(filepath.Join(root, "SHA256SUMS"), sums, 0644)
	if _, e = VerifyChecksums(root); e != nil {
		t.Fatal(e)
	}
	for _, bad := range []string{Digest([]byte("1")) + "  one\n", Digest([]byte("x")) + "  ../outside\n"} {
		os.WriteFile(filepath.Join(root, "SHA256SUMS"), []byte(bad), 0644)
		if _, e = VerifyChecksums(root); e == nil {
			t.Fatal("nonexact/unsafe manifest accepted")
		}
	}
	os.WriteFile(filepath.Join(root, "SHA256SUMS"), sums, 0644)
	os.Symlink("one", filepath.Join(root, "link"))
	if _, e = VerifyChecksums(root); e == nil {
		t.Fatal("symlink artifact accepted")
	}
}

func TestSealedContentExactSet(t *testing.T) {
	a, f := fixture()
	root := t.TempDir()
	for p, v := range f {
		dest := filepath.Join(root, p)
		os.MkdirAll(filepath.Dir(dest), 0755)
		os.WriteFile(dest, v.Data, 0644)
	}
	if e := Write(root, &a); e != nil {
		t.Fatal(e)
	}
	if _, e := Read(os.DirFS(root)); e != nil {
		t.Fatal(e)
	}
	os.WriteFile(filepath.Join(root, "unexpected"), []byte("extra"), 0644)
	if _, e := Read(os.DirFS(root)); e == nil {
		t.Fatal("unreviewed extra content accepted")
	}
}
