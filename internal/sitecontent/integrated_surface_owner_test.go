package sitecontent

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestIntegratedSurfaceReadonlyOwner(t *testing.T) {
	finish := contentidentity.BeginMemoization()
	defer finish()
	root, closure, _, _ := structuredLifecycleFixture(t, "site-v2-shell")
	refs := []Reference{closure}
	eager := newClosureValidation(root)
	if err := eager.currentInputs("zh-CN"); err != nil {
		t.Fatal(err)
	}
	want, err := eager.buildIntegratedSurfaceScope(root, "zh-CN", refs)
	if err != nil {
		t.Fatal(err)
	}
	if err := eager.verify(); err != nil {
		t.Fatal(err)
	}
	wantBytes, _ := Encode(want)
	makeValidation := func() *closureValidation {
		v := newClosureValidation(root)
		v.profile = func(name string) func() {
			if name == "integrated-surface-scope" {
				if v.readonlySuccessors == nil || !v.readonlySuccessors.readonlyOwner {
					t.Fatal("missing readonly invocation owner")
				}
				v.readonlySuccessors.historicalMetrics = map[string]int{}
			}
			return func() {}
		}
		if err := v.currentInputs("zh-CN"); err != nil {
			t.Fatal(err)
		}
		return v
	}
	var previous *structuredSuccessorValidation
	for invocation := 0; invocation < 2; invocation++ {
		v := makeValidation()
		got, err := v.buildIntegratedSurfaceScopeReadonly(root, "zh-CN", refs)
		if err != nil {
			t.Fatal(err)
		}
		gotBytes, _ := Encode(got)
		owner := v.readonlySuccessors
		if !bytes.Equal(wantBytes, gotBytes) || got.Identity != want.Identity || owner == previous || owner.readonlyOwner || owner.historicalMetrics["aggregate-verifies"] != 1 || owner.historicalMetrics["BuildGlobal"] != 1 {
			t.Fatal("scope bytes/identity, lifetime or final guard changed", owner.historicalMetrics)
		}
		previous = owner
	}
	public, err := BuildIntegratedSurfaceScope(root, "zh-CN", refs)
	publicBytes, _ := Encode(public)
	if err != nil || !bytes.Equal(publicBytes, wantBytes) {
		t.Fatal("public boundary differs", err)
	}
	// Mutate an actual source-lineage dependency observed only by the nested
	// owner, after all scope work. The owner's final guard must catch it even
	// when the outer closure graph has no observation of that path.
	v := makeValidation()
	var changed string
	var original []byte
	v.profile = func(name string) func() {
		if name != "integrated-surface-scope" {
			return func() {}
		}
		return func() {
			var paths []string
			for path := range v.readonlySuccessors.guard.deps {
				if _, outer := v.deps[path]; !outer {
					paths = append(paths, path)
				}
			}
			sort.Strings(paths)
			for _, path := range paths {
				b, err := readRegular(root, path)
				if err != nil || len(b) == 0 {
					continue
				}
				changed, original = path, b
				info, err := os.Stat(filepath.Join(root, path))
				if err != nil {
					t.Fatal(err)
				}
				bad := bytes.Clone(b)
				bad[len(bad)/2] ^= 1
				writeFixture(t, root, path, bad)
				if err := os.Chtimes(filepath.Join(root, path), info.ModTime(), info.ModTime()); err != nil {
					t.Fatal(err)
				}
				return
			}
			t.Fatal("fixture has no exclusively nested dependency")
		}
	}
	if _, err := v.buildIntegratedSurfaceScopeReadonly(root, "zh-CN", refs); err == nil || changed == "" || v.readonlySuccessors.readonlyOwner {
		t.Fatal("nested mutation escaped fresh final guard", changed, err)
	}
	writeFixture(t, root, changed, original)
	t.Log("scope bytes/identity unchanged; fresh owner each invocation; final nested same-size/mtime drift rejected")
}
