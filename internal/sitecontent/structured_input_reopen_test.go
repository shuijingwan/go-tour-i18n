package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// A frozen event may legitimately have fewer semantic slots in identical raw
// YAML. Recompute its persisted masked structure, never change source bytes.
func inputReopenOldDocument(d Document, raw []byte) Document {
	d.Units = append([]Unit{}, d.Units...)
	kept := d.Units[:0]
	for _, u := range d.Units {
		if u.Source != "PDF" && u.Source != "GitHub" {
			kept = append(kept, u)
		}
	}
	d.Units = kept
	var masked bytes.Buffer
	pos := 0
	for _, u := range d.Units {
		end := strings.LastIndexByte(u.Locator, ':')
		context := strings.TrimSuffix(strings.TrimPrefix(u.Locator[:end], d.Path+"#"), ":"+u.SourceSHA)
		masked.Write(raw[pos:u.Start])
		masked.WriteString("\x00" + context + "\x00" + protectionIdentity(u.Protected))
		pos = u.End
	}
	masked.Write(raw[pos:])
	d.StructureSHA = identity(struct {
		Structure    string
		Dependencies []Dependency
	}{digest(masked.Bytes()), d.Dependencies})
	return d
}

func TestStructuredInputReopenSemanticBoundary(t *testing.T) {
	raw := []byte("- title: Example guide\n  eyebrow: PDF\n  url: https://example.com/a\n- title: Another guide\n  eyebrow: GitHub\n  url: https://example.com/b\n")
	s := Source{Path: "_content/learn/example.yaml", Kind: "data", SHA256: digest(raw), Dependencies: []Dependency{}}
	current, err := ParseDocument(s, raw)
	if err != nil {
		t.Fatal(err)
	}
	old := inputReopenOldDocument(*current, raw)
	added, err := semanticExpansion(old, *current, raw, raw)
	if err != nil || len(added) != 2 || added[0].Source != "PDF" || added[1].Source != "GitHub" {
		t.Fatal(added, err)
	}
	for _, tc := range []struct {
		name   string
		change func(*Document, *Document, *[]byte)
	}{
		{"raw-source-changed", func(_, _ *Document, b *[]byte) { *b = append(bytes.Clone(*b), ' ') }},
		{"removed-existing-atom", func(_, d *Document, _ *[]byte) { d.Units = d.Units[1:] }},
		{"changed-existing-atom", func(_, d *Document, _ *[]byte) { d.Units[0].Source = "Changed" }},
		{"unit-ID-churn", func(_, d *Document, _ *[]byte) { d.Units[0].ID = strings.Repeat("0", 64) }},
		{"zero-added", func(o, d *Document, _ *[]byte) { *d = *o }},
		{"non-localizable-added", func(_, d *Document, _ *[]byte) { d.Units[1].Source = "1234" }},
		{"dependency-drift", func(_, d *Document, _ *[]byte) { d.Dependencies = append(d.Dependencies, Dependency{Path: "unknown"}) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, _ := json.Marshal([]Document{old, *current})
			var docs []Document
			_ = json.Unmarshal(data, &docs)
			b := bytes.Clone(raw)
			tc.change(&docs[0], &docs[1], &b)
			if _, err := semanticExpansion(docs[0], docs[1], raw, b); err == nil {
				t.Fatal("unsafe expansion accepted")
			}
		})
	}
}

