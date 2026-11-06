package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotWriteOnceAfterDependencyCheck(t *testing.T) {
	root := t.TempDir()
	options := QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "isolated-write-once"}
	s := &QualityCheckSnapshotManifest{SchemaVersion: QualityCheckSnapshotSchemaVersion, Locale: options.Locale, SnapshotID: options.SnapshotID}
	final := filepath.Join(root, "data/quality-check-snapshots", options.Locale, options.SnapshotID)
	check := func() error { return os.MkdirAll(final, 0755) }
	if _, _, err := writeQualityCheckCandidateSnapshot(root, options, s, check); err == nil {
		t.Fatal("directory created during checks was overwritten")
	}
	if _, err := os.Lstat(filepath.Join(final, "manifest.json")); !os.IsNotExist(err) {
		t.Fatal("published into an existing destination", err)
	}
}

func TestCurrentValidationSnapshotRejectsIncompleteAndDuplicateScope(t *testing.T) {
	root, c, o, e := importedCommentProcessFixture(t)
	if r, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: o.Locale, BatchID: e.BatchID}); err != nil || r.ValidationPassed != 9 {
		t.Fatal(r, err)
	}
	for _, mode := range []string{"missing", "duplicate"} {
		t.Run(mode, func(t *testing.T) {
			copy := *c
			copy.TranslationScope = append([]string(nil), c.TranslationScope...)
			if mode == "missing" {
				copy.TranslationScope = copy.TranslationScope[:len(copy.TranslationScope)-1]
			} else {
				copy.TranslationScope = append(copy.TranslationScope, copy.TranslationScope[0])
			}
			if _, _, err := CreateQualityCheckCandidateSnapshot(root, &copy, QualityCheckSnapshotOptions{Locale: o.Locale, SnapshotID: "rejected-" + mode}); err == nil {
				t.Fatal("invalid exact package membership accepted")
			}
		})
	}
}
