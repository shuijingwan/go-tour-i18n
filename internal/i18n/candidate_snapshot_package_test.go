package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// A single registered physical Page, using the existing test-only adapter.
// The production Homepage successor is separately checked read-only through
// its real Shell adapter/authority, without generating or recording evidence.
func packageSnapshotFixture(t *testing.T) (string, *Catalog, string) {
	t.Helper()
	root, source, ids := futureContentFixture(t)
	d := source.TranslationPackages[0]
	d.ID, d.AuthoritySHA256, d.Units = "site-v2-shell", sum([]byte("synthetic-frozen-package/v1")), d.Units[:1]
	c := &Catalog{}
	if err := c.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	e, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitIDs: ids[:1], Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	b, m, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	_, files, err := readGenerationBundle(b)
	if err != nil {
		t.Fatal(err)
	}
	raw := files["formal/inputs/"+filepath.Base(m.ExpectedOutputs[0].BundlePath)]
	raw = bytes.ReplaceAll(raw, []byte("A future card"), []byte("未来卡片"))
	raw = bytes.ReplaceAll(raw, []byte("Read this message."), []byte("阅读此消息。"))
	futureImport(t, root, c, b, map[string][]byte{ids[0]: raw})
	if r, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID}); err != nil || r.ValidationPassed != 1 {
		t.Fatal(r, err)
	}
	return root, c, ids[0]
}

func TestPackageCandidateSnapshotScopeAndLegacyReader(t *testing.T) {
	root, scoped, id := packageSnapshotFixture(t)
	c := *scoped
	c.TranslationScope = nil
	o := QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "explicit-shell", ContentPackage: "site-v2-shell"}
	p, err := PrepareQualityCheckCandidateSnapshot(root, &c, o)
	if err != nil || p.UnitCount != 1 || p.PageCount != 1 || p.ExampleCount != 0 || p.Units[0].UnitID != id || p.ContentPackage == nil || !reflect.DeepEqual(p.ContentPackage.UnitIDs, []string{id}) {
		t.Fatalf("preflight=%+v err=%v", p, err)
	}
	if len(c.TranslationScope) != 0 {
		t.Fatal("package selection mutated caller catalog")
	}
	if _, err := os.Stat(filepath.Join(root, "data/quality-check-snapshots/zh-CN", o.SnapshotID)); !os.IsNotExist(err) {
		t.Fatal("readonly preflight created Snapshot")
	}
	m, path, err := CreateQualityCheckCandidateSnapshot(root, &c, o)
	if err != nil || !reflect.DeepEqual(p, m) {
		t.Fatal(m, err)
	}
	before, _ := os.ReadFile(filepath.Join(root, path))
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, &c, o); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatal("existing Snapshot not rejected", err)
	}
	after, _ := os.ReadFile(filepath.Join(root, path))
	if !bytes.Equal(before, after) {
		t.Fatal("existing Snapshot overwritten")
	}
	if _, err := readQualityCheckSnapshot(root, o.Locale, o.SnapshotID, true, scoped); err != nil {
		t.Fatal("downstream selected package reader", err)
	}
	if _, err := readQualityCheckSnapshot(root, o.Locale, o.SnapshotID, true, &c); err == nil {
		t.Fatal("package Snapshot admitted to default Tour reader")
	}
	selected, ok, err := selectSnapshotTranslationPackage(root, &c, o.Locale, o.SnapshotID)
	if err != nil || !ok || !reflect.DeepEqual(selected.TranslationScope, []string{id}) {
		t.Fatal("explicit bound scope discovery failed", selected, ok, err)
	}
	// Legacy manifests retain their original reader; no implicit schema rewrite.
	legacy, _, err := CreateQualityCheckCandidateSnapshot(root, scoped, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "legacy-scoped"})
	if err != nil || legacy.ContentPackage != nil {
		t.Fatal("legacy scoped API/manifest changed", legacy, err)
	}
	if _, err := readQualityCheckSnapshot(root, "zh-CN", "legacy-scoped", true, scoped); err != nil {
		t.Fatal(err)
	}
}