// Rebind a synthetic, already verified completion graph to a legitimate frozen
// 10-slot event. All source and accepted target bytes stay exact. This fixture
// writes only t.TempDir; production historical artifacts are never rewritten.
func inputReopenLifecycleFixture(t *testing.T) (string, Reference, Reference, *Reference) {
	t.Helper()
	root, _, gen, page := structuredLifecycleFixture(t, "learn-docs-v1")
	r, _, inputs, err := readStructuredGeneration(root, gen)
	if err != nil {
		t.Fatal(err)
	}
	const asset = "source:_content/learn/books.yaml"
	original, err := readRegularReference(root, r.Bundle)
	if err != nil {
		t.Fatal(err)
	}
	bundle := rewriteRecoveryBundle(t, original, func(m *StructuredBundle, fs map[string][]byte) {
		for i := range m.Plan.Assets {
			a := &m.Plan.Assets[i]
			if a.ID == asset {
				d := inputReopenOldDocument(*a.Document, fs["source/"+a.Document.Path])
				a.Document = &d
			}
		}
	})
	m, _, err := readStructuredBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	r.Plan, r.Output.BundleSHA = m.Plan, m.Identity
	ids := map[string]bool{}
	for _, a := range r.Plan.Assets {
		if a.ID == asset {
			for _, u := range a.Document.Units {
				ids[u.ID] = true
			}
			if len(ids) != 10 {
				t.Fatal("fixture old slot count", len(ids))
			}
		}
	}
	for i := range r.Output.Outputs {
		a := &r.Output.Outputs[i]
		if a.Path == "locales/zh-CN/structured/_content/learn/books.yaml" {
			kept := []Target{}
			for _, u := range a.Slots {
				if ids[u.ID] {
					kept = append(kept, u)
				}
			}
			a.Slots = kept
		}
	}
	r.Bundle = Reference{"data/locale-language/zh-CN/structured-bundles/" + digest(bundle) + ".zip", digest(bundle)}
	writeFixture(t, root, r.Bundle.Path, bundle)
	r.Identity = ""
	r.Identity = identity(*r)
	newGen := Reference{"data/locale-language/zh-CN/structured-generation/" + r.Identity + ".json", ""}
	b, _ := Encode(r)
	newGen.SHA256 = digest(b)
	writeFixture(t, root, newGen.Path, b)
	if _, _, _, err := historicalStructuredGeneration(root, newGen); err != nil {
		t.Fatal("frozen baseline", err)
	}
	var state Locale
	b, _ = readRegular(root, LocalePath("zh-CN"))
	if err := StrictJSON(b, &state); err != nil {
		t.Fatal(err)
	}
	replaced := map[Reference]Reference{gen: newGen}
	var wanted Reference
	for i := range state.Packages {
		c := &state.Packages[i]
		var closure PackageClosure
		if err := readReference(root, c.Evidence[0], &closure); err != nil {
			t.Fatal(err)
		}
		oldRef := c.Evidence[0]
		for j := range closure.Structured {
			if closure.Structured[j] == gen {
				closure.Structured[j] = newGen
			}
		}
		closure.Identity = ""
		closure.Identity = identity(closure)
		b, _ := Encode(closure)
		ref := Reference{workflowPath("zh-CN", closure.Package, "closure", closure.Identity), digest(b)}
		writeFixture(t, root, ref.Path, b)
		replaced[oldRef] = ref
		c.Evidence[0], c.ContextIdentity = ref, closure.Identity
		if closure.Package == "learn-docs-v1" {
			wanted = ref
		}
	}
	var surface IntegratedSurfaceReceipt
	oldSurface := state.Packages[0].Evidence[1]
	if err := readReference(root, oldSurface, &surface); err != nil {
		t.Fatal(err)
	}
	for i := range surface.Scope.Closures {
		if x, ok := replaced[surface.Scope.Closures[i]]; ok {
			surface.Scope.Closures[i] = x
		}
	}
	for i := range surface.Scope.Reviewed {
		for j, ref := range surface.Scope.Reviewed[i].Evidence {
			if x, ok := replaced[ref]; ok {
				surface.Scope.Reviewed[i].Evidence[j] = x
			}
		}
	}
	surface.Scope.Identity = ""
	surface.Scope.Identity = identity(surface.Scope)
	surface.Identity = ""
	surface.Identity = identity(surface)
	b, _ = Encode(surface)
	writeFixture(t, root, oldSurface.Path, b)
	newSurface := Reference{oldSurface.Path, digest(b)}
	for i := range state.Packages {
		c := &state.Packages[i]
		c.Evidence[1] = newSurface
		c.EvidenceIdentity = identity(c.Evidence)
	}
	b, _ = Encode(state)
	writeFixture(t, root, LocalePath("zh-CN"), b)
	_ = inputs
	return root, wanted, newGen, page
}

