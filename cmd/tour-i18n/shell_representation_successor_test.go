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

func TestShellRepresentationCommandBoundaries(t *testing.T) {
	root := t.TempDir()
	raw := []byte(`{"decision":"APPROVE_RECONCILIATION"}`)
	if err := os.WriteFile(filepath.Join(root, "semantic.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	sha := fmt.Sprintf("%x", sha256.Sum256(raw))
	for _, args := range [][]string{
		{"shell-representation-successor", "--apply"},
		{"shell-representation-successor", "--parser", "unknown", "--source", "_content/other.md"},
		{"shell-representation-successor", "--parser", "unknown", "--package", "learn-docs-v1"},
		{"shell-representation-successor", "--input", "semantic.json", "--apply"},
		{"shell-representation-successor", "--input", "semantic.json", "--input-sha256", sha, "--apply"},
		{"shell-representation-check", "--input", "semantic.json", "--input-sha256", sha, "--apply"},
		{"shell-representation-successor", "--input", "semantic.json", "--input-sha256", sha, "--prior-identity", "x"},
	} {
		if err := siteContentCommand(root, args); err == nil {
			t.Fatal("invalid command accepted", args)
		}
	}
}

func TestShellRepresentationCommandInputContract(t *testing.T) {
	root := t.TempDir()
	// Only schemas are needed at this parsing boundary. Authority eligibility
	// remains the responsibility of the full reader after operation selection.
	plan := []byte(`{"schema":"` + sitecontent.ShellRepresentationPlanSchema + `"}`)
	receipt := []byte(`{"schema":"` + sitecontent.ShellRepresentationSchema + `"}`)
	for _, tc := range []struct {
		command, name, operation string
		raw                      []byte
		apply                    bool
	}{
		{"shell-representation-successor", "plan.json", "plan-inspection", plan, false},
		{"shell-representation-successor", "plan.json", "apply", plan, true},
		{"shell-representation-check", "receipt.json", "receipt-verification", receipt, false},
	} {
		if err := os.WriteFile(filepath.Join(root, tc.name), tc.raw, 0600); err != nil {
			t.Fatal(err)
		}
		ref := sitecontent.Reference{Path: tc.name, SHA256: fmt.Sprintf("%x", sha256.Sum256(tc.raw))}
		got, err := shellRepresentationInputOperation(root, tc.command, ref, tc.apply)
		if err != nil || got != tc.operation {
			t.Fatalf("%s input %s: got %q, %v; want %q", tc.command, tc.name, got, err, tc.operation)
		}
	}
	for _, tc := range []struct{ command, name, want string }{
		{"shell-representation-check", "plan.json", "requires receipt input"},
		{"shell-representation-successor", "receipt.json", "requires plan input"},
		{"shell-representation-successor", filepath.Join(root, "plan.json"), "repository-relative"},
		{"shell-representation-check", filepath.Join(root, "receipt.json"), "repository-relative"},
	} {
		raw := plan
		if filepath.Base(tc.name) == "receipt.json" {
			raw = receipt
		}
		err := siteContentCommand(root, []string{tc.command, "--input", tc.name, "--input-sha256", fmt.Sprintf("%x", sha256.Sum256(raw))})
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s input %s: wanted %q, got %v", tc.command, tc.name, tc.want, err)
		}
	}
	for _, raw := range [][]byte{
		[]byte(`{"schema":"` + sitecontent.ShellRepresentationPlanSchema + `","unexpected":true}`),
		[]byte(`{"schema":"` + sitecontent.ShellRepresentationPlanSchema + `","schema":"` + sitecontent.ShellRepresentationPlanSchema + `"}`),
	} {
		if err := os.WriteFile(filepath.Join(root, "bad.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
		ref := sitecontent.Reference{Path: "bad.json", SHA256: fmt.Sprintf("%x", sha256.Sum256(raw))}
		if _, err := shellRepresentationInputOperation(root, "shell-representation-successor", ref, false); err == nil {
			t.Fatal("non-strict plan accepted")
		}
	}
	if _, err := shellRepresentationInputOperation(root, "shell-representation-successor", sitecontent.Reference{Path: "plan.json", SHA256: strings.Repeat("0", 64)}, false); err == nil {
		t.Fatal("wrong input SHA accepted")
	}
}
