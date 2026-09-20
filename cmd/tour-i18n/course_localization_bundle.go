package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func exportCourseLocalizationGenerationBundleCommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("course-metadata localization-bundle", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	output := fs.String("output", "", "Course SEO generation upload ZIP output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *output == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: course-metadata localization-bundle --locale <locale> --output <output.zip>")
	}
	data, manifest, err := i18n.ExportCourseLocalizationGenerationBundle(root, *locale, catalog)
	if err != nil {
		return err
	}
	path, err := writeLocaleSurfaceReviewOutput(*output, data)
	if err != nil {
		return err
	}
	fmt.Printf("Course SEO localization bundle exported: %s (locale=%s pages=%d context_sha256=%s authority=%d)\n", path, *locale, manifest.PageCount, manifest.Context.SHA256, len(manifest.Authority))
	return nil
}
