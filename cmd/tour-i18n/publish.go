package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/releasecontract"
	"github.com/shuijingwan/go-tour-i18n/internal/site"
	"github.com/shuijingwan/go-tour-i18n/internal/tour"
)

const (
	releaseManifestSchemaVersion = 3
)

type publishOptions struct {
	Locale        string
	Output        string
	PublishedAt   string
	InputBuild    string
	BuildIdentity string
}

type releaseManifest = releasecontract.Manifest

// buildProductionBinary is replaceable only by package tests to exercise
// staging cleanup. Production publishes always use buildProductionBinaryGo.
var buildProductionBinary = buildProductionBinaryGo

// prerenderProductionPages is replaceable only by package tests. Production
// publishes always execute the Chrome implementation.
var prerenderProductionPages = prerenderProductionPagesChrome

// publishOutput lets publish-batch keep --json stdout machine-readable while
// preserving the existing single-publish terminal output.
var publishOutput io.Writer = os.Stdout

func publishLocale(root string, catalog *i18n.Catalog, args []string) error {
	options, err := parsePublishOptions(args)
	if err != nil {
		return err
	}
	return publishBundle(root, catalog, options)
}

func parsePublishOptions(args []string) (publishOptions, error) {
	fs := flag.NewFlagSet("publish", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	output := fs.String("output", "", "production bundle output directory")
	publishedAt := fs.String("published-at", "", "production bundle publication time (RFC 3339 UTC)")
	inputBuild := fs.String("input-build", "", "consume an existing sealed build without rendering it again")
	buildID := fs.String("build-identity", "", "expected sealed build identity (required with --input-build)")
	if err := fs.Parse(args); err != nil {
		return publishOptions{}, err
	}
	if strings.TrimSpace(*locale) == "" {
		return publishOptions{}, fmt.Errorf("--locale is required")
	}
	if strings.TrimSpace(*output) == "" {
		return publishOptions{}, fmt.Errorf("--output is required")
	}
	if strings.TrimSpace(*publishedAt) == "" {
		return publishOptions{}, fmt.Errorf("--published-at is required")
	}
	published, err := time.Parse(time.RFC3339, *publishedAt)
	if err != nil || published.Location() != time.UTC {
		return publishOptions{}, fmt.Errorf("--published-at must be RFC 3339 UTC")
	}
	if fs.NArg() != 0 {
		return publishOptions{}, fmt.Errorf("unexpected publish arguments: %s", strings.Join(fs.Args(), " "))
	}
	if (*inputBuild == "") != (*buildID == "") {
		return publishOptions{}, fmt.Errorf("--input-build and --build-identity must be supplied together")
	}
	if *buildID != "" && !releasecontract.ValidSHA(*buildID) {
		return publishOptions{}, fmt.Errorf("--build-identity must be exact build SHA256SUMS identity")
	}
	return publishOptions{Locale: *locale, Output: *output, PublishedAt: published.Format(time.RFC3339), InputBuild: *inputBuild, BuildIdentity: *buildID}, nil
}

func publishBundle(root string, catalog *i18n.Catalog, options publishOptions) (err error) {
	phase := func(name string) func() {
		started := time.Now()
		return func() {
			fmt.Fprintf(publishOutput, "publish phase %s: %s\n", name, time.Since(started).Round(time.Millisecond))
		}
	}
	if err := requireLocaleInitializationComplete(root, options.Locale); err != nil {
		return err
	}
	output, err := filepath.Abs(options.Output)
	if err != nil {
		return fmt.Errorf("resolve output: %w", err)
	}
	if _, err := os.Lstat(output); err == nil {
		return fmt.Errorf("output %q already exists", output)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect output %q: %w", output, err)
	}
	parent := filepath.Dir(output)
	parentInfo, err := os.Stat(parent)
	if err != nil {
		return fmt.Errorf("inspect output parent %q: %w", parent, err)
	}
	if !parentInfo.IsDir() {
		return fmt.Errorf("output parent %q is not a directory", parent)
	}

	staging, err := os.MkdirTemp(parent, "."+filepath.Base(output)+".staging-")
	if err != nil {
		return fmt.Errorf("create publish staging: %w", err)
	}
	completed := false
	defer func() {
		if !completed {
			_ = os.RemoveAll(staging)
		}
	}()

	projectionDone := phase("projection/build")
	var projection *i18n.LocaleProjection
	var artifact *releasecontract.Artifact
	var inputBuildSHA string
	if options.InputBuild != "" {
		info, checkErr := inspectRelease(options.InputBuild)
		if checkErr != nil {
			return checkErr
		}
		if info.Kind != "build" || info.Locale != options.Locale || info.InputIdentity != options.BuildIdentity {
			return fmt.Errorf("supplied build identity/locale mismatch")
		}
		if err = checkReleaseCurrent(root, info.Content); err != nil {
			return err
		}
		if _, err = os.Stat(filepath.Join(options.InputBuild, "release.json")); err == nil {
			return fmt.Errorf("--input-build must be a build, not an older production bundle")
		}
		b, readErr := os.ReadFile(filepath.Join(options.InputBuild, "build.json"))
		if readErr != nil {
			return readErr
		}
		var desc releaseBuild
		if err = json.Unmarshal(b, &desc); err != nil {
			return err
		}
		if err = os.CopyFS(filepath.Join(staging, "_content"), os.DirFS(filepath.Join(options.InputBuild, "_content"))); err != nil {
			return err
		}
		projection = &i18n.LocaleProjection{Root: staging, ContentDir: filepath.Join(staging, "_content"), Locale: desc.Locale, UnitCount: desc.Units, Ready: desc.Units, PageCount: desc.Pages, ExampleCount: desc.Examples, ArticleCount: desc.Articles}
		artifact = info.Content
		inputBuildSHA = info.InputIdentity
	} else {
		projection, artifact, err = buildReleaseContent(root, catalog, options.Locale, staging)
		if err == nil {
			inputBuildSHA, err = releasecontract.VerifyChecksums(staging)
		}
		if err == nil {
			err = os.Remove(filepath.Join(staging, "build.json"))
		}
		if err == nil {
			err = os.Remove(filepath.Join(staging, "SHA256SUMS"))
		}
	}
	projectionDone()
	if err != nil {
		return err
	}
	if err = checkReleaseCurrent(root, artifact); err != nil {
		return err
	}
	if err := validatePublishProjection(catalog, projection); err != nil {
		return err
	}
	manifest, err := buildReleaseManifest(catalog, projection, options)
	if err != nil {
		return err
	}
	manifest.SiteIdentity = artifact.Identity
	manifest.BuildIdentity = artifact.Identity
	manifest.InputBuildSHA256 = inputBuildSHA
	if err = tour.CheckReleaseCanonicalOrigin(options.Locale, artifact.Origin); err != nil {
		return err
	}
	manifest.SharedAssetsIdentity = artifact.SharedAssetsIdentity
	metadata := tour.SiteMetadata{
		Locale:             options.Locale,
		PublishedAt:        options.PublishedAt,
		UpstreamCommit:     tour.FrozenUpstreamCommit,
		UpstreamCommitTime: tour.FrozenUpstreamCommitTime,
		Pages:              projection.PageCount,
		Articles:           projection.ArticleCount,
	}
	if err := tour.WriteSiteMetadata(projection.ContentDir, metadata); err != nil {
		return err
	}
	buildDone := phase("production binary build")
	binaryPath := filepath.Join(staging, "bin", "tour")
	err = buildProductionBinary(root, options.Locale, binaryPath)
	buildDone()
	if err != nil {
		return err
	}
	prerenderDone := phase("Chrome prerender")
	err = prerenderProductionPages(projection.ContentDir, options.Locale, projection.PageCount)
	prerenderDone()
	if err != nil {
		return fmt.Errorf("prerender course pages: %w", err)
	}
	// This is the exact load-time validation performed by the production
	// binary.  Chrome's rendered-page checks are necessary but must not be a
	// weaker, divergent substitute for startup validation.
	runtimeValidationDone := phase("production runtime validation")
	var tourHandler http.Handler
	tourHandler, err = tour.NewProductionHandler(os.DirFS(projection.ContentDir), options.Locale)
	if err == nil {
		_, err = site.NewReleaseHandler(os.DirFS(projection.ContentDir), options.Locale, tourHandler, false)
	}
	runtimeValidationDone()
	if err != nil {
		return fmt.Errorf("validate production prerender bundle: %w", err)
	}
	manifestDone := phase("manifest/checksum")
	err = writeReleaseManifest(staging, manifest)
	if err == nil {
		checksums, checksumErr := bundleChecksums(staging)
		if checksumErr != nil {
			err = checksumErr
		} else {
			err = os.WriteFile(filepath.Join(staging, "SHA256SUMS"), checksums, 0644)
			if err == nil {
				err = validateBundle(staging, manifest, checksums)
			}
		}
	}
	manifestDone()
	if err != nil {
		return err
	}
	// Revalidate the immutable build and current inputs at the publication
	// boundary. A concurrent source/payload change cannot produce a PASS bundle.
	if err = checkReleaseCurrent(root, artifact); err != nil {
		return err
	}
	if err := os.Rename(staging, output); err != nil {
		return fmt.Errorf("publish bundle: %w", err)
	}
	completed = true
	fmt.Fprintf(publishOutput, "production bundle: %s\n", output)
	fmt.Fprintf(publishOutput, "locale=%s ready=%d pending=%d blocked=%d pages=%d articles=%d\n",
		projection.Locale, projection.Ready, projection.Pending, projection.Blocked, projection.PageCount, projection.ArticleCount)
	return nil
}

func buildReleaseManifest(catalog *i18n.Catalog, projection *i18n.LocaleProjection, options publishOptions) (releaseManifest, error) {
	if projection == nil {
		return releaseManifest{}, fmt.Errorf("locale projection is required")
	}
	total, pages, examples, err := i18n.LocaleWorkflowUnitCounts(catalog)
	if err != nil {
		return releaseManifest{}, fmt.Errorf("determine release workflow: %w", err)
	}
	if total != projection.UnitCount || pages != projection.PageCount || examples != projection.ExampleCount {
		return releaseManifest{}, fmt.Errorf("release workflow counts do not match projection: workflow units=%d pages=%d examples=%d; projection units=%d pages=%d examples=%d",
			total, pages, examples, projection.UnitCount, projection.PageCount, projection.ExampleCount)
	}
	return releaseManifest{
		SchemaVersion:      releaseManifestSchemaVersion,
		Locale:             options.Locale,
		PublishedAt:        options.PublishedAt,
		UpstreamCommit:     tour.FrozenUpstreamCommit,
		UpstreamCommitTime: tour.FrozenUpstreamCommitTime,
		TranslationUnits:   total,
		Pages:              pages,
		EligibleExamples:   examples,
		Articles:           projection.ArticleCount,
		ExecutionTransport: "http-playground-proxy",
		ExecutionProvider:  "play.golang.org",
		LocalSocketEnabled: false,
		GOOS:               runtime.GOOS,
		GOARCH:             runtime.GOARCH,
	}, nil
}

func validatePublishProjection(catalog *i18n.Catalog, projection *i18n.LocaleProjection) error {
	if projection == nil {
		return fmt.Errorf("locale projection is required")
	}
	total, pages, examples, err := i18n.LocaleWorkflowUnitCounts(catalog)
	if err != nil {
		return fmt.Errorf("determine publish workflow: %w", err)
	}
	articles := map[string]bool{}
	for _, p := range catalog.Pages {
		articles[p.Article] = true
	}
	expectedArticles := len(articles)
	if projection.UnitCount != total || projection.Ready != total || projection.Pending != 0 || projection.Blocked != 0 || projection.PageCount != pages || projection.ExampleCount != examples || projection.ArticleCount != expectedArticles {
		return fmt.Errorf("publish requires units=%d ready=%d pending=0 blocked=0 pages=%d examples=%d articles=%d; got units=%d ready=%d pending=%d blocked=%d pages=%d examples=%d articles=%d",
			total, total, pages, examples, expectedArticles,
			projection.UnitCount, projection.Ready, projection.Pending, projection.Blocked, projection.PageCount, projection.ExampleCount, projection.ArticleCount)
	}
	return nil
}

func buildProductionBinaryGo(root, locale, binaryPath string) error {
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0755); err != nil {
		return fmt.Errorf("create binary directory: %w", err)
	}
	command := exec.Command("go", "build", "-mod=readonly", "-trimpath", "-buildvcs=false", "-ldflags=-buildid= -X main.productionLocale="+locale, "-o", binaryPath, "./cmd/tour-production")
	command.Dir = root
	command.Env = productionBuildEnv()
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("build production binary: %w", err)
	}
	return nil
}

