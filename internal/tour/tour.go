// Copyright 2013 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tour

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/shuijingwan/go-tour-i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
	"golang.org/x/tools/present"
)

var (
	uiContent      []byte
	homeContent    []byte
	footerContent  []byte
	lessons        = make(map[string][]byte)
	lessonNotFound = fmt.Errorf("lesson not found")
)

var contentTour = website.TourOnly()

// useContent selects the content tree used by subsequent Tour initialization.
// A Tour process initializes one content tree and locale; runtime switching is
// deliberately unsupported.
func useContent(content fs.FS) error {
	if content == nil {
		return fmt.Errorf("Tour content is required")
	}
	if _, err := fs.Stat(content, "tour/template/index.tmpl"); err != nil {
		return fmt.Errorf("invalid Tour content: %w", err)
	}
	contentTour = content
	uiContent = nil
	homeContent = nil
	footerContent = nil
	lessons = make(map[string][]byte)
	sitemapContent = nil
	return nil
}

// initTour loads tour.article and the relevant HTML templates from root.
func initTour(mux *http.ServeMux, transport, locale, playgroundBaseURL string) error {
	// Make sure playground is enabled before rendering.
	present.PlayEnabled = true

	// Set up templates.
	tmpl, err := present.Template().ParseFS(contentTour, "tour/template/action.tmpl")
	if err != nil {
		return fmt.Errorf("parse templates: %v", err)
	}

	// Init lessons.
	if err := initLessons(tmpl); err != nil {
		return fmt.Errorf("init lessons: %v", err)
	}
	if err := initSEO(); err != nil {
		return fmt.Errorf("init SEO documents: %v", err)
	}

	// Load the build-selected UI locale once during initialization.
	catalog, err := ui.Load(locale)
	if err != nil {
		return fmt.Errorf("load UI catalog %q: %w", locale, err)
	}
	metadata, err := loadSiteMetadata(contentTour)
	if err != nil {
		return err
	}
	uiContent, err = renderIndex(catalog, metadata)
	if err != nil {
		return err
	}
	homeContent, err = renderHome(catalog, metadata)
	if err != nil {
		return err
	}
	footerContent, err = renderFooter(catalog, metadata)
	if err != nil {
		return err
	}

	mux.HandleFunc("/tour/", rootHandler)
	mux.HandleFunc("/tour/lesson/", lessonHandler)
	mux.HandleFunc("/tour/footer.html", footerHandler)
	mux.Handle("/tour/static/", http.FileServer(http.FS(contentTour)))

	return initScript(mux, socketAddr(), transport, playgroundBaseURL, catalog)
}

type pageTemplateData struct {
	HTMLLang            string
	Metadata            SiteMetadata
	Development         bool
	PublishedAt         string
	UpstreamCommitTime  string
	ShortUpstreamCommit string
	UpstreamCommitURL   string
	SiteURL             string
	OfficialTourURL     string
	GitHubURL           string
	GitHubIssuesURL     string
	DevelopmentLogURL   string
	UpstreamURL         string
	ICPURL              string
	ICPNumber           string
	CopyrightHolder     string
}

func newPageTemplateData(catalog ui.Catalog, metadata SiteMetadata) (pageTemplateData, error) {
	publishedAt := ""
	if !metadata.Development {
		var err error
		publishedAt, err = metadata.PublishedAtBeijing()
		if err != nil {
			return pageTemplateData{}, err
		}
	}
	return pageTemplateData{
		HTMLLang:            catalog.HTMLLang,
		Metadata:            metadata,
		Development:         metadata.Development,
		PublishedAt:         publishedAt,
		UpstreamCommitTime:  "2026-07-23 04:05:40（北京时间）",
		ShortUpstreamCommit: metadata.UpstreamCommit[:8],
		UpstreamCommitURL:   Project.UpstreamURL + "/commit/" + metadata.UpstreamCommit,
		SiteURL:             Project.SiteURL,
		OfficialTourURL:     Project.OfficialTourURL,
		GitHubURL:           Project.GitHubURL,
		GitHubIssuesURL:     Project.GitHubIssuesURL,
		DevelopmentLogURL:   Project.DevelopmentLogURL,
		UpstreamURL:         Project.UpstreamURL,
		ICPURL:              Project.ICPURL,
		ICPNumber:           Project.ICPNumber,
		CopyrightHolder:     Project.CopyrightHolder,
	}, nil
}

