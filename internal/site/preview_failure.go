package site

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/releasecontract"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// This reader is shared by every registered content package. Unsupported
// representations/checks fail closed; no route, locale or package allowlist.
func AttachPreviewFailureAuthority(c *i18n.Catalog) { c.PreviewFailureReader = verifyPreviewFailure }

var previewFailureContracts = []string{
	"go.mod", "go.sum", "scripts/verify-preview-browser.py", "scripts/browser_acceptance.py",
	"internal/site/browser_plan.go", "internal/site/content.go", "internal/site/render.go",
	"internal/site/shell_render.go", "internal/site/runtime.go", "internal/site/release_resources.go",
	"internal/site/preview_failure.go", "internal/sitecontent/render_input.go",
	"internal/i18n/preview_failure.go", "internal/i18n/quality_check_reopen.go",
	"internal/sitecontent/units.go", "internal/sitecontent/formal_page.go",
	"internal/sitecontent/translation_adapter.go", "internal/sitecontent/page_structure.go",
	"internal/sitecontent/linguistic.go", "cmd/tour-i18n/preview_failure.go",
	"internal/sitecontent/preview_failure.go",
	"docs/PREVIEW_TU_FAILURE.md",
}

func previewDigest(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func previewBase(raw string) (*url.URL, error) {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "http" || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost" && u.Hostname() != "::1") || u.Port() == "" || u.Path != "/" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("explicit loopback preview origin required")
	}
	return u, nil
}
func observePreview(base, route string) ([]byte, error) {
	u, err := previewBase(base)
	if err != nil {
		return nil, err
	}
	if !releasecontract.ValidRoute(route) {
		return nil, fmt.Errorf("invalid preview route")
	}
	u.Path = route
	client := http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	resp, err := client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("preview must return HTTP 200 without redirect, got %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20+1))
	if err != nil {
		return nil, err
	}
	if len(b) > 16<<20 {
		return nil, fmt.Errorf("preview response too large")
	}
	return b, nil
}

// Proof is about the TU's own content, independently of chrome/navigation.
// A runtime or composition loss with an intact TU cannot pass this check.
func missingTULinks(source, target []byte, format string) ([]string, error) {
	return sitecontent.MissingPageSourceLinks(source, target, format)
}

func previewArtifactAdapter(c *i18n.Catalog, p *i18n.PreviewTUFailure) (sitecontent.PageAdapter, error) {
	count := 0
	var adapter sitecontent.PageAdapter
	for _, d := range c.TranslationPackages {
		for _, u := range d.Units {
			if u.Unit.ID == p.Artifact.Unit.UnitID && u.Route == p.Route && d.ID == p.Package && d.AuthoritySHA256 == p.PackageSHA256 {
				a, ok := u.Adapter.(sitecontent.PageAdapter)
				if ok && a.Document.Route == p.Route && a.Document.Path == p.Artifact.Unit.SourcePath {
					adapter = a
					count++
				}
			}
		}
	}
	if count != 1 {
		return adapter, fmt.Errorf("preview failure requires exact registered Page/package/route mapping")
	}
	return adapter, nil
}

func previewCurrentContext(root string, p *i18n.PreviewTUFailure) (*Content, string, []i18n.GlossaryArchiveReference, error) {
	origin, err := sitecontent.LocaleOrigin(root, p.Locale)
	if err != nil {
		return nil, "", nil, err
	}
	c, err := LoadContent(root, p.Locale, origin)
	if err != nil {
		return nil, "", nil, err
	}
	refs := []i18n.GlossaryArchiveReference{}
	for _, path := range previewFailureContracts {
		r, e := sitecontent.ReferenceFile(root, path)
		if e != nil {
			return nil, "", nil, e
		}
		refs = append(refs, i18n.GlossaryArchiveReference{Path: r.Path, SHA256: r.SHA256})
	}
	context := struct {
		Content   *Content
		Origin    string
		Files     map[string][]byte
		Contracts []i18n.GlossaryArchiveReference
	}{c, origin, map[string][]byte{}, refs}
	for _, path := range []string{sitecontent.GlobalPath, sitecontent.LocalePath(p.Locale), "production/identity.json"} {
		b, e := sitecontent.ReadArtifact(root, path)
		if e != nil {
			return nil, "", nil, e
		}
		context.Files[path] = b
	}
	b, err := json.Marshal(context)
	return c, previewDigest(b), refs, err
}

func verifyPreviewFailure(root string, catalog *i18n.Catalog, p *i18n.PreviewTUFailure, current bool) error {
	return verifyPreviewFailureWithContext(root, catalog, p, current, previewCurrentContext)
}

func verifyPreviewFailureWithContext(root string, catalog *i18n.Catalog, p *i18n.PreviewTUFailure, current bool, load func(string, *i18n.PreviewTUFailure) (*Content, string, []i18n.GlossaryArchiveReference, error)) error {
	if _, err := previewBase(p.PreviewURL); err != nil {
		return err
	}
	if !releasecontract.ValidRoute(p.Route) {
		return fmt.Errorf("invalid preview route")
	}
	a, err := previewArtifactAdapter(catalog, p)
	if err != nil {
		return err
	}
	target, err := sitecontent.ReadArtifact(root, p.Artifact.Unit.CandidatePath)
	if err != nil {
		return err
	}
	if err := a.VerifyFrozenPreviewTUFailure(target, p); err != nil {
		return err
	}
	missing := p.MissingLinks
	if !current {
		return nil
	}
	c, identity, refs, err := load(root, p)
	if err != nil {
		return err
	}
	if identity != p.ContextSHA256 || !reflect.DeepEqual(refs, p.Contracts) {
		return fmt.Errorf("preview/build/verifier context is stale")
	}
	page, ok := c.Pages[p.Route]
	if !ok || page.Path != a.Document.Path || !bytes.Equal(bytes.TrimRight(page.Content, "\n"), bytes.TrimRight(target, "\n")) {
		return fmt.Errorf("preview page is not the exact finalized TU projection")
	}
	body, err := c.renderPage(p.Locale, page)
	if err != nil {
		return err
	}
	body, err = DeliveryResources(p.Route, body, true)
	if err != nil {
		return err
	}
	if string(body) != p.ResponseHTML {
		return fmt.Errorf("preview/build response differs from current rendered candidate")
	}
	sourcePage := page
	sourcePage.Content = a.Source
	sourceHTML, err := c.renderPage(p.Locale, sourcePage)
	if err != nil {
		return err
	}
	sourceHTML, err = DeliveryResources(p.Route, sourceHTML, true)
	if err != nil {
		return err
	}
	expected, err := sitecontent.RenderedExternalLinks(sourceHTML)
	if err != nil {
		return err
	}
	for _, href := range missing {
		resolved, e := c.Resolver.ResolveLink(p.Route, href)
		if e != nil || resolved != href || !expected[href] {
			return fmt.Errorf("link proof requires unchanged external source-bound href")
		}
	}
	live, err := observePreview(p.PreviewURL, p.Route)
	if err != nil {
		return err
	}
	if string(live) != p.ResponseHTML {
		return fmt.Errorf("live preview observation is stale")
	}
	return nil
}

// Capture re-observes one exact machine check; it is not a full browser rerun.
// It neither changes Surface/QC evidence nor supplies a language judgment.
func CapturePreviewTUFailure(root string, c *i18n.Catalog, locale, snapshot, unitID, route, base, id string) (string, error) {
	p, err := i18n.PreparePreviewTUFailure(root, c, locale, snapshot, unitID)
	if err != nil {
		return "", err
	}
	p.Route, p.PreviewURL, p.FailureID = route, base, id
	for _, d := range c.TranslationPackages {
		for _, u := range d.Units {
			if u.Unit.ID == unitID && u.Route == route {
				p.Package, p.PackageSHA256 = d.ID, d.AuthoritySHA256
			}
		}
	}
	a, err := previewArtifactAdapter(c, p)
	if err != nil {
		return "", err
	}
	target, err := sitecontent.ReadArtifact(root, p.Artifact.Unit.CandidatePath)
	if err != nil {
		return "", err
	}
	p.MissingLinks, err = missingTULinks(a.Source, target, strings.TrimPrefix(path.Ext(a.Document.Path), "."))
	if err != nil {
		return "", err
	}
	_, p.ContextSHA256, p.Contracts, err = previewCurrentContext(root, p)
	if err != nil {
		return "", err
	}
	body, err := observePreview(base, route)
	if err != nil {
		return "", err
	}
	p.ResponseHTML, p.ResponseSHA256 = string(body), previewDigest(body)
	return i18n.RecordPreviewTUFailure(root, c, *p)
}
