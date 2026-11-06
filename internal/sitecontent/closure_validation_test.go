package sitecontent

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func closureDeltaPerformanceFixture(t *testing.T) (string, Reference, []Reference, *Reference) {
	t.Helper()
	// Exercise the shared immutable graph without manufacturing zh-CN Page
	// candidates by copying real English teaching comments. The active comment
	// gate correctly rejects that legacy setup. Full Page/composition coverage
	// is independently provided by the opt-in real-authority readonly builder.
	root, baseline, initial, page := structuredLifecycleFixture(t, "site-v2-shell")
	const term = "Learning resources"
	structuredGlossarySuccessor(t, root, term)
	b, m, err := ExportStructuredAssets(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	var old StructuredGeneration
	if err := readReference(root, initial, &old); err != nil {
		t.Fatal(err)
	}
	o := StructuredOutput{Schema: StructuredSchema, BundleSHA: m.Identity, Outputs: []StructuredTarget{}}
	for _, wanted := range m.Plan.Expected {
		for _, target := range old.Output.Outputs {
			if target.Path != wanted {
				continue
			}
			target.Slots = append([]Target(nil), target.Slots...)
			for i := range target.Slots {
				target.Slots[i].Text = strings.ReplaceAll(target.Slots[i].Text, term, "Synthetic replacement label")
			}
			o.Outputs = append(o.Outputs, target)
		}
	}
	encoded, _ := Encode(o)
	_, name, err := ImportStructuredAssets(root, b, encoded, old.Provenance)
	if err != nil {
		t.Fatal(err)
	}
	delta, err := ReferenceFile(root, name)
	if err != nil {
		t.Fatal(err)
	}
	return root, baseline, []Reference{initial, delta}, page
}

func measuredClosure(t *testing.T, root, pkg string, page *Reference, refs []Reference, disabled bool) (*PackageClosure, map[string][]byte, *closureValidation) {
	t.Helper()
	v := newClosureValidation(root)
	v.disabled = disabled
	if err := v.currentInputs("zh-CN"); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	c, files, err := v.buildPackageClosure(root, "zh-CN", pkg, page, refs)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.verify(); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s memo_disabled=%v wall=%s counts=%v", pkg, disabled, time.Since(start), v.counts)
	return c, files, v
}

func TestClosureValidationGraphBaselineDelta(t *testing.T) {
	root, baseline, refs, page := closureDeltaPerformanceFixture(t)
	want, wantFiles, uncached := measuredClosure(t, root, "site-v2-shell", page, refs, true)
	got, files, v := measuredClosure(t, root, "site-v2-shell", page, refs, false)
	wantBytes, _ := Encode(want)
	gotBytes, _ := Encode(got)
	if !bytes.Equal(wantBytes, gotBytes) || !reflect.DeepEqual(files, wantFiles) || len(files) != 2 || got.StructuredBaseline == nil || *got.StructuredBaseline != baseline {
		t.Fatal("memoization changed closure bytes/identity, baseline or structured coverage")
	}
	for key, count := range v.counts {
		if count != 1 {
			t.Fatal("dependency fully validated repeatedly", key, count)
		}
	}
	if uncached.counts["closure:"+baseline.Path] <= 1 {
		t.Fatal("fixture did not exercise repeated baseline graph")
	}
	// A second access in this invocation reuses the full frozen proof, but
	// rechecks dependency bytes instead of trusting an old PASS or path alone.
	if _, err := v.historicalStructuredClosure(root, baseline, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	if v.counts["closure:"+baseline.Path] != 1 {
		t.Fatal("same immutable closure replayed")
	}
	if _, err := v.historicalStructuredClosure(root, Reference{baseline.Path, strings.Repeat("0", 64)}, map[string]bool{}); err == nil {
		t.Fatal("SHA mismatch hit cached path")
	}
	for _, ref := range refs {
		if v.counts["generation:"+ref.Path] != 1 {
			t.Fatal("same immutable generation replayed", ref)
		}
	}
	for key, cached := range v.current {
		_, reused, err := v.checkStructuredGeneration(root, key.ref, key.pkg)
		if err != nil || !reflect.DeepEqual(reused, cached.files) || v.counts["current-generation:"+key.ref.Path+":"+key.pkg] != 1 {
			t.Fatal("same current generation/scope fully validated more than once", err)
		}
	}
	// Publish only in the isolated fixture; production evidence is never touched.
	name := workflowPath(got.Locale, got.Package, "closure", got.Identity)
	if err := saveImmutable(root, name, got); err != nil {
		t.Fatal(err)
	}
	closureRef, _ := ReferenceFile(root, name)
	if err := v.bind(closureRef); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CheckPackageClosure(root, closureRef); err != nil {
		t.Fatal(err)
	}
	var generation StructuredGeneration
	if err := readReference(root, refs[1], &generation); err != nil {
		t.Fatal(err)
	}
	for label, name := range map[string]string{
		"glossary-drift":       "locales/zh-CN/glossary.yaml",
		"review-drift":         generation.Plan.Review.Path,
		"source-drift":         "_content/tour/welcome.article",
		"generation-tampering": refs[1].Path,
		"bundle-tampering":     generation.Bundle.Path,
		"closure-tampering":    closureRef.Path,
		"baseline-tampering":   baseline.Path,
	} {
		t.Run(label, func(t *testing.T) {
			before, err := readRegular(root, name)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, name, before)
			writeFixture(t, root, name, append(bytes.Clone(before), []byte("tamper\n")...))
			if _, _, err := CheckPackageClosure(root, closureRef); err == nil {
				t.Fatal("current checker accepted drift")
			}
			_, err = v.historicalStructuredClosure(root, baseline, map[string]bool{})
			if err == nil {
				err = v.verify()
			}
			if err == nil {
				t.Fatal("same invocation cache accepted changed dependency at same path")
			}
		})
	}
	// A delta may inherit unaffected assets from its exact accepted baseline;
	// it need not redundantly name the baseline's original Generation again.
	deltaOnly, deltaFiles, err := BuildPackageClosure(root, "zh-CN", "site-v2-shell", page, refs[1:])
	if err != nil || !reflect.DeepEqual(deltaFiles, files) || deltaOnly.StructuredBaseline == nil || *deltaOnly.StructuredBaseline != baseline {
		t.Fatal("exact baseline + delta carry changed", err)
	}
	for _, set := range [][]Reference{refs[:1], {refs[0], refs[1], refs[1]}} {
		if _, _, err := BuildPackageClosure(root, "zh-CN", "site-v2-shell", page, set); err == nil {
			t.Fatal("missing/extra/conflicting generation accepted", set)
		}
	}
	// A new generation with distinct provenance is still a conflicting delta.
	other := generation
	other.Provenance.GeneratedAt = "2026-10-06T03:00:00Z"
	other.Identity = ""
	other.Identity = identity(other)
	otherPath := "data/locale-language/zh-CN/structured-generation/" + other.Identity + ".json"
	if err := saveImmutable(root, otherPath, other); err != nil {
		t.Fatal(err)
	}
	otherRef, _ := ReferenceFile(root, otherPath)
	if _, _, err := BuildPackageClosure(root, "zh-CN", "site-v2-shell", page, append(append([]Reference(nil), refs...), otherRef)); err == nil {
		t.Fatal("conflicting distinct generation accepted")
	}
}

func TestClosureValidationGraphInitialCarry(t *testing.T) {
	root, b, o, options, initial := structuredRecoveryFixture(t)
	writeFixture(t, root, "AGENTS.md", []byte("Synthetic execution-only mutation\n"))
	options.Apply = true
	recovered, err := RecoverStructuredAssets(root, b, o, options)
	if err != nil {
		t.Fatal(err)
	}
	refs := []Reference{initial, recovered.Generation}
	want, wantFiles, _ := measuredClosure(t, root, "site-v2-shell", nil, refs, true)
	got, gotFiles, v := measuredClosure(t, root, "site-v2-shell", nil, refs, false)
	a, _ := Encode(want)
	z, _ := Encode(got)
	if !bytes.Equal(a, z) || !reflect.DeepEqual(wantFiles, gotFiles) {
		t.Fatal("initial + carry bytes/identity changed")
	}
	for key, count := range v.counts {
		if count != 1 {
			t.Fatal("repeated validation", key, count)
		}
	}
	before, err := readRegular(root, recovered.OriginalOutput.Path)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, recovered.OriginalOutput.Path, append(bytes.Clone(before), '\n'))
	if _, _, _, err := v.historicalStructuredGeneration(root, recovered.Generation); err == nil {
		t.Fatal("carry cache accepted changed original output")
	}
	writeFixture(t, root, recovered.OriginalOutput.Path, before)
	if err := os.Remove(filepath.Join(root, initial.Path)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := BuildPackageClosure(root, "zh-CN", "site-v2-shell", nil, refs); err == nil {
		t.Fatal("new invocation trusted old cache")
	}
}

func TestClosureValidationInputGuard(t *testing.T) {
	for _, mutation := range []string{"same-path-same-length", "mode", "symlink", "directory-addition", "absent-appeared"} {
		t.Run(mutation, func(t *testing.T) {
			root := t.TempDir()
			name := "inputs/value.json"
			original := []byte("{\"value\":1}\n")
			writeFixture(t, root, name, original)
			v := newClosureValidation(root)
			proof := v.beginProof()
			ref := Reference{name, digest(original)}
			if err := v.bind(ref); err != nil {
				t.Fatal(err)
			}
			if err := v.directory("inputs"); err != nil {
				t.Fatal(err)
			}
			if err := v.optional("not-present.json"); err != nil {
				t.Fatal(err)
			}
			v.endProof()
			v.proofs["fixture"] = proof
			if err := v.reuse("fixture"); err != nil {
				t.Fatal(err)
			}
			if err := v.bind(Reference{name, strings.Repeat("0", 64)}); err == nil {
				t.Fatal("path alone accepted wrong SHA")
			}
			switch mutation {
			case "same-path-same-length":
				st, _ := os.Stat(filepath.Join(root, name))
				writeFixture(t, root, name, []byte("{\"value\":2}\n"))
				if err := os.Chtimes(filepath.Join(root, name), st.ModTime(), st.ModTime()); err != nil {
					t.Fatal(err)
				}
			case "mode":
				if err := os.Chmod(filepath.Join(root, name), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Remove(filepath.Join(root, name)); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("elsewhere", filepath.Join(root, name)); err != nil {
					t.Fatal(err)
				}
			case "directory-addition":
				writeFixture(t, root, "inputs/extra.json", original)
			case "absent-appeared":
				writeFixture(t, root, "not-present.json", original)
			}
			err := v.reuse("fixture")
			if err == nil {
				err = v.verify()
			}
			if err == nil {
				t.Fatal("changed identity hit stale invocation cache")
			}
		})
	}
}
