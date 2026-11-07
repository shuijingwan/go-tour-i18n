package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// run supplies a freshly hydrated/registered Catalog. Both transport operations
// use the same complete native/registered selector, never a batch Unit subset.
// Keeping the raw shared CLI entry here also permits isolated full dispatch
// contract tests without loading a real locale's authority graph.
func generationBundleLifecycleCommand(root string, catalog *i18n.Catalog, args []string) error {
	contentPackage, remaining, err := parseTranslationContentPackageArgs(args)
	if err != nil {
		return err
	}
	if len(remaining) < 2 || remaining[0] != "generation-bundle" {
		return fmt.Errorf("generation-bundle export/current-check lifecycle command required")
	}
	if contentPackage != "" {
		catalog, err = i18n.SelectCurrentValidationReopenPackage(catalog, contentPackage)
		if err != nil {
			return err
		}
	}
	switch remaining[1] {
	case "export":
		return exportGenerationBundleCommand(root, catalog, remaining[2:], contentPackage)
	case "current-check":
		return checkTranslationUnitGenerationBundleCommand(root, catalog, remaining[2:], contentPackage)
	default:
		return fmt.Errorf("generation-bundle export/current-check lifecycle command required")
	}
}

// Only canonical exporter output or bytes independently verified CURRENT may
// reach this helper. It enforces CLI flag presence, never infers authority from
// an unvalidated manifest or changes the domain API's package proof.
func requireCurrentValidationBundlePackage(data []byte, contentPackage string) error {
	if contentPackage != "" {
		return nil
	}
	files, err := i18n.ReadTransportBundle(data, i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
	if err != nil {
		return err
	}
	var batch i18n.RetranslationBatchManifest
	if err := sitecontent.StrictJSON(files["formal/manifest.json"], &batch); err != nil {
		return err
	}
	for _, unit := range batch.Units {
		if unit.RevisionFeedbackSource == i18n.CurrentValidationRevisionFeedback {
			return fmt.Errorf("current-validation Generation Bundle requires exactly one explicit --content-package")
		}
	}
	return nil
}

func checkTranslationUnitGenerationBundleCommand(root string, catalog *i18n.Catalog, args []string, contentPackage string) error {
	fs := flag.NewFlagSet("generation-bundle current-check", flag.ContinueOnError)
	path := fs.String("bundle", "", "original TranslationUnit generation ZIP")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: generation-bundle current-check --bundle <generation.zip>")
	}
	b, err := readRegularBundleFile(*path)
	if err != nil {
		return err
	}
	m, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, catalog, b)
	if err != nil {
		return err
	}
	if err := requireCurrentValidationBundlePackage(b, contentPackage); err != nil {
		return err
	}
	fmt.Printf("Generation bundle: CURRENT (locale=%s batch=%s task=%s attempt=%d identity=%s)\n", m.Locale, m.BatchID, m.TaskKind, m.Attempt, m.InputIdentitySHA256)
	return nil
}

