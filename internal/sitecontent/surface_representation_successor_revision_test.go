package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const surface008FindingPath = "data/locale-language/zh-CN/surface/20261031-zh-CN-integrated-surface-008.json"
const surface008FindingSHA = "b525bc49d7e50625c245c647c39dd666c2ca3cdd64974190047c28b0c700b18f"

type scopeStructuralDifference struct {
	Field   string `json:"field"`
	Old     any    `json:"old"`
	Current any    `json:"current"`
}

// The positive path uses the actual recorded-finding reader and exporter, not
// an injected scope builder or a test-only narrowing of Plan.Expected.
func TestSurfaceRepresentationSuccessorRevisionAuthority(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, closure, generation, _ := sharedRepresentationFixtureWithHomepage(t, true)
	scope, err := BuildIntegratedSurfaceScope(root, sharedFixtureLocale, []Reference{closure})
	if err != nil {
		t.Fatal(err)
	}
	md := []byte("# Independent structured representation finding\nNatural language action is frozen as a machine atom.\n")
	evidence := "data/failed-shared-finding.md"
	writeFixture(t, root, evidence, md)
	r := IntegratedSurfaceReceipt{Schema: IntegratedSurfaceSchema, Scope: *scope, ReviewID: "failed-shared-finding", Reviewer: "synthetic-independent-reviewer", Decision: "failed", Findings: []string{"natural language atom"}, ExactFindings: []SurfaceFinding{{ID: "_content/site.tmpl", Class: "structured-surface-language", Finding: "natural language atom"}}, Evidence: Reference{evidence, digest(md)}}
	b, _ := Encode(r)
	sealed, path, err := RecordIntegratedSurface(root, b)
	if err != nil {
		t.Fatal(err)
	}
	finding, _ := ReferenceFile(root, path)
	frozen, _ := readRegular(root, path)
	raw, _ := readRegular(root, "_content/site.tmpl")
	old, err := ExtractShellLanguageSource("_content/site.tmpl", raw)
	if err != nil {
		t.Fatal(err)
	}
	q := SharedRepresentationRequest{Schema: SharedRepresentationSchema + "/request", Package: "site-v2-shell", AssetID: "source:_content/site.tmpl", Path: "_content/site.tmpl", SourceSHA: digest(raw), PreviousIdentity: identity(*old), Generation: generation, Rule: SharedRepresentationRule, Slots: []string{"0719e14f5f889a9c0dccb5eb291c9fe1de9604b6f5423b3c1497204c4276a85d"}}
	request := landSharedReattestationFixture(t, root, "data/shared-request.json", q)
	p, err := PlanSharedStructuredRepresentation(root, request)
	if err != nil {
		t.Fatal(err)
	}
	input := landSharedReattestationFixture(t, root, "data/shared-plan.json", p)
	if _, err := ApplySharedStructuredRepresentation(root, input, p.PriorIdentity, p.PriorPointerSHA); err != nil {
		t.Fatal(err)
	}
	parser := sharedRepresentationParserPaths[0]
	code := mustSharedFixtureBytes(t, root, parser)
	writeFixture(t, root, parser, append(bytes.Clone(code), []byte("\n// audited currentness dependency epoch\n")...))
	if _, err := validatedSurfaceFinding(root, sharedFixtureLocale, finding); err == nil {
		t.Fatal("stale dependencies accepted")
	}
	rea, err := PlanSharedRepresentationReattestation(root, p.Receipt)
	if err != nil {
		t.Fatal(err)
	}
	reviewed := landSharedReattestationFixture(t, root, "data/shared-dependency-plan.json", rea)
	if _, err := InspectSharedRepresentationReattestation(root, reviewed); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplySharedRepresentationReattestation(root, reviewed, rea.PriorPointerSHA); err != nil {
		t.Fatal(err)
	}
	if current, err := BuildIntegratedSurfaceScope(root, sharedFixtureLocale, sealed.Scope.Closures); err == nil || reflect.DeepEqual(&sealed.Scope, current) {
		t.Fatal("representation change silently became a current Surface scope")
	}
	if current, err := validatedSurfaceFinding(root, sharedFixtureLocale, finding); err != nil || !reflect.DeepEqual(current, sealed) {
		t.Fatal("full finding validation", err)
	}
	_, bundle, err := ExportStructuredAssetsRevision(root, sharedFixtureLocale, finding)
	if err != nil {
		t.Fatal("full revision exporter", err)
	}
	want := []string{"locales/" + sharedFixtureLocale + "/structured/_content/site.tmpl"}
	if !reflect.DeepEqual(bundle.Plan.Expected, want) {
		t.Fatal("revision expanded", bundle.Plan.Expected)
	}
	for _, a := range bundle.Plan.Assets {
		if a.ID != q.AssetID {
			continue
		}
		if a.State != "generation-required" || a.Document == nil {
			t.Fatal("old language acceptance carried", a.State)
		}
		for _, u := range a.Document.Units {
			for _, token := range u.Protected {
				if token.Raw == "activate/deactivate" {
					t.Fatal("released natural language still frozen")
				}
			}
		}
	}
	if current := mustSharedFixtureBytes(t, root, path); !bytes.Equal(current, frozen) {
		t.Fatal("historical judgment changed")
	}
	if _, _, _, err := historicalStructuredGeneration(root, generation); err != nil {
		t.Fatal("history changed", err)
	}
	t.Log("HISTORICAL_SCOPE_EQUAL_CURRENT=no SUCCESSOR_AUTHORIZATION=PASS FULL_VALIDATED_SURFACE_FINDING=PASS REVISION_EXPECTED_COUNT=1")

	// Forged but internally sealed receipts must still fail at the exact owner,
	// predecessor/scope proof. No mock currentness builder is involved.
	for _, name := range []string{"package", "path", "target", "closure", "glossary", "review", "unrelated", "class", "id", "tu", "tu-artifacts", "duplicate"} {
		t.Run(name, func(t *testing.T) {
			var bad IntegratedSurfaceReceipt
			_ = StrictJSON(frozen, &bad)
			mutate := func(fn func(*SurfaceAsset)) {
				for i := range bad.Scope.Carried {
					if bad.Scope.Carried[i].ID == q.Path {
						fn(&bad.Scope.Carried[i])
					}
				}
				for i := range bad.Scope.Reviewed {
					if bad.Scope.Reviewed[i].ID == q.Path {
						fn(&bad.Scope.Reviewed[i])
					}
				}
			}
			switch name {
			case "package":
				mutate(func(a *SurfaceAsset) { a.Package = "other-package" })
			case "path":
				mutate(func(a *SurfaceAsset) { a.ID = "_content/menus.yaml" })
			case "target":
				mutate(func(a *SurfaceAsset) { a.TargetSHA = strings.Repeat("a", 64) })
			case "closure":
				bad.Scope.Closures[0].SHA256 = strings.Repeat("a", 64)
			case "glossary":
				bad.Scope.GlossarySHA = strings.Repeat("a", 64)
			case "review":
				bad.Scope.UnifiedReview.SHA256 = strings.Repeat("a", 64)
			case "unrelated":
				bad.Scope.Routes = append(bad.Scope.Routes, "/unrelated/")
			case "class":
				bad.ExactFindings[0].Class = "surface-integration"
			case "id":
				bad.ExactFindings[0].ID = "_content/unknown.yaml"
			case "tu":
				mutate(func(a *SurfaceAsset) { a.UnitID = "injected-tu" })
			case "tu-artifacts":
				mutate(func(a *SurfaceAsset) { a.CandidateArtifacts = []i18n.SurfaceCandidateArtifact{{}} })
			case "duplicate":
				bad.Scope.Reviewed = append(bad.Scope.Reviewed, ownersForTest(t, root, sealed)[q.Path])
			}
			bad.Scope.Identity = ""
			bad.Scope.Identity = identity(bad.Scope)
			bad.Identity = ""
			bad.Identity = identity(bad)
			ref := landSharedReattestationFixture(t, root, path, bad)
			defer writeFixture(t, root, path, frozen)
			if _, err := validatedSurfaceFinding(root, sharedFixtureLocale, ref); err == nil {
				t.Fatal("unrelated/forged authority accepted")
			}
		})
	}
	for _, name := range []string{"no-successor", "wrong-successor", "raw", "target-bytes", "missing-reattestation", "stale-reattestation", "released"} {
		t.Run(name, func(t *testing.T) {
			file := p.CurrentPath
			value := []byte("{}\n")
			switch name {
			case "raw":
				file = q.Path
				value = append(bytes.Clone(raw), []byte("changed")...)
			case "target-bytes":
				file = want[0]
				value = []byte("unauthorized target")
			case "missing-reattestation":
				file = rea.CurrentPath
			case "stale-reattestation":
				file = parser
				value = append(mustSharedFixtureBytes(t, root, parser), []byte("\n// drift\n")...)
			case "released":
				file = p.Receipt.Path
				var bad SharedRepresentationSuccessor
				_ = readReference(root, p.Receipt, &bad)
				bad.Released[0].Token.Raw = "forged"
				value, _ = Encode(bad)
			case "wrong-successor":
				value, _ = Encode(Reference{Path: "data/site-shell/representation-successors/current.json", SHA256: strings.Repeat("a", 64)})
			}
			before, readErr := readRegular(root, file)
			if readErr != nil && !os.IsNotExist(readErr) {
				t.Fatal(readErr)
			}
			if os.IsNotExist(readErr) {
				defer os.Remove(pathJoin(root, file))
			} else {
				defer writeFixture(t, root, file, before)
			}
			candidate := finding
			if name == "no-successor" {
				// An unchanged pre-adoption scope is still a valid ordinary
				// finding. It must not rescue a genuinely stale scope without
				// the formal lineage required by this extension.
				bad := *sealed
				bad.Scope.Routes = append(append([]string{}, sealed.Scope.Routes...), "/unrelated/")
				bad.Scope.Identity = ""
				bad.Scope.Identity = identity(bad.Scope)
				bad.Identity = ""
				bad.Identity = identity(bad)
				candidate = landSharedReattestationFixture(t, root, path, bad)
				defer writeFixture(t, root, path, frozen)
			}
			if name == "no-successor" || name == "missing-reattestation" {
				if err := os.Remove(pathJoin(root, file)); err != nil {
					t.Fatal(err)
				}
			} else {
				writeFixture(t, root, file, value)
			}
			if _, err := validatedSurfaceFinding(root, sharedFixtureLocale, candidate); err == nil {
				t.Fatal("invalid authority accepted")
			}
		})
	}
}

