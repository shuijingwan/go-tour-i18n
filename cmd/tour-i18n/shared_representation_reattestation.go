package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func sharedRepresentationReattestationCommand(root string, args []string) error {
	f := flag.NewFlagSet("shared-structured-representation-reattest", flag.ContinueOnError)
	base := f.String("base", "", "repository-relative installed shared base receipt")
	baseSHA := f.String("base-sha256", "", "exact base receipt file SHA256")
	input := f.String("input", "", "repository-relative reviewed re-attestation plan")
	sha := f.String("input-sha256", "", "exact reviewed plan file SHA256")
	apply := f.Bool("apply", false, "publish dependency re-attestation only")
	prior := f.String("prior-pointer-sha256", "", "exact prior re-attestation pointer SHA256")
	if err := f.Parse(args); err != nil {
		return err
	}
	supplied := map[string]bool{}
	f.Visit(func(f *flag.Flag) { supplied[f.Name] = true })
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected shared re-attestation arguments")
	}
	if *base != "" {
		if *baseSHA == "" || supplied["input"] || supplied["input-sha256"] || supplied["apply"] || supplied["prior-pointer-sha256"] {
			return fmt.Errorf("prospective requires only exact --base and --base-sha256")
		}
		ref := sitecontent.Reference{Path: *base, SHA256: *baseSHA}
		var r sitecontent.SharedRepresentationSuccessor
		if err := readSharedReattestationCLIInput(root, ref, &r, sitecontent.SharedRepresentationSchema); err != nil {
			return err
		}
		p, err := sitecontent.PlanSharedRepresentationReattestation(root, ref)
		if err != nil {
			return err
		}
		return printJSON(p)
	}
	if *input == "" || *sha == "" || supplied["base"] || supplied["base-sha256"] || (!*apply && (supplied["apply"] || supplied["prior-pointer-sha256"])) || (*apply && *prior == "") {
		return fmt.Errorf("exact reviewed plan input/SHA and mode-specific CAS required")
	}
	ref := sitecontent.Reference{Path: *input, SHA256: *sha}
	var p sitecontent.SharedRepresentationReattestationPlan
	if err := readSharedReattestationCLIInput(root, ref, &p, sitecontent.SharedReattestationSchema+"/plan"); err != nil {
		return err
	}
	if *apply {
		r, err := sitecontent.ApplySharedRepresentationReattestation(root, ref, *prior)
		if err != nil {
			return err
		}
		return printJSON(r)
	}
	r, err := sitecontent.InspectSharedRepresentationReattestation(root, ref)
	if err != nil {
		return err
	}
	return printJSON(r)
}

func readSharedReattestationCLIInput(root string, ref sitecontent.Reference, out any, schema string) error {
	b, err := sitecontent.ReadArtifact(root, ref.Path)
	if err != nil {
		return err
	}
	if contentidentity.SHA256Hex(b) != ref.SHA256 {
		return fmt.Errorf("exact shared re-attestation input SHA required")
	}
	if err := sitecontent.StrictJSON(b, out); err != nil {
		return err
	}
	actual := ""
	switch r := out.(type) {
	case *sitecontent.SharedRepresentationSuccessor:
		actual = r.Schema
	case *sitecontent.SharedRepresentationReattestationPlan:
		actual = r.Schema
	}
	if actual != schema {
		return fmt.Errorf("shared re-attestation input schema %s required", schema)
	}
	return nil
}
