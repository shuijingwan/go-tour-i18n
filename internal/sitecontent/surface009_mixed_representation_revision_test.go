package sitecontent

import (
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const surface009MixedPath = "data/locale-language/zh-CN/surface/20261102-zh-CN-integrated-surface-009.json"
const surface009MixedSHA = "b85e406302a5c2fed6c5b950c6040d872042cb04bba3484a31aeb52bd7fef601"

// Isolate class routing and its early rejection boundaries. These in-memory
// values are not recorded judgments; the opt-in real test below validates the
// entire receipt, registered owner/lineage and exact predecessor scope.
func TestSurfaceMixedRepresentationFindingSelection(t *testing.T) {
	root := t.TempDir()
	artifact := i18n.SurfaceCandidateArtifact{SnapshotID: "synthetic-page"}
	r := IntegratedSurfaceReceipt{
		Scope: IntegratedSurfaceScope{Reviewed: []SurfaceAsset{
			{ID: "_content/page.html", Package: "page-package", CandidateArtifacts: []i18n.SurfaceCandidateArtifact{artifact}},
			{ID: "_content/chrome.tmpl", Package: "shell-package"},
			{ID: "data/prose.html", Package: "shell-package"},
		}},
		ExactFindings: []SurfaceFinding{
			{ID: "_content/page.html", Class: i18n.SurfaceTranslationUnitLanguage, Finding: "synthetic TU finding", Artifacts: []i18n.SurfaceCandidateArtifact{artifact}},
			{ID: "_content/chrome.tmpl", Class: "structured-surface-language", Finding: "synthetic shared finding"},
			{ID: "data/prose.html", Class: "structured-surface-language", Finding: "synthetic ordinary structured finding"},
		},
	}
	before, _ := Encode(r)
	plan := StructuredPlan{Assets: []StructuredAsset{
		{Package: "page-package", Document: &Document{Path: "_content/page.html"}, TargetPath: "forbidden-page-output"},
		{Package: "shell-package", Document: &Document{Path: "_content/chrome.tmpl"}, TargetPath: "selected-chrome"},
		{Package: "shell-package", Document: &Document{Path: "data/prose.html"}, TargetPath: "selected-prose"},
	}}
	if err := selectStructuredSurfaceRevision(root, &plan, &r); err != nil || !reflect.DeepEqual(plan.Expected, []string{"selected-chrome", "selected-prose"}) {
		t.Fatal("mixed 1 TU + 2 structured selection", plan.Expected, err)
	}
	after, _ := Encode(r)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("non-structured finding/artifacts were rewritten")
	}
	for _, name := range []string{"no-structured", "malformed-structured", "wrong-owner", "unknown-class"} {
		t.Run(name, func(t *testing.T) {
			var bad IntegratedSurfaceReceipt
			if err := StrictJSON(before, &bad); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "no-structured":
				bad.ExactFindings = bad.ExactFindings[:1]
			case "malformed-structured":
				bad.ExactFindings[1].Artifacts = []i18n.SurfaceCandidateArtifact{artifact}
			case "wrong-owner":
				bad.ExactFindings[1].ID = "not-reviewed"
			case "unknown-class":
				bad.ExactFindings[1].Class = "unknown-class"
			}
			if _, err := representationRevisionScope(root, &bad); err == nil {
				t.Fatal("foreign/malformed finding acquired representation authority")
			}
		})
	}
}

