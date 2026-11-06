package sitecontent

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestSurfaceMixedFindingRouting(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, c, ids := surfaceFixture(t, "learn-docs", 6)
	f := surfaceStart(t, root, c, ids)
	pkg := c.TranslationPackages[0].ID
	const structuredPath = "data/site-shell/translation.html"
	scope := IntegratedSurfaceScope{Schema: IntegratedSurfaceSchema, Locale: "zh-CN", Carried: []SurfaceAsset{{ID: "carried-only", Package: pkg}}}
	findings := []SurfaceFinding{}
	docs := []Document{}
	for _, u := range f.Units {
		a := i18n.SurfaceCandidateArtifact{SnapshotID: f.SnapshotID, Unit: u.Snapshot}
		scope.Reviewed = append(scope.Reviewed, SurfaceAsset{ID: u.Snapshot.SourcePath, SourcePath: u.Snapshot.SourcePath, Package: pkg, CandidateArtifacts: []i18n.SurfaceCandidateArtifact{a}})
		findings = append(findings, SurfaceFinding{ID: u.Snapshot.SourcePath, Finding: "Correct this exact Page message.", Class: i18n.SurfaceTranslationUnitLanguage, Artifacts: []i18n.SurfaceCandidateArtifact{a}})
		unit, err := c.Unit(u.UnitID)
		if err != nil {
			t.Fatal(err)
		}
		docs = append(docs, unit.Adapter.(PageAdapter).Document)
	}
	scope.Reviewed = append(scope.Reviewed, SurfaceAsset{ID: structuredPath, Package: "site-v2-shell"})
	scope.Identity = identity(scope)
	findings = append(findings, SurfaceFinding{ID: structuredPath, Finding: "Correct the exact structured message.", Class: "structured-surface-language"})
	build := func(string, string, []Reference) (*IntegratedSurfaceScope, error) { copy := scope; return &copy, nil }
	r, ref := surfaceReceipt(t, root, c, build, nil, "mixed-routing", "failed", findings)
	original, err := readRegular(root, ref.Path)
	if err != nil {
		t.Fatal(err)
	}
	validate := func(ref Reference) (*IntegratedSurfaceReceipt, error) {
		return validatedSurfaceFindingWithScope(root, "zh-CN", ref, c, build)
	}
	plan := func() *StructuredPlan {
		return &StructuredPlan{Assets: []StructuredAsset{
			{ID: structuredPath, Package: "site-v2-shell", TargetPath: "structured-target.html", Document: &Document{Path: structuredPath}},
			// Deliberate path collisions cannot cross either class or owner.
			{ID: "page-collision", Package: pkg, TargetPath: "forbidden-page-target", Document: &docs[0]},
			{ID: "foreign-collision", Package: "other-package", TargetPath: "forbidden-foreign-target", Document: &Document{Path: structuredPath}},
		}}
	}
	checked, err := validate(ref)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := selectSurfaceRevisionPages(root, checked, pkg, docs)
	if err != nil || !reflect.DeepEqual(selected, uniqueStrings(ids)) {
		t.Fatal("mixed receipt did not route exact six Pages", selected, err)
	}
	p := plan()
	if err := selectStructuredSurfaceRevision(root, p, checked); err != nil || !reflect.DeepEqual(p.Expected, []string{"structured-target.html"}) {
		t.Fatal("mixed receipt did not route exact structured owner", p.Expected, err)
	}
	for _, kind := range []string{"outside-scope", "duplicate", "carried", "artifact", "missing-finding", "decision", "reviewer", "markdown"} {
		t.Run(kind, func(t *testing.T) {
			var bad IntegratedSurfaceReceipt
			if err := json.Unmarshal(original, &bad); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "outside-scope":
				bad.ExactFindings[0].ID = "not-reviewed"
			case "duplicate":
				bad.ExactFindings = append(bad.ExactFindings, bad.ExactFindings[0])
			case "carried":
				bad.ExactFindings[0].ID = "carried-only"
			case "artifact":
				bad.ExactFindings[0].Artifacts[0].Unit.CandidateSHA256 = digest([]byte("wrong candidate"))
			case "missing-finding":
				bad.ExactFindings[0].Finding = ""
			case "decision":
				bad.Decision = "passed"
			case "reviewer":
				bad.Reviewer = "invalid/session"
			case "markdown":
				bad.Evidence.SHA256 = digest([]byte("wrong evidence"))
			}
			bad.Identity = ""
			bad.Identity = identity(bad)
			b, _ := Encode(bad)
			writeFixture(t, root, ref.Path, b)
			defer writeFixture(t, root, ref.Path, original)
			if _, err := validate(Reference{Path: ref.Path, SHA256: digest(b)}); err == nil {
				t.Fatal("global receipt validation weakened", kind)
			}
		})
	}
	pageOnly := *r
	pageOnly.ExactFindings = append([]SurfaceFinding{}, r.ExactFindings[:6]...)
	p = plan()
	if err := selectStructuredSurfaceRevision(root, p, &pageOnly); err != nil || len(p.Expected) != 0 {
		t.Fatal("TU finding authorized structured replacement", err, p.Expected)
	}
	structuredOnly := *r
	structuredOnly.ExactFindings = append([]SurfaceFinding{}, r.ExactFindings[6:]...)
	if _, err := selectSurfaceRevisionPages(root, &structuredOnly, pkg, docs); err == nil {
		t.Fatal("foreign structured finding authorized Page replacement")
	}
	wrongClass := *r
	wrongClass.ExactFindings = append([]SurfaceFinding{}, r.ExactFindings...)
	wrongClass.ExactFindings[0].Class, wrongClass.ExactFindings[0].Artifacts = "structured-surface-language", nil
	if _, err := selectSurfaceRevisionPages(root, &wrongClass, pkg, docs); err == nil {
		t.Fatal("same-package Page class mismatch accepted")
	}
	if _, err := selectSurfaceRevisionPages(root, r, pkg, docs[1:]); err == nil {
		t.Fatal("missing Page Document silently omitted")
	}
	if _, err := selectSurfaceRevisionPages(root, r, pkg, append(append([]Document{}, docs...), docs[0])); err == nil {
		t.Fatal("ambiguous Page Document accepted")
	}
	unknownClass := *r
	unknownClass.ExactFindings = append([]SurfaceFinding{}, r.ExactFindings...)
	unknownClass.ExactFindings[6].Class = "unknown-class"
	if err := selectStructuredSurfaceRevision(root, plan(), &unknownClass); err == nil {
		t.Fatal("unknown structured class silently authorized")
	}
	// Exact native candidate bytes are checked even when the current consumer
	// would route that finding to a different owner.
	path := f.Units[0].Snapshot.CandidatePath
	before, err := readRegular(root, path)
	if err != nil {
		t.Fatal(err)
	}
	defer writeFixture(t, root, path, before)
	writeFixture(t, root, path, append(bytes.Clone(before), ' '))
	if _, err := validate(ref); err == nil {
		t.Fatal("native candidate artifact tamper accepted")
	}
	t.Log("mixed 6 TU + 1 structured: exact owner/class routing; global invalid scope/duplicate/artifact rejection retained")
}
