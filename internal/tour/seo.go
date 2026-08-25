package tour

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type seoDocuments struct {
	origin       string
	sitemap      []byte
	courseRoutes []CourseRoute
}

// CourseRoute is one canonical lesson/page route derived from the same
// parsed lesson data used to build sitemap.xml.
type CourseRoute struct {
	Path        string
	Canonical   string
	PageTitle   string
	LessonTitle string
	Files       []string
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
	var courseRoutes []CourseRoute
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
				Path:        routePath,
				Canonical:   canonical,
				PageTitle:   page.Title,
				LessonTitle: l.Title,
				Files:       files,
			})
		}
	}
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
	for _, loc := range urls {
		fmt.Fprintf(&b, "  <url><loc>%s</loc></url>\n", loc)
	}
	b.WriteString("</urlset>\n")
	return seoDocuments{origin: origin, sitemap: []byte(b.String()), courseRoutes: courseRoutes}, nil
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
