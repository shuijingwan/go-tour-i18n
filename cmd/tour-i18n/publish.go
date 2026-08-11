package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const (
	frozenUpstreamCommit    = "e11dacba76c5aae474746e9eedee19693f492803"
	expectedPublishPages    = 103
	expectedPublishArticles = 7
)

type publishOptions struct {
	Locale string
	Output string
}

type releaseManifest struct {
	SchemaVersion      int    `json:"schema_version"`
	Locale             string `json:"locale"`
	UpstreamCommit     string `json:"upstream_commit"`
	Pages              int    `json:"pages"`
	Articles           int    `json:"articles"`
	ExecutionTransport string `json:"execution_transport"`
	ExecutionProvider  string `json:"execution_provider"`
	LocalSocketEnabled bool   `json:"local_socket_enabled"`
	GOOS               string `json:"goos"`
	GOARCH             string `json:"goarch"`
}

// buildProductionBinary is replaceable only by package tests to exercise
// staging cleanup. Production publishes always use buildProductionBinaryGo.
var buildProductionBinary = buildProductionBinaryGo

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
	if err := fs.Parse(args); err != nil {
		return publishOptions{}, err
	}
	if strings.TrimSpace(*locale) == "" {
		return publishOptions{}, fmt.Errorf("--locale is required")
	}
	if strings.TrimSpace(*output) == "" {
		return publishOptions{}, fmt.Errorf("--output is required")
	}
	if fs.NArg() != 0 {
		return publishOptions{}, fmt.Errorf("unexpected publish arguments: %s", strings.Join(fs.Args(), " "))
	}
	return publishOptions{Locale: *locale, Output: *output}, nil
}

func publishBundle(root string, catalog *i18n.Catalog, options publishOptions) (err error) {
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

	projection, err := i18n.BuildLocaleProjection(root, catalog, options.Locale, staging)
	if err != nil {
		return err
	}
	if err := validatePublishProjection(projection); err != nil {
		return err
	}
	binaryPath := filepath.Join(staging, "bin", "tour")
	if err := buildProductionBinary(root, options.Locale, binaryPath); err != nil {
		return err
	}
	manifest := releaseManifest{
		SchemaVersion:      1,
		Locale:             options.Locale,
		UpstreamCommit:     frozenUpstreamCommit,
		Pages:              projection.PageCount,
		Articles:           projection.ArticleCount,
		ExecutionTransport: "http-playground-proxy",
		ExecutionProvider:  "play.golang.org",
		LocalSocketEnabled: false,
		GOOS:               runtime.GOOS,
		GOARCH:             runtime.GOARCH,
	}
	if err := writeReleaseManifest(staging, manifest); err != nil {
		return err
	}
	checksums, err := bundleChecksums(staging)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(staging, "SHA256SUMS"), checksums, 0644); err != nil {
		return fmt.Errorf("write SHA256SUMS: %w", err)
	}
	if err := validateBundle(staging, manifest, checksums); err != nil {
		return err
	}
	if err := os.Rename(staging, output); err != nil {
		return fmt.Errorf("publish bundle: %w", err)
	}
	completed = true
	fmt.Printf("production bundle: %s\n", output)
	fmt.Printf("locale=%s ready=%d pending=%d blocked=%d pages=%d articles=%d\n",
		projection.Locale, projection.Ready, projection.Pending, projection.Blocked, projection.PageCount, projection.ArticleCount)
	return nil
}

func validatePublishProjection(projection *i18n.LocaleProjection) error {
	if projection == nil {
		return fmt.Errorf("locale projection is required")
	}
	if projection.Ready != expectedPublishPages || projection.Pending != 0 || projection.Blocked != 0 || projection.PageCount != expectedPublishPages || projection.ArticleCount != expectedPublishArticles {
		return fmt.Errorf("publish requires ready=%d pending=0 blocked=0 pages=%d articles=%d; got ready=%d pending=%d blocked=%d pages=%d articles=%d",
			expectedPublishPages, expectedPublishPages, expectedPublishArticles,
			projection.Ready, projection.Pending, projection.Blocked, projection.PageCount, projection.ArticleCount)
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

func bundleChecksums(root string) ([]byte, error) {
	var paths []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root || entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("bundle contains symlink %q", path)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("bundle contains unsupported file %q", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel != "SHA256SUMS" {
			paths = append(paths, rel)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list bundle files: %w", err)
	}
	sort.Strings(paths)
	var result bytes.Buffer
	for _, rel := range paths {
		file, err := os.Open(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, err
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		fmt.Fprintf(&result, "%s  %s\n", hex.EncodeToString(hash.Sum(nil)), rel)
	}
	return result.Bytes(), nil
}

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
	return nil
}
