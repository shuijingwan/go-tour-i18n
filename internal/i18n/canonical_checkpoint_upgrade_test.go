package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func installCheckpointTargets(t *testing.T, root string, c *Catalog, a *CurrentTranslationAuthorization) {
	t.Helper()
	for _, x := range a.Units {
		u, err := c.Unit(x.Snapshot.UnitID)
		if err != nil {
			t.Fatal(err)
		}
		path, err := canonicalTranslationUnitCandidatePath(a.Locale, u)
		if err != nil {
			t.Fatal(err)
		}
		b, err := readCompatibilityFile(root, x.Snapshot.CandidatePath)
		if err != nil {
			t.Fatal(err)
		}
		if u.Kind == UnitKindPage {
			b = canonicalizeCandidateEOF(b)
		}
		writeSnapshotSource(t, root, path, b)
	}
}

func checkpointEpochNames(t *testing.T, root string, a *CurrentTranslationAuthorization) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(filepath.Join(root, CurrentTranslationAuthorizationPath(a))))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestCanonicalCheckpointLanguageUpgrade(t *testing.T) {
	root, c, _, snapshot := checkpointFixture(t)
	language, oldPath, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot)
	if err != nil || len(language.Canonical) != 0 || len(checkpointEpochNames(t, root, language)) != 1 {
		t.Fatal(language, err)
	}
	before, err := readCompatibilityFile(root, oldPath)
	if err != nil {
		t.Fatal(err)
	}
	again, path, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot)
	if err != nil || path != oldPath || !reflect.DeepEqual(language, again) || len(checkpointEpochNames(t, root, language)) != 1 {
		t.Fatal("language idempotence", err)
	}
	if d, err := DiagnoseQualityCheckAuthorization(root, c, "zh-CN", snapshot); err != nil || d.CanonicalAuthorized {
		t.Fatal("language-only diagnosis authorized install", d, err)
	}
	installCheckpointTargets(t, root, c, language)
	canonical, canonicalPath, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot, true)
	if err != nil || canonical.Identity == language.Identity || canonicalPath == oldPath || len(canonical.Canonical) != len(canonical.Units) || len(checkpointEpochNames(t, root, canonical)) != 2 {
		t.Fatal("canonical successor epoch", canonical, err)
	}
	after, err := readCompatibilityFile(root, oldPath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("language-only epoch rewritten", err)
	}
	for i := 0; i < 2; i++ {
		got, path, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot, true)
		if err != nil || path != canonicalPath || !reflect.DeepEqual(got, canonical) || len(checkpointEpochNames(t, root, canonical)) != 2 {
			t.Fatal("canonical idempotence", err)
		}
		selected, err := FindCurrentTranslationAuthorization(root, c, "zh-CN", snapshot)
		if err != nil || !reflect.DeepEqual(selected, canonical) {
			t.Fatal("canonical sibling selection", err)
		}
		selected, err = findScopeCurrentAuthorization(root, c, "zh-CN")
		if err != nil || !reflect.DeepEqual(selected, canonical) {
			t.Fatal("scope sibling selection", err)
		}
		if d, err := DiagnoseQualityCheckAuthorization(root, c, "zh-CN", snapshot); err != nil || !d.CanonicalAuthorized || d.Reason != "verified-current-canonical-checkpoint" {
			t.Fatal("checkpoint ignored by diagnosis", d, err)
		}
		if _, refs, err := RequireCurrentCanonicalQualityCheck(root, "zh-CN", c); err != nil || len(refs) != len(canonical.Units)+1 {
			t.Fatal("canonical consumer ignored sibling", err)
		}
	}
	for _, path := range []string{oldPath, canonicalPath} {
		t.Run(filepath.Base(path), func(t *testing.T) {
			b, err := readCompatibilityFile(root, path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeSnapshotSource(t, root, path, b)
			writeSnapshotSource(t, root, path, []byte("{corrupt sibling\n"))
			if _, err := FindCurrentTranslationAuthorization(root, c, "zh-CN", snapshot); err == nil {
				t.Fatal("corrupt sibling concealed")
			}
			if _, _, err := RequireCurrentCanonicalQualityCheck(root, "zh-CN", c); err == nil {
				t.Fatal("canonical consumer concealed corrupt sibling")
			}
			if _, _, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot, true); err == nil {
				t.Fatal("apply concealed corrupt sibling")
			}
		})
	}
}

func TestCanonicalCheckpointUpgradeFreshValidation(t *testing.T) {
	for _, kind := range []string{"missing-target", "changed-target", "current-validator", "current-review"} {
		t.Run(kind, func(t *testing.T) {
			root, c, _, snapshot := checkpointFixture(t)
			a, oldPath, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot)
			if err != nil {
				t.Fatal(err)
			}
			before, err := readCompatibilityFile(root, oldPath)
			if err != nil {
				t.Fatal(err)
			}
			installCheckpointTargets(t, root, c, a)
			switch kind {
			case "missing-target":
				u, _ := c.Unit(a.Units[0].Snapshot.UnitID)
				path, _ := canonicalTranslationUnitCandidatePath("zh-CN", u)
				if err := os.Remove(filepath.Join(root, path)); err != nil {
					t.Fatal(err)
				}
			case "changed-target":
				u, _ := c.Unit(a.Units[0].Snapshot.UnitID)
				path, _ := canonicalTranslationUnitCandidatePath("zh-CN", u)
				writeSnapshotSource(t, root, path, []byte("unaccepted installed bytes\n"))
			case "current-validator":
				c.Adapted[0].Adapter = checkpointRejectingAdapter{c.Adapted[0].Adapter.(futureCardAdapter)}
				c.TranslationPackages[0].Units[0] = c.Adapted[0]
			case "current-review":
				b, err := readCompatibilityFile(root, a.Glossary.Path)
				if err != nil {
					t.Fatal(err)
				}
				writeSnapshotSource(t, root, a.Glossary.Path, append(b, []byte("# unreviewed\n")...))
			}
			if _, _, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot, true); err == nil {
				t.Fatal("upgrade skipped fresh checks", kind)
			}
			after, err := readCompatibilityFile(root, oldPath)
			if err != nil || !bytes.Equal(before, after) || len(checkpointEpochNames(t, root, a)) != 1 {
				t.Fatal("failed upgrade published/rewrote an epoch", err)
			}
		})
	}
}
