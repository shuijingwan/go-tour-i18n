package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func shellRepresentationReattestationCommand(root string, args []string) error {
	f := flag.NewFlagSet("shell-representation-reattest", flag.ContinueOnError)
	input := f.String("input", "", "repository-relative dependency re-attestation plan path")
	sha := f.String("input-sha256", "", "exact plan file SHA256")
	apply := f.Bool("apply", false, "publish dependency binding only; never rewrite base successor")
	prior := f.String("prior-pointer-sha256", "", "exact re-attestation plan prior pointer SHA256")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("no positional arguments allowed")
	}
	if *input == "" {
		if f.NFlag() != 0 {
			return fmt.Errorf("readonly prospective preflight takes no flags")
		}
		p, err := sitecontent.PlanShellRepresentationReattestation(root)
		if err != nil {
			return err
		}
		return printJSON(p)
	}
	if *sha == "" || (*apply && *prior == "") || (!*apply && *prior != "") {
		return fmt.Errorf("input requires exact --input-sha256; only --apply requires --prior-pointer-sha256")
	}
	ref := sitecontent.Reference{Path: *input, SHA256: *sha}
	if err := validateShellReattestationInput(root, ref); err != nil {
		return err
	}
	if *apply {
		r, err := sitecontent.ApplyShellRepresentationReattestation(root, ref, *prior)
		if err != nil {
			return err
		}
		return printJSON(r)
	}
	r, err := sitecontent.InspectShellRepresentationReattestation(root, ref)
	if err != nil {
		return err
	}
	return printJSON(r)
}

func validateShellReattestationInput(root string, ref sitecontent.Reference) error {
	raw, err := sitecontent.ReadArtifact(root, ref.Path)
	if err != nil {
		return fmt.Errorf("--input requires a readable safe repository-relative path: %w", err)
	}
	if contentidentity.SHA256Hex(raw) != ref.SHA256 {
		return fmt.Errorf("--input-sha256 mismatch")
	}
	var p sitecontent.ShellRepresentationReattestationPlan
	if err := sitecontent.StrictJSON(raw, &p); err != nil {
		return fmt.Errorf("dependency re-attestation requires plan input: %w", err)
	}
	if p.Schema != sitecontent.ShellRepresentationReattestationSchema+"/plan" {
		return fmt.Errorf("dependency re-attestation requires plan input schema")
	}
	return nil
}
