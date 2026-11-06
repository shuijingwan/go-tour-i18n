package i18n

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestNestedQCRevisionTransportCurrentness(t *testing.T) {
	root, c, o := nestedQCRevisionFixture(t)
	e, err := ExportRetranslationBatch(root, c, o)
	if err != nil {
		t.Fatal(err)
	}
	// Inspect the shared predecessor graph without exporting an executable
	// Bundle: this historical restricted fixture is now correctly blocked by
	// validateOrdinaryQCRevisionTemplates. The authority/drift tests remain.
	dir := filepath.Join(root, "data/retranslation-runs", o.Locale, e.BatchID)
	probe := func(cat *Catalog) ([]generationBundleInput, error) {
		m, err := readRetranslationProcessManifest(dir, o.Locale, e.BatchID)
		if err != nil {
			return nil, err
		}
		for _, r := range m.Units {
			input, err := os.ReadFile(filepath.Join(dir, r.InputPath))
			if err != nil || sum(input) != r.InputSHA256 {
				return nil, fmt.Errorf("fixture input changed: %s", r.UnitID)
			}
		}

		return revisionGenerationInputs(root, cat, o.Locale, m.Units)
	}
	baselineInputs, err := probe(c)
	if err != nil {
		t.Fatal(err)
	}
	m := readRetranslationManifest(t, root, e.BatchID)
	s, err := readQualityCheckSnapshot(root, o.Locale, o.PreviousSnapshotID, false)
	if err != nil {
		t.Fatal(err)
	}
	review, err := RequireUnifiedGlossaryReview(root, o.Locale)
	if err != nil {
		t.Fatal(err)
	}
	u := s.Units[4]
	vdata, err := os.ReadFile(filepath.Join(root, u.ValidationPath))
	if err != nil {
		t.Fatal(err)
	}
	var v RetranslationValidation
	if err = json.Unmarshal(vdata, &v); err != nil {
		t.Fatal(err)
	}
	paths := []struct{ name, path string }{
		{"snapshot", "data/quality-check-snapshots/" + o.Locale + "/" + o.PreviousSnapshotID + "/manifest.json"},
		{"quality-check-results", "data/quality-check-snapshots/" + o.Locale + "/" + o.PreviousSnapshotID + "/quality-check-results.json"},
		{"candidate", u.CandidatePath}, {"validation", u.ValidationPath},
		{"provenance-raw", "data/retranslation-runs/" + o.Locale + "/" + u.SelectedBatchID + "/" + v.RawResponsePath},
		{"glossary", glossaryReviewGlossaryPath(o.Locale)}, {"unified-review", review.Path},
		{"generation-input", "data/retranslation-runs/" + o.Locale + "/" + e.BatchID + "/" + m.Units[0].InputPath},
	}
	baseline := readRetranslationManifest(t, root, u.SelectedBatchID).Units[0].PreviousSnapshotID
	paths = append(paths, struct{ name, path string }{"finalization", "data/quality-check-snapshots/" + o.Locale + "/" + baseline + "/finalization.json"})
	_, _, scope, _, err := translationAuthorizationScope(c)
	if err != nil {
		t.Fatal(err)
	}
	auths, err := filepath.Glob(filepath.Join(root, "data/translation-authorizations", o.Locale, scope, "*", "*.json"))
	if err != nil || len(auths) != 1 {
		t.Fatal("fixture authorization", auths, err)
	}
	authPath, _ := filepath.Rel(root, auths[0])
	paths = append(paths, struct{ name, path string }{"authorization", filepath.ToSlash(authPath)})
	done := contentidentity.BeginMemoization()
	defer done()
	for _, x := range paths {
		t.Run(x.name, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join(root, x.path))
			if err != nil {
				t.Fatal(err)
			}
			changed := append(bytes.Clone(b), '\n')
			if x.name == "authorization" {
				var a CurrentTranslationAuthorization
				if err := json.Unmarshal(b, &a); err != nil {
					t.Fatal(err)
				}
				a.Identity = strings.Repeat("0", 64)
				changed = append(mustJSON(a), '\n')
			}
			if x.name == "quality-check-results" {
				changed = bytes.Replace(b, []byte("Exact independent finding"), []byte("Changed independent finding"), 1)
			}
			writeSnapshotSource(t, root, x.path, changed)
			defer writeSnapshotSource(t, root, x.path, b)
			if got, err := probe(c); err == nil && reflect.DeepEqual(got, baselineInputs) {
				t.Fatal("drift CURRENT")
			}
		})
	}
	manifestPath := "data/retranslation-runs/" + o.Locale + "/" + e.BatchID + "/manifest.json"
	original, err := os.ReadFile(filepath.Join(root, manifestPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"wrong-finding", "missing-finding", "wrong-rating", "wrong-snapshot"} {
		t.Run(name, func(t *testing.T) {
			var changed RetranslationBatchManifest
			if err := json.Unmarshal(original, &changed); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "wrong-finding":
				changed.Units[0].PreviousFinding = "Another finding"
			case "missing-finding":
				changed.Units[0].PreviousFinding = ""
			case "wrong-rating":
				changed.Units[0].PreviousRating = "A"
			case "wrong-snapshot":
				changed.Units[0].PreviousSnapshotID = "unknown-snapshot"
			}
			writeSnapshotSource(t, root, manifestPath, append(mustJSON(changed), '\n'))
			defer writeSnapshotSource(t, root, manifestPath, original)
			if _, err := probe(c); err == nil {
				t.Fatal("invalid feedback accepted")
			}
		})
	}
	t.Run("partial-package", func(t *testing.T) {
		copy := *c
		copy.TranslationScope = copy.TranslationScope[:4]
		if _, err := probe(&copy); err == nil {
			t.Fatal("partial scope accepted")
		}
	})
	t.Run("source-fresh-catalog", func(t *testing.T) {
		copy := *c
		copy.Adapted = append([]AdaptedTranslationUnit(nil), c.Adapted...)
		b := copy.Adapted[4]
		b.Unit.Source = append(bytes.Clone(b.Unit.Source), '\n')
		b.Unit.SourceSHA256 = sum(b.Unit.Source)
		copy.Adapted[4] = b
		if _, err := probe(&copy); err == nil {
			t.Fatal("source drift accepted")
		}
	})
	t.Run("wrong-package", func(t *testing.T) {
		copy := *c
		copy.TranslationScope = []string{"other-package/unit"}
		if _, err := probe(&copy); err == nil {
			t.Fatal("wrong package accepted")
		}
	})
	t.Run("A-revision", func(t *testing.T) {
		bad := o
		bad.UnitIDs = []string{s.Units[0].UnitID}
		if _, err := ExportRetranslationBatch(root, c, bad); err == nil {
			t.Fatal("ordinary A revision accepted")
		}
	})
	if got, err := probe(c); err != nil || !reflect.DeepEqual(got, baselineInputs) {
		t.Fatal("restored exact authority no longer CURRENT", err)
	}
}

