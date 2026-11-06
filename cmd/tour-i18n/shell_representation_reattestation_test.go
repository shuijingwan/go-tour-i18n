package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func TestShellRepresentationReattestationCommandContract(t *testing.T) {
	root := t.TempDir()
	plan := []byte(`{"schema":"` + sitecontent.ShellRepresentationReattestationSchema + `/plan"}`)
	if err := os.WriteFile(filepath.Join(root, "plan.json"), plan, 0600); err != nil {
		t.Fatal(err)
	}
	sha := fmt.Sprintf("%x", sha256.Sum256(plan))
	ref := sitecontent.Reference{Path: "plan.json", SHA256: sha}
	if err := validateShellReattestationInput(root, ref); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"--apply"}, "readonly prospective preflight"},
		{[]string{"--input", ""}, "readonly prospective preflight"},
		{[]string{"--record"}, "flag provided but not defined"},
		{[]string{"--locale", "zh-CN"}, "flag provided but not defined"},
		{[]string{"--input", "plan.json"}, "input-sha256"},
		{[]string{"--input", "plan.json", "--input-sha256", sha, "--apply"}, "prior-pointer-sha256"},
		{[]string{"--input", "plan.json", "--input-sha256", sha, "--prior-pointer-sha256", sha}, "only --apply"},
		{[]string{"--input", filepath.Join(root, "plan.json"), "--input-sha256", sha}, "repository-relative"},
		{[]string{"--input", "plan.json", "--input-sha256", strings.Repeat("0", 64)}, "mismatch"},
		{[]string{"unexpected"}, "positional"},
	} {
		err := siteContentCommand(root, append([]string{"shell-representation-reattest"}, tc.args...))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%v: got %v; want %q", tc.args, err, tc.want)
		}
	}
	for _, raw := range [][]byte{
		[]byte(`{"schema":"` + sitecontent.ShellRepresentationSchema + `"}`),
		[]byte(`{"schema":"` + sitecontent.ShellRepresentationPlanSchema + `"}`),
		[]byte(`{"schema":"` + sitecontent.ShellRepresentationReattestationSchema + `/plan","unexpected":true}`),
	} {
		if err := os.WriteFile(filepath.Join(root, "wrong-kind.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
		if err := validateShellReattestationInput(root, sitecontent.Reference{Path: "wrong-kind.json", SHA256: fmt.Sprintf("%x", sha256.Sum256(raw))}); err == nil {
			t.Fatal("wrong authority kind accepted as re-attestation plan")
		}
	}
	doc, err := os.ReadFile("../../docs/SHELL_REPRESENTATION_REATTESTATION.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"OPERATION", "INPUT_KIND", "INPUT_PATH_CONTRACT", "MUTATION_EXPECTATION", "prospective-preflight", "plan-inspection", "--prior-pointer-sha256", "already_current=true, applied=false, mutation=none", "mutation=unknown", "Formal Command Contract v3"} {
		if !strings.Contains(string(doc), field) {
			t.Error("canonical command contract missing", field)
		}
	}
}
