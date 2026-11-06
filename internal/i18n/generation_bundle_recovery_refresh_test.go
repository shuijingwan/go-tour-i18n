package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerationBundleRecoverAuthorityRefreshKeepsTrustedOutputs(t *testing.T) {
	root, catalog, batchID, oldBundle, manifest := generationRecoveryFixture(t, 6)
	source := t.TempDir()
	trusted := map[string][]byte{}
	for _, out := range manifest.ExpectedOutputs[:2] {
		name := filepath.Base(out.BundlePath)
		trusted[name] = generationRecoveryInput(t, root, batchID, out)
		if err := os.WriteFile(filepath.Join(source, name), trusted[name], 0600); err != nil {
			t.Fatal(err)
		}
	}
	// An authority-only refresh changes the aggregate transport identity;
	// all six saved per-Unit inputs and the batch authorization stay intact.
	authority := filepath.Join(root, "docs", "TRANSLATION_TASK_SPEC.md")
	before, err := os.ReadFile(authority)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(authority, append(before, []byte("\nFixture authority clarification.\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	staleTarget := filepath.Join(t.TempDir(), "stale")
	if _, err := RecoverGenerationOutputs(root, catalog, oldBundle, []string{source}, staleTarget); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("old aggregate bundle remained current: %v", err)
	}
	if _, err := os.Lstat(staleTarget); !os.IsNotExist(err) {
		t.Fatal("stale recovery wrote staging", err)
	}
	current, next, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: batchID})
	if err != nil || bytes.Equal(current, oldBundle) || next.InputIdentitySHA256 == manifest.InputIdentitySHA256 || next.BatchID != batchID {
		t.Fatal("same-batch canonical refresh failed", err)
	}
	target := filepath.Join(t.TempDir(), "recovered")
	result, err := RecoverGenerationOutputs(root, catalog, current, []string{source}, target)
	if err != nil || len(result.Recovered) != 2 || len(result.Missing) != 4 {
		t.Fatal("per-Unit reuse failed after aggregate identity drift", result, err)
	}
	for name, want := range trusted {
		got, err := os.ReadFile(filepath.Join(target, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatal("trusted output bytes changed", name, err)
		}
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	for _, name := range []string{"raw-responses", "candidates", "validation", "result.json", "generation-imports"} {
		if _, err := os.Lstat(filepath.Join(batchDir, name)); !os.IsNotExist(err) {
			t.Fatal("recovery mutated formal state", name, err)
		}
	}
}
