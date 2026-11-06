package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Exercise the real revision Bundle/current-check consumers with independent
// invocation lifetimes. Cache failure is never a formal failure; formal evidence
// mutation is always a failure even after the exact previous bytes were warm.
func TestRevisionBundlePersistentCacheFallback(t *testing.T) {
	root, catalog, options := revisionGenerationFixture(t)
	want, manifest, err := ExportTranslationUnitGenerationBundle(root, catalog, options)
	if err != nil {
		t.Fatal(err)
	}
	cacheHome := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	check := func() {
		t.Helper()
		done := contentidentity.BeginMemoization()
		got, current, err := ExportTranslationUnitGenerationBundle(root, catalog, options)
		done()
		if err != nil || !bytes.Equal(got, want) || current.InputIdentitySHA256 != manifest.InputIdentitySHA256 {
			t.Fatal("cache changed canonical export", err)
		}
		done = contentidentity.BeginMemoization()
		checked, err := VerifyCurrentTranslationUnitGenerationBundle(root, catalog, got)
		done()
		if err != nil || checked.InputIdentitySHA256 != manifest.InputIdentitySHA256 {
			t.Fatal("cache changed independent current-check", err)
		}
	}
	for _, mode := range []string{"cold", "warm", "corrupt", "missing", "unwritable", "delete-then-recompute"} {
		t.Run(mode, func(t *testing.T) {
			switch mode {
			case "corrupt":
				err := filepath.WalkDir(cacheHome, func(path string, d os.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if !d.IsDir() {
						return os.WriteFile(path, []byte("interrupted cache payload"), 0600)
					}
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			case "missing", "delete-then-recompute":
				if err := os.RemoveAll(filepath.Join(cacheHome, "go-tour-i18n")); err != nil {
					t.Fatal(err)
				}
			case "unwritable":
				blocked := filepath.Join(t.TempDir(), "not-a-directory")
				if err := os.WriteFile(blocked, []byte("cache unavailable"), 0600); err != nil {
					t.Fatal(err)
				}
				t.Setenv("XDG_CACHE_HOME", blocked)
			}
			check()
		})
	}
	snapshot, err := readQualityCheckSnapshot(root, options.Locale, "qc-001", false, catalog)
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]string{
		"Snapshot":   "data/quality-check-snapshots/zh-CN/qc-001/manifest.json",
		"QC":         "data/quality-check-snapshots/zh-CN/qc-001/quality-check-results.json",
		"candidate":  snapshot.Units[0].CandidatePath,
		"validation": snapshot.Units[0].ValidationPath,
		"glossary":   "locales/zh-CN/glossary.yaml",
	}
	for name, path := range paths {
		t.Run("fresh-"+name+"-drift", func(t *testing.T) {
			absolute := filepath.Join(root, path)
			original, err := os.ReadFile(absolute)
			if err != nil {
				t.Fatal(err)
			}
			defer os.WriteFile(absolute, original, 0600)
			changed := append(bytes.Clone(original), ' ')
			if name == "QC" {
				var results QualityCheckResults
				if err := json.Unmarshal(original, &results); err != nil {
					t.Fatal(err)
				}
				results.Results[0].Rating = "A"
				changed, _ = json.Marshal(results)
			}
			if err := os.WriteFile(absolute, changed, 0600); err != nil {
				t.Fatal(err)
			}
			done := contentidentity.BeginMemoization()
			_, err = VerifyCurrentTranslationUnitGenerationBundle(root, catalog, want)
			done()
			if err == nil {
				t.Fatal("warm cache concealed formal evidence drift")
			}
		})
	}
}

func TestGenerationArchiveMalformedPersistentIndexFallback(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	archive := observedArchiveFixture(t, []byte("exact frozen bytes"))
	want, files, err := decodeGenerationBundle(archive)
	if err != nil {
		t.Fatal(err)
	}
	done := contentidentity.BeginMemoization()
	_, _, err = readGenerationBundle(archive)
	done()
	if err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob(filepath.Join(os.Getenv("XDG_CACHE_HOME"), "go-tour-i18n", "deterministic-v1", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Skip("persistent exact-image cache unavailable")
	}
	found := false
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		head, payload, ok := bytes.Cut(b, []byte("\n"))
		if !ok || !bytes.Contains(payload, []byte("\"Members\"")) {
			continue
		}
		var header map[string]any
		if err := json.Unmarshal(head, &header); err != nil {
			t.Fatal(err)
		}
		bad := []byte("invalid decoded archive representation")
		header["Bytes"], header["OutputSHA256"] = len(bad), sum(bad)
		head, _ = json.Marshal(header)
		if err := os.WriteFile(path, append(append(head, '\n'), bad...), 0600); err != nil {
			t.Fatal(err)
		}
		found = true
	}
	if !found {
		t.Fatal("archive transform entry missing")
	}
	done = contentidentity.BeginMemoization()
	got, actual, err := readGenerationBundle(archive)
	done()
	if err != nil || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(actual, files) {
		t.Fatal("damaged derived index became a workflow dependency", err)
	}
}
