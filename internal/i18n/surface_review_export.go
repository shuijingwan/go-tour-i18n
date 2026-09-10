package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
)

// LocaleSurfaceReviewPackage is deterministic review input, not review
// evidence or a gate. All content is read from the current working tree.
type LocaleSurfaceReviewPackage struct {
	SchemaVersion            int                                `json:"schema_version"`
	Kind                     string                             `json:"kind"`
	Locale                   string                             `json:"locale"`
	Inputs                   LocaleSurfaceReviewAInputs         `json:"inputs"`
	ProductionPublicIdentity SurfaceReviewPublicIdentity        `json:"production_public_identity"`
	Coverage                 LocaleSurfaceReviewPackageCoverage `json:"coverage"`
	Glossary                 LocaleSurfaceReviewGlossary        `json:"glossary"`
	UI                       []LocaleSurfaceReviewUIMessage     `json:"ui"`
	Articles                 []LocaleSurfaceReviewArticle       `json:"articles"`
	CoursePages              []LocaleSurfaceReviewCoursePage    `json:"course_pages"`
	OtherSurfaces            []LocaleSurfaceReviewOtherSurface  `json:"other_surfaces"`
}

type SurfaceReviewPublicIdentity struct {
	Locale              string `json:"locale"`
	ProductionHostname  string `json:"production_hostname"`
	ProductionPublicURL string `json:"production_public_url"`
}
type LocaleSurfaceReviewPackageCoverage struct {
	Pages            int `json:"pages"`
	UI               int `json:"ui"`
	Articles         int `json:"articles"`
	TranslationUnits int `json:"translation_units"`
	OtherSurfaces    int `json:"other_surfaces"`
}
type LocaleSurfaceReviewGlossary struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Text   string `json:"text"`
}
type LocaleSurfaceReviewUIMessage struct {
	Key    string `json:"key"`
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Target string `json:"target"`
}
type LocaleSurfaceReviewArticle struct {
	Article        string `json:"article"`
	SourcePath     string `json:"source_path"`
	SourceTitle    string `json:"source_title"`
	SourceSubtitle string `json:"source_subtitle"`
	TargetTitle    string `json:"target_title"`
	TargetSubtitle string `json:"target_subtitle"`
}
type LocaleSurfaceReviewCoursePage struct {
	PageID       string `json:"page_id"`
	Route        string `json:"route"`
	Source       string `json:"source"`
	Target       string `json:"target"`
	Description  string `json:"description"`
	SourceSHA256 string `json:"source_sha256"`
	TargetSHA256 string `json:"target_sha256"`
}
type LocaleSurfaceReviewOtherSurface struct {
	ID           string   `json:"id"`
	Context      string   `json:"context"`
	Path         string   `json:"path"`
	SHA256       string   `json:"sha256"`
	SourceText   string   `json:"source_text"`
	UIReferences []string `json:"ui_references,omitempty"`
}

