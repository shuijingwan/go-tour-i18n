package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
	"golang.org/x/tools/present"
)

// LocaleProjection describes a complete, locally runnable Tour content tree.
// Root contains ContentDir as its _content child so the result stays separate
// from the checked-in English source.
type LocaleProjection struct {
	Root         string
	ContentDir   string
	Locale       string
	Ready        int
	Pending      int
	Blocked      int
	UnitCount    int
	PageCount    int
	ExampleCount int
	ArticleCount int
}

// BuildLocaleProjection builds the complete formal projection for locale from
// the committed catalog, locale status, and canonical ready candidates. It
// never reads development attempts and never falls back to source content.
func BuildLocaleProjection(root string, catalog *Catalog, locale, outputRoot string) (_ *LocaleProjection, err error) {
	if catalog == nil {
		return nil, fmt.Errorf("catalog is required")
	}
	if _, err := ui.Load(locale); err != nil {
		return nil, fmt.Errorf("locale %q has no complete Tour UI catalog: %w", locale, err)
	}
	if err := CheckStatus(root, locale, catalog); err != nil {
		return nil, fmt.Errorf("formal locale status: %w", err)
	}
	metadataByArticle, err := LoadArticleMetadata(root, locale, catalog)
	if err != nil {
		return nil, err
	}
	statuses, err := ReadStatuses(filepath.Join(root, "locales", locale, "status.tsv"))
	if err != nil {
		return nil, fmt.Errorf("read formal locale status: %w", err)
	}

	statusByID := make(map[string]Status, len(statuses))
	result := &LocaleProjection{Locale: locale}
	for _, status := range statuses {
		_, unitErr := catalog.Unit(status.UnitID)
		if unitErr != nil {
			return nil, unitErr
		}
		if _, exists := statusByID[status.UnitID]; exists {
			return nil, fmt.Errorf("duplicate status unit_id %q", status.UnitID)
		}
		statusByID[status.UnitID] = status
		switch status.State {
		case "ready":
			result.Ready++
		case "pending":
			result.Pending++
		case "blocked":
			result.Blocked++
		}
	}

	candidates := make(map[string][]byte, len(catalog.Pages))
	exampleCandidates := make(map[string][]byte)
	pagesByArticle := make(map[string][]Page)
	seenIDs := make(map[string]bool, len(catalog.Pages))
	seenLocations := make(map[string]string, len(catalog.Pages))
	var unavailable []string
	for _, page := range catalog.Pages {
		if seenIDs[page.ID] {
			return nil, fmt.Errorf("duplicate catalog page_id %q", page.ID)
		}
		seenIDs[page.ID] = true
		if page.Article == "" || filepath.Base(page.Article) != page.Article || filepath.Ext(page.Article) != ".article" {
			return nil, fmt.Errorf("%s: unsafe article path %q", page.ID, page.Article)
		}
		if page.SectionNumber < 1 {
			return nil, fmt.Errorf("%s: invalid section number %d", page.ID, page.SectionNumber)
		}
		location := fmt.Sprintf("%s:%d", page.Article, page.SectionNumber)
		if previous := seenLocations[location]; previous != "" {
			return nil, fmt.Errorf("catalog pages %s and %s share %s", previous, page.ID, location)
		}
		seenLocations[location] = page.ID
		status, ok := statusByID[page.ID]
		if !ok {
			return nil, fmt.Errorf("catalog page %q is missing from status", page.ID)
		}
		if status.State != "ready" {
			unavailable = append(unavailable, fmt.Sprintf("%s=%s", page.ID, status.State))
			continue
		}
		candidate, err := loadReadyCandidate(root, catalog, page.ID, locale, &status)
		if err != nil {
			return nil, err
		}
		candidates[page.ID] = candidate
		pagesByArticle[page.Article] = append(pagesByArticle[page.Article], page)
	}
	workflow, _, exampleCount, err := localeWorkflowUnits(catalog)
	if err != nil {
		return nil, err
	}
	for unitID, unit := range workflow {
		if unit.Kind != UnitKindExample {
			continue
		}
		status := statusByID[unitID]
		if status.State != "ready" {
			unavailable = append(unavailable, fmt.Sprintf("%s=%s", unitID, status.State))
			continue
		}
		candidate, err := loadReadyTranslationUnitCandidate(root, catalog, unit, locale, &status)
		if err != nil {
			return nil, err
		}
		exampleCandidates[unitID] = candidate
	}
	if len(unavailable) > 0 {
		sort.Strings(unavailable)
		return nil, fmt.Errorf("complete projection requires all %d workflow translation units to be ready; unavailable: %s", len(workflow), strings.Join(unavailable, ", "))
	}

	if strings.TrimSpace(outputRoot) == "" {
		return nil, fmt.Errorf("output directory is required")
	}
	outputRoot, err = filepath.Abs(outputRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve output directory: %w", err)
	}
	if err := prepareProjectionRoot(outputRoot); err != nil {
		return nil, err
	}
	contentDir := filepath.Join(outputRoot, "_content")
	sourceContentDir, err := filepath.Abs(filepath.Join(root, "_content"))
	if err != nil {
		return nil, fmt.Errorf("resolve source content directory: %w", err)
	}
	if rel, relErr := filepath.Rel(sourceContentDir, contentDir); relErr != nil || rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return nil, fmt.Errorf("output content directory %q must not be inside source content %q", contentDir, sourceContentDir)
	}
	createdContent := false
	defer func() {
		if err != nil && createdContent {
			_ = os.RemoveAll(contentDir)
		}
	}()
	createdContent = true
	if err = os.CopyFS(contentDir, os.DirFS(sourceContentDir)); err != nil {
		return nil, fmt.Errorf("copy Tour content: %w", err)
	}
	for unitID, candidate := range exampleCandidates {
		unit := workflow[unitID]
		relative, relErr := filepath.Rel("_content", filepath.FromSlash(unit.SourcePath))
		if relErr != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("%s: unsafe Example source path %q", unitID, unit.SourcePath)
		}
		target := filepath.Join(contentDir, relative)
		if err = os.WriteFile(target, candidate, 0644); err != nil {
			return nil, fmt.Errorf("write projected Example %s: %w", unitID, err)
		}
	}

	if err = validateProjectionArticleSet(contentDir, pagesByArticle); err != nil {
		return nil, err
	}
	articleNames := make([]string, 0, len(pagesByArticle))
	for article := range pagesByArticle {
		articleNames = append(articleNames, article)
	}
	sort.Strings(articleNames)
	for _, article := range articleNames {
		articlePath := filepath.Join(contentDir, "tour", article)
		source, readErr := os.ReadFile(articlePath)
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", article, readErr)
		}
		replacements := make(map[int][]byte, len(pagesByArticle[article]))
		for _, page := range pagesByArticle[article] {
			replacements[page.SectionNumber] = candidates[page.ID]
		}
		projected, projectErr := projectLocalizedArticle(source, article, metadataByArticle[article], replacements)
		if projectErr != nil {
			return nil, fmt.Errorf("project %s: %w", article, projectErr)
		}
		if writeErr := os.WriteFile(articlePath, projected, 0644); writeErr != nil {
			return nil, fmt.Errorf("write %s: %w", article, writeErr)
		}
	}
	if err = validateCompleteProjection(root, contentDir, catalog, locale, candidates, metadataByArticle); err != nil {
		return nil, fmt.Errorf("validate complete projection: %w", err)
	}

	result.Root = outputRoot
	result.ContentDir = contentDir
	result.UnitCount = len(workflow)
	result.PageCount = len(catalog.Pages)
	result.ExampleCount = exampleCount
	result.ArticleCount = len(pagesByArticle)
	return result, nil
}

