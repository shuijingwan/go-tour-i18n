package sitecontent

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Optional readonly registration evidence; never prepares/publishes a Snapshot.
func TestMembershipHistoricalAuthorityReadonlyEvidence(t *testing.T) {
	out, snapshot := os.Getenv("GO_TOUR_HISTORICAL_REGISTRATION_EVIDENCE"), os.Getenv("GO_TOUR_HISTORICAL_REGISTRATION_SNAPSHOT")
	if out == "" || snapshot == "" {
		t.Skip("explicit readonly binding/evidence paths required")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	base, err := loadSourceCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	c, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackagesReadonly(root, c); err != nil {
		t.Fatal(err)
	}
	data, err := readRegular(root, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var s i18n.QualityCheckSnapshotManifest
	if err := StrictJSON(data, &s); err != nil || s.ContentPackage == nil {
		t.Fatal("missing frozen package binding", err)
	}
	type descriptor struct {
		ID, Authority string
		Units         []i18n.TranslationUnit
	}
	list := func(ds []i18n.TranslationPackageDescriptor) []descriptor {
		out := []descriptor{}
		for _, d := range ds {
			x := descriptor{ID: d.ID, Authority: d.AuthoritySHA256}
			for _, u := range d.Units {
				v := u.Unit
				v.Source = nil
				v.Adapter = nil
				x.Units = append(x.Units, v)
			}
			out = append(out, x)
		}
		return out
	}
	h, lookupErr := i18n.HistoricalTranslationPackageCatalog(c, s.ContentPackage.ID, s.ContentPackage.AuthoritySHA256)
	message := "PASS"
	if lookupErr != nil {
		message = lookupErr.Error()
	}
	report := map[string]any{"snapshot_binding": s.ContentPackage, "source_packages": base.Packages, "source_historical_packages": base.HistoricalPackages, "current_packages": g.Packages, "current_historical_packages": g.HistoricalPackages, "catalog_current": list(c.TranslationPackages), "catalog_historical": list(c.HistoricalTranslationPackages), "lookup": message}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if lookupErr != nil {
		t.Fatal(lookupErr)
	}
	if len(h.TranslationScope) != len(s.ContentPackage.UnitIDs) {
		t.Fatal("historical complete scope mismatch")
	}
	m, err := readPackageMembership(root)
	if err != nil || m == nil {
		t.Fatal("membership source required", err)
	}
	for _, old := range m.Predecessors {
		view, err := i18n.HistoricalTranslationPackageCatalog(c, old.Package, old.Authority)
		if err != nil {
			t.Fatal(err)
		}
		nested, err := i18n.HistoricalTranslationPackageCatalog(view, s.ContentPackage.ID, s.ContentPackage.AuthoritySHA256)
		if err != nil {
			t.Fatal("nested historical registration", err)
		}
		if !reflect.DeepEqual(nested.TranslationScope, h.TranslationScope) {
			t.Fatal("nested frozen authority changed complete membership")
		}
	}
}

func TestMembershipHistoricalResultSurvivesPredecessorViews(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	f := membershipRegistrationOnlyFixture(t)
	current := f.current.TranslationPackages[0]
	if len(current.Units) != 51 {
		t.Fatal("complete current membership required")
	}
	for _, old := range f.membership.Predecessors {
		view, err := i18n.HistoricalTranslationPackageCatalog(f.current, old.Package, old.Authority)
		if err != nil {
			t.Fatal(err)
		}
		d := view.TranslationPackages[0]
		if d.ID != old.Package || d.AuthoritySHA256 != old.Authority || len(d.Units) != len(old.Sources) {
			t.Fatal("predecessor descriptor replaced by current membership")
		}
		for i, u := range d.Units {
			if u.Unit.SourcePath != old.Sources[i].Path || u.Unit.SourceSHA256 != old.Sources[i].SHA256 || digest(u.Unit.Source) != old.Sources[i].SHA256 {
				t.Fatal("predecessor ordered source inventory changed")
			}
		}
		nested, err := i18n.HistoricalTranslationPackageCatalog(view, current.ID, current.AuthoritySHA256)
		if err != nil {
			t.Fatal("membership result lost from historical projection", err)
		}
		if !reflect.DeepEqual(nested.TranslationScope, f.ids) {
			t.Fatal("wrong exact historical result inventory")
		}
		for i, u := range nested.TranslationPackages[0].Units {
			want := current.Units[i]
			if u.Unit.ID != want.Unit.ID || u.Unit.SourcePath != want.Unit.SourcePath || u.Unit.SourceSHA256 != want.Unit.SourceSHA256 || digest(u.Unit.Source) != want.Unit.SourceSHA256 || u.StableIndex != want.StableIndex {
				t.Fatal("historical source identity/order changed")
			}
		}
		if _, err := i18n.HistoricalTranslationPackageCatalog(view, current.ID, digest([]byte("wrong exact authority"))); err == nil {
			t.Fatal("unknown SHA fell back to same-ID current package")
		}
	}
	if err := f.current.UseTranslationPackage(f.membership.Predecessors[1].Package); err == nil {
		t.Fatal("retired sibling became current")
	}
	if err := f.current.UseTranslationPackage(current.ID); err != nil || len(f.current.TranslationScope) != 51 {
		t.Fatal("historical scope promoted over current membership", err)
	}
	missing := *f.current
	missing.HistoricalTranslationPackages = nil
	for _, d := range f.current.HistoricalTranslationPackages {
		if d.ID != current.ID || d.AuthoritySHA256 != current.AuthoritySHA256 {
			missing.HistoricalTranslationPackages = append(missing.HistoricalTranslationPackages, d)
		}
	}
	old := f.membership.Predecessors[0]
	view, err := i18n.HistoricalTranslationPackageCatalog(&missing, old.Package, old.Authority)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.HistoricalTranslationPackageCatalog(view, current.ID, current.AuthoritySHA256); err == nil {
		t.Fatal("missing historical registration accepted")
	}
	partial := *f.current
	partial.TranslationScope = append([]string(nil), f.ids[:50]...)
	if _, err := i18n.PrepareQualityCheckCandidateSnapshot(f.root, &partial, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "incomplete-membership", ContentPackage: current.ID}); err == nil || !strings.Contains(err.Error(), "complete selected registered package") {
		t.Fatal("incomplete current scope accepted", err)
	}
}

// Reuse the source-only card fixture and the real membership projection,
// without invoking language generation, QC or composed Surface publishers.
func membershipRegistrationOnlyFixture(t *testing.T) surfaceComposedAuthorityFixture {
	t.Helper()
	root, c, ids := surfaceFixture(t, "registration-membership", 51)
	base, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	old, sibling := c.TranslationPackages[0], c.TranslationPackages[0]
	old.Units = append([]i18n.AdaptedTranslationUnit(nil), old.Units[:49]...)
	sibling.ID = "registration-sibling"
	sibling.Units = append([]i18n.AdaptedTranslationUnit(nil), sibling.Units[49:]...)
	m := PackageMembership{Schema: PackageMembershipSchema, Package: old.ID}
	registry := PageRegistry{Schema: PageRegistrySchema}
	for _, d := range []*i18n.TranslationPackageDescriptor{&old, &sibling} {
		p := Package{ID: d.ID, Activation: "atomic-locale-package", ParserContract: UnitContract, Surfaces: []string{"source-" + d.ID}}
		for _, u := range d.Units {
			p.Routes = append(p.Routes, u.Route)
		}
		p.Identity = identity(p)
		d.AuthoritySHA256 = p.Identity
		base.Packages = append(base.Packages, p)
		base.Surfaces = append(base.Surfaces, Surface{ID: p.Surfaces[0], Package: p.ID})
		base.Versions[1].Packages = append(base.Versions[1].Packages, p.ID)
		pred := MembershipPredecessor{Package: p.ID, Authority: p.Identity}
		for _, u := range d.Units {
			pred.Sources = append(pred.Sources, Reference{Path: u.Unit.SourcePath, SHA256: u.Unit.SourceSHA256})
			base.Sources = append(base.Sources, Source{Path: u.Unit.SourcePath, Kind: "page", Origin: "repository", Package: p.ID, Surface: p.Surfaces[0], Route: u.Route, SHA256: u.Unit.SourceSHA256})
			registry.Pages = append(registry.Pages, StablePage{Index: len(registry.Pages) + 1, Path: u.Unit.SourcePath, Surface: p.Surfaces[0]})
		}
		m.Predecessors = append(m.Predecessors, pred)
		if len(registry.Pages) == 49 {
			sealPages(&registry)
			b, _ := Encode(registry)
			m.PreviousRegistry = Reference{Path: "data/registration-old-registry.json", SHA256: digest(b)}
			writeFixture(t, root, m.PreviousRegistry.Path, b)
		}
	}
	sealPages(&registry)
	b, _ := Encode(registry)
	m.Registry = Reference{Path: "data/registration-current-registry.json", SHA256: digest(b)}
	writeFixture(t, root, m.Registry.Path, b)
	writeFixture(t, root, PageRegistryPath, b)
	m.Identity = identity(m)
	b, _ = Encode(m)
	ref := Reference{Path: fmt.Sprintf("data/content-package-memberships/receipts/%s.json", m.Identity), SHA256: digest(b)}
	writeFixture(t, root, ref.Path, b)
	b, _ = Encode(ref)
	writeFixture(t, root, PackageMembershipPath, b)
	projected, err := applyPackageMembership(root, base)
	if err != nil {
		t.Fatal(err)
	}
	c.TranslationPackages = []i18n.TranslationPackageDescriptor{old, sibling}
	c.HistoricalTranslationPackages = nil
	if err := projectPackageMembershipCatalog(root, c, &m, projected); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage(m.Package); err != nil {
		t.Fatal(err)
	}
	return surfaceComposedAuthorityFixture{root: root, current: c, ids: ids, membership: m, membershipRef: ref}
}