func TestPackageCandidateSnapshotRejectsWrongPartialAndMixedScope(t *testing.T) {
	root, c, ids := futureContentFixture(t)
	c.TranslationPackages[0].AuthoritySHA256 = sum([]byte("complete registered authority"))
	for _, pkg := range []string{"unknown", "site-v2-shell"} {
		if _, err := PrepareQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "wrong-package", ContentPackage: pkg}); err == nil {
			t.Fatal("unregistered package accepted")
		}
	}
	for _, scope := range [][]string{ids[:1], {ids[0], ids[0]}, {ids[2], ids[1], ids[0]}} {
		partial := *c
		partial.TranslationScope = scope
		if _, err := PrepareQualityCheckCandidateSnapshot(root, &partial, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "partial", ContentPackage: c.TranslationPackages[0].ID}); err == nil {
			t.Fatal("partial/duplicate/reordered scope accepted")
		}
	}
	missing := *c
	missing.TranslationPackages = append([]TranslationPackageDescriptor(nil), c.TranslationPackages...)
	missing.TranslationPackages[0].AuthoritySHA256 = ""
	if _, _, err := snapshotPackageCatalog(&missing, missing.TranslationPackages[0].ID); err == nil {
		t.Fatal("package without frozen source authority accepted")
	}
}

func TestPackageCandidateSnapshotCommitDrift(t *testing.T) {
	for _, kind := range []string{"package", "source", "candidate", "validation", "mode"} {
		t.Run(kind, func(t *testing.T) {
			root, c, _ := packageSnapshotFixture(t)
			o := QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "must-not-publish", ContentPackage: "site-v2-shell"}
			var checks []func() error
			m, err := prepareQualityCheckCandidateSnapshot(root, c, o, &checks)
			if err != nil {
				t.Fatal(err)
			}
			path := m.Units[0].SourcePath
			switch kind {
			case "package":
				c.TranslationPackages[0].AuthoritySHA256 = sum([]byte("changed authority"))
			case "candidate":
				path = m.Units[0].CandidatePath
			case "validation":
				path = m.Units[0].ValidationPath
			}
			if kind == "mode" {
				if err := os.Chmod(filepath.Join(root, path), 0700); err != nil {
					t.Fatal(err)
				}
			} else if kind != "package" {
				b, _ := os.ReadFile(filepath.Join(root, path))
				writeSnapshotSource(t, root, path, append(b, ' '))
			}
			if _, _, err := writeQualityCheckCandidateSnapshot(root, o, m, checks...); err == nil {
				t.Fatal("mid-invocation drift accepted")
			}
			if _, err := os.Stat(filepath.Join(root, "data/quality-check-snapshots/zh-CN", o.SnapshotID)); !os.IsNotExist(err) {
				t.Fatal("failed preflight published Snapshot")
			}
		})
	}
}

func TestPackageCandidateSnapshotBindingTamper(t *testing.T) {
	root, c, _ := packageSnapshotFixture(t)
	o := QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "bound", ContentPackage: "site-v2-shell"}
	m, path, err := CreateQualityCheckCandidateSnapshot(root, c, o)
	if err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(filepath.Join(root, path))
	for _, kind := range []string{"package", "authority", "scope", "member", "refs", "source"} {
		t.Run(kind, func(t *testing.T) {
			var s QualityCheckSnapshotManifest
			if err := json.Unmarshal(original, &s); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "package":
				s.ContentPackage.ID = "wrong"
			case "authority":
				s.ContentPackage.AuthoritySHA256 = sum([]byte("wrong"))
			case "scope":
				s.ContentPackage.ScopeIdentitySHA256 = sum([]byte("wrong"))
			case "member":
				s.ContentPackage.UnitIDs[0] = "wrong"
			case "refs":
				s.ContentPackage.AuthorityReferences = nil
			case "source":
				s.Units[0].SourceSHA256 = sum([]byte("wrong"))
			}
			if err := writeTranslationJSON(filepath.Join(root, path), &s); err != nil {
				t.Fatal(err)
			}
			if _, err := readQualityCheckSnapshot(root, "zh-CN", m.SnapshotID, true, c); err == nil {
				t.Fatal("tampered package binding accepted")
			}
		})
	}
}
