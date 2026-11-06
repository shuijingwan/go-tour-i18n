package sitecontent

import (
	"bytes"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"reflect"
	"testing"
)

func TestPostActivationHistoricalInventory(t *testing.T) {
	root := activationAuthorizationFixture(t)
	gate, err := ReferenceFile(root, "data/locale-language/zh-CN/surface/20261014-zh-CN-integrated-surface-faa455c8-002.json")
	if err != nil {
		t.Fatal(err)
	}
	var receipt IntegratedSurfaceReceipt
	if err := readReference(root, gate, &receipt); err != nil {
		t.Fatal(err)
	}
	g, err := reviewedActivationGlobal(root)
	if err != nil {
		t.Fatal(err)
	}
	state, err := readLocaleAuthority(root, g, receipt.Scope.Locale)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := reviewedActivationBaseline(root, g, receipt, *state)
	if err != nil {
		t.Fatal(err)
	}
	if digest(baseline) != "9bcdf339dfe64d182003bb77e49befb3ac4ca72e38a1210ffd7f3adb0de77879" {
		t.Fatal("original activation baseline changed")
	}
	original := map[string][]byte{}
	for _, ref := range append(append([]Reference{}, receipt.Scope.Closures...), gate, receipt.Evidence) {
		b, err := readRegular(root, ref.Path)
		if err != nil {
			t.Fatal(err)
		}
		original[ref.Path] = b
	}
	var atStep Locale
	if err := StrictJSON(baseline, &atStep); err != nil {
		t.Fatal(err)
	}
	hashes := []string{"ecaefa36027627c4458f112cf458d59019abe6324be45a0aa70135f336c97082", "f6ed27536bcf171fea171cccf29810e3846e949dfccd5f1c76e2e19c6102d290"}
	v := newClosureValidation(root)
	for i, ref := range receipt.Scope.Closures {
		var closure PackageClosure
		if err := readReference(root, ref, &closure); err != nil {
			t.Fatal(err)
		}
		var installed Completion
		for _, c := range state.Packages {
			if c.Package == closure.Package {
				installed = c
			}
		}
		for j, c := range atStep.Packages {
			if c.Package == installed.Package {
				atStep.Packages[j] = installed
			}
		}
		current, _ := Encode(atStep)
		if digest(current) != hashes[i] {
			t.Fatal("wrong intermediate/final inventory", i, digest(current))
		}
		writeFixture(t, root, LocalePath(receipt.Scope.Locale), current)
		if err := proveReviewedActivationTransition(root, gate, receipt); err != nil {
			t.Fatal(err)
		}
		history, err := v.historicalStructuredCompletion(root, receipt.Scope.Locale, installed)
		if err != nil {
			t.Fatal(err)
		}
		if len(history.Files) != len(closure.Files) {
			t.Fatal("historical inventory count")
		}
		for _, f := range closure.Files {
			if digest(history.Files[f.Path]) != f.SHA256 {
				t.Fatal("historical path/hash", f.Path)
			}
		}
		if closure.StructuredBaseline == nil {
			t.Fatal("fixture needs explicit frozen predecessor")
		}
		graph := newClosureValidation(root)
		checked, files, err := graph.checkPackageClosure(root, ref)
		if err != nil {
			t.Fatal("saved closure used final coverage as its own baseline", err)
		}
		if !reflect.DeepEqual(checked, &closure) || len(files) != len(closure.Files) {
			t.Fatal("saved closure changed")
		}
		if _, ok := graph.closures[*closure.StructuredBaseline]; !ok {
			t.Fatal("did not validate exact predecessor")
		}
		t.Logf("STEP=%d SHA=%s PACKAGE=%s HISTORICAL_FILES=%d", i+1, digest(current), installed.Package, len(history.Files))
	}
	// The original baseline remains available after both completions, exactly.
	old, err := readHistoricalActivationContext(root, receipt.Scope.Locale, i18n.GlossaryArchiveReference{Path: LocalePath(receipt.Scope.Locale), SHA256: digest(baseline)})
	if err != nil || !bytes.Equal(old, baseline) {
		t.Fatal("predecessor context bytes", err)
	}
	// Fully current coverage is the same boundary used by Preview's resolver.
	if _, _, err := LocalCoverage(root, receipt.Scope.Locale); err != nil {
		t.Fatal("preview/bootstrap coverage", err)
	}
	// Re-seal only synthetic test closures so missing/extra inventories reach the
	// actual historical exact-set gate rather than merely a changed outer SHA.
	var shell PackageClosure
	if err := readReference(root, receipt.Scope.Closures[1], &shell); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"missing", "extra", "target-hash", "outer-sha", "Surface-sha"} {
		t.Run(kind, func(t *testing.T) {
			bad := shell
			bad.Files = append([]Reference{}, shell.Files...)
			switch kind {
			case "missing":
				bad.Files = bad.Files[1:]
			case "extra":
				bad.Files = append(bad.Files, Reference{Path: "unexpected.md", SHA256: digest([]byte("extra"))})
			case "target-hash":
				bad.Files[0].SHA256 = digest([]byte("changed"))
			}
			if kind == "Surface-sha" {
				c := state.Packages[2]
				c.Evidence = append([]Reference{}, c.Evidence...)
				c.Evidence[1].SHA256 = digest([]byte("fake"))
				c.EvidenceIdentity = identity(c.Evidence)
				if _, err := historicalStructuredCompletion(root, receipt.Scope.Locale, c); err == nil {
					t.Fatal("Surface drift accepted")
				}
				return
			}
			bad.Identity = ""
			bad.Identity = identity(bad)
			path := workflowPath(bad.Locale, bad.Package, "closure", bad.Identity)
			if err := saveImmutable(root, path, bad); err != nil {
				t.Fatal(err)
			}
			ref, _ := ReferenceFile(root, path)
			if kind == "outer-sha" {
				ref.SHA256 = digest([]byte("wrong outer hash"))
			}
			if _, err := historicalStructuredClosure(root, ref, map[string]bool{}); err == nil {
				t.Fatal("historical inventory tamper accepted", kind)
			}
		})
	}
	for p, b := range original {
		after, err := readRegular(root, p)
		if err != nil || !bytes.Equal(b, after) {
			t.Fatal("immutable artifact mutation", p, err)
		}
	}
}
