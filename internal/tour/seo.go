package tour

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type seoDocuments struct {
	origin                 string
	sitemap                []byte
	courseRoutes           []CourseRoute
	listRoute              ListRoute
	descriptions           map[string]string
	courseMetadataComplete bool
}

// CourseRoute is one canonical lesson/page route derived from the same
// parsed lesson data used to build sitemap.xml.
type CourseRoute struct {
	Path              string
	Canonical         string
	PageTitle         string
	LessonTitle       string
	LessonDescription string
	Files             []string
	Description       string
}

// ListRoute is the locale-level, non-TranslationUnit SEO identity for the
// sitemap's /tour/list surface. Course metadata remains only on CourseRoute.
type ListRoute struct {
	Path        string
	Canonical   string
	PageTitle   string
	Description string
	Heading     string
	Modules     []ListModule
	Lessons     []CourseRoute
}

type ListModule struct {
	Title       string
	Description string
}

func productionOriginForLocale(locale string) (string, error) {
	for _, language := range languageRegistry {
		if language.Locale != locale {
			continue
		}
		u, err := url.Parse(language.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
			return "", fmt.Errorf("invalid production URL for locale %q: %q", locale, language.URL)
		}
		return u.Scheme + "://" + u.Host, nil
	}
	return "", fmt.Errorf("missing production URL for locale %q", locale)
}

func initSEO(locale string) (seoDocuments, error) {
	origin, err := productionOriginForLocale(locale)
	if err != nil {
		return seoDocuments{}, err
	}
	urls := []string{origin + "/", origin + "/tour/list"}
	descriptions, metadataPresent, err := loadProjectedCourseSEO(contentTour)
	if err != nil {
		return seoDocuments{}, err
	}
	var courseRoutes []CourseRoute
	seenRoutes := make(map[string]bool)
	articles := make([]string, 0, len(lessons))
	for article := range lessons {
		articles = append(articles, article)
	}
	sort.Strings(articles)
	for _, name := range articles {
		var l lesson
		if err := json.Unmarshal(lessons[name], &l); err != nil {
			return seoDocuments{}, fmt.Errorf("parse lesson %s for sitemap: %w", name, err)
		}
		for pageIndex, page := range l.Pages {
			routePath := fmt.Sprintf("/tour/%s/%d", name, pageIndex+1)
			canonical := origin + routePath
			urls = append(urls, canonical)
			files := make([]string, len(page.Files))
			for i := range page.Files {
				files[i] = page.Files[i].Content
			}
			courseRoutes = append(courseRoutes, CourseRoute{
				Path:              routePath,
				Canonical:         canonical,
				PageTitle:         page.Title,
				LessonTitle:       l.Title,
				LessonDescription: l.Description,
				Files:             files,
				Description:       descriptions[routePath],
			})
			seenRoutes[routePath] = true
		}
	}
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
	for _, loc := range urls {
		fmt.Fprintf(&b, "  <url><loc>%s</loc></url>\n", loc)
	}
	b.WriteString("</urlset>\n")
	if metadataPresent {
		for _, route := range courseRoutes {
			if route.Description == "" {
				return seoDocuments{}, fmt.Errorf("projected course SEO is missing route %q", route.Path)
			}
		}
		for route := range descriptions {
			if !seenRoutes[route] {
				return seoDocuments{}, fmt.Errorf("projected course SEO has unknown route %q", route)
			}
		}
	}
	return seoDocuments{origin: origin, sitemap: []byte(b.String()), courseRoutes: courseRoutes, descriptions: descriptions, courseMetadataComplete: metadataPresent}, nil
}

type projectedCourseSEOFile struct {
	Pages []projectedCourseSEOPage `json:"pages"`
}

type projectedCourseSEOPage struct {
	Route       string `json:"route"`
	Description string `json:"description"`
}

func loadProjectedCourseSEO(content fs.FS) (map[string]string, bool, error) {
	data, err := fs.ReadFile(content, "tour/course-seo.json")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]string{}, false, nil
		}
		return nil, false, fmt.Errorf("read projected course SEO: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var file projectedCourseSEOFile
	if err := decoder.Decode(&file); err != nil {
		return nil, false, fmt.Errorf("parse projected course SEO: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, false, fmt.Errorf("parse projected course SEO: trailing JSON")
	}
	result := make(map[string]string, len(file.Pages))
	for _, page := range file.Pages {
		if page.Route == "" || page.Description == "" {
			return nil, false, fmt.Errorf("projected course SEO contains an empty route or description")
		}
		if _, exists := result[page.Route]; exists {
			return nil, false, fmt.Errorf("projected course SEO has duplicate route %q", page.Route)
		}
		result[page.Route] = page.Description
	}
	return result, true, nil
}

func robotsHandler(documents seoDocuments) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "User-agent: *\nAllow: /\n\nSitemap: %s/sitemap.xml\n", documents.origin)
	}
}

func sitemapHandler(documents seoDocuments) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write(documents.sitemap)
	}
}
