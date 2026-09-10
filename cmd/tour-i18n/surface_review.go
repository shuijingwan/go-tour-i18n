package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func exportLocaleSurfaceReviewCommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("surface-review export", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	output := fs.String("output", "", "review package JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *output == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: surface-review export --locale <locale> --output <output.json>")
	}
	data, coverage, err := i18n.ExportLocaleSurfaceReviewPackage(root, *locale, catalog)
	if err != nil {
		return err
	}
	path, err := filepath.Abs(*output)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".surface-review-package-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, err = temp.Write(data); err == nil {
		err = temp.Chmod(0644)
	}
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	fmt.Printf("Locale Surface Review package exported: %s (locale=%s pages=%d ui=%d articles=%d translation_units=%d other_surfaces=%d)\n", path, *locale, coverage.Pages, coverage.UI, coverage.Articles, coverage.TranslationUnits, coverage.OtherSurfaces)
	return nil
}

func checkLocaleSurfaceReviewACommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("surface-review check-a", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: surface-review check-a --locale <locale>")
	}
	if err := i18n.RequireCurrentLocaleSurfaceReviewA(root, *locale, catalog); err != nil {
		return err
	}
	fmt.Printf("Locale Surface Review A gate: PASS (locale=%s)\n", *locale)
	return nil
}

func recordLocaleSurfaceReviewACommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("surface-review record-a", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	reviewID := fs.String("review-id", "", "review identity")
	reviewer := fs.String("reviewer", "", "human reviewer")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *reviewID == "" || *reviewer == "" {
		return fmt.Errorf("--locale, --review-id, and --reviewer are required")
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected surface-review record-a arguments: %s", strings.Join(fs.Args(), " "))
	}
	if err := requireFirstProductionPlaceholderForRecordA(root, *locale, *reviewID); err != nil {
		return err
	}
	gate, path, err := i18n.RecordLocaleSurfaceReviewA(root, *locale, *reviewID, *reviewer, catalog)
	if err != nil {
		return err
	}
	fmt.Printf("Locale Surface Review A gate recorded: locale=%s review_id=%s reviewer=%s path=%s\n", gate.Locale, gate.ReviewID, gate.Reviewer, path)
	return nil
}

func requireFirstProductionPlaceholderForRecordA(root, locale, reviewID string) error {
	data, err := os.ReadFile(filepath.Join(root, "production", "identity.json"))
	if err != nil {
		return fmt.Errorf("read production identity: %w", err)
	}
	var identity finalizeIdentity
	if err := json.Unmarshal(data, &identity); err != nil {
		return fmt.Errorf("malformed production identity")
	}
	profiles := []finalizeProfile{}
	for _, profile := range identity.Locales {
		if profile.Locale == locale {
			profiles = append(profiles, profile)
		}
	}
	if len(profiles) != 1 {
		return fmt.Errorf("production identity must contain exactly one locale %s", locale)
	}
	if profiles[0].State != "first-production" {
		return nil
	}
	gatePath, err := i18n.LocaleSurfaceReviewAGatePath(root, locale, reviewID)
	if err != nil {
		return err
	}
	evidence, err := os.ReadFile(strings.TrimSuffix(gatePath, ".a-gate.json") + ".md")
	if err != nil {
		return fmt.Errorf("first-production Surface Review evidence must exist before record-a: %w", err)
	}
	return validateFinalizationPlaceholder(evidence)
}
