package sitecontent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// This bounded probe consumes exact sealed inputs. It never builds a Surface
// scope, replays a closure, or writes repository/formal evidence.
func TestIntegratedSurfaceCurrentAuthorizationProofDiagnosis(t *testing.T) {
	if os.Getenv("GO_TOUR_CURRENT_PROOF_DIAGNOSIS") != "1" {
		t.Skip("explicit readonly current proof diagnosis")
	}
	root := repositoryRoot(t)
	// A conservative whole-workspace inventory also covers untracked formal
	// files, staging, canonical targets and future artifact directories.
	fingerprint := func() map[string]string {
		t.Helper()
		out := map[string]string{}
		err := filepath.WalkDir(root, func(path string, e fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path == filepath.Join(root, ".git") {
				return filepath.SkipDir
			}
			if e.IsDir() {
				return nil
			}
			info, err := e.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("non-regular inventory member: %s", path)
			}
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			h := sha256.New()
			_, readErr := io.Copy(h, f)
			closeErr := f.Close()
			if readErr != nil {
				return readErr
			}
			if closeErr != nil {
				return closeErr
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			out[filepath.ToSlash(rel)] = hex.EncodeToString(h.Sum(nil))
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	before := fingerprint()
	defer func() {
		after := fingerprint()
		changes, err := contentidentity.CompareArtifactDigests(before, after)
		if err != nil {
			t.Error(err)
			return
		}
		t.Logf("INVENTORY_SCOPE=whole-workspace-except-.git BEFORE_PATHS=%d AFTER_PATHS=%d", len(before), len(after))
		t.Logf("FORMAL_ARTIFACT_PATHS_ADDED=%d FORMAL_ARTIFACT_PATHS_REMOVED=%d FORMAL_ARTIFACT_BYTES_CHANGED=%d", changes.PathsAdded, changes.PathsRemoved, changes.BytesChanged)
		if !changes.Unchanged() {
			t.Error("workspace/formal artifact mutation detected")
		}
	}()
	done := contentidentity.BeginMemoization()
	defer done()
	base, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackagesReadonly(root, base); err != nil {
		t.Fatal(err)
	}
	AttachIntegratedSurfaceAuthority(base)
	if os.Getenv("GO_TOUR_CURRENT_PROOF_TOUR_ONLY") == "1" {
		diagnoseRetainedTourAuthorizationProof(t, root, base)
		return
	}
	type input struct {
		prefix, pkg      string
		closure, receipt Reference
	}
	inputs := []input{
		{"LEARN_DOCS", "learn-docs-v1", Reference{"data/site-workflow/zh-CN/learn-docs-v1/closure/1380af4ecfef28e9ea70d41672881532d8bae8cc655c3b308d15082af34e1e21.json", "464593c57c7d4b5490ca973f649cdf2c189040a124715940a7b794f94083ce9b"}, Reference{"data/translation-authorizations/zh-CN/19cca3742fc2b55984c00c8d06e0fef6c085c97bbe133ef198dbfa29cdc82e8f/learn-docs-v1-qc-023/59d21a1aadeccf70afe0f92b30b484ef04d5475d4d3580de51db109640383081.json", "f39f141153598eab90d7f7ee63be1aaf70810510c734475949c74eb0087e53c6"}},
		{"SHELL", "site-v2-shell", Reference{"data/site-workflow/zh-CN/site-v2-shell/closure/68cd5c0142c1a73e06375d45dce5e45a2f8441f6d682bdf7db58b7b4336d4e42.json", "5fe1e74d2f85f45693575184c790ed5a9ecbe8f0d9a7f680daa0cfaf62335194"}, Reference{"data/translation-authorizations/zh-CN/a03a8d8174fbafd8a01589a162af42f4c3adbb816f4131f2f2e0abbbe16d8ca2/site-v2-shell-qc-004/3e525d19caf38b758445a4af65b4b41a99800329fc932ae9f765635631e2c653.json", "1c1fe52ab37bf02d887828c43bced30b95df892c1f2a575c94fb4489c6b778f8"}},
	}
	report := map[string]any{}
	failed := []string{}
	for _, in := range inputs {
		var closure PackageClosure
		if err := readReference(root, in.closure, &closure); err != nil {
			t.Fatal(err)
		}
		if closure.Package != in.pkg || closure.PageFinalization == nil {
			t.Fatal("unexpected closure package/PageFinalization", in.pkg)
		}
		var final i18n.QualityCheckFinalization
		if err := readReference(root, *closure.PageFinalization, &final); err != nil {
			t.Fatal(err)
		}
		catalog := *base
		if err := catalog.UseTranslationPackage(in.pkg); err != nil {
			t.Fatal(err)
		}
		current, gateErr := i18n.RequireCurrentTranslationAuthorization(root, &catalog, "zh-CN", final.SnapshotID)
		result, gateError := "PASS", ""
		if gateErr != nil {
			result, gateError = "FAILED", gateErr.Error()
			failed = append(failed, in.pkg)
		}
		if gateErr != nil || current == nil {
			current, err = i18n.AssessCurrentTranslation(root, &catalog, "zh-CN", final.SnapshotID)
			if err != nil {
				t.Fatal("fresh assessment", in.pkg, err)
			}
		}
		var saved i18n.CurrentTranslationAuthorization
		if err := readReference(root, in.receipt, &saved); err != nil {
			t.Fatal(err)
		}
		if saved.Package != in.pkg || saved.Historical.Finalization.SnapshotID != final.SnapshotID || in.receipt.Path != i18n.CurrentTranslationAuthorizationPath(&saved) || len(saved.Canonical) != 0 {
			t.Fatal("exact saved receipt or normalization mismatch", in.pkg)
		}
		// A separate exact-path check proves this particular sealed receipt fails
		// (rather than assuming the directory selector chose it).
		_, exactErr := i18n.CheckCurrentTranslationAuthorization(root, &catalog, in.receipt.Path)
		if (gateErr == nil) != (exactErr == nil) || (gateErr != nil && gateErr.Error() != exactErr.Error()) {
			t.Fatal("Require/exact receipt checker disagreement", gateErr, exactErr)
		}
		normalized := *current
		normalized.UnifiedReview = saved.UnifiedReview
		normalized.Identity = ""
		normalized.Identity = identity(normalized)
		difference := firstAuthorizationDiagnosticDifference(reflect.ValueOf(saved), reflect.ValueOf(normalized), "")
		if (gateErr == nil) != (difference == "") {
			t.Fatal("checker/normalized comparison disagreement", in.pkg, difference)
		}
		for _, field := range []string{"package", "package_authority_sha256", "scope_identity_sha256", "historical_integrity", "historical_integrity.immutable_references", "current_glossary", "current_unified_review", "units", "identity_sha256"} {
			a := currentProofDiagnosticValue(reflect.ValueOf(saved), field)
			b := currentProofDiagnosticValue(reflect.ValueOf(normalized), field)
			t.Logf("%s_COMPARE_FIELD=%s EQUAL=%t", in.prefix, field, reflect.DeepEqual(a, b))
		}
		a, _ := json.Marshal(currentProofDiagnosticValue(reflect.ValueOf(saved), difference))
		b, _ := json.Marshal(currentProofDiagnosticValue(reflect.ValueOf(normalized), difference))
		t.Logf("%s_REQUIRE_RESULT=%s\n%s_REQUIRE_ERROR=%s\n%s_SAVED_IDENTITY=%s\n%s_FRESH_IDENTITY=%s\n%s_FIRST_DIFFERENCE=%s\n%s_SAVED_VALUE=%s\n%s_FRESH_VALUE=%s\n%s_COMPATIBLE=%d\n%s_AFFECTED=%d\n%s_STALE=%d\n%s_UNKNOWN=%d", in.prefix, result, in.prefix, gateError, in.prefix, saved.Identity, in.prefix, normalized.Identity, in.prefix, difference, in.prefix, a, in.prefix, b, in.prefix, current.Compatible, in.prefix, current.Affected, in.prefix, current.Stale, in.prefix, current.Unknown)
		t.Logf("%s_SNAPSHOT=%s SAVED_RECEIPT=%s SHA256=%s", in.prefix, final.SnapshotID, in.receipt.Path, in.receipt.SHA256)
		report[in.prefix] = map[string]any{"closure": in.closure, "receipt": in.receipt, "require_result": result, "require_error": gateError, "saved": saved, "fresh": current, "normalized_fresh": normalized, "first_difference": difference}
	}
	stale := "neither"
	if len(failed) == 2 {
		stale = "both"
	} else if len(failed) == 1 {
		stale = failed[0]
	}
	t.Logf("STALE_PACKAGE=%s", stale)
	// Optional engineering evidence goes outside the repository, never to a
	// formal receipt directory. The default emits only test output.
	if path := os.Getenv("GO_TOUR_CURRENT_PROOF_REPORT"); path != "" {
		if !strings.HasPrefix(filepath.Clean(path), "/tmp/") {
			t.Fatal("diagnostic report must be outside repository under /tmp")
		}
		b, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(b, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
		t.Logf("DIAGNOSTIC_REPORT=%s SHA256=%s", path, digest(append(b, '\n')))
	}
}

// The next independent consumer after both prospective closures is retained
// Tour. Diagnose its native canonical checkpoint without constructing a scope.
func diagnoseRetainedTourAuthorizationProof(t *testing.T, root string, base *i18n.Catalog) {
	t.Helper()
	catalog := *base
	catalog.TranslationScope = nil
	_, _, gateErr := i18n.RequireCurrentCanonicalQualityCheck(root, "zh-CN", &catalog)
	t.Logf("RETAINED_TOUR_CANONICAL_REQUIRE_ERROR=%v", gateErr)
	const dir = "data/translation-authorizations/zh-CN/88337426ad44d3e7193d668a062bed7022c230691d97412be70a49703652b1a6/qc-010"
	paths, err := filepath.Glob(filepath.Join(root, dir, "*.json"))
	if err != nil || len(paths) != 2 {
		t.Fatal("unexpected retained Tour sealed receipt set", paths, err)
	}
	fresh, err := i18n.AssessCurrentTranslation(root, &catalog, "zh-CN", "qc-010")
	if err != nil {
		t.Fatal("fresh retained Tour assessment", err)
	}
	canonicalPaths, err := i18n.CanonicalTranslationUnitTargetPaths(&catalog, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	canonical := []i18n.GlossaryArchiveReference{}
	for path := range canonicalPaths {
		b, err := readRegular(root, path)
		if err != nil {
			t.Fatal(err)
		}
		canonical = append(canonical, i18n.GlossaryArchiveReference{Path: path, SHA256: digest(b)})
	}
	sort.Slice(canonical, func(i, j int) bool { return canonical[i].Path < canonical[j].Path })
	report := map[string]any{"canonical_consumer_error": fmt.Sprint(gateErr)}
	for _, path := range paths {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatal(err)
		}
		rel = filepath.ToSlash(rel)
		b, err := readRegular(root, rel)
		if err != nil {
			t.Fatal(err)
		}
		var saved i18n.CurrentTranslationAuthorization
		if err := StrictJSON(b, &saved); err != nil {
			t.Fatal(err)
		}
		_, exactErr := i18n.CheckCurrentTranslationAuthorization(root, &catalog, rel)
		// An exact stale error means the checker's canonical target reconstruction
		// succeeded, including equality to accepted (EOF-normalized) candidates.
		if exactErr == nil || !strings.HasPrefix(exactErr.Error(), "current authorization stale:") {
			t.Fatal("unexpected retained Tour checker result", rel, exactErr)
		}
		normalized := *fresh
		normalized.UnifiedReview = saved.UnifiedReview
		normalized.Canonical = canonical
		normalized.Identity = ""
		normalized.Identity = identity(normalized)
		difference := firstAuthorizationDiagnosticDifference(reflect.ValueOf(saved), reflect.ValueOf(normalized), "")
		savedValue, _ := json.Marshal(currentProofDiagnosticValue(reflect.ValueOf(saved), difference))
		freshValue, _ := json.Marshal(currentProofDiagnosticValue(reflect.ValueOf(normalized), difference))
		t.Logf("RETAINED_TOUR_RECEIPT=%s SHA256=%s REQUIRE_ERROR=%v SAVED_IDENTITY=%s FRESH_IDENTITY=%s FIRST_DIFFERENCE=%s SAVED_VALUE=%s FRESH_VALUE=%s COMPATIBLE=%d AFFECTED=%d STALE=%d UNKNOWN=%d", rel, digest(b), exactErr, saved.Identity, normalized.Identity, difference, savedValue, freshValue, fresh.Compatible, fresh.Affected, fresh.Stale, fresh.Unknown)
		for _, field := range []string{"package", "package_authority_sha256", "scope_identity_sha256", "historical_integrity", "historical_integrity.immutable_references", "current_glossary", "current_unified_review", "current_canonical_targets", "units", "identity_sha256"} {
			a := currentProofDiagnosticValue(reflect.ValueOf(saved), field)
			b := currentProofDiagnosticValue(reflect.ValueOf(normalized), field)
			t.Logf("RETAINED_TOUR_COMPARE_FIELD=%s EQUAL=%t", field, reflect.DeepEqual(a, b))
		}
		report[rel] = map[string]any{"saved": saved, "fresh": fresh, "normalized_fresh": normalized, "first_difference": difference, "saved_value": json.RawMessage(savedValue), "fresh_value": json.RawMessage(freshValue), "checker_error": exactErr.Error()}
	}
	if gateErr == nil || gateErr.Error() != "current authorization stale: affected=0 stale=0 unknown=0" {
		t.Fatal("retained Tour did not reproduce the exact consumer failure", gateErr)
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	const path = "/tmp/integrated-surface-retained-tour-proof-20261030.json"
	if err := os.WriteFile(path, append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("RETAINED_TOUR_DIAGNOSTIC_REPORT=%s SHA256=%s", path, digest(append(b, '\n')))
}

func currentProofDiagnosticValue(v reflect.Value, path string) any {
	if path == "" {
		return nil
	}
	part, rest, _ := strings.Cut(path, ".")
	if part == "length" && v.Kind() == reflect.Slice {
		return v.Len()
	}
	name, index, indexed := strings.Cut(part, "[")
	for i := 0; i < v.NumField(); i++ {
		if strings.Split(v.Type().Field(i).Tag.Get("json"), ",")[0] != name {
			continue
		}
		value := v.Field(i)
		if indexed {
			var n int
			if _, err := fmt.Sscanf(index, "%d]", &n); err != nil {
				panic(err)
			}
			value = value.Index(n)
		}
		if rest != "" {
			return currentProofDiagnosticValue(value, rest)
		}
		return value.Interface()
	}
	panic("unknown diagnostic field: " + path)
}
