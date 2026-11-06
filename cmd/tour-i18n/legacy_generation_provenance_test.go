package main

import (
	"strings"
	"testing"
)

func TestLegacyGenerationProvenanceCLIContract(t *testing.T) {
	args := []string{"--locale", "zh-CN", "--content-package", "registered-pages", "--batch-id", "chatgpt-zh-CN-031", "--unit-id", strings.Repeat("a", 64), "--current-unit-id", strings.Repeat("b", 64), "--attempt", "1", "--task-kind", "translation-unit-revision", "--import-receipt", "data/retranslation-runs/zh-CN/chatgpt-zh-CN-031/generation-imports/initial-manifest.json", "--import-receipt-sha256", strings.Repeat("c", 64), "--original-bundle-sha256", strings.Repeat("d", 64), "--previous-snapshot-id", "reviewed", "--previous-snapshot-sha256", strings.Repeat("e", 64)}
	o, record, id, err := parseLegacyGenerationProvenanceOptions(args)
	if err != nil || record || id != "" || o.ContentPackage != "registered-pages" {
		t.Fatal(o, record, id, err)
	}
	_, record, id, err = parseLegacyGenerationProvenanceOptions(append(append([]string{}, args...), "--record", "--expected-identity", strings.Repeat("f", 64)))
	if err != nil || !record || id != strings.Repeat("f", 64) {
		t.Fatal(err)
	}
	for _, suffix := range [][]string{{"--record"}, {"--expected-identity", "x"}, {"--unit-id", "other"}, {"--content-package=other"}, {"--apply"}, {"--all"}, {"--bundle", "fake.zip"}, {"--provider", "chatgpt"}, {"--model", "invented"}, {"--target-occurrence", "1"}, {"positional"}} {
		if _, _, _, err = parseLegacyGenerationProvenanceOptions(append(append([]string{}, args...), suffix...)); err == nil {
			t.Fatal("invalid flags accepted", suffix)
		}
	}
	for i := 0; i < len(args); i += 2 {
		missing := append(append([]string{}, args[:i]...), args[i+2:]...)
		if _, _, _, err = parseLegacyGenerationProvenanceOptions(missing); err == nil {
			t.Fatal("missing selector accepted", args[i])
		}
	}
	full := append([]string{"generation-bundle", "legacy-provenance"}, args...)
	pkg, rest, err := parseTranslationContentPackageArgs(full)
	if err != nil || pkg != "" || len(rest) != len(full) {
		t.Fatal("generic dispatcher consumed explicit authority selector", err)
	}
	if !readonlyGenerationCatalog(full) {
		t.Fatal("legacy entry did not select readonly registration")
	}
	// The command entry rejects invalid flags before catalog/filesystem access.
	// Never invokes a real record.
	if err = legacyGenerationProvenanceCommand("/unavailable", nil, []string{"--apply"}); err == nil {
		t.Fatal("missing CLI dispatch validation")
	}
}
