package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func parseRepresentationReconciliationOptions(args []string) (i18n.RepresentationReconciliationOptions, error) {
	fs := flag.NewFlagSet("quality-check representation-reconcile", flag.ContinueOnError)
	var o i18n.RepresentationReconciliationOptions
	addRepresentationReconciliationFlags(fs, &o)
	fs.BoolVar(&o.Apply, "apply", false, "materialize machine evidence only after complete exact-set preflight")
	fs.Func("language-authority-review-id", "exact immutable independent semantic review ID (repeatable)", func(s string) error { o.LanguageReviewIDs = append(o.LanguageReviewIDs, s); return nil })
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if err := validateRepresentationOptions(fs, o); err != nil {
		return o, err
	}
	return o, nil
}
func addRepresentationReconciliationFlags(fs *flag.FlagSet, o *i18n.RepresentationReconciliationOptions) {
	fs.StringVar(&o.Locale, "locale", "", "target locale")
	fs.StringVar(&o.ReconciliationID, "reconciliation-id", "", "immutable machine reconciliation ID")
	fs.StringVar(&o.PreviousSnapshotID, "previous-snapshot-id", "", "trusted reviewed predecessor Snapshot")
	fs.StringVar(&o.SnapshotSHA256, "previous-snapshot-sha256", "", "exact predecessor manifest SHA256")
	fs.StringVar(&o.Impact.Path, "formal-impact", "", "explicit representation impact evidence")
	fs.StringVar(&o.Impact.SHA256, "formal-impact-sha256", "", "exact impact evidence SHA256")
	fs.StringVar(&o.HistoricalDocuments.Path, "historical-documents", "", "complete frozen old representations")
	fs.StringVar(&o.HistoricalDocuments.SHA256, "historical-documents-sha256", "", "exact historical representations SHA256")
}
func validateRepresentationOptions(fs *flag.FlagSet, o i18n.RepresentationReconciliationOptions) error {
	if fs.NArg() != 0 || o.Locale == "" || o.ReconciliationID == "" || o.PreviousSnapshotID == "" || o.SnapshotSHA256 == "" || o.Impact.Path == "" || o.Impact.SHA256 == "" || o.HistoricalDocuments.Path == "" || o.HistoricalDocuments.SHA256 == "" {
		return fmt.Errorf("explicit locale, reconciliation, Snapshot, impact and frozen representation path/SHA pairs required; select --content-package")
	}
	return nil
}
func reconcileRepresentationsCommand(root string, c *i18n.Catalog, args []string) error {
	o, err := parseRepresentationReconciliationOptions(args)
	if err != nil {
		return err
	}
	plan, err := i18n.ReconcileTranslationUnitRepresentations(root, c, o)
	if plan != nil {
		if outputErr := printJSON(plan); outputErr != nil {
			return outputErr
		}
	}
	if err != nil {
		return err
	}
	if plan.Blocked > 0 {
		return fmt.Errorf("representation reconciliation has %d unresolved proofs; mutation=none", plan.Blocked)
	}
	return nil
}
