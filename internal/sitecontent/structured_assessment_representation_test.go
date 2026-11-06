package sitecontent

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestStructuredAssessmentCanonicalRepresentation(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, closureRef, genRef, adoptionRef := shellAdoptionFixture(t)
	v := newClosureValidation(root)
	b, err := v.historicalStructuredClosure(root, closureRef, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	g := mustGlobal(t, root)
	l, err := readLocaleAuthority(root, g, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	b.Surface = l.Packages[0].Evidence[1]
	r, _, _, err := readStructuredGeneration(root, genRef)
	if err != nil {
		t.Fatal(err)
	}
	var a StructuredAsset
	for _, asset := range r.Plan.Assets {
		if asset.Document != nil && asset.Document.Path == "data/site-shell/translation.html" {
			a = asset
		}
	}
	if a.Document == nil {
		t.Fatal("fixture translation asset missing")
	}
	corpus, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		t.Fatal(err)
	}
	groups, err := structuredContributorGroups(corpus.Contributors)
	if err != nil {
		t.Fatal(err)
	}
	a.SourceSHA = identity(groups[a.ID])
	a.State = "carry"
	review, err := v.unifiedReview("zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	gb, err := readRegular(root, "locales/zh-CN/glossary.yaml")
	if err != nil {
		t.Fatal(err)
	}
	p := StructuredPlan{Locale: "zh-CN", GlossarySHA: digest(gb), Review: Reference{review.Path, review.SHA256}, Expected: []string{}}
	x, err := v.assessStructuredAsset(root, p, a, b)
	if err != nil || x.Classification != "compatible" {
		t.Fatal(x, err)
	}
	a.Assessment = x
	p.Assets = []StructuredAsset{a}
	if err := newClosureValidation(root).validateStructuredDelta(root, &StructuredGeneration{Plan: p}, &StructuredBundle{}); err != nil {
		t.Fatal("identical frozen/current assessment", err)
	}
	for _, field := range []string{"Classification", "Reason", "Closure", "Generation", "Surface", "OldGlossarySHA", "GlossarySHA", "Review", "SourceSHA", "TargetSHA", "Identity"} {
		t.Run("assessment-"+field, func(t *testing.T) {
			changed := *x
			switch field {
			case "Classification":
				changed.Classification = "affected"
			case "Reason":
				changed.Reason = "forged"
			case "Closure":
				changed.Closure.SHA256 = strings.Repeat("0", 64)
			case "Generation":
				changed.Generation.SHA256 = strings.Repeat("0", 64)
			case "Surface":
				changed.Surface.SHA256 = strings.Repeat("0", 64)
			case "OldGlossarySHA":
				changed.OldGlossarySHA = strings.Repeat("0", 64)
			case "GlossarySHA":
				changed.GlossarySHA = strings.Repeat("0", 64)
			case "Review":
				changed.Review.SHA256 = strings.Repeat("0", 64)
			case "SourceSHA":
				changed.SourceSHA = strings.Repeat("0", 64)
			case "TargetSHA":
				changed.TargetSHA = strings.Repeat("0", 64)
			case "Identity":
				changed.Identity = strings.Repeat("0", 64)
			}
			q := p
			asset := a
			asset.Assessment = &changed
			q.Assets = []StructuredAsset{asset}
			if err := newClosureValidation(root).validateStructuredDelta(root, &StructuredGeneration{Plan: q}, &StructuredBundle{}); err == nil {
				t.Fatal("assessment binding mutation accepted", field)
			}
		})
	}
	for _, field := range []string{"glossary", "review", "contributor", "raw-source", "accepted-target", "impact"} {
		t.Run("current-"+field, func(t *testing.T) {
			q := p
			asset := a
			switch field {
			case "glossary":
				q.GlossarySHA = strings.Repeat("0", 64)
			case "review":
				q.Review.SHA256 = strings.Repeat("0", 64)
			case "contributor":
				asset.SourceSHA = strings.Repeat("0", 64)
			case "raw-source":
				doc := *asset.Document
				doc.SourceSHA = strings.Repeat("0", 64)
				asset.Document = &doc
			case "accepted-target":
				old := bytes.Clone(b.Files[a.Document.Path])
				b.Files[a.Document.Path] = []byte("tamper")
				defer func() { b.Files[a.Document.Path] = old }()
				want, err := v.assessStructuredAsset(root, p, a, b)
				if err != nil || len(assessmentFieldDifferences(*x, *want)) == 0 {
					t.Fatal("accepted target drift undetected", err)
				}
				return
			case "impact":
				glossary, err := readRegular(root, "locales/zh-CN/glossary.yaml")
				if err != nil {
					t.Fatal(err)
				}
				writeFixture(t, root, "locales/zh-CN/glossary.yaml", append(glossary, []byte("\nforbidden:\n  - Synthetic\n")...))
				defer writeFixture(t, root, "locales/zh-CN/glossary.yaml", glossary)
			}
			q.Assets = []StructuredAsset{asset}
			if err := newClosureValidation(root).validateStructuredDelta(root, &StructuredGeneration{Plan: q}, &StructuredBundle{}); err == nil {
				t.Fatal("current input mutation accepted", field)
			}
		})
	}
	// Freeze a synthetic older protection decision inside a native, canonical
	// adoption. This models implementation evolution without changing source.
	var adoption ShellAdoption
	if err := readReference(root, adoptionRef, &adoption); err != nil {
		t.Fatal(err)
	}
	var foundation ShellSuccessorAuthority
	if err := readReference(root, adoption.Foundation, &foundation); err != nil {
		t.Fatal(err)
	}
	d := foundation.TranslationBody
	if len(d.Units) == 0 {
		t.Fatal("missing translation slots")
	}
	u := &d.Units[0]
	priorSHA := u.SourceSHA
	u.Source = "⟦P0⟧"
	raw := b.Inputs[genRef.Path]["source/"+d.Path]
	u.Protected = []ProtectedToken{{Token: "⟦P0⟧", Raw: string(raw[u.Start:u.End]), Role: "atom"}}
	u.SourceSHA = unitSourceIdentity(u.Source, u.Protected)
	u.Locator = strings.Replace(u.Locator, ":"+priorSHA+":", ":"+u.SourceSHA+":", 1)
	u.ID = digest([]byte(UnitContract + "\n" + u.Locator))
	var shape bytes.Buffer
	pos := 0
	for _, u := range d.Units {
		at := strings.LastIndexByte(u.Locator, ':')
		context := strings.TrimSuffix(strings.TrimPrefix(u.Locator[:at], d.Path+"#"), ":"+u.SourceSHA)
		shape.Write(raw[pos:u.Start])
		shape.WriteString("\x00" + context + "\x00" + protectionIdentity(u.Protected))
		pos = u.End
	}
	shape.Write(raw[pos:])
	d.StructureSHA = identity(struct {
		Structure    string
		Dependencies []Dependency
	}{digest(shape.Bytes()), d.Dependencies})
	foundation.TranslationBody = d
	foundation.Identity = ""
	foundation.Identity = identity(foundation)
	foundationRef, err := shellAuthorityReference(&foundation)
	if err != nil {
		t.Fatal(err)
	}
	if err := saveImmutable(root, foundationRef.Path, &foundation); err != nil {
		t.Fatal(err)
	}
	approved, err := BuildShellAdoption(root, foundationRef)
	if err != nil {
		t.Fatal(err)
	}
	if err := saveImmutable(root, shellAdoptionPath(approved), approved); err != nil {
		t.Fatal(err)
	}
	approvedRef, err := ReferenceFile(root, shellAdoptionPath(approved))
	if err != nil {
		t.Fatal(err)
	}
	pointer, _ := Encode(approvedRef)
	writeFixture(t, root, ShellAdoptionCurrent, pointer)
	a.Document = &d
	target := StructuredTarget{Path: a.TargetPath, Slots: []Target{}}
	for _, u := range d.Units {
		target.Slots = append(target.Slots, Target{ID: u.ID, Text: u.Source})
	}
	glossary, err := i18n.ParseGlossaryBytes("zh-CN", gb)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reconstructStructuredTarget(a, raw, target, glossary); err == nil {
		t.Fatal("fixture did not exercise changed parser")
	}
	out, err := newClosureValidation(root).reconstructAssessedStructuredTarget(root, a, raw, target, glossary)
	if err != nil || !bytes.Equal(out, raw) {
		t.Fatal("formally approved current representation rejected", err)
	}
	for name, path := range map[string]string{"adoption-tamper": approvedRef.Path, "current-source-drift": d.Path} {
		t.Run(name, func(t *testing.T) {
			before, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, path, before)
			writeFixture(t, root, path, append(bytes.Clone(before), ' '))
			if _, err := newClosureValidation(root).reconstructAssessedStructuredTarget(root, a, raw, target, glossary); err == nil {
				t.Fatal("unverified representation accepted")
			}
		})
	}
	t.Run("without-adoption-proof", func(t *testing.T) {
		path := filepath.Join(root, ShellAdoptionCurrent)
		if err := os.Rename(path, path+".backup"); err != nil {
			t.Fatal(err)
		}
		defer os.Rename(path+".backup", path)
		if _, err := newClosureValidation(root).reconstructAssessedStructuredTarget(root, a, raw, target, glossary); err == nil {
			t.Fatal("raw equality guessed representation compatibility")
		}
	})
}
