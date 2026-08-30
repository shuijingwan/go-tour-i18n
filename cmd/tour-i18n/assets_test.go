package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/assets"
)

func TestAssetsExportContainsOnlyAllowlistedFilesAndValidChecksums(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "assets")
	if err := exportAssets(root, []string{"--output", output}); err != nil {
		t.Fatal(err)
	}
	if err := validateAssetsExport(output); err != nil {
		t.Fatal(err)
	}
	for _, logicalPath := range assets.SharedPaths() {
		got, err := os.ReadFile(filepath.Join(output, filepath.FromSlash(logicalPath)))
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join(root, "_content", filepath.FromSlash(logicalPath)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("exported %s differs from source", logicalPath)
		}
	}
	for _, forbidden := range []string{
		"tour/script.js",
		"tour/static/partials/editor.html",
		"tour/static/img/tree.png",
		"images/icons/github.svg",
		"tour/site-metadata.json",
	} {
		if _, err := os.Stat(filepath.Join(output, filepath.FromSlash(forbidden))); !os.IsNotExist(err) {
			t.Errorf("forbidden export path %s exists or returned unexpected error: %v", forbidden, err)
		}
	}
	if got := assetPayloadSize(output); got != 224211 {
		t.Fatalf("asset payload size = %d, want 224211", got)
	}
}

func TestAssetsExportRejectsExistingOutputWithoutChangingIt(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "existing")
	if err := os.Mkdir(output, 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(output, "keep.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	err := exportAssets(root, []string{"--output", output})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("existing output error = %v", err)
	}
	if data, err := os.ReadFile(marker); err != nil || string(data) != "keep" {
		t.Fatalf("existing output changed: %q, %v", data, err)
	}
}

func TestValidateAssetsAcceptsOnlyFormalExport(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "assets")
	if err := exportAssets(root, []string{"--output", output}); err != nil {
		t.Fatal(err)
	}
	if err := validateAssets(root, []string{"--input", output}); err != nil {
		t.Fatalf("formal export rejected: %v", err)
	}
	if err := os.WriteFile(filepath.Join(output, "extra.txt"), []byte("extra"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateAssets(root, []string{"--input", output}); err == nil {
		t.Fatal("export with an unauthorized file was accepted")
	}
}

func TestValidateAssetsRejectsModifiedFormalPath(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "assets")
	if err := exportAssets(root, []string{"--output", output}); err != nil {
		t.Fatal(err)
	}
	assetPath := filepath.Join(output, "tour", "static", "css", "app.css")
	if err := os.WriteFile(assetPath, []byte("self-consistent but not formal\n"), 0644); err != nil {
		t.Fatal(err)
	}
	checksums, err := bundleChecksums(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "SHA256SUMS"), checksums, 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateAssets(root, []string{"--input", output}); err == nil {
		t.Fatal("self-consistent export with modified formal asset was accepted")
	}
}

func TestValidateAssetsRejectsSymlinkInput(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	output := filepath.Join(parent, "assets")
	if err := exportAssets(root, []string{"--output", output}); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(parent, "assets-link")
	if err := os.Symlink(output, link); err != nil {
		t.Fatal(err)
	}
	if err := validateAssets(root, []string{"--input", link}); err == nil {
		t.Fatal("symlink input was accepted")
	}
}