// ExportLocaleSurfaceReviewPackage validates every formal input and returns
// byte-stable JSON for an external language reviewer. It never writes files.
func ExportLocaleSurfaceReviewPackage(root, locale string, catalog *Catalog) ([]byte, LocaleSurfaceReviewPackageCoverage, error) {
	if catalog == nil {
		return nil, LocaleSurfaceReviewPackageCoverage{}, fmt.Errorf("catalog is required")
	}
	if err := ValidateLocaleName(locale); err != nil {
		return nil, LocaleSurfaceReviewPackageCoverage{}, err
	}
	if _, err := os.Lstat(filepath.Join(root, "locales", locale, ".locale-init-incomplete")); err == nil {
		return nil, LocaleSurfaceReviewPackageCoverage{}, fmt.Errorf("locale %s initialization is incomplete", locale)
	} else if !os.IsNotExist(err) {
		return nil, LocaleSurfaceReviewPackageCoverage{}, fmt.Errorf("check locale initialization marker: %w", err)
	}
	if err := CheckStatus(root, locale, catalog); err != nil {
		return nil, LocaleSurfaceReviewPackageCoverage{}, fmt.Errorf("formal locale status: %w", err)
	}
	if _, err := LoadGlossary(root, locale); err != nil {
		return nil, LocaleSurfaceReviewPackageCoverage{}, err
	}
	glossaryPath := filepath.Join(root, "locales", locale, "glossary.yaml")
	glossary, err := os.ReadFile(glossaryPath)
	if err != nil {
		return nil, LocaleSurfaceReviewPackageCoverage{}, err
	}
	uiRoot := filepath.Join(root, "internal", "tour", "ui")
	sourceUI, err := ui.LoadFromFS("en", os.DirFS(uiRoot))
	if err != nil {
		return nil, LocaleSurfaceReviewPackageCoverage{}, fmt.Errorf("load English UI catalog: %w", err)
	}
	targetUI, err := ui.LoadFromFS(locale, os.DirFS(uiRoot))
	if err != nil {
		return nil, LocaleSurfaceReviewPackageCoverage{}, fmt.Errorf("load locale UI catalog: %w", err)
	}
	metadata, err := LoadArticleMetadata(root, locale, catalog)
	if err != nil {
		return nil, LocaleSurfaceReviewPackageCoverage{}, err
	}
	course, err := LoadCourseMetadata(root, locale, catalog)
	if err != nil {
		return nil, LocaleSurfaceReviewPackageCoverage{}, err
	}
	inputs, err := CurrentLocaleSurfaceReviewAInputs(root, locale, catalog)
	if err != nil {
		return nil, LocaleSurfaceReviewPackageCoverage{}, err
	}
	public, err := currentSurfaceReviewPublicIdentity(root, locale)
	if err != nil {
		return nil, LocaleSurfaceReviewPackageCoverage{}, err
	}

	statuses, err := ReadStatuses(filepath.Join(root, "locales", locale, "status.tsv"))
	if err != nil {
		return nil, LocaleSurfaceReviewPackageCoverage{}, err
	}
	statusByID := map[string]*Status{}
	for i := range statuses {
		statusByID[statuses[i].UnitID] = &statuses[i]
	}
	workflow, _, _, err := localeWorkflowUnits(catalog)
	if err != nil {
		return nil, LocaleSurfaceReviewPackageCoverage{}, err
	}
	for id, unit := range workflow {
		status := statusByID[id]
		if status == nil {
			return nil, LocaleSurfaceReviewPackageCoverage{}, fmt.Errorf("%s: formal locale status is missing", id)
		}
		if _, err := loadReadyTranslationUnitCandidate(root, catalog, unit, locale, status); err != nil {
			return nil, LocaleSurfaceReviewPackageCoverage{}, err
		}
	}

	uiKeys := make([]string, 0, len(sourceUI.Messages))
	for key := range sourceUI.Messages {
		uiKeys = append(uiKeys, key)
	}
	sort.Strings(uiKeys)
	uiEntries := make([]LocaleSurfaceReviewUIMessage, 0, len(uiKeys))
	for _, key := range uiKeys {
		s, t := sourceUI.Messages[key], targetUI.Messages[key]
		uiEntries = append(uiEntries, LocaleSurfaceReviewUIMessage{Key: key, Kind: s.Kind, Source: s.Text, Target: t.Text})
	}
	articleNames := make([]string, 0, len(metadata))
	for name := range metadata {
		articleNames = append(articleNames, name)
	}
	sort.Strings(articleNames)
	articles := make([]LocaleSurfaceReviewArticle, 0, len(articleNames))
	for _, name := range articleNames {
		title, subtitle, err := sourceArticleHeader(filepath.Join(root, "_content", "tour", name))
		if err != nil {
			return nil, LocaleSurfaceReviewPackageCoverage{}, err
		}
		m := metadata[name]
		articles = append(articles, LocaleSurfaceReviewArticle{Article: name, SourcePath: filepath.ToSlash(filepath.Join("_content", "tour", name)), SourceTitle: title, SourceSubtitle: subtitle, TargetTitle: m.Title, TargetSubtitle: m.Subtitle})
	}
	courseByID := map[string]CoursePageMetadata{}
	for _, entry := range course.Pages {
		courseByID[entry.PageID] = entry
	}
	pages := make([]LocaleSurfaceReviewCoursePage, 0, len(catalog.Pages))
	for _, page := range catalog.Pages {
		target, err := loadReadyCandidate(root, catalog, page.ID, locale, statusByID[page.ID])
		if err != nil {
			return nil, LocaleSurfaceReviewPackageCoverage{}, err
		}
		entry, ok := courseByID[page.ID]
		if !ok {
			return nil, LocaleSurfaceReviewPackageCoverage{}, fmt.Errorf("course metadata is missing page %s", page.ID)
		}
		pages = append(pages, LocaleSurfaceReviewCoursePage{PageID: page.ID, Route: page.Route, Source: string(page.Source), Target: string(target), Description: entry.Description, SourceSHA256: page.SourceSHA256, TargetSHA256: entry.TargetSHA256})
	}
	others, err := exportOtherLocaleSurfaces(root, uiKeys)
	if err != nil {
		return nil, LocaleSurfaceReviewPackageCoverage{}, err
	}
	coverage := LocaleSurfaceReviewPackageCoverage{Pages: len(pages), UI: len(uiEntries), Articles: len(articles), TranslationUnits: len(workflow), OtherSurfaces: len(others)}
	pkg := LocaleSurfaceReviewPackage{SchemaVersion: 1, Kind: "go-tour-i18n/locale-surface-review-package", Locale: locale, Inputs: inputs, ProductionPublicIdentity: public, Coverage: coverage, Glossary: LocaleSurfaceReviewGlossary{Path: filepath.ToSlash(filepath.Join("locales", locale, "glossary.yaml")), SHA256: sum(glossary), Text: string(glossary)}, UI: uiEntries, Articles: articles, CoursePages: pages, OtherSurfaces: others}
	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return nil, LocaleSurfaceReviewPackageCoverage{}, err
	}
	return append(data, '\n'), coverage, nil
}

