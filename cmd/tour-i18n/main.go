package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "tour-i18n:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := loadProjectEnv(root); err != nil {
		return err
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: tour-i18n <catalog|upstream|page|status|candidate|translate> <command>")
	}
	current, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		return err
	}
	catalog, err := i18n.ReadCatalog(root)
	if err != nil {
		return err
	}
	if err := i18n.HydrateCatalogSources(catalog, current); err != nil {
		return err
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
		fmt.Printf("catalog OK: %d standalone pages, %d conditional pages\n", len(catalog.Pages), len(catalog.Conditional))
		return nil
	case "catalog write":
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
		fmt.Printf("wrote %d standalone pages and %d conditional pages; persistent page IDs preserved\n", len(reconciled.Pages), len(reconciled.Conditional))
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
	case "translate run":
		fs := flag.NewFlagSet("translate run", flag.ContinueOnError)
		locale := fs.String("locale", "", "target locale")
		id := fs.String("id", "", "persistent page_id")
		dev := fs.Bool("dev", false, "development calibration mode: one attempt per command; never use for production batch translation")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *locale == "" || *id == "" {
			return fmt.Errorf("--locale and --id are required")
		}
		runner := i18n.TranslationRunner{Root: root, Catalog: catalog, Dev: *dev}
		result, err := runner.Run(context.Background(), *id, *locale, os.Getenv("ZHIPU_API_KEY"))
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
	fmt.Printf("standalone pages: %d\n", len(next.Pages))
	fmt.Printf("conditional pages: %d\n", len(next.Conditional))
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
