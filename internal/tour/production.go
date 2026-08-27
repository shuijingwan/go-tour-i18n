// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tour

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	productionPlaygroundBaseURL = "https://play.go-dev.shuijingwanwq.com:8443"
	productionPlaygroundURL     = "https://go.dev/_"
	playgroundRequestLimit      = 1 << 20 // 1 MiB
	playgroundResponseLimit     = 4 << 20 // 4 MiB
	playgroundTimeout           = 20 * time.Second
)

// NewProductionHandler creates the public Tour handler for one content tree
// and one build-selected locale. It uses the remote Playground HTTP protocol
// and never registers the local WebSocket execution handler.
func NewProductionHandler(content fs.FS, locale string) (http.Handler, error) {
	proxy, err := newPlaygroundProxy(productionPlaygroundURL)
	if err != nil {
		return nil, err
	}
	handler, documents, err := newTourHandler(content, locale, proxy, true, true)
	if err != nil {
		return nil, err
	}
	if !documents.courseMetadataComplete {
		return nil, fmt.Errorf("formal projected course SEO is missing")
	}
	return handler, nil
}

// NewPreviewHandler creates the complete-locale preview handler used for
// rendered surface acceptance. It uses HTTPTransport and public SEO identity,
// while omitting prerendered pages, analytics, and advertising runtime
// configuration. Its browser Playground requests stay same-origin so this
// handler can proxy them to the real Playground without requiring localhost in
// the production proxy's Origin allowlist.
func NewPreviewHandler(content fs.FS, locale string) (http.Handler, error) {
	proxy, err := newPlaygroundProxy(productionPlaygroundURL)
	if err != nil {
		return nil, err
	}
	handler, documents, err := newTourHandlerWithPlaygroundBase(content, locale, proxy, "", false, false)
	if err != nil {
		return nil, err
	}
	if !documents.courseMetadataComplete {
		return nil, fmt.Errorf("formal projected course SEO is missing")
	}
	return handler, nil
}

// PrerenderSource is the private build server and its canonical course routes.
// It deliberately omits runtime analytics and advertising configuration so a
// publish cannot freeze third-party DOM into the release artifact.
type PrerenderSource struct {
	Handler http.Handler
	Routes  []CourseRoute
}

// NewPrerenderSource creates the local-only source server used by publish.
func NewPrerenderSource(content fs.FS, locale string) (*PrerenderSource, error) {
	proxy, err := newPlaygroundProxy(productionPlaygroundURL)
	if err != nil {
		return nil, err
	}
	handler, documents, err := newTourHandler(content, locale, proxy, false, false)
	if err != nil {
		return nil, err
	}
	routes := append([]CourseRoute(nil), documents.courseRoutes...)
	return &PrerenderSource{Handler: handler, Routes: routes}, nil
}

func newProductionHandler(content fs.FS, locale string, proxy *playgroundProxy) (http.Handler, error) {
	handler, _, err := newTourHandler(content, locale, proxy, true, true)
	return handler, err
}

func newTourHandler(content fs.FS, locale string, proxy *playgroundProxy, requirePrerender, includeRuntimeHead bool) (http.Handler, seoDocuments, error) {
	return newTourHandlerWithPlaygroundBase(content, locale, proxy, productionPlaygroundBaseURL, requirePrerender, includeRuntimeHead)
}

func newTourHandlerWithPlaygroundBase(content fs.FS, locale string, proxy *playgroundProxy, playgroundBaseURL string, requirePrerender, includeRuntimeHead bool) (http.Handler, seoDocuments, error) {
	if proxy == nil {
		return nil, seoDocuments{}, fmt.Errorf("Playground proxy is required")
	}
	if err := useContent(content); err != nil {
		return nil, seoDocuments{}, err
	}

	mux := http.NewServeMux()
	documents, err := registerHandlersLocaleWithRuntimeHead(mux, locale, playgroundBaseURL, includeRuntimeHead)
	if err != nil {
		return nil, seoDocuments{}, err
	}
	metadata, err := loadSiteMetadata(contentTour)
	if err != nil {
		return nil, seoDocuments{}, err
	}
	if requirePrerender && !metadata.Development {
		pages, err := loadPrerenderedPages(contentTour, documents.courseRoutes)
		if err != nil {
			return nil, seoDocuments{}, err
		}
		registerPrerenderedPages(mux, pages)
	}

	// These routes are deliberately registered before the SPA fallback. The
	// production process never exposes the local code-execution WebSocket.
	mux.HandleFunc("/socket", notFound)
	mux.HandleFunc("/socket/", notFound)
	mux.HandleFunc("/_/compile", proxy.compile)
	mux.HandleFunc("/_/fmt", proxy.format)
	mux.HandleFunc("/_/share", notFound)
	mux.HandleFunc("/_/", notFound)

	contentServer := http.FileServer(http.FS(contentTour))
	mux.Handle("/robots.txt", robotsHandler(documents))
	mux.Handle("/sitemap.xml", sitemapHandler(documents))
	mux.Handle("/favicon.ico", contentServer)
	mux.Handle("/images/", contentServer)
	mux.HandleFunc("/", rootHandler)
	return mux, documents, nil
}

