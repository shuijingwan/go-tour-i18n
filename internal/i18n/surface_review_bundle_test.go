package i18n

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestExportLocaleSurfaceReviewReviewerBundleDeterministicAndComplete(t *testing.T) {
	root := surfaceReviewExportRoot(t)
	catalog := hydratedSurfaceReviewExportCatalog(t, root)
	first, manifest, err := ExportLocaleSurfaceReviewReviewerBundle(root, "tr-TR", catalog)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := ExportLocaleSurfaceReviewReviewerBundle(root, "tr-TR", catalog)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("identical working-tree inputs produced different reviewer bundles")
	}
	if manifest.SchemaVersion != 1 || manifest.Locale != "tr-TR" || manifest.ReviewPackage.SHA256 == "" {
		t.Fatalf("unexpected manifest identity: %+v", manifest)
	}
	if len(manifest.Authority) != len(localeSurfaceReviewReviewerAuthorityPaths) {
		t.Fatalf("authority count=%d want %d", len(manifest.Authority), len(localeSurfaceReviewReviewerAuthorityPaths))
	}

	reader, err := zip.NewReader(bytes.NewReader(first), int64(len(first)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, file := range reader.File {
		r, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		files[file.Name] = data
	}
	if len(files) != 2+len(localeSurfaceReviewReviewerAuthorityPaths) {
		t.Fatalf("bundle file count=%d", len(files))
	}
	if sum(files["surface-review.json"]) != manifest.ReviewPackage.SHA256 {
		t.Fatal("review package hash mismatch")
	}
	var embeddedManifest LocaleSurfaceReviewReviewerBundleManifest
	if err := json.Unmarshal(files["manifest.json"], &embeddedManifest); err != nil {
		t.Fatal(err)
	}
	if embeddedManifest.ReviewPackage.SHA256 != manifest.ReviewPackage.SHA256 {
		t.Fatal("embedded manifest differs from returned manifest")
	}
	for _, authority := range manifest.Authority {
		if sum(files[authority.BundlePath]) != authority.SHA256 {
			t.Fatalf("authority hash mismatch: %s", authority.RepositoryPath)
		}
		current, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(authority.RepositoryPath)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(files[authority.BundlePath], current) {
			t.Fatalf("authority bytes differ: %s", authority.RepositoryPath)
		}
	}
}
