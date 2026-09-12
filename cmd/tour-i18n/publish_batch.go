package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

type publishBatchOptions struct {
	OutputRoot string
	Locales    []string
	JSON       bool
}

type publishBatchRelease struct {
	Locale      string `json:"locale"`
	PublishedAt string `json:"published_at"`
	ReleaseDir  string `json:"release_dir"`
	Result      string `json:"result"`
}

type publishBatchResult struct {
	RepositoryHead string                `json:"repository_head"`
	Releases       []publishBatchRelease `json:"releases"`
}

var publishBatchClock = time.Now
var publishBatchRepositoryIdentity = inspectPublishBatchRepository
var publishBatchBundle = publishBundle

func parsePublishBatchOptions(args []string) (publishBatchOptions, error) {
	fs := flag.NewFlagSet("publish-batch", flag.ContinueOnError)
	outputRoot := fs.String("output-root", "", "parent directory for generated release bundles")
	jsonOutput := fs.Bool("json", false, "write a machine-readable summary to stdout")
	var locales repeatedStrings
	fs.Var(&locales, "locale", "locale to publish; repeat for multiple locales")
	if err := fs.Parse(args); err != nil {
		return publishBatchOptions{}, err
	}
	if strings.TrimSpace(*outputRoot) == "" {
		return publishBatchOptions{}, fmt.Errorf("--output-root is required")
	}
	if len(locales) == 0 {
		return publishBatchOptions{}, fmt.Errorf("at least one --locale is required")
	}
	if fs.NArg() != 0 {
		return publishBatchOptions{}, fmt.Errorf("unexpected publish-batch arguments: %s", strings.Join(fs.Args(), " "))
	}
	return publishBatchOptions{OutputRoot: *outputRoot, Locales: locales, JSON: *jsonOutput}, nil
}

func inspectPublishBatchRepository(root string) (string, error) {
	status := exec.Command("git", "status", "--porcelain", "--untracked-files=normal")
	status.Dir = root
	dirty, err := status.Output()
	if err != nil {
		return "", fmt.Errorf("inspect repository working tree: %w", err)
	}
	if len(bytes.TrimSpace(dirty)) != 0 {
		return "", fmt.Errorf("repository working tree must be clean before batch publish")
	}
	head := exec.Command("git", "rev-parse", "HEAD")
	head.Dir = root
	value, err := head.Output()
	if err != nil {
		return "", fmt.Errorf("read repository HEAD: %w", err)
	}
	result := strings.TrimSpace(string(value))
	if matched, _ := regexp.MatchString(`^[0-9a-f]{40,64}$`, result); !matched {
		return "", fmt.Errorf("repository HEAD is not a full hexadecimal object ID")
	}
	return result, nil
}

func preflightPublishBatch(root string, catalog *i18n.Catalog, options publishBatchOptions) (string, string, error) {
	outputRoot, err := filepath.Abs(options.OutputRoot)
	if err != nil {
		return "", "", fmt.Errorf("resolve output root: %w", err)
	}
	info, err := os.Lstat(outputRoot)
	if err != nil {
		return "", "", fmt.Errorf("inspect output root: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", "", fmt.Errorf("output root must be a real directory")
	}
	repositoryRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}
	if relative, relErr := filepath.Rel(repositoryRoot, outputRoot); relErr == nil &&
		(relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))) {
		return "", "", fmt.Errorf("output root must be outside the repository")
	}
	seen := make(map[string]bool, len(options.Locales))
	for _, locale := range options.Locales {
		if err := i18n.ValidateLocaleName(locale); err != nil {
			return "", "", err
		}
		if seen[locale] {
			return "", "", fmt.Errorf("duplicate locale %q", locale)
		}
		seen[locale] = true
		if err := requireLocaleInitializationComplete(root, locale); err != nil {
			return "", "", err
		}
		if err := i18n.CheckStatus(root, locale, catalog); err != nil {
			return "", "", fmt.Errorf("locale %s publish readiness: %w", locale, err)
		}
		statuses, err := i18n.ReadStatuses(filepath.Join(root, "locales", locale, "status.tsv"))
		if err != nil {
			return "", "", err
		}
		for _, status := range statuses {
			if status.State != "ready" {
				return "", "", fmt.Errorf("locale %s publish readiness: %s=%s", locale, status.UnitID, status.State)
			}
		}
	}
	head, err := publishBatchRepositoryIdentity(root)
	if err != nil {
		return "", "", err
	}
	return outputRoot, head, nil
}

func executePublishBatch(root string, catalog *i18n.Catalog, options publishBatchOptions, output io.Writer) (*publishBatchResult, error) {
	outputRoot, head, err := preflightPublishBatch(root, catalog, options)
	if err != nil {
		return nil, err
	}
	result := &publishBatchResult{RepositoryHead: head}
	shortHead := head[:12]
	for _, locale := range options.Locales {
		publishedAt := publishBatchClock().UTC().Truncate(time.Second)
		stamp := publishedAt.Format("20060102T150405Z")
		releaseDir := filepath.Join(outputRoot, fmt.Sprintf("go-tour-release-%s-%s-%s", stamp, locale, shortHead))
		if _, err := os.Lstat(releaseDir); err == nil {
			return result, fmt.Errorf("batch publish output %q already exists", releaseDir)
		} else if !os.IsNotExist(err) {
			return result, fmt.Errorf("inspect batch publish output: %w", err)
		}
		item := publishBatchRelease{Locale: locale, PublishedAt: publishedAt.Format(time.RFC3339), ReleaseDir: releaseDir, Result: "PASS"}
		if err := publishBatchBundle(root, catalog, publishOptions{Locale: locale, Output: releaseDir, PublishedAt: item.PublishedAt}); err != nil {
			item.Result = "FAILED"
			result.Releases = append(result.Releases, item)
			return result, fmt.Errorf("publish locale %s: %w", locale, err)
		}
		result.Releases = append(result.Releases, item)
	}
	if options.JSON {
		encoder := json.NewEncoder(output)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			return result, err
		}
	} else {
		fmt.Fprintf(output, "repository_head: %s\n", result.RepositoryHead)
		fmt.Fprintln(output, "locale | published_at | release_dir | result")
		for _, item := range result.Releases {
			fmt.Fprintf(output, "%s | %s | %s | %s\n", item.Locale, item.PublishedAt, item.ReleaseDir, item.Result)
		}
	}
	return result, nil
}

func publishBatchCommand(root string, catalog *i18n.Catalog, args []string) error {
	options, err := parsePublishBatchOptions(args)
	if err != nil {
		return err
	}
	previous := publishOutput
	if options.JSON {
		publishOutput = os.Stderr
	}
	defer func() { publishOutput = previous }()
	_, err = executePublishBatch(root, catalog, options, os.Stdout)
	return err
}
