package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"os"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func representationReviewCommand(root string, c *i18n.Catalog, action string, args []string) error {
	fs := flag.NewFlagSet("quality-check representation-review-"+action, flag.ContinueOnError)
	var o i18n.RepresentationReviewOptions
	var output, bundle, evidence string
	if action == "bundle" {
		addRepresentationReconciliationFlags(fs, &o.Reconciliation)
		fs.StringVar(&o.ReviewID, "review-id", "", "exact independent semantic question ID")
		fs.StringVar(&o.UnitID, "id", "", "single unresolved current Unit")
		fs.StringVar(&output, "output", "", "new Reviewer transport ZIP path")
	} else {
		fs.StringVar(&bundle, "bundle", "", "self-contained Reviewer ZIP")
		if action == "record" {
			fs.StringVar(&evidence, "evidence", "", "exact independent Reviewer JSON response")
		}
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected semantic review arguments")
	}
	if action == "bundle" {
		if err := validateRepresentationOptions(fs, o.Reconciliation); err != nil {
			return err
		}
		if o.ReviewID == "" || o.UnitID == "" || output == "" {
			return fmt.Errorf("explicit review-id, id and output required")
		}
		b, m, err := i18n.ExportRepresentationReviewBundle(root, c, o)
		if err != nil {
			return err
		}
		file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return err
		}
		_, writeErr := file.Write(b)
		closeErr := file.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
		return printJSON(map[string]any{"review_id": m.Options.ReviewID, "bundle": output, "sha256": fmt.Sprintf("%x", sha256.Sum256(b)), "identity": m.Identity, "formal_artifact_mutation": "none"})
	}
	if bundle == "" {
		return fmt.Errorf("bundle required")
	}
	b, err := os.ReadFile(bundle)
	if err != nil {
		return err
	}
	if action == "current-check" {
		m, err := i18n.CheckRepresentationReviewBundle(root, c, b)
		if err != nil {
			return err
		}
		return printJSON(map[string]any{"current": true, "identity": m.Identity, "sha256": fmt.Sprintf("%x", sha256.Sum256(b))})
	}
	if action != "record" || evidence == "" {
		return fmt.Errorf("record requires exact evidence file")
	}
	e, err := os.ReadFile(evidence)
	if err != nil {
		return err
	}
	r, path, err := i18n.RecordRepresentationReview(root, c, b, e)
	if err != nil {
		return err
	}
	return printJSON(map[string]any{"receipt": path, "identity": r.Identity, "decision": r.Decision.Decision})
}