func notFound(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

// The structures and browser-facing conversion below follow the frozen
// golang/website internal/play proxy protocol at
// 645042eb697eaf69e33a9af00c6b5b3fffdead5a. Execution remains at the remote
// Playground; this package only performs bounded HTTP conversion.
type playgroundCompileResponse struct {
	Errors      string
	Events      []playgroundEvent
	VetErrors   string
	IsTest      bool
	TestsFailed int
}

type playgroundEvent struct {
	Message string
	Kind    string
	Delay   time.Duration
}

type playgroundProxy struct {
	compileURL string
	formatURL  string
	client     *http.Client
	logger     *log.Logger
}

var requestIDFallback atomic.Uint64

func newPlaygroundProxy(baseURL string) (*playgroundProxy, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse Playground URL: %w", err)
	}
	if base.Scheme != "http" && base.Scheme != "https" || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("invalid Playground URL %q", baseURL)
	}
	base.Path = strings.TrimRight(base.Path, "/")
	return &playgroundProxy{
		compileURL: base.String() + "/compile",
		formatURL:  base.String() + "/fmt",
		client: &http.Client{
			Timeout: playgroundTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		logger: log.Default(),
	}, nil
}

func (p *playgroundProxy) compile(w http.ResponseWriter, r *http.Request) {
	requestID := newRequestID()
	w.Header().Set("X-Request-ID", requestID)
	if !requireFormPost(w, r) {
		return
	}

	form := url.Values{
		"version": {r.PostFormValue("version")},
		"body":    {r.PostFormValue("body")},
		"withVet": {r.PostFormValue("withVet")},
	}
	upstream, err := http.NewRequestWithContext(r.Context(), http.MethodPost, p.compileURL, strings.NewReader(form.Encode()))
	if err != nil {
		http.Error(w, "create Playground request", http.StatusInternalServerError)
		return
	}
	upstream.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	started := time.Now()
	status, _, body, err := p.do(upstream)
	if err != nil || status != http.StatusOK {
		p.logFailure(requestID, "compile", started, status, err)
		http.Error(w, "Playground compile unavailable", http.StatusBadGateway)
		return
	}
	var result playgroundCompileResponse
	if err := json.Unmarshal(body, &result); err != nil {
		p.logFailure(requestID, "compile", started, status, fmt.Errorf("invalid Playground compile response: %w", err))
		http.Error(w, "invalid Playground compile response", http.StatusBadGateway)
		return
	}

	var browserResponse any = result
	if r.PostFormValue("version") != "2" {
		browserResponse = struct {
			CompileErrors string `json:"compile_errors"`
			Output        string `json:"output"`
		}{result.Errors, flattenEvents(result.Events)}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if err := json.NewEncoder(w).Encode(browserResponse); err != nil {
		return
	}
}

func (p *playgroundProxy) format(w http.ResponseWriter, r *http.Request) {
	requestID := newRequestID()
	w.Header().Set("X-Request-ID", requestID)
	if !requireFormPost(w, r) {
		return
	}
	form := url.Values{
		"body":    {r.PostFormValue("body")},
		"imports": {r.PostFormValue("imports")},
	}
	upstream, err := http.NewRequestWithContext(r.Context(), http.MethodPost, p.formatURL, strings.NewReader(form.Encode()))
	if err != nil {
		http.Error(w, "create Playground request", http.StatusInternalServerError)
		return
	}
	upstream.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	started := time.Now()
	status, contentType, body, err := p.do(upstream)
	if err != nil || status < 200 || status >= 300 {
		p.logFailure(requestID, "format", started, status, err)
		http.Error(w, "Playground format unavailable", http.StatusBadGateway)
		return
	}
	if contentType == "" {
		contentType = "application/json"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func requireFormPost(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/x-www-form-urlencoded" {
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, playgroundRequestLimit)
	if err := r.ParseForm(); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, "invalid form request", http.StatusBadRequest)
		}
		return false
	}
	return true
}

func (p *playgroundProxy) do(req *http.Request) (int, string, []byte, error) {
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, "", nil, err
	}
	defer resp.Body.Close()
	body, err := readLimited(resp.Body, playgroundResponseLimit)
	if err != nil {
		return resp.StatusCode, resp.Header.Get("Content-Type"), nil, err
	}
	return resp.StatusCode, resp.Header.Get("Content-Type"), body, nil
}

func (p *playgroundProxy) logFailure(requestID, operation string, started time.Time, status int, err error) {
	if p.logger == nil {
		return
	}
	if err != nil {
		if status != 0 {
			p.logger.Printf("playground request_id=%s operation=%s duration=%s upstream_status=%d error=%q", requestID, operation, time.Since(started), status, err)
			return
		}
		p.logger.Printf("playground request_id=%s operation=%s duration=%s error=%q", requestID, operation, time.Since(started), err)
		return
	}
	p.logger.Printf("playground request_id=%s operation=%s duration=%s upstream_status=%d", requestID, operation, time.Since(started), status)
}

func newRequestID() string {
	var raw [8]byte
	if _, err := cryptorand.Read(raw[:]); err == nil {
		return hex.EncodeToString(raw[:])
	}
	sequence := requestIDFallback.Add(1)
	return strconv.FormatInt(time.Now().UnixNano(), 16) + "-" + strconv.FormatUint(sequence, 16)
}

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("Playground response exceeds %d bytes", limit)
	}
	return body, nil
}

func flattenEvents(events []playgroundEvent) string {
	var out strings.Builder
	for _, event := range events {
		out.WriteString(event.Message)
	}
	return out.String()
}