func TestStructuredInputReopenLifecycle(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, baseline, gen, page := inputReopenLifecycleFixture(t)
	const asset = "source:_content/learn/books.yaml"
	const target = "locales/zh-CN/structured/_content/learn/books.yaml"
	before, err := PlanStructuredAssets(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range before.Assets {
		if a.ID == asset && (a.State != "stale" || a.Assessment.Reason != "current exact source/input drift") {
			t.Fatal("no receipt must retain stale")
		}
	}
	for _, p := range before.Expected {
		if p == target {
			t.Fatal("unauthorized expansion Generation scope")
		}
	}
	r, err := ReopenStructuredInput(root, "zh-CN", asset, false)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Ready || r.Applied || r.Receipt.OldAtomCount != 10 || r.Receipt.CurrentAtomCount != 12 || !reflect.DeepEqual(r.ProjectedExpected, []string{target}) {
		t.Fatal("preflight", r)
	}
	if _, err := os.Stat(filepath.Join(root, r.Reference.Path)); !os.IsNotExist(err) {
		t.Fatal("preflight saved receipt")
	}
	valid := r.Receipt
	validation := newClosureValidation(root)
	var state Locale
	stateRaw, err := readRegular(root, LocalePath("zh-CN"))
	if err != nil {
		t.Fatal(err)
	}
	if err := StrictJSON(stateRaw, &state); err != nil {
		t.Fatal(err)
	}
	var frozen *structuredBaseline
	for _, c := range state.Packages {
		if c.Package == "learn-docs-v1" {
			frozen, err = validation.historicalStructuredCompletion(root, "zh-CN", c)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	var currentAsset StructuredAsset
	for _, a := range before.Assets {
		if a.ID == asset {
			currentAsset = a
		}
	}
	// Every self-bound field is rederived from the verified completion graph.
	for _, tc := range []struct {
		name   string
		change func(*StructuredInputReopen)
	}{
		{"closure-binding", func(x *StructuredInputReopen) { x.Closure.SHA256 = strings.Repeat("0", 64) }},
		{"generation-binding", func(x *StructuredInputReopen) { x.Generation.SHA256 = strings.Repeat("0", 64) }},
		{"accepted-target", func(x *StructuredInputReopen) { x.AcceptedTargetSHA = strings.Repeat("0", 64) }},
		{"glossary-binding", func(x *StructuredInputReopen) { x.Glossary.SHA256 = strings.Repeat("0", 64) }},
		{"review-binding", func(x *StructuredInputReopen) { x.Review.SHA256 = strings.Repeat("0", 64) }},
		{"old-structure", func(x *StructuredInputReopen) { x.OldStructureSHA = strings.Repeat("0", 64) }},
		{"current-structure", func(x *StructuredInputReopen) { x.StructureSHA = strings.Repeat("0", 64) }},
		{"added-inventory", func(x *StructuredInputReopen) { x.AddedUnitIDs = nil }},
		{"source-contributor", func(x *StructuredInputReopen) { x.CurrentSourceSHA = strings.Repeat("0", 64) }},
		{"asset-scope", func(x *StructuredInputReopen) { x.AssetID = "source:other" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := valid
			tc.change(&x)
			x.Identity = ""
			x.Identity = identity(x)
			// Scope inventory is fixed to the asset under test; misplaced claims
			// cannot be accepted even if the attacker recomputes every digest.
			p := inputReopenDirectory(valid.Locale, valid.Package, valid.AssetID) + "/" + x.Identity + ".json"
			b, _ := Encode(x)
			writeFixture(t, root, p, b)
			defer os.Remove(filepath.Join(root, p))
			if _, err := newClosureValidation(root).findInputReopen(root, *before, currentAsset, frozen); err == nil {
				t.Fatal("tampered receipt accepted")
			}
		})
	}
	applied, err := ReopenStructuredInput(root, "zh-CN", asset, true)
	if err != nil || !applied.Applied {
		t.Fatal("synthetic apply", err)
	}
	saved, _ := readRegular(root, applied.Reference.Path)
	// saveImmutable is idempotent for identical bytes and never overwrites.
	if err := saveImmutable(root, applied.Reference.Path, &valid); err != nil {
		t.Fatal(err)
	}
	bad := valid
	bad.Identity = "tampered"
	if err := saveImmutable(root, applied.Reference.Path, &bad); err == nil {
		t.Fatal("overwrote immutable receipt")
	}
	after, _ := readRegular(root, applied.Reference.Path)
	if !bytes.Equal(saved, after) {
		t.Fatal("receipt overwritten")
	}
	p, err := PlanStructuredAssets(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.Expected, []string{target}) {
		t.Fatal(p.Expected)
	}
	t.Run("stale-receipt-next-Document", func(t *testing.T) {
		next := currentAsset
		d := *next.Document
		d.StructureSHA = strings.Repeat("2", 64)
		next.Document = &d
		x, err := newClosureValidation(root).assessStructuredAsset(root, *before, next, frozen)
		if err != nil || x.Classification != "stale" || x.InputReopen != nil {
			t.Fatal("old receipt authorized a future Document", x, err)
		}
		if err := newClosureValidation(root).checkInputReopenCurrent(root, *before, next, frozen.Inputs[valid.Generation.Path]["source/"+valid.SourcePath]); err == nil {
			t.Fatal("noncanonical/non-localizable Document authorized")
		}
	})
	for _, a := range p.Assets {
		if a.ID == asset {
			if a.State != "generation-required" || a.Assessment.Classification != "affected" || a.Assessment.InputReopen == nil || len(a.Document.Units) != 12 {
				t.Fatal("full current asset required", a)
			}
		} else {
			for _, old := range before.Assets {
				if old.ID == a.ID && old.State != a.State && !(a.State == "context-only" && a.Assessment == nil && old.State == "generation-required") {
					t.Fatal("unrelated Tour/shell asset changed", a.ID)
				}
			}
		}
	}
	if _, _, err := BuildPackageClosure(root, "zh-CN", "learn-docs-v1", page, []Reference{gen}); err == nil {
		t.Fatal("receipt carried missing replacement Generation")
	}
	// Current machine authority is invalidated by actual input mutations.
	for _, name := range []string{valid.SourcePath, valid.Glossary.Path, valid.Review.Path, valid.Closure.Path, valid.Generation.Path} {
		t.Run("current-or-baseline-tamper/"+name, func(t *testing.T) {
			b, err := readRegular(root, name)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, name, b)
			writeFixture(t, root, name, append(bytes.Clone(b), ' '))
			if _, err := PlanStructuredAssets(root, "zh-CN"); err == nil {
				t.Fatal("input tamper accepted")
			}
		})
	}
	t.Run("identity-tamper", func(t *testing.T) {
		defer writeFixture(t, root, applied.Reference.Path, saved)
		x := valid
		x.Identity = "invalid"
		b, _ := Encode(x)
		writeFixture(t, root, applied.Reference.Path, b)
		if _, err := PlanStructuredAssets(root, "zh-CN"); err == nil {
			t.Fatal("identity tamper")
		}
	})
	t.Run("conflicting-receipts", func(t *testing.T) {
		x := valid
		x.AcceptedTargetSHA = strings.Repeat("1", 64)
		x.Identity = ""
		x.Identity = identity(x)
		p := inputReopenPath(x)
		b, _ := Encode(x)
		writeFixture(t, root, p, b)
		defer os.Remove(filepath.Join(root, p))
		if _, err := PlanStructuredAssets(root, "zh-CN"); err == nil {
			t.Fatal("conflict accepted")
		}
	})
	// Synthetic complete replacement uses the existing asset-level full output,
	// existing importer, current checker, closure and delta-aware Surface scope.
	bundle, m, err := ExportStructuredAssets(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	output := StructuredOutput{Schema: StructuredSchema, BundleSHA: m.Identity, Outputs: []StructuredTarget{}}
	for _, a := range m.Plan.Assets {
		if a.ID == asset {
			slots := []Target{}
			for _, u := range a.Document.Units {
				slots = append(slots, Target{u.ID, u.Source})
			}
			output.Outputs = append(output.Outputs, StructuredTarget{Path: a.TargetPath, Slots: slots})
		}
	}
	b, _ := Encode(output)
	_, deltaPath, err := ImportStructuredAssets(root, bundle, b, Provenance{Provider: "codex", Model: "gpt-5.6-sol-high", Session: "synthetic-generation", GeneratedAt: "2026-10-17T00:00:00Z"})
	if err != nil {
		t.Fatal("replacement import", err)
	}
	delta, err := ReferenceFile(root, deltaPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := CheckStructuredGeneration(root, delta); err != nil {
		t.Fatal("replacement current checker", err)
	}
	closure, _, err := BuildPackageClosure(root, "zh-CN", "learn-docs-v1", page, []Reference{gen, delta})
	if err != nil {
		t.Fatal("existing closure lifecycle", err)
	}
	if closure.StructuredBaseline == nil || *closure.StructuredBaseline != baseline {
		t.Fatal("predecessor closure lost")
	}
	t.Log("full current 12-slot replacement + existing delta closure PASS; no synthetic Reviewer or Surface acceptance granted by reopen")
}