func renderIndex(catalog ui.Catalog, metadata SiteMetadata) ([]byte, error) {
	tmpl, err := template.New("index.tmpl").Funcs(template.FuncMap{"ui": catalog.Plain}).ParseFS(contentTour, "tour/template/index.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse index.tmpl: %w", err)
	}
	buf := new(bytes.Buffer)
	data, err := newPageTemplateData(catalog, metadata)
	if err != nil {
		return nil, err
	}
	dataWithHeadHTML := struct {
		pageTemplateData
		AnalyticsHTML template.HTML
		AdSenseHTML   template.HTML
	}{data, analyticsHTML, adsenseHTML}
	if err := tmpl.Execute(buf, dataWithHeadHTML); err != nil {
		return nil, fmt.Errorf("render index.tmpl: %w", err)
	}
	return buf.Bytes(), nil
}

func renderHome(catalog ui.Catalog, metadata SiteMetadata) ([]byte, error) {
	tmpl, err := template.New("home.tmpl").Funcs(template.FuncMap{"ui": catalog.Plain}).ParseFS(contentTour, "tour/template/index.tmpl", "tour/template/home.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse home.tmpl: %w", err)
	}
	data, err := newPageTemplateData(catalog, metadata)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	dataWithHeadHTML := struct {
		pageTemplateData
		AnalyticsHTML template.HTML
		AdSenseHTML   template.HTML
	}{data, analyticsHTML, adsenseHTML}
	if err := tmpl.ExecuteTemplate(buf, "home.tmpl", dataWithHeadHTML); err != nil {
		return nil, fmt.Errorf("render home.tmpl: %w", err)
	}
	return buf.Bytes(), nil
}

func renderFooter(catalog ui.Catalog, metadata SiteMetadata) ([]byte, error) {
	tmpl, err := template.New("index.tmpl").Funcs(template.FuncMap{"ui": catalog.Plain}).ParseFS(contentTour, "tour/template/index.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse index.tmpl: %w", err)
	}
	data, err := newPageTemplateData(catalog, metadata)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(buf, "footer", data); err != nil {
		return nil, fmt.Errorf("render footer: %w", err)
	}
	return buf.Bytes(), nil
}

// initLessons finds all the lessons in the content directory, renders them,
// using the given template and saves the content in the lessons map.
func initLessons(tmpl *template.Template) error {
	files, err := fs.ReadDir(contentTour, "tour")
	if err != nil {
		return err
	}
	for _, f := range files {
		if path.Ext(f.Name()) != ".article" {
			continue
		}
		content, err := parseLesson(f.Name(), tmpl)
		if err != nil {
			return fmt.Errorf("parsing %v: %v", f.Name(), err)
		}
		name := strings.TrimSuffix(f.Name(), ".article")
		lessons[name] = content
	}
	return nil
}

// file defines the JSON form of a code file in a page.
type file struct {
	Name    string
	Content string
	Hash    string
}

// page defines the JSON form of a tour lesson page.
type page struct {
	Title   string
	Content string
	Files   []file
}

// lesson defines the JSON form of a tour lesson.
type lesson struct {
	Title       string
	Description string
	Pages       []page
}

