package i18n

import "testing"

func TestReplacementHistoricalDescriptorAfterPackageMembership(t *testing.T) {
	root, c, _ := futureContentFixture(t)
	original := c.TranslationPackages[0]
	original.AuthoritySHA256 = sum([]byte("original-package-source-authority"))
	historical := *c
	historical.TranslationPackages = []TranslationPackageDescriptor{original}
	current := original
	current.AuthoritySHA256 = sum([]byte("unified-package-membership"))
	current.Units = append([]AdaptedTranslationUnit{}, original.Units...)
	for i := range current.Units {
		current.Units[i].StableIndex += 10
	}
	c.TranslationPackages = []TranslationPackageDescriptor{current}
	c.HistoricalTranslationPackages = []TranslationPackageDescriptor{original, original}
	u := original.Units[0]
	input, err := prepareTranslationUnitInput(&TranslationUnit{ID: u.Unit.ID, Kind: u.Unit.Kind, SourcePath: u.Unit.SourcePath, Source: u.Unit.Source, SourceSHA256: u.Unit.SourceSHA256, Adapter: u.Adapter}, &Glossary{})
	if err != nil {
		t.Fatal(err)
	}
	sha := sum(canonicalizeRetranslationArtifactEOF([]byte(input.Text)))
	ref := GlossaryArchiveReference{Path: "fixtures/historical-representation.json", SHA256: sum([]byte("{}\n"))}
	writeSnapshotSource(t, root, ref.Path, []byte("{}\n"))
	r := &TranslationUnitReplacementResolution{HistoricalCatalog: &historical, Units: map[string]TranslationUnitReplacement{u.Unit.ID: {OldUnitID: u.Unit.ID, CurrentUnitID: u.Unit.ID, Package: original.ID, PackageSHA256: original.AuthoritySHA256, SourcePath: u.Unit.SourcePath, SourceSHA256: u.Unit.SourceSHA256, StableIndex: u.StableIndex, Route: u.Route, OldInputSHA256: sha, CurrentInputSHA256: sha, ReconciliationSHA256: sum([]byte("frozen-reconciliation"))}}}
	c.ReplacementResolver = func(string, *Catalog, GlossaryArchiveReference) (*TranslationUnitReplacementResolution, error) {
		return r, nil
	}
	ctx := &TranslationUnitReplacementContext{HistoricalDocuments: ref}
	if _, err := replacementResolution(root, c, ctx); err != nil {
		t.Fatal("historical origin mapping after membership merge", err)
	}
	c.HistoricalTranslationPackages = nil
	if _, err := replacementResolution(root, c, ctx); err == nil {
		t.Fatal("unregistered historical mapping accepted")
	}
	bad := original
	bad.Units = append(append([]AdaptedTranslationUnit{}, original.Units...), u)
	c.HistoricalTranslationPackages = []TranslationPackageDescriptor{bad}
	if _, err := replacementResolution(root, c, ctx); err == nil {
		t.Fatal("duplicate Unit within historical descriptor accepted")
	}
}
