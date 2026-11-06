package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func shellSuccessorCommand(root string, args []string) error {
	f := flag.NewFlagSet("site-content "+args[0], flag.ContinueOnError)
	locale := f.String("locale", "", "accepted predecessor locale for exact affected scope")
	input := f.String("input", "", "repository-relative successor affected artifact")
	apply := f.Bool("apply", false, "save additive immutable source and affected-scope authority only")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	var invalid bool
	f.Visit(func(v *flag.Flag) {
		if args[0] == "shell-successor-check" {
			invalid = invalid || v.Name != "input"
		} else {
			invalid = invalid || v.Name == "input"
		}
	})
	if f.NArg() != 0 || invalid {
		return fmt.Errorf("unexpected shell successor flags")
	}
	if args[0] == "shell-successor-check" {
		if *input == "" {
			return fmt.Errorf("--input is required")
		}
		s, err := sitecontent.CheckShellSuccessorFoundation(root, *input)
		if err != nil {
			return err
		}
		return printJSON(struct{ State, SourceIdentity, ScopeIdentity string }{"CURRENT", s.SourceIdentity, s.Identity})
	}
	if *locale == "" {
		return fmt.Errorf("--locale is required")
	}
	if *apply {
		a, s, name, err := sitecontent.SaveShellSuccessorFoundation(root, *locale)
		if err != nil {
			return err
		}
		return printJSON(struct {
			State          string `json:"state"`
			SourcePath     string `json:"source_authority_path"`
			SourceIdentity string `json:"source_identity_sha256"`
			AffectedPath   string `json:"affected_scope_path"`
			ScopeIdentity  string `json:"scope_identity_sha256"`
		}{"PENDING", s.SourceAuthority.Path, a.Identity, name, s.Identity})
	}
	a, err := sitecontent.BuildShellSuccessorAuthority(root)
	if err != nil {
		return err
	}
	s, err := sitecontent.BuildShellSuccessorAffectedScope(root, *locale, a)
	if err != nil {
		return err
	}
	return printJSON(s)
}
