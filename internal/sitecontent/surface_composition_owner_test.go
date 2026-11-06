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
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestSurfaceSharedCompositionFindingAuthority(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	// Every authority and publication below belongs to this disposable fixture.
	// No real locale, Reviewer result or repository formal artifact is written.
	root, oldClosure, oldGeneration, _ := shellAdoptionFixture(t)
	for _, name := range shellRepresentationParserPaths {
		b, err := readRegular(repositoryRoot(t), name)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, name, b)
	}
	representation, err := PlanShellRepresentationSuccessor(root, ShellTeachingRepresentation)
	if err != nil {
		t.Fatal(err)
	}
	if err := saveImmutable(root, "data/composition-representation-plan.json", representation); err != nil {
		t.Fatal(err)
	}
	input, err := ReferenceFile(root, "data/composition-representation-plan.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyShellRepresentationSuccessor(root, input, representation.PriorIdentity, representation.PriorPointerSHA); err != nil {
		t.Fatal(err)
	}
	c, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackagesReadonly(root, c); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage("site-v2-shell"); err != nil {
		t.Fatal(err)
	}
	f := compositionPageFixture(t, root, c)
	page, err := ReferenceFile(root, "data/quality-check-snapshots/zh-CN/"+f.SnapshotID+"/finalization.json")
	if err != nil {
		t.Fatal(err)
	}
	_, generation := importStructuredFixture(t, root)
	_, path, err := FinalizePackageClosure(root, "zh-CN", "site-v2-shell", &page, []Reference{oldGeneration, generation})
	if err != nil {
		t.Fatal(err)
	}
	closure, err := ReferenceFile(root, path)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := BuildIntegratedSurfaceScope(root, "zh-CN", []Reference{closure})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanStructuredAssets(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	var shared StructuredAsset
	sharedCount := 0
	for _, a := range plan.Assets {
		if a.Document != nil && a.Document.Contract == ShellExtractionContract && a.Document.Kind == "data" {
			sharedCount++
			if a.Document.Path == "_content/site.tmpl" {
				shared = a
			}
		}
	}
	if sharedCount != 4 || shared.Document == nil {
		t.Fatal("missing existing four shared structured owners")
	}
	// Isolate the same composition judgment as a successor with already carried
	// shared files. The injected scope builder supplies this frozen exact scope;
	// all Page/structured/closure evidence remains real synthetic lifecycle proof.
	fresh := []SurfaceAsset{}
	for _, a := range scope.Reviewed {
		if len(a.CandidateArtifacts) > 0 {
			fresh = append(fresh, a)
		} else {
			a.State = "carry"
			scope.Carried = append(scope.Carried, a)
		}
	}
	scope.Reviewed = fresh
	scope.Identity = ""
	scope.Identity = identity(*scope)
	if len(scope.Reviewed) != 1 || scope.Reviewed[0].State != "context-review-required" || len(scope.Carried) != 5 {
		t.Fatal("fixture must retain one fresh Page and five carried structured files")
	}
	build := func(string, string, []Reference) (*IntegratedSurfaceScope, error) { copy := *scope; return &copy, nil }
	beforeScope, _ := Encode(scope)
	oldBytes, err := readRegular(root, oldClosure.Path)
	if err != nil {
		t.Fatal(err)
	}
	finding := SurfaceFinding{ID: shared.Document.Path, Class: "structured-surface-language", Finding: "Correct this exact shared structured message.", Artifacts: []i18n.SurfaceCandidateArtifact{}}
	receipt, ref := surfaceReceipt(t, root, c, build, []Reference{closure}, "shared-composition-failed", "failed", []SurfaceFinding{finding})
	checked, err := validatedSurfaceFindingWithScope(root, "zh-CN", ref, c, build)
	if err != nil {
		t.Fatal("recorded result not eligible for structured revision", err)
	}
	clonePlan := func() *StructuredPlan {
		b, _ := Encode(plan)
		var p StructuredPlan
		if err := StrictJSON(b, &p); err != nil {
			t.Fatal(err)
		}
		return &p
	}
	p := clonePlan()
	if err := selectStructuredSurfaceRevision(root, p, checked); err != nil || !reflect.DeepEqual(p.Expected, []string{shared.TargetPath}) {
		t.Fatal("exact shared owner did not select exactly one output", p.Expected, err)
	}
	for _, a := range p.Assets {
		if a.TargetPath != shared.TargetPath && a.State != "context-only" {
			t.Fatal("finding expanded into unrelated structured assets", a.ID)
		}
	}
	afterScope, _ := Encode(&checked.Scope)
	if !bytes.Equal(beforeScope, afterScope) {
		t.Fatal("ownership changed scope bytes or fresh/carried roles")
	}
	afterOld, _ := readRegular(root, oldClosure.Path)
	if !bytes.Equal(oldBytes, afterOld) {
		t.Fatal("historical authority rewritten")
	}
	t.Run("inherited-structured-owner", func(t *testing.T) {
		var child PackageClosure
		if err := readReference(root, closure, &child); err != nil {
			t.Fatal(err)
		}
		child.Structured, child.StructuredBaseline = []Reference{}, &closure
		child.Identity = ""
		child.Identity = identity(child)
		path := workflowPath(child.Locale, child.Package, "closure", child.Identity)
		if err := saveImmutable(root, path, child); err != nil {
			t.Fatal(err)
		}
		childRef, err := ReferenceFile(root, path)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := Encode(scope)
		var inherited IntegratedSurfaceScope
		if err := StrictJSON(b, &inherited); err != nil {
			t.Fatal(err)
		}
		inherited.Closures = []Reference{childRef}
		for _, assets := range [][]SurfaceAsset{inherited.Reviewed, inherited.Carried} {
			for i := range assets {
				assets[i].Evidence = []Reference{childRef}
			}
		}
		inherited.Identity = ""
		inherited.Identity = identity(inherited)
		build := func(string, string, []Reference) (*IntegratedSurfaceScope, error) {
			copy := inherited
			return &copy, nil
		}
		_, ref := surfaceReceipt(t, root, c, build, []Reference{childRef}, "shared-inherited-failed", "failed", []SurfaceFinding{finding})
		checked, err := validatedSurfaceFindingWithScope(root, "zh-CN", ref, c, build)
		if err != nil {
			t.Fatal(err)
		}
		p := clonePlan()
		if err := selectStructuredSurfaceRevision(root, p, checked); err != nil || !reflect.DeepEqual(p.Expected, []string{shared.TargetPath}) {
			t.Fatal("inherited shared owner not selectable", err, p.Expected)
		}
		// The supersession reader receives a Generation whose archive binds the
		// sealed finding. Test its exact inherited parent edge without performing
		// any replacement Generation/import or real repository mutation.
		manifest, _ := Encode(StructuredBundle{RevisionFinding: &ref})
		archive, err := i18n.WriteDeterministicTransportBundle(manifest, nil)
		if err != nil {
			t.Fatal(err)
		}
		const bundlePath = "data/synthetic-revision-owner.zip"
		writeFixture(t, root, bundlePath, archive)
		g := &StructuredGeneration{Plan: StructuredPlan{Locale: "zh-CN"}, Bundle: Reference{Path: bundlePath, SHA256: digest(archive)}}
		if !structuredSupersedes(root, g, generation, finding.ID) || structuredSupersedes(root, g, oldGeneration, finding.ID) || structuredSupersedes(root, g, generation, "_content/menus.yaml") {
			t.Fatal("supersession did not preserve exact inherited asset/parent")
		}
		landClosure := func(c PackageClosure) Reference {
			c.Identity = ""
			c.Identity = identity(c)
			return landSharedReattestationFixture(t, root, workflowPath(c.Locale, c.Package, "closure", c.Identity), c)
		}
		// These reference-level fixtures isolate package selection. The two-delta
		// integration regression separately rebuilds complete current packages.
		unrelatedBase := child
		unrelatedBase.Package = "learn-docs-v1"
		unrelatedBase.Structured, unrelatedBase.StructuredBaseline = []Reference{generation}, nil
		unrelatedBaseRef := landClosure(unrelatedBase)
		unrelated := unrelatedBase
		unrelated.Structured, unrelated.StructuredBaseline = []Reference{}, &unrelatedBaseRef
		unrelatedRef := landClosure(unrelated)
		unrelatedDirect := unrelatedBase
		unrelatedDirect.Structured = []Reference{oldGeneration}
		unrelatedDirectRef := landClosure(unrelatedDirect)
		var sibling StructuredGeneration
		if err := readReference(root, generation, &sibling); err != nil {
			t.Fatal(err)
		}
		sibling.Provenance.GeneratedAt = "2026-10-31T02:00:00Z"
		sibling.Identity = ""
		sibling.Identity = identity(sibling)
		siblingRef := landSharedReattestationFixture(t, root, "data/locale-language/zh-CN/structured-generation/"+sibling.Identity+".json", sibling)
		badUnrelatedHash := unrelatedRef
		badUnrelatedHash.SHA256 = strings.Repeat("a", 64)
		const badJSONPath = "data/malformed-unrelated-closure.json"
		writeFixture(t, root, badJSONPath, []byte("{\n"))
		badUnrelatedJSON, err := ReferenceFile(root, badJSONPath)
		if err != nil {
			t.Fatal(err)
		}
		badRelevantHash := childRef
		badRelevantHash.SHA256 = strings.Repeat("a", 64)
		badBaseline := closure
		badBaseline.SHA256 = strings.Repeat("a", 64)
		badChild := child
		badChild.StructuredBaseline = &badBaseline
		badChildRef := landClosure(badChild)
		for _, tc := range []struct {
			name    string
			refs    []Reference
			parent  Reference
			path    string
			class   string
			allowed bool
		}{
			{"unrelated-first", []Reference{unrelatedRef, childRef}, generation, finding.ID, finding.Class, true},
			{"relevant-first", []Reference{childRef, unrelatedRef}, generation, finding.ID, finding.Class, true},
			{"no-relevant-package", []Reference{unrelatedRef}, generation, finding.ID, finding.Class, false},
			{"wrong-parent", []Reference{unrelatedRef, childRef}, oldGeneration, finding.ID, finding.Class, false},
			{"sibling-parent", []Reference{unrelatedRef, childRef}, siblingRef, finding.ID, finding.Class, false},
			{"wrong-path", []Reference{unrelatedRef, childRef}, generation, "_content/menus.yaml", finding.Class, false},
			{"wrong-class", []Reference{unrelatedRef, childRef}, generation, finding.ID, "surface-integration", false},
			{"unrelated-direct-parent", []Reference{unrelatedDirectRef, childRef}, oldGeneration, finding.ID, finding.Class, false},
			{"unrelated-hash-first", []Reference{badUnrelatedHash, childRef}, generation, finding.ID, finding.Class, false},
			{"unrelated-hash-last", []Reference{childRef, badUnrelatedHash}, generation, finding.ID, finding.Class, false},
			{"unrelated-malformed-first", []Reference{badUnrelatedJSON, childRef}, generation, finding.ID, finding.Class, false},
			{"unrelated-malformed-last", []Reference{childRef, badUnrelatedJSON}, generation, finding.ID, finding.Class, false},
			{"relevant-hash", []Reference{unrelatedRef, badRelevantHash}, generation, finding.ID, finding.Class, false},
			{"relevant-baseline", []Reference{unrelatedRef, badChildRef}, generation, finding.ID, finding.Class, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				b, _ := Encode(checked)
				var r IntegratedSurfaceReceipt
				if err := StrictJSON(b, &r); err != nil {
					t.Fatal(err)
				}
				r.Scope.Closures = tc.refs
				r.ExactFindings[0].Class = tc.class
				if tc.name == "relevant-baseline" {
					for _, assets := range [][]SurfaceAsset{r.Scope.Reviewed, r.Scope.Carried} {
						for i := range assets {
							assets[i].Evidence = []Reference{badChildRef}
						}
					}
				}
				r.Scope.Identity = ""
				r.Scope.Identity = identity(r.Scope)
				r.ReviewID = "multi-package-" + tc.name
				r.Identity = ""
				r.Identity = identity(r)
				ref := landSharedReattestationFixture(t, root, "data/locale-language/zh-CN/surface/"+r.ReviewID+".json", r)
				manifest, _ := Encode(StructuredBundle{RevisionFinding: &ref})
				archive, err := i18n.WriteDeterministicTransportBundle(manifest, nil)
				if err != nil {
					t.Fatal(err)
				}
				path := "data/" + r.ReviewID + ".zip"
				writeFixture(t, root, path, archive)
				g := &StructuredGeneration{Plan: StructuredPlan{Locale: "zh-CN"}, Bundle: Reference{Path: path, SHA256: digest(archive)}}
				if got := structuredSupersedes(root, g, tc.parent, tc.path); got != tc.allowed {
					t.Fatalf("supersession = %v, want %v", got, tc.allowed)
				}
			})
		}
	})
	for _, kind := range []string{"arbitrary-integration", "unknown", "ordinary-carried", "duplicate", "homepage-disguised", "structured-owner-TU-artifact", "finding-TU-artifact", "missing-anchor", "missing-closure-binding", "wrong-target", "scope-stale"} {
		t.Run(kind, func(t *testing.T) {
			var bad IntegratedSurfaceReceipt
			b, _ := Encode(receipt)
			if err := json.Unmarshal(b, &bad); err != nil {
				t.Fatal(err)
			}
			bad.Identity, bad.ReviewID = "", "reject-"+kind
			switch kind {
			case "arbitrary-integration":
				bad.ExactFindings[0].ID = "internal/site/runtime.go"
			case "unknown":
				bad.ExactFindings[0].ID = "unknown-owner"
			case "ordinary-carried":
				bad.ExactFindings[0].ID = "data/site-shell/translation.html"
			case "duplicate":
				bad.ExactFindings = append(bad.ExactFindings, bad.ExactFindings[0])
			case "homepage-disguised":
				bad.ExactFindings[0].ID = bad.Scope.Reviewed[0].ID
			case "structured-owner-TU-artifact", "missing-closure-binding", "wrong-target":
				for i := range bad.Scope.Carried {
					if bad.Scope.Carried[i].ID != finding.ID {
						continue
					}
					switch kind {
					case "structured-owner-TU-artifact":
						bad.Scope.Carried[i].CandidateArtifacts = bad.Scope.Reviewed[0].CandidateArtifacts
					case "missing-closure-binding":
						bad.Scope.Carried[i].Evidence = nil
					case "wrong-target":
						bad.Scope.Carried[i].TargetSHA = digest([]byte("wrong target"))
					}
				}
			case "finding-TU-artifact":
				bad.ExactFindings[0].Artifacts = bad.Scope.Reviewed[0].CandidateArtifacts
			case "missing-anchor":
				bad.Scope.Reviewed[0].State = "review-required"
			case "scope-stale":
				bad.Scope.GlossarySHA = digest([]byte("stale glossary"))
			}
			// Also test owner validation directly so forged scope changes cannot
			// pass solely because the recorder's current-scope comparison rejects.
			if kind != "scope-stale" {
				if _, err := reviewedFindingAssets(root, &bad); err == nil {
					t.Fatal("invalid owner accepted", kind)
				}
			}
			b, _ = Encode(bad)
			if _, _, err := recordIntegratedSurface(root, b, c, build); err == nil {
				t.Fatal("invalid Reviewer result recorded", kind)
			}
			if _, err := os.Stat(filepath.Join(root, "data/locale-language/zh-CN/surface", bad.ReviewID+".json")); !os.IsNotExist(err) {
				t.Fatal("rejected result published a receipt", err)
			}
		})
	}
	t.Run("ambiguous-plan-owner", func(t *testing.T) {
		p := clonePlan()
		collision := shared
		collision.TargetPath = "forbidden-second-output"
		p.Assets = append(p.Assets, collision)
		if err := selectStructuredSurfaceRevision(root, p, checked); err == nil {
			t.Fatal("ambiguous owner expanded exact revision")
		}
	})
	// surface-integration remains eligible on the fresh Page, without granting
	// any structured revision authority or converting it into a language finding.
	_, _ = surfaceReceipt(t, root, c, build, []Reference{closure}, "composition-runtime-failed", "failed", []SurfaceFinding{{ID: scope.Reviewed[0].ID, Class: "surface-integration", Finding: "Correct this bounded runtime composition."}})
	// Empty class preserves the legacy structured wire contract on the same
	// machine-owned structured asset; it never accepts the Homepage TU owner.
	legacy := *checked
	legacy.ExactFindings = []SurfaceFinding{finding}
	legacy.ExactFindings[0].Class = ""
	if err := selectStructuredSurfaceRevision(root, clonePlan(), &legacy); err != nil {
		t.Fatal("legacy structured class changed", err)
	}
	t.Log("synthetic Reviewer JSON -> recorder -> sealed finding reader -> one structured output; scope unchanged; arbitrary context/TU/ambiguous expansion rejected")
}

