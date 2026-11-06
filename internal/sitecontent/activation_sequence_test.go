package sitecontent

import (
	"bytes"
	"testing"
)

// Reuse immutable reviewed closures and their baseline in an isolated copy.
// No repository completion, receipt or language evidence is written here.
func TestIntegratedSurfaceSequentialActivation(t *testing.T) {
	root, _ := crossPackageReplacementFixture(t)
	var reviewed IntegratedSurfaceReceipt
	ref, err := ReferenceFile(root, "data/locale-language/zh-CN/surface/20261011-zh-CN-integrated-surface-002.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := readReference(root, ref, &reviewed); err != nil {
		t.Fatal(err)
	}
	baseline, err := Encode(reviewed.Scope.ExistingCoverage)
	if err != nil {
		t.Fatal(err)
	}
	if digest(baseline) != "5c65e39616d77c0020de9ed7f9bdffba845c5c860595b780a47f70b45dac4fb6" {
		t.Fatal("fixture does not reproduce the reviewed baseline")
	}
	writeFixture(t, root, LocalePath(reviewed.Scope.Locale), baseline)
	if len(reviewed.Scope.Closures) != 2 {
		t.Fatal("expected two reviewed prospective closures")
	}
	// Apply runs ActivationPreflight itself; avoid repeating the expensive
	// full replacement proof before the same fixture mutation.
	plan, err := ApplyActivation(root, reviewed.Scope.Closures[0], digest(baseline))
	if err != nil {
		t.Fatalf("first preflight/apply: %v", err)
	}
	t.Logf("%s preflight/apply PASS", plan.Package)
	if _, err := ActivationPreflight(root, reviewed.Scope.Closures[1]); err != nil {
		t.Fatalf("second preflight: %v", err)
	}
}

// The maintainer has already applied A. Reuse that byte-exact state to test B
// without repeating the same expensive first activation during diagnosis.
func TestIntegratedSurfaceActivatedClosurePreflight(t *testing.T) {
	root, _ := crossPackageReplacementFixture(t)
	gate, err := ReferenceFile(root, "data/locale-language/zh-CN/surface/20261011-zh-CN-integrated-surface-002.json")
	if err != nil {
		t.Fatal(err)
	}
	var r IntegratedSurfaceReceipt
	if err := readReference(root, gate, &r); err != nil {
		t.Fatal(err)
	}
	before, err := readRegular(root, LocalePath(r.Scope.Locale))
	if err != nil || digest(before) != "709893036b1619eb80e4b477360e29d548ceb3e1ebffc0ddaa8ea4cd6fe7df20" {
		t.Fatal("fixture must reuse the accepted first activation", err)
	}
	plan, err := ActivationPreflight(root, r.Scope.Closures[1])
	if err != nil {
		t.Fatal("second closure preflight after accepted first activation", err)
	}
	if plan.Package != "site-v2-shell" || plan.AlreadyActive || plan.PriorSHA != digest(before) || plan.Completion.Evidence[1] != gate {
		t.Fatal("remaining closure does not bind the same PASS and current CAS state")
	}
	after, _ := readRegular(root, LocalePath(r.Scope.Locale))
	if !bytes.Equal(before, after) {
		t.Fatal("preflight mutated coverage")
	}
	t.Log("site-v2-shell preflight PASS after byte-exact accepted Learn/Docs activation")
}
