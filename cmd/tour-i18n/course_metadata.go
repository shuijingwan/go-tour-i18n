package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func assembleCourseMetadata(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("course-metadata assemble", flag.ContinueOnError)
	locale := fs.String("locale", "", "target locale")
	descriptionsPath := fs.String("descriptions", "", "strict page_id and description JSON input")
	provider := fs.String("provider", "", "generation provider provenance")
	model := fs.String("model", "", "generation model provenance")
	generatedAt := fs.String("generated-at", "", "generation time (RFC 3339 UTC)")
	output := fs.String("output", "", "assembled formal metadata output path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *descriptionsPath == "" || *provider == "" || *model == "" || *generatedAt == "" || *output == "" {
		return fmt.Errorf("--locale, --descriptions, --provider, --model, --generated-at, and --output are required")
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected course-metadata assemble arguments: %s", strings.Join(fs.Args(), " "))
	}
	descriptions, err := os.ReadFile(*descriptionsPath)
	if err != nil {
		return fmt.Errorf("read course descriptions: %w", err)
	}
	assembled, err := i18n.AssembleCourseMetadata(root, catalog, i18n.CourseMetadataAssemblyOptions{
		Locale: *locale, Provider: *provider, Model: *model, GeneratedAt: *generatedAt, Descriptions: descriptions,
	})
	if err != nil {
		return err
	}
	if err := writeCourseMetadataAtomic(*output, assembled); err != nil {
		return err
	}
	fmt.Printf("assembled course metadata: %s (locale=%s pages=%d)\n", *output, *locale, len(catalog.Pages))
	return nil
}

func refreshCourseMetadata(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("course-metadata refresh", flag.ContinueOnError)
	locale := fs.String("locale", "", "target locale")
	descriptionsPath := fs.String("descriptions", "", "strict stale page_id and description JSON input")
	provider := fs.String("provider", "", "generation provider provenance for stale Pages")
	model := fs.String("model", "", "generation model provenance for stale Pages")
	generatedAt := fs.String("generated-at", "", "generation time for stale Pages (RFC 3339 UTC)")
	output := fs.String("output", "", "refreshed formal metadata output path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *descriptionsPath == "" || *provider == "" || *model == "" || *generatedAt == "" || *output == "" {
		return fmt.Errorf("--locale, --descriptions, --provider, --model, --generated-at, and --output are required")
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected course-metadata refresh arguments: %s", strings.Join(fs.Args(), " "))
	}
	descriptions, err := os.ReadFile(*descriptionsPath)
	if err != nil {
		return fmt.Errorf("read course refresh descriptions: %w", err)
	}
	refreshed, stale, err := i18n.RefreshCourseMetadata(root, catalog, i18n.CourseMetadataRefreshOptions{
		Locale: *locale, Provider: *provider, Model: *model, GeneratedAt: *generatedAt, Descriptions: descriptions,
	})
	if err != nil {
		return err
	}
	if err := writeCourseMetadataAtomic(*output, refreshed); err != nil {
		return err
	}
	fmt.Printf("refreshed course metadata: %s (locale=%s stale_pages=%d: %s)\n", *output, *locale, len(stale), strings.Join(stale, ", "))
	return nil
}

func writeCourseMetadataAtomic(output string, data []byte) (err error) {
	if strings.TrimSpace(output) == "" {
		return fmt.Errorf("course metadata output path is required")
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return fmt.Errorf("resolve course metadata output: %w", err)
	}
	parent := filepath.Dir(output)
	info, err := os.Stat(parent)
	if err != nil {
		return fmt.Errorf("inspect course metadata output parent: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("course metadata output parent %q is not a directory", parent)
	}
	temporary, err := os.CreateTemp(parent, "."+filepath.Base(output)+".staging-")
	if err != nil {
		return fmt.Errorf("create course metadata staging file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		if err != nil {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err = temporary.Chmod(0644); err != nil {
		return fmt.Errorf("set course metadata staging mode: %w", err)
	}
	if _, err = temporary.Write(data); err != nil {
		return fmt.Errorf("write course metadata staging file: %w", err)
	}
	if err = temporary.Sync(); err != nil {
		return fmt.Errorf("sync course metadata staging file: %w", err)
	}
	if err = temporary.Close(); err != nil {
		return fmt.Errorf("close course metadata staging file: %w", err)
	}
	if err = os.Rename(temporaryPath, output); err != nil {
		return fmt.Errorf("install course metadata output: %w", err)
	}
	return nil
}