func exportOtherLocaleSurfaces(root string, uiKeys []string) ([]LocaleSurfaceReviewOtherSurface, error) {
	files := []localeSurfaceReviewSourceFile{
		{"language-registry-and-locale-profile", "Rendered language selector display identity (locale, EnglishName, Autonym, label, URL, official/current) and locale-visible time labels.", "internal/tour/languages.go", uiKeys},
		{"tour-shell-and-runtime", "Tour shell, navigation and runtime message composition; inspect every user-visible string that may bypass the UI catalog.", "internal/tour/tour.go", uiKeys},
		{"production-runtime-transport", "HTTP Playground transport and production runtime response handling; inspect direct user-visible runtime text and UI message composition.", "internal/tour/production.go", []string{"execution.exited", "execution.running", "execution.error"}},
		{"project-visible-identity", "Project ownership, upstream and public-link identity used by templates.", "internal/tour/project.go", nil},
		{"seo-visible-identity", "Canonical origin, robots, sitemap and locale-level SEO source/context.", "internal/tour/seo.go", []string{"tour.list_title", "tour.list_description"}},
		{"playground-runtime", "Shared first-party Playground transport and output rendering prepended to /tour/script.js; inspect hard-coded user-visible runtime text as well as UI catalog references.", "_content/js/playground.js", []string{"execution.exited", "execution.vet_failed", "execution.build_failed", "execution.communication_error", "execution.test_failed", "execution.tests_failed", "execution.tests_passed"}},
	}
	var err error
	files, err = appendLocaleSurfaceReviewDirectory(files, root, "_content/tour/static/js", ".js", "tour-runtime/", "First-party Tour JavaScript runtime source. The directory is scanned deterministically so new first-party runtime files enter the review package without duplicating initScript's dependency list.", uiKeys)
	if err != nil {
		return nil, err
	}
	files, err = appendLocaleSurfaceReviewDirectory(files, root, "_content/tour/template", ".tmpl", "tour-template/", "Formal Go template source used to compose the homepage, Tour shell, footer, and rendered lesson content.", uiKeys)
	if err != nil {
		return nil, err
	}
	files, err = appendLocaleSurfaceReviewDirectory(files, root, "_content/tour/static/partials", ".html", "tour-partial/", "First-party Angular view source used to compose the Tour list, editor, navigation, and feedback surfaces.", uiKeys)
	if err != nil {
		return nil, err
	}

	result := make([]LocaleSurfaceReviewOtherSurface, 0, len(files))
	for _, file := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file.path)))
		if err != nil {
			return nil, fmt.Errorf("read other locale-visible surface %s: %w", file.path, err)
		}
		result = append(result, LocaleSurfaceReviewOtherSurface{ID: file.id, Context: file.context, Path: file.path, SHA256: sum(data), SourceText: string(data), UIReferences: file.refs})
	}
	return result, nil
}

