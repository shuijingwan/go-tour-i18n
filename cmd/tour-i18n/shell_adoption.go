package main

import (
	"flag"
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func shellAdoptionCommand(root string, args []string) error {
	f := flag.NewFlagSet(args[0], flag.ContinueOnError)
	input := f.String("input", "", "frozen shell foundation / adoption artifact")
	prior := f.String("prior-sha256", "", "exact historical global SHA for source adoption CAS")
	locale := f.String("locale", "", "locale for shared Homepage plan")
	apply := f.Bool("apply", false, "adopt current shell source authority; no locale activation")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	if args[0] == "homepage-plan" {
		if *locale == "" || *input != "" || *prior != "" || *apply {
			return fmt.Errorf("homepage-plan requires only --locale")
		}
		p, err := sitecontent.PlanHomepage(root, *locale)
		if err != nil {
			return err
		}
		return printJSON(p)
	}
	if *input == "" || *locale != "" {
		return fmt.Errorf("--input required")
	}
	ref, err := sitecontent.ReferenceFile(root, *input)
	if err != nil {
		return err
	}
	if args[0] == "shell-adoption-check" {
		if *apply || *prior != "" {
			return fmt.Errorf("unexpected check flags")
		}
		a, err := sitecontent.CheckShellAdoption(root, ref)
		if err != nil {
			return err
		}
		return printJSON(struct{ State, Identity string }{"CURRENT", a.Identity})
	}
	var a *sitecontent.ShellAdoption
	if *apply {
		a, err = sitecontent.AdoptShellSuccessor(root, ref, *prior)
	} else {
		if *prior != "" {
			return fmt.Errorf("CAS applies only with --apply")
		}
		a, err = sitecontent.BuildShellAdoption(root, ref)
	}
	if err != nil {
		return err
	}
	return printJSON(a)
}
