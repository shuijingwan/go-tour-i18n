package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const batchTestHead = "0123456789abcdef0123456789abcdef01234567"

func installPublishBatchFakes(t *testing.T) {
	t.Helper()
	originalClock := publishBatchClock
	originalIdentity := publishBatchRepositoryIdentity
	originalBundle := publishBatchBundle
	t.Cleanup(func() {
		publishBatchClock = originalClock
		publishBatchRepositoryIdentity = originalIdentity
		publishBatchBundle = originalBundle
	})
}

func TestPublishBatchPreflightRejectsDuplicateInvalidAndDirtyBeforePublish(t *testing.T) {
	root, catalog := publishTestCatalog(t)
	installPublishBatchFakes(t)
	outputRoot := t.TempDir()
	publishCalls := 0
	publishBatchBundle = func(string, *i18n.Catalog, publishOptions) error { publishCalls++; return nil }
	publishBatchRepositoryIdentity = func(string) (string, error) { return batchTestHead, nil }
	for name, locales := range map[string][]string{
		"duplicate": {"de-DE", "de-DE"},
		"invalid":   {"not/a-locale"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := executePublishBatch(root, catalog, publishBatchOptions{OutputRoot: outputRoot, Locales: locales}, &bytes.Buffer{})
			if err == nil {
				t.Fatal("invalid batch preflight passed")
			}
		})
	}
	publishBatchRepositoryIdentity = func(string) (string, error) { return "", errors.New("repository working tree must be clean") }
	if _, err := executePublishBatch(root, catalog, publishBatchOptions{OutputRoot: outputRoot, Locales: []string{"de-DE"}}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "clean") {
		t.Fatalf("dirty repository error = %v", err)
	}
	if publishCalls != 0 {
		t.Fatalf("preflight failures invoked publish %d times", publishCalls)
	}
}

func TestPublishBatchGetsHeadOnceAndFreshTimestampPerLocale(t *testing.T) {
	root, catalog := publishTestCatalog(t)
	installPublishBatchFakes(t)
	outputRoot := t.TempDir()
	headCalls := 0
	publishBatchRepositoryIdentity = func(string) (string, error) { headCalls++; return batchTestHead, nil }
	times := []time.Time{
		time.Date(2026, 9, 11, 20, 30, 1, 0, time.UTC),
		time.Date(2026, 9, 11, 20, 31, 46, 0, time.UTC),
		time.Date(2026, 9, 11, 20, 33, 25, 0, time.UTC),
	}
	clockCalls := 0
	publishBatchClock = func() time.Time { value := times[clockCalls]; clockCalls++; return value }
	var options []publishOptions
	publishBatchBundle = func(_ string, _ *i18n.Catalog, option publishOptions) error {
		options = append(options, option)
		return nil
	}
	var output bytes.Buffer
	result, err := executePublishBatch(root, catalog, publishBatchOptions{
		OutputRoot: outputRoot, Locales: []string{"de-DE", "es-ES", "fr-FR"}, JSON: true,
	}, &output)
	if err != nil {
		t.Fatal(err)
	}
	if headCalls != 1 || clockCalls != 3 {
		t.Fatalf("head calls=%d clock calls=%d", headCalls, clockCalls)
	}
	if got := []string{options[0].PublishedAt, options[1].PublishedAt, options[2].PublishedAt}; !reflect.DeepEqual(got, []string{"2026-09-11T20:30:01Z", "2026-09-11T20:31:46Z", "2026-09-11T20:33:25Z"}) {
		t.Fatalf("published_at values = %v", got)
	}
	for index, item := range result.Releases {
		if item.ReleaseDir != options[index].Output || !strings.Contains(item.ReleaseDir, item.Locale+"-0123456789ab") {
			t.Fatalf("release identity mismatch: item=%+v option=%+v", item, options[index])
		}
	}
	var decoded publishBatchResult
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("JSON summary is invalid: %v\n%s", err, output.Bytes())
	}
	if !reflect.DeepEqual(decoded, *result) {
		t.Fatalf("JSON summary = %+v, want %+v", decoded, *result)
	}
}

func TestPublishBatchIsStrictSerialAndStopsAfterFailure(t *testing.T) {
	root, catalog := publishTestCatalog(t)
	installPublishBatchFakes(t)
	publishBatchRepositoryIdentity = func(string) (string, error) { return batchTestHead, nil }
	publishBatchClock = func() time.Time { return time.Date(2026, 9, 11, 20, 30, 1, 0, time.UTC) }
	var calls []string
	publishBatchBundle = func(_ string, _ *i18n.Catalog, option publishOptions) error {
		calls = append(calls, option.Locale)
		if option.Locale == "es-ES" {
			return errors.New("Chrome prerender failed")
		}
		return nil
	}
	result, err := executePublishBatch(root, catalog, publishBatchOptions{
		OutputRoot: t.TempDir(), Locales: []string{"de-DE", "es-ES", "fr-FR"},
	}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "es-ES") {
		t.Fatalf("batch failure = %v", err)
	}
	if !reflect.DeepEqual(calls, []string{"de-DE", "es-ES"}) {
		t.Fatalf("publish order = %v", calls)
	}
	if got := []string{result.Releases[0].Result, result.Releases[1].Result}; !reflect.DeepEqual(got, []string{"PASS", "FAILED"}) {
		t.Fatalf("partial results = %v", got)
	}
}

func TestPublishBatchNeverOverwritesExistingOutput(t *testing.T) {
	root, catalog := publishTestCatalog(t)
	installPublishBatchFakes(t)
	outputRoot := t.TempDir()
	publishBatchRepositoryIdentity = func(string) (string, error) { return batchTestHead, nil }
	publishBatchClock = func() time.Time { return time.Date(2026, 9, 11, 20, 30, 1, 0, time.UTC) }
	existing := filepath.Join(outputRoot, "go-tour-release-20260911T203001Z-de-DE-0123456789ab")
	if err := os.Mkdir(existing, 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(existing, "keep")
	if err := os.WriteFile(marker, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	called := false
	publishBatchBundle = func(string, *i18n.Catalog, publishOptions) error { called = true; return nil }
	if _, err := executePublishBatch(root, catalog, publishBatchOptions{OutputRoot: outputRoot, Locales: []string{"de-DE"}}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("existing output error = %v", err)
	}
	if called {
		t.Fatal("existing output reached publishBundle")
	}
	if data, err := os.ReadFile(marker); err != nil || string(data) != "keep" {
		t.Fatalf("existing output changed: %q %v", data, err)
	}
}

func TestPublishBatchSourceHasNoProductionMutation(t *testing.T) {
	source, err := os.ReadFile("publish_batch.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"deploy-production", "maintenance-production", "production-cdn", "indexnow", "ssh"} {
		if strings.Contains(strings.ToLower(string(source)), forbidden) {
			t.Fatalf("publish-batch contains production side effect %q", forbidden)
		}
	}
}
