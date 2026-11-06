package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func sharedStructuredRepresentationCommand(root string, args []string) error {
	f := flag.NewFlagSet("shared-structured-representation-successor", flag.ContinueOnError)
	request := f.String("request", "", "repository-relative exact source-coordinate request")
	requestSHA := f.String("request-sha256", "", "exact request file SHA256")
	input := f.String("input", "", "repository-relative reviewed plan")
	inputSHA := f.String("input-sha256", "", "exact plan file SHA256")
	apply := f.Bool("apply", false, "publish immutable authority and keyed current head")
	prior := f.String("prior-identity", "", "exact predecessor representation identity")
	pointer := f.String("prior-pointer-sha256", "", "exact previous keyed head SHA256")
	if err := f.Parse(args); err != nil {
		return err
	}
	supplied := map[string]bool{}
	f.Visit(func(flag *flag.Flag) { supplied[flag.Name] = true })
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected shared representation arguments")
	}
	if *request != "" {
		if *requestSHA == "" || supplied["input"] || supplied["input-sha256"] || supplied["apply"] || supplied["prior-identity"] || supplied["prior-pointer-sha256"] {
			return fmt.Errorf("prospective mode requires only --request and --request-sha256")
		}
		ref := sitecontent.Reference{Path: *request, SHA256: *requestSHA}
		var q sitecontent.SharedRepresentationRequest
		if err := readSharedRepresentationCLIInput(root, ref, &q, sitecontent.SharedRepresentationSchema+"/request"); err != nil {
			return err
		}
		p, err := sitecontent.PlanSharedStructuredRepresentation(root, ref)
		if err != nil {
			return err
		}
		return printJSON(p)
	}
	if *input == "" || *inputSHA == "" || supplied["request"] || supplied["request-sha256"] || (!*apply && (supplied["apply"] || supplied["prior-identity"] || supplied["prior-pointer-sha256"])) || (*apply && (*prior == "" || *pointer == "")) {
		return fmt.Errorf("exact plan input/SHA and mode-specific CAS flags required")
	}
	ref := sitecontent.Reference{Path: *input, SHA256: *inputSHA}
	var p sitecontent.SharedRepresentationPlan
	if err := readSharedRepresentationCLIInput(root, ref, &p, sitecontent.SharedRepresentationSchema+"/plan"); err != nil {
		return err
	}
	if *apply {
		r, err := sitecontent.ApplySharedStructuredRepresentation(root, ref, *prior, *pointer)
		if err != nil {
			return err
		}
		return printJSON(r)
	}
	r, err := sitecontent.InspectSharedStructuredRepresentation(root, ref)
	if err != nil {
		return err
	}
	return printJSON(r)
}

func readSharedRepresentationCLIInput(root string, ref sitecontent.Reference, out any, schema string) error {
	b, err := sitecontent.ReadArtifact(root, ref.Path)
	if err != nil {
		return err
	}
	if contentidentity.SHA256Hex(b) != ref.SHA256 {
		return fmt.Errorf("exact shared representation input SHA required")
	}
	if err := sitecontent.StrictJSON(b, out); err != nil {
		return err
	}
	actual := ""
	switch x := out.(type) {
	case *sitecontent.SharedRepresentationRequest:
		actual = x.Schema
	case *sitecontent.SharedRepresentationPlan:
		actual = x.Schema
	}
	if actual != schema {
		return fmt.Errorf("shared representation input schema %s required", schema)
	}
	return nil
}
