package tour

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
	"golang.org/x/net/html"
)

// The caller supplies its validated production/site identity, never the
// request's Host. Languages remain exclusively owned by languageRegistry.
type SharedChromeModel struct {
	Locale, Home, About, AboutPath, LanguageLabel, CurrentLabel, CurrentAutonym string
	Unofficial, HomeLabel, GitHubLabel, GitHub, Copyright, ICP, ICPURL          string
	Languages                                                                   []LanguageLink
}

const ProjectCopyrightYear = 2026

func NewSharedChromeModel(locale, origin string) (SharedChromeModel, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" && u.Path != "/" || u.RawQuery != "" || u.Fragment != "" {
		return SharedChromeModel{}, fmt.Errorf("invalid shared chrome production origin")
	}
	catalog, err := ui.Load(locale)
	if err != nil {
		return SharedChromeModel{}, err
	}
	languages, err := languagesFor(locale)
	if err != nil {
		return SharedChromeModel{}, err
	}
	current, err := currentLanguage(languages)
	if err != nil {
		return SharedChromeModel{}, err
	}
	m := SharedChromeModel{Locale: locale, Home: strings.TrimSuffix(origin, "/") + "/", Languages: languages,
		CurrentAutonym: current.Autonym, GitHub: Project.GitHubURL, AboutPath: "/translation/",
		Copyright: fmt.Sprintf("© %d %s", ProjectCopyrightYear, Project.CopyrightHolder)}
	for key, target := range map[string]*string{"header.about_project": &m.About, "site.language_versions": &m.LanguageLabel,
		"site.current_language": &m.CurrentLabel, "footer.unofficial": &m.Unofficial, "footer.home": &m.HomeLabel, "footer.github": &m.GitHubLabel} {
		*target, err = catalog.Plain(key)
		if err != nil {
			return SharedChromeModel{}, err
		}
	}
	// Site legal policy, independent of language labels and navigation registry.
	if locale == "zh-CN" {
		m.ICP, m.ICPURL = Project.ICPNumber, Project.ICPURL
	}
	return m, nil
}

//go:embed shared_chrome.tmpl
var sharedChromeTemplate string

//go:embed shared_chrome.css
var sharedChromeCSS string

//go:embed shared_chrome.js
var sharedChromeJS string

