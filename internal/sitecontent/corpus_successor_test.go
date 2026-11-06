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

const shellCorpusAffectedInput = ShellSuccessorRoot + "affected/zh-CN/b106897bf70ef8c8bd3d439bebca598ea911635364c8a1000a40b3ce83b599e2.json"

func corpusSuccessorFixture(t *testing.T) (string, string) {
	t.Helper()
	real := shellTestRoot(t)
	root := t.TempDir()
	copied := map[string]bool{}
	copyFile := func(name string) {
		if copied[name] {
			return
		}
		b, err := readRegular(real, name)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, name, b)
		copied[name] = true
	}
	// This legacy transition has an immutable basis. Real current corpus may
	// legitimately be stale between source replacement and corpus publication.
	raw, err := readRegular(real, shellCorpusAffectedInput)
	if err != nil {
		t.Fatal(err)
	}
	var s ShellAffectedScope
	if err := StrictJSON(raw, &s); err != nil {
		t.Fatal(err)
	}
	sealed := s
	sealed.Identity = ""
	if s.Identity != identity(sealed) || shellCorpusAffectedInput != ShellSuccessorRoot+"affected/zh-CN/"+s.Identity+".json" {
		t.Fatal("invalid historical fixture scope")
	}
	archive := "data/glossary-corpus-history/" + s.Terminology.CurrentCorpus.SHA256 + ".json"
	old, err := i18n.LoadGlossaryCorpusAt(real, archive)
	if err != nil {
		t.Fatal(err)
	}
	basis, err := readRegular(real, archive)
	if err != nil || digest(basis) != s.Terminology.CurrentCorpus.SHA256 || old.Identity != s.Terminology.CurrentIdentity {
		t.Fatal("historical fixture corpus binding", err)
	}
	writeFixture(t, root, i18n.GlossaryCorpusPath, basis)
	copyFile(archive)
	for _, r := range old.Sources {
		if r.Path != SnapshotPath {
			copyFile(r.Path)
		}
	}
	var a ShellSuccessorAuthority
	if err := readReference(real, s.SourceAuthority, &a); err != nil {
		t.Fatal(err)
	}
	// Use an isolated, resealed authority for the same frozen source projection;
	// current parser files must not masquerade as historical dependency bytes.
	for i, r := range a.ParserReferences {
		copyFile(r.Path)
		a.ParserReferences[i], err = ReferenceFile(root, r.Path)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, source := range a.Sources {
		copyFile(source.Path)
	}
	for _, name := range []string{SourceManifestPath, a.HistoricalHome.Path, a.TranslationBody.Path, "locales/zh-CN/glossary.yaml", "locales/zh-CN/locale.json"} {
		copyFile(name)
	}
	completion, err := readHistoricalActivationContext(real, s.Locale, i18n.GlossaryArchiveReference{Path: s.CurrentCompletion.Path, SHA256: s.CurrentCompletion.SHA256})
	if err != nil || digest(completion) != s.CurrentCompletion.SHA256 {
		t.Fatal("historical fixture completion binding", err)
	}
	writeFixture(t, root, s.CurrentCompletion.Path, completion)
	a.Identity = ""
	a.Identity = identity(a)
	s.SourceAuthority, err = shellAuthorityReference(&a)
	if err != nil {
		t.Fatal(err)
	}
	s.SourceIdentity = a.Identity
	if err := saveImmutable(root, s.SourceAuthority.Path, &a); err != nil {
		t.Fatal(err)
	}
	s.Identity = ""
	s.Identity = identity(s)
	input := ShellSuccessorRoot + "affected/zh-CN/" + s.Identity + ".json"
	if err := saveImmutable(root, input, &s); err != nil {
		t.Fatal(err)
	}
	paths, err := i18n.CorpusStorageRequirements(old)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range paths {
		copyFile(name)
	}
	// Installed successor currentness loads the frozen global scope, whose
	// source ZIP reconstruction consumes this exact registered subset (including
	// article/default templates), not just the corpus's linguistic Page paths.
	manifest, err := loadSourceManifest(real)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range manifest.Legacy.Paths {
		copyFile(adoptedSourcePath(name))
	}
	if _, err := reconstructSourceSubset(root, manifest); err != nil {
		t.Fatal("fixture frozen source subset must remain byte-exact", err)
	}
	// BuildGlobal also reconstructs the Tour catalog, including .play source
	// files. Copy only the frozen repository inventory, with exact SHA checks.
	copyFile(GlobalPath)
	var global Global
	globalBytes, err := readRegular(root, GlobalPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := StrictJSON(globalBytes, &global); err != nil {
		t.Fatal(err)
	}
	if err := ValidateGlobal(global); err != nil {
		t.Fatal(err)
	}
	for _, source := range global.Sources {
		if source.Origin == "repository" {
			copyFile(source.Path)
			if _, err := readRegularReference(root, Reference{source.Path, source.SHA256}); err != nil {
				t.Fatal("fixture repository source must remain byte-exact", err)
			}
		}
	}
	err = filepath.WalkDir(filepath.Join(real, "data/source-storage-compatibility"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			name, _ := filepath.Rel(real, path)
			copyFile(filepath.ToSlash(name))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return root, input
}

func TestCorpusSuccessorPreflightExactDeltaAndDeterminism(t *testing.T) {
	root, input := corpusSuccessorFixture(t)
	before, err := readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil {
		t.Fatal(err)
	}
	completion, err := readRegular(root, LocalePath("zh-CN"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := BuildCorpusSuccessor(root, input)
	if err != nil {
		t.Fatal(err)
	}
	again, err := BuildCorpusSuccessor(root, input)
	if err != nil || !reflect.DeepEqual(p, again) {
		t.Fatal("non-deterministic preflight", err)
	}
	var scope ShellAffectedScope
	if err := readReference(root, Reference{input, p.Affected.SHA256}, &scope); err != nil {
		t.Fatal(err)
	}
	if p.Source != scope.SourceAuthority || p.SourceIdentity != scope.SourceIdentity || p.Affected.Path != input || p.AffectedIdentity != scope.Identity || len(p.Additions) != 5 || p.ProposedContexts != 235 || len(p.Retirements) != 1 || p.Retirements[0].Path != "data/site-shell/home.html" || p.NetContexts != 230 {
		t.Fatal("incorrect exact successor delta")
	}
	old, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		t.Fatal(err)
	}
	oldByID := map[string]i18n.LanguageContributor{}
	for _, c := range old.Contributors {
		oldByID[c.ID] = c
	}
	for _, c := range p.Result.Contributors {
		if prior, ok := oldByID[c.ID]; ok && !reflect.DeepEqual(prior, c) {
			t.Fatal("unaffected contributor changed", c.ID)
		}
	}
	if p.UnchangedCount != 427 || len(p.Result.Contributors) != 432 {
		t.Fatal("contributor membership count changed")
	}
	after, _ := readRegular(root, i18n.GlossaryCorpusPath)
	if !bytes.Equal(before, after) {
		t.Fatal("preflight wrote corpus")
	}
	after, _ = readRegular(root, LocalePath("zh-CN"))
	if !bytes.Equal(completion, after) || digest(after) != "9bcdf339dfe64d182003bb77e49befb3ac4ca72e38a1210ffd7f3adb0de77879" {
		t.Fatal("preflight changed locale completion")
	}
	if _, err := os.Lstat(pathJoin(root, CorpusSuccessorPath(p))); !os.IsNotExist(err) {
		t.Fatal("preflight wrote successor evidence")
	}
	t.Logf("predecessor=%s file=%s next=%s file=%s contexts=%d delta=%d", p.PredecessorIdentity, p.Predecessor.SHA256, p.Result.Identity, p.ResultSHA, corpusContextCount(p.Result), p.NetContexts)
}

func corpusContextCount(c i18n.GlossarySourceCorpus) int {
	n := 0
	for _, x := range c.Contributors {
		n += len(x.Contexts)
	}
	return n
}

func TestCorpusSuccessorApplyCASAndReviewNonCurrent(t *testing.T) {
	root, input := corpusSuccessorFixture(t)
	p, err := BuildCorpusSuccessor(root, input)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := readRegular(root, i18n.GlossaryCorpusPath)
	locale, _ := readRegular(root, LocalePath("zh-CN"))
	if _, err = ApplyCorpusSuccessor(root, input, strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong CAS accepted")
	}
	if err = installCorpusSuccessor(root, p, p.Predecessor.SHA256); err != nil {
		t.Fatal(err)
	}
	current, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil || current.Identity != p.Result.Identity {
		t.Fatal("installed corpus cannot be loaded", err)
	}
	archive := "data/glossary-corpus-history/" + p.Predecessor.SHA256 + ".json"
	got, err := readRegular(root, archive)
	if err != nil || !bytes.Equal(got, before) {
		t.Fatal("archive bytes changed", err)
	}
	if _, err = i18n.LoadGlossaryCorpusAt(root, archive); err != nil {
		t.Fatal("exact archived storage alias rejected", err)
	}
	if installed, err := CheckAppliedCorpusSuccessor(root); err != nil || !installed {
		t.Fatal("installed successor currentness", err)
	}
	if err = CheckUnifiedCorpus(root); err != nil {
		t.Fatal("current corpus rejects installed successor", err)
	}
	if _, err = RefreshSharedCorpusAuthority(root, p.ResultSHA, false); err == nil {
		t.Fatal("legacy command can silently revert successor")
	}
	if err = installCorpusSuccessor(root, p, p.Predecessor.SHA256); err == nil {
		t.Fatal("replay bypasses predecessor CAS")
	}
	if err = SaveArtifact(root, archive, []byte("overwrite")); err == nil {
		t.Fatal("archive overwritten")
	}
	if err = SaveArtifact(root, CorpusSuccessorPath(p), []byte("overwrite")); err == nil {
		t.Fatal("successor evidence overwritten")
	}
	got, _ = readRegular(root, LocalePath("zh-CN"))
	if !bytes.Equal(got, locale) {
		t.Fatal("apply changed locale completion")
	}
	real := shellTestRoot(t)
	reviewPath := i18n.UnifiedGlossaryReviewPath("zh-CN", "20261011-zh-CN-glossary-review-010")
	reviewBytes, err := readRegular(real, reviewPath)
	if err != nil {
		t.Fatal(err)
	}
	var review i18n.UnifiedGlossaryReview
	if err = StrictJSON(reviewBytes, &review); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, reviewPath, reviewBytes)
	bundlePath := "data/unified-glossary-reviews/zh-CN-bundles/" + review.BundleSHA + ".zip"
	bundle, err := readRegular(real, bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, bundlePath, bundle)
	if _, err = i18n.RequireUnifiedGlossaryReview(root, "zh-CN"); err == nil || !strings.Contains(err.Error(), "missing/stale/ambiguous") {
		t.Fatal("old PASS remains current or unexpected failure", err)
	}
	got, _ = readRegular(root, reviewPath)
	if !bytes.Equal(got, reviewBytes) {
		t.Fatal("old review changed")
	}
}

func TestCorpusSuccessorMutationAndInputFailClosed(t *testing.T) {
	for _, kind := range []string{"concurrent", "pending", "symlink", "affected", "source", "parser", "corpus", "traversal"} {
		t.Run(kind, func(t *testing.T) {
			root, input := corpusSuccessorFixture(t)
			p, err := BuildCorpusSuccessor(root, input)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "concurrent":
				writeFixture(t, root, i18n.GlossaryCorpusPath, []byte("concurrent mutation"))
				if err = installCorpusSuccessor(root, p, p.Predecessor.SHA256); err == nil {
					t.Fatal("concurrent mutation accepted")
				}
				return
			case "pending":
				writeFixture(t, root, "data/.corpus-successor-pending", []byte(p.Identity))
				if _, err = i18n.LoadUnifiedGlossaryCorpus(root); err == nil {
					t.Fatal("unknown publication exposed corpus")
				}
				return
			case "symlink":
				name := pathJoin(root, "_content/index.md")
				if err = os.Remove(name); err != nil {
					t.Fatal(err)
				}
				if err = os.Symlink(filepath.Join(shellTestRoot(t), "_content/index.md"), name); err != nil {
					t.Fatal(err)
				}
			case "affected":
				raw, _ := readRegular(root, input)
				writeFixture(t, root, input, bytes.Replace(raw, []byte(`"schema":`), []byte(`"unknown":true,"schema":`), 1))
			case "source":
				raw, _ := readRegular(root, p.Source.Path)
				writeFixture(t, root, p.Source.Path, append(raw, '\n'))
			case "parser":
				writeFixture(t, root, "internal/sitecontent/shell_language_source.go", []byte("changed parser"))
			case "corpus":
				raw, _ := readRegular(root, i18n.GlossaryCorpusPath)
				writeFixture(t, root, i18n.GlossaryCorpusPath, bytes.Replace(raw, []byte(`"schema":`), []byte(`"unknown":true,"schema":`), 1))
			case "traversal":
				if _, err = BuildCorpusSuccessor(root, "../"+input); err == nil {
					t.Fatal("traversal accepted")
				}
				return
			}
			if _, err = BuildCorpusSuccessor(root, input); err == nil {
				t.Fatal("tampered authority accepted")
			}
		})
	}
}

func TestCorpusSuccessorRetirementNotPresentAndIndependentConsumer(t *testing.T) {
	root, input := corpusSuccessorFixture(t)
	a, s, _, err := readPendingShellCorpusInputs(root, input)
	if err != nil {
		t.Fatal(err)
	}
	old, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		t.Fatal(err)
	}
	filtered := *old
	filtered.Contributors = []i18n.LanguageContributor{}
	for _, c := range old.Contributors {
		if c.Path != "data/site-shell/home.html" {
			filtered.Contributors = append(filtered.Contributors, c)
		}
	}
	p, err := projectCorpusSuccessor(&filtered, a, s, old.Sources)
	if err != nil || len(p.Retirements) != 0 || !reflect.DeepEqual(p.NotPresent, []string{"data/site-shell/home.html"}) {
		t.Fatal("NOT_PRESENT fabricated a deletion", err)
	}
	copy := *old
	copy.Contributors = append([]i18n.LanguageContributor{}, old.Contributors...)
	for _, c := range old.Contributors {
		if c.Path == "data/site-shell/home.html" {
			c.ID = "independent-consumer"
			c.Package = "tour-v1"
			copy.Contributors = append(copy.Contributors, c)
			break
		}
	}
	if _, err = projectCorpusSuccessor(&copy, a, s, old.Sources); err == nil {
		t.Fatal("independent source consumer silently retired")
	}
}