func loadReadyTranslationUnitCandidate(root string, catalog *Catalog, unit *TranslationUnit, locale string, status *Status) ([]byte, error) {
	if unit == nil || status == nil || status.State != "ready" || status.CandidatePath == "" {
		return nil, fmt.Errorf("%s is not a ready translation unit", unit.ID)
	}
	wantPath, err := canonicalTranslationUnitCandidatePath(locale, unit)
	if err != nil {
		return nil, err
	}
	if status.CandidatePath != wantPath || status.SourceSHA256 != unit.SourceSHA256 || sum(unit.Source) != unit.SourceSHA256 {
		return nil, fmt.Errorf("%s: ready candidate identity/path mismatch", unit.ID)
	}
	candidate, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(status.CandidatePath)))
	if err != nil {
		return nil, fmt.Errorf("%s: read canonical candidate: %w", unit.ID, err)
	}
	if err := ValidateTranslationUnitCandidate(root, catalog, unit.ID, locale, candidate); err != nil {
		return nil, fmt.Errorf("%s: candidate validation: %w", unit.ID, err)
	}
	return candidate, nil
}

func canonicalCandidatePath(locale, pageID string) string {
	path, err := canonicalTranslationUnitCandidatePath(locale, &TranslationUnit{ID: pageID, Kind: UnitKindPage})
	if err != nil {
		panic(err)
	}
	return path
}

