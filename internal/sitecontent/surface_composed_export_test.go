package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

type surfaceComposedAuthorityFixture struct {
	root          string
	current       *i18n.Catalog
	ids           []string
	base          *Global
	membership    PackageMembership
	membershipRef Reference
	closureRef    Reference
	finals        []*i18n.QualityCheckFinalization
	inputs        []i18n.TranslationFinalizationInput
	build         integratedScopeBuilder
}

func newSurfaceComposedAuthorityFixture(t *testing.T, historicalGlossary bool) surfaceComposedAuthorityFixture {
	t.Helper()
	// Synthetic Page package, real membership projection/composition and shared
	// Surface/reopen/export readers. Only the source graph loader is injected.
	root, current, ids := surfaceFixture(t, "learn-docs", 51)
	base, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	merged := current.TranslationPackages[0]
	old, sibling := merged, merged
	old.Units = append([]i18n.AdaptedTranslationUnit(nil), merged.Units[:49]...)
	sibling.ID, sibling.Units = "test-docs-entry", append([]i18n.AdaptedTranslationUnit(nil), merged.Units[49:]...)
	predecessors := []*i18n.TranslationPackageDescriptor{&old, &sibling}
	m := PackageMembership{Schema: PackageMembershipSchema, Package: merged.ID}
	registry := PageRegistry{Schema: PageRegistrySchema}
	for _, d := range predecessors {
		p := Package{ID: d.ID, Activation: "atomic-locale-package", ParserContract: UnitContract}
		surface := "surface-" + d.ID
		p.Surfaces = []string{surface}
		for _, u := range d.Units {
			p.Routes = append(p.Routes, u.Route)
		}
		p.Identity = identity(p)
		d.AuthoritySHA256 = p.Identity
		base.Packages = append(base.Packages, p)
		base.Surfaces = append(base.Surfaces, Surface{ID: surface, Package: p.ID})
		base.Versions[1].Packages = append(base.Versions[1].Packages, p.ID)
		pred := MembershipPredecessor{Package: p.ID, Authority: p.Identity}
		for _, u := range d.Units {
			pred.Sources = append(pred.Sources, Reference{Path: u.Unit.SourcePath, SHA256: u.Unit.SourceSHA256})
			base.Sources = append(base.Sources, Source{Path: u.Unit.SourcePath, Kind: "page", Origin: "repository", Package: p.ID, Surface: surface, Route: u.Route, SHA256: u.Unit.SourceSHA256})
			registry.Pages = append(registry.Pages, StablePage{Index: len(registry.Pages) + 1, Path: u.Unit.SourcePath, Surface: surface})
		}
		m.Predecessors = append(m.Predecessors, pred)
		if len(registry.Pages) == 49 {
			sealPages(&registry)
			b, _ := Encode(registry)
			m.PreviousRegistry = Reference{Path: "data/content-package-memberships/registries/test-old.json", SHA256: digest(b)}
			writeFixture(t, root, m.PreviousRegistry.Path, b)
		}
	}
	sealPages(&registry)
	b, _ := Encode(registry)
	m.Registry = Reference{Path: "data/content-package-memberships/registries/test-current.json", SHA256: digest(b)}
	writeFixture(t, root, m.Registry.Path, b)
	writeFixture(t, root, PageRegistryPath, b)
	m.Identity = identity(m)
	b, _ = Encode(m)
	membership := Reference{Path: "data/content-package-memberships/receipts/" + m.Identity + ".json", SHA256: digest(b)}
	writeFixture(t, root, membership.Path, b)
	b, _ = Encode(membership)
	writeFixture(t, root, PackageMembershipPath, b)
	projected, err := applyPackageMembership(root, base)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := packageByID(projected, merged.ID)
	if err != nil {
		t.Fatal(err)
	}
	merged.AuthoritySHA256 = pkg.Identity
	current.TranslationPackages = []i18n.TranslationPackageDescriptor{old, sibling}
	current.HistoricalTranslationPackages = nil
	if err := projectPackageMembershipCatalog(root, current, &m, projected); err != nil {
		t.Fatal(err)
	}
	merged = current.TranslationPackages[0]
	if err := current.UseTranslationPackage(merged.ID); err != nil {
		t.Fatal(err)
	}
	finals := []*i18n.QualityCheckFinalization{}
	inputs := []i18n.TranslationFinalizationInput{}
	for i, d := range predecessors {
		historical, err := i18n.HistoricalTranslationPackageCatalog(current, d.ID, d.AuthoritySHA256)
		if err != nil {
			t.Fatal(err)
		}
		selection := append([]string(nil), historical.TranslationScope...)
		bindingGenerateFixture(t, root, historical, selection)
		f := surfaceFinalize(t, root, historical, fmt.Sprintf("predecessor-%d", i), "", selection)
		finals = append(finals, f)
		ref, err := ReferenceFile(root, "data/quality-check-snapshots/zh-CN/"+f.SnapshotID+"/finalization.json")
		if err != nil {
			t.Fatal(err)
		}
		inputs = append(inputs, i18n.TranslationFinalizationInput{Package: d.ID, PackageAuthority: d.AuthoritySHA256, Finalization: i18n.GlossaryArchiveReference{Path: ref.Path, SHA256: ref.SHA256}})
	}
	if historicalGlossary {
		structuredGlossarySuccessor(t, root, "unused-e2e-term")
	}
	closure := PackageClosure{Schema: ClosureSchema, Locale: "zh-CN", Package: merged.ID, PackageSHA: merged.AuthoritySHA256, Membership: &membership, PageComposition: inputs}
	closure.Identity = identity(closure)
	b, _ = Encode(closure)
	closureRef := Reference{Path: workflowPath("zh-CN", merged.ID, "closure", closure.Identity), SHA256: digest(b)}
	writeFixture(t, root, closureRef.Path, b)
	build := func(root, locale string, refs []Reference) (*IntegratedSurfaceScope, error) {
		g, err := applyPackageMembership(root, base)
		if err != nil || g.Membership == nil {
			return nil, fmt.Errorf("current membership required: %v", err)
		}
		if !reflect.DeepEqual(refs, []Reference{closureRef}) {
			return nil, fmt.Errorf("exact current closure required")
		}
		var bound PackageClosure
		if err := readReference(root, refs[0], &bound); err != nil {
			return nil, err
		}
		if bound.Membership == nil || *bound.Membership != *g.Membership {
			return nil, fmt.Errorf("membership/closure mismatch")
		}
		v := newClosureValidation(root)
		v.global = g // The production reader obtains this same graph via LoadCurrent.
		composition, _, err := v.composePackage(root, current, locale, bound.PageComposition)
		if err != nil {
			return nil, err
		}
		if err := v.verify(); err != nil {
			return nil, err
		}
		scope := &IntegratedSurfaceScope{Schema: IntegratedSurfaceSchema, Locale: locale, Closures: refs}
		review, err := i18n.RequireUnifiedGlossaryReview(root, locale)
		if err != nil {
			return nil, err
		}
		scope.UnifiedReview = Reference{Path: review.Path, SHA256: review.SHA256}
		for _, u := range composition.Units {
			scope.Reviewed = append(scope.Reviewed, SurfaceAsset{ID: u.SourcePath, Package: merged.ID, TargetSHA: u.CandidateSHA256})
		}
		if err := BindIntegratedSurfaceCandidates(root, current, scope, finals...); err != nil {
			return nil, err
		}
		scope.Identity = identity(*scope)
		return scope, nil
	}
	attachIntegratedSurfaceAuthority(current, build)
	return surfaceComposedAuthorityFixture{root: root, current: current, ids: ids, base: base, membership: m, membershipRef: membership, closureRef: closureRef, finals: finals, inputs: inputs, build: build}
}

