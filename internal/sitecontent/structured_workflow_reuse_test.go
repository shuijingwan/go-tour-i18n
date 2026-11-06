package sitecontent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestStructuredWorkflowInvocationGraph(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, baseline, generation, _ := structuredLifecycleFixture(t, "site-v2-shell")
	v := newClosureValidation(root)
	if err := v.currentInputs("zh-CN"); err != nil {
		t.Fatal(err)
	}
	p, err := v.planStructuredAssets(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	bundle, m, err := v.exportStructuredAssets(root, "zh-CN", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.verify(); err != nil {
		t.Fatal(err)
	}
	planBytes, _ := Encode(p)
	exportedPlan, _ := Encode(m.Plan)
	if !bytes.Equal(planBytes, exportedPlan) {
		t.Fatal("export changed exact plan/expected semantics")
	}
	if v.counts["current-review"] != 1 || v.counts["current-corpus"] != 1 || v.counts["generation:"+generation.Path] != 1 || v.counts["closure:"+baseline.Path] != 1 {
		t.Fatal("invocation repeated complete global/historical validation", v.counts)
	}
	// Disable only proof reuse: every original validator still runs. This
	// differential fixture is synthetic; no accepted real Bundle is exported.
	uncached := newClosureValidation(root)
	uncached.disabled = true
	if err := uncached.currentInputs("zh-CN"); err != nil {
		t.Fatal(err)
	}
	want, _, err := uncached.exportStructuredAssets(root, "zh-CN", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := uncached.verify(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bundle, want) {
		t.Fatal("proof reuse changed canonical Bundle bytes/schema/identity")
	}
	if _, err := CheckStructuredAssetsBundle(root, bundle); err != nil {
		t.Fatal("synthetic CLI checker backend", err)
	}
	if _, _, _, err := v.historicalStructuredGeneration(root, Reference{generation.Path, strings.Repeat("0", 64)}); err == nil {
		t.Fatal("cached path accepted wrong expected SHA")
	}
	var r StructuredGeneration
	if err := readReference(root, generation, &r); err != nil {
		t.Fatal(err)
	}
	for label, name := range map[string]string{
		"current-corpus": "data/glossary-source-corpus.json", "current-glossary": "locales/zh-CN/glossary.yaml", "current-review": p.Review.Path,
		"generation": generation.Path, "archived-inputs": r.Bundle.Path, "baseline": baseline.Path, "current-global": GlobalPath, "current-source-authority": SnapshotPath, "locale-completion": LocalePath("zh-CN"),
	} {
		t.Run(label, func(t *testing.T) {
			before, err := readRegular(root, name)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, name, before)
			writeFixture(t, root, name, append(bytes.Clone(before), ' '))
			if err := v.verify(); err == nil {
				t.Fatal("invocation graph accepted mutable dependency drift", name)
			}
			// Mutable corpus/completion JSON is semantically canonicalized by a
			// fresh invocation; harmless whitespace need not invalidate it. Only
			// the in-flight proof above promises exact observed bytes.
			if name == "data/glossary-source-corpus.json" || name == LocalePath("zh-CN") {
				return
			}
			if _, _, err := ExportStructuredAssets(root, "zh-CN"); err == nil {
				t.Fatal("fresh export accepted stale authority", name)
			}
		})
	}
	for _, name := range []string{"data/glossary-source-corpus.json", LocalePath("zh-CN")} {
		t.Run("fresh-semantic-mutation/"+name, func(t *testing.T) {
			before, err := readRegular(root, name)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, name, before)
			// Change an actual authority field, not merely its JSON formatting.
			var object map[string]any
			if err := StrictJSON(before, &object); err != nil {
				t.Fatal(err)
			}
			object["schema"] = "unsupported/changed"
			b, _ := Encode(object)
			writeFixture(t, root, name, b)
			if _, _, err := ExportStructuredAssets(root, "zh-CN"); err == nil {
				t.Fatal("semantic authority mutation accepted", name)
			}
		})
	}
}

func TestHistoricalPageParserBoundary(t *testing.T) {
	_, old, current, _ := replacementPageFixture(t)
	// The frozen representation differs from today's canonical parser. Its
	// integrity is nevertheless exact, with no use of an ignored current parse.
	before := []Document{}
	for _, binding := range old.TranslationPackages[0].Units {
		before = append(before, binding.Adapter.(PageAdapter).Document)
	}
	if err := validateDocumentRepresentations(before, false); err != nil {
		t.Fatal("frozen exact event", err)
	}
	if err := validateDocumentRepresentations(before, true); err == nil {
		t.Fatal("historical representation treated as current")
	}
	after := []Document{}
	for _, binding := range current.TranslationPackages[0].Units {
		after = append(after, binding.Adapter.(PageAdapter).Document)
	}
	if err := validateDocumentRepresentations(after, true); err != nil {
		t.Fatal("current canonical protection", err)
	}
	for _, kind := range []string{"source", "coordinates", "protection", "ID"} {
		t.Run(kind, func(t *testing.T) {
			b, _ := Encode(before)
			var docs []Document
			if err := StrictJSON(b, &docs); err != nil {
				t.Fatal(err)
			}
			d := &docs[0]
			switch kind {
			case "source":
				d.SourceSHA = strings.Repeat("0", 64)
			case "coordinates":
				d.Slots[0].End++
			case "protection":
				d.Units[0].Source += "tampered"
			case "ID":
				d.Units[0].ID = strings.Repeat("0", 64)
			}
			if err := validateDocumentRepresentations(docs, false); err == nil {
				t.Fatal("historical gate weakened", kind)
			}
		})
	}
}