// productionBuildEnv makes the production binary independent of the build
// host's libc. Keep the caller's other build settings (including GOOS and
// GOARCH) unchanged, but always disable cgo for this release artifact.
func productionBuildEnv() []string {
	env := os.Environ()
	for i, value := range env {
		if strings.HasPrefix(value, "CGO_ENABLED=") {
			env[i] = "CGO_ENABLED=0"
			return env
		}
	}
	return append(env, "CGO_ENABLED=0")
}

func writeReleaseManifest(root string, manifest releaseManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode release manifest: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(root, "release.json"), data, 0644); err != nil {
		return fmt.Errorf("write release manifest: %w", err)
	}
	return nil
}

func bundleChecksums(root string) ([]byte, error) { return releasecontract.Checksums(root) }

func validateBundle(root string, wantManifest releaseManifest, wantChecksums []byte) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	wantEntries := map[string]bool{"bin": true, "_content": true, "release.json": true, "SHA256SUMS": true}
	if len(entries) != len(wantEntries) {
		return fmt.Errorf("bundle has %d root entries, want %d", len(entries), len(wantEntries))
	}
	for _, entry := range entries {
		if !wantEntries[entry.Name()] {
			return fmt.Errorf("bundle contains unexpected root entry %q", entry.Name())
		}
	}
	info, err := os.Stat(filepath.Join(root, "bin", "tour"))
	if err != nil {
		return fmt.Errorf("production binary: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		return fmt.Errorf("production binary is not executable")
	}
	manifestData, err := os.ReadFile(filepath.Join(root, "release.json"))
	if err != nil {
		return err
	}
	var actualManifest releaseManifest
	if err := json.Unmarshal(manifestData, &actualManifest); err != nil {
		return fmt.Errorf("read release manifest: %w", err)
	}
	if actualManifest.SchemaVersion != releaseManifestSchemaVersion {
		return fmt.Errorf("unsupported release manifest schema version %d", actualManifest.SchemaVersion)
	}
	if actualManifest != wantManifest {
		return fmt.Errorf("release manifest verification failed")
	}
	forbidden := map[string]bool{"locales": true, "status.tsv": true, "candidate": true, "translation-runs": true, "attempt": true}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != root && forbidden[entry.Name()] {
			return fmt.Errorf("bundle contains development artifact %q", path)
		}
		return nil
	}); err != nil {
		return err
	}
	actualChecksums, err := bundleChecksums(root)
	if err != nil {
		return err
	}
	if !bytes.Equal(actualChecksums, wantChecksums) {
		return errors.New("SHA256SUMS verification failed")
	}
	_, _, err = releasecontract.VerifyBundle(root)
	return err
}