func ownersForTest(t *testing.T, root string, r *IntegratedSurfaceReceipt) map[string]SurfaceAsset {
	t.Helper()
	a, err := reviewedFindingAssets(root, r)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func exactScopeDifferences(old, current any, path string) []scopeStructuralDifference {
	if reflect.DeepEqual(old, current) {
		return nil
	}
	if a, ok := old.(map[string]any); ok {
		if b, ok := current.(map[string]any); ok {
			keys := map[string]bool{}
			for k := range a {
				keys[k] = true
			}
			for k := range b {
				keys[k] = true
			}
			ordered := []string{}
			for k := range keys {
				ordered = append(ordered, k)
			}
			sort.Strings(ordered)
			out := []scopeStructuralDifference{}
			for _, k := range ordered {
				out = append(out, exactScopeDifferences(a[k], b[k], path+"."+k)...)
			}
			return out
		}
	}
	if a, ok := old.([]any); ok {
		if b, ok := current.([]any); ok && len(a) == len(b) {
			out := []scopeStructuralDifference{}
			for i := range a {
				out = append(out, exactScopeDifferences(a[i], b[i], fmt.Sprintf("%s[%d]", path, i))...)
			}
			return out
		}
	}
	return []scopeStructuralDifference{{path, old, current}}
}

func TestSurface008RepresentationSuccessorScopeDiagnosis(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	b, err := readRegular(root, surface008FindingPath)
	if os.IsNotExist(err) {
		t.Skip("retained Surface evidence unavailable")
	}
	if err != nil || digest(b) != surface008FindingSHA {
		t.Fatal("finding changed", err)
	}
	var receipt IntegratedSurfaceReceipt
	if err := StrictJSON(b, &receipt); err != nil {
		t.Fatal(err)
	}
	current, buildErr := BuildIntegratedSurfaceScope(root, "zh-CN", receipt.Scope.Closures)
	report := map[string]any{"frozen_scope": receipt.Scope, "current_scope": current, "scope_equal": reflect.DeepEqual(&receipt.Scope, current)}
	if buildErr != nil {
		report["current_build_error"] = buildErr.Error()
	}
	if current != nil {
		ob, _ := json.Marshal(receipt.Scope)
		cb, _ := json.Marshal(current)
		var old, next any
		_ = json.Unmarshal(ob, &old)
		_ = json.Unmarshal(cb, &next)
		report["structural_diff"] = exactScopeDifferences(old, next, "scope")
		report["closures_equal"] = reflect.DeepEqual(receipt.Scope.Closures, current.Closures)
		report["glossary_equal"] = receipt.Scope.GlossarySHA == current.GlossarySHA
		report["unified_review_equal"] = receipt.Scope.UnifiedReview == current.UnifiedReview
		report["existing_coverage_equal"] = reflect.DeepEqual(receipt.Scope.ExistingCoverage, current.ExistingCoverage)
		report["reviewed_set_equal"] = reflect.DeepEqual(receipt.Scope.Reviewed, current.Reviewed)
		report["carried_set_equal"] = reflect.DeepEqual(receipt.Scope.Carried, current.Carried)
	}
	for _, set := range []struct {
		name   string
		assets []SurfaceAsset
	}{{"old", append(append([]SurfaceAsset{}, receipt.Scope.Carried...), receipt.Scope.Reviewed...)}} {
		for _, asset := range set.assets {
			if asset.ID == "_content/site.tmpl" {
				report[set.name+"_sitetmpl"] = asset
			}
		}
	}
	if current != nil {
		for _, asset := range append(append([]SurfaceAsset{}, current.Carried...), current.Reviewed...) {
			if asset.ID == "_content/site.tmpl" {
				report["current_sitetmpl"] = asset
			}
		}
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("/tmp/surface008-representation-successor-scope-diff.json", append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Log("SCOPE_DIFF_EVIDENCE=/tmp/surface008-representation-successor-scope-diff.json", "CURRENT_BUILD_ERROR=", buildErr)
}

// Extend the saved ordinary-build diagnosis without repeating its expensive
// failed current scope build. A predecessor projection is explicitly labelled;
// it does not manufacture a current scope after the ordinary gate failed.
func TestSurface008RepresentationSuccessorRevisionDiagnosis(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	var r IntegratedSurfaceReceipt
	if err := readReference(root, Reference{surface008FindingPath, surface008FindingSHA}, &r); err != nil {
		if os.IsNotExist(err) {
			t.Skip("retained Surface evidence unavailable")
		}
		t.Fatal(err)
	}
	projection, err := representationRevisionScope(root, &r)
	if err != nil {
		t.Fatal("exact predecessor proof", err)
	}
	var report map[string]any
	data, err := os.ReadFile("/tmp/surface008-representation-successor-scope-diff.json")
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("bounded ordinary-scope diagnosis has not been run")
		}
		t.Fatal("run ordinary diagnosis first", err)
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report["current_build_error"] != "missing/duplicate affected delta: _content/site.tmpl" {
		t.Fatal("unexpected ordinary-build evidence", report["current_build_error"])
	}
	report["revision_predecessor_projection"] = projection
	report["projection_kind"] = "revision-only exact predecessor proof; not current Surface eligibility"
	checks := map[string]bool{"closures_equal": reflect.DeepEqual(r.Scope.Closures, projection.Closures), "glossary_equal": r.Scope.GlossarySHA == projection.GlossarySHA, "unified_review_equal": r.Scope.UnifiedReview == projection.UnifiedReview, "existing_coverage_equal": reflect.DeepEqual(r.Scope.ExistingCoverage, projection.ExistingCoverage), "reviewed_set_equal": reflect.DeepEqual(r.Scope.Reviewed, projection.Reviewed), "carried_set_equal": reflect.DeepEqual(r.Scope.Carried, projection.Carried)}
	report["predecessor_checks"] = checks
	ob, _ := json.Marshal(r.Scope)
	cb, _ := json.Marshal(projection)
	var a, b any
	_ = json.Unmarshal(ob, &a)
	_ = json.Unmarshal(cb, &b)
	report["predecessor_structural_diff"] = exactScopeDifferences(a, b, "scope")
	plan, err := PlanStructuredAssets(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	report["global_expected"] = plan.Expected
	var current StructuredAsset
	for _, a := range plan.Assets {
		if a.ID == "source:_content/site.tmpl" {
			current = a
		}
	}
	if current.Document == nil {
		t.Fatal("current Document missing")
	}
	report["current_structured_sitetmpl"] = current
	owners := ownersForTest(t, root, &r)
	owner := owners["_content/site.tmpl"]
	for _, ref := range r.Scope.Closures {
		var c PackageClosure
		if err := readReference(root, ref, &c); err != nil {
			t.Fatal(err)
		}
		if c.Package != owner.Package {
			continue
		}
		gen, err := compositionStructuredOwner(root, ref, "zh-CN", owner, map[Reference]bool{})
		if err != nil {
			t.Fatal(err)
		}
		h, files, _, err := historicalStructuredGeneration(root, gen)
		if err != nil {
			t.Fatal(err)
		}
		for _, a := range h.Plan.Assets {
			if a.ID == current.ID {
				report["old_structured_sitetmpl"] = a
				oldJSON, _ := json.Marshal(a.Document)
				newJSON, _ := json.Marshal(current.Document)
				var oldDoc, newDoc any
				_ = json.Unmarshal(oldJSON, &oldDoc)
				_ = json.Unmarshal(newJSON, &newDoc)
				report["exact_document_diff"] = exactScopeDifferences(oldDoc, newDoc, "document")
			}
		}
		report["old_sitetmpl_generation"] = gen
		report["current_sitetmpl_target_sha256"] = digest(files[current.TargetPath])
		report["target_predecessor_exact"] = owner.TargetSHA == digest(files[current.TargetPath])
		report["predecessor_target_authority"] = "frozen accepted Generation selected by exact reviewed closure; TargetPath is virtual"
	}
	data, err = json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("/tmp/surface008-representation-successor-scope-diff.json", append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Log("PREDECESSOR_SCOPE_EXACT=yes GLOBAL_EXPECTED_COUNT=", len(plan.Expected))
}

// A published dependency authority is not silently widened by this fix. This
// readonly probe is intentionally separate from all isolated publications.
func TestSurface008RevisionInstalledDependencyGate(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	ref := Reference{"data/site-shell/shared-representation-reattestations/receipts/4a85573f41957442729cdbc6e645db4c0814690180d6f82e59d0197dd8509b09.json", "ac536845cbfa75ba52718d431e6d832312fe9f862ba2b7cd0d2990524de1d9ed"}
	var r SharedRepresentationReattestation
	if err := readReference(root, ref, &r); err != nil {
		if os.IsNotExist(err) {
			t.Skip("retained dependency evidence unavailable")
		}
		t.Fatal(err)
	}
	current, err := readRegular(root, "internal/sitecontent/structured.go")
	if err != nil {
		t.Fatal(err)
	}
	recorded := ""
	for _, input := range r.ValidationInputs {
		if input.Reference.Path == "internal/sitecontent/structured.go" {
			recorded = input.Reference.SHA256
		}
	}
	if recorded == digest(current) {
		t.Skip("exporter dependency unchanged")
	}
	var head Reference
	if err := StrictJSON(mustSharedFixtureBytes(t, root, sharedReattestationPath(r.BaseIdentity)), &head); err != nil {
		t.Fatal(err)
	}
	if head != ref {
		t.Skip("a later explicit dependency authority has been adopted")
	}
	_, readErr := PlanStructuredAssets(root, "zh-CN")
	if readErr == nil || !strings.Contains(readErr.Error(), "shared dependency re-attestation stale") {
		t.Fatal("new implementation silently accepted by old authority", readErr)
	}
	report := map[string]any{"dependency": "internal/sitecontent/structured.go", "recorded_sha256": recorded, "current_sha256": digest(current), "installed_reattestation": ref, "current_reader_error": readErr.Error(), "real_export_allowed": false, "required_authority": "existing shared dependency re-attestation, new first-real-use gate; no automatic mutation"}
	b, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile("/tmp/surface008-successor-exporter-dependency-gate.json", append(b, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Log("CURRENT_READER_FAIL_CLOSED=PASS", readErr)
}