type localeSurfaceReviewSourceFile struct {
	id, context, path string
	refs              []string
}

func appendLocaleSurfaceReviewDirectory(files []localeSurfaceReviewSourceFile, root, directory, extension, idPrefix, context string, refs []string) ([]localeSurfaceReviewSourceFile, error) {
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(directory)))
	if err != nil {
		return nil, fmt.Errorf("read locale-visible source directory %s: %w", directory, err)
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Type().IsRegular() && filepath.Ext(entry.Name()) == extension {
			paths = append(paths, filepath.ToSlash(filepath.Join(directory, entry.Name())))
		}
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("locale-visible source directory %s has no %s files", directory, extension)
	}
	for _, path := range paths {
		files = append(files, localeSurfaceReviewSourceFile{
			id:      idPrefix + strings.TrimSuffix(filepath.Base(path), extension),
			context: context,
			path:    path,
			refs:    refs,
		})
	}
	return files, nil
}

func sourceArticleHeader(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read article source %s: %w", filepath.Base(path), err)
	}
	lines := bytes.SplitN(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")), []byte("\n"), 3)
	if len(lines) < 3 || strings.TrimSpace(string(lines[0])) == "" || strings.TrimSpace(string(lines[1])) == "" {
		return "", "", fmt.Errorf("article source %s lacks formal title/subtitle", filepath.Base(path))
	}
	return string(lines[0]), string(lines[1]), nil
}

func currentSurfaceReviewPublicIdentity(root, locale string) (SurfaceReviewPublicIdentity, error) {
	data, err := os.ReadFile(filepath.Join(root, "production", "identity.json"))
	if err != nil {
		return SurfaceReviewPublicIdentity{}, err
	}
	var identity localeSurfaceReviewProductionIdentity
	if err := json.Unmarshal(data, &identity); err != nil {
		return SurfaceReviewPublicIdentity{}, fmt.Errorf("parse production identity: %w", err)
	}
	var profiles []localeSurfaceReviewProductionProfile
	for _, profile := range identity.Locales {
		if profile.Locale == locale {
			profiles = append(profiles, profile)
		}
	}
	if len(profiles) != 1 || profiles[0].ProductionHostname == "" || profiles[0].ProductionPublicURL == "" {
		return SurfaceReviewPublicIdentity{}, fmt.Errorf("production public identity requires exactly one complete profile for %s", locale)
	}
	return SurfaceReviewPublicIdentity{Locale: profiles[0].Locale, ProductionHostname: profiles[0].ProductionHostname, ProductionPublicURL: profiles[0].ProductionPublicURL}, nil
}
