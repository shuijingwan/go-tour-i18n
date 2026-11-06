package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func siteSourceCommand(root string, args []string) error {
	f := flag.NewFlagSet("site-content "+args[0], flag.ContinueOnError)
	website := f.String("website-root", "", "read-only Git object source at the exact frozen commit")
	apply := f.Bool("apply", false, "install only missing adopted source; preserve all existing bytes")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	var invalidFlag bool
	f.Visit(func(flag *flag.Flag) {
		invalidFlag = invalidFlag || (args[0] != "source-migrate" && !(args[0] == "source-storage-compatibility" && flag.Name == "apply"))
	})
	if f.NArg() != 0 || invalidFlag {
		return fmt.Errorf("unexpected source authority arguments")
	}
	switch args[0] {
	case "source-storage-compatibility":
		p, path, err := sitecontent.CreateSourceStorageCompatibility(root, *apply)
		if err != nil {
			return err
		}
		return printJSON(struct {
			Path       string `json:"path"`
			Identity   string `json:"identity"`
			References int    `json:"covered_corpus_references"`
			Entries    int    `json:"covered_source_entries"`
		}{path, p.Identity, len(p.Corpora), len(p.Entries)})
	case "source-migrate":
		if *website == "" {
			return fmt.Errorf("--website-root is required")
		}
		r, err := sitecontent.MigrateFrozenSources(root, *website, *apply)
		if err != nil {
			return err
		}
		return printJSON(r)
	case "source-check":
		m, err := sitecontent.CheckFrozenSources(root)
		if err != nil {
			return err
		}
		g, err := sitecontent.LoadCurrent(root)
		if err != nil {
			return err
		}
		return printJSON(struct {
			ManifestIdentity string `json:"manifest_identity"`
			GlobalIdentity   string `json:"unchanged_global_identity"`
		}{m.Identity, g.Identity})
	case "source-retirement-check":
		r, err := sitecontent.CheckSourceZIPRetirement(root)
		if err != nil {
			return err
		}
		if err := printJSON(r); err != nil {
			return err
		}
		if r.Status != "READY" {
			return fmt.Errorf("ZIP_RETIREMENT_BLOCKED: exact storage compatibility missing")
		}
		return nil
	}
	return fmt.Errorf("unknown source authority command")
}