func chromeClass(n *html.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, c := range strings.Fields(a.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}
func chromeFind(n *html.Node, match func(*html.Node) bool) *html.Node {
	if match(n) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if got := chromeFind(c, match); got != nil {
			return got
		}
	}
	return nil
}
func chromeByClass(n *html.Node, class string) *html.Node {
	return chromeFind(n, func(n *html.Node) bool { return chromeClass(n, class) })
}
func chromeAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// ApplySharedChrome is the single semantic implementation for both Site and
// Tour delivery. Frozen renderers/templates and article payloads stay intact.
func ApplySharedChrome(raw []byte, locale, origin string) ([]byte, error) {
	m, err := NewSharedChromeModel(locale, origin)
	if err != nil {
		return nil, err
	}
	doc, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	// A sealed Site page already carries this exact delivery component.
	if chromeFind(doc, func(n *html.Node) bool { return chromeAttr(n, "id") == "project-chrome-style" }) != nil {
		return nil, fmt.Errorf("shared chrome applied twice")
	}
	header := chromeByClass(doc, "Site-header")
	site := header != nil
	if header == nil {
		header = chromeByClass(doc, "top-bar")
	}
	footer := chromeByClass(doc, "Site-footer")
	if footer == nil {
		footer = chromeByClass(doc, "site-footer")
	}
	head := chromeFind(doc, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "head" })
	body := chromeFind(doc, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "body" })
	if header == nil || footer == nil || head == nil || body == nil {
		return nil, fmt.Errorf("shared chrome requires document shell")
	}
	old := chromeByClass(header, "header-project-nav")
	if old != nil {
		old.Parent.RemoveChild(old)
	}
	desktop := chromeByClass(header, "Header-rightContent")
	if desktop == nil {
		desktop = chromeByClass(header, "right")
	}
	if desktop == nil {
		desktop = header
	}
	tmpl, err := template.New("chrome").Parse(sharedChromeTemplate)
	if err != nil {
		return nil, err
	}
	part := func(name string) (*html.Node, error) {
		var b bytes.Buffer
		if e := tmpl.ExecuteTemplate(&b, name, m); e != nil {
			return nil, e
		}
		d, e := html.Parse(&b)
		if e != nil {
			return nil, e
		}
		container := chromeFind(d, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "body" })
		n := container.FirstChild
		container.RemoveChild(n)
		return n, nil
	}
	nav, err := part("navigation")
	if err != nil {
		return nil, err
	}
	host := "tour"
	if site {
		host = "site"
	}
	nav.Attr = append(nav.Attr, html.Attribute{Key: "data-project-host", Val: host})
	if !site {
		for _, node := range []*html.Node{chromeFind(nav, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "a" }), chromeFind(nav, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "summary" })} {
			if node != nil && node.Type == html.ElementNode {
				node.Attr = append(node.Attr, html.Attribute{Key: "class", Val: "header-control"})
			}
		}
	}
	desktop.AppendChild(nav)
	projectFooter, err := part("footer")
	if err != nil {
		return nil, err
	}
	projectFooter.Attr = append(projectFooter.Attr, html.Attribute{Key: "data-project-host", Val: host})
	if site {
		bottom := chromeByClass(footer, "Footer-bottom")
		primary := chromeByClass(footer, "Footer-links")
		if bottom == nil || primary == nil {
			return nil, fmt.Errorf("upstream footer structure missing")
		}
		before := bottom
		for before.Parent != footer {
			before = before.Parent
			if before == nil {
				return nil, fmt.Errorf("upstream footer boundary missing")
			}
		}
		wrapper := &html.Node{Type: html.ElementNode, Data: "div", Attr: []html.Attribute{{Key: "class", Val: "Footer"}}}
		container := &html.Node{Type: html.ElementNode, Data: "div", Attr: []html.Attribute{{Key: "class", Val: "Container Container--padded"}}}
		wrapper.AppendChild(container)
		container.AppendChild(projectFooter)
		footer.InsertBefore(wrapper, before)
	} else {
		for footer.FirstChild != nil {
			footer.RemoveChild(footer.FirstChild)
		}
		footer.AppendChild(projectFooter)
	}
	style := &html.Node{Type: html.ElementNode, Data: "style", Attr: []html.Attribute{{Key: "id", Val: "project-chrome-style"}}}
	style.AppendChild(&html.Node{Type: html.TextNode, Data: sharedChromeCSS})
	head.AppendChild(style)
	script := &html.Node{Type: html.ElementNode, Data: "script", Attr: []html.Attribute{{Key: "id", Val: "project-chrome-runtime"}}}
	script.AppendChild(&html.Node{Type: html.TextNode, Data: sharedChromeJS})
	body.AppendChild(script)
	var out bytes.Buffer
	err = html.Render(&out, doc)
	return out.Bytes(), err
}

type chromeResponse struct {
	body    bytes.Buffer
	status  int
	headers http.Header
}

func (w *chromeResponse) Header() http.Header { return w.headers }
func (w *chromeResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}
func (w *chromeResponse) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	return w.body.Write(b)
}

// WithSharedChrome adapts canonical Tour HTML only; transports and resources
// retain the original handler, status and bytes.
func WithSharedChrome(h http.Handler, locale, origin string) http.Handler {
	if h == nil {
		return nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		r := &chromeResponse{headers: make(http.Header)}
		h.ServeHTTP(r, q)
		b := r.body.Bytes()
		kind := r.Header().Get("Content-Type")
		if kind == "" {
			kind = http.DetectContentType(b)
		}
		if r.status == 200 && strings.HasPrefix(kind, "text/html") {
			var err error
			b, err = ApplySharedChrome(b, locale, origin)
			if err != nil {
				http.Error(w, "shared chrome failed", 500)
				return
			}
			r.Header().Del("Content-Length")
			r.Header().Del("ETag")
			r.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		for k, vs := range r.Header() {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		if r.status == 0 {
			r.status = 200
		}
		w.WriteHeader(r.status)
		_, _ = w.Write(b)
	})
}