// Explicit opt-in, readonly real-case proof. No exporter, Bundle serialization,
// output landing, apply or record is invoked; the original judgment is retained.
func TestSurface009MixedRepresentationRevisionReadonly(t *testing.T) {
	if os.Getenv("GO_TOUR_SURFACE009_READONLY") != "1" {
		t.Skip("explicit Surface009 readonly planning/selection proof")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	ref := Reference{surface009MixedPath, surface009MixedSHA}
	var original IntegratedSurfaceReceipt
	if err := readReference(root, ref, &original); err != nil {
		t.Fatal(err)
	}
	checked, err := validatedSurfaceFinding(root, "zh-CN", ref)
	if err != nil {
		t.Fatal("full receipt + native TU + representation predecessor proof", err)
	}
	if !reflect.DeepEqual(checked, &original) {
		t.Fatal("finding classes/artifacts/scope were rewritten")
	}
	plan, err := PlanStructuredAssets(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if err := selectStructuredSurfaceRevision(root, plan, checked); err != nil {
		t.Fatal(err)
	}
	want := []string{"locales/zh-CN/structured/_content/site.tmpl", "locales/zh-CN/structured/data/site-shell/translation.html"}
	if !reflect.DeepEqual(plan.Expected, want) {
		t.Fatal("wrong exact structured outputs", plan.Expected)
	}
	sources := []string{}
	for _, a := range plan.Assets {
		if a.State == "generation-required" {
			if a.Document == nil || a.Package != "site-v2-shell" {
				t.Fatal("non-structured/wrong-package output", a)
			}
			sources = append(sources, a.Document.Path)
		}
	}
	sort.Strings(sources)
	if !reflect.DeepEqual(sources, []string{"_content/site.tmpl", "data/site-shell/translation.html"}) {
		t.Fatal("TU finding entered structured outputs", sources)
	}
	structured, nonstructured := 0, 0
	for _, f := range checked.ExactFindings {
		if f.Class == "structured-surface-language" {
			structured++
		} else {
			nonstructured++
		}
	}
	if structured != 2 || nonstructured != 1 {
		t.Fatal("real mixed finding set changed", structured, nonstructured)
	}
	t.Logf("REAL_SURFACE009_PROOF=PASS STRUCTURED_FINDINGS_SELECTED=%d NON_STRUCTURED_FINDINGS_PRESERVED=%d SOURCES=%v EXPECTED=%v", structured, nonstructured, sources, plan.Expected)

	for _, name := range []string{"malformed-structured", "wrong-owner", "wrong-package", "wrong-target", "unknown-class", "no-structured"} {
		t.Run(name, func(t *testing.T) {
			b, _ := Encode(original)
			var bad IntegratedSurfaceReceipt
			if err := StrictJSON(b, &bad); err != nil {
				t.Fatal(err)
			}
			for i := range bad.ExactFindings {
				if bad.ExactFindings[i].ID != "_content/site.tmpl" {
					continue
				}
				switch name {
				case "malformed-structured":
					bad.ExactFindings[i].Artifacts = []i18n.SurfaceCandidateArtifact{{}}
				case "wrong-owner":
					bad.ExactFindings[i].ID = "_content/not-a-reviewed-owner.tmpl"
				case "unknown-class":
					bad.ExactFindings[i].Class = "unknown-structured-class"
				}
			}
			for _, assets := range [][]SurfaceAsset{bad.Scope.Reviewed, bad.Scope.Carried} {
				for i := range assets {
					if assets[i].ID == "_content/site.tmpl" {
						if name == "wrong-package" {
							assets[i].Package = "other-package"
						}
						if name == "wrong-target" {
							assets[i].TargetSHA = strings.Repeat("a", 64)
						}
					}
				}
			}
			if name == "no-structured" {
				bad.ExactFindings = nil
				for _, f := range original.ExactFindings {
					if f.Class == i18n.SurfaceTranslationUnitLanguage {
						bad.ExactFindings = append(bad.ExactFindings, f)
					}
				}
			}
			bad.Scope.Identity = ""
			bad.Scope.Identity = identity(bad.Scope)
			bad.Identity = ""
			bad.Identity = identity(bad)
			if _, err := representationRevisionScope(root, &bad); err == nil {
				t.Fatal("malformed/misowned finding acquired representation authority")
			}
		})
	}
	var after IntegratedSurfaceReceipt
	if err := readReference(root, ref, &after); err != nil || !reflect.DeepEqual(after, original) {
		t.Fatal("real receipt bytes/identity changed", err)
	}
}