func exportGenerationBundleCommand(root string, catalog *i18n.Catalog, args []string, contentPackage string) error {
	fs := flag.NewFlagSet("generation-bundle export", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	batchID := fs.String("batch-id", "", "retranslation batch id")
	unitID := fs.String("unit-id", "", "retry TranslationUnit id; omit for initial/revision batch")
	output := fs.String("output", "", "generation ZIP output")
	selection := fs.String("approved-interval-selection", "", "explicit repository-relative approved interval selection JSON (export only)")
	selectionFlags := 0
	for _, arg := range args {
		if arg == "--approved-interval-selection" || strings.HasPrefix(arg, "--approved-interval-selection=") || arg == "-approved-interval-selection" || strings.HasPrefix(arg, "-approved-interval-selection=") {
			selectionFlags++
		}
	}
	if selectionFlags > 1 {
		return fmt.Errorf("approved interval selection flag must appear once")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *batchID == "" || *output == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: generation-bundle export --locale <locale> --batch-id <batch-id> [--unit-id <retry-unit-id>] --output <output.zip>")
	}
	if selectionFlags > 0 && (*selection == "" || contentPackage == "") {
		return fmt.Errorf("approved interval selection requires a path and explicit --content-package")
	}
	data, manifest, err := i18n.ExportTranslationUnitGenerationBundle(root, catalog, i18n.GenerationBundleOptions{Locale: *locale, BatchID: *batchID, UnitID: *unitID, ApprovedIntervalSelection: *selection})
	if err != nil {
		return err
	}
	if err := requireCurrentValidationBundlePackage(data, contentPackage); err != nil {
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
	if *taskKind == "glossary" {
		if err := sitecontent.CheckUnifiedCorpus(root); err != nil {
			return err
		}
	}
	if *taskKind == "locale-assets" {
		if *reviewID != "" {
			return fmt.Errorf("locale-assets cannot carry review-id")
		}
		data, m, err := sitecontent.ExportStructuredAssets(root, *locale)
		if err != nil {
			return err
		}
		p, err := writeNewTransportOutput(*output, data)
		if err != nil {
			return err
		}
		fmt.Printf("Unified structured Generation bundle: %s outputs=%d identity=%s\n", p, len(m.Plan.Expected), m.Identity)
		return nil
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
	files, err := i18n.ReadTransportBundle(bundle, 256, 32<<20)
	if err != nil {
		return err
	}
	if strings.Contains(string(files["manifest.json"]), sitecontent.StructuredSchema) {
		m, err := sitecontent.CheckStructuredAssetsBundle(root, bundle)
		if err != nil {
			return err
		}
		fmt.Printf("Unified structured bundle CURRENT: %s\n", m.Identity)
		return nil
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
	model := fs.String("model", "", "actual formal generation model (defaults by provider)")
	inputDir := fs.String("input-dir", "", "directory containing the exact generated output file set")
	output := fs.String("output", "", "generation result ZIP output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *bundlePath == "" || *provider == "" || *inputDir == "" || *output == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: generation-bundle result-pack --bundle <generation.zip> --provider <chatgpt|codex> [--model <provider-specific-formal-model>] --input-dir <outputs> --output <result.zip>")
	}
	bundle, err := readRegularBundleFile(*bundlePath)
	if err != nil {
		return err
	}
	actualModel := *model
	if actualModel == "" {
		actualModel, err = i18n.DefaultGenerationModelForProvider(*provider)
		if err != nil {
			return err
		}
	}
	data, manifest, err := i18n.PackGenerationResultBundle(root, catalog, bundle, *provider, actualModel, *inputDir)
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

func recoverGenerationBundleCommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("generation-bundle recover", flag.ContinueOnError)
	bundlePath := fs.String("bundle", "", "current generation ZIP")
	outputDir := fs.String("output-dir", "", "new staging directory for recovered outputs")
	jsonOutput := fs.Bool("json", false, "emit stable JSON result")
	var sources repeatedStrings
	fs.Var(&sources, "source", "partial staging file/directory or interrupted tar/tar.gz artifact; repeat as needed")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *bundlePath == "" || *outputDir == "" || len(sources) == 0 || fs.NArg() != 0 {
		return fmt.Errorf("usage: generation-bundle recover --bundle <generation.zip> --source <partial-source> [--source <partial-source>...] --output-dir <new-staging-dir> [--json]")
	}
	bundle, err := readRegularBundleFile(*bundlePath)
	if err != nil {
		return err
	}
	result, err := i18n.RecoverGenerationOutputs(root, catalog, bundle, sources, *outputDir)
	if err != nil {
		return err
	}
	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	status := "PARTIAL"
	if len(result.Missing) == 0 {
		status = "COMPLETE"
	} else if len(result.Recovered) == 0 {
		status = "NONE"
	}
	fmt.Printf("Generation recovery preflight: %s (locale=%s batch=%s task=%s attempt=%d recovered=%d missing=%d issues=%d identity=%s)\n",
		status, result.Locale, result.BatchID, result.TaskKind, result.Attempt, len(result.Recovered), len(result.Missing), len(result.Issues), result.InputIdentitySHA256)
	fmt.Printf("staging: %s\n", result.OutputDir)
	for _, missing := range result.Missing {
		fmt.Printf("missing: %s\n", filepath.Base(filepath.FromSlash(missing)))
	}
	for _, issue := range result.Issues {
		if issue.Output != "" {
			fmt.Printf("issue: %s [%s]: %s\n", issue.Source, issue.Output, issue.Reason)
		} else {
			fmt.Printf("issue: %s: %s\n", issue.Source, issue.Reason)
		}
	}
	return nil
}

func importGenerationResultBundleCommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("generation-bundle import", flag.ContinueOnError)
	executionPath := fs.String("generation-execution", "", "exact new model execution object for recovery")
	executionSHA := fs.String("generation-execution-sha256", "", "exact execution object SHA")
	bundlePath := fs.String("bundle", "", "original generation ZIP")
	resultPath := fs.String("result", "", "generation result ZIP (transport fallback)")
	inputDir := fs.String("input-dir", "", "local directory containing the exact generated output file set")
	provider := fs.String("provider", "", "actual generation provider for --input-dir: chatgpt or codex")
	model := fs.String("model", "", "actual formal generation model for --input-dir")
	hasExecution := false
	for _, arg := range args {
		if arg == "--generation-execution" || arg == "--generation-execution-sha256" || strings.HasPrefix(arg, "--generation-execution=") || strings.HasPrefix(arg, "--generation-execution-sha256=") {
			hasExecution = true
		}
	}
	if hasExecution {
		seen, err := parseUniqueRecoveryFlags(fs, args)
		if err != nil {
			return err
		}
		if !seen["generation-execution"] || !seen["generation-execution-sha256"] {
			return fmt.Errorf("execution path/SHA flags must occur together")
		}
	} else if err := fs.Parse(args); err != nil {
		return err
	}
	if *bundlePath == "" || fs.NArg() != 0 || (*resultPath == "") == (*inputDir == "") {
		return fmt.Errorf("usage: generation-bundle import --bundle <generation.zip> (--result <result.zip> | --input-dir <staging-dir> --provider <chatgpt|codex> --model <provider-specific-formal-model>)")
	}
	bundle, err := readRegularBundleFile(*bundlePath)
	if err != nil {
		return err
	}
	var imported *i18n.GenerationImportResult
	if *inputDir != "" {
		if *provider == "" || *model == "" {
			return fmt.Errorf("--provider and --model are required with --input-dir")
		}
		catalog, err = approvedIntervalImportCatalog(catalog, bundle)
		if err != nil {
			return err
		}
		if hasExecution {
			imported, err = i18n.ImportTransportRecoveryOutputDirectory(root, catalog, bundle, *provider, *model, *inputDir, *executionPath, *executionSHA)
		} else {
			imported, err = i18n.ImportGenerationOutputDirectory(root, catalog, bundle, *provider, *model, *inputDir)
		}
	} else {
		if hasExecution {
			return fmt.Errorf("recovery execution only supports directory import")
		}
		if *provider != "" || *model != "" {
			return fmt.Errorf("--provider and --model are only valid with --input-dir")
		}
		result, readErr := readRegularBundleFile(*resultPath)
		if readErr != nil {
			return readErr
		}
		catalog, err = approvedIntervalImportCatalog(catalog, bundle)
		if err != nil {
			return err
		}
		imported, err = i18n.ImportGenerationResultBundle(root, catalog, bundle, result)
	}
	if err != nil {
		return err
	}
	fmt.Printf("Generation result imported: locale=%s batch=%s task=%s attempt=%d provider=%s model=%s outputs=%d identity=%s generation_bundle_sha256=%s\n",
		imported.Locale, imported.BatchID, imported.TaskKind, imported.Attempt, imported.Provider, imported.Model, len(imported.InstalledPaths), imported.InputIdentitySHA256, imported.GenerationBundleSHA256)
	if imported.RecoveryIdentity != "" {
		fmt.Printf("recovery_identity=%s generation_execution_identity=%s\n", imported.RecoveryIdentity, imported.GenerationExecutionIdentity)
	}
	for _, path := range imported.InstalledPaths {
		fmt.Printf("installed: %s\n", path)
	}
	return nil
}

// Import has no external package selector. A frozen approved request supplies
// its explicit package; this only selects a complete registered scope on a copy.
// The domain importer still freshly reconstructs and verifies the entire graph
// before installing any output. Ordinary imports keep their existing Catalog.
func approvedIntervalImportCatalog(c *i18n.Catalog, bundle []byte) (*i18n.Catalog, error) {
	files, err := i18n.ReadTransportBundle(bundle, i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
	if err != nil {
		return nil, err
	}
	request, present := files["formal/approved-interval-selection.json"]
	if !present {
		return c, nil
	}
	var manifest i18n.GenerationBundleManifest
	if err := sitecontent.StrictJSON(files["manifest.json"], &manifest); err != nil {
		return nil, err
	}
	if err := i18n.ValidateTransportBundleInventory(files, manifest.Files, true); err != nil {
		return nil, err
	}
	var selection i18n.ApprovedIntervalSelection
	if err := sitecontent.StrictJSON(request, &selection); err != nil {
		return nil, err
	}
	if selection.Schema != i18n.ApprovedIntervalSelectionSchema || selection.Locale != manifest.Locale || selection.BatchID != manifest.BatchID {
		return nil, fmt.Errorf("approved interval import selection identity mismatch")
	}
	return i18n.SelectCurrentValidationReopenPackage(c, selection.ContentPackage)
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
