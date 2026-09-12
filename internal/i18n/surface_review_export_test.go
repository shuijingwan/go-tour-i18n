package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportLocaleSurfaceReviewPackageDeterministicCurrentTree(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "data", "tour-pages.tsv")); os.IsNotExist(err) {
		root = filepath.Clean(filepath.Join(root, "..", ".."))
	}
	catalog, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	current, err := BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := HydrateCatalogSources(catalog, current); err != nil {
		t.Fatal(err)
	}
	statusPath := filepath.Join(root, "locales", "tr-TR", "status.tsv")
	identityPath := filepath.Join(root, "production", "identity.json")
	beforeStatus, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeIdentity, err := os.ReadFile(identityPath)
	if err != nil {
		t.Fatal(err)
	}
	first, coverage, err := ExportLocaleSurfaceReviewPackage(root, "tr-TR", catalog)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := ExportLocaleSurfaceReviewPackage(root, "tr-TR", catalog)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("identical working-tree inputs produced different packages")
	}
	afterStatus, _ := os.ReadFile(statusPath)
	afterIdentity, _ := os.ReadFile(identityPath)
	if !bytes.Equal(beforeStatus, afterStatus) || !bytes.Equal(beforeIdentity, afterIdentity) {
		t.Fatal("export mutated formal status or production identity")
	}
	var pkg LocaleSurfaceReviewPackage
	if err := json.Unmarshal(first, &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg.SchemaVersion != 1 || pkg.Kind != "go-tour-i18n/locale-surface-review-package" || pkg.Locale != "tr-TR" {
		t.Fatalf("package identity=%+v", pkg)
	}
	if coverage.Pages != len(catalog.Pages) || len(pkg.CoursePages) != len(catalog.Pages) || len(pkg.UI) != coverage.UI || len(pkg.Articles) != coverage.Articles {
		t.Fatalf("coverage=%+v package=%+v", coverage, pkg.Coverage)
	}
	for i, page := range catalog.Pages {
		entry := pkg.CoursePages[i]
		if entry.PageID != page.ID || entry.Route != page.Route || entry.Source == "" || entry.Target == "" || entry.Description == "" {
			t.Fatalf("page %d is incomplete or out of catalog order: %+v", i, entry)
		}
	}
	if pkg.ProductionPublicIdentity.Locale != "tr-TR" || pkg.ProductionPublicIdentity.ProductionHostname == "" || pkg.Glossary.Text == "" {
		t.Fatal("missing public identity or glossary")
	}
}

