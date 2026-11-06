package i18n

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOrdinaryQCRestrictedPredecessorRejected(t *testing.T) {
	root, c, o := nestedQCRevisionFixture(t)
	e, err := ExportRetranslationBatch(root, c, o)
	if err != nil {
		t.Fatal(err)
	}
	m := readRetranslationManifest(t, root, e.BatchID)
	g, err := LoadGlossary(root, o.Locale)
	if err != nil {
		t.Fatal(err)
	}
	inputs, err := revisionGenerationInputs(root, c, o.Locale, m.Units)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range m.Units {
		one := m
		one.Units = []RetranslationBatchUnit{r}
		one.UnitCount = 1
		if err := validateOrdinaryQCRevisionTemplates(root, c, g, &one, inputs); !errors.Is(err, ErrRevisionTemplateIncapable) {
			t.Fatal("opaque predecessor accepted", r.UnitID, err)
		}
	}
	before, err := os.ReadFile(filepath.Join(root, "data/retranslation-runs", o.Locale, e.BatchID, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if b, _, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: o.Locale, BatchID: e.BatchID}); !errors.Is(err, ErrRevisionTemplateIncapable) || len(b) != 0 {
		t.Fatal("export did not stop before unusable transport", err)
	}
	after, _ := os.ReadFile(filepath.Join(root, "data/retranslation-runs", o.Locale, e.BatchID, "manifest.json"))
	if !bytes.Equal(before, after) {
		t.Fatal("failed export mutated batch")
	}
}

func TestOrdinaryQCRevisionTemplateArtifactFraming(t *testing.T) {
	c := retranslationTestCatalog(1)
	u, err := c.Unit("lesson/1")
	if err != nil {
		t.Fatal(err)
	}
	g := &Glossary{}
	p, err := prepareTranslationUnitInput(u, g)
	if err != nil {
		t.Fatal(err)
	}
	restored, failures := p.restore(p.Text)
	if len(failures) != 0 {
		t.Fatal(failures)
	}
	// Historical candidate EOF need not already have the new batch's framing.
	candidate := bytes.TrimRight([]byte(restored), "\n")
	m := &RetranslationBatchManifest{ArtifactEOF: retranslationArtifactEOFSingleLF, Units: []RetranslationBatchUnit{{UnitID: u.ID, InputPath: "inputs/fixture.article", RevisionFeedbackSource: "quality_check"}}}
	inputs := []generationBundleInput{{path: "formal/protected-candidates/fixture.article", data: []byte(p.Text)}, {path: "formal/candidates/fixture.article", data: candidate}}
	if err := validateOrdinaryQCRevisionTemplates("", c, g, m, inputs); err != nil {
		t.Fatal("ordinary baseline framing rejected", err)
	}
}
