package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func shellReplacementProofFixture(t *testing.T) (string, *i18n.Catalog, ShellPageAdapter, ShellPageAdapter, HistoricalPageRepresentations) {
	t.Helper()
	root := t.TempDir()
	raw, err := readRegular(repositoryRoot(t), "_content/index.md")
	if err != nil {
		t.Fatal(err)
	}
	oldSource, err := ExtractShellLanguageSource("_content/index.md", raw)
	if err != nil {
		t.Fatal(err)
	}
	nextSource, err := ExtractShellPageRepresentation("_content/index.md", raw)
	if err != nil {
		t.Fatal(err)
	}
	old, next := ShellPageAdapter{*oldSource, raw}, ShellPageAdapter{*nextSource, raw}
	pkg := i18n.TranslationPackageDescriptor{ID: "shell-proof-fixture", AuthoritySHA256: digest([]byte("package"))}
	pkg.Units = []i18n.AdaptedTranslationUnit{{Unit: i18n.TranslationUnit{ID: nextSource.Page.Units[0].ID, Kind: i18n.UnitKindPage, SourcePath: nextSource.Path, SourceSHA256: digest(raw), Source: raw}, Adapter: next, StableIndex: 1, Route: "/"}}
	c := &i18n.Catalog{}
	if err := c.RegisterTranslationPackage(pkg); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage(pkg.ID); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, nextSource.Path, raw)
	frozen := HistoricalPageRepresentations{Package: pkg.ID, PackageSHA256: pkg.AuthoritySHA256, Documents: []Document{*oldSource.Page}}
	return root, c, old, next, frozen
}

func TestShellPageReplacementProof(t *testing.T) {
	root, c, old, next, frozen := shellReplacementProofFixture(t)
	ref := freezeReplacementFixture(t, root, frozen)
	r, err := resolvePageReplacements(root, c, ref)
	if err != nil {
		t.Fatal(err)
	}
	p := r.Units[old.Source.Page.Units[0].ID]
	if p.CurrentUnitID != next.Source.Page.Units[0].ID || p.OldUnitID == p.CurrentUnitID || p.OldInputSHA256 == p.CurrentInputSHA256 {
		t.Fatal("missing exact replacement", p)
	}
	u, err := r.HistoricalCatalog.Unit(p.OldUnitID)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := u.Adapter.(ShellPageAdapter); !ok {
		t.Fatal("historical adapter lost shell validation")
	}
	raw := []byte(old.Source.Page.Units[0].Source)
	got, err := next.ReconcileRepresentation(u.Adapter, raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := old.Restore(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	b, err := next.Restore(string(got))
	if err != nil || !bytes.Equal(a, b) {
		t.Fatal("changed output", err)
	}
	if err := next.Validate(b, &i18n.Glossary{}); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"unit", "source", "route", "structure", "slot"} {
		t.Run(field, func(t *testing.T) {
			data, _ := json.Marshal(frozen)
			var bad HistoricalPageRepresentations
			_ = json.Unmarshal(data, &bad)
			d := &bad.Documents[0]
			switch field {
			case "unit":
				d.Units[0].ID = strings.Repeat("0", 64)
			case "source":
				d.SourceSHA = strings.Repeat("0", 64)
			case "route":
				d.Route = "/unrelated"
			case "structure":
				d.StructureSHA = strings.Repeat("0", 64)
			case "slot":
				d.Slots[0].Source += "changed"
			}
			if _, err := resolvePageReplacements(root, c, freezeReplacementFixture(t, root, bad)); err == nil {
				t.Fatal("accepted tamper")
			}
		})
	}
	if _, err := next.ReconcileRepresentation(old, []byte("missing tokens")); err == nil {
		t.Fatal("accepted raw tamper")
	}
	bad := next
	bad.Raw = append(bytes.Clone(next.Raw), 'x')
	if _, err := bad.ReconcileRepresentation(old, raw); err == nil {
		t.Fatal("accepted source drift")
	}
	if _, err := next.ReconcileRepresentation(PageAdapter{*old.Source.Page, old.Raw}, raw); err == nil {
		t.Fatal("accepted wrong predecessor adapter")
	}
}

// Explicit, read-only exact-machine preflight. No receipts, Snapshot or
// authorization are created; the existing shared workflow does all proofs.
func TestShellRepresentationRecoveryReadonly(t *testing.T) {
	if os.Getenv("GO_TOUR_SHELL_RECONCILIATION_PREFLIGHT") != "1" {
		t.Skip("explicit shell recovery preflight")
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
	if err := c.UseTranslationPackage("site-v2-shell"); err != nil {
		t.Fatal(err)
	}
	ref := func(path string) i18n.GlossaryArchiveReference {
		t.Helper()
		r, err := ReferenceFile(root, path)
		if err != nil {
			t.Fatal(err)
		}
		return i18n.GlossaryArchiveReference{Path: r.Path, SHA256: r.SHA256}
	}
	snapshot := ref("data/quality-check-snapshots/zh-CN/site-v2-shell-qc-001/manifest.json")
	o := i18n.RepresentationReconciliationOptions{Locale: "zh-CN", Package: "site-v2-shell", ReconciliationID: "site-shell-qc001-machine-successors", PreviousSnapshotID: "site-v2-shell-qc-001", SnapshotSHA256: snapshot.SHA256, Impact: ref("docs/PROTECTED_REPRESENTATION_FORMAL_IMPACT.json"), HistoricalDocuments: ref("data/site-shell/successor/sources/30d7c2ef242aa2aa13e64aa5a8917baca377710cb0ef77a68dba3cfb27694272.json")}
	plan, err := i18n.ReconcileTranslationUnitRepresentations(root, c, o)
	data, _ := json.MarshalIndent(map[string]any{"options": o, "plan": plan, "error": errString(err)}, "", "  ")
	path := "/tmp/zh-CN-site-shell-representation-recovery-plan-20261018.json"
	if writeErr := os.WriteFile(path, append(data, '\n'), 0600); writeErr != nil {
		t.Fatal(writeErr)
	}
	if err != nil {
		t.Fatal(err)
	}
	if plan.Blocked != 0 || plan.Ready != 1 || plan.Applied {
		t.Fatalf("preflight not ready; evidence=%s", path)
	}
	if _, err := os.Stat(filepath.Join(root, "data/translation-unit-reconciliations/zh-CN", o.ReconciliationID)); !os.IsNotExist(err) {
		t.Fatal("unexpected formal receipt directory", err)
	}
	t.Logf("READY=%d BLOCKED=%d APPLIED=%t EVIDENCE=%s", plan.Ready, plan.Blocked, plan.Applied, path)
}

func errString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