func TestExportLocaleSurfaceReviewPackageUsesFilesystemUIAndFailsClosed(t *testing.T) {
	root := copySurfaceReviewExportTree(t)
	uiPath := filepath.Join(root, "internal", "tour", "ui", "tr-TR.json")
	uiData, err := os.ReadFile(uiPath)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(uiData), `"Program sonlandı"`, `"Program tamamlandı"`, 1)
	if changed == string(uiData) {
		t.Fatal("fixture UI target was not found")
	}
	if err := os.WriteFile(uiPath, []byte(changed), 0644); err != nil {
		t.Fatal(err)
	}
	catalog := hydratedSurfaceReviewExportCatalog(t, root)
	data, _, err := ExportLocaleSurfaceReviewPackage(root, "tr-TR", catalog)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("Program tamamlandı")) {
		t.Fatal("package did not use filesystem UI target")
	}
	if err := os.WriteFile(uiPath, []byte("{"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ExportLocaleSurfaceReviewPackage(root, "tr-TR", catalog); err == nil {
		t.Fatal("malformed filesystem UI accepted")
	}
	if err := os.WriteFile(uiPath, uiData, 0644); err != nil {
		t.Fatal(err)
	}
	statusPath := filepath.Join(root, "locales", "tr-TR", "status.tsv")
	status, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	nonReady := strings.Replace(string(status), "\tready\t", "\tpending\t", 1)
	if nonReady == string(status) {
		t.Fatal("fixture ready status was not found")
	}
	if err := os.WriteFile(statusPath, []byte(nonReady), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ExportLocaleSurfaceReviewPackage(root, "tr-TR", catalog); err == nil {
		t.Fatal("non-ready TranslationUnit accepted")
	}
	if err := os.WriteFile(statusPath, status, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "locales", "tr-TR", "course-metadata.json"), []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ExportLocaleSurfaceReviewPackage(root, "tr-TR", catalog); err == nil {
		t.Fatal("incomplete course metadata accepted")
	}
}

func TestExportLocaleSurfaceReviewPackageIncludesConcreteOtherSurfaces(t *testing.T) {
	root := surfaceReviewExportRoot(t)
	catalog := hydratedSurfaceReviewExportCatalog(t, root)
	data, _, err := ExportLocaleSurfaceReviewPackage(root, "tr-TR", catalog)
	if err != nil {
		t.Fatal(err)
	}
	var pkg LocaleSurfaceReviewPackage
	if err := json.Unmarshal(data, &pkg); err != nil {
		t.Fatal(err)
	}
	texts := map[string]string{}
	ids := map[string]bool{}
	for _, surface := range pkg.OtherSurfaces {
		if surface.ID == "" || surface.Context == "" || surface.Path == "" || surface.SHA256 == "" || surface.SourceText == "" {
			t.Fatalf("incomplete source context entry: %+v", surface)
		}
		if ids[surface.ID] {
			t.Fatalf("duplicate source context id %q", surface.ID)
		}
		ids[surface.ID] = true
		if surface.SHA256 != sum([]byte(surface.SourceText)) {
			t.Fatalf("source context hash does not match full text for %s", surface.Path)
		}
		texts[surface.Path] = surface.SourceText
	}
	for _, path := range []string{"internal/tour/languages.go", "internal/tour/tour.go", "internal/tour/production.go", "internal/tour/project.go", "internal/tour/seo.go"} {
		if texts[path] == "" {
			t.Fatalf("missing concrete other surface %s", path)
		}
	}
	if !strings.Contains(texts["internal/tour/languages.go"], `TimeLabel:         "yerel saat"`) || !strings.Contains(texts["internal/tour/languages.go"], `Autonym: "Türkçe"`) {
		t.Fatal("locale-visible profile/selector identity missing")
	}
	if !strings.Contains(texts["internal/tour/tour.go"], "execution.exited") {
		t.Fatal("runtime source context missing")
	}
	for _, path := range []string{
		"_content/js/playground.js",
		"_content/tour/static/js/app.js",
		"_content/tour/static/js/controllers.js",
		"_content/tour/static/js/directives.js",
		"_content/tour/static/js/page.js",
		"_content/tour/static/js/services.js",
		"_content/tour/static/js/support.js",
		"_content/tour/static/js/values.js",
		"_content/tour/template/action.tmpl",
		"_content/tour/template/home.tmpl",
		"_content/tour/template/index.tmpl",
		"_content/tour/static/partials/list.html",
	} {
		if texts[path] == "" {
			t.Fatalf("missing first-party runtime/template source %s", path)
		}
	}
	if !strings.Contains(texts["_content/tour/template/home.tmpl"], `{{template "footer" .}}`) ||
		!strings.Contains(texts["_content/tour/template/index.tmpl"], `{{define "footer"}}`) ||
		!strings.Contains(texts["_content/tour/static/partials/list.html"], `ng-repeat="m in toc.modules"`) {
		t.Fatal("homepage, Tour shell/list, or footer composition source is incomplete")
	}
	for path := range texts {
		if strings.HasPrefix(path, "_content/tour/static/lib/") {
			t.Fatalf("vendored static library included in review package: %s", path)
		}
	}
	if pkg.Coverage.OtherSurfaces != len(pkg.OtherSurfaces) || pkg.Coverage.OtherSurfaces != len(texts) {
		t.Fatalf("other surface coverage=%d entries=%d unique_paths=%d", pkg.Coverage.OtherSurfaces, len(pkg.OtherSurfaces), len(texts))
	}
}

func TestExportLocaleSurfaceReviewPackageIncludesFilesystemRuntimeMutation(t *testing.T) {
	root := copySurfaceReviewExportTree(t)
	playgroundPath := filepath.Join(root, "_content", "js", "playground.js")
	playground, err := os.ReadFile(playgroundPath)
	if err != nil {
		t.Fatal(err)
	}
	const sentinel = `Locale Surface Review runtime sentinel visible to users.`
	playground = bytes.Replace(
		playground,
		[]byte(`output.removeClass('error').text('Waiting for remote server...');`),
		[]byte(`output.removeClass('error').text('`+sentinel+`');`),
		1,
	)
	if !bytes.Contains(playground, []byte(sentinel)) {
		t.Fatal("fixture user-visible runtime string was not found")
	}
	if err := os.WriteFile(playgroundPath, playground, 0644); err != nil {
		t.Fatal(err)
	}

	catalog := hydratedSurfaceReviewExportCatalog(t, root)
	data, _, err := ExportLocaleSurfaceReviewPackage(root, "tr-TR", catalog)
	if err != nil {
		t.Fatal(err)
	}
	var pkg LocaleSurfaceReviewPackage
	if err := json.Unmarshal(data, &pkg); err != nil {
		t.Fatal(err)
	}
	for _, surface := range pkg.OtherSurfaces {
		if surface.Path != "_content/js/playground.js" {
			continue
		}
		if !strings.Contains(surface.SourceText, sentinel) {
			t.Fatal("filesystem runtime mutation missing from playground source context")
		}
		if surface.SHA256 != sum(playground) {
			t.Fatalf("playground source hash=%q want %q", surface.SHA256, sum(playground))
		}
		return
	}
	t.Fatal("playground.js missing from runtime source context")
}

func surfaceReviewExportRoot(t *testing.T) string {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "data", "tour-pages.tsv")); os.IsNotExist(err) {
		root = filepath.Clean(filepath.Join(root, "..", ".."))
	}
	return root
}

func hydratedSurfaceReviewExportCatalog(t *testing.T, root string) *Catalog {
	t.Helper()
	catalog, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	current, err := BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := HydrateCatalogSources(catalog, current); err != nil {
		t.Fatal(err)
	}
	return catalog
}

func copySurfaceReviewExportTree(t *testing.T) string {
	t.Helper()
	source := surfaceReviewExportRoot(t)
	destination := t.TempDir()
	for _, name := range []string{"_content", "data", "internal", "locales", "production"} {
		if err := os.CopyFS(filepath.Join(destination, name), os.DirFS(filepath.Join(source, name))); err != nil {
			t.Fatal(err)
		}
	}
	return destination
}
