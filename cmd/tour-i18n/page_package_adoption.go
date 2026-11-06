package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func pagePackageAdoptionCommand(root string, args []string) error {
	f := flag.NewFlagSet(args[0], flag.ContinueOnError)
	input := f.String("input", "", "repository-relative proposal or recorded adoption")
	sha := f.String("input-sha256", "", "exact input SHA-256")
	prior := f.String("prior-identity", "", "exact predecessor scope identity; apply only")
	apply := f.Bool("apply", false, "publish source registration only; no language state")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	if f.NArg() != 0 || *input == "" || *sha == "" {
		return fmt.Errorf("--input and --input-sha256 are required; no positional arguments")
	}
	ref, err := sitecontent.ReferenceFile(root, *input)
	if err != nil {
		return err
	}
	if ref.SHA256 != *sha {
		return fmt.Errorf("input SHA mismatch")
	}
	if args[0] == "page-package-adoption-check" {
		if *apply || *prior != "" {
			return fmt.Errorf("check is readonly")
		}
		a, err := sitecontent.CheckPagePackageAdoption(root, ref)
		if err != nil {
			return err
		}
		g, err := sitecontent.LoadCurrent(root)
		if err != nil {
			return err
		}
		found := false
		for _, current := range g.PackageAdoptions {
			found = found || current == ref
		}
		if !found {
			return fmt.Errorf("valid receipt is not current source authority")
		}
		return printJSON(struct {
			State     string                           `json:"state"`
			Authority *sitecontent.PagePackageAdoption `json:"authority"`
			Mutation  string                           `json:"mutation"`
		}{"CURRENT", a, "none"})
	}
	var a *sitecontent.PagePackageAdoption
	if *apply {
		a, err = sitecontent.AdoptPagePackage(root, ref, *prior)
	} else {
		if *prior != "" {
			return fmt.Errorf("--prior-identity requires --apply")
		}
		a, err = sitecontent.BuildPagePackageAdoption(root, ref)
	}
	if err != nil {
		return err
	}
	mutation := "none"
	if *apply {
		mutation = "source-authority-only"
	}
	return printJSON(struct {
		Authority *sitecontent.PagePackageAdoption `json:"authority"`
		Path      string                           `json:"receipt_path"`
		Applied   bool                             `json:"applied"`
		Mutation  string                           `json:"mutation"`
	}{a, sitecontent.PagePackageAdoptionPath(a), *apply, mutation})
}
