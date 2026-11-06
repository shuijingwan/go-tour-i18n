package sitecontent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Read-only diagnosis follows the closure Page gates before retained Tour.
// Never builds/saves a Surface scope or records new authorization.
func TestIntegratedSurfaceStaleOneReadonly(t *testing.T) {
	if os.Getenv("GO_TOUR_SURFACE_STALE_DIAGNOSIS") != "1" {
		t.Skip("explicit real Integrated Surface stale=1 diagnosis")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	closures := []Reference{
		{"data/site-workflow/zh-CN/learn-docs-v1/closure/fe866b9aa3a9b9ce2b26757b2bd6da513cabe41fe1f414ef6430ab6fa5202db9.json", "47588d761b615abb7f435b488c1058a83cc06e2b833d9118ccac5282a692e88e"},
		{"data/site-workflow/zh-CN/site-v2-shell/closure/41ec108c37469562da5f8870b4d3fcc84ce1afefda148da1d8bad091ca9717d3.json", "dcb7fcafd1708c4198747356996b51776c2e1c874bd86ff7d0110c31135e8ae9"},
	}
	type summary struct {
		AuthorityKind   string                                `json:"authority_kind"`
		Package         string                                `json:"package"`
		Snapshot        string                                `json:"snapshot_id"`
		Identity        string                                `json:"authorization_identity"`
		Compatible      int                                   `json:"compatible"`
		Affected        int                                   `json:"affected"`
		Stale           int                                   `json:"stale"`
		Unknown         int                                   `json:"unknown"`
		Error           string                                `json:"consumer_error,omitempty"`
		SavedPath       string                                `json:"saved_path,omitempty"`
		Saved           *i18n.CurrentTranslationAuthorization `json:"saved,omitempty"`
		Current         *i18n.CurrentTranslationAuthorization `json:"current"`
		Compared        *i18n.CurrentTranslationAuthorization `json:"current_with_checker_normalization,omitempty"`
		FirstDifference string                                `json:"first_differing_field,omitempty"`
	}
	report := map[string]any{
		"surface_scope_input_closures": closures,
		"call_chain":                   []string{"siteContentCommand(surface-scope)", "BuildIntegratedSurfaceScope", "closureValidation.buildIntegratedSurfaceScope", "closureValidation.checkPackageClosure", "computePackageClosure", "buildPackageClosure", "projectSharedPackageFinalization", "FinalizedTranslationPackageCandidates", "RequireCurrentTranslationAuthorization", "FindCurrentTranslationAuthorization", "CheckCurrentTranslationAuthorization"},
		"retained_tour_path":           []string{"planLocaleLanguage(tourOnly=true)", "currentTourCheckpoint", "RequireCurrentCanonicalQualityCheck"},
	}
	base, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackages(root, base); err != nil {
		t.Fatal(err)
	}
	AttachIntegratedSurfaceAuthority(base)
	checks := []summary{}
	staleUnits := []map[string]any{}
	for _, ref := range closures {
		var closure PackageClosure
		if err := readReference(root, ref, &closure); err != nil {
			t.Fatal(err)
		}
		if closure.PageFinalization == nil {
			t.Fatal("missing Page authority", ref)
		}
		var finalization i18n.QualityCheckFinalization
		if err := readReference(root, *closure.PageFinalization, &finalization); err != nil {
			t.Fatal(err)
		}
		catalog := *base
		if err := catalog.UseTranslationPackage(closure.Package); err != nil {
			t.Fatal(err)
		}
		a, gateErr := i18n.RequireCurrentTranslationAuthorization(root, &catalog, "zh-CN", finalization.SnapshotID)
		if a == nil {
			a, err = i18n.AssessCurrentTranslation(root, &catalog, "zh-CN", finalization.SnapshotID)
			if err != nil {
				t.Fatal("fresh assessment", closure.Package, err)
			}
		}
		x := summary{AuthorityKind: "current-translation-authorization", Package: closure.Package, Snapshot: finalization.SnapshotID, Identity: a.Identity, Compatible: a.Compatible, Affected: a.Affected, Stale: a.Stale, Unknown: a.Unknown, Current: a}
		if gateErr != nil {
			x.Error = gateErr.Error()
		}
		matches, err := filepath.Glob(filepath.Join(root, "data/translation-authorizations/zh-CN", a.ScopeIdentity, finalization.SnapshotID, "*.json"))
		if err != nil || len(matches) > 1 {
			t.Fatal("ambiguous saved authority", matches, err)
		}
		if len(matches) == 1 {
			b, err := os.ReadFile(matches[0])
			if err != nil {
				t.Fatal(err)
			}
			var saved i18n.CurrentTranslationAuthorization
			if err := StrictJSON(b, &saved); err != nil {
				t.Fatal(err)
			}
			x.Saved, x.SavedPath = &saved, strings.TrimPrefix(matches[0], root+string(filepath.Separator))
			if len(saved.Canonical) != 0 {
				t.Fatal("diagnostic needs exact canonical normalization", x.SavedPath)
			}
			compare := *a
			// Match CheckCurrentTranslationAuthorization's intentional full-Review
			// normalization before comparing identity, not an eligibility waiver.
			compare.UnifiedReview = saved.UnifiedReview
			compare.Identity = ""
			compare.Identity = identity(compare)
			x.Compared = &compare
			x.FirstDifference = firstAuthorizationDiagnosticDifference(reflect.ValueOf(saved), reflect.ValueOf(compare), "")
		}
		for i, u := range a.Units {
			if u.Classification != "stale" {
				continue
			}
			currentID := catalog.TranslationScope[i]
			unit, err := catalog.Unit(currentID)
			if err != nil {
				t.Fatal(err)
			}
			staleUnits = append(staleUnits, map[string]any{"package": closure.Package, "snapshot_id": finalization.SnapshotID, "unit_id": u.Snapshot.UnitID, "current_unit_id": currentID, "source_path": u.Snapshot.SourcePath, "saved_source_sha256": u.Snapshot.SourceSHA256, "current_source_sha256": unit.SourceSHA256, "reasons": u.Reasons})
		}
		checks = append(checks, x)
	}
	// Tour uses its current native installation Snapshot, never qc-008.
	tour := *base
	tour.TranslationScope = nil
	d, err := i18n.DiagnoseQualityCheckAuthorization(root, &tour, "zh-CN", "qc-010")
	report["tour_qc010_canonical_diagnostic"] = d
	if err != nil {
		report["tour_qc010_canonical_error"] = err.Error()
	}
	tourPass := err == nil && d != nil && d.CanonicalAuthorized
	report["tour_qc010_current_authorization_pass"] = tourPass
	const nativePath = "locales/zh-CN/candidates/.promotion-d190ac2f8b31bebee9c167c49ca9a481f8bd7a8b6ab165f83dac6f3fad459ebe.json"
	nativeBytes, err := os.ReadFile(filepath.Join(root, nativePath))
	if err != nil {
		t.Fatal(err)
	}
	var native i18n.TargetSuccessorEvidence
	if err := StrictJSON(nativeBytes, &native); err != nil {
		t.Fatal(err)
	}
	if native.SnapshotID != "qc-010" {
		t.Fatal("unexpected native proof Snapshot", native.SnapshotID)
	}
	// Native Tour consumes successor authority rather than a persisted shared
	// CurrentTranslationAuthorization. Do not invent classification counters.
	report["tour_authority"] = map[string]any{"package": "tour-v1", "snapshot_id": native.SnapshotID, "authority_kind": native.Kind, "authority_identity": native.Identity, "path": nativePath, "file_sha256": digest(nativeBytes), "compatible": nil, "affected": nil, "stale": nil, "unknown": nil, "classification_note": "native canonical authorization; shared classification counters not applicable", "canonical_authorized": tourPass, "diagnostic": d}
	report["checked_authorization_count"] = len(checks) + 1
	// Reproduce the exact failing closure consumer without constructing a scope.
	_, _, closureErr := CheckPackageClosure(root, closures[1])
	if closureErr != nil {
		report["shell_closure_current_check_error"] = closureErr.Error()
	}
	report["authorizations"], report["stale_units"] = checks, staleUnits
	report["surface_phase_reached"] = "prospective closure Page authorization; retained Tour follows only after both closures pass"
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	b = append(b, '\n')
	const path = "/tmp/zh-CN-integrated-surface-stale1-diagnosis-20261018.json"
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	for _, x := range checks {
		t.Logf("PACKAGE=%s SNAPSHOT_ID=%s AUTHORIZATION_IDENTITY=%s COMPATIBLE=%d AFFECTED=%d STALE=%d UNKNOWN=%d", x.Package, x.Snapshot, x.Identity, x.Compatible, x.Affected, x.Stale, x.Unknown)
	}
	t.Logf("EVIDENCE=%s SHA256=%s", path, digest(b))
	t.Logf("TOUR_SNAPSHOT=qc-010 NATIVE_AUTHORITY_IDENTITY=%s CURRENT_AUTHORIZATION_PASS=%t", native.Identity, tourPass)
	if len(staleUnits) != 1 || closureErr == nil || closureErr.Error() != "current authorization stale: affected=0 stale=1 unknown=0" {
		t.Fatal("unexpected stale=1 diagnosis", staleUnits, closureErr)
	}
	if !tourPass {
		t.Fatal("Tour qc-010 canonical diagnostic did not pass", d)
	}
}

func firstAuthorizationDiagnosticDifference(a, b reflect.Value, path string) string {
	if reflect.DeepEqual(a.Interface(), b.Interface()) {
		return ""
	}
	switch a.Kind() {
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			name := strings.Split(a.Type().Field(i).Tag.Get("json"), ",")[0]
			p := name
			if path != "" {
				p = path + "." + name
			}
			if d := firstAuthorizationDiagnosticDifference(a.Field(i), b.Field(i), p); d != "" {
				return d
			}
		}
	case reflect.Slice:
		if a.Len() != b.Len() {
			return path + ".length"
		}
		for i := 0; i < a.Len(); i++ {
			if d := firstAuthorizationDiagnosticDifference(a.Index(i), b.Index(i), fmt.Sprintf("%s[%d]", path, i)); d != "" {
				return d
			}
		}
	}
	return path
}
