package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func TestSharedReattestationCommandContract(t *testing.T) {
	doc, err := os.ReadFile("../../docs/SHARED_STRUCTURED_REPRESENTATION_REATTESTATION.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"COMMAND", "OPERATION", "INPUT_KIND", "INPUT_PATH_CONTRACT", "REQUIRED_FLAGS", "FORBIDDEN_FLAGS", "STDOUT_KIND", "MUTATION_EXPECTATION", "IDEMPOTENCE", "CURRENTNESS_CHECK", "PRIOR_OR_CAS", "SUCCESS_STOP", "FAILURE_STOP", "RECOVERY_REPLAY", "current/<base-identity>.json", "all-locales", "added_dependencies=[]", "removed_dependencies=[]", "first-real-use", "canonical profile fingerprints", "old_currentness_inputs", "current_currentness_inputs", "validation_inputs", "extended-v1", "language-audit-v2"} {
		if !strings.Contains(string(doc), field) {
			t.Fatal("canonical contract missing", field)
		}
	}
	root := t.TempDir()
	base := []byte(`{"schema":"` + sitecontent.SharedRepresentationSchema + `"}`)
	plan := []byte(`{"schema":"` + sitecontent.SharedReattestationSchema + `/plan"}`)
	for p, b := range map[string][]byte{"base.json": base, "plan.json": plan} {
		if err := os.WriteFile(filepath.Join(root, p), b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	bsha, psha := contentidentity.SHA256Hex(base), contentidentity.SHA256Hex(plan)
	var b sitecontent.SharedRepresentationSuccessor
	if err := readSharedReattestationCLIInput(root, sitecontent.Reference{Path: "base.json", SHA256: bsha}, &b, sitecontent.SharedRepresentationSchema); err != nil {
		t.Fatal(err)
	}
	var p sitecontent.SharedRepresentationReattestationPlan
	if err := readSharedReattestationCLIInput(root, sitecontent.Reference{Path: "plan.json", SHA256: psha}, &p, sitecontent.SharedReattestationSchema+"/plan"); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{}, {"--base", "base.json"}, {"--base-sha256", bsha},
		{"--base", "base.json", "--base-sha256", bsha, "--apply=false"},
		{"--base", "base.json", "--base-sha256", bsha, "--input", ""},
		{"--base", "base.json", "--base-sha256", bsha, "--prior-pointer-sha256", ""},
		{"--base", "base.json", "--base-sha256", bsha, "--input", "plan.json", "--input-sha256", psha},
		{"--base", filepath.Join(root, "base.json"), "--base-sha256", bsha},
		{"--base", "../base.json", "--base-sha256", bsha},
		{"--base", "base.json", "--base-sha256", strings.Repeat("0", 64)},
		{"--base", "plan.json", "--base-sha256", psha},
		{"--input", "base.json", "--input-sha256", bsha},
		{"--input", "plan.json"}, {"--input-sha256", psha},
		{"--input", "plan.json", "--input-sha256", psha, "--apply=false"},
		{"--input", "plan.json", "--input-sha256", psha, "--prior-pointer-sha256", ""},
		{"--input", "plan.json", "--input-sha256", psha, "--base", ""},
		{"--input", "plan.json", "--input-sha256", psha, "--apply"},
		{"--locale", "zh-CN"}, {"--package", "site-v2-shell"}, {"--parser", "new"}, {"--source", "_content/site.tmpl"}, {"--output", "out.json"}, {"--request", "request.json"}, {"--prior-identity", "x"}, {"extra"},
	} {
		if err := siteContentCommand(root, append([]string{"shared-structured-representation-reattest"}, args...)); err == nil {
			t.Fatal("invalid flags/input accepted", args)
		}
	}
	if err := os.Symlink("base.json", filepath.Join(root, "symlink.json")); err != nil {
		t.Fatal(err)
	}
	if err := siteContentCommand(root, []string{"shared-structured-representation-reattest", "--base", "symlink.json", "--base-sha256", bsha}); err == nil {
		t.Fatal("symlink accepted")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 3 {
		t.Fatal("invalid CLI mutated lifecycle", entries, err)
	}
}
