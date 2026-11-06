package sitecontent

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func terminologyTestFiles() map[string][]byte {
	files := map[string][]byte{}
	for _, source := range expectedTerminologySources() {
		var b []byte
		switch source.Parser {
		case "markdown-visible/v1":
			b = []byte("---\ntitle: Frozen terminology\n---\n\nVisible terminology context.\n")
		case "html-visible/v1":
			b = []byte("<h1>Frozen terminology</h1><p>Visible terminology context.</p>")
		case "go-doc-visible/v1":
			b = []byte("// Package sample documents visible terminology.\npackage sample\n\n// Exported documents an API.\nfunc Exported() {}\n")
		}
		files[source.SnapshotPath] = b
	}
	return files
}

func terminologyTestAuthority(t *testing.T, files map[string][]byte) (TerminologyDiscoveryAuthority, []byte) {
	t.Helper()
	snapshot, err := writeTerminologySnapshot(files)
	if err != nil {
		t.Fatal(err)
	}
	authority := TerminologyDiscoveryAuthority{
		Schema:   TerminologyAuthoritySchema,
		Website:  TerminologySourceOrigin{Repository: "https://github.com/golang/website.git", Commit: TerminologyWebsiteCommit},
		GoSource: TerminologySourceOrigin{Version: TerminologyGoVersion, ArchiveSHA256: TerminologyGoArchiveSHA},
		Sources:  expectedTerminologySources(), SnapshotSHA256: digest(snapshot),
	}
	for i := range authority.Sources {
		b := files[authority.Sources[i].SnapshotPath]
		authority.Sources[i].SHA256 = digest(b)
		authority.Sources[i].Bytes = int64(len(b))
	}
	authority.Identity = terminologyAuthorityIdentity(authority)
	return authority, snapshot
}