func canonicalTranslationUnitCandidatePath(locale string, unit *TranslationUnit) (string, error) {
	if unit == nil {
		return "", fmt.Errorf("translation unit is required")
	}
	if unit.Kind != UnitKindPage && unit.Kind != UnitKindExample {
		return "", fmt.Errorf("%s: unsupported translation unit kind %q", unit.ID, unit.Kind)
	}
	name := retranslationUnitCandidateName(unit)
	if name == "" || filepath.Base(name) != name {
		return "", fmt.Errorf("%s: unsafe canonical candidate name %q", unit.ID, name)
	}
	return filepath.ToSlash(filepath.Join("locales", locale, "candidates", name)), nil
}

func flattenedPageArticleName(pageID string) string {
	return strings.ReplaceAll(pageID, "/", "-") + ".article"
}

func loadReadyCandidate(root string, catalog *Catalog, pageID, locale string, status *Status) ([]byte, error) {
	page, err := catalog.Page(pageID)
	if err != nil {
		return nil, err
	}
	if status == nil {
		loaded, err := loadPageStatus(root, pageID, locale)
		if err != nil {
			return nil, err
		}
		status = loaded
	}
	if status.State != "ready" || status.CandidatePath == "" {
		return nil, fmt.Errorf("%s/%s is not a ready candidate", locale, pageID)
	}
	wantPath := canonicalCandidatePath(locale, pageID)
	if status.CandidatePath != wantPath {
		return nil, fmt.Errorf("%s/%s candidate_path %q is not canonical %q", locale, pageID, status.CandidatePath, wantPath)
	}
	if status.SourceSHA256 != page.SourceSHA256 || sum(page.Source) != status.SourceSHA256 {
		return nil, fmt.Errorf("%s/%s candidate source hash does not match current source", locale, pageID)
	}
	candidate, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(status.CandidatePath)))
	if err != nil {
		return nil, fmt.Errorf("%s/%s read canonical candidate: %w", locale, pageID, err)
	}
	if containsProtectedTranslationToken(candidate) {
		return nil, fmt.Errorf("%s/%s canonical candidate contains a protected token", locale, pageID)
	}
	if err := ValidateCandidateForLocale(root, catalog, pageID, locale, candidate); err != nil {
		return nil, fmt.Errorf("%s/%s candidate validation: %w", locale, pageID, err)
	}
	return candidate, nil
}

func loadPageStatus(root, pageID, locale string) (*Status, error) {
	statuses, err := ReadStatuses(filepath.Join(root, "locales", locale, "status.tsv"))
	if err != nil {
		return nil, err
	}
	for i := range statuses {
		if statuses[i].UnitID == pageID {
			return &statuses[i], nil
		}
	}
	return nil, fmt.Errorf("unknown page_id %q", pageID)
}

func containsProtectedTranslationToken(data []byte) bool {
	return translationTokenRE.Match(data) || bytes.Contains(data, []byte("⟪GTI18N_"))
}

func prepareProjectionRoot(outputRoot string) error {
	if strings.TrimSpace(outputRoot) == "" {
		return fmt.Errorf("output directory is required")
	}
	info, err := os.Stat(outputRoot)
	switch {
	case os.IsNotExist(err):
		if err := os.MkdirAll(outputRoot, 0755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("inspect output directory: %w", err)
	case !info.IsDir():
		return fmt.Errorf("output path %q is not a directory", outputRoot)
	}
	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		return fmt.Errorf("read output directory: %w", err)
	}
	if len(entries) != 0 {
		return fmt.Errorf("output directory %q is not empty", outputRoot)
	}
	return nil
}

// projectCandidateSections is shared by single-page preview and complete
// locale projection. The publication projection (including welcome's special
// branches) happens once, then replacements are applied from the last Section
// backwards so offsets cannot invalidate another replacement.
func projectCandidateSections(article []byte, articleName string, replacements map[int][]byte) ([]byte, error) {
	projected, err := projectPublishedArticle(article, articleName)
	if err != nil {
		return nil, err
	}
	sections := make([]int, 0, len(replacements))
	for section := range replacements {
		sections = append(sections, section)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sections)))
	for _, section := range sections {
		projected, err = replacePreviewSection(projected, section, replacements[section])
		if err != nil {
			return nil, err
		}
	}
	return projected, nil
}

