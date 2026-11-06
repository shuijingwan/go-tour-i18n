package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func qualityCheckSurfaceReopenCommand(root string, catalog *i18n.Catalog, args []string) error {
	options, err := parseQualityCheckSurfaceReopenOptions(args)
	if err != nil {
		return err
	}
	receipt, path, err := i18n.RecordQualityCheckSurfaceReopen(root, catalog, options)
	if err != nil {
		return err
	}
	fmt.Printf("Quality Check surface correction reopened: locale=%s predecessor=%s reopen_id=%s units=%d path=%s\n", receipt.Locale, receipt.PreviousSnapshotID, receipt.ReopenID, receipt.UnitCount, path)
	return nil
}

func parseQualityCheckSurfaceReopenOptions(args []string) (i18n.QualityCheckSurfaceReopenOptions, error) {
	fs := flag.NewFlagSet("quality-check surface-reopen", flag.ContinueOnError)
	var o i18n.QualityCheckSurfaceReopenOptions
	fs.StringVar(&o.Locale, "locale", "", "target locale")
	fs.StringVar(&o.AuthoritySource, "authority-source", "", "replacement authority: quality-check; omitted retains Surface/Preview")
	fs.StringVar(&o.PreviousRating, "previous-rating", "", "exact original QC B/C/D rating")
	var proofPath string
	fs.StringVar(&proofPath, "representation-proof", "", "immutable engineering necessity proof path")
	fs.StringVar(&o.ReopenID, "reopen-id", "", "immutable surface correction authorization id")
	fs.StringVar(&o.PreviousSnapshotID, "previous-snapshot-id", "", "exact finalized predecessor Snapshot identity; Integrated Surface separately requires current authorization")
	fs.StringVar(&o.SurfaceReviewID, "surface-review-id", "", "formal Surface Review finding id")
	fs.BoolVar(&o.IntegratedSurface, "integrated-surface", false, "bind current Integrated Surface receipt and exact candidate artifacts")
	fs.StringVar(&o.PreviousReopenID, "previous-reopen-id", "", "immutable old-ID reopen for an explicit replacement successor")
	var historicalDocuments string
	var previewFailure string
	fs.StringVar(&previewFailure, "preview-failure", "", "immutable machine preview TU failure receipt (exclusive of Surface finding flags)")
	fs.StringVar(&historicalDocuments, "historical-documents", "", "repository path of frozen Page representation/package evidence")
	var findings, unitIDs repeatedStrings
	fs.Var(&findings, "finding", "exact correction finding; repeat for Integrated Surface findings")
	fs.Var(&unitIDs, "unit-id", "exact TranslationUnit scope; repeat as needed")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	o.Findings, o.UnitIDs = findings, unitIDs
	if o.AuthoritySource != "" {
		if o.AuthoritySource != "quality-check" || o.Locale == "" || o.ReopenID == "" || o.PreviousSnapshotID == "" || o.PreviousReopenID != "" || o.SurfaceReviewID != "" || o.IntegratedSurface || previewFailure != "" || historicalDocuments == "" || proofPath == "" || len(findings) == 0 || len(unitIDs) == 0 || len(unitIDs) > i18n.MaxRetranslationExportLimit || (len(findings) != 1 && len(findings) != len(unitIDs)) || (o.PreviousRating != "B" && o.PreviousRating != "C" && o.PreviousRating != "D") || fs.NArg() != 0 {
			return o, fmt.Errorf("QC reconciliation requires exact Snapshot/rating/finding/current Unit, historical documents and representation proof; excludes Surface/Preview authority")
		}
		for _, item := range []struct {
			path string
			ref  *i18n.GlossaryArchiveReference
		}{{proofPath, &o.RepresentationProof}, {historicalDocuments, &o.HistoricalDocuments}} {
			if filepath.IsAbs(item.path) || filepath.Clean(item.path) == "." || strings.HasPrefix(filepath.ToSlash(filepath.Clean(item.path)), "../") {
				return o, fmt.Errorf("reconciliation evidence must be repository-relative")
			}
			data, err := os.ReadFile(item.path)
			if err != nil {
				return o, err
			}
			*item.ref = i18n.GlossaryArchiveReference{Path: item.path, SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}
		}
		if len(findings) == 1 {
			o.Finding, o.Findings = findings[0], nil
		}
		return o, nil
	}
	if proofPath != "" || o.PreviousRating != "" {
		return o, fmt.Errorf("QC proof/rating requires --authority-source quality-check")
	}
	if previewFailure != "" {
		if o.Locale == "" || o.ReopenID == "" || o.PreviousSnapshotID == "" || len(unitIDs) != 1 || o.SurfaceReviewID != "" || len(findings) != 0 || o.IntegratedSurface || historicalDocuments != "" || o.PreviousReopenID != "" || fs.NArg() != 0 {
			return o, fmt.Errorf("--preview-failure requires exact locale/reopen/predecessor/one Unit and excludes Surface/replacement flags")
		}
		if filepath.IsAbs(previewFailure) || filepath.Clean(previewFailure) == "." || strings.HasPrefix(filepath.ToSlash(filepath.Clean(previewFailure)), "../") {
			return o, fmt.Errorf("preview-failure must be repository-relative")
		}
		data, err := os.ReadFile(previewFailure)
		if err != nil {
			return o, err
		}
		o.PreviewFailure = i18n.GlossaryArchiveReference{Path: previewFailure, SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}
		return o, nil
	}
	if (o.PreviousReopenID == "") != (historicalDocuments == "") || (o.PreviousReopenID != "" && !o.IntegratedSurface) {
		return o, fmt.Errorf("replacement successor requires --previous-reopen-id, --historical-documents and --integrated-surface together")
	}
	if historicalDocuments != "" {
		if filepath.IsAbs(historicalDocuments) || filepath.Clean(historicalDocuments) == "." || strings.HasPrefix(filepath.ToSlash(filepath.Clean(historicalDocuments)), "../") {
			return o, fmt.Errorf("historical-documents must be a repository-relative path")
		}
		data, err := os.ReadFile(historicalDocuments)
		if err != nil {
			return o, err
		}
		o.HistoricalDocuments = i18n.GlossaryArchiveReference{Path: historicalDocuments, SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}
	}
	if o.Locale == "" || o.ReopenID == "" || o.PreviousSnapshotID == "" || o.SurfaceReviewID == "" || len(findings) == 0 || len(unitIDs) == 0 || fs.NArg() != 0 {
		return o, fmt.Errorf("usage: quality-check surface-reopen --locale <locale> --reopen-id <id> --previous-snapshot-id <finalized-snapshot> --surface-review-id <id> --finding <text> [--finding <text> ...] [--integrated-surface] --unit-id <id>...")
	}
	return o, nil
}
