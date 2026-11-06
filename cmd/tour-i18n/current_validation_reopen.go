package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func parseCurrentValidationReopenOptions(args []string) (i18n.CurrentValidationReopenOptions, bool, error) {
	fs := flag.NewFlagSet("quality-check current-validation-reopen", flag.ContinueOnError)
	var o i18n.CurrentValidationReopenOptions
	var packages repeatedStrings
	fs.Var(&packages, "content-package", "required complete canonical Tour-native tour-v1 or registered content package")
	fs.StringVar(&o.Locale, "locale", "", "target locale")
	fs.StringVar(&o.ReopenID, "reopen-id", "", "shared immutable reopen ID")
	fs.StringVar(&o.PreviousSnapshotID, "previous-snapshot-id", "", "finalized predecessor")
	fs.StringVar(&o.SnapshotSHA256, "previous-snapshot-sha256", "", "exact manifest SHA256")
	fs.StringVar(&o.Evidence.Path, "current-reattestation", "", "complete current re-attestation JSON path")
	fs.StringVar(&o.Evidence.SHA256, "current-reattestation-sha256", "", "exact evidence bytes SHA256")
	fs.StringVar(&o.AssessmentIdentity, "current-reattestation-identity", "", "exact assessment identity_sha256")
	var ids repeatedStrings
	fs.Var(&ids, "unit-id", "optional exact complete affected Unit set, stable order")
	record := fs.Bool("record", false, "explicitly record shared reopen; default read-only preflight")
	if err := fs.Parse(args); err != nil {
		return o, false, err
	}
	o.UnitIDs = ids
	if len(packages) != 1 || packages[0] == "" {
		return o, false, fmt.Errorf("exactly one explicit --content-package is required")
	}
	o.ContentPackage = packages[0]
	if fs.NArg() != 0 || o.Locale == "" || o.ReopenID == "" || o.PreviousSnapshotID == "" || o.SnapshotSHA256 == "" || o.Evidence.Path == "" || o.Evidence.SHA256 == "" || o.AssessmentIdentity == "" {
		return o, false, fmt.Errorf("exact locale/reopen/predecessor path-SHA/re-attestation path-SHA-identity required; select complete --content-package")
	}
	return o, *record, nil
}

func currentValidationReopenCommand(root string, c *i18n.Catalog, args []string) error {
	o, record, err := parseCurrentValidationReopenOptions(args)
	if err != nil {
		return err
	}
	c, err = i18n.SelectCurrentValidationReopenPackage(c, o.ContentPackage)
	if err != nil {
		return err
	}
	var r *i18n.QualityCheckSurfaceReopen
	var path string
	if record {
		r, path, err = i18n.RecordQualityCheckSurfaceReopen(root, c, i18n.QualityCheckSurfaceReopenOptions{AuthoritySource: i18n.CurrentValidationAuthority, Locale: o.Locale, ReopenID: o.ReopenID, PreviousSnapshotID: o.PreviousSnapshotID, CurrentValidation: &o})
	} else {
		r, path, err = i18n.InspectCurrentValidationReopen(root, c, o)
	}
	if err != nil {
		return err
	}
	mutation := "none"
	if record {
		mutation = "known"
	}
	return printJSON(map[string]any{"authority": r, "receipt_path": path, "recorded": record, "mutation": mutation})
}
