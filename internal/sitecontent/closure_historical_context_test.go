package sitecontent

import (
	"os"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// The unit fixture isolates graph consumption and uses the real canonical
// lineage reader. The full original A proof is exercised by the repository
// historical-closure regression below, not replaced by a synthetic callback.
func historicalClosureBindingFixture(t *testing.T) (string, *i18n.Catalog, *i18n.HistoricalFinalizationIntegrity, i18n.GlossaryArchiveReference, Reference, *ShellRepresentationReattestation) {
	t.Helper()
	root, ref, ancestor, head := historicalReattestationFixture(t)
	c := &i18n.Catalog{HistoricalContextReader: readHistoricalLanguageContext}
	p := &i18n.HistoricalFinalizationIntegrity{
		Finalization: i18n.QualityCheckFinalization{Locale: "fr-FR", SnapshotID: "synthetic-original-A"},
		References:   []i18n.GlossaryArchiveReference{ref},
	}
	return root, c, p, ref, ancestor, head
}

func TestHistoricalClosureContextBinding(t *testing.T) {
	root, c, p, ref, _, _ := historicalClosureBindingFixture(t)
	v := newClosureValidation(root)
	frame := v.beginProof()
	if err := v.bindHistoricalTranslationReference(c, p, ref); err != nil {
		t.Fatal("canonical historical context did not survive graph binding", err)
	}
	v.endProof()
	v.proofs["historical"] = frame
	physical, err := readRegular(root, ref.Path)
	if err != nil {
		t.Fatal(err)
	}
	if v.deps[ref.Path] != digest(physical) || v.deps[ref.Path] == ref.SHA256 || frame.files[ref.Path] != digest(physical) || len(frame.historicalContexts) != 1 {
		t.Fatal("historical SHA contaminated physical dependency observations")
	}
	outer := v.beginProof()
	if err := v.reuse("historical"); err != nil {
		t.Fatal("canonical context failed on cache reuse", err)
	}
	v.endProof()
	if len(outer.historicalContexts) != 1 || outer.files[ref.Path] != digest(physical) {
		t.Fatal("cached proof lost its historical/current dependency domains")
	}
	if err := v.verify(); err != nil {
		t.Fatal("canonical context failed at final verification", err)
	}
	if err := v.bind(Reference{ref.Path, ref.SHA256}); err == nil {
		t.Fatal("historical binding weakened ordinary physical bind")
	}
}

func TestHistoricalClosureContextBindingFailClosed(t *testing.T) {
	for _, name := range []string{"missing-ancestor", "broken-head-hash", "wrong-frozen-sha", "outside-proof", "missing-reader"} {
		t.Run(name, func(t *testing.T) {
			root, c, p, ref, ancestor, head := historicalClosureBindingFixture(t)
			switch name {
			case "missing-ancestor":
				if err := os.Remove(pathJoin(root, ancestor.Path)); err != nil {
					t.Fatal(err)
				}
			case "broken-head-hash":
				writeFixture(t, root, shellReattestationReceipt(head).Path, []byte("corrupt immutable head"))
			case "wrong-frozen-sha":
				ref.SHA256 = digest([]byte("not a canonical historical Reference"))
				p.References[0] = ref
			case "outside-proof":
				p.References = nil
			case "missing-reader":
				c.HistoricalContextReader = nil
			}
			if err := newClosureValidation(root).bindHistoricalTranslationReference(c, p, ref); err == nil {
				t.Fatal("unproven historical context entered dependency graph")
			}
		})
	}
}

func TestHistoricalClosureContextRechecksLineage(t *testing.T) {
	for _, name := range []string{"reuse", "verify"} {
		t.Run(name, func(t *testing.T) {
			root, c, p, ref, ancestor, _ := historicalClosureBindingFixture(t)
			v := newClosureValidation(root)
			frame := v.beginProof()
			if err := v.bindHistoricalTranslationReference(c, p, ref); err != nil {
				t.Fatal(err)
			}
			v.endProof()
			v.proofs["historical"] = frame
			// The pointer bytes remain unchanged. A cached PASS cannot hide a
			// subsequently missing immutable member of its historical lineage.
			if err := os.Remove(pathJoin(root, ancestor.Path)); err != nil {
				t.Fatal(err)
			}
			err := v.verify()
			if name == "reuse" {
				err = v.reuse("historical")
			}
			if err == nil {
				t.Fatal("historical lineage PASS was cached across evidence loss")
			}
		})
	}
}

func TestHistoricalClosureContextPhysicalDriftRejected(t *testing.T) {
	for _, name := range []string{"bytes", "mode"} {
		t.Run(name, func(t *testing.T) {
			root, c, p, ref, ancestor, _ := historicalClosureBindingFixture(t)
			v := newClosureValidation(root)
			if err := v.bindHistoricalTranslationReference(c, p, ref); err != nil {
				t.Fatal(err)
			}
			if name == "bytes" {
				// Even an admissible older head is a physical mutation during
				// this invocation; historical authority cannot excuse the drift.
				b, _ := Encode(ancestor)
				writeFixture(t, root, ref.Path, b)
			} else if err := os.Chmod(pathJoin(root, ref.Path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := v.verify(); err == nil {
				t.Fatal("historical context suppressed physical dependency drift")
			}
		})
	}
}

func TestCurrentClosurePhysicalDependencyRemainsStrict(t *testing.T) {
	root := t.TempDir()
	path := "data/current-input.json"
	b := []byte("current physical bytes\n")
	writeFixture(t, root, path, b)
	v := newClosureValidation(root)
	ref := Reference{path, digest(b)}
	if err := v.bind(ref); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, path, []byte("changed current physical bytes\n"))
	if err := v.bind(ref); err == nil {
		t.Fatal("changed current dependency accepted")
	}
	if err := v.verify(); err == nil {
		t.Fatal("changed current dependency survived final verification")
	}
}

func TestSurface009HistoricalStructuredClosureContextReadonly(t *testing.T) {
	if os.Getenv("GO_TOUR_SURFACE009_READONLY") != "1" {
		t.Skip("explicit Surface009 readonly historical closure regression")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	var receipt IntegratedSurfaceReceipt
	if err := readReference(root, Reference{surface009MixedPath, surface009MixedSHA}, &receipt); err != nil {
		t.Fatal(err)
	}
	for _, ref := range receipt.Scope.Closures {
		var c PackageClosure
		if err := readReference(root, ref, &c); err != nil {
			t.Fatal(err)
		}
		if c.Package != "site-v2-shell" {
			continue
		}
		v := newClosureValidation(root)
		baseline, err := v.historicalStructuredClosure(root, ref, map[string]bool{})
		if err != nil {
			t.Fatal("full original Page A + frozen structured closure", err)
		}
		for _, file := range c.Files {
			b, ok := baseline.Files[file.Path]
			if !ok || digest(b) != file.SHA256 {
				t.Fatal("frozen localized output changed", file.Path)
			}
		}
		if len(v.historicalContexts) == 0 {
			t.Fatal("real historical context was not bound")
		}
		for key := range v.historicalContexts {
			if v.deps[key.reference.Path] == key.reference.SHA256 {
				t.Fatal("physical and historical SHA were conflated")
			}
		}
		if err := v.verify(); err != nil {
			t.Fatal("final historical/physical guard", err)
		}
		t.Logf("REAL_HISTORICAL_CLOSURE=PASS CLOSURE=%s FROZEN_OUTPUTS_EXACT=%d HISTORICAL_CONTEXTS=%d CURRENT_LANGUAGE_ELIGIBILITY=not-granted", ref.Path, len(c.Files), len(v.historicalContexts))
		return
	}
	t.Fatal("no recorded shell closure")
}
