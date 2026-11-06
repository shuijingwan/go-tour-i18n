package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func sourceEnvironmentFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	from := repoRoot(t)
	err := filepath.WalkDir(filepath.Join(from, "_content/tour"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(from, path)
		target := filepath.Join(root, relative)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(from, "UPSTREAM_MANIFEST.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "UPSTREAM_MANIFEST.tsv"), b, 0644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestSourceAuthorityEnvironmentPersistent(t *testing.T) {
	root := sourceEnvironmentFixture(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	start := time.Now()
	want, err := buildCatalog(root, false, true)
	t.Logf("CANONICAL_REFERENCE_SECONDS=%.6f", time.Since(start).Seconds())
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	run := func() *Catalog {
		t.Helper()
		done := contentidentity.BeginMemoization()
		got, err := observedSourceCatalog(root, false, true, func(o sourceCatalogObservation) (*Catalog, error) {
			calls++
			return deriveSourceCatalog(o)
		})
		done()
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	first := run()
	if calls != 1 || !reflect.DeepEqual(first, want) {
		t.Fatal("cold source graph differs", calls)
	}
	first.Pages[0].Source[0] ^= 1
	first.Examples[0].ReferencedBy = append(first.Examples[0].ReferencedBy, "caller mutation")
	if got := run(); calls != 1 || !reflect.DeepEqual(got, want) {
		t.Fatal("warm graph aliased or reconstructed", calls)
	}
	if got, err := BuildSourceCatalog(root); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("public eager API differs", err)
	}

	path := filepath.Join(root, "_content/tour/basics.article")
	original, _ := os.ReadFile(path)
	if err := os.WriteFile(path, append(bytes.Clone(original), []byte("\nSynthetic observed prose.\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	next := run()
	if calls != 2 || reflect.DeepEqual(next, want) {
		t.Fatal("source change reused old graph", calls)
	}
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}

	// A fresh manifest comparison must reject changed source before using any
	// graph; it is not a cached PASS from the original successful observation.
	goPath := filepath.Join(root, filepath.FromSlash(want.Examples[0].SourcePath))
	goBytes, _ := os.ReadFile(goPath)
	os.WriteFile(goPath, append(bytes.Clone(goBytes), []byte("\n// Changed source.\n")...), 0644)
	done := contentidentity.BeginMemoization()
	_, err = BuildSourceCatalog(root)
	done()
	os.WriteFile(goPath, goBytes, 0644)
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatal("fresh source/manifest mismatch accepted", err)
	}
	manifestPath := filepath.Join(root, "UPSTREAM_MANIFEST.tsv")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	wrongManifest := bytes.ReplaceAll(manifestBytes, []byte(want.Examples[0].SourceSHA256), []byte(strings.Repeat("0", 64)))
	if bytes.Equal(wrongManifest, manifestBytes) {
		t.Fatal("fixture manifest does not bind the selected example")
	}
	if err := os.WriteFile(manifestPath, wrongManifest, 0644); err != nil {
		t.Fatal(err)
	}
	done = contentidentity.BeginMemoization()
	_, err = BuildSourceCatalog(root)
	done()
	os.WriteFile(manifestPath, manifestBytes, 0644)
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatal("changed manifest reused an earlier source verdict", err)
	}

	// Directory discovery is an input, not only the previously known file list.
	extra := filepath.Join(root, "_content/tour/new-data.txt")
	os.WriteFile(extra, []byte("new observed member"), 0644)
	if got := run(); calls != 3 || !reflect.DeepEqual(got, want) {
		t.Fatal("discovery omitted from input identity", calls)
	}
	os.Remove(extra)

	// Re-observation closes the derivation phase, including a cold callback
	// that changes the repository after computing its immutable input graph.
	o, key, err := observeSourceCatalog(root, false, true)
	if err != nil || len(key) == 0 || len(o.Files) == 0 {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	done = contentidentity.BeginMemoization()
	_, err = observedSourceCatalog(root, false, true, func(o sourceCatalogObservation) (*Catalog, error) {
		c, err := deriveSourceCatalog(o)
		os.WriteFile(goPath, append(bytes.Clone(goBytes), []byte("\n// Concurrent source drift.\n")...), 0644)
		return c, err
	})
	done()
	os.WriteFile(goPath, goBytes, 0644)
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatal("concurrent mutation not detected", err)
	}
}

func TestSourceAuthorityObservedDerivationIsPure(t *testing.T) {
	root := sourceEnvironmentFixture(t)
	want, err := buildCatalog(root, false, true)
	if err != nil {
		t.Fatal(err)
	}
	o, _, err := observeSourceCatalog(root, false, true)
	if err != nil {
		t.Fatal(err)
	}
	// Only this disposable fixture is removed. Every nested parser, .play
	// reader and example/manifest consumer must use the owner's byte world.
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	got, err := deriveSourceCatalog(o)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("deterministic consumers reopened the repository", err)
	}
	t.Logf("SOURCE_ENVIRONMENT observed_files=%d derived_filesystem_reads=0 manifest_parses=1", len(o.Files))
}

func TestSourceAuthorityEnvironmentProcess(t *testing.T) {
	if root := os.Getenv("GO_TOUR_SOURCE_ENV_CHILD_ROOT"); root != "" {
		count := 0
		done := contentidentity.BeginMemoization()
		start := time.Now()
		got, err := observedSourceCatalog(root, false, true, func(o sourceCatalogObservation) (*Catalog, error) { count++; return deriveSourceCatalog(o) })
		done()
		if err != nil {
			t.Fatal(err)
		}
		encoded, _ := json.Marshal(got)
		fmt.Printf("SOURCE_ENVIRONMENT=%s\n", mustJSON(map[string]any{"builds": count, "sha256": sum(encoded), "seconds": time.Since(start).Seconds()}))
		return
	}
	root := sourceEnvironmentFixture(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var digest string
	for pass := 0; pass < 2; pass++ {
		cmd := exec.Command(os.Args[0], "-test.run=^TestSourceAuthorityEnvironmentProcess$", "-test.count=1")
		cmd.Env = append(os.Environ(), "GO_TOUR_SOURCE_ENV_CHILD_ROOT="+root)
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(string(b), err)
		}
		var row struct {
			Builds  int
			SHA256  string
			Seconds float64
		}
		found := false
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "SOURCE_ENVIRONMENT=") {
				if json.Unmarshal([]byte(strings.TrimPrefix(line, "SOURCE_ENVIRONMENT=")), &row) != nil {
					t.Fatal(line)
				}
				found = true
			}
		}
		if !found {
			t.Fatal(string(b))
		}
		if pass == 0 {
			digest = row.SHA256
			if row.Builds != 1 {
				t.Fatal("cold builder", row)
			}
		}
		if pass == 1 && (row.Builds != 0 || row.SHA256 != digest) {
			t.Fatal("cross-command reuse missing", row)
		}
		t.Logf("SOURCE_AUTHORITY phase=%d builds=%d seconds=%.6f identity=%s", pass, row.Builds, row.Seconds, row.SHA256)
	}
}

