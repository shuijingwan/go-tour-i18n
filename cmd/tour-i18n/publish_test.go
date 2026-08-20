package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/tour"
)

const testPublishedAt = "2026-08-12T00:00:00Z"

func TestPublishCLIRequiresLocaleAndOutput(t *testing.T) {
	if _, err := parsePublishOptions([]string{"--output", "bundle"}); err == nil || !strings.Contains(err.Error(), "--locale is required") {
		t.Fatalf("missing locale error = %v", err)
	}
	if _, err := parsePublishOptions([]string{"--locale", "zh-CN"}); err == nil || !strings.Contains(err.Error(), "--output is required") {
		t.Fatalf("missing output error = %v", err)
	}
	if _, err := parsePublishOptions([]string{"--locale", "zh-CN", "--output", "bundle"}); err == nil || !strings.Contains(err.Error(), "--published-at is required") {
		t.Fatalf("missing published-at error = %v", err)
	}
	if _, err := parsePublishOptions([]string{"--locale", "zh-CN", "--output", "bundle", "--published-at", "2026-08-12T00:00:00+08:00"}); err == nil || !strings.Contains(err.Error(), "RFC 3339 UTC") {
		t.Fatalf("non-UTC published-at error = %v", err)
	}
	if err := run([]string{"publish", "--output", "bundle"}); err == nil || !strings.Contains(err.Error(), "--locale is required") {
		t.Fatalf("CLI missing locale error = %v", err)
	}
	if err := run([]string{"publish", "--locale", "zh-CN"}); err == nil || !strings.Contains(err.Error(), "--output is required") {
		t.Fatalf("CLI missing output error = %v", err)
	}
}

func TestProductionBuildEnvDisablesCGOWithoutChangingOtherSettings(t *testing.T) {
	original, hadOriginal := os.LookupEnv("CGO_ENABLED")
	if err := os.Setenv("CGO_ENABLED", "1"); err != nil {
		t.Fatal(err)
	}
	if hadOriginal {
		t.Cleanup(func() { _ = os.Setenv("CGO_ENABLED", original) })
	} else {
		t.Cleanup(func() { _ = os.Unsetenv("CGO_ENABLED") })
	}

	env := productionBuildEnv()
	var cgoValues []string
	for _, value := range env {
		if strings.HasPrefix(value, "CGO_ENABLED=") {
			cgoValues = append(cgoValues, value)
		}
	}
	if !reflect.DeepEqual(cgoValues, []string{"CGO_ENABLED=0"}) {
		t.Fatalf("CGO_ENABLED entries = %v, want [CGO_ENABLED=0]", cgoValues)
	}
}

func TestPublishRejectsExistingOutputWithoutChangingIt(t *testing.T) {
	root, catalog := publishTestCatalog(t)
	output := filepath.Join(t.TempDir(), "existing-bundle")
	if err := os.Mkdir(output, 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(output, "keep.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	err := publishBundle(root, catalog, publishOptions{Locale: "zh-CN", Output: output, PublishedAt: testPublishedAt})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("existing output error = %v", err)
	}
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != "keep" {
		t.Fatalf("existing output changed: data=%q err=%v", data, err)
	}
}

func TestPublishRequiresCompleteTranslationUnitWorkflow(t *testing.T) {
	root, catalog, pendingUnit := incompletePublishTestCatalog(t)
	parent := t.TempDir()
	first := filepath.Join(parent, "release-a")
	err := publishBundle(root, catalog, publishOptions{Locale: "zh-CN", Output: first, PublishedAt: testPublishedAt})
	if err == nil || !strings.Contains(err.Error(), "workflow translation units") || !strings.Contains(err.Error(), pendingUnit+"=pending") {
		t.Fatalf("incomplete workflow publish error = %v", err)
	}
	if _, statErr := os.Lstat(first); !os.IsNotExist(statErr) {
		t.Fatalf("blocked publish created output: %v", statErr)
	}
}

func TestPublishFailureCleansStaging(t *testing.T) {
	root, catalog, _ := incompletePublishTestCatalog(t)
	parent := t.TempDir()
	output := filepath.Join(parent, "failed-release")
	if err := publishBundle(root, catalog, publishOptions{Locale: "zh-CN", Output: output, PublishedAt: testPublishedAt}); err == nil || !strings.Contains(err.Error(), "workflow translation units") {
		t.Fatalf("publish failure = %v", err)
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		t.Fatalf("failed publish created output: %v", err)
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".staging-") {
			t.Fatalf("staging directory remains after failure: %s", entry.Name())
		}
	}
}

