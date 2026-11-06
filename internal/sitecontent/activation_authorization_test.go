package sitecontent

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Copy the accepted artifacts; every mutation below stays in this test's
// isolated repository. No generation, reviewer, activation apply or re-sealing.
func activationAuthorizationFixture(t *testing.T) string {
	t.Helper()
	return copiedCanonicalLocaleFixture(t)
}

// Reuse the canonical source/evidence copy for read-only locale consumers too.
// This helper only copies bytes into t.TempDir; it never runs activation.
func copiedCanonicalLocaleFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	repository := repositoryRoot(t)
	copyAuthorities := append([]string{}, PackageAuthorities...)
	for _, name := range copyAuthorities {
		b, err := os.ReadFile(filepath.Join(repository, name))
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, name, b)
	}
	for _, name := range []string{"_content", "internal", "docs", "go.mod", "go.sum", "UPSTREAM_MANIFEST.tsv", "production/identity.json", "locales/zh-CN", GlobalPath, PageRegistryPath, i18n.GlossaryCorpusPath, "data/tour-pages.tsv", "data/tour-conditional-pages.tsv", "data/tour-examples.tsv", SourceManifestPath, "data/site-source-contracts", "data/site-shell", "data/source-storage-compatibility", "data/glossary-corpus-history", "data/unified-glossary-corpus-successors", "data/translation-authorizations", "data/generation-archives/zh-CN", "data/site-workflow/zh-CN", "data/locale-language/zh-CN", "data/quality-check-reopens/zh-CN", "data/translation-unit-replacements/zh-CN", "data/translation-unit-reconciliations/zh-CN", "data/representation-reconciliation-reviews/zh-CN", "data/quality-check-snapshots/zh-CN", "data/retranslation-runs/zh-CN", "data/glossary-history/zh-CN", "data/glossary-compatibility/zh-CN", "data/glossary-reviews/zh-CN", "data/unified-glossary-reviews/zh-CN", "data/unified-glossary-reviews/zh-CN-bundles", "data/locale-surface-reviews/zh-CN", "data/course-seo", "data/surface-config-history", "data/glossary-review-legacy-coverage.json", "data/terminology-discovery-sources.json", "data/terminology-discovery-sources.zip", "data/promotion-recoveries/zh-CN"} {
		// Legacy stores may be absent; actual readers still require every ref.
		if _, err := os.Lstat(filepath.Join(repository, name)); os.IsNotExist(err) {
			continue
		}
		if err := filepath.WalkDir(filepath.Join(repository, name), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(repository, p)
			if err != nil {
				return err
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			writeFixture(t, root, filepath.ToSlash(rel), b)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestActivationUsesProspectivePackageAuthorization(t *testing.T) {
	root := activationAuthorizationFixture(t)
	gate, err := ReferenceFile(root, "data/locale-language/zh-CN/surface/20261014-zh-CN-integrated-surface-faa455c8-002.json")
	if err != nil {
		t.Fatal(err)
	}
	var r IntegratedSurfaceReceipt
	if err := readReference(root, gate, &r); err != nil {
		t.Fatal(err)
	}
	cat, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackages(root, cat); err != nil {
		t.Fatal(err)
	}
	AttachIntegratedSurfaceAuthority(cat)
	scoped, registered, err := i18n.SelectFinalizedTranslationPackage(root, cat, r.Scope.Locale, "learn-docs-v1-qc-012")
	if err != nil || !registered {
		t.Fatal("package diagnostic", err)
	}
	a, err := i18n.RequireCurrentTranslationAuthorization(root, scoped, r.Scope.Locale, "learn-docs-v1-qc-012")
	if err != nil || !a.Ready() {
		t.Fatal("current authorization", err)
	}
	old, err := i18n.AssessCurrentTranslation(root, scoped, r.Scope.Locale, "learn-docs-v1-qc-010")
	if err != nil || old.Affected == 0 {
		t.Fatal("fixture must retain genuinely affected historical baseline", err)
	}
	corpusBytes, err := readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil {
		t.Fatal(err)
	}
	var installed CorpusSuccessor
	entries, err := os.ReadDir(pathJoin(root, CorpusSuccessorRoot))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		var p CorpusSuccessor
		b, err := readRegular(root, CorpusSuccessorRoot+entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if err := StrictJSON(b, &p); err != nil {
			t.Fatal(err)
		}
		if p.ResultSHA == digest(corpusBytes) {
			installed = p
		}
	}
	if installed.Identity == "" {
		t.Fatal("fixture must include installed corpus successor")
	}
	// The real repository may already have legally activated both closures.
	// Start the isolated sequential regression at the receipt's exact prior.
	g, err := reviewedActivationGlobal(root)
	if err != nil {
		t.Fatal(err)
	}
	state, err := readLocaleAuthority(root, g, r.Scope.Locale)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := reviewedActivationBaseline(root, g, r, *state)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, LocalePath(r.Scope.Locale), baseline)
	sealed := map[string][]byte{}
	for _, p := range []string{gate.Path, r.Evidence.Path, i18n.CurrentTranslationAuthorizationPath(a), a.Historical.Finalization.SnapshotManifestPath, "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-012/finalization.json"} {
		b, err := readRegular(root, p)
		if err != nil {
			t.Fatal(err)
		}
		sealed[p] = b
	}
	for _, ref := range r.Scope.Closures {
		b, err := readRegular(root, ref.Path)
		if err != nil {
			t.Fatal(err)
		}
		sealed[ref.Path] = b
	}
	// Preflight must authorize today's exact package and preserve every other
	// completion. Simulate the returned CAS bytes only in the isolated fixture.
	for _, ref := range r.Scope.Closures {
		prior, _ := readRegular(root, LocalePath(r.Scope.Locale))
		plan, err := ActivationPreflight(root, ref)
		if err != nil {
			t.Fatal("prospective preflight", err)
		}
		after, _ := readRegular(root, LocalePath(r.Scope.Locale))
		if !bytes.Equal(prior, after) || plan.PriorSHA != digest(prior) || plan.Completion.Evidence[1] != gate {
			t.Fatal("preflight mutation/CAS/Surface mismatch")
		}
		writeFixture(t, root, LocalePath(r.Scope.Locale), plan.result)
		if err := proveReviewedActivationTransition(root, gate, r); err != nil {
			t.Fatal("sequential replacement transition", err)
		}
		if _, _, _, err := readPendingShellCorpusInputs(root, installed.Affected.Path); err == nil {
			t.Fatal("pending corpus creation accepted noncurrent completion")
		}
		if applied, err := CheckAppliedCorpusSuccessor(root); err != nil || !applied {
			t.Fatal("installed corpus replayed mutable predecessor", err)
		}
	}
	checked, err := i18n.RequireCurrentTranslationAuthorization(root, scoped, r.Scope.Locale, "learn-docs-v1-qc-012")
	if err != nil || checked.Identity != a.Identity {
		t.Fatal("Surface/activation changed TU classification", err)
	}
	for p, b := range sealed {
		after, _ := readRegular(root, p)
		if !bytes.Equal(b, after) {
			t.Fatal("trusted evidence rewritten", p)
		}
	}
}
