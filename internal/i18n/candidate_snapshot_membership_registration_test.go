package i18n

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestSnapshotRevisionCatalogUsesExactFrozenAuthority(t *testing.T) {
	root, current, _ := packageSnapshotFixture(t)
	s, path, err := CreateQualityCheckCandidateSnapshot(root, current, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "revision-scope", ContentPackage: current.TranslationPackages[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	fallback := *current
	fallback.TranslationScope = []string{"different-predecessor-scope"}
	choice := selectedRetranslationUnit{manifest: RetranslationBatchUnit{RevisionFeedbackSource: CurrentValidationRevisionFeedback, PreviousSnapshotID: s.SnapshotID}}
	frozen, err := snapshotRevisionEvidenceCatalog(root, current, "zh-CN", choice, &fallback)
	if err != nil || !reflect.DeepEqual(frozen.TranslationScope, current.TranslationScope) || frozen.TranslationPackages[0].AuthoritySHA256 != s.ContentPackage.AuthoritySHA256 {
		t.Fatal("decoder used membership child instead of exact frozen package", err)
	}
	missing := choice
	missing.manifest.PreviousSnapshotID = "missing-snapshot"
	if _, err := snapshotRevisionEvidenceCatalog(root, current, "zh-CN", missing, &fallback); err == nil {
		t.Fatal("missing frozen Snapshot fell back to predecessor scope")
	}
	unregistered := *current
	unregistered.TranslationPackages, unregistered.HistoricalTranslationPackages = nil, nil
	if _, err := snapshotRevisionEvidenceCatalog(root, &unregistered, "zh-CN", choice, &fallback); err == nil {
		t.Fatal("unregistered frozen authority accepted")
	}
	s.ContentPackage.AuthoritySHA256 = sum([]byte("wrong historical package authority"))
	if err := writeTranslationJSON(filepath.Join(root, filepath.FromSlash(path)), s); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshotRevisionEvidenceCatalog(root, current, "zh-CN", choice, &fallback); err == nil {
		t.Fatal("wrong SHA fell back to same-ID package")
	}
	ordinary := selectedRetranslationUnit{}
	if got, err := snapshotRevisionEvidenceCatalog(root, current, "zh-CN", ordinary, &fallback); err != nil || got != &fallback {
		t.Fatal("ordinary candidate scope changed", err)
	}
}
