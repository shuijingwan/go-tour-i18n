package sitecontent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestFrozenStructuredRepresentation(t *testing.T) {
	raw := []byte("- title: Example guide\n  eyebrow: PDF\n  url: https://example.com/guide\n")
	s := Source{Path: "_content/learn/example.yaml", Kind: "data", SHA256: digest(raw), Dependencies: []Dependency{}}
	current, err := ParseDocument(s, raw)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate an archived protector that did not expose the machine-only PDF
	// scalar. This is a different frozen representation of exactly these bytes.
	old := *current
	old.Units = []Unit{}
	for _, u := range current.Units {
		if u.Source != "PDF" {
			old.Units = append(old.Units, u)
		}
	}
	if len(old.Units) != len(current.Units)-1 {
		t.Fatal("fixture must exercise representation drift")
	}
	var structure bytes.Buffer
	pos := 0
	for _, u := range old.Units {
		context := strings.TrimSuffix(strings.TrimPrefix(u.Locator, s.Path+"#"), ":"+u.SourceSHA+":1")
		structure.Write(raw[pos:u.Start])
		structure.WriteString("\x00" + context + "\x00" + protectionIdentity(u.Protected))
		pos = u.End
	}
	structure.Write(raw[pos:])
	old.StructureSHA = identity(struct {
		Structure    string
		Dependencies []Dependency
	}{digest(structure.Bytes()), old.Dependencies})
	for name, d := range map[string]Document{"unchanged": *current, "evolved-parser": old} {
		t.Run(name, func(t *testing.T) {
			values := map[string]string{}
			for _, u := range d.Units {
				values[u.ID] = u.Source
			}
			out, err := reconstructFrozenStructuredRepresentation(d, raw, values)
			if err != nil || !bytes.Equal(out, raw) {
				t.Fatal("historical reconstruction", err)
			}
			translated := map[string]string{}
			for k, v := range values {
				translated[k] = strings.ReplaceAll(v, "Example guide", "Synthetic guide")
			}
			if _, err := reconstructFrozenStructuredRepresentation(d, raw, translated); err != nil {
				t.Fatal(err)
			}
		})
	}
	clone := func() Document {
		b, _ := Encode(old)
		var d Document
		if err := StrictJSON(b, &d); err != nil {
			t.Fatal(err)
		}
		return d
	}
	for _, tc := range []struct {
		name   string
		mutate func(*Document)
	}{
		{"document-source-hash", func(d *Document) { d.SourceSHA = strings.Repeat("0", 64) }},
		{"slot-text", func(d *Document) { d.Units[0].Source = "forged" }},
		{"slot-id", func(d *Document) { d.Units[0].ID = strings.Repeat("0", 64) }},
		{"slot-coordinates", func(d *Document) { d.Units[0].End++ }},
		{"negative-coordinates", func(d *Document) { d.Units[0].Start = -1 }},
		{"slot-omitted", func(d *Document) { d.Units = nil }},
		{"structure-hash", func(d *Document) { d.StructureSHA = strings.Repeat("0", 64) }},
		{"dependency", func(d *Document) {
			d.Dependencies = append(d.Dependencies, Dependency{Kind: "asset", Path: "asset.png", SHA256: strings.Repeat("1", 64)})
		}},
		{"locator-path", func(d *Document) { d.Path = "_content/learn/other.yaml" }},
		{"unknown-contract", func(d *Document) { d.Contract = "unknown/v1" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := clone()
			tc.mutate(&d)
			if verifyFrozenStructuredRepresentation(d, raw) == nil {
				t.Fatal("tampered frozen Document accepted")
			}
		})
	}
	t.Run("source-bytes", func(t *testing.T) {
		if verifyFrozenStructuredRepresentation(old, append(bytes.Clone(raw), ' ')) == nil {
			t.Fatal("source tamper accepted")
		}
	})
	for name, values := range map[string]map[string]string{
		"missing-output":   {},
		"wrong-slot":       {"unknown": "Synthetic target"},
		"syntax-injection": {old.Units[0].ID: "Injected\n  url: https://evil.example"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := reconstructFrozenStructuredRepresentation(old, raw, values); err == nil {
				t.Fatal("invalid target accepted")
			}
		})
	}
	// A frozen proof is not authority to carry into a different current input.
	// Current eligibility must inspect registered source/reopen authority.
	// An empty root fails before the stale classification under that contract.
	root := siteFixture(t, "current-authorization-fresh-fixture")
	ref := Reference{"historical-generation.json", strings.Repeat("1", 64)}
	a := StructuredAsset{ID: "source:" + s.Path, Package: "learn-docs-v1", SourceSHA: strings.Repeat("2", 64), Document: &old, TargetPath: "target.yaml"}
	b := &structuredBaseline{Closure: PackageClosure{Locale: "zh-CN", Package: a.Package}, Owners: map[string]Reference{s.Path: ref}, Files: map[string][]byte{s.Path: raw}, Generations: map[string]*StructuredGeneration{ref.Path: {Plan: StructuredPlan{Assets: []StructuredAsset{a}}, Output: StructuredOutput{Outputs: []StructuredTarget{{Path: a.TargetPath}}}}}, Inputs: map[string]map[string][]byte{ref.Path: {}}}
	for _, tc := range []struct {
		name     string
		document Document
	}{
		{"current-representation-drift", *current},
		{"current-source-drift", func() Document { d := old; d.SourceSHA = digest([]byte("changed")); return d }()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a.Document = &tc.document
			x, err := newClosureValidation(root).assessStructuredAsset(root, StructuredPlan{Locale: "zh-CN"}, a, b)
			if err != nil || x.Classification != "stale" {
				t.Fatal("historical proof bypassed current eligibility", x, err)
			}
		})
	}
}

