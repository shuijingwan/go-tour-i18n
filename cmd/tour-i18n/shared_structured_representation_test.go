package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func TestSharedStructuredRepresentationCommandContract(t *testing.T) {
	doc, err := os.ReadFile("../../docs/SHARED_STRUCTURED_REPRESENTATION_SUCCESSOR.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"COMMAND", "OPERATION", "INPUT_KIND", "INPUT_PATH_CONTRACT", "REQUIRED_FLAGS", "FORBIDDEN_FLAGS", "STDOUT_KIND", "MUTATION_EXPECTATION", "IDEMPOTENCE", "CURRENTNESS_CHECK", "PRIOR_OR_CAS", "SUCCESS_STOP", "FAILURE_STOP", "RECOVERY_REPLAY", "current/<key>.json", "all-locales", "slot_ids", "first-real-use"} {
		if !strings.Contains(string(doc), field) {
			t.Fatal("canonical command contract missing", field)
		}
	}
	for _, rule := range []string{sitecontent.SharedRepresentationRule, sitecontent.SharedSentencePunctuationRule} {
		if !strings.Contains(string(doc), "`"+rule+"`") {
			t.Fatal("canonical request rule missing", rule)
		}
	}
	root := t.TempDir()
	raw := []byte(`{"schema":"` + sitecontent.SharedRepresentationSchema + `/request"}`)
	plan := []byte(`{"schema":"` + sitecontent.SharedRepresentationSchema + `/plan"}`)
	if err := os.WriteFile(filepath.Join(root, "request.json"), raw, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "plan.json"), plan, 0644); err != nil {
		t.Fatal(err)
	}
	sha := contentidentity.SHA256Hex(raw)
	psha := contentidentity.SHA256Hex(plan)
	var q sitecontent.SharedRepresentationRequest
	if err := readSharedRepresentationCLIInput(root, sitecontent.Reference{Path: "request.json", SHA256: sha}, &q, sitecontent.SharedRepresentationSchema+"/request"); err != nil {
		t.Fatal("request path/hash/kind", err)
	}
	var p sitecontent.SharedRepresentationPlan
	if err := readSharedRepresentationCLIInput(root, sitecontent.Reference{Path: "plan.json", SHA256: psha}, &p, sitecontent.SharedRepresentationSchema+"/plan"); err != nil {
		t.Fatal("plan path/hash/kind", err)
	}
	for _, args := range [][]string{
		{}, {"--request", "request.json"}, {"--input", "plan.json"},
		{"--request", "request.json", "--request-sha256", sha, "--apply"},
		{"--request", "request.json", "--request-sha256", sha, "--apply=false"},
		{"--request", "request.json", "--request-sha256", sha, "--input", ""},
		{"--request", "request.json", "--request-sha256", sha, "--prior-identity", "x"},
		{"--request", "request.json", "--request-sha256", sha, "--input", "plan.json"},
		{"--request", filepath.Join(root, "request.json"), "--request-sha256", sha},
		{"--request", "../request.json", "--request-sha256", sha},
		{"--request", "request.json", "--request-sha256", strings.Repeat("0", 64)},
		{"--request", "plan.json", "--request-sha256", psha},
		{"--input", "request.json", "--input-sha256", sha},
		{"--input", "plan.json", "--input-sha256", psha, "--prior-identity", "x"},
		{"--input", "plan.json", "--input-sha256", psha, "--prior-identity", ""},
		{"--input", "plan.json", "--input-sha256", psha, "--apply=false"},
		{"--input", "plan.json", "--input-sha256", psha, "--apply"},
		{"--input", "plan.json", "--input-sha256", psha, "--apply", "--prior-identity", "x"},
		{"--locale", "zh-CN"}, {"--source", "_content/site.tmpl"}, {"--package", "site-v2-shell"}, {"--parser", "unknown"}, {"--output", "out.json"}, {"extra"},
	} {
		if err := siteContentCommand(root, append([]string{"shared-structured-representation-successor"}, args...)); err == nil {
			t.Fatal("invalid command accepted", args)
		}
	}
	if err := os.Symlink("request.json", filepath.Join(root, "symlink.json")); err != nil {
		t.Fatal(err)
	}
	if err := siteContentCommand(root, []string{"shared-structured-representation-successor", "--request", "symlink.json", "--request-sha256", sha}); err == nil {
		t.Fatal("symlink accepted")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 3 {
		t.Fatal("invalid CLI mutated lifecycle state", entries, err)
	}
}