func validateProjectionArticleSet(contentDir string, pagesByArticle map[string][]Page) error {
	matches, err := filepath.Glob(filepath.Join(contentDir, "tour", "*.article"))
	if err != nil {
		return err
	}
	actual := make(map[string]bool, len(matches))
	for _, match := range matches {
		actual[filepath.Base(match)] = true
	}
	for article := range pagesByArticle {
		if !actual[article] {
			return fmt.Errorf("catalog article %q is missing from copied Tour content", article)
		}
	}
	for article := range actual {
		if _, ok := pagesByArticle[article]; !ok {
			return fmt.Errorf("copied Tour content has article %q absent from catalog", article)
		}
	}
	return nil
}

func validateCompleteProjection(root, contentDir string, catalog *Catalog, locale string, candidates map[string][]byte, metadataByArticle map[string]ArticleMetadata) error {
	pagesByArticle := make(map[string][]Page)
	for _, page := range catalog.Pages {
		pagesByArticle[page.Article] = append(pagesByArticle[page.Article], page)
	}
	seen := make(map[string]bool, len(catalog.Pages))
	for article, pages := range pagesByArticle {
		data, err := os.ReadFile(filepath.Join(contentDir, "tour", article))
		if err != nil {
			return err
		}
		if containsProtectedTranslationToken(data) {
			return fmt.Errorf("%s contains a protected token", article)
		}
		doc, err := parseProjectedArticle(contentDir, article, data)
		if err != nil {
			return fmt.Errorf("%s: %w", article, err)
		}
		metadata, ok := metadataByArticle[article]
		if !ok {
			return fmt.Errorf("%s has no article metadata", article)
		}
		if doc.Title != metadata.Title {
			return fmt.Errorf("%s final title %q does not equal locale metadata %q", article, doc.Title, metadata.Title)
		}
		if doc.Subtitle != metadata.Subtitle {
			return fmt.Errorf("%s final subtitle %q does not equal locale metadata %q", article, doc.Subtitle, metadata.Subtitle)
		}
		sections, _, err := splitArticle(normalizeLF(data), article)
		if err != nil {
			return fmt.Errorf("%s sections: %w", article, err)
		}
		if len(doc.Sections) != len(sections) {
			return fmt.Errorf("%s parsed Section count %d differs from source split %d", article, len(doc.Sections), len(sections))
		}
		if len(sections) != len(pages) {
			return fmt.Errorf("%s final pages = %d, catalog pages = %d", article, len(sections), len(pages))
		}
		for _, page := range pages {
			if page.SectionNumber < 1 || page.SectionNumber > len(sections) {
				return fmt.Errorf("%s: final Section %d is missing", page.ID, page.SectionNumber)
			}
			if seen[page.ID] {
				return fmt.Errorf("duplicate final page_id %q", page.ID)
			}
			seen[page.ID] = true
			actual := sections[page.SectionNumber-1]
			candidate, ok := candidates[page.ID]
			if !ok {
				return fmt.Errorf("%s has no canonical candidate in projection", page.ID)
			}
			if !sameProjectedSection(actual, candidate) {
				return fmt.Errorf("%s final Section does not equal its canonical candidate", page.ID)
			}
			if err := ValidateCandidateForLocale(root, catalog, page.ID, locale, actual); err != nil {
				return fmt.Errorf("%s final Section validation: %w", page.ID, err)
			}
		}
	}
	if len(seen) != len(catalog.Pages) {
		return fmt.Errorf("final page_id set has %d pages, catalog has %d", len(seen), len(catalog.Pages))
	}
	return nil
}

func sameProjectedSection(actual, candidate []byte) bool {
	trim := func(data []byte) []byte { return bytes.TrimRight(normalizeLF(data), "\n") }
	return bytes.Equal(trim(actual), trim(candidate))
}

func parseProjectedArticle(contentDir, article string, source []byte) (*present.Doc, error) {
	present.PlayEnabled = true
	ctx := &present.Context{ReadFile: func(name string) ([]byte, error) {
		clean := filepath.Clean(filepath.FromSlash(name))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("unsafe referenced path %q", name)
		}
		return os.ReadFile(filepath.Join(contentDir, "tour", clean))
	}}
	doc, err := ctx.Parse(bytes.NewReader(source), article, 0)
	if err != nil {
		return nil, fmt.Errorf("present parse: %w", err)
	}
	return doc, nil
}
