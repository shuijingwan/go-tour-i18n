package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "tour-i18n:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: tour-i18n <catalog|page|status|candidate> <command>")
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	catalog, err := i18n.BuildCatalog(root)
	if err != nil {
		return err
	}
	switch args[0] + " " + args[1] {
	case "catalog check":
		if err := i18n.CheckCatalogFiles(root, catalog); err != nil {
			return err
		}
		fmt.Printf("catalog OK: %d standalone pages, %d conditional pages\n", len(catalog.Pages), len(catalog.Conditional))
		return nil
	case "catalog write":
		pages, conditional, err := i18n.CatalogBytes(catalog)
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
		fmt.Printf("wrote %d standalone pages and %d conditional pages\n", len(catalog.Pages), len(catalog.Conditional))
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
	default:
		return fmt.Errorf("unknown command %q", args[0]+" "+args[1])
	}
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
