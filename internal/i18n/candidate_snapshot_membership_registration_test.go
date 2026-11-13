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
	bridge := choice
	bridge.manifest.RevisionFeedbackSource = "quality_check"
	bridge.manifest.RepresentationBridge = &GlossaryArchiveReference{Path: "frozen-bridge.json", SHA256: sum([]byte("bridge"))}
	if got, err := snapshotRevisionEvidenceCatalog(root, current, "zh-CN", bridge, &fallback); err != nil || !reflect.DeepEqual(got.TranslationScope, frozen.TranslationScope) {
		t.Fatal("bridge decoder used membership child instead of frozen Snapshot package", err)
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
	if _, err := snapshotRevisionEvidenceCatalog(root, &unregistered, "zh-CN", bridge, &fallback); err == nil {
		t.Fatal("bridge used an unregistered frozen authority")
	}
	s.ContentPackage.AuthoritySHA256 = sum([]byte("wrong historical package authority"))
	if err := writeTranslationJSON(filepath.Join(root, filepath.FromSlash(path)), s); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshotRevisionEvidenceCatalog(root, current, "zh-CN", choice, &fallback); err == nil {
		t.Fatal("wrong SHA fell back to same-ID package")
	}
	if _, err := snapshotRevisionEvidenceCatalog(root, current, "zh-CN", bridge, &fallback); err == nil {
		t.Fatal("bridge wrong SHA fell back to same-ID package")
	}
	ordinary := selectedRetranslationUnit{}
	if got, err := snapshotRevisionEvidenceCatalog(root, current, "zh-CN", ordinary, &fallback); err != nil || got != &fallback {
		t.Fatal("ordinary candidate scope changed", err)
	}
}

func TestSnapshotBridgeRevisionCatalogAfterMembershipSuccessor(t *testing.T) {
	root, old, id := packageSnapshotFixture(t)
	current, added := expandedMachinePackage(t, root, old, 2)
	sibling, err := HistoricalTranslationPackageCatalog(current, added.ID, added.AuthoritySHA256)
	if err != nil {
		t.Fatal(err)
	}
	checkpointGenerate(t, root, sibling, sibling.TranslationScope)
	s, _, err := CreateQualityCheckCandidateSnapshot(root, current, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "bridge-after-composition", ContentPackage: current.TranslationPackages[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	declaration := TranslationPackageMembership{Predecessors: []TranslationPackagePredecessor{
		{ID: old.TranslationPackages[0].ID, AuthoritySHA256: old.TranslationPackages[0].AuthoritySHA256},
		{ID: added.ID, AuthoritySHA256: added.AuthoritySHA256},
	}}
	data := mustJSON(declaration)
	path := "fixtures/bridge-membership.json"
	writeSnapshotSource(t, root, path, data)
	declaration.References = []GlossaryArchiveReference{{Path: path, SHA256: sum(data)}}
	current.TranslationPackages[0].Membership = &declaration
	members, check, err := snapshotMembershipCatalogs(root, current)
	if err != nil || len(members) != 3 || len(members[id].TranslationScope) != 1 {
		t.Fatal("invalid membership fixture", err)
	}
	if err := check(); err != nil {
		t.Fatal(err)
	}
	choice := selectedRetranslationUnit{manifest: RetranslationBatchUnit{
		UnitID: id, PreviousSnapshotID: s.SnapshotID, RevisionFeedbackSource: "quality_check",
		RepresentationBridge: &GlossaryArchiveReference{Path: "frozen-bridge.json", SHA256: sum([]byte("bridge"))},
	}}
	frozen, err := snapshotRevisionEvidenceCatalog(root, current, "zh-CN", choice, members[id])
	if err != nil || !reflect.DeepEqual(frozen.TranslationScope, s.ContentPackage.UnitIDs) || frozen.TranslationPackages[0].AuthoritySHA256 != s.ContentPackage.AuthoritySHA256 {
		t.Fatal("composed bridge revision lost its exact frozen scope", err)
	}
	if len(members[id].TranslationScope) != 1 || len(current.TranslationScope) != 3 {
		t.Fatal("historical/current caller scopes mutated")
	}
	t.Run("missing-snapshot", func(t *testing.T) {
		bad := choice
		bad.manifest.PreviousSnapshotID = "unrecorded-snapshot"
		if _, err := snapshotRevisionEvidenceCatalog(root, current, "zh-CN", bad, members[id]); err == nil {
			t.Fatal("bridge guessed a scope without its Snapshot")
		}
	})
	t.Run("unrelated-package-authority", func(t *testing.T) {
		bad := *current
		bad.TranslationPackages = append([]TranslationPackageDescriptor(nil), current.TranslationPackages...)
		bad.TranslationPackages[0].AuthoritySHA256 = sum([]byte("unrelated successor"))
		if _, err := snapshotRevisionEvidenceCatalog(root, &bad, "zh-CN", choice, members[id]); err == nil {
			t.Fatal("unrelated same-name package substituted for frozen authority")
		}
	})
	t.Run("substituted-ordered-scope", func(t *testing.T) {
		bad := *current
		bad.TranslationPackages = append([]TranslationPackageDescriptor(nil), current.TranslationPackages...)
		bad.TranslationPackages[0].Units = append([]AdaptedTranslationUnit(nil), current.TranslationPackages[0].Units...)
		bad.TranslationPackages[0].Units[0], bad.TranslationPackages[0].Units[1] = bad.TranslationPackages[0].Units[1], bad.TranslationPackages[0].Units[0]
		if _, err := snapshotRevisionEvidenceCatalog(root, &bad, "zh-CN", choice, members[id]); err == nil {
			t.Fatal("same-authority scope substitution accepted")
		}
	})
	t.Run("wrong-membership-predecessor", func(t *testing.T) {
		bad := *current
		bad.TranslationPackages = append([]TranslationPackageDescriptor(nil), current.TranslationPackages...)
		decl := declaration
		decl.Predecessors = append([]TranslationPackagePredecessor(nil), declaration.Predecessors...)
		decl.Predecessors[0].AuthoritySHA256 = sum([]byte("wrong predecessor"))
		bad.TranslationPackages[0].Membership = &decl
		if _, _, err := snapshotMembershipCatalogs(root, &bad); err == nil {
			t.Fatal("unregistered membership predecessor accepted")
		}
	})
	t.Run("tampered-membership", func(t *testing.T) {
		writeSnapshotSource(t, root, path, append(append([]byte(nil), data...), '\n'))
		defer writeSnapshotSource(t, root, path, data)
		if _, _, err := snapshotMembershipCatalogs(root, current); err == nil {
			t.Fatal("tampered membership dependency accepted")
		}
	})
}