func compositionPageFixture(t *testing.T, root string, c *i18n.Catalog) *i18n.QualityCheckFinalization {
	t.Helper()
	e, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: c.TranslationScope, UnitKind: i18n.UnitKindPage})
	if err != nil {
		t.Fatal(err)
	}
	b, m, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	files, err := i18n.ReadTransportBundle(b, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	outputs := map[string][]byte{}
	for _, o := range m.ExpectedOutputs {
		raw := string(files["formal/inputs/"+filepath.Base(o.BundlePath)])
		// Synthetic fixture messages satisfy the current teaching-comment gate;
		// copying these English source comments verbatim into zh-CN would fail.
		for i, source := range []string{"You can edit this code!", "Click here and start typing."} {
			if !strings.Contains(raw, source) {
				t.Fatal("missing expected synthetic comment coordinate", i)
			}
			raw = strings.ReplaceAll(raw, source, "合成测试注释。")
		}
		outputs[o.UnitID] = []byte(raw)
	}
	surfaceImport(t, root, c, b, outputs)
	if r, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID}); err != nil || r.ValidationPassed != len(c.TranslationScope) {
		t.Fatal("synthetic Homepage validation", r, err)
	}
	const snapshot = "composition-page"
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: snapshot, ContentPackage: "site-v2-shell"}); err != nil {
		t.Fatal("bound synthetic Snapshot", err)
	}
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: snapshot, ReviewerSession: "synthetic-independent-reviewer", UnitIDs: c.TranslationScope, Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	f, _, err := i18n.FinalizeQualityCheck(root, c, i18n.QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	return f
}