func TestSourceAuthorityDerivedCacheFailureFallback(t *testing.T) {
	root := sourceEnvironmentFixture(t)
	want, err := buildCatalog(root, false, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	done := contentidentity.BeginMemoization()
	defer done()
	_, key, err := observeSourceCatalog(root, false, true)
	if err != nil {
		t.Fatal(err)
	}
	// A complete cache envelope may still contain an unusable component codec.
	// The component must recompute instead of turning cache storage into a gate.
	if _, err := contentidentity.Transform("source-catalog/environment/v1", key, func() ([]byte, error) { return []byte("invalid source graph codec"), nil }); err != nil {
		t.Fatal(err)
	}
	calls := 0
	got, err := observedSourceCatalog(root, false, true, func(o sourceCatalogObservation) (*Catalog, error) { calls++; return deriveSourceCatalog(o) })
	if err != nil || calls != 1 || !reflect.DeepEqual(got, want) {
		t.Fatal("derived cache failure blocked source reconstruction", calls, err)
	}
}

func TestSourceAuthorityNonsemanticObservationFallback(t *testing.T) {
	root := sourceEnvironmentFixture(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	want, err := buildCatalog(root, false, true)
	if err != nil {
		t.Fatal(err)
	}
	done := contentidentity.BeginMemoization()
	got, err := observedSourceCatalog(root, false, true, func(o sourceCatalogObservation) (*Catalog, error) {
		c, err := deriveSourceCatalog(o)
		os.WriteFile(filepath.Join(root, "_content/tour/unreferenced.data"), []byte("unrelated discovery"), 0644)
		return c, err
	})
	done()
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("cache observation added a formal currentness gate", err)
	}
	// Lossy JSON path keys are never admitted. The canonical eager parser still
	// decides whether an otherwise irrelevant unsupported filename matters.
	name := filepath.Join(root, "_content/tour", string([]byte{0xff})+".data")
	if err := os.WriteFile(name, []byte("unsupported path encoding"), 0644); err != nil {
		t.Fatal(err)
	}
	done = contentidentity.BeginMemoization()
	calls := 0
	got, err = observedSourceCatalog(root, false, true, func(o sourceCatalogObservation) (*Catalog, error) { calls++; return deriveSourceCatalog(o) })
	done()
	if err != nil || calls != 0 || !reflect.DeepEqual(got, want) {
		t.Fatal("unsupported observation did not retain eager semantics", calls, err)
	}
}