func rawZip(t *testing.T, entries []struct {
	name string
	mode os.FileMode
	data []byte
}) []byte {
	t.Helper()
	var buffer bytes.Buffer
	z := zip.NewWriter(&buffer)
	for _, entry := range entries {
		h := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		h.SetMode(entry.mode)
		w, err := z.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(entry.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestTerminologyManifestSnapshotStrictValidation(t *testing.T) {
	files := terminologyTestFiles()
	authority, snapshot := terminologyTestAuthority(t, files)
	if err := ValidateTerminologyAuthority(authority); err != nil {
		t.Fatal(err)
	}
	unknownParser := authority
	unknownParser.Sources = append([]TerminologySource{}, authority.Sources...)
	unknownParser.Sources[0].Parser = "unknown-visible/v1"
	unknownParser.Identity = terminologyAuthorityIdentity(unknownParser)
	if ValidateTerminologyAuthority(unknownParser) == nil {
		t.Fatal("unknown discovery parser accepted")
	}
	again, err := writeTerminologySnapshot(files)
	if err != nil || !bytes.Equal(snapshot, again) {
		t.Fatalf("snapshot is not reproducible: %v", err)
	}
	loaded, err := readTerminologySnapshot(authority, snapshot)
	if err != nil || len(loaded) != 25 {
		t.Fatalf("snapshot: files=%d err=%v", len(loaded), err)
	}
	encoded, err := Encode(authority)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range [][]byte{
		bytes.Replace(encoded, []byte(`"schema":`), []byte(`"unknown":1,"schema":`), 1),
		bytes.Replace(encoded, []byte(`"schema":`), []byte(`"schema":"duplicate","schema":`), 1),
		append(append([]byte{}, encoded...), []byte("{}")...),
	} {
		var parsed TerminologyDiscoveryAuthority
		if StrictJSON(raw, &parsed) == nil {
			t.Fatal("strict manifest parser accepted unknown/duplicate/trailing JSON")
		}
	}

	entries := make([]struct {
		name string
		mode os.FileMode
		data []byte
	}, 0, len(authority.Sources))
	for _, source := range authority.Sources {
		entries = append(entries, struct {
			name string
			mode os.FileMode
			data []byte
		}{source.SnapshotPath, 0644, files[source.SnapshotPath]})
	}

	t.Run("duplicate", func(t *testing.T) {
		bad := append([]struct {
			name string
			mode os.FileMode
			data []byte
		}{}, entries[:len(entries)-1]...)
		bad = append(bad, entries[0])
		if _, err := readTerminologySnapshot(authority, rawZip(t, bad)); err == nil {
			t.Fatal("duplicate snapshot member accepted")
		}
	})
	t.Run("traversal", func(t *testing.T) {
		bad := append([]struct {
			name string
			mode os.FileMode
			data []byte
		}{}, entries...)
		bad[0].name = "../escape"
		if _, err := readTerminologySnapshot(authority, rawZip(t, bad)); err == nil {
			t.Fatal("traversal snapshot member accepted")
		}
	})
	t.Run("symlink", func(t *testing.T) {
		bad := append([]struct {
			name string
			mode os.FileMode
			data []byte
		}{}, entries...)
		bad[0].mode = os.ModeSymlink | 0777
		if _, err := readTerminologySnapshot(authority, rawZip(t, bad)); err == nil {
			t.Fatal("symlink snapshot member accepted")
		}
	})
	t.Run("oversize", func(t *testing.T) {
		badAuthority := authority
		badAuthority.Sources = append([]TerminologySource{}, authority.Sources...)
		badAuthority.Sources[0].Bytes = 16<<20 + 1
		badAuthority.Sources[0].SHA256 = digest(bytes.Repeat([]byte("x"), 16<<20+1))
		bad := append([]struct {
			name string
			mode os.FileMode
			data []byte
		}{}, entries...)
		bad[0].data = bytes.Repeat([]byte("x"), 16<<20+1)
		if _, err := readTerminologySnapshot(badAuthority, rawZip(t, bad)); err == nil {
			t.Fatal("oversize snapshot member accepted")
		}
	})
}

func gitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func writeWebsiteFixture(t *testing.T, root string) {
	t.Helper()
	for _, p := range terminologyWebsitePaths {
		name := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte("# Visible terminology\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTerminologyWebsitePinnedCleanCheckout(t *testing.T) {
	root := t.TempDir()
	gitOutput(t, root, "init", "-q")
	gitOutput(t, root, "config", "user.name", "fixture")
	gitOutput(t, root, "config", "user.email", "fixture@example.test")
	writeWebsiteFixture(t, root)
	gitOutput(t, root, "add", ".")
	gitOutput(t, root, "commit", "-qm", "fixture")
	head := gitOutput(t, root, "rev-parse", "HEAD")
	if files, err := readWebsiteTerminologySources(root, head); err != nil || len(files) != len(terminologyWebsitePaths) {
		t.Fatalf("clean pinned checkout: files=%d err=%v", len(files), err)
	}
	if _, err := readWebsiteTerminologySources(root, strings.Repeat("0", 40)); err == nil {
		t.Fatal("website HEAD mismatch accepted")
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(terminologyWebsitePaths[0])), []byte("dirty\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readWebsiteTerminologySources(root, head); err == nil {
		t.Fatal("dirty website checkout accepted")
	}
	gitOutput(t, root, "checkout", "--", terminologyWebsitePaths[0])
	first := filepath.Join(root, filepath.FromSlash(terminologyWebsitePaths[0]))
	if err := os.Remove(first); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, filepath.FromSlash(terminologyWebsitePaths[1])), first); err != nil {
		t.Fatal(err)
	}
	gitOutput(t, root, "add", "-A")
	gitOutput(t, root, "commit", "-qm", "symlink fixture")
	symlinkHead := gitOutput(t, root, "rev-parse", "HEAD")
	if _, err := readWebsiteTerminologySources(root, symlinkHead); err == nil {
		t.Fatal("website source symlink accepted")
	}
}

func makeGoArchive(t *testing.T, mutate func(*tar.Header, *[]byte, int)) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	tw := tar.NewWriter(gz)
	for i, p := range terminologyGoPaths {
		data := []byte("// Package sample documents terminology.\npackage sample\n")
		h := &tar.Header{Name: p, Mode: 0644, Size: int64(len(data)), Typeflag: tar.TypeReg}
		if mutate != nil {
			mutate(h, &data, i)
			h.Size = int64(len(data))
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			if _, err := tw.Write(data); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestTerminologyGoArchivePinAndSafety(t *testing.T) {
	archive := makeGoArchive(t, nil)
	name := filepath.Join(t.TempDir(), "go.src.tar.gz")
	if err := os.WriteFile(name, archive, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readGoTerminologySources(name, strings.Repeat("0", 64)); err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
		t.Fatalf("archive SHA mismatch not rejected: %v", err)
	}
	if files, err := readGoTerminologySources(name, digest(archive)); err != nil || len(files) != len(terminologyGoPaths) {
		t.Fatalf("valid archive: files=%d err=%v", len(files), err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*tar.Header, *[]byte, int)
	}{
		{"traversal", func(h *tar.Header, _ *[]byte, i int) {
			if i == 0 {
				h.Name = "../escape"
			}
		}},
		{"symlink", func(h *tar.Header, data *[]byte, i int) {
			if i == 0 {
				h.Typeflag = tar.TypeSymlink
				h.Linkname = "/tmp/escape"
				*data = nil
			}
		}},
		{"duplicate", func(h *tar.Header, _ *[]byte, i int) {
			if i == 1 {
				h.Name = terminologyGoPaths[0]
			}
		}},
		{"missing exact path", func(h *tar.Header, _ *[]byte, i int) {
			if i == 0 {
				h.Name = "go/extra/not-selected.go"
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			bad := makeGoArchive(t, test.mutate)
			path := filepath.Join(t.TempDir(), "bad.tar.gz")
			if err := os.WriteFile(path, bad, 0644); err != nil {
				t.Fatal(err)
			}
			if _, err := readGoTerminologySources(path, digest(bad)); err == nil {
				t.Fatal("unsafe Go archive accepted")
			}
		})
	}
}

func TestTerminologyDeterministicParserProjection(t *testing.T) {
	markdown := []byte("---\ntitle: Visible title\nsummary: ignored metadata\n---\n\nVisible prose with `go test` and [documentation](/doc/).\n\n```go\nsecretCode()\n```\n\n<script>secretScript()</script>\n")
	contexts, err := projectMarkupTerminology(TerminologySource{Path: "x.md", Parser: "markdown-visible/v1"}, markdown)
	if err != nil {
		t.Fatal(err)
	}
	encoded := fmt.Sprint(contexts)
	for _, visible := range []string{"Visible title", "Visible prose", "documentation"} {
		if !strings.Contains(encoded, visible) {
			t.Fatalf("missing visible projection %q: %s", visible, encoded)
		}
	}
	for _, excluded := range []string{"ignored metadata", "secretCode", "secretScript"} {
		if strings.Contains(encoded, excluded) {
			t.Fatalf("machine/code content projected: %q", excluded)
		}
	}
	if !strings.Contains(encoded, "go test") || !strings.Contains(encoded, "/doc/") {
		t.Fatalf("compact protected context missing: %s", encoded)
	}

	goSource := []byte(`// Package sample explains package terminology.
package sample

// internal implementation note must not be projected.
var internal = "secretImplementation"

// ExportedThing explains an exported declaration.
//
//	secretCodeExample()
var ExportedThing = 1

func hidden() { secretBody() }
`)
	goContexts, err := projectGoDocTerminology("sample.go", goSource)
	if err != nil {
		t.Fatal(err)
	}
	goEncoded := fmt.Sprint(goContexts)
	if !strings.Contains(goEncoded, "package terminology") || !strings.Contains(goEncoded, "exported declaration") {
		t.Fatalf("package/exported docs missing: %s", goEncoded)
	}
	for _, excluded := range []string{"implementation note", "secretImplementation", "secretCodeExample", "secretBody"} {
		if strings.Contains(goEncoded, excluded) {
			t.Fatalf("ordinary implementation/code projected: %q", excluded)
		}
	}
	if _, err := projectGoDocTerminology("bad.go", []byte("package")); err == nil {
		t.Fatal("malformed Go source accepted")
	}
	if _, err := projectMarkupTerminology(TerminologySource{Path: "bad.md", Parser: "markdown-visible/v1"}, []byte("---\ntitle: broken\n")); err == nil {
		t.Fatal("malformed website metadata accepted")
	}
}

func TestTerminologyPurposeIdentityAndConsumerIsolation(t *testing.T) {
	files := terminologyTestFiles()
	authority, snapshot := terminologyTestAuthority(t, files)
	discovery, err := ProjectTerminologyContributors(authority, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	for _, contributor := range discovery {
		if contributor.Purpose != i18n.ContributorPurposeTerminologyDiscovery || contributor.Structured {
			t.Fatalf("discovery contributor escaped isolation: %+v", contributor)
		}
	}
	if groups, err := structuredContributorGroups(discovery); err != nil || len(groups) != 0 {
		t.Fatalf("discovery entered structured plan: groups=%v err=%v", groups, err)
	}
	malicious := discovery[0]
	malicious.Structured = true
	if _, err := structuredContributorGroups([]i18n.LanguageContributor{malicious}); err == nil {
		t.Fatal("structured discovery contributor accepted")
	}

	active, err := BuildUnifiedCorpus(repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if active.Schema != i18n.GlossaryCorpusSchema {
		t.Fatalf("inactive infrastructure changed current corpus schema: %s", active.Schema)
	}
	for _, contributor := range active.Contributors {
		if i18n.EffectiveContributorPurpose(contributor) != i18n.ContributorPurposeActiveContent {
			t.Fatalf("old active contributor lost active-content semantics: %s", contributor.ID)
		}
	}
	candidate, err := buildUnifiedCorpus(repositoryRoot(t), &authority, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Schema != i18n.GlossaryCorpusSchemaV2 || candidate.Identity == active.Identity {
		t.Fatal("projected discovery language did not change v2 corpus identity")
	}
	for _, contributor := range candidate.Contributors {
		if contributor.Purpose == "" {
			t.Fatalf("v2 contributor purpose missing: %s", contributor.ID)
		}
	}
	invalid := *candidate
	invalid.Contributors = append([]i18n.LanguageContributor{}, candidate.Contributors...)
	invalid.Contributors[0].Purpose = "unknown-purpose"
	invalid.Contributors[0].SourceSHA = i18n.ContributorIdentity(invalid.Contributors[0])
	invalid.Identity = i18n.CorpusIdentity(invalid)
	if i18n.ValidateGlossaryCorpus(&invalid) == nil {
		t.Fatal("unknown contributor purpose accepted")
	}

	global, err := LoadCurrent(repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	packages := []string{}
	for _, pkg := range global.Packages {
		packages = append(packages, pkg.ID)
	}
	if !reflect.DeepEqual(packages, []string{"tour-v1", "site-v2-shell", "learn-docs-v1"}) {
		t.Fatalf("terminology discovery changed package coverage/runtime registry: %v", packages)
	}
	states := PackageStates(global, &Locale{Schema: LocaleSchema, Locale: "fixture", Packages: []Completion{}})
	if len(states) != 3 {
		t.Fatalf("terminology discovery created completion work: %+v", states)
	}
	for _, source := range global.Sources {
		if source.Package == terminologyPackage {
			t.Fatal("terminology discovery entered active source/routes")
		}
	}
}

func TestTerminologyProjectedVersusRawIdentity(t *testing.T) {
	base := []byte("// Package sample documents visible language.\npackage sample\nfunc hidden() { println(1) }\n")
	rawChanged := bytes.Replace(base, []byte("println(1)"), []byte("println(2)"), 1)
	visibleChanged := bytes.Replace(base, []byte("visible language"), []byte("changed language"), 1)
	project := func(b []byte) i18n.LanguageContributor {
		contexts, err := projectGoDocTerminology("sample.go", b)
		if err != nil {
			t.Fatal(err)
		}
		contributor := i18n.LanguageContributor{ID: "discovery:go-source:sample.go", Parser: "go-doc-visible/v1", Package: terminologyPackage, Path: "go-source/sample.go", Purpose: i18n.ContributorPurposeTerminologyDiscovery, Contexts: contexts}
		contributor.SourceSHA = i18n.ContributorIdentity(contributor)
		return contributor
	}
	first, machine, visible := project(base), project(rawChanged), project(visibleChanged)
	if !reflect.DeepEqual(first, machine) {
		t.Fatal("machine-only Go bytes changed projected contributor identity")
	}
	if reflect.DeepEqual(first, visible) {
		t.Fatal("visible Go documentation did not change projected contributor identity")
	}
	corpusIdentity := func(contributor i18n.LanguageContributor) string {
		corpus := i18n.GlossarySourceCorpus{Schema: i18n.GlossaryCorpusSchemaV2, Contributors: []i18n.LanguageContributor{contributor}, Sources: []i18n.GlossaryArchiveReference{}}
		corpus.Identity = i18n.CorpusIdentity(corpus)
		return corpus.Identity
	}
	if corpusIdentity(first) != corpusIdentity(machine) || corpusIdentity(first) == corpusIdentity(visible) {
		t.Fatal("raw/projected corpus identity contract violated")
	}
}

func TestTerminologyCorpusChangeStalesOldUnifiedReview(t *testing.T) {
	root := siteFixture(t, "current-authorization-fresh-fixture")
	if _, err := i18n.RequireUnifiedGlossaryReview(root, "zh-CN"); err != nil {
		t.Fatal(err)
	}
	current, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		t.Fatal(err)
	}
	next := *current
	next.Schema = i18n.GlossaryCorpusSchemaV2
	next.Contributors = append([]i18n.LanguageContributor{}, current.Contributors...)
	for i := range next.Contributors {
		next.Contributors[i].Purpose = i18n.ContributorPurposeActiveContent
		next.Contributors[i].SourceSHA = i18n.ContributorIdentity(next.Contributors[i])
	}
	discovery := i18n.LanguageContributor{ID: "discovery:website:fixture.md", Parser: "markdown-visible/v1", Package: terminologyPackage, Path: "website/fixture.md", Purpose: i18n.ContributorPurposeTerminologyDiscovery, Contexts: []i18n.LanguageContext{{ID: "context", Text: "New governance-significant terminology.", Protected: []string{}}}}
	discovery.SourceSHA = i18n.ContributorIdentity(discovery)
	next.Contributors = append(next.Contributors, discovery)
	// The synthetic ID sorts before the first active source in this fixture.
	sort.Slice(next.Contributors, func(i, j int) bool { return next.Contributors[i].ID < next.Contributors[j].ID })
	next.Identity = i18n.CorpusIdentity(next)
	if err := i18n.ValidateGlossaryCorpus(&next); err != nil {
		t.Fatal(err)
	}
	b, err := Encode(next)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, i18n.GlossaryCorpusPath, b)
	if _, err := i18n.RequireUnifiedGlossaryReview(root, "zh-CN"); err == nil || !strings.Contains(err.Error(), "missing/stale") {
		t.Fatalf("old unified review remained current after corpus identity change: %v", err)
	}
}

func TestTerminologyOfflineCurrentCheckAndCAS(t *testing.T) {
	root := siteFixture(t, "current-authorization-fresh-fixture")
	files := terminologyTestFiles()
	authority, snapshot := terminologyTestAuthority(t, files)
	manifest, err := Encode(authority)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, TerminologyManifestPath, manifest)
	writeFixture(t, root, TerminologySnapshotPath, snapshot)
	corpus, err := buildUnifiedCorpus(root, &authority, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	corpusBytes, err := Encode(corpus)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, i18n.GlossaryCorpusPath, corpusBytes)
	summary, err := CheckTerminologyAuthority(root)
	if err != nil || !summary.Applied || summary.Sources != 25 || summary.CorpusIdentity != corpus.Identity {
		t.Fatalf("offline current-check: summary=%+v err=%v", summary, err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(TerminologySnapshotPath)), append(snapshot, 0), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckTerminologyAuthority(root); err == nil {
		t.Fatal("snapshot drift accepted")
	}

	casRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(casRoot, "data"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeCASFile(casRoot, "data/authority", []byte("first"), "", true); err != nil {
		t.Fatal(err)
	}
	if err := writeCASFile(casRoot, "data/authority", []byte("wrong"), strings.Repeat("0", 64), false); err == nil {
		t.Fatal("wrong CAS prior accepted")
	}
	current, err := os.ReadFile(filepath.Join(casRoot, "data", "authority"))
	if err != nil || string(current) != "first" {
		t.Fatalf("failed CAS mutated target: %q %v", current, err)
	}
	if err := writeCASFile(casRoot, "data/authority", []byte("second"), digest(current), false); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(casRoot, "data", "unrelated"), []byte("owned"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeCASFile(casRoot, "data/unrelated", []byte("replacement"), "", true); err == nil {
		t.Fatal("initial creation overwrote unrelated authority")
	}
}