func TestValidatePublishProjectionUsesDynamicWorkflowCounts(t *testing.T) {
	_, catalog := publishTestCatalog(t)
	total, pages, examples, err := i18n.LocaleWorkflowUnitCounts(catalog)
	if err != nil {
		t.Fatal(err)
	}
	projection := &i18n.LocaleProjection{
		UnitCount: total, Ready: total, PageCount: pages, ExampleCount: examples, ArticleCount: expectedPublishArticles,
	}
	if err := validatePublishProjection(catalog, projection); err != nil {
		t.Fatalf("complete dynamic workflow rejected: %v", err)
	}
	projection.Ready--
	projection.Pending++
	if err := validatePublishProjection(catalog, projection); err == nil {
		t.Fatal("incomplete dynamic workflow accepted")
	}
}

func TestBuildReleaseManifestSchemaV2UsesWorkflowCounts(t *testing.T) {
	_, catalog := publishTestCatalog(t)
	total, pages, examples, err := i18n.LocaleWorkflowUnitCounts(catalog)
	if err != nil {
		t.Fatal(err)
	}
	projection := &i18n.LocaleProjection{
		UnitCount: total, PageCount: pages, ExampleCount: examples, ArticleCount: expectedPublishArticles,
	}
	manifest, err := buildReleaseManifest(catalog, projection, publishOptions{Locale: "zh-CN", PublishedAt: testPublishedAt})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 2 || manifest.TranslationUnits != 122 || manifest.Pages != 103 || manifest.EligibleExamples != 19 || manifest.Articles != 7 {
		t.Fatalf("current workflow manifest = %+v", manifest)
	}

	pageSource := []byte("* Page\n")
	exampleSource := []byte("package main\n\n// Explain this example.\nfunc main() {}\n")
	pageHash := fmt.Sprintf("%x", sha256.Sum256(pageSource))
	exampleHash := fmt.Sprintf("%x", sha256.Sum256(exampleSource))
	smallCatalog := &i18n.Catalog{
		Pages:    []i18n.Page{{ID: "small/1", Article: "small.article", SectionNumber: 1, Source: pageSource, SourceSHA256: pageHash}},
		Examples: []i18n.Example{{ID: "example:small/example.go", SourcePath: "_content/tour/small/example.go", Source: exampleSource, SourceSHA256: exampleHash}},
	}
	smallProjection := &i18n.LocaleProjection{UnitCount: 2, PageCount: 1, ExampleCount: 1, ArticleCount: 1}
	smallManifest, err := buildReleaseManifest(smallCatalog, smallProjection, publishOptions{Locale: "test", PublishedAt: testPublishedAt})
	if err != nil {
		t.Fatal(err)
	}
	if smallManifest.TranslationUnits != 2 || smallManifest.Pages != 1 || smallManifest.EligibleExamples != 1 || smallManifest.Articles != 1 {
		t.Fatalf("dynamic workflow manifest = %+v", smallManifest)
	}
	smallProjection.ExampleCount = 0
	if _, err := buildReleaseManifest(smallCatalog, smallProjection, publishOptions{}); err == nil || !strings.Contains(err.Error(), "do not match projection") {
		t.Fatalf("mismatched workflow/projection error = %v", err)
	}
}

func TestValidateBundleAcceptsSchemaV2AndChecksums(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "_content"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "tour"), []byte("test binary\n"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := releaseManifest{
		SchemaVersion: releaseManifestSchemaVersion, Locale: "zh-CN", TranslationUnits: 122,
		Pages: 103, EligibleExamples: 19, Articles: 7,
	}
	if err := writeReleaseManifest(root, manifest); err != nil {
		t.Fatal(err)
	}
	checksums, err := bundleChecksums(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "SHA256SUMS"), checksums, 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateBundle(root, manifest, checksums); err != nil {
		t.Fatalf("schema v2 bundle validation failed: %v", err)
	}
}

func publishTestCatalog(t *testing.T) (string, *i18n.Catalog) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	current, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := i18n.ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := i18n.HydrateCatalogSources(catalog, current); err != nil {
		t.Fatal(err)
	}
	return root, catalog
}

func incompletePublishTestCatalog(t *testing.T) (string, *i18n.Catalog, string) {
	t.Helper()
	sourceRoot, catalog := publishTestCatalog(t)
	root := t.TempDir()
	for _, relative := range []string{"_content", filepath.Join("locales", "zh-CN")} {
		if err := os.CopyFS(filepath.Join(root, relative), os.DirFS(filepath.Join(sourceRoot, relative))); err != nil {
			t.Fatal(err)
		}
	}
	return root, catalog, makeEligibleExamplePending(t, root, catalog)
}

