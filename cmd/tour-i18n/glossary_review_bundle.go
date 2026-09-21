package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func exportGlossaryReviewerBundleCommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("glossary-review reviewer-bundle", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	output := fs.String("output", "", "reviewer upload ZIP output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *output == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: glossary-review reviewer-bundle --locale <locale> --output <output.zip>")
	}
	data, manifest, err := i18n.ExportGlossaryReviewerBundle(root, *locale, catalog)
	if err != nil {
		return err
	}
	path, err := writeNewTransportOutput(*output, data)
	if err != nil {
		return err
	}
	fmt.Printf("Glossary Review reviewer bundle exported: %s (locale=%s units=%d pages=%d examples=%d glossary_sha256=%s identity=%s)\n",
		path, manifest.Locale, manifest.UnitCount, manifest.PageCount, manifest.ExampleCount, manifest.GlossarySHA256, manifest.InputIdentitySHA256)
	return nil
}

func checkGlossaryReviewerBundleCommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("glossary-review reviewer-bundle-check", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	bundlePath := fs.String("bundle", "", "reviewer ZIP to verify against the current working tree")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *bundlePath == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: glossary-review reviewer-bundle-check --locale <locale> --bundle <reviewer.zip>")
	}
	bundle, err := readRegularBundleFile(*bundlePath)
	if err != nil {
		return err
	}
	manifest, err := i18n.VerifyCurrentGlossaryReviewerBundle(root, *locale, catalog, bundle)
	if err != nil {
		return err
	}
	fmt.Printf("Glossary Review reviewer bundle: CURRENT (locale=%s glossary_sha256=%s identity=%s)\n", manifest.Locale, manifest.GlossarySHA256, manifest.InputIdentitySHA256)
	return nil
}
