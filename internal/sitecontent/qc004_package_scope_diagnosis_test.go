package sitecontent

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Read only the selector and its real registered inventory. In particular this
// does not run authorization, reattestation, scope building or a formal writer.
func TestQC004PackageScopeDiagnosis(t *testing.T) {
	mode := os.Getenv("GO_TOUR_QC004_SCOPE_DIAG")
	if mode == "" {
		t.Skip("explicit readonly selector diagnosis")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	c, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackagesReadonly(root, c); err != nil {
		t.Fatal(err)
	}
	AttachIntegratedSurfaceAuthority(c)
	if err := c.UseTranslationPackage("site-v2-shell"); err != nil {
		t.Fatal(err)
	}
	const snapshot = "site-v2-shell-qc-004"
	base := "data/quality-check-snapshots/zh-CN/" + snapshot + "/"
	var s i18n.QualityCheckSnapshotManifest
	b, err := readRegular(root, base+"manifest.json")
	if err != nil || json.Unmarshal(b, &s) != nil {
		t.Fatal(err)
	}
	var f i18n.QualityCheckFinalization
	b, err = readRegular(root, base+"finalization.json")
	if err != nil || json.Unmarshal(b, &f) != nil {
		t.Fatal(err)
	}
	if s.ContentPackage == nil {
		t.Fatal("explicit binding missing")
	}
	t.Logf("REQUESTED_SNAPSHOT=%s QC004_CONTENT_PACKAGE_PRESENT=yes QC004_CONTENT_PACKAGE_ID=%s QC004_CONTENT_PACKAGE_AUTHORITY=%s", snapshot, s.ContentPackage.ID, s.ContentPackage.AuthoritySHA256)
	count := 0
	for _, d := range append(append([]i18n.TranslationPackageDescriptor{}, c.TranslationPackages...), c.HistoricalTranslationPackages...) {
		if len(d.Units) != len(f.Units) {
			continue
		}
		match := true
		for i, u := range d.Units {
			match = match && u.Unit.SourcePath == f.Units[i].Snapshot.SourcePath && u.Unit.Kind == f.Units[i].Snapshot.UnitKind
		}
		if !match {
			continue
		}
		count++
		for _, u := range d.Units {
			t.Logf("MATCH_%d_PACKAGE_ID=%s MATCH_%d_AUTHORITY_SHA256=%s MATCH_%d_UNIT_COUNT=%d MATCH_%d_UNIT_ID=%s MATCH_%d_SOURCE_PATH=%s", count, d.ID, count, d.AuthoritySHA256, count, len(d.Units), count, u.Unit.ID, count, u.Unit.SourcePath)
		}
	}
	t.Logf("AMBIGUITY_TRIGGER_SNAPSHOT=%s AMBIGUITY_TRIGGER_PREVIOUS_SNAPSHOT=%s AMBIGUITY_TRIGGER_CONTENT_PACKAGE_PRESENT=yes SELECTOR_MATCH_COUNT=%d", f.SnapshotID, f.PreviousSnapshotID, count)
	if count < 2 {
		t.Fatal("missing real ambiguity topology")
	}
	// A finalization-only disposable root proves the pre-fix error precedes
	// every manifest/predecessor read, rather than arising in qc-002 recursion.
	isolated := t.TempDir()
	writeFixture(t, isolated, base+"finalization.json", b)
	_, _, isolatedErr := i18n.SelectFinalizedTranslationPackage(isolated, c, "zh-CN", snapshot)
	selected, registered, err := i18n.SelectFinalizedTranslationPackage(root, c, "zh-CN", snapshot)
	t.Logf("SELECTOR_ERROR=%v FINALIZATION_ONLY_ROOT_ERROR=%v", err, isolatedErr)
	if mode == "before" {
		if err == nil || isolatedErr == nil || err.Error() != "ambiguous finalized package scope" || isolatedErr.Error() != err.Error() {
			t.Fatal("exact original selector failure not reproduced", err, isolatedErr)
		}
	} else {
		if err != nil || !registered || selected.TranslationPackages[0].AuthoritySHA256 != s.ContentPackage.AuthoritySHA256 {
			t.Fatal("explicit current binding unresolved", err)
		}
		if isolatedErr == nil {
			t.Fatal("missing bound manifest accepted")
		}
		// The installed completion names the frozen old authority; never infer
		// it from today's same-path descriptors or choose their latest entry.
		var installed Locale
		b, err := readRegular(root, LocalePath(s.Locale))
		if err != nil || StrictJSON(b, &installed) != nil {
			t.Fatal("installed authority", err)
		}
		found := false
		for _, completion := range installed.Packages {
			if completion.Package != s.ContentPackage.ID {
				continue
			}
			historical, err := i18n.HistoricalTranslationPackageCatalog(c, completion.Package, completion.PackageIdentity)
			if err != nil {
				t.Fatal(err)
			}
			// Ordinary historical consumers reuse the existing flattened sealed
			// proof; they do not replay a pre-checkpoint current projection.
			if _, err := i18n.ReadHistoricalTranslationIntegrity(root, historical, s.Locale, f.PreviousSnapshotID); err != nil {
				t.Fatal("exact predecessor integrity", err)
			}
			t.Logf("HISTORICAL_PREDECESSOR=%s HISTORICAL_AUTHORITY=%s HISTORICAL_INTEGRITY=PASS", f.PreviousSnapshotID, completion.PackageIdentity)
			found = true
		}
		if !found {
			t.Fatal("missing exact installed predecessor authority")
		}
	}
}
