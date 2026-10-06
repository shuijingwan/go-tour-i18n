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
	bundle := fs.String("bundle", "", "current unified Reviewer bundle")
	generationSession := fs.String("generation-session", "", "long-term Generation session")
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
	if *bundle != "" {
		b, err := readRegularBundleFile(*bundle)
		if err != nil {
			return err
		}
		m, err := i18n.CheckUnifiedGlossaryBundle(root, b)
		if err != nil {
			return err
		}
		if m.Locale != *locale {
			return fmt.Errorf("locale mismatch")
		}
		r, p, err := i18n.RecordUnifiedGlossaryReview(root, b, *reviewID, *reviewer, *generationSession, *decision, findings)
		if err != nil {
			return err
		}
		if r.Locale != *locale {
			return fmt.Errorf("locale mismatch")
		}
		fmt.Printf("Unified Glossary Review recorded: locale=%s path=%s\n", r.Locale, p)
		return nil
	}
	return fmt.Errorf("new Glossary Review requires --bundle and --generation-session; old receipts remain historical")
}

// Historical receipt API remains readable; new CLI writes unified evidence.
func recordLegacyGlossaryReview(root, locale, reviewID, reviewer, decision string, findings []string) error {
	receipt, path, err := i18n.RecordGlossaryReview(root, locale, reviewID, reviewer, decision, findings)
	if err != nil {
		return err
	}
	fmt.Printf("Glossary Review recorded: locale=%s review_id=%s decision=%s reviewer=%s path=%s\n", receipt.Locale, receipt.ReviewID, receipt.Decision, receipt.Reviewer, path)
	return nil
}

func checkGlossaryReviewCommand(root string, args []string) error {
	fs := flag.NewFlagSet("glossary-review check", flag.ContinueOnError)
	coverage := fs.String("coverage", "tour", "tour historical/current or unified current")
	locale := fs.String("locale", "", "locale")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: glossary-review check --locale <locale>")
	}
	if *coverage == "unified" {
		_, err := i18n.RequireUnifiedGlossaryReview(root, *locale)
		if err != nil {
			return err
		}
	} else if *coverage != "tour" {
		return fmt.Errorf("unknown glossary coverage")
	} else if err := i18n.RequireCurrentGlossaryReview(root, *locale); err != nil {
		return err
	}
	fmt.Printf("Glossary Review gate: PASS (locale=%s)\n", *locale)
	return nil
}
