package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/tour"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "tour-i18n:", err)
		os.Exit(1)
	}
}

type repeatedStrings []string

func (values *repeatedStrings) String() string { return strings.Join(*values, ",") }

func (values *repeatedStrings) Set(value string) error {
	*values = append(*values, value)
	return nil
}

func run(args []string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := loadProjectEnv(root); err != nil {
		return err
	}
	if len(args) == 0 {
		return fmt.Errorf("usage: tour-i18n <assets|catalog|upstream|page|locale|status|candidate|translate|retranslation|quality-check|course-metadata|build|preview|publish> <command or flags>")
	}
	if args[0] == "assets" {
		if len(args) < 2 {
			return fmt.Errorf("usage: tour-i18n assets <export --output|validate --input> <directory>")
		}
		switch args[1] {
		case "export":
			return exportAssets(root, args[2:])
		case "validate":
			return validateAssets(root, args[2:])
		default:
			return fmt.Errorf("usage: tour-i18n assets <export --output|validate --input> <directory>")
		}
	}
	var publish *publishOptions
	if args[0] == "publish" {
		options, err := parsePublishOptions(args[1:])
		if err != nil {
			return err
		}
		publish = &options
	}
	current, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		return err
	}
	catalog, err := i18n.ReadCatalog(root)
	if err != nil {
		return err
	}
	// catalog write intentionally reconciles compatible source changes below;
	// every other command requires the committed source lock to match first.
	if args[0] != "catalog" || args[1] != "write" {
		if err := i18n.HydrateCatalogSources(catalog, current); err != nil {
			return err
		}
	}
	if args[0] == "preview" {
		return previewCandidate(root, catalog, args[1:])
	}
	if args[0] == "build" {
		return buildLocale(root, catalog, args[1:])
	}
	if publish != nil {
		return publishBundle(root, catalog, *publish)
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: tour-i18n <assets|catalog|upstream|page|locale|status|candidate|translate|retranslation|quality-check|course-metadata|build|preview|publish> <command or flags>")
	}
	switch args[0] + " " + args[1] {
	case "locale init":
		result, err := initializeLocaleCommand(root, catalog, args[2:])
		if err != nil {
			return err
		}
		fmt.Printf("Locale 骨架初始化：PASS\nlocale: %s\nlocale_dir: %s\nui_catalog: %s\nstatus: %d TranslationUnits（%d Page，%d Example）\n下一步：完成人工 glossary、UI 与 metadata；promotion 后生成正式 course metadata，全部完成后删除 %s。\n",
			result.Locale, result.LocaleDir, result.UICatalog, result.UnitCount, result.PageCount, result.ExampleCount, localeInitIncompleteMarker)
		return nil
	case "course-metadata assemble":
		return assembleCourseMetadata(root, catalog, args[2:])
	case "course-metadata refresh":
		return refreshCourseMetadata(root, catalog, args[2:])
	case "catalog check":
		report, err := i18n.PreviewCatalog(catalog, current)
		if err != nil {
			return err
		}
		if !report.SafeForCatalogWrite() || report.Count(i18n.ContentChanged)+report.Count(i18n.Moved) != 0 {
			return fmt.Errorf("current English source differs from the formal catalog; catalog check does not migrate page IDs: run upstream preview")
		}
		if err := i18n.CheckCatalogFiles(root, catalog); err != nil {
			return err
		}
		fmt.Printf("catalog OK: %d published pages, %d conditional source records, %d play examples\n", len(catalog.Pages), len(catalog.Conditional), len(catalog.Examples))
		return nil
	case "catalog write":
		fs := flag.NewFlagSet("catalog write", flag.ContinueOnError)
		allowSourceChange := fs.Bool("allow-source-change", false, "rebuild catalogs after an explicitly verified manual upstream source update; page routes and conditional identities must be unchanged")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if len(fs.Args()) != 0 {
			return fmt.Errorf("unexpected catalog write arguments: %s", strings.Join(fs.Args(), " "))
		}
		var reconciled *i18n.Catalog
		if *allowSourceChange {
			reconciled, err = i18n.ReconcileCatalogAfterSourceChange(catalog, current)
		} else {
			legacy, legacyErr := i18n.BuildLegacySourceCatalog(root)
			if legacyErr != nil {
				return legacyErr
			}
			if hydrateErr := i18n.HydrateCatalogSources(catalog, current); hydrateErr != nil {
				if legacyErr := i18n.HydrateCatalogSources(catalog, legacy); legacyErr != nil {
					return hydrateErr
				}
			}
			report, previewErr := i18n.PreviewCatalog(catalog, current)
			if previewErr != nil {
				return previewErr
			}
			reconciled, err = i18n.ReconcileCatalog(catalog, current, report)
		}
		if err != nil {
			return err
		}
		pages, conditional, err := i18n.CatalogBytes(reconciled)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(root, "data"), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(root, "data", "tour-pages.tsv"), pages, 0644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(root, "data", "tour-conditional-pages.tsv"), conditional, 0644); err != nil {
			return err
		}
		examples, err := i18n.ExampleCatalogBytes(current)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(root, "data", "tour-examples.tsv"), examples, 0644); err != nil {
			return err
		}
		fmt.Printf("wrote %d published pages, %d conditional source records, and %d play examples; persistent page IDs preserved\n", len(reconciled.Pages), len(reconciled.Conditional), len(current.Examples))
		return nil
	case "upstream preview":
		fs := flag.NewFlagSet("upstream preview", flag.ContinueOnError)
		sourceRoot := fs.String("source-root", "", "prospective website source root")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *sourceRoot == "" {
			return fmt.Errorf("--source-root is required")
		}
		next, err := i18n.BuildSourceCatalog(*sourceRoot)
		if err != nil {
			return err
		}
		report, err := i18n.PreviewCatalog(catalog, next)
		if err != nil {
			return err
		}
		printPreview(root, *sourceRoot, next, report)
		return nil
	case "page export":
		return exportPage(root, catalog, args[2:])
	case "status init":
		locale, result, err := initializeLocaleStatusCommand(root, catalog, args[2:])
		if err != nil {
			return err
		}
		fmt.Printf("status initialized: %d translation units for %s (%d pages, %d examples)\n", result.Total, locale, result.Pages, result.Examples)
		return nil
	case "status check":
		fs := flag.NewFlagSet("status check", flag.ContinueOnError)
		locale := fs.String("locale", "", "locale")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *locale == "" {
			return fmt.Errorf("--locale is required")
		}
		if err := i18n.CheckStatus(root, *locale, catalog); err != nil {
			return err
		}
		total, pages, examples, err := i18n.LocaleWorkflowUnitCounts(catalog)
		if err != nil {
			return err
		}
		fmt.Printf("status OK: %d translation units for %s (%d pages, %d examples)\n", total, *locale, pages, examples)
		return nil
	case "candidate validate":
		fs := flag.NewFlagSet("candidate validate", flag.ContinueOnError)
		locale := fs.String("locale", "", "locale")
		id := fs.String("id", "", "page id")
		file := fs.String("file", "", "candidate file")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *locale == "" || *id == "" || *file == "" {
			return fmt.Errorf("--locale, --id, and --file are required")
		}
		if err := i18n.ValidateLocaleName(*locale); err != nil {
			return err
		}
		data, err := os.ReadFile(*file)
		if err != nil {
			return err
		}
		if err := i18n.ValidateCandidateForLocale(root, catalog, *id, *locale, data); err != nil {
			return err
		}
		fmt.Printf("candidate OK: locale=%s page_id=%s\n", *locale, *id)
		return nil
	case "retranslation export":
		fs := flag.NewFlagSet("retranslation export", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		batchID := fs.String("batch-id", "", "optional explicit batch id")
		unitKind := fs.String("unit-kind", "", "自动选批的翻译单元类型：page（默认）或 example")
		limit := fs.Int("limit", i18n.DefaultRetranslationExportLimit, "自动批次中最多包含的独立翻译单元数（上限 30）")
		jsonOutput := fs.Bool("json", false, "输出完整 machine-readable JSON")
		allowReexport := fs.Bool("allow-reexport", false, "allow explicitly requested page ids to be exported again")
		previousSnapshotID := fs.String("previous-snapshot-id", "", "previous Candidate Snapshot id containing eligible QC or Final Review revision evidence")
		var pageIDs repeatedStrings
		fs.Var(&pageIDs, "id", "optional translation unit id; repeat for multiple units")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *locale == "" {
			return fmt.Errorf("--locale is required")
		}
		if *allowReexport && *previousSnapshotID == "" {
			return fmt.Errorf("--allow-reexport revision mode requires --previous-snapshot-id")
		}
		result, err := i18n.ExportRetranslationBatch(root, catalog, i18n.RetranslationExportOptions{
			Locale: *locale, BatchID: *batchID, UnitIDs: pageIDs, UnitKind: i18n.UnitKind(*unitKind), Limit: *limit, AllowReexport: *allowReexport, PreviousSnapshotID: *previousSnapshotID,
		})
		if err != nil {
			return err
		}
		if result.AllExported {
			if *jsonOutput {
				return printJSON(result)
			}
			completedKind := i18n.UnitKind(*unitKind)
			if completedKind == "" {
				completedKind = i18n.UnitKindPage
			}
			fmt.Printf("没有需要导出的翻译单元：%s 的 %s 已全部完成重译输入导出。\n", *locale, completedKind)
			return nil
		}
		if *jsonOutput {
			return printJSON(result)
		}
		printRetranslationExportSummary(result)
		return nil
	case "retranslation process":
		fs := flag.NewFlagSet("retranslation process", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		batchID := fs.String("batch-id", "", "optional explicit batch id")
		jsonOutput := fs.Bool("json", false, "输出完整 machine-readable JSON")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *locale == "" {
			return fmt.Errorf("--locale is required")
		}
		result, err := i18n.ProcessRetranslationBatch(root, catalog, i18n.RetranslationProcessOptions{Locale: *locale, BatchID: *batchID})
		if err != nil {
			return err
		}
		if result.NoPendingBatches {
			if *jsonOutput {
				return printJSON(result)
			}
			fmt.Printf("没有待处理的重译批次：%s。\n", *locale)
			return nil
		}
		if *jsonOutput {
			return printJSON(result)
		}
		printRetranslationProcessSummary(result)
		return nil
	case "retranslation retry":
		fs := flag.NewFlagSet("retranslation retry", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		batchID := fs.String("batch-id", "", "包含失败翻译单元的批次 ID")
		unitID := fs.String("unit-id", "", "失败翻译单元 ID")
		jsonOutput := fs.Bool("json", false, "输出完整 machine-readable JSON")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *locale == "" || *batchID == "" || *unitID == "" {
			return fmt.Errorf("--locale、--batch-id 和 --unit-id 为必填")
		}
		result, err := i18n.ProcessRetranslationRetry(root, catalog, i18n.RetranslationRetryOptions{Locale: *locale, BatchID: *batchID, UnitID: *unitID})
		if err != nil {
			return err
		}
		return writeRetranslationRetryOutput(os.Stdout, result, *unitID, *jsonOutput)
	case "retranslation revalidate":
		fs := flag.NewFlagSet("retranslation revalidate", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		batchID := fs.String("batch-id", "", "已处理批次 ID")
		unitID := fs.String("unit-id", "", "已有 candidate 的翻译单元 ID")
		jsonOutput := fs.Bool("json", false, "输出 machine-readable JSON")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *locale == "" || *batchID == "" || *unitID == "" {
			return fmt.Errorf("--locale、--batch-id 和 --unit-id 为必填")
		}
		result, err := i18n.RevalidateRetranslationCandidate(root, catalog, i18n.RetranslationRevalidateOptions{Locale: *locale, BatchID: *batchID, UnitID: *unitID})
		if err != nil {
			return err
		}
		return writeRetranslationRevalidationOutput(os.Stdout, result, *jsonOutput)
	case "retranslation review":
		if len(args) < 3 || (args[2] != "check" && args[2] != "scope" && args[2] != "record" && args[2] != "record-batch" && args[2] != "supersede") {
			return fmt.Errorf("usage: tour-i18n retranslation review <check|scope|record|record-batch|supersede> ...")
		}
		if args[2] == "scope" {
			fs := flag.NewFlagSet("retranslation review scope", flag.ContinueOnError)
			locale := fs.String("locale", "", "target locale")
			snapshotID := fs.String("snapshot-id", "", "Candidate Snapshot id")
			jsonOutput := fs.Bool("json", false, "输出包含完整 pending 列表的 machine-readable JSON")
			if err := fs.Parse(args[3:]); err != nil {
				return err
			}
			if *locale == "" || *snapshotID == "" {
				return fmt.Errorf("--locale and --snapshot-id are required")
			}
			if fs.NArg() != 0 {
				return fmt.Errorf("unexpected retranslation review scope arguments: %s", strings.Join(fs.Args(), " "))
			}
			scope, err := i18n.BuildRetranslationReviewScope(root, catalog, i18n.RetranslationReviewScopeOptions{Locale: *locale, SnapshotID: *snapshotID})
			if err != nil {
				return err
			}
			if *jsonOutput {
				return printJSON(scope)
			}
			printRetranslationReviewScopeSummary(scope)
			return nil
		}
		if args[2] == "record-batch" {
			fs := flag.NewFlagSet("retranslation review record-batch", flag.ContinueOnError)
			locale := fs.String("locale", "", "target locale")
			snapshotID := fs.String("snapshot-id", "", "Candidate Snapshot id")
			startIndex := fs.Int("start-index", 1, "first stable Candidate Snapshot index (1-based)")
			limit := fs.Int("limit", i18n.DefaultRetranslationReviewBatchLimit, "maximum consecutive Candidate Snapshot units to record")
			rating := fs.String("rating", "", "quality rating: A, B, C, or D")
			decision := fs.String("decision", "", "workflow decision: approved or rejected")
			summary := fs.String("summary", "", "review summary")
			reviewer := fs.String("reviewer", "", "reviewer identifier")
			rubric := fs.String("rubric", "", "rubric identifier")
			var issues repeatedStrings
			fs.Var(&issues, "issue", "specific issue; repeat for multiple issues")
			if err := fs.Parse(args[3:]); err != nil {
				return err
			}
			if *locale == "" || *snapshotID == "" || *rating == "" || *decision == "" || *summary == "" || *reviewer == "" || *rubric == "" {
				return fmt.Errorf("--locale, --snapshot-id, --rating, --decision, --summary, --reviewer, and --rubric are required")
			}
			if *startIndex < 1 {
				return fmt.Errorf("--start-index must be at least 1")
			}
			if *limit < 1 {
				return fmt.Errorf("--limit must be at least 1")
			}
			if fs.NArg() != 0 {
				return fmt.Errorf("unexpected retranslation review record-batch arguments: %s", strings.Join(fs.Args(), " "))
			}
			result, err := i18n.RecordRetranslationReviewBatch(root, catalog, i18n.RetranslationReviewBatchRecordOptions{
				Locale: *locale, SnapshotID: *snapshotID, StartIndex: *startIndex, Limit: *limit,
				Rating: *rating, Decision: *decision, Summary: *summary, Reviewer: *reviewer, Rubric: *rubric, Issues: issues,
			})
			if err != nil {
				return err
			}
			fmt.Printf("wrote batch review evidence: snapshot=%s indexes=%d-%d units=%d\n", result.SnapshotID, result.StartIndex, result.EndIndex, result.RecordedCount)
			return nil
		}
		if args[2] == "record" {
			fs := flag.NewFlagSet("retranslation review record", flag.ContinueOnError)
			locale := fs.String("locale", "", "target locale (used to locate the batch)")
			batchID := fs.String("batch-id", "", "batch id (used to locate the batch)")
			unitID := fs.String("unit-id", "", "translation unit id")
			rating := fs.String("rating", "", "quality rating: A, B, C, or D")
			decision := fs.String("decision", "", "workflow decision: approved or rejected")
			summary := fs.String("summary", "", "review summary")
			reviewer := fs.String("reviewer", "", "reviewer identifier")
			rubric := fs.String("rubric", "", "rubric identifier")
			var issues repeatedStrings
			fs.Var(&issues, "issue", "specific issue; repeat for multiple issues")
			if err := fs.Parse(args[3:]); err != nil {
				return err
			}
			if *locale == "" || *batchID == "" || *unitID == "" || *rating == "" || *decision == "" || *summary == "" || *reviewer == "" || *rubric == "" {
				return fmt.Errorf("--locale, --batch-id, --unit-id, --rating, --decision, --summary, --reviewer, and --rubric are required")
			}
			if fs.NArg() != 0 {
				return fmt.Errorf("unexpected retranslation review record arguments: %s", strings.Join(fs.Args(), " "))
			}
			review, path, err := i18n.RecordRetranslationReview(root, catalog, i18n.RetranslationReviewRecordOptions{
				Locale: *locale, BatchID: *batchID, UnitID: *unitID, Rating: *rating, Decision: *decision,
				Summary: *summary, Issues: issues, Reviewer: *reviewer, Rubric: *rubric,
			})
			if err != nil {
				return err
			}
			fmt.Printf("wrote review evidence: %s (unit=%s decision=%s rating=%s)\n", path, review.UnitID, review.Decision, review.Rating)
			return nil
		}
		if args[2] == "supersede" {
			fs := flag.NewFlagSet("retranslation review supersede", flag.ContinueOnError)
			locale := fs.String("locale", "", "target locale")
			snapshotID := fs.String("snapshot-id", "", "Candidate Snapshot id")
			unitID := fs.String("unit-id", "", "translation unit id")
			rating := fs.String("rating", "", "must be A")
			decision := fs.String("decision", "", "must be approved")
			summary := fs.String("summary", "", "new Final Review summary")
			reviewer := fs.String("reviewer", "", "reviewer identifier")
			rubric := fs.String("rubric", "", "current rubric identifier")
			var issues repeatedStrings
			fs.Var(&issues, "issue", "specific issue; repeat for multiple issues")
			if err := fs.Parse(args[3:]); err != nil {
				return err
			}
			if *locale == "" || *snapshotID == "" || *unitID == "" || *rating == "" || *decision == "" || *summary == "" || *reviewer == "" || *rubric == "" {
				return fmt.Errorf("--locale, --snapshot-id, --unit-id, --rating, --decision, --summary, --reviewer, and --rubric are required")
			}
			if fs.NArg() != 0 {
				return fmt.Errorf("unexpected retranslation review supersede arguments: %s", strings.Join(fs.Args(), " "))
			}
			result, err := i18n.SupersedeRetranslationReview(root, catalog, i18n.RetranslationReviewSupersedeOptions{
				Locale: *locale, SnapshotID: *snapshotID, UnitID: *unitID, Rating: *rating, Decision: *decision,
				Summary: *summary, Issues: issues, Reviewer: *reviewer, Rubric: *rubric,
			})
			if err != nil {
				return err
			}
			return printJSON(result)
		}
		fs := flag.NewFlagSet("retranslation review check", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		batchID := fs.String("batch-id", "", "batch id")
		if err := fs.Parse(args[3:]); err != nil {
			return err
		}
		if *locale == "" || *batchID == "" {
			return fmt.Errorf("--locale and --batch-id are required")
		}
		if fs.NArg() != 0 {
			return fmt.Errorf("unexpected retranslation review check arguments: %s", strings.Join(fs.Args(), " "))
		}
		report, err := i18n.CheckRetranslationReviews(root, catalog, i18n.RetranslationReviewCheckOptions{Locale: *locale, BatchID: *batchID})
		if err != nil {
			return err
		}
		if report.Rejected == 0 {
			fmt.Printf("review OK: %d units approved\n", report.Approved)
		} else {
			fmt.Printf("review OK: %d units approved, %d units rejected\n", report.Approved, report.Rejected)
		}
		return nil
	case "retranslation promote":
		fs := flag.NewFlagSet("retranslation promote", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		apply := fs.Bool("apply", false, "apply the fully validated promotion plan")
		jsonOutput := fs.Bool("json", false, "输出包含完整 units 列表的 machine-readable JSON")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *locale == "" {
			return fmt.Errorf("--locale is required")
		}
		if fs.NArg() != 0 {
			return fmt.Errorf("unexpected retranslation promote arguments: %s", strings.Join(fs.Args(), " "))
		}
		result, err := i18n.PromoteRetranslation(root, catalog, i18n.RetranslationPromoteOptions{Locale: *locale, Apply: *apply})
		if err != nil {
			return err
		}
		return writeRetranslationPromotionOutput(os.Stdout, result, *apply, *jsonOutput)
	case "quality-check scope":
		fs := flag.NewFlagSet("quality-check scope", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		snapshotID := fs.String("snapshot-id", "", "current Candidate Snapshot id")
		previousSnapshotID := fs.String("previous-snapshot-id", "", "previous Quality Check Snapshot id for carry-forward")
		jsonOutput := fs.Bool("json", false, "输出包含完整 carry-forward/pending 列表的 machine-readable JSON")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *locale == "" || *snapshotID == "" {
			return fmt.Errorf("--locale and --snapshot-id are required")
		}
		if fs.NArg() != 0 {
			return fmt.Errorf("unexpected quality-check scope arguments: %s", strings.Join(fs.Args(), " "))
		}
		scope, err := i18n.BuildQualityCheckScope(root, catalog, i18n.QualityCheckScopeOptions{
			Locale: *locale, SnapshotID: *snapshotID, PreviousSnapshotID: *previousSnapshotID,
		})
		if err != nil {
			return err
		}
		if *jsonOutput {
			return printJSON(scope)
		}
		printQualityCheckScopeSummary(scope)
		return nil
	case "quality-check record":
		fs := flag.NewFlagSet("quality-check record", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		snapshotID := fs.String("snapshot-id", "", "Candidate Snapshot id")
		previousSnapshotID := fs.String("previous-snapshot-id", "", "previous Quality Check Snapshot id for carry-forward")
		rating := fs.String("rating", "", "Quality Check rating: A, B, C, or D")
		finding := fs.String("finding", "", "per-TranslationUnit finding; required for B/C/D")
		var unitIDs repeatedStrings
		fs.Var(&unitIDs, "unit-id", "TranslationUnit id; repeat for multiple units with the same rating")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *locale == "" || *snapshotID == "" || *rating == "" || len(unitIDs) == 0 {
			return fmt.Errorf("--locale, --snapshot-id, at least one --unit-id, and --rating are required")
		}
		if fs.NArg() != 0 {
			return fmt.Errorf("unexpected quality-check record arguments: %s", strings.Join(fs.Args(), " "))
		}
		result, err := i18n.RecordQualityCheckResults(root, catalog, i18n.QualityCheckRecordOptions{
			Locale: *locale, SnapshotID: *snapshotID, PreviousSnapshotID: *previousSnapshotID,
			UnitIDs: unitIDs, Rating: *rating, Finding: *finding,
		})
		if err != nil {
			return err
		}
		return printJSON(result)
	case "quality-check record-batch":
		fs := flag.NewFlagSet("quality-check record-batch", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		snapshotID := fs.String("snapshot-id", "", "Candidate Snapshot id")
		previousSnapshotID := fs.String("previous-snapshot-id", "", "previous Quality Check Snapshot id for carry-forward")
		startIndex := fs.Int("start-index", 1, "first stable Candidate Snapshot index (1-based)")
		limit := fs.Int("limit", i18n.DefaultRetranslationReviewBatchLimit, "maximum TranslationUnits to record")
		rating := fs.String("rating", "", "Quality Check rating: A, B, C, or D")
		finding := fs.String("finding", "", "finding shared by this explicit unit group; required for B/C/D")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *locale == "" || *snapshotID == "" || *rating == "" {
			return fmt.Errorf("--locale, --snapshot-id, and --rating are required")
		}
		if fs.NArg() != 0 {
			return fmt.Errorf("unexpected quality-check record-batch arguments: %s", strings.Join(fs.Args(), " "))
		}
		result, err := i18n.RecordQualityCheckResultBatch(root, catalog, i18n.QualityCheckRecordBatchOptions{
			Locale: *locale, SnapshotID: *snapshotID, PreviousSnapshotID: *previousSnapshotID,
			StartIndex: *startIndex, Limit: *limit, Rating: *rating, Finding: *finding,
		})
		if err != nil {
			return err
		}
		return printJSON(result)
	case "quality-check backfill-finding":
		fs := flag.NewFlagSet("quality-check backfill-finding", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		snapshotID := fs.String("snapshot-id", "", "Candidate Snapshot id")
		unitID := fs.String("unit-id", "", "existing B/C/D TranslationUnit result")
		finding := fs.String("finding", "", "finding to add")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *locale == "" || *snapshotID == "" || *unitID == "" || *finding == "" {
			return fmt.Errorf("--locale, --snapshot-id, --unit-id, and --finding are required")
		}
		if fs.NArg() != 0 {
			return fmt.Errorf("unexpected quality-check backfill-finding arguments: %s", strings.Join(fs.Args(), " "))
		}
		result, err := i18n.BackfillQualityCheckFinding(root, catalog, i18n.QualityCheckFindingBackfillOptions{Locale: *locale, SnapshotID: *snapshotID, UnitID: *unitID, Finding: *finding})
		if err != nil {
			return err
		}
		return printJSON(result)
	case "quality-check snapshot":
		fs := flag.NewFlagSet("quality-check snapshot", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		snapshotID := fs.String("snapshot-id", "", "immutable quality-check candidate snapshot id")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *locale == "" || *snapshotID == "" {
			return fmt.Errorf("--locale and --snapshot-id are required")
		}
		if fs.NArg() != 0 {
			return fmt.Errorf("unexpected quality-check snapshot arguments: %s", strings.Join(fs.Args(), " "))
		}
		manifest, path, err := i18n.CreateQualityCheckCandidateSnapshot(root, catalog, i18n.QualityCheckSnapshotOptions{
			Locale: *locale, SnapshotID: *snapshotID,
		})
		if err != nil {
			return err
		}
		fmt.Printf("wrote quality-check candidate snapshot: %s (%d units: %d pages, %d examples)\n", path, manifest.UnitCount, manifest.PageCount, manifest.ExampleCount)
		return nil
	case "translate run":
		fs := flag.NewFlagSet("translate run", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		id := fs.String("id", "", "persistent page_id")
		dev := fs.Bool("dev", false, "development calibration mode: one attempt per command; never use for production batch translation")
		devAttempts := fs.Int("dev-attempts", 1, "development attempts in one command (1-3; requires --dev)")
		rawInput := fs.Bool("raw-input", false, "experimental: send the hydrated production page without protected-token replacement or response restore")
		minimalProtect := fs.Bool("minimal-protect", false, "experimental: protect only complete .play directive lines")
		devStaticContext := fs.Bool("dev-static-context", false, "experimental dev-only: add protected static code as read-only translation context")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *locale == "" || *id == "" {
			return fmt.Errorf("--locale and --id are required")
		}
		if *rawInput && *minimalProtect {
			return fmt.Errorf("--raw-input and --minimal-protect are mutually exclusive")
		}
		if *devStaticContext && !*dev {
			return fmt.Errorf("--dev-static-context requires --dev")
		}
		if *devStaticContext && *rawInput {
			return fmt.Errorf("--dev-static-context cannot be used with --raw-input")
		}
		if *devStaticContext && *minimalProtect {
			return fmt.Errorf("--dev-static-context cannot be used with --minimal-protect")
		}
		devAttemptsSet := false
		fs.Visit(func(f *flag.Flag) {
			devAttemptsSet = devAttemptsSet || f.Name == "dev-attempts"
		})
		if devAttemptsSet && !*dev {
			return fmt.Errorf("--dev-attempts requires --dev")
		}
		runnerDevAttempts := 0
		if *dev {
			runnerDevAttempts = *devAttempts
		}
		runner := i18n.TranslationRunner{Root: root, Catalog: catalog, Dev: *dev, DevAttempts: runnerDevAttempts, RawInput: *rawInput, MinimalProtect: *minimalProtect, DevStaticContext: *devStaticContext}
		result, err := runner.Run(context.Background(), *id, *locale, os.Getenv("ZHIPU_API_KEY"))
		if err != nil {
			return err
		}
		return printJSON(result)
	case "translate recover-network":
		fs := flag.NewFlagSet("translate recover-network", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		id := fs.String("id", "", "persistent page_id")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *locale == "" || *id == "" {
			return fmt.Errorf("--locale and --id are required")
		}
		result, err := i18n.RecoverNetworkBlockedTranslation(root, catalog, *id, *locale, nil)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "translate revalidate-response":
		fs := flag.NewFlagSet("translate revalidate-response", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		id := fs.String("id", "", "persistent page_id")
		attempt := fs.Int("attempt", 0, "historical attempt number")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *locale == "" || *id == "" || *attempt <= 0 {
			return fmt.Errorf("--locale, --id, and positive --attempt are required")
		}
		result, err := i18n.RevalidateSavedTranslationResponse(root, catalog, *id, *locale, *attempt, nil)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "translate show":
		fs := flag.NewFlagSet("translate show", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		id := fs.String("id", "", "persistent page_id")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *locale == "" || *id == "" {
			return fmt.Errorf("--locale and --id are required")
		}
		status, candidate, err := i18n.LoadTranslationResult(root, *id, *locale)
		if err != nil {
			return err
		}
		return printJSON(struct {
			*i18n.Status
			Candidate string `json:"candidate_content"`
		}{status, candidate})
	default:
		return fmt.Errorf("unknown command %q", args[0]+" "+args[1])
	}
}

func initializeLocaleStatusCommand(root string, catalog *i18n.Catalog, args []string) (string, *i18n.StatusInitializationResult, error) {
	fs := flag.NewFlagSet("status init", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	if err := fs.Parse(args); err != nil {
		return "", nil, err
	}
	if *locale == "" {
		return "", nil, fmt.Errorf("--locale is required")
	}
	if fs.NArg() != 0 {
		return "", nil, fmt.Errorf("unexpected status init arguments: %s", strings.Join(fs.Args(), " "))
	}
	result, err := i18n.InitializeLocaleStatus(root, *locale, catalog)
	if err != nil {
		return "", nil, err
	}
	return *locale, result, nil
}

func previewCandidate(root string, catalog *i18n.Catalog, args []string) error {
	options, err := parsePreviewOptions(args)
	if err != nil {
		return err
	}
	if options.ID != "" {
		tempRoot := filepath.Join(os.TempDir(), "go-tour-i18n-preview", options.Locale, strings.ReplaceAll(options.ID, "/", "-"))
		preview, err := i18n.BuildCandidatePreview(root, catalog, options.ID, options.Locale, tempRoot)
		if err != nil {
			return err
		}
		fmt.Printf("preview URL: http://%s/tour/%s\n", options.HTTPAddr, options.ID)
		fmt.Printf("temporary content: %s\n", preview.ContentDir)
		command := exec.Command("go", "run", "./tour", "-http", options.HTTPAddr, "-openbrowser=false", "-content", preview.ContentDir, "-locale", options.Locale)
		command.Dir = root
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		command.Stdin = os.Stdin
		return command.Run()
	}
	if err := requireLocaleInitializationComplete(root, options.Locale); err != nil {
		return err
	}

	tempRoot, err := os.MkdirTemp("", "go-tour-i18n-preview-"+strings.ReplaceAll(options.Locale, "/", "-")+"-")
	if err != nil {
		return fmt.Errorf("create preview directory: %w", err)
	}
	projection, err := i18n.BuildLocaleProjection(root, catalog, options.Locale, tempRoot)
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		return err
	}
	handler, err := tour.NewPreviewHandler(os.DirFS(projection.ContentDir), projection.Locale)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", options.HTTPAddr)
	if err != nil {
		return err
	}
	defer listener.Close()
	fmt.Printf("local complete preview URL: http://%s/\n", listener.Addr())
	fmt.Printf("projection: locale=%s ready=%d pending=%d blocked=%d pages=%d articles=%d\n",
		projection.Locale, projection.Ready, projection.Pending, projection.Blocked, projection.PageCount, projection.ArticleCount)
	fmt.Printf("temporary content: %s\n", projection.ContentDir)
	return (&http.Server{Handler: handler}).Serve(listener)
}

type previewOptions struct {
	Locale   string
	ID       string
	HTTPAddr string
}

func parsePreviewOptions(args []string) (previewOptions, error) {
	fs := flag.NewFlagSet("preview", flag.ContinueOnError)
	locale := fs.String("locale", "", "candidate locale")
	id := fs.String("id", "", "persistent page_id")
	httpAddr := fs.String("http", "127.0.0.1:3999", "preview host:port")
	if err := fs.Parse(args); err != nil {
		return previewOptions{}, err
	}
	if *locale == "" {
		return previewOptions{}, fmt.Errorf("--locale is required")
	}
	if fs.NArg() != 0 {
		return previewOptions{}, fmt.Errorf("unexpected preview arguments: %s", strings.Join(fs.Args(), " "))
	}
	return previewOptions{Locale: *locale, ID: *id, HTTPAddr: *httpAddr}, nil
}

func buildLocale(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	output := fs.String("output", "", "projection output directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" {
		return fmt.Errorf("--locale is required")
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected build arguments: %s", strings.Join(fs.Args(), " "))
	}
	if err := requireLocaleInitializationComplete(root, *locale); err != nil {
		return err
	}
	outputRoot := *output
	automatic := false
	if outputRoot == "" {
		var err error
		outputRoot, err = os.MkdirTemp("", "go-tour-i18n-build-"+strings.ReplaceAll(*locale, "/", "-")+"-")
		if err != nil {
			return fmt.Errorf("create build directory: %w", err)
		}
		automatic = true
	}
	projection, err := i18n.BuildLocaleProjection(root, catalog, *locale, outputRoot)
	if err != nil {
		if automatic {
			_ = os.RemoveAll(outputRoot)
		}
		return err
	}
	fmt.Printf("locale projection: %s\n", projection.Root)
	fmt.Printf("Tour content: %s\n", projection.ContentDir)
	fmt.Printf("locale=%s ready=%d pending=%d blocked=%d pages=%d articles=%d\n",
		projection.Locale, projection.Ready, projection.Pending, projection.Blocked, projection.PageCount, projection.ArticleCount)
	return nil
}

func loadProjectEnv(root string) error {
	path := filepath.Join(root, ".env")
	values, err := godotenv.Read(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("load project .env: %w", err)
	}
	for key, value := range values {
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("load project .env variable %q: %w", key, err)
		}
	}
	return nil
}

func printJSON(value any) error {
	return writeJSON(os.Stdout, value)
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func printRetranslationExportSummary(result *i18n.RetranslationExportResult) {
	fmt.Printf("重译导出：PASS\n")
	fmt.Printf("batch_id: %s\nlocale: %s\nunit_kind: %s\nunit_count: %d\nbatch_path: %s\n",
		result.BatchID, result.Locale, result.UnitKind, result.UnitCount, result.BatchPath)
}

func printRetranslationProcessSummary(result *i18n.RetranslationProcessResult) {
	overall := "PASS"
	if result.RestoreFailed != 0 || result.ValidationFailed != 0 {
		overall = "FAILED"
	}
	fmt.Printf("重译处理：%s\n", overall)
	fmt.Printf("batch_id: %s\nlocale: %s\nunit_count: %d\nrestore_passed: %d\nrestore_failed: %d\nvalidation_passed: %d\nvalidation_failed: %d\n",
		result.BatchID, result.Locale, result.UnitCount, result.RestorePassed, result.RestoreFailed, result.ValidationPassed, result.ValidationFailed)
	for _, unit := range result.Units {
		if unit.Status == "passed" {
			continue
		}
		fmt.Printf("失败 Unit：unit_id=%s status=%s validation_path=%s", unit.UnitID, unit.Status, unit.ValidationPath)
		if unit.CandidatePath != "" {
			fmt.Printf(" candidate_path=%s", unit.CandidatePath)
		}
		if unit.Error != "" {
			fmt.Printf(" reason=%q", unit.Error)
		}
		fmt.Println()
	}
}

func writeRetranslationRetryOutput(w io.Writer, result *i18n.RetranslationProcessResult, unitID string, jsonOutput bool) error {
	if jsonOutput {
		return writeJSON(w, result)
	}
	var target *i18n.RetranslationUnitResult
	for i := range result.Units {
		if result.Units[i].UnitID == unitID {
			target = &result.Units[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("retry result does not contain translation unit %q", unitID)
	}
	overall := "PASS"
	if target.Status != "passed" {
		overall = "FAILED"
	}
	fmt.Fprintf(w, "重译重试：%s\n", overall)
	fmt.Fprintf(w, "batch_id: %s\nlocale: %s\nunit_id: %s\nattempt: %d\nstatus: %s\n", result.BatchID, result.Locale, target.UnitID, result.RetryAttempt, target.Status)
	fmt.Fprintf(w, "unit_count: %d\nrestore_passed: %d\nrestore_failed: %d\nvalidation_passed: %d\nvalidation_failed: %d\n",
		result.UnitCount, result.RestorePassed, result.RestoreFailed, result.ValidationPassed, result.ValidationFailed)
	if target.Status == "passed" {
		return nil
	}
	fmt.Fprintf(w, "失败 Unit：unit_id=%s status=%s validation_path=%s", target.UnitID, target.Status, target.ValidationPath)
	if target.CandidatePath != "" {
		fmt.Fprintf(w, " candidate_path=%s", target.CandidatePath)
	}
	if target.Error != "" {
		fmt.Fprintf(w, " reason=%q", target.Error)
	}
	fmt.Fprintln(w)
	return nil
}

func writeRetranslationRevalidationOutput(w io.Writer, result *i18n.RetranslationRevalidationResult, jsonOutput bool) error {
	if jsonOutput {
		return writeJSON(w, result)
	}
	overall := "PASS"
	if result.Status != "passed" {
		overall = "FAILED"
	}
	fmt.Fprintf(w, "重译重新验证：%s\n", overall)
	fmt.Fprintf(w, "batch_id: %s\nlocale: %s\nunit_id: %s\nattempt: %d\nrevalidation: %d\nprevious_status: %s\nstatus: %s\n", result.BatchID, result.Locale, result.UnitID, result.Attempt, result.Revalidation, result.PreviousStatus, result.Status)
	fmt.Fprintf(w, "history_path: %s\nvalidation_path: %s\nresult_path: %s\nvalidation_passed: %d\nvalidation_failed: %d\n", result.HistoryPath, result.ValidationPath, result.ResultPath, result.ValidationPassed, result.ValidationFailed)
	if result.Error != "" {
		fmt.Fprintf(w, "reason: %q\n", result.Error)
	}
	return nil
}

func writeRetranslationPromotionOutput(w io.Writer, plan *i18n.RetranslationPromotionPlan, applied, jsonOutput bool) error {
	if jsonOutput {
		return writeJSON(w, plan)
	}
	status := "READY"
	mode := "dry-run"
	if applied {
		status = "APPLIED"
		mode = "apply"
	} else if !plan.CanApply {
		status = "BLOCKED"
	}
	fmt.Fprintf(w, "重译提升：%s\n", status)
	fmt.Fprintf(w, "locale: %s\nmode: %s\nunit_count: %d（%d Page，%d Example）\n", plan.Locale, mode, plan.UnitCount, plan.PageCount, plan.ExampleCount)
	fmt.Fprintf(w, "review_approved_count: %d\nchanged: %d\nunchanged: %d\neof_normalized: %d\ncan_apply: %t\n",
		plan.ReviewApprovedCount, plan.ChangedCount, plan.UnchangedCount, plan.EOFNormalizedCount, plan.CanApply)
	if applied {
		fmt.Fprintln(w, "applied: true")
		return nil
	}
	if plan.CanApply {
		fmt.Fprintln(w, "下一步：添加 --apply 应用此 promotion plan。")
		return nil
	}
	writePromotionFailures := func(label string, values []string) {
		if len(values) == 0 {
			return
		}
		fmt.Fprintf(w, "%s (%d):\n", label, len(values))
		for _, value := range values {
			fmt.Fprintf(w, "- %s\n", value)
		}
	}
	writePromotionFailures("missing_evidence", plan.MissingEvidence)
	writePromotionFailures("missing_review", plan.MissingReview)
	writePromotionFailures("rejected_review", plan.RejectedReview)
	writePromotionFailures("invalid_review", plan.InvalidReview)
	return nil
}

func printQualityCheckScopeSummary(scope *i18n.QualityCheckScope) {
	fmt.Printf("Quality Check scope：locale=%s snapshot=%s total=%d current=%d carry_forward=%d pending=%d A/B/C/D=%d/%d/%d/%d ready_for_final_review=%t\n",
		scope.Locale, scope.SnapshotID, scope.UnitCount, scope.CurrentResultCount, scope.CarryForwardCount,
		scope.PendingCount, scope.ACount, scope.BCount, scope.CCount, scope.DCount, scope.ReadyForFinalReview)
	printQualityCheckPending(scope.Pending)
}

func printQualityCheckPending(units []i18n.QualityCheckScopeUnit) {
	limit := i18n.DefaultRetranslationReviewBatchLimit
	if len(units) < limit {
		limit = len(units)
	}
	if limit > 0 {
		for i := 1; i < limit; i++ {
			if units[i].UnitKind != units[0].UnitKind {
				limit = i
				break
			}
		}
	}
	for _, unit := range units[:limit] {
		fmt.Printf("pending: index=%d unit_id=%s unit_kind=%s batch_id=%s reason=%s action=%s\n",
			unit.Index, unit.UnitID, unit.UnitKind, unit.BatchID, unit.Reason, unit.RequiredAction)
	}
	if len(units) > limit {
		fmt.Printf("其余 pending Unit：%d；使用 --json 获取完整列表。\n", len(units)-limit)
	}
}

func printRetranslationReviewScopeSummary(scope *i18n.RetranslationReviewScope) {
	fmt.Printf("Final Review scope：locale=%s snapshot=%s total=%d reusable=%d pending=%d\n",
		scope.Locale, scope.SnapshotID, scope.UnitCount, scope.ReusableCount, scope.PendingCount)
	limit := i18n.DefaultRetranslationReviewBatchLimit
	if len(scope.Pending) < limit {
		limit = len(scope.Pending)
	}
	if limit > 0 {
		for i := 1; i < limit; i++ {
			if scope.Pending[i].UnitKind != scope.Pending[0].UnitKind {
				limit = i
				break
			}
		}
	}
	for _, unit := range scope.Pending[:limit] {
		fmt.Printf("pending: index=%d unit_id=%s unit_kind=%s batch_id=%s reason=%s action=%s\n",
			unit.Index, unit.UnitID, unit.UnitKind, unit.BatchID, unit.Reason, unit.RequiredAction)
	}
	if len(scope.Pending) > limit {
		fmt.Printf("其余 pending Unit：%d；使用 --json 获取完整列表。\n", len(scope.Pending)-limit)
	}
}

func printPreview(root, sourceRoot string, next *i18n.Catalog, report *i18n.PreviewReport) {
	fmt.Printf("catalog baseline: %s/data/tour-pages.tsv\n", root)
	fmt.Printf("source root: %s\n", sourceRoot)
	fmt.Printf("published pages: %d\n", len(next.Pages))
	fmt.Printf("conditional source records: %d\n", len(next.Conditional))
	fmt.Printf("example source records: %d\n", len(next.Examples))
	fmt.Println("Page:")
	for _, kind := range i18n.ChangeKinds {
		fmt.Printf("%s: %d\n", kind, report.Count(kind))
	}
	fmt.Println("Example:")
	for _, kind := range []i18n.ChangeKind{i18n.Unchanged, i18n.ContentChanged, i18n.Added, i18n.Removed} {
		fmt.Printf("%s: %d\n", kind, report.ExampleCount(kind))
	}
	for _, kind := range i18n.ChangeKinds {
		fmt.Printf("conditional %s: %d\n", kind, report.ConditionalCount(kind))
	}
	for _, change := range append(append([]i18n.PageChange(nil), report.Changes...), report.ConditionalChanges...) {
		if change.Kind == i18n.Unchanged {
			continue
		}
		fmt.Printf("detail: kind=%s page_id=%q old=%s:%d %s %q %s new=%s:%d %s %q %s reason=%q\n",
			change.Kind, change.PageID,
			change.OldArticle, change.OldSectionNumber, change.OldRoute, change.OldSourceTitle, change.OldSourceSHA256,
			change.NewArticle, change.NewSectionNumber, change.NewRoute, change.NewSourceTitle, change.NewSourceSHA256,
			change.Reason)
	}
	for _, change := range report.ExampleChanges {
		if change.Kind == i18n.Unchanged {
			continue
		}
		translation := "upstream drift only; not in translation workflow"
		if change.NewEligibleTranslation {
			translation = "eligible translation example; retranslation may be required"
		}
		fmt.Printf("example detail: kind=%s example_path=%q old_sha256=%s old_eligible=%t new_sha256=%s new_eligible=%t classification_changed=%t reason=%q action=%q\n",
			change.Kind, change.ExamplePath, change.OldSourceSHA256, change.OldEligibleTranslation,
			change.NewSourceSHA256, change.NewEligibleTranslation, change.ClassificationChanged, change.Reason, translation)
	}
	fmt.Printf("safe for catalog write: %t\n", report.SafeForCatalogWrite())
	fmt.Printf("manual mapping required: %t\n", report.NeedsManualMapping())
}

func exportPage(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("page export", flag.ContinueOnError)
	id := fs.String("id", "", "page id")
	output := fs.String("output", "", "output path or -")
	force := fs.Bool("force", false, "overwrite output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *id == "" || *output == "" {
		return fmt.Errorf("--id and --output are required")
	}
	page, err := catalog.Page(*id)
	if err != nil {
		return err
	}
	if *output == "-" {
		_, err = os.Stdout.Write(page.Source)
		return err
	}
	if !*force {
		if _, err := os.Stat(*output); err == nil {
			return fmt.Errorf("output %s already exists (use --force)", *output)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.WriteFile(*output, page.Source, 0644); err != nil {
		return err
	}
	data, err := os.ReadFile(*output)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, page.Source) {
		return fmt.Errorf("export verification failed")
	}
	fmt.Printf("exported %s to %s sha256=%s\n", *id, *output, page.SourceSHA256)
	return nil
}
