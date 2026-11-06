package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func runCanonicalReattestFixture(t *testing.T, root string, c *i18n.Catalog, canonical bool) i18n.CurrentTranslationAuthorization {
	t.Helper()
	args := []string{"--locale", "zh-CN", "--snapshot-id", "diagnostic-finalized", "--apply"}
	if canonical {
		args = append(args, "--canonical")
	}
	b, err := authorizationDiagnosisOutput(t, func() error { return translationReattestationCommand(root, c, args) })
	if err != nil {
		t.Fatal(err)
	}
	var a i18n.CurrentTranslationAuthorization
	if err := json.Unmarshal(b, &a); err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(b, &wire); err != nil {
		t.Fatal(err)
	}
	if _, exists := wire["canonical"]; exists {
		t.Fatal("unexpected canonical JSON field; use current_canonical_targets")
	}
	if canonical && len(wire["current_canonical_targets"]) == 0 {
		t.Fatal("canonical command did not emit the formal wire field")
	}
	return a
}

func TestReattestCanonicalCLIUpgrade(t *testing.T) {
	root, c, _ := authorizationDiagnosisFixture(t, 2)
	if err := c.UseTranslationPackage(c.TranslationPackages[0].ID); err != nil {
		t.Fatal(err)
	}
	language := runCanonicalReattestFixture(t, root, c, false)
	oldPath := i18n.CurrentTranslationAuthorizationPath(&language)
	oldBytes := courseContextRead(t, root, oldPath)
	before := authorizationDiagnosisInventory(t, root)
	again := runCanonicalReattestFixture(t, root, c, false)
	if len(language.Canonical) != 0 || !reflect.DeepEqual(language, again) || !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, root)) {
		t.Fatal("language idempotence")
	}
	for _, x := range language.Units {
		one := *c
		one.TranslationScope = []string{x.Snapshot.UnitID}
		paths, err := i18n.CanonicalTranslationUnitTargetPaths(&one, "zh-CN")
		if err != nil || len(paths) != 1 {
			t.Fatal(paths, err)
		}
		for path := range paths {
			courseContextWrite(t, root, path, courseContextRead(t, root, x.Snapshot.CandidatePath))
		}
	}
	canonical := runCanonicalReattestFixture(t, root, c, true)
	if canonical.Identity == language.Identity || len(canonical.Canonical) != 2 || !bytes.Equal(oldBytes, courseContextRead(t, root, oldPath)) {
		t.Fatal("canonical upgrade or immutable old epoch")
	}
	before = authorizationDiagnosisInventory(t, root)
	again = runCanonicalReattestFixture(t, root, c, true)
	if !reflect.DeepEqual(canonical, again) || !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, root)) {
		t.Fatal("canonical command is not idempotent")
	}
}

