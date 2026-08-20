package tour

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

const sitemapHost = "https://go-dev.shuijingwanwq.com"

var sitemapContent []byte

func initSEO() error {
	urls := []string{sitemapHost + "/", sitemapHost + "/tour/list"}
	articles := make([]string, 0, len(lessons))
	for article := range lessons {
		articles = append(articles, article)
	}
	sort.Strings(articles)
	for _, name := range articles {
		var l lesson
		if err := json.Unmarshal(lessons[name], &l); err != nil {
			return fmt.Errorf("parse lesson %s for sitemap: %w", name, err)
		}
		for page := range l.Pages {
			urls = append(urls, fmt.Sprintf("%s/tour/%s/%d", sitemapHost, name, page+1))
		}
	}
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
	for _, loc := range urls {
		fmt.Fprintf(&b, "  <url><loc>%s</loc></url>\n", loc)
	}
	b.WriteString("</urlset>\n")
	sitemapContent = []byte(b.String())
	return nil
}

func robotsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "User-agent: *\nAllow: /\n\nSitemap: %s/sitemap.xml\n", sitemapHost)
}

func sitemapHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write(sitemapContent)
}
