package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func parseQualityCheckSnapshotOptions(args []string) (i18n.QualityCheckSnapshotOptions, bool, error) {
	var o i18n.QualityCheckSnapshotOptions
	f := flag.NewFlagSet("quality-check snapshot", flag.ContinueOnError)
	f.StringVar(&o.Locale, "locale", "", "target locale")
	f.StringVar(&o.SnapshotID, "snapshot-id", "", "new immutable Snapshot ID; never determines package")
	var packages repeatedStrings
	f.Var(&packages, "content-package", "complete registered package; omitted selects native Tour")
	preflight := f.Bool("preflight", false, "readonly exact Snapshot selection/currentness; creates no evidence")
	if err := f.Parse(args); err != nil {
		return o, false, err
	}
	if o.Locale == "" || o.SnapshotID == "" || f.NArg() != 0 || len(packages) > 1 || (len(packages) == 1 && packages[0] == "") {
		return o, false, fmt.Errorf("--locale and new --snapshot-id required; optional single nonempty --content-package; no positional arguments or Unit subset")
	}
	if len(packages) == 1 {
		o.ContentPackage = packages[0]
	}
	return o, *preflight, nil
}

func qualityCheckSnapshotCommand(root string, c *i18n.Catalog, args []string) error {
	o, preflight, err := parseQualityCheckSnapshotOptions(args)
	if err != nil {
		return err
	}
	if c == nil || (o.ContentPackage == "" && len(c.TranslationScope) > 0) {
		return fmt.Errorf("Snapshot CLI requires explicit --content-package for a non-native catalog; no hidden scope inference")
	}
	if preflight {
		m, err := i18n.PrepareQualityCheckCandidateSnapshot(root, c, o)
		if err != nil {
			return err
		}
		return printJSON(struct {
			Manifest *i18n.QualityCheckSnapshotManifest `json:"manifest"`
			Recorded bool                               `json:"recorded"`
			Mutation string                             `json:"mutation"`
		}{m, false, "none"})
	}
	m, path, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, o)
	if err != nil {
		return err
	}
	fmt.Printf("wrote quality-check candidate snapshot: %s (%d units: %d pages, %d examples)\n", path, m.UnitCount, m.PageCount, m.ExampleCount)
	if m.ContentPackage != nil {
		fmt.Printf("content_package: %s\npackage_authority_sha256: %s\nscope_identity_sha256: %s\n", m.ContentPackage.ID, m.ContentPackage.AuthoritySHA256, m.ContentPackage.ScopeIdentitySHA256)
	}
	return nil
}