// parseLesson parses and returns a lesson content given its path
// relative to root ('/'-separated) and the template to render it.
func parseLesson(path string, tmpl *template.Template) ([]byte, error) {
	f, err := contentTour.Open("tour/" + path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	ctx := &present.Context{
		ReadFile: func(filename string) ([]byte, error) {
			return fs.ReadFile(contentTour, "tour/"+filepath.ToSlash(filename))
		},
	}
	doc, err := ctx.Parse(prepContent(f), path, 0)
	if err != nil {
		return nil, err
	}

	lesson := lesson{
		doc.Title,
		doc.Subtitle,
		make([]page, len(doc.Sections)),
	}

	for i, sec := range doc.Sections {
		p := &lesson.Pages[i]
		w := new(bytes.Buffer)
		if err := sec.Render(w, tmpl); err != nil {
			return nil, fmt.Errorf("render section: %v", err)
		}
		p.Title = sec.Title
		p.Content = w.String()
		codes := findPlayCode(sec)
		p.Files = make([]file, len(codes))
		for i, c := range codes {
			f := &p.Files[i]
			f.Name = c.FileName
			f.Content = string(c.Raw)
			hash := sha1.Sum(c.Raw)
			f.Hash = base64.StdEncoding.EncodeToString(hash[:])
		}
	}

	w := new(bytes.Buffer)
	if err := json.NewEncoder(w).Encode(lesson); err != nil {
		return nil, fmt.Errorf("encode lesson: %v", err)
	}
	return w.Bytes(), nil
}

// findPlayCode returns a slide with all the Code elements in the given
// Elem with Play set to true.
func findPlayCode(e present.Elem) []*present.Code {
	var r []*present.Code
	switch v := e.(type) {
	case present.Code:
		if v.Play {
			r = append(r, &v)
		}
	case present.Section:
		for _, s := range v.Elem {
			r = append(r, findPlayCode(s)...)
		}
	}
	return r
}

// writeLesson writes the tour content to the provided Writer.
func writeLesson(name string, w io.Writer) error {
	if uiContent == nil {
		panic("writeLesson called before successful initTour")
	}
	if len(name) == 0 {
		return writeAllLessons(w)
	}
	l, ok := lessons[name]
	if !ok {
		return lessonNotFound
	}
	_, err := w.Write(l)
	return err
}

func writeAllLessons(w io.Writer) error {
	if _, err := fmt.Fprint(w, "{"); err != nil {
		return err
	}
	nLessons := len(lessons)
	for k, v := range lessons {
		if _, err := fmt.Fprintf(w, "%q:%s", k, v); err != nil {
			return err
		}
		nLessons--
		if nLessons != 0 {
			if _, err := fmt.Fprint(w, ","); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprint(w, "}")
	return err
}

// renderUI writes the tour UI to the provided Writer.
func renderUI(w io.Writer) error {
	if uiContent == nil {
		panic("renderUI called before successful initTour")
	}
	_, err := w.Write(uiContent)
	return err
}

func renderHomePage(w io.Writer) error {
	if homeContent == nil {
		panic("renderHomePage called before successful initTour")
	}
	_, err := w.Write(homeContent)
	return err
}

func footerHandler(w http.ResponseWriter, r *http.Request) {
	if footerContent == nil {
		panic("footerHandler called before successful initTour")
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(footerContent)
}

// initScript concatenates all the javascript files needed to render
// the tour UI and serves the result on /script.js.
func initScript(mux *http.ServeMux, socketAddr, transport, playgroundBaseURL string, catalog ui.Catalog) error {
	modTime := time.Now()
	b := new(bytes.Buffer)
	bootstrap, err := jsBootstrap(catalog)
	if err != nil {
		return err
	}
	b.Write(bootstrap)

	// Keep this list in dependency order
	files := []string{
		"../js/playground.js",
		"static/lib/jquery.min.js",
		"static/lib/jquery-ui.min.js",
		"static/lib/angular.min.js",
		"static/lib/codemirror/lib/codemirror.js",
		"static/lib/codemirror/mode/go/go.js",
		"static/lib/angular-ui.min.js",
		"static/js/app.js",
		"static/js/controllers.js",
		"static/js/directives.js",
		"static/js/services.js",
		"static/js/values.js",
	}

	for _, file := range files {
		f, err := fs.ReadFile(contentTour, path.Clean("tour/"+file))
		if err != nil {
			return err
		}
		b.Write(f)
	}

	f, err := fs.ReadFile(contentTour, "tour/static/js/page.js")
	if err != nil {
		return err
	}
	s := string(f)
	s = strings.ReplaceAll(s, "{{.SocketAddr}}", socketAddr)
	s = strings.ReplaceAll(s, "{{.Transport}}", transport)
	playgroundBaseURLJSON, err := json.Marshal(playgroundBaseURL)
	if err != nil {
		return fmt.Errorf("encode Playground base URL: %w", err)
	}
	s = strings.ReplaceAll(s, "{{.PlaygroundBaseURL}}", string(playgroundBaseURLJSON))
	b.WriteString(s)

	mux.HandleFunc("/tour/script.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "application/javascript")
		// Set expiration time in one week.
		w.Header().Set("Cache-control", "max-age=604800")
		http.ServeContent(w, r, "", modTime, bytes.NewReader(b.Bytes()))
	})

	return nil
}

var jsI18nKeys = []string{
	"tour.list_heading",
	"toc.title",
	"execution.waiting",
	"execution.exited",
	"feedback.open",
	"feedback.issue_title",
	"feedback.issue_body",
	"feedback.context",
	"editor.syntax",
	"editor.imports",
	"editor.run",
	"editor.kill",
	"editor.format",
	"editor.reset",
}

type jsModule struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

var jsModules = []struct {
	ID             string
	TitleKey       string
	DescriptionKey string
}{
	{"mechanics", "module.using_tour.title", "module.using_tour.description"},
	{"basics", "module.basics.title", "module.basics.description"},
	{"methods", "module.methods.title", "module.methods.description"},
	{"generics", "module.generics.title", "module.generics.description"},
	{"concurrency", "module.concurrency.title", "module.concurrency.description"},
}

func jsBootstrap(catalog ui.Catalog) ([]byte, error) {
	i18n, err := jsI18nBootstrap(catalog)
	if err != nil {
		return nil, err
	}
	modules, err := jsModuleBootstrap(catalog)
	if err != nil {
		return nil, err
	}
	return append(i18n, modules...), nil
}

func jsI18nBootstrap(catalog ui.Catalog) ([]byte, error) {
	messages := make(map[string]string, len(jsI18nKeys))
	for _, key := range jsI18nKeys {
		message, err := catalog.Plain(key)
		if err != nil {
			return nil, fmt.Errorf("read JavaScript UI message %q: %w", key, err)
		}
		messages[key] = message
	}
	encoded, err := json.Marshal(messages)
	if err != nil {
		return nil, fmt.Errorf("encode JavaScript UI messages: %w", err)
	}
	return append(append([]byte("window.__tourUIMessages = "), encoded...), ";\n"...), nil
}

func jsModuleBootstrap(catalog ui.Catalog) ([]byte, error) {
	modules := make(map[string]jsModule, len(jsModules))
	for _, module := range jsModules {
		title, err := catalog.Plain(module.TitleKey)
		if err != nil {
			return nil, fmt.Errorf("read module title %q: %w", module.TitleKey, err)
		}
		description, err := catalog.Rich(module.DescriptionKey)
		if err != nil {
			return nil, fmt.Errorf("read module description %q: %w", module.DescriptionKey, err)
		}
		modules[module.ID] = jsModule{Title: title, Description: description}
	}
	encoded, err := json.Marshal(modules)
	if err != nil {
		return nil, fmt.Errorf("encode localized Tour modules: %w", err)
	}
	return append(append([]byte("window.__tourModules = "), encoded...), ";\n"...), nil
}