// Mutate after the nested predecessor projection is prepared, while a later
// selected Unit is being consumed. This is not a prior-command stale test.
type qcRevisionMutationAdapter struct {
	validationReopenAdapter
	ready, changed *bool
	mutate         func()
	last, trigger  bool
}

func (a qcRevisionMutationAdapter) CommentRevisionBodies(b []byte) ([]CommentRevisionBody, error) {
	out, err := a.validationReopenAdapter.CommentRevisionBodies(b)
	if a.last && err == nil {
		*a.ready = true
	}
	return out, err
}
func (a qcRevisionMutationAdapter) ProtectedInput(g *Glossary) (string, []string, error) {
	if a.trigger && *a.ready && !*a.changed {
		*a.changed = true
		a.mutate()
	}
	return a.validationReopenAdapter.ProtectedInput(g)
}

func TestNestedQCRevisionConcurrentDependencyMutation(t *testing.T) {
	for _, kind := range []string{"candidate", "quality-check-results", "raw", "glossary", "unified-review"} {
		t.Run(kind, func(t *testing.T) {
			root, c, o := nestedQCRevisionFixture(t)
			e, err := ExportRetranslationBatch(root, c, o)
			if err != nil {
				t.Fatal(err)
			}
			s, err := readQualityCheckSnapshot(root, o.Locale, o.PreviousSnapshotID, false)
			if err != nil {
				t.Fatal(err)
			}
			path := s.Units[4].CandidatePath
			switch kind {
			case "quality-check-results":
				path = "data/quality-check-snapshots/" + o.Locale + "/" + o.PreviousSnapshotID + "/quality-check-results.json"
			case "raw":
				data, err := os.ReadFile(filepath.Join(root, s.Units[4].ValidationPath))
				if err != nil {
					t.Fatal(err)
				}
				var v RetranslationValidation
				if err = json.Unmarshal(data, &v); err != nil {
					t.Fatal(err)
				}
				path = "data/retranslation-runs/" + o.Locale + "/" + s.Units[4].SelectedBatchID + "/" + v.RawResponsePath
			case "glossary":
				path = glossaryReviewGlossaryPath(o.Locale)
			case "unified-review":
				r, err := RequireUnifiedGlossaryReview(root, o.Locale)
				if err != nil {
					t.Fatal(err)
				}
				path = r.Path
			}
			ready, changed := false, false
			mutate := func() {
				data, err := os.ReadFile(filepath.Join(root, path))
				if err != nil {
					t.Fatal(err)
				}
				writeSnapshotSource(t, root, path, append(data, '\n'))
			}
			for i, b := range c.Adapted {
				b.Adapter = qcRevisionMutationAdapter{b.Adapter.(validationReopenAdapter), &ready, &changed, mutate, i == 8, i == 5}
				c.Adapted[i] = b
				c.TranslationPackages[0].Units[i] = b
			}
			if _, _, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: o.Locale, BatchID: e.BatchID}); err == nil || errors.Is(err, ErrRevisionTemplateIncapable) {
				t.Fatal("concurrent mutation must fail before the template capability guard", kind, err)
			}
			if !changed {
				t.Fatal("mutation did not execute")
			}
		})
	}
}