func TestSurfaceReopenComposedPackageExport(t *testing.T) {
	f := newSurfaceComposedAuthorityFixture(t, false)
	root, current, ids, finals := f.root, f.current, f.ids, f.finals
	base, m, membership, inputs, closureRef, build := f.base, f.membership, f.membershipRef, f.inputs, f.closureRef, f.build
	findings, texts := []SurfaceFinding{}, []string{}
	for _, u := range finals[0].Units[:5] {
		text := "Correct exact Surface defect: " + u.UnitID
		texts = append(texts, text)
		findings = append(findings, SurfaceFinding{ID: u.Snapshot.SourcePath, Finding: text, Class: i18n.SurfaceTranslationUnitLanguage, Artifacts: []i18n.SurfaceCandidateArtifact{{SnapshotID: finals[0].SnapshotID, Unit: u.Snapshot}}})
	}
	r, _ := surfaceReceipt(t, root, current, build, []Reference{closureRef}, "composed-surface-failed", "failed", findings)
	reopen := i18n.QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: "composed-surface-reopen", PreviousSnapshotID: finals[0].SnapshotID, SurfaceReviewID: r.ReviewID, Findings: texts, UnitIDs: ids[:5], IntegratedSurface: true}
	_, reopenPath, err := i18n.RecordQualityCheckSurfaceReopen(root, current, reopen)
	if err != nil {
		t.Fatal(err)
	}
	opts := i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, BatchID: "chatgpt-zh-CN-060", UnitKind: i18n.UnitKindPage, PreviousSnapshotID: finals[0].SnapshotID, SurfaceReopenID: reopen.ReopenID, AllowReexport: true, UnitIDs: ids[:5]}
	reject := func(t *testing.T, bad i18n.RetranslationExportOptions) {
		t.Helper()
		if _, err := i18n.ExportRetranslationBatch(root, current, bad); err == nil {
			t.Fatal("invalid composed Surface export accepted")
		}
		if _, err := os.Stat(filepath.Join(root, "data/retranslation-runs/zh-CN", bad.BatchID)); !os.IsNotExist(err) {
			t.Fatal("rejection published batch", err)
		}
	}
	for _, name := range []string{"subset", "superset", "duplicate", "sibling", "wrong-snapshot", "wrong-reopen"} {
		t.Run(name, func(t *testing.T) {
			o := opts
			o.UnitIDs = append([]string(nil), opts.UnitIDs...)
			switch name {
			case "subset":
				o.UnitIDs = o.UnitIDs[:4]
			case "superset":
				o.UnitIDs = ids[:6]
			case "duplicate":
				o.UnitIDs[4] = o.UnitIDs[0]
			case "sibling":
				o.UnitIDs[4] = ids[49]
			case "wrong-snapshot":
				o.PreviousSnapshotID = finals[1].SnapshotID
			case "wrong-reopen":
				o.SurfaceReopenID = "missing-reopen"
			}
			reject(t, o)
		})
	}
	for name, path := range map[string]string{"membership": membership.Path, "registry": PageRegistryPath, "sibling-finalization": inputs[1].Finalization.Path, "historical-finalization": inputs[0].Finalization.Path, "sibling-candidate": finals[1].Units[0].Snapshot.CandidatePath, "validation": finals[0].Units[0].Snapshot.ValidationPath, "source": finals[0].Units[0].Snapshot.SourcePath, "reopen": reopenPath, "closure": closureRef.Path} {
		t.Run("tampered-"+name, func(t *testing.T) {
			before, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			writeFixture(t, root, path, append(bytes.Clone(before), []byte("tampered")...))
			defer writeFixture(t, root, path, before)
			reject(t, opts)
		})
	}
	t.Run("missing-composition", func(t *testing.T) {
		before, _ := readRegular(root, PackageMembershipPath)
		if err := os.Remove(filepath.Join(root, PackageMembershipPath)); err != nil {
			t.Fatal(err)
		}
		defer writeFixture(t, root, PackageMembershipPath, before)
		reject(t, opts) // An arbitrary current 51/historical 49 registry is insufficient.
	})
	t.Run("wrong-membership-authority", func(t *testing.T) {
		bad := m
		bad.Predecessors = append([]MembershipPredecessor(nil), m.Predecessors...)
		bad.Predecessors[1].Authority = digest([]byte("wrong sibling authority"))
		bad.Identity = ""
		bad.Identity = identity(bad)
		b, _ := Encode(bad)
		ref := Reference{Path: "data/content-package-memberships/receipts/" + bad.Identity + ".json", SHA256: digest(b)}
		writeFixture(t, root, ref.Path, b)
		before, _ := readRegular(root, PackageMembershipPath)
		b, _ = Encode(ref)
		writeFixture(t, root, PackageMembershipPath, b)
		defer writeFixture(t, root, PackageMembershipPath, before)
		reject(t, opts)
	})
	t.Run("wrong-sibling-finalization", func(t *testing.T) {
		bad := append([]i18n.TranslationFinalizationInput(nil), inputs...)
		bad[1].Finalization = inputs[0].Finalization // Valid bytes, wrong predecessor scope.
		g, err := applyPackageMembership(root, base)
		if err != nil {
			t.Fatal(err)
		}
		v := newClosureValidation(root)
		v.global = g
		if _, _, err := v.composePackage(root, current, "zh-CN", bad); err == nil {
			t.Fatal("valid but wrong sibling finalization accepted")
		}
	})
	export, err := i18n.ExportRetranslationBatch(root, current, opts)
	if err != nil || export.UnitCount != 5 || len(current.TranslationScope) != 51 {
		t.Fatal(export, err)
	}
	var manifest i18n.RetranslationBatchManifest
	manifestData, err := readRegular(root, export.BatchPath+"/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := StrictJSON(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	selected := []string{}
	for _, u := range manifest.Units {
		if u.PreviousSnapshotID != finals[0].SnapshotID || u.RevisionAuthorizationID != reopen.ReopenID || u.RevisionFeedbackSource != "surface_review" {
			t.Fatal("revision provenance changed", u)
		}
		selected = append(selected, u.UnitID)
	}
	sort.Strings(selected)
	want := append([]string(nil), ids[:5]...)
	sort.Strings(want)
	if !reflect.DeepEqual(selected, want) {
		t.Fatal("historical/sibling boundary crossed", selected)
	}
	opts.BatchID = "chatgpt-zh-CN-061"
	reject(t, opts) // Export-only state already consumes the exact reopen.
}
