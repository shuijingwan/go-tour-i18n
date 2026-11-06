package sitecontent

import (
	"bytes"
	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"testing"
)

func TestStructuredExportReadonlyOwner(t *testing.T) {
	finish := contentidentity.BeginMemoization()
	defer finish()
	root, ref := structuredTransportReplacementFixture(t)
	makeValidation := func() *closureValidation {
		v := newClosureValidation(root)
		v.profile = func(name string) func() {
			if name == "structured-export" {
				v.readonlySuccessors.historicalMetrics = map[string]int{}
			}
			return func() {}
		}
		if err := v.currentInputs("zh-CN"); err != nil {
			t.Fatal(err)
		}
		return v
	}
	first := makeValidation()
	got, m, err := first.exportStructuredAssetsReadonly(root, "zh-CN", nil)
	if err != nil {
		t.Fatal(err)
	}
	owner := first.readonlySuccessors
	if owner == nil || owner.readonlyOwner || owner.historicalMetrics["aggregate-verifies"] != 1 || owner.historicalMetrics["BuildGlobal"] != 1 || owner.historicalMetrics["snapshotFS"] != 1 {
		t.Fatal("owner not bounded/same frozen authority rebuilt", owner.historicalMetrics)
	}
	eager := newClosureValidation(root)
	eager.disabled = true
	if err = eager.currentInputs("zh-CN"); err != nil {
		t.Fatal(err)
	}
	want, wantManifest, err := eager.exportStructuredAssets(root, "zh-CN", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = eager.verify(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) || m.Identity != wantManifest.Identity {
		t.Fatal("owner reuse changed canonical transport/semantic identity")
	}
	second := makeValidation()
	again, _, err := second.exportStructuredAssetsReadonly(root, "zh-CN", nil)
	if err != nil {
		t.Fatal(err)
	}
	if second.readonlySuccessors == owner || second.readonlySuccessors.historical == owner.historical || !bytes.Equal(got, again) || second.readonlySuccessors.historicalMetrics["BuildGlobal"] != 1 {
		t.Fatal("independent invocation reused state")
	}
	// A context ceases being deferred at return; ordinary callers remain eager.
	if err = owner.verifyObserved(); err != nil || owner.historicalMetrics["aggregate-verifies"] != 2 {
		t.Fatal("default eager behavior changed", err)
	}
	var a StructuredSourceSuccessor
	if err = readReference(root, ref, &a); err != nil {
		t.Fatal(err)
	}
	original, err := readRegular(root, a.Body.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer writeFixture(t, root, a.Body.Path, original)
	mutated := makeValidation()
	mutated.profile = func(name string) func() {
		if name != "structured-export" {
			return func() {}
		}
		return func() {
			changed := bytes.Clone(original)
			changed[len(changed)/2] ^= 1
			writeFixture(t, root, a.Body.Path, changed)
		}
	}
	if _, _, err = mutated.exportStructuredAssetsReadonly(root, "zh-CN", nil); err == nil {
		t.Fatal("mutation before final aggregate guard accepted")
	}
}