func TestFrozenStructuredProtectedBindings(t *testing.T) {
	_, d, raw := roundtripFixture(t, "historical.md", "page", "Build a CLI with [a guide](/fixed).\n")
	u := d.Units[0]
	if len(u.Protected) != 3 {
		t.Fatal("expected frozen atom and pair")
	}
	if _, err := reconstructFrozenStructuredRepresentation(*d, raw, map[string]string{u.ID: "使用 ⟦P0⟧，参阅 ⟦P1⟧指南⟦P2⟧。"}); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"missing tokens", "⟦P0⟧ ⟦P1⟧⟦P1⟧⟦P2⟧", "⟦P0⟧ ⟦P2⟧⟦P1⟧", "⟦P0⟧ ⟦P1⟧guide⟦P2⟧ ⟦P999⟧"} {
		if _, err := reconstructFrozenStructuredRepresentation(*d, raw, map[string]string{u.ID: bad}); err == nil {
			t.Fatal("protected binding mutation accepted", bad)
		}
	}
	d.Units[0].Protected = append(d.Units[0].Protected, d.Units[0].Protected[0])
	if verifyFrozenStructuredRepresentation(*d, raw) == nil {
		t.Fatal("duplicate frozen token accepted")
	}
}

func TestFrozenStructuredGenerationIntegrity(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	finish := contentidentity.BeginMemoization()
	defer finish()
	root, _, _, _, ref := structuredRecoveryFixture(t)
	r, m, inputs, err := readStructuredGeneration(root, ref)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := historicalStructuredGeneration(root, ref); err != nil {
		t.Fatal("unchanged historical event", err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*StructuredGeneration, map[string][]byte)
	}{
		{"frozen-source-bytes", func(r *StructuredGeneration, fs map[string][]byte) {
			for _, a := range r.Plan.Assets {
				if a.Document != nil {
					p := "source/" + a.Document.Path
					fs[p] = append(bytes.Clone(fs[p]), ' ')
					return
				}
			}
		}},
		{"frozen-Document", func(r *StructuredGeneration, _ map[string][]byte) {
			for i := range r.Plan.Assets {
				if r.Plan.Assets[i].Document != nil {
					r.Plan.Assets[i].Document.Units[0].End++
					return
				}
			}
		}},
		{"old-glossary-identity", func(r *StructuredGeneration, _ map[string][]byte) { r.Plan.GlossarySHA = strings.Repeat("0", 64) }},
		{"old-glossary-bytes", func(r *StructuredGeneration, fs map[string][]byte) {
			p := "inputs/locales/" + r.Plan.Locale + "/glossary.yaml"
			fs[p] = append(bytes.Clone(fs[p]), ' ')
		}},
		{"input-corpus-identity", func(r *StructuredGeneration, _ map[string][]byte) { r.Plan.CorpusSHA = strings.Repeat("0", 64) }},
		{"output-exact-set", func(r *StructuredGeneration, _ map[string][]byte) { r.Output.Outputs = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := Encode(r)
			var copy StructuredGeneration
			if err := StrictJSON(b, &copy); err != nil {
				t.Fatal(err)
			}
			fs := map[string][]byte{}
			for p, b := range inputs {
				fs[p] = bytes.Clone(b)
			}
			tc.mutate(&copy, fs)
			if _, _, _, err := validateFrozenStructuredGeneration(root, &copy, m, fs); err == nil {
				t.Fatal("tampered historical event accepted")
			}
		})
	}
	for name, path := range map[string]string{"Bundle-tamper": r.Bundle.Path, "Generation-output-tamper": ref.Path} {
		t.Run(name, func(t *testing.T) {
			b, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, path, b)
			writeFixture(t, root, path, append(bytes.Clone(b), []byte("tamper")...))
			if _, _, _, err := historicalStructuredGeneration(root, ref); err == nil {
				t.Fatal("immutable archive tamper accepted")
			}
		})
	}
}