// Explicit opt-in. Replays real qc-010 only in a repository copy. The supplied
// 30e846 artifact already has 122 current_canonical_targets, so verify that real
// state first; derive a separate language-only fixture without changing bytes.
func TestCanonicalCheckpointRealQC010Copy(t *testing.T) {
	out := os.Getenv("GO_TOUR_CANONICAL_CHECKPOINT_COPY_PROOF")
	if out == "" {
		t.Skip("explicit isolated real qc-010 proof")
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if root == repo {
		t.Fatal("proof must be isolated")
	}
	for _, dir := range []string{"data", "locales", "internal", "docs", "_content"} {
		if b, err := exec.Command("cp", "-a", "--reflink=auto", filepath.Join(repo, dir), root).CombinedOutput(); err != nil {
			t.Fatal("copy fixture", dir, err, string(b))
		}
	}
	for _, path := range []string{"go.mod", "go.sum", "AGENTS.md", "README.md", "README.en.md"} {
		courseContextWrite(t, root, path, courseContextRead(t, repo, path))
	}
	const relative = "data/translation-authorizations/zh-CN/88337426ad44d3e7193d668a062bed7022c230691d97412be70a49703652b1a6/qc-010/30e846f84bf779df3646393116907b8f1372fb16165d897fc92be54f39d97f13.json"
	original := courseContextRead(t, root, relative)
	if courseContextSHA(original) != "e3b6d3075a2420b373230bccf86e697df24845e9265f160f15385f1c89b94884" {
		t.Fatal("real input SHA changed")
	}
	var preexisting i18n.CurrentTranslationAuthorization
	if err := json.Unmarshal(original, &preexisting); err != nil {
		t.Fatal(err)
	}
	if len(preexisting.Canonical) != 122 {
		t.Fatal("actual input must be reported honestly", len(preexisting.Canonical))
	}
	done := contentidentity.BeginMemoization()
	defer done()
	c, err := i18n.BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	apply := func(canonical bool, name string) i18n.CurrentTranslationAuthorization {
		t.Helper()
		args := []string{"--locale", "zh-CN", "--snapshot-id", "qc-010", "--apply"}
		if canonical {
			args = append(args, "--canonical")
		}
		b, err := authorizationDiagnosisOutput(t, func() error { return translationReattestationCommand(root, c, args) })
		if err != nil {
			t.Fatal(name, err)
		}
		var a i18n.CurrentTranslationAuthorization
		if err := json.Unmarshal(b, &a); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(out, name+".json"), b, 0600); err != nil {
			t.Fatal(err)
		}
		return a
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}
	before := authorizationDiagnosisInventory(t, root)
	replay := apply(true, "real-state-canonical-replay")
	if replay.Identity != preexisting.Identity || len(replay.Canonical) != 122 || !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, root)) {
		t.Fatal("real existing canonical epoch is not idempotent")
	}
	b, err := authorizationDiagnosisOutput(t, func() error {
		return qualityCheckAuthorizationCommand(root, c, []string{"--locale", "zh-CN", "--snapshot-id", "qc-010"})
	})
	if err != nil {
		t.Fatal("native diagnostic ignored valid shared checkpoint", err)
	}
	var diagnostic i18n.QualityCheckAuthorizationDiagnostic
	if json.Unmarshal(b, &diagnostic) != nil || !diagnostic.CanonicalAuthorized || diagnostic.UnitCount != 122 {
		t.Fatal("native diagnosis", diagnostic)
	}
	if err := os.WriteFile(filepath.Join(out, "native-authorization-diagnosis.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	language := apply(false, "fixture-language-only")
	if len(language.Canonical) != 0 || language.Identity == preexisting.Identity {
		t.Fatal("language-only fixture identity")
	}
	oldPath := i18n.CurrentTranslationAuthorizationPath(&language)
	oldBytes := courseContextRead(t, root, oldPath)
	// Fixture setup only: retain original canonical epoch bytes outside the
	// fixture's reader directory, leaving the genuine language-only sibling.
	dir := filepath.Dir(filepath.Join(root, relative))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(root, "fixture-canonical-epochs")
	if err := os.Mkdir(archive, 0700); err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var a i18n.CurrentTranslationAuthorization
		if err := json.Unmarshal(data, &a); err != nil {
			t.Fatal(err)
		}
		if len(a.Canonical) > 0 {
			if err := os.Rename(path, filepath.Join(archive, e.Name())); err != nil {
				t.Fatal(err)
			}
		}
	}
	before = authorizationDiagnosisInventory(t, root)
	upgraded := apply(true, "fixture-canonical-upgrade")
	after := authorizationDiagnosisInventory(t, root)
	added := 0
	for path := range after {
		if _, ok := before[path]; !ok {
			added++
		}
	}
	if added != 1 || upgraded.Identity == language.Identity || len(upgraded.Canonical) != 122 || !upgraded.Ready() || !bytes.Equal(oldBytes, courseContextRead(t, root, oldPath)) {
		t.Fatal("fixture language-to-canonical upgrade", added)
	}
	second := apply(true, "fixture-canonical-second-apply")
	if second.Identity != upgraded.Identity || !reflect.DeepEqual(after, authorizationDiagnosisInventory(t, root)) {
		t.Fatal("second apply changed artifacts")
	}
	if _, refs, err := i18n.RequireCurrentCanonicalQualityCheck(root, "zh-CN", c); err != nil || len(refs) != 123 {
		t.Fatal("canonical consumer", err)
	}
	if !bytes.Equal(original, courseContextRead(t, repo, relative)) {
		t.Fatal("real artifact changed")
	}
	report := map[string]any{"real_preexisting_identity": preexisting.Identity, "real_preexisting_canonical_refs": len(preexisting.Canonical), "real_state_replay_idempotent": true, "language_fixture_identity": language.Identity, "language_fixture_canonical_refs": 0, "new_canonical_epoch": true, "new_canonical_identity": upgraded.Identity, "unit_count": len(upgraded.Units), "compatible": upgraded.Compatible, "affected": upgraded.Affected, "stale": upgraded.Stale, "unknown": upgraded.Unknown, "canonical_refs": len(upgraded.Canonical), "old_language_only_preserved": true, "second_apply_already_current": true, "second_apply_new_artifacts": 0, "native_diagnosis_pass": true, "real_mutation_executed": false}
	if err := os.WriteFile(filepath.Join(out, "real-copy-proof-summary.json"), append(courseContextJSON(t, report), '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	t.Log("real canonical epoch replay PASS; isolated language-only upgrade PASS; native diagnosis PASS; real mutation=no")
}
