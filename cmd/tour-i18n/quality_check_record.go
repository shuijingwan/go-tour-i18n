package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Both modes use the existing results writer and lineage contract. The JSON
// array is command transport only; it carries no indexes or authority claims.
func qualityCheckRecordCommand(root string, catalog *i18n.Catalog, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("quality-check record", flag.ContinueOnError)
	locale := fs.String("locale", "", "target locale")
	reviewerSession := fs.String("reviewer-session", "", "独立 Reviewer session")
	snapshotID := fs.String("snapshot-id", "", "Candidate Snapshot id")
	previousSnapshotID := fs.String("previous-snapshot-id", "", "previous Quality Check Snapshot id for carry-forward")
	fullRereview := fs.Bool("full-rereview", false, "explicitly accept a full repeated review with no predecessor")
	rating := fs.String("rating", "", "Quality Check rating: A, B, C, or D")
	finding := fs.String("finding", "", "per-TranslationUnit finding; required for B/C/D")
	var unitIDs, files repeatedStrings
	fs.Var(&unitIDs, "unit-id", "TranslationUnit id; repeat for multiple units with the same rating")
	fs.Var(&files, "judgments-file", "JSON array of explicit per-Unit judgments from one Reviewer working set")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected quality-check record arguments: %s", strings.Join(fs.Args(), " "))
	}
	var result *i18n.QualityCheckRecordResult
	var err error
	if len(files) > 0 {
		if len(files) != 1 || files[0] == "" {
			return fmt.Errorf("exactly one non-empty --judgments-file is required")
		}
		mixed := false
		fs.Visit(func(f *flag.Flag) {
			if f.Name == "unit-id" || f.Name == "rating" || f.Name == "finding" {
				mixed = true
			}
		})
		if mixed {
			return fmt.Errorf("--judgments-file is mutually exclusive with --unit-id, --rating, and --finding")
		}
		if *locale == "" || *snapshotID == "" || *reviewerSession == "" {
			return fmt.Errorf("--locale, --snapshot-id, and --reviewer-session are required with --judgments-file")
		}
		data, readErr := os.ReadFile(files[0])
		if readErr != nil {
			return readErr
		}
		var judgments []i18n.QualityCheckJudgment
		if err := contentidentity.StrictJSON(data, &judgments); err != nil {
			return fmt.Errorf("quality-check judgments input: %w", err)
		}
		result, err = i18n.RecordQualityCheckJudgments(root, catalog, i18n.QualityCheckRecordJudgmentsOptions{
			ReviewerSession: *reviewerSession, FullRereview: *fullRereview, Locale: *locale, SnapshotID: *snapshotID, PreviousSnapshotID: *previousSnapshotID, Judgments: judgments,
		})
	} else {
		if *locale == "" || *snapshotID == "" || *rating == "" || len(unitIDs) == 0 {
			return fmt.Errorf("--locale, --snapshot-id, at least one --unit-id, and --rating are required")
		}
		result, err = i18n.RecordQualityCheckResults(root, catalog, i18n.QualityCheckRecordOptions{
			ReviewerSession: *reviewerSession, FullRereview: *fullRereview, Locale: *locale, SnapshotID: *snapshotID, PreviousSnapshotID: *previousSnapshotID, UnitIDs: unitIDs, Rating: *rating, Finding: *finding,
		})
	}
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
