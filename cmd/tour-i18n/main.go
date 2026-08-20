package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
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
		return fmt.Errorf("usage: tour-i18n <catalog|upstream|page|status|candidate|translate|retranslation|build|preview|publish> <command or flags>")
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
		return fmt.Errorf("usage: tour-i18n <catalog|upstream|page|status|candidate|translate|retranslation|build|preview|publish> <command or flags>")
	}
	switch args[0] + " " + args[1] {
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
		legacy, err := i18n.BuildLegacySourceCatalog(root)
		if err != nil {
			return err
		}
		if err := i18n.HydrateCatalogSources(catalog, current); err != nil {
			if legacyErr := i18n.HydrateCatalogSources(catalog, legacy); legacyErr != nil {
				return err
			}
		}
		report, err := i18n.PreviewCatalog(catalog, current)
		if err != nil {
			return err
		}
		reconciled, err := i18n.ReconcileCatalog(catalog, current, report)
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
		fmt.Printf("status OK: %d pages for %s\n", len(catalog.Pages), *locale)
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
		if *locale != "zh-CN" {
			return fmt.Errorf("unsupported locale %q", *locale)
		}
		data, err := os.ReadFile(*file)
		if err != nil {
			return err
		}
		if err := i18n.ValidateCandidate(root, catalog, *id, data); err != nil {
			return err
		}
		fmt.Printf("candidate OK: locale=%s page_id=%s\n", *locale, *id)
		return nil
	case "retranslation export":
		fs := flag.NewFlagSet("retranslation export", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		batchID := fs.String("batch-id", "", "optional explicit batch id")
		unitKind := fs.String("unit-kind", "", "自动选批的翻译单元类型：page（默认）或 example")
		limit := fs.Int("limit", 10, "自动批次中最多包含的独立翻译单元数")
		allowReexport := fs.Bool("allow-reexport", false, "allow explicitly requested page ids to be exported again")
		var pageIDs repeatedStrings
		fs.Var(&pageIDs, "id", "optional translation unit id; repeat for multiple units")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *locale == "" {
			return fmt.Errorf("--locale is required")
		}
		result, err := i18n.ExportRetranslationBatch(root, catalog, i18n.RetranslationExportOptions{
			Locale: *locale, BatchID: *batchID, PageIDs: pageIDs, UnitKind: i18n.UnitKind(*unitKind), Limit: *limit, AllowReexport: *allowReexport,
		})
		if err != nil {
			return err
		}
		if result.AllExported {
			fmt.Printf("没有需要导出的页面：%s 已全部完成重译输入导出。\n", *locale)
			return nil
		}
		return printJSON(result)
	case "retranslation process":
		fs := flag.NewFlagSet("retranslation process", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		batchID := fs.String("batch-id", "", "optional explicit batch id")
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
			fmt.Printf("没有待处理的重译批次：%s。\n", *locale)
			return nil
		}
		return printJSON(result)
	case "retranslation retry":
		fs := flag.NewFlagSet("retranslation retry", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		batchID := fs.String("batch-id", "", "包含失败翻译单元的批次 ID")
		unitID := fs.String("unit-id", "", "失败翻译单元 ID")
		pageID := fs.String("page-id", "", "历史兼容：失败课程页面 ID")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *unitID != "" && *pageID != "" {
			return fmt.Errorf("--unit-id 和 --page-id 不能同时指定")
		}
		if *locale == "" || *batchID == "" || (*unitID == "" && *pageID == "") {
			return fmt.Errorf("--locale、--batch-id，以及 --unit-id/--page-id 二者之一为必填")
		}
		result, err := i18n.ProcessRetranslationRetry(root, catalog, i18n.RetranslationRetryOptions{Locale: *locale, BatchID: *batchID, UnitID: *unitID, PageID: *pageID})
		if err != nil {
			return err
		}
		return printJSON(result)
	case "retranslation promote":
		fs := flag.NewFlagSet("retranslation promote", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		apply := fs.Bool("apply", false, "apply the fully validated promotion plan")
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
		return printJSON(result)
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

func previewCandidate(root string, catalog *i18n.Catalog, args []string) error {
	options, err := parsePreviewOptions(args)
	if err != nil {
		return err
	}
	var preview *i18n.PreviewContent
	if options.ID != "" {
		tempRoot := filepath.Join(os.TempDir(), "go-tour-i18n-preview", options.Locale, strings.ReplaceAll(options.ID, "/", "-"))
		preview, err = i18n.BuildCandidatePreview(root, catalog, options.ID, options.Locale, tempRoot)
		if err != nil {
			return err
		}
		fmt.Printf("preview URL: http://%s/tour/%s\n", options.HTTPAddr, options.ID)
	} else {
		tempRoot, err := os.MkdirTemp("", "go-tour-i18n-preview-"+strings.ReplaceAll(options.Locale, "/", "-")+"-")
		if err != nil {
			return fmt.Errorf("create preview directory: %w", err)
		}
		projection, err := i18n.BuildLocaleProjection(root, catalog, options.Locale, tempRoot)
		if err != nil {
			_ = os.RemoveAll(tempRoot)
			return err
		}
		preview = &i18n.PreviewContent{Root: projection.Root, ContentDir: projection.ContentDir, Locale: projection.Locale}
		fmt.Printf("local complete preview URL: http://%s/\n", options.HTTPAddr)
		fmt.Printf("projection: locale=%s ready=%d pending=%d blocked=%d pages=%d articles=%d\n",
			projection.Locale, projection.Ready, projection.Pending, projection.Blocked, projection.PageCount, projection.ArticleCount)
	}
	fmt.Printf("temporary content: %s\n", preview.ContentDir)
	command := exec.Command("go", "run", "./tour", "-http", options.HTTPAddr, "-openbrowser=false", "-content", preview.ContentDir, "-locale", options.Locale)
	command.Dir = root
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin
	return command.Run()
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
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func printPreview(root, sourceRoot string, next *i18n.Catalog, report *i18n.PreviewReport) {
	fmt.Printf("catalog baseline: %s/data/tour-pages.tsv\n", root)
	fmt.Printf("source root: %s\n", sourceRoot)
	fmt.Printf("published pages: %d\n", len(next.Pages))
	fmt.Printf("conditional source records: %d\n", len(next.Conditional))
	for _, kind := range i18n.ChangeKinds {
		fmt.Printf("%s: %d\n", kind, report.Count(kind))
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
