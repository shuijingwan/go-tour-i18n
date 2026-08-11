package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ArticleMetadata is the locale-specific document header for one formal Tour
// article. It is intentionally separate from Section candidates and Tour UI.
type ArticleMetadata struct {
	Article  string `json:"article"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
}

type articleMetadataFile struct {
	Locale   string            `json:"locale"`
	Articles []ArticleMetadata `json:"articles"`
}

// LoadArticleMetadata reads and strictly validates a locale's complete formal
// article metadata set. The required article set comes from the page catalog,
// so the mechanism is reusable by every locale and catalog revision.
func LoadArticleMetadata(root, locale string, catalog *Catalog) (map[string]ArticleMetadata, error) {
	if catalog == nil {
		return nil, fmt.Errorf("catalog is required")
	}
	path := filepath.Join(root, "locales", locale, "article-metadata.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read article metadata: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var file articleMetadataFile
	if err := decoder.Decode(&file); err != nil {
		return nil, fmt.Errorf("parse article metadata: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("parse article metadata: multiple JSON values")
		}
		return nil, fmt.Errorf("parse article metadata: %w", err)
	}
	if file.Locale != locale {
		return nil, fmt.Errorf("article metadata locale %q does not match requested locale %q", file.Locale, locale)
	}

	expected := catalogArticleSet(catalog)
	metadata := make(map[string]ArticleMetadata, len(file.Articles))
	for _, entry := range file.Articles {
		if entry.Article == "" || filepath.Base(entry.Article) != entry.Article || filepath.Ext(entry.Article) != ".article" {
			return nil, fmt.Errorf("article metadata has unsafe article %q", entry.Article)
		}
		if _, exists := metadata[entry.Article]; exists {
			return nil, fmt.Errorf("article metadata has duplicate article %q", entry.Article)
		}
		if _, ok := expected[entry.Article]; !ok {
			return nil, fmt.Errorf("article metadata has extra article %q", entry.Article)
		}
		if strings.TrimSpace(entry.Title) == "" {
			return nil, fmt.Errorf("article metadata %q has empty title", entry.Article)
		}
		if strings.TrimSpace(entry.Subtitle) == "" {
			return nil, fmt.Errorf("article metadata %q has empty subtitle", entry.Article)
		}
		if strings.ContainsAny(entry.Title, "\r\n") || strings.ContainsAny(entry.Subtitle, "\r\n") {
			return nil, fmt.Errorf("article metadata %q title and subtitle must be single lines", entry.Article)
		}
		metadata[entry.Article] = entry
	}
	missing := make([]string, 0)
	for article := range expected {
		if _, ok := metadata[article]; !ok {
			missing = append(missing, article)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("article metadata is missing article(s): %s", strings.Join(missing, ", "))
	}
	return metadata, nil
}

func catalogArticleSet(catalog *Catalog) map[string]struct{} {
	articles := make(map[string]struct{})
	for _, page := range catalog.Pages {
		articles[page.Article] = struct{}{}
	}
	return articles
}

// projectLocalizedArticle applies the localized document header and then the
// shared canonical Section projection. Both full and single-page previews use
// this path so their article metadata behavior cannot diverge.
func projectLocalizedArticle(article []byte, articleName string, metadata ArticleMetadata, replacements map[int][]byte) ([]byte, error) {
	if metadata.Article != articleName {
		return nil, fmt.Errorf("metadata article %q does not match %q", metadata.Article, articleName)
	}
	localized, err := applyArticleMetadata(article, metadata)
	if err != nil {
		return nil, err
	}
	return projectCandidateSections(localized, articleName, replacements)
}

func applyArticleMetadata(article []byte, metadata ArticleMetadata) ([]byte, error) {
	article = normalizeLF(article)
	firstEnd := bytes.IndexByte(article, '\n')
	if firstEnd < 0 {
		return nil, fmt.Errorf("article has no title line")
	}
	remainder := article[firstEnd+1:]
	secondEnd := bytes.IndexByte(remainder, '\n')
	if secondEnd < 0 {
		return nil, fmt.Errorf("article has no subtitle line")
	}
	var out bytes.Buffer
	out.Grow(len(article) + len(metadata.Title) + len(metadata.Subtitle))
	out.WriteString(metadata.Title)
	out.WriteByte('\n')
	out.WriteString(metadata.Subtitle)
	out.WriteByte('\n')
	out.Write(remainder[secondEnd+1:])
	return out.Bytes(), nil
}
