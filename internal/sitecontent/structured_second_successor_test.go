package sitecontent

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// All evidence and language-like values here are synthetic t.TempDir fixtures.
func structuredSecondSuccessor(t *testing.T, root string, first, ancestor, firstDelta Reference, page *Reference, pkg, term, changed string) {
	t.Helper()
	scope, err := BuildIntegratedSurfaceScope(root, "zh-CN", []Reference{first})
	if err != nil {
		t.Fatal(err)
	}
	md := []byte("# Synthetic successor Surface\n\nPASS\n")
	writeFixture(t, root, "data/second-successor-surface.md", md)
	receipt := IntegratedSurfaceReceipt{Schema: IntegratedSurfaceSchema, Scope: *scope, ReviewID: "first-successor", Reviewer: "synthetic-independent-reviewer", Decision: "passed", Findings: []string{}, Evidence: Reference{"data/second-successor-surface.md", digest(md)}}
	raw, _ := Encode(receipt)
	_, surfacePath, err := RecordIntegratedSurface(root, raw)
	if err != nil {
		t.Fatal(err)
	}
	surface, err := ReferenceFile(root, surfacePath)
	if err != nil {
		t.Fatal(err)
	}
	before, err := historicalStructuredClosure(root, first, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	state, err := readLocaleAuthority(root, mustGlobal(t, root), "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	for i := range state.Packages {
		c := &state.Packages[i]
		if c.Package == pkg {
			c.Evidence = []Reference{first, surface}
			c.EvidenceIdentity, c.ContextIdentity = identity(c.Evidence), before.Closure.Identity
		}
	}
	raw, _ = Encode(state)
	writeFixture(t, root, LocalePath("zh-CN"), raw)
	structuredGlossarySuccessorWithLabel(t, root, term, "Synthetic successor label", "second-successor-review")
	bundle, manifest, err := ExportStructuredAssets(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	wantTarget := "locales/zh-CN/structured/" + changed
	if !reflect.DeepEqual(manifest.Plan.Expected, []string{wantTarget}) {
		t.Fatal("second successor must replace exactly one asset", manifest.Plan.Expected)
	}
	var prior StructuredGeneration
	if err := readReference(root, firstDelta, &prior); err != nil {
		t.Fatal(err)
	}
	output := prior.Output
	output.BundleSHA = manifest.Identity
	for i := range output.Outputs {
		for j := range output.Outputs[i].Slots {
			s := &output.Outputs[i].Slots[j]
			s.Text = strings.ReplaceAll(s.Text, "Synthetic replacement label", "Synthetic successor label")
		}
	}
	raw, _ = Encode(output)
	provenance := Provenance{Provider: "codex", Model: "gpt-5.6-sol-high", Session: "synthetic-generation", GeneratedAt: "2026-10-07T01:00:00Z"}
	_, deltaPath, err := ImportStructuredAssets(root, bundle, raw, provenance)
	if err != nil {
		t.Fatal(err)
	}
	delta, _ := ReferenceFile(root, deltaPath)
	for name, refs := range map[string][]Reference{
		"missing-affected-delta":           nil,
		"duplicate-current-delta":          {delta, delta},
		"wrong-baseline-delta":             {firstDelta},
		"conflicting-baseline-deltas":      {delta, firstDelta},
		"compatible-arbitrary-replacement": {ancestor},
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, _, err := newClosureValidation(root).mergeStructuredBaselineDelta(root, "zh-CN", pkg, refs); err == nil {
				t.Fatal("invalid successor accepted")
			}
		})
	}
	provenance.GeneratedAt = "2026-10-07T02:00:00Z"
	_, conflictPath, err := ImportStructuredAssets(root, bundle, raw, provenance)
	if err != nil {
		t.Fatal(err)
	}
	conflict, _ := ReferenceFile(root, conflictPath)
	if _, _, _, err := newClosureValidation(root).mergeStructuredBaselineDelta(root, "zh-CN", pkg, []Reference{delta, conflict}); err == nil {
		t.Fatal("conflicting independent current deltas accepted")
	}
	closure, path, err := FinalizePackageClosure(root, "zh-CN", pkg, page, []Reference{delta})
	if err != nil {
		t.Fatal("second successor with current delta only", err)
	}
	if closure.StructuredBaseline == nil || *closure.StructuredBaseline != first || !reflect.DeepEqual(closure.Structured, []Reference{delta}) {
		t.Fatal("immediate predecessor/current-only delta not preserved", closure)
	}
	ref, _ := ReferenceFile(root, path)
	if _, _, err := CheckPackageClosure(root, ref); err != nil {
		t.Fatal("second successor current-check", err)
	}
	after, err := historicalStructuredClosure(root, ref, map[string]bool{})
	if err != nil {
		t.Fatal("recursive second successor historical integrity", err)
	}
	carried := 0
	for source, owner := range before.Owners {
		if source == changed {
			if after.Owners[source] != delta || bytes.Equal(before.Files[source], after.Files[source]) {
				t.Fatal("replacement owner/output not current")
			}
			continue
		}
		if after.Owners[source] != owner || !bytes.Equal(before.Files[source], after.Files[source]) {
			t.Fatal("carried bytes/owner lineage changed", source)
		}
		carried++
	}
	if carried == 0 {
		t.Fatal("fixture must exercise inherited compatible owners")
	}
	var original StructuredGeneration
	if err := readReference(root, ancestor, &original); err != nil {
		t.Fatal(err)
	}
	for label, name := range map[string]string{
		"ancestor-generation":           ancestor.Path,
		"ancestor-bundle":               original.Bundle.Path,
		"immediate-predecessor-closure": first.Path,
		"predecessor-surface":           surface.Path,
	} {
		t.Run(label+"-tamper", func(t *testing.T) {
			old, err := readRegular(root, name)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, name, old)
			writeFixture(t, root, name, append(bytes.Clone(old), ' '))
			if _, err := historicalStructuredClosure(root, ref, map[string]bool{}); err == nil {
				t.Fatal("recursive historical tamper accepted")
			}
		})
	}
	bad := before.Closure
	bad.Files = append([]Reference(nil), bad.Files...)
	bad.Files[0].SHA256 = strings.Repeat("0", 64)
	bad.Identity = ""
	bad.Identity = identity(bad)
	badPath := workflowPath(bad.Locale, bad.Package, "closure", bad.Identity)
	if err := saveImmutable(root, badPath, bad); err != nil {
		t.Fatal(err)
	}
	badRef, _ := ReferenceFile(root, badPath)
	if _, err := historicalStructuredClosure(root, badRef, map[string]bool{}); err == nil {
		t.Fatal("self-consistent closure with wrong historical target accepted")
	}
	if page != nil {
		t.Run("page-finalization-tamper", func(t *testing.T) {
			old, err := readRegular(root, page.Path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, page.Path, old)
			writeFixture(t, root, page.Path, append(bytes.Clone(old), ' '))
			if _, _, err := BuildPackageClosure(root, "zh-CN", pkg, page, []Reference{delta}); err == nil {
				t.Fatal("Page authority bypassed by structured successor")
			}
		})
	}
}
