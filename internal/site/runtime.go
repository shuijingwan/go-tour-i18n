// Package site owns opt-in Site v2 public routing without changing legacy Tour.
package site

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
	"github.com/shuijingwan/go-tour-i18n/internal/tourpolicy"
)

type Resolver struct {
	origin    string
	routes    map[string]bool
	aliases   map[string]string
	canonical map[string]bool
}

// Load verifies repository scope and completion evidence before making coverage
// visible. Production identity remains independent of content completion.
func Load(root, locale, origin string) (*Resolver, error) {
	formal, err := sitecontent.LocaleOrigin(root, locale)
	if err != nil {
		return nil, err
	}
	if strings.TrimSuffix(formal, "/") != strings.TrimSuffix(origin, "/") {
		return nil, fmt.Errorf("locale hostname differs from deployment identity")
	}
	g, l, err := sitecontent.LocalCoverage(root, locale)
	if err != nil {
		return nil, err
	}
	aliases, err := sitecontent.RedirectContracts(root, g)
	if err != nil {
		return nil, err
	}
	return newResolver(g, l, origin, aliases)
}
func newResolver(g *sitecontent.Global, l *sitecontent.Locale, origin string, aliases map[string]string) (*Resolver, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, fmt.Errorf("invalid locale origin")
	}
	if err := sitecontent.ValidateLocale(g, *l); err != nil {
		return nil, err
	}
	r := &Resolver{origin: strings.TrimSuffix(origin, "/"), routes: map[string]bool{}, aliases: aliases, canonical: map[string]bool{}}
	for _, p := range g.Packages {
		for _, route := range p.Routes {
			r.canonical[route] = true
		}
	}
	for _, p := range l.Packages {
		for _, route := range p.Routes {
			r.routes[route] = true
		}
	}
	return r, nil
}
func (r *Resolver) Covered(route string) bool { return r.routes[route] }

// Destination is a navigation decision. It never registers fallback HTTP routes.
func (r *Resolver) Destination(destination string) (string, error) {
	u, err := url.Parse(destination)
	if err != nil || u.User != nil {
		return "", fmt.Errorf("invalid destination")
	}
	if u.IsAbs() || u.Host != "" {
		if (u.Scheme == "https" || u.Scheme == "http") && u.Host == "" {
			return "", fmt.Errorf("external HTTP URL requires host")
		}
		if u.Scheme != "" && u.Scheme != "https" && u.Scheme != "http" && u.Scheme != "mailto" {
			return "", fmt.Errorf("unsupported external scheme")
		}
		return destination, nil
	}
	if !strings.HasPrefix(u.Path, "/") || strings.HasPrefix(u.Path, "//") || strings.ContainsAny(u.Path, "\\\x00") || (path.Clean(u.Path) != u.Path && path.Clean(u.Path)+"/" != u.Path) {
		return "", fmt.Errorf("canonical absolute route required")
	}
	route := u.Path
	for _, suffix := range []string{".html", ".md"} {
		if strings.HasSuffix(route, suffix) && r.canonical[strings.TrimSuffix(route, suffix)] {
			route = strings.TrimSuffix(route, suffix)
		}
	}
	if target := r.aliasTarget(route); target != "" {
		if strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "http://") {
			return target, nil
		}
		route = target
	}
	u.Path = route
	u.RawPath = ""
	base := "https://go.dev"
	if r.routes[route] {
		base = r.origin
	}
	return base + u.String(), nil
}

// ResolveLink gives relative source links their route context before applying
// the same coverage policy; original external URLs remain external.
func (r *Resolver) ResolveLink(from, destination string) (string, error) {
	u, err := url.Parse(destination)
	if err != nil {
		return "", err
	}
	if u.IsAbs() || u.Host != "" {
		return r.Destination(destination)
	}
	base, err := url.Parse(from)
	if err != nil || !strings.HasPrefix(base.Path, "/") {
		return "", fmt.Errorf("invalid route context")
	}
	return r.Destination(base.ResolveReference(u).String())
}
func (r *Resolver) aliasTarget(route string) string {
	if t := r.aliases[route]; t != "" {
		return t
	}
	best := ""
	for k := range r.aliases {
		if strings.HasSuffix(k, "/") && (strings.HasPrefix(route, k) || route+"/" == k) && len(k) > len(best) {
			best = k
		}
	}
	return r.aliases[best]
}
func (r *Resolver) Canonical(route string) (string, error) {
	if !r.routes[route] {
		return "", fmt.Errorf("no local canonical coverage")
	}
	return r.origin + route, nil
}
func (r *Resolver) Sitemap() ([]byte, error) {
	type entry struct {
		Loc string `xml:"loc"`
	}
	type set struct {
		XMLName xml.Name `xml:"urlset"`
		XMLNS   string   `xml:"xmlns,attr"`
		URLs    []entry  `xml:"url"`
	}
	routes := []string{}
	for route := range r.routes {
		routes = append(routes, route)
	}
	sort.Strings(routes)
	s := set{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	for _, route := range routes {
		s.URLs = append(s.URLs, entry{r.origin + route})
	}
	b, err := xml.Marshal(s)
	return append([]byte(xml.Header), b...), err
}

// Handler requires concrete content handlers for every covered route. It does
// not render shells or redirect to English for a missing local request.
func (r *Resolver) Handler(pages map[string]http.Handler) (http.Handler, error) {
	for route := range r.routes {
		if pages[route] == nil {
			return nil, fmt.Errorf("covered route has no content handler: %s", route)
		}
	}
	for route := range pages {
		if !r.routes[route] {
			return nil, fmt.Errorf("handler lacks completion coverage: %s", route)
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		if q.URL.Path == "/sitemap.xml" {
			b, err := r.Sitemap()
			if err != nil {
				http.Error(w, "sitemap unavailable", 500)
				return
			}
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			_, _ = w.Write(b)
			return
		}
		if r.routes[q.URL.Path] {
			pages[q.URL.Path].ServeHTTP(w, q)
			return
		}
		target := r.aliasTarget(q.URL.Path)
		if r.routes[target] {
			http.Redirect(w, q, target, http.StatusMovedPermanently)
			return
		}
		http.NotFound(w, q)
	}), nil
}

type Advertising = tourpolicy.Advertising

const (
	AdsUnsupported Advertising = tourpolicy.AdsUnsupported
	GoLocal        Advertising = tourpolicy.AdsGoLocal
	Standard       Advertising = tourpolicy.AdsStandard
)

// AdPolicy projects the shared maintainer-frozen advertising authority into
// Site v2 route eligibility. It intentionally owns no locale set of its own.
type AdPolicy struct{}

func (AdPolicy) ForLocale(locale string) Advertising {
	return tourpolicy.AdvertisingForLocale(locale)
}
func (p AdPolicy) Enabled(locale, canonical string, covered bool) bool {
	if !covered || canonical == "/" || canonical == "/translation/" {
		return false
	}
	policy := p.ForLocale(locale)
	if policy == AdsUnsupported {
		return false
	}
	if strings.HasPrefix(canonical, "/tour/") {
		return policy == Standard
	}
	return canonical == "/learn/" || strings.HasPrefix(canonical, "/doc/tutorial/") || strings.HasPrefix(canonical, "/doc/database/") || strings.HasPrefix(canonical, "/doc/modules/") || strings.HasPrefix(canonical, "/doc/security/")
}
func (AdPolicy) ProductionPreflight() error {
	// The explicit unsupported set is now frozen in internal/tourpolicy and is
	// shared by Site v2 and the legacy Tour runtime.
	return nil
}
