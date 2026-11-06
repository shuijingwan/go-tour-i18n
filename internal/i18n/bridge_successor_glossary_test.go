package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHistoricalBridgeFrozenGlossary(t *testing.T) {
	old := []byte("locale: zh-CN\nmandatory:\n  Go: Go\n")
	sha := sum(old)
	canonical := glossaryReviewGlossaryPath("zh-CN")
	archive, err := glossaryArchivePath("zh-CN", sha)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"same-current", "legal-successor", "missing-archive", "tampered-archive",
		"wrong-filename", "other-locale-archive", "arbitrary-same-hash",
		"archive-symlink", "archive-parent-symlink", "archive-directory",
		"wrong-frozen-sha", "wrong-archive-ref", "wrong-locale-ref", "other-dependency-drift",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeSnapshotSource(t, root, canonical, old)
			writeSnapshotSource(t, root, archive, old)
			receipt := []byte("immutable bridge receipt fixture\n")
			bridge := GlossaryArchiveReference{Path: "data/quality-check-reopens/zh-CN/fixture.json", SHA256: sum(receipt)}
			writeSnapshotSource(t, root, bridge.Path, receipt)
			lineage := bridgeResponseLineage{Bridge: bridge, Dependencies: []GlossaryArchiveReference{bridge, {Path: canonical, SHA256: sha}}}
			if name != "same-current" {
				writeSnapshotSource(t, root, canonical, append(bytes.Clone(old), []byte("# current successor\n")...))
			}
			removeArchive := func() {
				t.Helper()
				if err := os.Remove(filepath.Join(root, archive)); err != nil {
					t.Fatal(err)
				}
			}
			switch name {
			case "missing-archive":
				removeArchive()
			case "tampered-archive":
				writeSnapshotSource(t, root, archive, append(bytes.Clone(old), ' '))
			case "wrong-filename", "other-locale-archive", "arbitrary-same-hash":
				removeArchive()
				path := "unrelated/" + sha + ".yaml"
				if name == "wrong-filename" {
					path = "data/glossary-history/zh-CN/" + strings.Repeat("0", 64) + ".yaml"
				} else if name == "other-locale-archive" {
					path = "data/glossary-history/fr-FR/" + sha + ".yaml"
				}
				writeSnapshotSource(t, root, path, old)
			case "archive-symlink":
				removeArchive()
				writeSnapshotSource(t, root, "unrelated.yaml", old)
				if err := os.Symlink(filepath.Join(root, "unrelated.yaml"), filepath.Join(root, archive)); err != nil {
					t.Fatal(err)
				}
			case "archive-parent-symlink":
				parent := filepath.Join(root, "data/glossary-history/zh-CN")
				if err := os.Rename(parent, parent+"-saved"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(parent+"-saved", parent); err != nil {
					t.Fatal(err)
				}
			case "archive-directory":
				removeArchive()
				if err := os.Mkdir(filepath.Join(root, archive), 0700); err != nil {
					t.Fatal(err)
				}
			case "wrong-frozen-sha":
				lineage.Dependencies[1].SHA256 = strings.Repeat("0", 64)
			case "wrong-archive-ref":
				lineage.Dependencies[1].Path = "data/glossary-history/zh-CN/" + strings.Repeat("0", 64) + ".yaml"
				writeSnapshotSource(t, root, lineage.Dependencies[1].Path, old)
			case "wrong-locale-ref":
				lineage.Dependencies[1].Path = glossaryReviewGlossaryPath("fr-FR")
				writeSnapshotSource(t, root, lineage.Dependencies[1].Path, old)
			case "other-dependency-drift":
				writeSnapshotSource(t, root, bridge.Path, append(bytes.Clone(receipt), ' '))
				writeSnapshotSource(t, root, "unrelated-same-hash.json", receipt)
			}
			before := mustJSON(lineage)
			err := lineage.check(root, "zh-CN")
			wantPass := name == "same-current" || name == "legal-successor"
			if (err == nil) != wantPass {
				t.Fatalf("pass=%v, error=%v", wantPass, err)
			}
			if !bytes.Equal(before, mustJSON(lineage)) {
				t.Fatal("historical lineage bytes changed")
			}
			if wantPass {
				resolved, b, err := readHistoricalBridgeReference(root, "zh-CN", lineage.Dependencies[1])
				if err != nil || resolved.Path != archive || resolved.SHA256 != sha || !bytes.Equal(b, old) {
					t.Fatal("historical proof did not bind exact immutable archive", resolved, err)
				}
				got, err := os.ReadFile(filepath.Join(root, bridge.Path))
				if err != nil || !bytes.Equal(got, receipt) {
					t.Fatal("historical receipt bytes changed", err)
				}
			}
		})
	}
}

func TestHistoricalBridgeGlossaryCurrentAuthorization(t *testing.T) {
	root, c, inputs := finalizedCompositionFixture(t)
	old, err := readCompatibilityFile(root, glossaryReviewGlossaryPath("zh-CN"))
	if err != nil {
		t.Fatal(err)
	}
	lineage := bridgeResponseLineage{Dependencies: []GlossaryArchiveReference{{Path: glossaryReviewGlossaryPath("zh-CN"), SHA256: sum(old)}}}
	checkpointGlossary(t, root, "\n# legitimate later current glossary\n", "bridge-current-review")
	if err := lineage.check(root, "zh-CN"); err != nil {
		t.Fatal(err)
	}
	current, _ := readCompatibilityFile(root, glossaryReviewGlossaryPath("zh-CN"))
	p, _, err := ComposeFinalizedTranslationPackage(root, c, "zh-CN", inputs)
	if err != nil {
		t.Fatal("compatible successor composition", err)
	}
	for _, a := range p.Assessments {
		if !a.Ready() || a.Glossary.Path != glossaryReviewGlossaryPath("zh-CN") || a.Glossary.SHA256 != sum(current) || a.Glossary.SHA256 == sum(old) || !strings.Contains(a.UnifiedReview.Path, "bridge-current-review") {
			t.Fatal("current authorization used historical glossary or Review", a.Glossary, a.UnifiedReview)
		}
	}
	writeSnapshotSource(t, root, glossaryReviewGlossaryPath("zh-CN"), append(current, []byte("# unreviewed change\n")...))
	if err := lineage.check(root, "zh-CN"); err != nil {
		t.Fatal("historical decoder inherited current Review gate", err)
	}
	if _, _, err := ComposeFinalizedTranslationPackage(root, c, "zh-CN", inputs); err == nil {
		t.Fatal("current authorization fell back to historical glossary/Review")
	}
}
