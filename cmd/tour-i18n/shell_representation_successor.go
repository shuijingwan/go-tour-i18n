package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func shellRepresentationCommand(root string, args []string) error {
	f := flag.NewFlagSet(args[0], flag.ContinueOnError)
	input := f.String("input", "", "repository-relative plan path for successor, receipt path for check")
	sha := f.String("input-sha256", "", "exact input file SHA256")
	parser := f.String("parser", "", "explicit prospective representation contract")
	source := f.String("source", "_content/index.md", "canonical Homepage source")
	pkg := f.String("package", "site-v2-shell", "source package")
	apply := f.Bool("apply", false, "publish exact canonical representation authority; no language acceptance")
	prior := f.String("prior-identity", "", "exact predecessor representation identity")
	pointer := f.String("prior-pointer-sha256", "", "exact predecessor pointer file SHA256; absent=digest(empty)")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	if f.NArg() != 0 || *source != "_content/index.md" || *pkg != "site-v2-shell" {
		return fmt.Errorf("exact Site shell Homepage scope required")
	}
	if *input == "" {
		if args[0] != "shell-representation-successor" || *parser == "" || *sha != "" || *apply || *prior != "" || *pointer != "" {
			return fmt.Errorf("prospective preflight requires only --parser and optional canonical --source/--package")
		}
		p, err := sitecontent.PlanShellRepresentationSuccessor(root, *parser)
		if err != nil {
			return err
		}
		return printJSON(p)
	}
	if *sha == "" || *parser != "" {
		return fmt.Errorf("exact --input/--input-sha256 required, without --parser")
	}
	ref := sitecontent.Reference{Path: *input, SHA256: *sha}
	if args[0] == "shell-representation-check" {
		if *apply || *prior != "" || *pointer != "" {
			return fmt.Errorf("check is readonly")
		}
	}
	if !*apply && (*prior != "" || *pointer != "") {
		return fmt.Errorf("CAS flags require --apply")
	}
	if *apply && (*prior == "" || *pointer == "") {
		return fmt.Errorf("--apply requires exact --prior-identity and --prior-pointer-sha256")
	}
	operation, err := shellRepresentationInputOperation(root, args[0], ref, *apply)
	if err != nil {
		return err
	}
	switch operation {
	case "receipt-verification":
		r, err := sitecontent.CheckShellRepresentationSuccessor(root, ref)
		if err != nil {
			return err
		}
		return printJSON(r)
	case "plan-inspection":
		r, err := sitecontent.InspectShellRepresentationSuccessor(root, ref)
		if err != nil {
			return err
		}
		return printJSON(r)
	case "apply":
		r, err := sitecontent.ApplyShellRepresentationSuccessor(root, ref, *prior, *pointer)
		if err != nil {
			return err
		}
		return printJSON(r)
	default:
		return fmt.Errorf("unknown Shell representation operation %q", operation)
	}
}

// Reject wrong input kinds before reconstructing source authority. This only
// selects the operation; each reader/publisher still rereads and fully validates
// the exact input, current dependencies, lineage and CAS under its own guard.
func shellRepresentationInputOperation(root, command string, ref sitecontent.Reference, apply bool) (string, error) {
	var target any
	var plan sitecontent.ShellRepresentationPlan
	var receipt sitecontent.ShellRepresentationSuccessor
	operation, kind := "plan-inspection", "plan"
	switch command {
	case "shell-representation-successor":
		target = &plan
		if apply {
			operation = "apply"
		}
	case "shell-representation-check":
		if apply {
			return "", fmt.Errorf("receipt verification is readonly")
		}
		operation, kind, target = "receipt-verification", "receipt", &receipt
	default:
		return "", fmt.Errorf("unknown Shell representation command %q", command)
	}
	raw, err := sitecontent.ReadArtifact(root, ref.Path)
	if err != nil {
		return "", fmt.Errorf("--input requires a readable safe repository-relative path: %w", err)
	}
	if contentidentity.SHA256Hex(raw) != ref.SHA256 {
		return "", fmt.Errorf("--input-sha256 mismatch")
	}
	if err := sitecontent.StrictJSON(raw, target); err != nil {
		return "", fmt.Errorf("%s requires %s input: %w", command, kind, err)
	}
	if (kind == "plan" && plan.Schema != sitecontent.ShellRepresentationPlanSchema) ||
		(kind == "receipt" && receipt.Schema != sitecontent.ShellRepresentationSchema) {
		return "", fmt.Errorf("%s requires %s input schema", command, kind)
	}
	return operation, nil
}
