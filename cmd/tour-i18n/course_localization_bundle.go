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
	path, err := writeNewTransportOutput(*output, data)
	if err != nil {
		return err
	}
	fmt.Printf("Course SEO localization bundle exported: %s (locale=%s pages=%d context_sha256=%s authority=%d)\n", path, *locale, manifest.PageCount, manifest.Context.SHA256, len(manifest.Authority))
	return nil
}

func checkCourseLocalizationGenerationBundleCommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("course-metadata localization-bundle-check", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	bundlePath := fs.String("bundle", "", "Course SEO localization generation ZIP")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *bundlePath == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: course-metadata localization-bundle-check --locale <locale> --bundle <generation.zip>")
	}
	bundle, err := readRegularBundleFile(*bundlePath)
	if err != nil {
		return err
	}
	manifest, err := i18n.VerifyCurrentCourseLocalizationGenerationBundle(root, *locale, catalog, bundle)
	if err != nil {
		return err
	}
	fmt.Printf("Course SEO localization generation bundle: CURRENT (locale=%s pages=%d context_sha256=%s)\n", manifest.Locale, manifest.PageCount, manifest.Context.SHA256)
	return nil
}

func exportCourseMaintenanceGenerationBundleCommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("course-metadata generation-bundle", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	taskKind := fs.String("task", "", "refresh or revise")
	var pageIDs repeatedStrings
	fs.Var(&pageIDs, "page-id", "revised page identity; repeat for revise")
	finding := fs.String("finding", "", "repository finding/evidence path; required for revise")
	output := fs.String("output", "", "Course SEO generation upload ZIP output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *taskKind == "" || *output == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: course-metadata generation-bundle --locale <locale> --task <refresh|revise> [--page-id <page-id> ... --finding <repository-path>] --output <output.zip>")
	}
	data, manifest, err := i18n.ExportCourseMaintenanceGenerationBundle(root, catalog, i18n.CourseMaintenanceGenerationBundleOptions{
		Locale: *locale, TaskKind: *taskKind, PageIDs: pageIDs, FindingPath: *finding,
	})
	if err != nil {
		return err
	}
	path, err := writeNewTransportOutput(*output, data)
	if err != nil {
		return err
	}
	fmt.Printf("Course SEO maintenance generation bundle exported: %s (locale=%s task=%s pages=%d identity=%s)\n", path, manifest.Locale, manifest.TaskKind, len(manifest.SelectedPageIDs), manifest.InputIdentitySHA256)
	return nil
}

func checkCourseMaintenanceGenerationBundleCommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("course-metadata generation-bundle-check", flag.ContinueOnError)
	bundlePath := fs.String("bundle", "", "Course SEO maintenance generation ZIP")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *bundlePath == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: course-metadata generation-bundle-check --bundle <generation.zip>")
	}
	bundle, err := readRegularBundleFile(*bundlePath)
	if err != nil {
		return err
	}
	manifest, err := i18n.VerifyCurrentCourseMaintenanceGenerationBundle(root, catalog, bundle)
	if err != nil {
		return err
	}
	fmt.Printf("Course SEO maintenance generation bundle: CURRENT (locale=%s task=%s pages=%d identity=%s)\n", manifest.Locale, manifest.TaskKind, len(manifest.SelectedPageIDs), manifest.InputIdentitySHA256)
	return nil
}
