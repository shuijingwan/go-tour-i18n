package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

var evidenceReviewIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func scaffoldLocaleSurfaceReviewEvidenceCommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("surface-review evidence-scaffold", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	reviewID := fs.String("review-id", "", "review identity")
	reviewer := fs.String("reviewer", "", "independent reviewer identity")
	date := fs.String("date", "", "review date in YYYY-MM-DD")
	bundlePath := fs.String("bundle", "", "current Surface Review reviewer ZIP")
	output := fs.String("output", "", "evidence Markdown output; defaults to the formal review path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *reviewID == "" || *reviewer == "" || *date == "" || *bundlePath == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: surface-review evidence-scaffold --locale <locale> --review-id <review-id> --reviewer <reviewer> --date <YYYY-MM-DD> --bundle <reviewer.zip> [--output <evidence.md>]")
	}
	if !evidenceReviewIDPattern.MatchString(*reviewID) || strings.TrimSpace(*reviewer) == "" || strings.ContainsAny(*reviewer, "\r\n") {
		return fmt.Errorf("invalid review-id or reviewer")
	}
	parsedDate, err := time.Parse("2006-01-02", *date)
	if err != nil || parsedDate.Format("2006-01-02") != *date {
		return fmt.Errorf("--date must be YYYY-MM-DD")
	}
	bundle, err := readRegularBundleFile(*bundlePath)
	if err != nil {
		return err
	}
	current, currentManifest, err := i18n.ExportLocaleSurfaceReviewReviewerBundle(root, *locale, catalog)
	if err != nil {
		return err
	}
	if !bytes.Equal(bundle, current) {
		return fmt.Errorf("Surface Review reviewer bundle is stale or non-canonical; export it again from the current working tree")
	}
	files, err := i18n.ReadTransportBundle(bundle, 256, 128<<20)
	if err != nil {
		return err
	}
	var embedded i18n.LocaleSurfaceReviewReviewerBundleManifest
	decoder := json.NewDecoder(bytes.NewReader(files["manifest.json"]))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&embedded); err != nil || embedded.Locale != *locale || embedded.ReviewPackage.SHA256 != currentManifest.ReviewPackage.SHA256 {
		return fmt.Errorf("Surface Review reviewer bundle manifest identity mismatch")
	}
	identityBytes, err := os.ReadFile(filepath.Join(root, "production", "identity.json"))
	if err != nil {
		return err
	}
	var identity finalizeIdentity
	if err := json.Unmarshal(identityBytes, &identity); err != nil {
		return fmt.Errorf("malformed production identity")
	}
	profiles := []finalizeProfile{}
	for _, profile := range identity.Locales {
		if profile.Locale == *locale {
			profiles = append(profiles, profile)
		}
	}
	if len(profiles) != 1 || profiles[0].Hostname == "" || profiles[0].PublicURL == "" || profiles[0].State == "" {
		return fmt.Errorf("production identity must contain exactly one complete profile for %s", *locale)
	}
	body := renderLocaleSurfaceReviewEvidenceScaffold(*locale, *reviewID, strings.TrimSpace(*reviewer), *date, hashCommandBytes(bundle), currentManifest, profiles[0])
	destination := *output
	if destination == "" {
		destination = filepath.Join(root, "data", "locale-surface-reviews", *locale, *reviewID+".md")
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	path, err := writeNewTransportOutput(destination, body)
	if err != nil {
		return err
	}
	fmt.Printf("Locale Surface Review evidence scaffolded: %s (locale=%s review_id=%s bundle_sha256=%s; no review decision was inferred)\n", path, *locale, *reviewID, hashCommandBytes(bundle))
	return nil
}

func renderLocaleSurfaceReviewEvidenceScaffold(locale, reviewID, reviewer, date, bundleSHA string, manifest i18n.LocaleSurfaceReviewReviewerBundleManifest, profile finalizeProfile) []byte {
	coverage := manifest.Coverage
	var body strings.Builder
	fmt.Fprintf(&body, "# Locale Surface Review Evidence: %s\n\n", locale)
	fmt.Fprintf(&body, "## Mechanical identity\n\n")
	fmt.Fprintf(&body, "- locale: `%s`\n- review-id: `%s`\n- reviewer: `%s`\n- date: `%s`\n", locale, reviewID, reviewer, date)
	fmt.Fprintf(&body, "- reviewer bundle SHA-256: `%s`\n- review package SHA-256: `%s`\n", bundleSHA, manifest.ReviewPackage.SHA256)
	fmt.Fprintf(&body, "- coverage: pages=%d, ui=%d, articles=%d, translation_units=%d, other_surfaces=%d\n", coverage.Pages, coverage.UI, coverage.Articles, coverage.TranslationUnits, coverage.OtherSurfaces)
	fmt.Fprintf(&body, "- production state at scaffold time: `%s`\n- public identity: `%s` (`%s`)\n", profile.State, profile.PublicURL, profile.Hostname)
	fmt.Fprintf(&body, "\n## Reviewer conclusions\n\n- language quality review result: `REVIEWER_TO_COMPLETE`\n- findings: `REVIEWER_TO_COMPLETE`\n- preview acceptance result: `OPERATOR_TO_COMPLETE`\n\n")
	if profile.State == "first-production" {
		body.WriteString("## First-production finalization\n\nThe production lifecycle conclusion is recorded only in the machine-finalizable block below.\n\n")
		body.WriteString(finalizationPlaceholder)
		body.WriteByte('\n')
	} else {
		body.WriteString("## Final decision\n\n- decision: `REVIEWER_TO_COMPLETE`\n")
	}
	return []byte(body.String())
}

func hashCommandBytes(data []byte) string {
	// The internal bundle manifest already uses SHA-256; this keeps the command
	// package independent of unexported i18n hashing helpers.
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:])
}
