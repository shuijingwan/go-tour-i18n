package main

import (
	"os"
	"strings"
	"testing"
)

func TestCurrentValidationReopenCLI(t *testing.T) {
	doc, err := os.ReadFile("../../docs/CURRENT_TRANSLATION_AUTHORIZATION.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range []string{"只提供一次", "canonical Tour-native `tour-v1`", "default complete Catalog", "UseTranslationPackage()", "不伪造 registration", "partial", "完整 ordered exact set"} {
		if !strings.Contains(string(doc), rule) {
			t.Fatal("canonical reopen selector contract missing", rule)
		}
	}
	sha := strings.Repeat("a", 64)
	args := []string{"--content-package", "learn-docs-v1", "--locale", "zh-CN", "--reopen-id", "machine-comments", "--previous-snapshot-id", "qc-before", "--previous-snapshot-sha256", sha, "--current-reattestation", "/tmp/assessment.json", "--current-reattestation-sha256", sha, "--current-reattestation-identity", sha}
	pkg, remaining, err := parseTranslationContentPackageArgs(append([]string{"quality-check", "current-validation-reopen"}, args...))
	if err != nil || pkg != "" || len(remaining) != len(args)+2 {
		t.Fatal("required package consumed by global parser", pkg, remaining, err)
	}
	o, record, err := parseCurrentValidationReopenOptions(remaining[2:])
	if err != nil || record || o.ContentPackage != "learn-docs-v1" || o.Evidence.Path != "/tmp/assessment.json" || o.AssessmentIdentity != sha {
		t.Fatal(o, record, err)
	}
	_, record, err = parseCurrentValidationReopenOptions(append(append([]string{}, args...), "--record"))
	if err != nil || !record {
		t.Fatal("explicit record flag", err)
	}
	for _, extra := range [][]string{{"--apply"}, {"--finding", "invented"}, {"--surface-review-id", "fake"}, {"--previous-rating", "B"}, {"--preview-failure", "fake"}} {
		if _, _, err := parseCurrentValidationReopenOptions(append(append([]string{}, args...), extra...)); err == nil {
			t.Fatal("mixed authority accepted", extra)
		}
	}
	if _, _, err := parseCurrentValidationReopenOptions(args[:len(args)-2]); err == nil {
		t.Fatal("missing identity accepted")
	}
	if _, _, err := parseCurrentValidationReopenOptions(args[2:]); err == nil {
		t.Fatal("implicit package accepted")
	}
	if _, _, err := parseCurrentValidationReopenOptions(append(append([]string{}, args...), "--content-package", "other")); err == nil {
		t.Fatal("duplicate package accepted")
	}
	tourArgs := append([]string{}, args...)
	tourArgs[1] = "tour-v1"
	pkg, remaining, err = parseTranslationContentPackageArgs(append([]string{"quality-check", "current-validation-reopen"}, tourArgs...))
	if err != nil || pkg != "" || len(remaining) != len(tourArgs)+2 {
		t.Fatal("explicit native selector consumed by global parser", pkg, err)
	}
	o, record, err = parseCurrentValidationReopenOptions(remaining[2:])
	if err != nil || record || o.ContentPackage != "tour-v1" {
		t.Fatal("explicit Tour readonly selector rejected", o, record, err)
	}
	for _, extra := range [][]string{{"--content-package", "tour-v1"}, {"--content-package", "learn-docs-v1"}, {"extra"}} {
		if _, _, err := parseCurrentValidationReopenOptions(append(append([]string{}, tourArgs...), extra...)); err == nil {
			t.Fatal("duplicate native selector or positional argument accepted", extra)
		}
	}
	if _, _, err := parseCurrentValidationReopenOptions(append([]string{"--content-package", ""}, tourArgs[2:]...)); err == nil {
		t.Fatal("empty selector implicitly selected Tour")
	}
}
