package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PreviewContent struct {
	Root       string
	ContentDir string
	PageID     string
	Locale     string
}

func BuildCandidatePreview(root string, catalog *Catalog, pageID, locale, tempRoot string) (*PreviewContent, error) {
	page, err := catalog.Page(pageID)
	if err != nil {
		return nil, err
	}
	status, candidate, err := LoadTranslationResult(root, pageID, locale)
	if err != nil {
		return nil, err
	}
	if status.State != "ready" || status.CandidatePath == "" {
		return nil, fmt.Errorf("%s/%s is not a ready candidate", locale, pageID)
	}
	if status.SourceSHA256 != page.SourceSHA256 || sum(page.Source) != status.SourceSHA256 {
		return nil, fmt.Errorf("%s/%s candidate source hash does not match current source", locale, pageID)
	}
	if err := ValidateCandidate(root, catalog, pageID, []byte(candidate)); err != nil {
		return nil, fmt.Errorf("%s/%s candidate validation: %w", locale, pageID, err)
	}
	tempRoot = filepath.Clean(tempRoot)
	tempBase := filepath.Clean(os.TempDir())
	rel, err := filepath.Rel(tempBase, tempRoot)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("unsafe preview temp root %q", tempRoot)
	}
	if err := os.RemoveAll(tempRoot); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(tempRoot, 0755); err != nil {
		return nil, err
	}
	contentDir := filepath.Join(tempRoot, "_content")
	if err := os.CopyFS(contentDir, os.DirFS(filepath.Join(root, "_content"))); err != nil {
		return nil, fmt.Errorf("copy Tour content: %w", err)
	}
	articlePath := filepath.Join(contentDir, "tour", page.Article)
	article, err := os.ReadFile(articlePath)
	if err != nil {
		return nil, err
	}
	article, err = projectPublishedArticle(article, page.Article)
	if err != nil {
		return nil, fmt.Errorf("project %s: %w", page.Article, err)
	}
	replaced, err := replacePreviewSection(article, page.SectionNumber, []byte(candidate))
	if err != nil {
		return nil, fmt.Errorf("replace %s: %w", pageID, err)
	}
	if err := os.WriteFile(articlePath, replaced, 0644); err != nil {
		return nil, err
	}
	return &PreviewContent{Root: tempRoot, ContentDir: contentDir, PageID: pageID, Locale: locale}, nil
}

func replacePreviewSection(article []byte, sectionNumber int, candidate []byte) ([]byte, error) {
	var starts []int
	offset := 0
	for _, line := range strings.SplitAfter(string(article), "\n") {
		if strings.HasPrefix(line, "* ") {
			starts = append(starts, offset)
		}
		offset += len(line)
	}
	if sectionNumber < 1 || sectionNumber > len(starts) {
		return nil, fmt.Errorf("section %d not found", sectionNumber)
	}
	start := starts[sectionNumber-1]
	end := len(article)
	if sectionNumber < len(starts) {
		end = starts[sectionNumber]
	}
	candidate = normalizeLF(candidate)
	candidate = bytes.TrimRight(candidate, "\n")
	candidate = append(candidate, '\n', '\n')
	out := make([]byte, 0, len(article)-end+start+len(candidate))
	out = append(out, article[:start]...)
	out = append(out, candidate...)
	out = append(out, article[end:]...)
	return out, nil
}