func makeEligibleExamplePending(t *testing.T, root string, catalog *i18n.Catalog) string {
	t.Helper()
	statusPath := filepath.Join(root, "locales", "zh-CN", "status.tsv")
	statuses, err := i18n.ReadStatuses(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	pendingUnit := ""
	for _, status := range statuses {
		unit, unitErr := catalog.Unit(status.UnitID)
		if unitErr != nil {
			t.Fatal(unitErr)
		}
		if unit.Kind == i18n.UnitKindExample {
			pendingUnit = status.UnitID
			break
		}
	}
	if pendingUnit == "" {
		t.Fatal("workflow has no eligible Example")
	}
	data, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	changed := false
	for index, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) != 7 || fields[0] != pendingUnit {
			continue
		}
		fields[1], fields[2], fields[4], fields[5], fields[6] = "pending", "0", "", "", ""
		lines[index] = strings.Join(fields, "\t")
		changed = true
		break
	}
	if !changed {
		t.Fatalf("status row for %s not found", pendingUnit)
	}
	if err := os.WriteFile(statusPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatal(err)
	}
	return pendingUnit
}

func verifyPublishedBundle(t *testing.T, root string) {
	t.Helper()
	for _, path := range []string{"bin/tour", "_content", "release.json", "SHA256SUMS"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
			t.Fatalf("bundle missing %s: %v", path, err)
		}
	}
	manifestData, err := os.ReadFile(filepath.Join(root, "release.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest releaseManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != releaseManifestSchemaVersion || manifest.Locale != "zh-CN" || manifest.PublishedAt != testPublishedAt || manifest.UpstreamCommit != tour.FrozenUpstreamCommit || manifest.UpstreamCommitTime != tour.FrozenUpstreamCommitTime || manifest.TranslationUnits != 122 || manifest.Pages != 103 || manifest.EligibleExamples != 19 || manifest.Articles != 7 || manifest.ExecutionTransport != "http-playground-proxy" || manifest.ExecutionProvider != "play.golang.org" || manifest.LocalSocketEnabled {
		t.Fatalf("unexpected release manifest: %+v", manifest)
	}
	metadataData, err := os.ReadFile(filepath.Join(root, "_content", "tour", "site-metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	var metadata tour.SiteMetadata
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.Locale != manifest.Locale || metadata.PublishedAt != manifest.PublishedAt || metadata.UpstreamCommit != manifest.UpstreamCommit || metadata.UpstreamCommitTime != manifest.UpstreamCommitTime || metadata.Pages != manifest.Pages || metadata.Articles != manifest.Articles {
		t.Fatalf("site metadata = %+v, want manifest values", metadata)
	}
	if metadata.Development || bytes.Contains(metadataData, []byte(`"development"`)) {
		t.Fatalf("production site metadata unexpectedly contains development state: %s", metadataData)
	}
	forbidden := map[string]bool{"locales": true, "status.tsv": true, "candidate": true, "translation-runs": true, "attempt": true}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != root && forbidden[entry.Name()] {
			return errors.New("forbidden bundle artifact: " + path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	checksums, err := os.ReadFile(filepath.Join(root, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	var previous string
	for _, line := range strings.Split(strings.TrimSpace(string(checksums)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			t.Fatalf("malformed checksum line %q", line)
		}
		if fields[1] == "SHA256SUMS" || filepath.IsAbs(fields[1]) || fields[1] < previous {
			t.Fatalf("unstable checksum path %q", fields[1])
		}
		previous = fields[1]
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(fields[1])))
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != fields[0] {
			t.Fatalf("checksum mismatch for %s: got %s want %s", fields[1], got, fields[0])
		}
	}
}

func assertTreeEqual(t *testing.T, first, second string) {
	t.Helper()
	files := func(root string) []string {
		var paths []string
		if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			paths = append(paths, filepath.ToSlash(rel))
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		sort.Strings(paths)
		return paths
	}
	firstPaths, secondPaths := files(first), files(second)
	if strings.Join(firstPaths, "\n") != strings.Join(secondPaths, "\n") {
		t.Fatalf("bundle file sets differ: %v != %v", firstPaths, secondPaths)
	}
	for _, rel := range firstPaths {
		left, err := os.ReadFile(filepath.Join(first, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		right, err := os.ReadFile(filepath.Join(second, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(left, right) {
			t.Fatalf("bundle file differs: %s", rel)
		}
	}
}
