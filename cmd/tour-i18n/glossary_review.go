package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func recordGlossaryReviewCommand(root string, args []string) error {
	fs := flag.NewFlagSet("glossary-review record", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	reviewID := fs.String("review-id", "", "review identity")
	reviewer := fs.String("reviewer", "", "independent reviewer")
	decision := fs.String("decision", "", "passed or failed")
	var findings repeatedStrings
	fs.Var(&findings, "finding", "review finding; repeat for multiple findings")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *reviewID == "" || *reviewer == "" || *decision == "" {
		return fmt.Errorf("--locale, --review-id, --reviewer, and --decision are required")
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected glossary-review record arguments: %s", strings.Join(fs.Args(), " "))
	}
	receipt, path, err := i18n.RecordGlossaryReview(root, *locale, *reviewID, *reviewer, *decision, findings)
	if err != nil {
		return err
	}
	fmt.Printf("Glossary Review recorded: locale=%s review_id=%s decision=%s reviewer=%s path=%s\n", receipt.Locale, receipt.ReviewID, receipt.Decision, receipt.Reviewer, path)
	return nil
}

func checkGlossaryReviewCommand(root string, args []string) error {
	fs := flag.NewFlagSet("glossary-review check", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: glossary-review check --locale <locale>")
	}
	if err := i18n.RequireCurrentGlossaryReview(root, *locale); err != nil {
		return err
	}
	fmt.Printf("Glossary Review gate: PASS (locale=%s)\n", *locale)
	return nil
}
