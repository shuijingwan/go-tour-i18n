package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// This is a focused docs contract check, not a registry or a claim that a test
// can prove a Coordinator has read the current implementation before handoff.
func TestFormalCommandFirstRealUseContract(t *testing.T) {
	b, err := os.ReadFile("../../docs/WORKFLOW_HANDOFF.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(b)
	start := strings.Index(doc, "COMMAND_CONTRACT_VERIFIED=<yes|no>")
	if start < 0 {
		t.Fatal("missing first-real-use handoff template")
	}
	end := strings.Index(doc[start:], "```")
	if end < 0 {
		t.Fatal("unterminated first-real-use template")
	}
	fields := map[string]bool{}
	for _, line := range strings.Split(doc[start:start+end], "\n") {
		if key, value, ok := strings.Cut(line, "="); ok && value != "" {
			fields[key] = true
		}
	}
	for _, name := range []string{"COMMAND_CONTRACT_VERIFIED", "COMMAND", "OPERATION", "INPUT_KIND", "INPUT_PATH_CONTRACT", "MUTATION_EXPECTATION", "IDEMPOTENCE", "CURRENTNESS_CHECK", "PRIOR_OR_CAS", "SUCCESS_STOP", "FAILURE_STOP"} {
		if !fields[name] {
			t.Errorf("first-real-use handoff missing %s", name)
		}
	}
	agents, err := os.ReadFile("../../AGENTS.md")
	if err != nil || !strings.Contains(string(agents), "docs/WORKFLOW_HANDOFF.md#formal-command-contract-v3") {
		t.Fatal("AGENTS first-real-use gate navigation missing", err)
	}
	b, err = os.ReadFile("../../docs/SHELL_REPRESENTATION_SUCCESSOR.md")
	if err != nil {
		t.Fatal(err)
	}
	matrix := string(b)
	for _, name := range []string{"COMMAND / mode", "OPERATION", "INPUT_KIND", "PATH CONTRACT", "MUTATION", "SUCCESS RESULT", "STOP CONDITION"} {
		if !strings.Contains(matrix, "| "+name+" |") {
			t.Errorf("Shell command matrix missing %s", name)
		}
	}
	operations := map[string]string{}
	commands := map[string]string{}
	paths := map[string]string{}
	for _, line := range strings.Split(matrix, "\n") {
		if !strings.HasPrefix(line, "| ") || strings.Contains(line, "COMMAND / mode") || strings.HasPrefix(line, "| ---") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if err := validateFormalCommandMatrixRow(cells); err != nil {
			t.Fatal(err)
		}
		operation := strings.TrimSpace(cells[1])
		if _, exists := operations[operation]; exists {
			t.Fatal("ambiguous command operation in matrix", operation)
		}
		operations[operation] = strings.TrimSpace(cells[2]) + "/" + strings.TrimSpace(cells[4])
		commands[operation] = strings.TrimSpace(cells[0])
		paths[operation] = strings.TrimSpace(cells[3])
	}
	for operation, command := range map[string]string{
		"plan-inspection":      "`shell-representation-successor --input <plan> --input-sha256 <sha>`",
		"receipt-verification": "`shell-representation-check --input <receipt> --input-sha256 <sha>`",
	} {
		if commands[operation] != command || paths[operation] != "repository-relative" {
			t.Errorf("%s command/input-path drift: %q / %q", operation, commands[operation], paths[operation])
		}
	}
	for _, operation := range []string{"plan-landing", "apply", "recovery-diagnosis"} {
		if _, exists := operations[operation]; !exists {
			t.Error("missing command lifecycle mode", operation)
		}
	}
	for operation, want := range map[string]string{
		"prospective-preflight":      "parser-rule/none",
		"plan-inspection":            "plan/none",
		"receipt-verification":       "receipt/none",
		"already-current-inspection": "plan/none",
	} {
		if operations[operation] != want {
			t.Errorf("%s: got %q; want %q", operation, operations[operation], want)
		}
	}
	// A command name alone cannot satisfy the concrete first-use matrix.
	for _, cells := range [][]string{
		{"shell-representation-check"},
		{"shell-representation-check", "", "receipt", "repository-relative", "none", "receipt", "stop"},
		{"shell-representation-check", "check", "", "repository-relative", "none", "receipt", "stop"},
		{"shell-representation-check", "check", "receipt", "repository-relative", "", "receipt", "stop"},
	} {
		if validateFormalCommandMatrixRow(cells) == nil {
			t.Fatal("incomplete first-use command contract accepted", cells)
		}
	}
}

func validateFormalCommandMatrixRow(cells []string) error {
	if len(cells) != 7 {
		return fmt.Errorf("command requires operation/input/path/mutation/result/stop contract")
	}
	for i, cell := range cells {
		if strings.TrimSpace(cell) == "" {
			return fmt.Errorf("command contract missing column %d", i)
		}
	}
	return nil
}
