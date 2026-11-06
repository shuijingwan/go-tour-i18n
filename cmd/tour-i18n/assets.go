package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/assets"
	"github.com/shuijingwan/go-tour-i18n/internal/releasecontract"
)

func exportAssets(root string, args []string) (err error) {
	fs := flag.NewFlagSet("assets export", flag.ContinueOnError)
	outputFlag := fs.String("output", "", "shared assets origin output directory")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if strings.TrimSpace(*outputFlag) == "" {
		return fmt.Errorf("--output is required")
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected assets export arguments: %s", strings.Join(fs.Args(), " "))
	}

	output, err := filepath.Abs(*outputFlag)
	if err != nil {
		return fmt.Errorf("resolve output: %w", err)
	}
	if _, err := os.Lstat(output); err == nil {
		return fmt.Errorf("output %q already exists", output)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect output %q: %w", output, err)
	}
	parent := filepath.Dir(output)
	if info, err := os.Stat(parent); err != nil {
		return fmt.Errorf("inspect output parent %q: %w", parent, err)
	} else if !info.IsDir() {
		return fmt.Errorf("output parent %q is not a directory", parent)
	}

	staging, err := os.MkdirTemp(parent, "."+filepath.Base(output)+".staging-")
	if err != nil {
		return fmt.Errorf("create assets staging: %w", err)
	}
	completed := false
	defer func() {
		if !completed {
			_ = os.RemoveAll(staging)
		}
	}()

	sharedPaths, err := assets.Paths(root)
	if err != nil {
		return err
	}
	for _, logicalPath := range sharedPaths {
		source := filepath.Join(root, "_content", filepath.FromSlash(logicalPath))
		target := filepath.Join(staging, filepath.FromSlash(logicalPath))
		if _, err := releasecontract.ReadRegular(root, "_content/"+logicalPath); err != nil {
			return err
		}
		if err := copyAssetFile(source, target); err != nil {
			return fmt.Errorf("export %s: %w", logicalPath, err)
		}
	}
	checksums, err := bundleChecksums(staging)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(staging, "SHA256SUMS"), checksums, 0644); err != nil {
		return fmt.Errorf("write SHA256SUMS: %w", err)
	}
	if err := validateAssetsExport(staging, sharedPaths); err != nil {
		return err
	}
	if err := os.Rename(staging, output); err != nil {
		return fmt.Errorf("export assets: %w", err)
	}
	completed = true
	fmt.Printf("shared assets: %s\n", output)
	fmt.Printf("files=%d bytes=%d\n", len(sharedPaths), assetPayloadSize(output))
	return nil
}

func validateAssets(root string, args []string) error {
	fs := flag.NewFlagSet("assets validate", flag.ContinueOnError)
	inputFlag := fs.String("input", "", "shared assets export directory")
	jsonOut := fs.Bool("json", false, "exact current shared-assets dependency proof")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if strings.TrimSpace(*inputFlag) == "" {
		return fmt.Errorf("--input is required")
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected assets validate arguments: %s", strings.Join(fs.Args(), " "))
	}
	input, err := filepath.Abs(*inputFlag)
	if err != nil {
		return fmt.Errorf("resolve input: %w", err)
	}
	info, err := os.Lstat(input)
	if err != nil {
		return fmt.Errorf("inspect input %q: %w", input, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("input must be a real directory, not a symlink: %q", input)
	}
	paths, err := assets.Paths(root)
	if err != nil {
		return err
	}
	if err := validateAssetsExport(input, paths); err != nil {
		return err
	}
	inventory := map[string]string{}
	for _, logicalPath := range paths {
		exported, err := os.ReadFile(filepath.Join(input, filepath.FromSlash(logicalPath)))
		if err != nil {
			return err
		}
		source, err := releasecontract.ReadRegular(root, "_content/"+logicalPath)
		if err != nil {
			return err
		}
		if !bytes.Equal(exported, source) {
			return fmt.Errorf("exported asset differs from repository source: %s", logicalPath)
		}
		inventory[logicalPath] = releasecontract.Digest(exported)
	}
	if *jsonOut {
		return json.NewEncoder(os.Stdout).Encode(struct {
			Result               string            `json:"result"`
			SharedAssetsIdentity string            `json:"shared_assets_identity"`
			SharedAssets         map[string]string `json:"shared_assets"`
		}{"PASS", releasecontract.Identity(inventory), inventory})
	}
	fmt.Printf("shared assets valid: %s\n", input)
	fmt.Printf("files=%d bytes=%d\n", len(paths), assetPayloadSize(input))
	return nil
}

func copyAssetFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("source is not a regular file")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func validateAssetsExport(root string, inventories ...[]string) error {
	want := assets.SharedPaths()
	var got []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root || entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return fmt.Errorf("assets export contains unsupported file %q", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		got = append(got, filepath.ToSlash(rel))
		return nil
	}); err != nil {
		return err
	}
	sort.Strings(got)
	if len(inventories) > 0 {
		want = append([]string{}, inventories[0]...)
	} else {
		set := map[string]bool{}
		for _, p := range want {
			set[p] = true
		}
		for _, p := range got {
			if p == "SHA256SUMS" {
				continue
			}
			if !assets.StaticPath(p) {
				return fmt.Errorf("non-static export path %s", p)
			}
			set[p] = true
		}
		want = nil
		for p := range set {
			want = append(want, p)
		}
	}
	sort.Strings(want)
	wantWithChecksums := append(want, "SHA256SUMS")
	sort.Strings(wantWithChecksums)
	if strings.Join(got, "\n") != strings.Join(wantWithChecksums, "\n") {
		return fmt.Errorf("assets export file set mismatch: got %v, want %v", got, wantWithChecksums)
	}

	wantChecksums, err := bundleChecksums(root)
	if err != nil {
		return err
	}
	actualChecksums, err := os.ReadFile(filepath.Join(root, "SHA256SUMS"))
	if err != nil {
		return err
	}
	if !bytes.Equal(actualChecksums, wantChecksums) {
		return fmt.Errorf("assets SHA256SUMS verification failed")
	}
	for _, line := range strings.Split(strings.TrimSpace(string(actualChecksums)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return fmt.Errorf("malformed assets checksum line %q", line)
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(fields[1])))
		if err != nil {
			return err
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != fields[0] {
			return fmt.Errorf("assets checksum mismatch for %s", fields[1])
		}
	}
	return nil
}

func assetPayloadSize(root string) int64 {
	var size int64
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
		if e == nil && !d.IsDir() && d.Name() != "SHA256SUMS" {
			if i, e := d.Info(); e == nil {
				size += i.Size()
			}
		}
		return e
	})
	return size
}
