package i18n

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQualityCheckReviewerWorkingSetKeepsSixtyAndKindBoundaries(t *testing.T) {
	scope := &QualityCheckScope{Pending: make([]QualityCheckScopeUnit, 0, 122)}
	for index := 1; index <= 122; index++ {
		kind := UnitKindPage
		if index > 103 {
			kind = UnitKindExample
		}
		scope.Pending = append(scope.Pending, QualityCheckScopeUnit{Index: index, UnitID: "unit", UnitKind: kind, RequiredAction: QualityCheckActionRequired})
	}
	for _, test := range []struct {
		start, limit, count, end int
		kind                     UnitKind
	}{
		{1, 60, 60, 60, UnitKindPage},
		{61, 60, 43, 103, UnitKindPage},
		{104, 60, 19, 122, UnitKindExample},
	} {
		selected, err := selectQualityCheckReviewerWorkingSet(scope, test.start, test.limit)
		if err != nil || len(selected) != test.count || selected[len(selected)-1].Index != test.end || selected[0].UnitKind != test.kind {
			t.Fatalf("start=%d selected=%v err=%v", test.start, selected, err)
		}
	}
	if _, _, err := ExportQualityCheckReviewerBundle("unused", &Catalog{}, QualityCheckReviewerBundleOptions{Locale: "zz-ZZ", SnapshotID: "snapshot", Limit: 61}); err == nil {
		t.Fatal("reviewer bundle accepted more than 60 TranslationUnits")
	}
}

func TestQualityCheckReviewerBundleDeterministicAndComplete(t *testing.T) {
	sourceRoot := repoRoot(t)
	catalog, err := BuildCatalog(sourceRoot)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	copyBundleAuthority(t, root, qualityCheckReviewerAuthorityPaths)
	copyQualityBundleTestTree(t, filepath.Join(sourceRoot, "data", "retranslation-runs", "uk-UA"), filepath.Join(root, "data", "retranslation-runs", "uk-UA"))
	copyQualityBundleTestFile(t, filepath.Join(sourceRoot, "data", "quality-check-snapshots", "uk-UA", "qc-001", "manifest.json"), filepath.Join(root, "data", "quality-check-snapshots", "uk-UA", "qc-001", "manifest.json"))
	copyQualityBundleTestFile(t, filepath.Join(sourceRoot, "locales", "uk-UA", "glossary.yaml"), filepath.Join(root, "locales", "uk-UA", "glossary.yaml"))
	var snapshot QualityCheckSnapshotManifest
	snapshotData, _ := os.ReadFile(filepath.Join(root, "data", "quality-check-snapshots", "uk-UA", "qc-001", "manifest.json"))
	if err := json.Unmarshal(snapshotData, &snapshot); err != nil {
		t.Fatal(err)
	}
	seenSource := map[string]bool{}
	for _, unit := range snapshot.Units {
		if !seenSource[unit.SourcePath] {
			seenSource[unit.SourcePath] = true
			copyQualityBundleTestFile(t, filepath.Join(sourceRoot, filepath.FromSlash(unit.SourcePath)), filepath.Join(root, filepath.FromSlash(unit.SourcePath)))
		}
	}
	first, manifest, err := ExportQualityCheckReviewerBundle(root, catalog, QualityCheckReviewerBundleOptions{Locale: "uk-UA", SnapshotID: "qc-001", StartIndex: 1, Limit: 60})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := ExportQualityCheckReviewerBundle(root, catalog, QualityCheckReviewerBundleOptions{Locale: "uk-UA", SnapshotID: "qc-001", StartIndex: 1, Limit: 60})
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("reviewer bundle is not deterministic: %v", err)
	}
	if manifest.UnitCount != 60 || manifest.WorkingSetKind != UnitKindPage || manifest.StartIndex != 1 || manifest.EndIndex != 60 {
		t.Fatalf("manifest=%+v", manifest)
	}
	files, err := ReadTransportBundle(first, 512, 128<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTransportBundleInventory(files, manifest.Files, true); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"quality-check.json", "formal/snapshot-manifest.json", "formal/glossary.yaml"} {
		if _, ok := files[required]; !ok {
			t.Fatalf("bundle missing %s", required)
		}
	}
	var context QualityCheckReviewerContext
	if err := json.Unmarshal(files["quality-check.json"], &context); err != nil {
		t.Fatal(err)
	}
	if len(context.Units) != 60 || context.Scope.PendingCount != 122 || context.Units[0].Source == "" || context.Units[0].CurrentTarget == "" || context.Units[0].RelevantInputPath == "" {
		t.Fatalf("incomplete reviewer context: units=%d pending=%d first=%+v", len(context.Units), context.Scope.PendingCount, context.Units[0])
	}
	if _, err := VerifyCurrentQualityCheckReviewerBundle(root, catalog, first); err != nil {
		t.Fatalf("current reviewer bundle rejected: %v", err)
	}
	authorityPath := filepath.Join(root, filepath.FromSlash(qualityCheckReviewerAuthorityPaths[0]))
	authority, _ := os.ReadFile(authorityPath)
	if err := os.WriteFile(authorityPath, append(authority, []byte("\n# changed after review export\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyCurrentQualityCheckReviewerBundle(root, catalog, first); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale reviewer bundle accepted: %v", err)
	}
}

func copyQualityBundleTestFile(t *testing.T, source, destination string) {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func copyQualityBundleTestTree(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
}
