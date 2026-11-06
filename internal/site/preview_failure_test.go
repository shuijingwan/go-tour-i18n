package site

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func TestPreviewFailureTUProofUsesRenderedLinks(t *testing.T) {
	const href = "https://example.invalid/certificate?a=1&b=2"
	for _, tc := range []struct {
		name, source, target, format string
		want                         bool
	}{
		{"used-collapsed-reference", "[certificate][]\n\n[certificate]: " + href, "[证书][]\n\n[certificate]: " + href, "md", true},
		{"used-full-reference", "[certificate][id]\n\n[id]: " + href, "plain prose\n\n[id]: " + href, "md", true},
		{"unused-reference", "ordinary prose\n\n[certificate]: " + href, "ordinary prose", "md", false},
		{"intact-used-link-runtime-loss", "[certificate][]\n\n[certificate]: " + href, "[certificate][]\n\n[certificate]: " + href, "md", false},
		{"inline", "[certificate](" + href + ")", "plain prose", "md", true},
		{"html", "<a href=\"https://example.invalid/certificate?a=1&amp;b=2\">certificate</a>", "plain prose", "html", true},
		{"code-is-not-usage", "`[certificate](" + href + ")`", "plain prose", "md", false},
		{"non-link-failure", "plain prose", "runtime failure", "md", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			links, err := missingTULinks([]byte(tc.source), []byte(tc.target), tc.format)
			if tc.want {
				if err != nil || len(links) != 1 || links[0] != href {
					t.Fatal(links, err)
				}
			} else if err == nil {
				t.Fatal("no TU loss should authorize reopen", links)
			}
		})
	}
}

func TestPreviewFailureFIPS140RealFixture(t *testing.T) {
	// Read frozen canonical content; construct only a temporary defect fixture.
	source, err := os.ReadFile("../../_content/doc/security/fips140.md")
	if err != nil {
		t.Fatal(err)
	}
	target := bytes.Replace(source, []byte("[CMVP validation certificate][]"), []byte("[CMVP 验证证书][]"), 1)
	links, err := missingTULinks(source, target, "md")
	if err != nil || len(links) != 1 || !strings.Contains(links[0], "ModuleName=Go+Cryptographic+Module") {
		t.Fatal(links, err)
	}
	if _, err := missingTULinks(source, source, "md"); err == nil {
		t.Fatal("intact canonical reference authorized reopen")
	}
}

func TestPreviewFailureCurrentObservationAndBuildBinding(t *testing.T) {
	root := t.TempDir()
	route := "/fixture/proof"
	source := []byte("---\ntitle: Fixture\n---\n\nRead [certificate][].\n\n[certificate]: https://example.invalid/required\n")
	target := bytes.Replace(source, []byte("Read [certificate][]."), []byte("阅读 [证书][]。"), 1)
	doc, err := sitecontent.ParseWorkflowDocument(sitecontent.Source{Path: "_content/fixture.md", Kind: "page", Route: route, SHA256: previewDigest(source)}, source, 1)
	if err != nil {
		t.Fatal(err)
	}
	u := i18n.AdaptedTranslationUnit{Unit: i18n.TranslationUnit{ID: doc.Units[0].ID, Kind: i18n.UnitKindPage, SourcePath: doc.Path, SourceSHA256: doc.SourceSHA, Source: source}, Adapter: sitecontent.PageAdapter{Document: *doc, Source: source}, Route: route, StableIndex: 1}
	catalog := &i18n.Catalog{}
	if err := catalog.RegisterTranslationPackage(i18n.TranslationPackageDescriptor{ID: "arbitrary-package", AuthoritySHA256: previewDigest([]byte("package")), Units: []i18n.AdaptedTranslationUnit{u}}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "candidate.md"), target, 0644); err != nil {
		t.Fatal(err)
	}
	content := &Content{Pages: map[string]ContentPage{route: {Route: route, Path: doc.Path, Format: "md", Content: target}}, Data: map[string][]byte{}, Resolver: &Resolver{origin: "https://locale.example", routes: map[string]bool{route: true}, canonical: map[string]bool{route: true}, aliases: map[string]string{}}}
	body, err := content.renderPage("fr-FR", content.Pages[route])
	if err != nil {
		t.Fatal(err)
	}
	body, err = DeliveryResources(route, body, true)
	if err != nil {
		t.Fatal(err)
	}
	observed := bytes.Clone(body)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != route {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(observed)
	}))
	defer server.Close()
	p := &i18n.PreviewTUFailure{Locale: "fr-FR", Package: "arbitrary-package", PackageSHA256: catalog.TranslationPackages[0].AuthoritySHA256, Route: route, PreviewURL: server.URL + "/", ContextSHA256: previewDigest([]byte("build")), ResponseHTML: string(body), MissingLinks: []string{"https://example.invalid/required"}, Artifact: i18n.SurfaceCandidateArtifact{Unit: i18n.QualityCheckSnapshotUnit{UnitID: u.Unit.ID, SourcePath: doc.Path, SourceSHA256: doc.SourceSHA, CandidatePath: "candidate.md"}}}
	load := func(string, *i18n.PreviewTUFailure) (*Content, string, []i18n.GlossaryArchiveReference, error) {
		return content, previewDigest([]byte("build")), nil, nil
	}
	if err := verifyPreviewFailureWithContext(root, catalog, p, true, load); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"build", "verifier-contract", "response", "live-preview", "candidate-projection", "wrong-route", "wrong-package"} {
		t.Run(kind, func(t *testing.T) {
			bad := *p
			switch kind {
			case "build":
				bad.ContextSHA256 = previewDigest([]byte("old build"))
			case "verifier-contract":
				bad.Contracts = []i18n.GlossaryArchiveReference{{Path: "scripts/verify-preview-browser.py", SHA256: previewDigest([]byte("old verifier"))}}
			case "response":
				bad.ResponseHTML = "<main>unbound user supplied HTML</main>"
			case "live-preview":
				observed = []byte("<main>different running build</main>")
				defer func() { observed = body }()
			case "candidate-projection":
				page := content.Pages[route]
				changed := page
				changed.Content = source
				content.Pages[route] = changed
				defer func() { content.Pages[route] = page }()
			case "wrong-route":
				bad.Route = "/different"
			case "wrong-package":
				bad.Package = "different-package"
			}
			if err := verifyPreviewFailureWithContext(root, catalog, &bad, true, load); err == nil {
				t.Fatal("stale/mismatched preview authority accepted")
			}
		})
	}
	server.Close()
	if err := verifyPreviewFailureWithContext(root, catalog, p, false, load); err != nil {
		t.Fatal("frozen invocation required old live preview", err)
	}
	for _, base := range []string{"https://example.com/", "http://127.0.0.1/", "http://user@127.0.0.1:1234/", "http://127.0.0.1:1234/?x=1"} {
		if _, err := previewBase(base); err == nil {
			t.Fatal("unsafe origin accepted", base)
		}
	}
}
