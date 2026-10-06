package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func glossaryCompatibilityCommand(root string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: glossary-compatibility <archive|assess|check|status|config-baseline> --locale <locale>")
	}
	fs := flag.NewFlagSet("glossary-compatibility "+args[0], flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	all := fs.Bool("all", false, "bootstrap current 65 locale immutable archives")
	old := fs.String("old-sha256", "", "old archived glossary SHA-256")
	scope := fs.String("scope", "*", "exact artifact scope (tu:, seo:, ui:, article:, surface:) or *")
	evidence := fs.String("evidence", "", "repository-relative compatibility evidence path")
	review := fs.String("review-id", "", "historical Surface review ID")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	if *all && (*locale != "" || (args[0] != "archive" && args[0] != "config-baseline")) {
		return fmt.Errorf("--all is only valid for archive/config-baseline without --locale")
	}
	allowed := map[string]map[string]bool{
		"archive": {"locale": true, "all": true}, "config-baseline": {"locale": true, "all": true, "review-id": true},
		"assess": {"locale": true, "old-sha256": true}, "check": {"locale": true, "evidence": true}, "status": {"locale": true, "old-sha256": true, "scope": true},
	}
	if allowed[args[0]] == nil {
		return fmt.Errorf("unknown glossary-compatibility command %q", args[0])
	}
	var invalidFlag string
	fs.Visit(func(f *flag.Flag) {
		if !allowed[args[0]][f.Name] {
			invalidFlag = f.Name
		}
	})
	if invalidFlag != "" {
		return fmt.Errorf("--%s is not valid for %s", invalidFlag, args[0])
	}
	write := func(v any) error { return json.NewEncoder(os.Stdout).Encode(v) }
	if args[0] == "archive" {
		locales := []string{*locale}
		if *all {
			var err error
			locales, err = sitecontent.LiveLocales(root)
			if err != nil {
				return err
			}
		}
		// Preflight full-byte current Review for the complete requested cohort.
		for _, l := range locales {
			if err := i18n.RequireCurrentGlossaryReview(root, l); err != nil {
				return err
			}
		}
		refs := map[string]i18n.GlossaryArchiveReference{}
		for _, l := range locales {
			ref, err := i18n.ArchiveCurrentGlossary(root, l)
			if err != nil {
				return err
			}
			refs[l] = ref
		}
		return write(refs)
	}
	if args[0] == "config-baseline" {
		if *all {
			if *review != "" {
				return fmt.Errorf("--review-id cannot be combined with --all")
			}
			locales, err := sitecontent.LiveLocales(root)
			if err != nil {
				return err
			}
			paths, err := i18n.BootstrapTourSurfaceConfigBaselines(root, locales)
			if err != nil {
				return err
			}
			return write(paths)
		}
		path, err := i18n.RecordTourSurfaceConfigBaseline(root, *locale, *review)
		if err != nil {
			return err
		}
		return write(map[string]string{"path": path})
	}
	if err := i18n.ValidateLocaleName(*locale); err != nil {
		return err
	}
	if args[0] == "check" {
		if *evidence == "" || filepath.IsAbs(*evidence) || strings.Contains(*evidence, "\\") {
			return fmt.Errorf("--evidence requires a repository path")
		}
		if err := i18n.CheckGlossaryCompatibilityEvidence(root, *locale, *evidence, nil); err != nil {
			return err
		}
		return write(map[string]string{"status": "CURRENT", "evidence": *evidence})
	}
	if args[0] == "assess" {
		e, path, err := i18n.AssessGlossaryCompatibility(root, *locale, *old, nil)
		if err != nil {
			return err
		}
		return write(struct {
			Path     string                              `json:"path"`
			Evidence *i18n.GlossaryCompatibilityEvidence `json:"evidence"`
		}{path, e})
	}
	if args[0] == "status" {
		return write(i18n.GlossaryCompatibilityStatus(root, *locale, *old, *scope, nil))
	}
	return fmt.Errorf("unknown glossary-compatibility command %q", args[0])
}
