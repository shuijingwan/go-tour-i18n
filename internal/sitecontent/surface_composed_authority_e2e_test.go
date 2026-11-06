package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

type e2eRejectingPage struct{ PageAdapter }

func (e2eRejectingPage) Validate([]byte, *i18n.Glossary) error {
	return fmt.Errorf("synthetic active validator rejects accepted candidate")
}

func TestIntegratedSurfaceComposedAuthorityE2ERegression(t *testing.T) {
	f := newSurfaceComposedAuthorityFixture(t, true)
	root, c, ids := f.root, f.current, f.ids
	if len(c.TranslationScope) != 51 || len(f.finals[0].Units) != 49 || len(f.finals[1].Units) != 2 {
		t.Fatal("fixture lost current/historical scope separation")
	}
	historicalA, err := i18n.HistoricalTranslationPackageCatalog(c, f.inputs[0].Package, f.inputs[0].PackageAuthority)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.VerifyQualityCheckFinalization(root, historicalA, "zh-CN", f.finals[0].SnapshotID); err == nil {
		t.Fatal("fixture did not exercise the historical/current glossary boundary before current authorization")
	}
	authorizations := []*i18n.CurrentTranslationAuthorization{}
	authorizationPaths := []string{}
	for i, in := range f.inputs {
		h, err := i18n.HistoricalTranslationPackageCatalog(c, in.Package, in.PackageAuthority)
		if err != nil {
			t.Fatal(err)
		}
		a, path, err := i18n.EstablishCurrentTranslationAuthorization(root, h, "zh-CN", f.finals[i].SnapshotID)
		if err != nil || a == nil || !a.Ready() || a.Glossary.SHA256 == a.Historical.Finalization.GlossarySHA256 {
			t.Fatal("historical A/current glossary authorization", a, err)
		}
		authorizations, authorizationPaths = append(authorizations, a), append(authorizationPaths, path)
	}
	const structuredID = "data/site-shell/translation.html"
	const structuredFinding = "Correct this exact structured message."
	const structuredTarget = "fixture-structured-target.html"
	build := func(root, locale string, refs []Reference) (*IntegratedSurfaceScope, error) {
		s, err := f.build(root, locale, refs)
		if err != nil {
			return nil, err
		}
		s.Reviewed = append(s.Reviewed, SurfaceAsset{ID: structuredID, SourcePath: structuredID, Package: "site-v2-shell", State: "review-required"})
		s.Identity = ""
		s.Identity = identity(*s)
		return s, nil
	}
	attachIntegratedSurfaceAuthority(c, build)
	findings, textsA := []SurfaceFinding{}, []string{}
	for _, u := range f.finals[0].Units[:5] {
		text := "Correct predecessor A exact defect: " + u.UnitID
		textsA = append(textsA, text)
		findings = append(findings, SurfaceFinding{ID: u.Snapshot.SourcePath, Finding: text, Class: i18n.SurfaceTranslationUnitLanguage, Artifacts: []i18n.SurfaceCandidateArtifact{{SnapshotID: f.finals[0].SnapshotID, Unit: u.Snapshot}}})
	}
	bUnit := f.finals[1].Units[0].Snapshot
	textB := "Correct predecessor B exact defect: " + bUnit.UnitID
	findings = append(findings,
		SurfaceFinding{ID: bUnit.SourcePath, Finding: textB, Class: i18n.SurfaceTranslationUnitLanguage, Artifacts: []i18n.SurfaceCandidateArtifact{{SnapshotID: f.finals[1].SnapshotID, Unit: bUnit}}},
		SurfaceFinding{ID: structuredID, Finding: structuredFinding, Class: "structured-surface-language"})
	r, ref := surfaceReceipt(t, root, c, build, []Reference{f.closureRef}, "mixed-composed-e2e-failed", "failed", findings)
	checked, err := validatedSurfaceFindingWithScope(root, "zh-CN", ref, c, build)
	if err != nil {
		t.Fatal("mixed receipt rejected before owner routing", err)
	}
	docs := []Document{}
	for _, id := range ids {
		u, err := c.Unit(id)
		if err != nil {
			t.Fatal(err)
		}
		docs = append(docs, u.Adapter.(PageAdapter).Document)
	}
	pageSelection, err := selectSurfaceRevisionPages(root, checked, c.TranslationPackages[0].ID, docs)
	wantPages := append(append([]string(nil), ids[:5]...), bUnit.UnitID)
	if err != nil || !reflect.DeepEqual(pageSelection, uniqueStrings(wantPages)) || len(checked.ExactFindings) != 7 {
		t.Fatal("mixed 6 TU/1 structured routing", pageSelection, err)
	}
	structuredPlan := func() *StructuredPlan {
		return &StructuredPlan{Assets: []StructuredAsset{
			{ID: structuredID, Package: "site-v2-shell", TargetPath: structuredTarget, Document: &Document{Path: structuredID}},
			{ID: "page-path-collision", Package: c.TranslationPackages[0].ID, TargetPath: "forbidden-page-target", Document: &docs[0]},
		}}
	}
	p := structuredPlan()
	if err := selectStructuredSurfaceRevision(root, p, checked); err != nil || !reflect.DeepEqual(p.Expected, []string{structuredTarget}) || p.Assets[1].State != "context-only" {
		t.Fatal("mixed structured owner selection", p.Expected, err)
	}
	pageOnly, structuredOnly := *checked, *checked
	pageOnly.ExactFindings, structuredOnly.ExactFindings = checked.ExactFindings[:6], checked.ExactFindings[6:]
	p = structuredPlan()
	if err := selectStructuredSurfaceRevision(root, p, &pageOnly); err != nil || len(p.Expected) != 0 {
		t.Fatal("Page finding authorized structured output", err)
	}
	if _, err := selectSurfaceRevisionPages(root, &structuredOnly, c.TranslationPackages[0].ID, docs); err == nil {
		t.Fatal("structured finding authorized Page output")
	}

	aOptions := i18n.QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: "e2e-predecessor-a", PreviousSnapshotID: f.finals[0].SnapshotID, SurfaceReviewID: r.ReviewID, Findings: textsA, UnitIDs: ids[:5], IntegratedSurface: true}
	bOptions := i18n.QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: "e2e-predecessor-b", PreviousSnapshotID: f.finals[1].SnapshotID, SurfaceReviewID: r.ReviewID, Finding: textB, UnitIDs: []string{bUnit.UnitID}, IntegratedSurface: true}
	rejectReopen := func(t *testing.T, o i18n.QualityCheckSurfaceReopenOptions) {
		t.Helper()
		o.ReopenID = "e2e-rejected"
		if _, _, err := i18n.RecordQualityCheckSurfaceReopen(root, c, o); err == nil {
			t.Fatal("invalid cross-owner/predecessor reopen accepted")
		}
		if _, err := os.Stat(filepath.Join(root, "data/quality-check-reopens/zh-CN", o.ReopenID+".json")); !os.IsNotExist(err) {
			t.Fatal("rejection published reopen", err)
		}
	}
	for _, name := range []string{"a-includes-b", "b-includes-a", "a-includes-structured", "structured-only", "duplicate", "wrong-snapshot"} {
		t.Run(name, func(t *testing.T) {
			o := aOptions
			o.UnitIDs = append([]string(nil), o.UnitIDs...)
			o.Findings = append([]string(nil), o.Findings...)
			switch name {
			case "a-includes-b":
				o.UnitIDs = append(o.UnitIDs, bUnit.UnitID)
			case "b-includes-a":
				o = bOptions
				o.UnitIDs = []string{bUnit.UnitID, ids[0]}
			case "a-includes-structured":
				o.Findings = append(o.Findings, structuredFinding)
			case "structured-only":
				o.Findings = []string{structuredFinding}
			case "duplicate":
				o.UnitIDs[4] = o.UnitIDs[0]
			case "wrong-snapshot":
				o.PreviousSnapshotID = bOptions.PreviousSnapshotID
			}
			rejectReopen(t, o)
		})
	}
	aReopen, aPath, err := i18n.RecordQualityCheckSurfaceReopen(root, c, aOptions)
	if err != nil || aReopen == nil || aReopen.UnitCount != 5 || len(aReopen.Findings) != 5 {
		t.Fatal("predecessor A aggregate reopen", aReopen, err)
	}
	for _, u := range aReopen.Units {
		if u.Snapshot.UnitKind != i18n.UnitKindPage || u.Snapshot.Index > 5 {
			t.Fatal("A reopen crossed exact historical boundary", u)
		}
	}
	exportOptions := func(o i18n.QualityCheckSurfaceReopenOptions, batch string) i18n.RetranslationExportOptions {
		return i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, BatchID: batch, UnitKind: i18n.UnitKindPage, PreviousSnapshotID: o.PreviousSnapshotID, SurfaceReopenID: o.ReopenID, AllowReexport: true, UnitIDs: append([]string(nil), o.UnitIDs...)}
	}
	aExport := exportOptions(aOptions, "chatgpt-zh-CN-060")
	rejectExport := func(t *testing.T, o i18n.RetranslationExportOptions) error {
		t.Helper()
		_, err := i18n.ExportRetranslationBatch(root, c, o)
		if err == nil {
			t.Fatal("invalid revision export accepted")
		}
		if _, e := os.Stat(filepath.Join(root, "data/retranslation-runs/zh-CN", o.BatchID)); !os.IsNotExist(e) {
			t.Fatal("rejection published batch", e)
		}
		return err
	}
	for _, name := range []string{"subset", "superset", "duplicate", "sibling", "wrong-snapshot"} {
		t.Run("export-"+name, func(t *testing.T) {
			o := exportOptions(aOptions, aExport.BatchID)
			switch name {
			case "subset":
				o.UnitIDs = o.UnitIDs[:4]
			case "superset":
				o.UnitIDs = ids[:6]
			case "duplicate":
				o.UnitIDs[4] = o.UnitIDs[0]
			case "sibling":
				o.UnitIDs[4] = bUnit.UnitID
			case "wrong-snapshot":
				o.PreviousSnapshotID = bOptions.PreviousSnapshotID
			}
			rejectExport(t, o)
		})
	}
	for name, path := range map[string]string{
		"membership": f.membershipRef.Path, "historical-finalization": f.inputs[0].Finalization.Path, "sibling-finalization": f.inputs[1].Finalization.Path,
		"candidate": f.finals[0].Units[0].Snapshot.CandidatePath, "validation": f.finals[0].Units[0].Snapshot.ValidationPath,
		"current-authorization": authorizationPaths[0], "current-review": authorizations[0].UnifiedReview.Path, "reopen": aPath,
	} {
		t.Run("tampered-"+name, func(t *testing.T) {
			before, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			writeFixture(t, root, path, append(bytes.Clone(before), []byte("tampered")...))
			defer writeFixture(t, root, path, before)
			rejectExport(t, aExport)
		})
	}
	for name, path := range map[string]string{"membership": PackageMembershipPath, "current-authorization": authorizations[0].UnifiedReview.Path} {
		t.Run("missing-"+name, func(t *testing.T) {
			before, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filepath.Join(root, path)); err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, path, before)
			rejectExport(t, aExport)
		})
	}
	t.Run("current-authorization-not-ready", func(t *testing.T) {
		oldAdapted, oldCurrent, oldHistorical := c.Adapted[0].Adapter, c.TranslationPackages[0].Units[0].Adapter, c.HistoricalTranslationPackages[0].Units[0].Adapter
		rejecting := e2eRejectingPage{oldAdapted.(PageAdapter)}
		c.Adapted[0].Adapter, c.TranslationPackages[0].Units[0].Adapter, c.HistoricalTranslationPackages[0].Units[0].Adapter = rejecting, rejecting, rejecting
		defer func() {
			c.Adapted[0].Adapter, c.TranslationPackages[0].Units[0].Adapter, c.HistoricalTranslationPackages[0].Units[0].Adapter = oldAdapted, oldCurrent, oldHistorical
		}()
		h, err := i18n.HistoricalTranslationPackageCatalog(c, f.inputs[0].Package, f.inputs[0].PackageAuthority)
		if err != nil {
			t.Fatal(err)
		}
		a, err := i18n.AssessCurrentTranslation(root, h, "zh-CN", aOptions.PreviousSnapshotID)
		if err != nil || a.Ready() || a.Affected != 1 {
			t.Fatal("fixture did not exercise current validator failure", a, err)
		}
		rejectExport(t, aExport)
	})
	t.Run("wrong-membership", func(t *testing.T) {
		bad := f.membership
		bad.Predecessors = append([]MembershipPredecessor(nil), bad.Predecessors...)
		bad.Predecessors[1].Authority = digest([]byte("wrong sibling authority"))
		bad.Identity = ""
		bad.Identity = identity(bad)
		b, _ := Encode(bad)
		badRef := Reference{Path: "data/content-package-memberships/receipts/" + bad.Identity + ".json", SHA256: digest(b)}
		writeFixture(t, root, badRef.Path, b)
		before, _ := readRegular(root, PackageMembershipPath)
		b, _ = Encode(badRef)
		writeFixture(t, root, PackageMembershipPath, b)
		defer writeFixture(t, root, PackageMembershipPath, before)
		rejectExport(t, aExport)
	})
	t.Run("wrong-sibling-predecessor", func(t *testing.T) {
		g, err := applyPackageMembership(root, f.base)
		if err != nil {
			t.Fatal(err)
		}
		bad := append([]i18n.TranslationFinalizationInput(nil), f.inputs...)
		bad[1].Finalization = bad[0].Finalization // Valid bytes, wrong scope.
		v := newClosureValidation(root)
		v.global = g
		if _, _, err := v.composePackage(root, c, "zh-CN", bad); err == nil {
			t.Fatal("wrong sibling finalization accepted")
		}
	})
	t.Run("surface-artifact-and-attempt", func(t *testing.T) {
		before, err := readRegular(root, ref.Path)
		if err != nil {
			t.Fatal(err)
		}
		var bad IntegratedSurfaceReceipt
		if err := json.Unmarshal(before, &bad); err != nil {
			t.Fatal(err)
		}
		bad.ExactFindings[0].Artifacts[0].Unit.Attempt++
		if err := i18n.ValidateHistoricalSurfaceCandidateArtifact(root, historicalA, "zh-CN", bad.ExactFindings[0].Artifacts[0]); err == nil {
			t.Fatal("wrong attempt accepted by native artifact verifier")
		}
		bad.Identity = ""
		bad.Identity = identity(bad)
		b, _ := Encode(bad)
		writeFixture(t, root, ref.Path, b)
		defer writeFixture(t, root, ref.Path, before)
		rejectExport(t, aExport)
	})
	assertExport := func(t *testing.T, o i18n.RetranslationExportOptions, want []string, expectedFinding map[string]string) {
		t.Helper()
		e, err := i18n.ExportRetranslationBatch(root, c, o)
		if err != nil || e == nil || e.UnitCount != len(want) || len(c.TranslationScope) != 51 {
			t.Fatal("exact historical revision export", e, err)
		}
		b, err := readRegular(root, e.BatchPath+"/manifest.json")
		if err != nil {
			t.Fatal(err)
		}
		var m i18n.RetranslationBatchManifest
		if err := StrictJSON(b, &m); err != nil {
			t.Fatal(err)
		}
		selected := []string{}
		for _, u := range m.Units {
			if u.UnitKind != i18n.UnitKindPage || u.PreviousSnapshotID != o.PreviousSnapshotID || u.RevisionFeedbackSource != "surface_review" || u.RevisionAuthorizationID != o.SurfaceReopenID || u.PreviousFinding != expectedFinding[u.UnitID] || u.PreviousFinding == structuredFinding {
				t.Fatal("cross-owner/snapshot revision feedback", u)
			}
			selected = append(selected, u.UnitID)
		}
		if m.UnitCount != len(want) || !reflect.DeepEqual(uniqueStrings(selected), uniqueStrings(want)) {
			t.Fatal("unexpected predecessor/sibling Unit exported", selected)
		}
	}
	feedbackA := map[string]string{}
	for i, id := range ids[:5] {
		feedbackA[id] = textsA[i]
	}
	assertExport(t, aExport, ids[:5], feedbackA)
	// A's export-only batch consumes only A. B still uses the same failed
	// receipt and complete 49+2 authority, without inheriting any A finding.
	bReopen, _, err := i18n.RecordQualityCheckSurfaceReopen(root, c, bOptions)
	if err != nil || bReopen == nil || bReopen.UnitCount != 1 || bReopen.Units[0].UnitID != bUnit.UnitID || bReopen.Units[0].Snapshot.UnitKind != i18n.UnitKindPage || bReopen.Finding != textB {
		t.Fatal("independent predecessor B reopen", bReopen, err)
	}
	bExport := exportOptions(bOptions, "chatgpt-zh-CN-061")
	assertExport(t, bExport, []string{bUnit.UnitID}, map[string]string{bUnit.UnitID: textB})
	for name, options := range map[string]i18n.RetranslationExportOptions{"a": aExport, "b": bExport} {
		t.Run("consumed-"+name, func(t *testing.T) { options.BatchID = "chatgpt-zh-CN-062"; rejectExport(t, options) })
	}
	conflicting := aOptions
	conflicting.ReopenID = "e2e-conflicting-successor"
	if _, _, err := i18n.RecordQualityCheckSurfaceReopen(root, c, conflicting); err != nil {
		t.Fatal(err)
	}
	rejectExport(t, exportOptions(conflicting, "chatgpt-zh-CN-062"))
	// The structured finding remains separately selectable after both TU
	// exports. This calls only selection, never structured export/publication.
	p = structuredPlan()
	if err := selectStructuredSurfaceRevision(root, p, checked); err != nil || !reflect.DeepEqual(p.Expected, []string{structuredTarget}) {
		t.Fatal("TU export consumed structured authority", err)
	}
	t.Log("current=51; predecessors=49+2; mixed=6 TU+1 structured; A reopen/export=5; B reopen/export=1; A other44/B other1 excluded; structured isolated; consumed/conflicts rejected")
}
