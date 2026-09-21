package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func exportGenerationBundleCommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("generation-bundle export", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	batchID := fs.String("batch-id", "", "retranslation batch id")
	unitID := fs.String("unit-id", "", "retry TranslationUnit id; omit for initial/revision batch")
	output := fs.String("output", "", "generation ZIP output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *batchID == "" || *output == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: generation-bundle export --locale <locale> --batch-id <batch-id> [--unit-id <retry-unit-id>] --output <output.zip>")
	}
	data, manifest, err := i18n.ExportTranslationUnitGenerationBundle(root, catalog, i18n.GenerationBundleOptions{Locale: *locale, BatchID: *batchID, UnitID: *unitID})
	if err != nil {
		return err
	}
	path, err := writeNewTransportOutput(*output, data)
	if err != nil {
		return err
	}
	fmt.Printf("Generation bundle exported: %s (locale=%s batch=%s task=%s kind=%s attempt=%d outputs=%d identity=%s)\n",
		path, manifest.Locale, manifest.BatchID, manifest.TaskKind, manifest.UnitKind, manifest.Attempt, len(manifest.ExpectedOutputs), manifest.InputIdentitySHA256)
	return nil
}

func exportLocaleGenerationBundleCommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("generation-bundle locale-export", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	taskKind := fs.String("task", "", "glossary, locale-assets, or surface-replacement")
	reviewID := fs.String("review-id", "", "Surface Review identity; required only for surface-replacement")
	output := fs.String("output", "", "generation ZIP output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *taskKind == "" || *output == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: generation-bundle locale-export --locale <locale> --task <glossary|locale-assets|surface-replacement> [--review-id <review-id>] --output <output.zip>")
	}
	data, manifest, err := i18n.ExportLocaleGenerationBundle(root, catalog, i18n.LocaleGenerationBundleOptions{Locale: *locale, TaskKind: *taskKind, ReviewID: *reviewID})
	if err != nil {
		return err
	}
	path, err := writeNewTransportOutput(*output, data)
	if err != nil {
		return err
	}
	fmt.Printf("Locale generation bundle exported: %s (locale=%s task=%s review_id=%s identity=%s)\n", path, manifest.Locale, manifest.TaskKind, manifest.ReviewID, manifest.InputIdentitySHA256)
	return nil
}

func checkLocaleGenerationBundleCommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("generation-bundle locale-check", flag.ContinueOnError)
	bundlePath := fs.String("bundle", "", "locale generation ZIP to verify against the current working tree")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *bundlePath == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: generation-bundle locale-check --bundle <generation.zip>")
	}
	bundle, err := readRegularBundleFile(*bundlePath)
	if err != nil {
		return err
	}
	manifest, err := i18n.VerifyCurrentLocaleGenerationBundle(root, catalog, bundle)
	if err != nil {
		return err
	}
	fmt.Printf("Locale generation bundle: CURRENT (locale=%s task=%s review_id=%s identity=%s)\n", manifest.Locale, manifest.TaskKind, manifest.ReviewID, manifest.InputIdentitySHA256)
	return nil
}

func packGenerationResultBundleCommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("generation-bundle result-pack", flag.ContinueOnError)
	bundlePath := fs.String("bundle", "", "original generation ZIP")
	provider := fs.String("provider", "", "actual generation provider: chatgpt or codex")
	model := fs.String("model", i18n.FormalGenerationModel, "actual formal generation model")
	inputDir := fs.String("input-dir", "", "directory containing the exact generated output file set")
	output := fs.String("output", "", "generation result ZIP output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *bundlePath == "" || *provider == "" || *inputDir == "" || *output == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: generation-bundle result-pack --bundle <generation.zip> --provider <chatgpt|codex> [--model gpt-5.6-sol-high] --input-dir <outputs> --output <result.zip>")
	}
	bundle, err := readRegularBundleFile(*bundlePath)
	if err != nil {
		return err
	}
	data, manifest, err := i18n.PackGenerationResultBundle(root, catalog, bundle, *provider, *model, *inputDir)
	if err != nil {
		return err
	}
	path, err := writeNewTransportOutput(*output, data)
	if err != nil {
		return err
	}
	fmt.Printf("Generation result bundle packed: %s (locale=%s batch=%s task=%s attempt=%d provider=%s model=%s outputs=%d)\n",
		path, manifest.Locale, manifest.BatchID, manifest.TaskKind, manifest.Attempt, manifest.Provider, manifest.Model, len(manifest.Outputs))
	return nil
}

func importGenerationResultBundleCommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("generation-bundle import", flag.ContinueOnError)
	bundlePath := fs.String("bundle", "", "original generation ZIP")
	resultPath := fs.String("result", "", "generation result ZIP")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *bundlePath == "" || *resultPath == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: generation-bundle import --bundle <generation.zip> --result <result.zip>")
	}
	bundle, err := readRegularBundleFile(*bundlePath)
	if err != nil {
		return err
	}
	result, err := readRegularBundleFile(*resultPath)
	if err != nil {
		return err
	}
	imported, err := i18n.ImportGenerationResultBundle(root, catalog, bundle, result)
	if err != nil {
		return err
	}
	fmt.Printf("Generation result imported: locale=%s batch=%s task=%s attempt=%d provider=%s model=%s outputs=%d identity=%s\n",
		imported.Locale, imported.BatchID, imported.TaskKind, imported.Attempt, imported.Provider, imported.Model, len(imported.InstalledPaths), imported.InputIdentitySHA256)
	for _, path := range imported.InstalledPaths {
		fmt.Printf("installed: %s\n", path)
	}
	return nil
}

func readRegularBundleFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("bundle must be a regular non-symlink file: %s", path)
	}
	return os.ReadFile(path)
}

func writeNewTransportOutput(output string, data []byte) (string, error) {
	path, err := filepath.Abs(output)
	if err != nil {
		return "", err
	}
	if _, err := os.Lstat(path); err == nil {
		return "", fmt.Errorf("output already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".transport-bundle-*")
	if err != nil {
		return "", err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if _, err = temporary.Write(data); err == nil {
		err = temporary.Chmod(0644)
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", err
	}
	if err := os.Link(name, path); err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("output already exists: %s", path)
		}
		return "", err
	}
	return path, nil
}
