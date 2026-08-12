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

func TestPublishBundleStructureIntegrityAndRepeatability(t *testing.T) {
	root, catalog := publishTestCatalog(t)
	parent := t.TempDir()
	first := filepath.Join(parent, "release-a")
	second := filepath.Join(parent, "release-b")
	if err := publishBundle(root, catalog, publishOptions{Locale: "zh-CN", Output: first, PublishedAt: testPublishedAt}); err != nil {
		t.Fatal(err)
	}
	if err := publishBundle(root, catalog, publishOptions{Locale: "zh-CN", Output: second, PublishedAt: testPublishedAt}); err != nil {
		t.Fatal(err)
	}
	verifyPublishedBundle(t, first)
	verifyPublishedBundle(t, second)
	assertTreeEqual(t, first, second)

	firstManifest, err := os.ReadFile(filepath.Join(first, "release.json"))
	if err != nil {
		t.Fatal(err)
	}
	secondManifest, err := os.ReadFile(filepath.Join(second, "release.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstManifest, secondManifest) {
		t.Fatal("release.json differs between identical publishes")
	}
	firstSums, err := os.ReadFile(filepath.Join(first, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	secondSums, err := os.ReadFile(filepath.Join(second, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstSums, secondSums) {
		t.Fatal("SHA256SUMS differs between identical publishes")
	}
}

func TestPublishFailureCleansStaging(t *testing.T) {
	root, catalog := publishTestCatalog(t)
	parent := t.TempDir()
	output := filepath.Join(parent, "failed-release")
	original := buildProductionBinary
	buildProductionBinary = func(string, string, string) error { return errors.New("injected binary build failure") }
	t.Cleanup(func() { buildProductionBinary = original })
	if err := publishBundle(root, catalog, publishOptions{Locale: "zh-CN", Output: output, PublishedAt: testPublishedAt}); err == nil || !strings.Contains(err.Error(), "injected binary build failure") {
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
	if manifest.Locale != "zh-CN" || manifest.PublishedAt != testPublishedAt || manifest.UpstreamCommit != tour.FrozenUpstreamCommit || manifest.UpstreamCommitTime != tour.FrozenUpstreamCommitTime || manifest.Pages != 103 || manifest.Articles != 7 || manifest.ExecutionTransport != "http-playground-proxy" || manifest.ExecutionProvider != "go.dev" || manifest.LocalSocketEnabled {
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
