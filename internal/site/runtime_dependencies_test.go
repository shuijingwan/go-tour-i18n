package site

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/shuijingwan/go-tour-i18n/internal/assets"
	"github.com/shuijingwan/go-tour-i18n/internal/releasecontract"
	"golang.org/x/net/html"
)

func downloadResolver(covered bool) *Resolver {
	r := &Resolver{origin: "https://go-dev.shuijingwanwq.com", routes: map[string]bool{"/": true}, canonical: map[string]bool{"/": true, "/dl/": true}, aliases: map[string]string{}}
	if covered {
		r.routes["/dl/"] = true
	}
	return r
}

func TestDownloadRuntimeProjection(t *testing.T) {
	root, _ := filepath.Abs("../..")
	for _, source := range []string{"js/site.js", "doc/download.js"} {
		t.Run(source, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, "_content", source))
			if err != nil {
				t.Fatal(err)
			}
			original := bytes.Clone(raw)
			for _, local := range []bool{false, true} {
				body, err := projectRuntimeScript(downloadResolver(local), source, raw)
				if err != nil {
					t.Fatal(err)
				}
				want := "https://go.dev/dl/?mode=json"
				if local {
					want = "/dl/?mode=json"
				} else if bytes.Contains(body, []byte("'/dl/")) || bytes.Contains(body, []byte("`/dl/")) {
					t.Fatal("uncovered same-origin runtime URL survived")
				}
				if !bytes.Contains(body, []byte(want)) || !bytes.Equal(raw, original) {
					t.Fatal("query or frozen source bytes changed")
				}
				if _, err := exec.LookPath("node"); err != nil {
					t.Fatal("node required for runtime execution regression")
				}
				input := map[string]any{"source": source, "body": string(body), "local": local}
				if source == "js/site.js" {
					input["nodes"], input["spans"] = downloadDOMFixture(t, "index.md")
				}
				runDownloadRuntime(t, input)
			}
		})
	}
}

// Read download hooks from the real frozen HTML shapes. Missing selectors stay
// absent: a generic button must never manufacture platform-specific anchors.
func downloadDOMFixture(t *testing.T, source string) (map[string]map[string]string, []map[string]string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("../../_content", source))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	nodes := map[string]map[string]string{}
	var spans []map[string]string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		attrs := map[string]string{}
		for _, a := range n.Attr {
			attrs[a.Key] = a.Val
		}
		for _, class := range strings.Fields(attrs["class"]) {
			if class == "GoVersionSpan" {
				spans = append(spans, map[string]string{"textContent": nodeText(n)})
			}
			switch class {
			case "js-latestGoVersion", "js-downloadBtn", "js-downloadWin", "js-downloadMac", "js-downloadLinux":
				node := map[string]string{}
				if href := attrs["href"]; href != "" {
					node["href"], err = downloadResolver(false).Destination(href)
					if err != nil {
						t.Fatal(err)
					}
				}
				nodes["."+class] = node
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return nodes, spans
}

func runDownloadRuntime(t *testing.T, input map[string]any) {
	t.Helper()
	cmd := exec.Command("node", "-e", downloadRuntimeHarness)
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdin = bytes.NewReader(data)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("runtime execution: %v\n%s", err, out)
	}
}

func TestDownloadRuntimeDOMShapes(t *testing.T) {
	raw, err := os.ReadFile("../../_content/js/site.js")
	if err != nil {
		t.Fatal(err)
	}
	body, err := projectRuntimeScript(downloadResolver(false), "js/site.js", raw)
	if err != nil {
		t.Fatal(err)
	}
	platforms := []string{".js-downloadWin", ".js-downloadMac", ".js-downloadLinux"}
	for _, source := range []string{"learn/index.md", "index.md"} {
		t.Run(source, func(t *testing.T) {
			nodes, spans := downloadDOMFixture(t, source)
			if nodes[".js-latestGoVersion"] == nil || nodes[".js-downloadBtn"] == nil {
				t.Fatal("real fixture missing generic download shape")
			}
			for _, selector := range platforms {
				if (nodes[selector] != nil) != (source == "index.md") {
					t.Fatalf("unexpected platform shape: %s", selector)
				}
			}
			if source == "learn/index.md" && len(spans) != 1 {
				t.Fatal("Learn version span not represented")
			}
			runDownloadRuntime(t, map[string]any{"source": "js/site.js", "body": string(body), "nodes": nodes, "spans": spans})
		})
	}
	for mask := 0; mask < 7; mask++ {
		if mask == 0 {
			t.Run("no-version-element", func(t *testing.T) {
				runDownloadRuntime(t, map[string]any{"source": "js/site.js", "body": string(body)})
			})
			continue
		}
		t.Run(fmt.Sprintf("partial-platforms-%d", mask), func(t *testing.T) {
			nodes, spans := downloadDOMFixture(t, "index.md")
			for i, selector := range platforms {
				if mask&(1<<i) == 0 {
					delete(nodes, selector)
				}
			}
			runDownloadRuntime(t, map[string]any{"source": "js/site.js", "body": string(body), "nodes": nodes, "spans": spans, "partial": true})
		})
	}
	if after, err := os.ReadFile("../../_content/js/site.js"); err != nil || !bytes.Equal(after, raw) {
		t.Fatal("frozen runtime input mutated")
	}
}

func TestDownloadRuntimeStructureDriftFailsClosed(t *testing.T) {
	raw, err := os.ReadFile("../../_content/js/site.js")
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string][]byte{
		"missing":    nil,
		"selector":   bytes.Replace(raw, []byte("'.js-downloadMac'"), []byte("'.js-otherMac'"), 1),
		"assignment": bytes.Replace(raw, []byte("anchorTagLinux.href = linuxDownloadUrl;"), []byte("anchorTagLinux.href = macDownloadUrl;"), 1),
		"duplicate":  append(bytes.Clone(raw), []byte("\n"+frozenSetDownloadLinks)...),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := projectRuntimeScript(downloadResolver(false), "js/site.js", body); err == nil {
				t.Fatal("unsupported frozen structure accepted")
			}
			w := httptest.NewRecorder()
			serveRuntimeScript(w, httptest.NewRequest("GET", runtimeScriptPrefix+"js/site.js", nil), downloadResolver(false), "js/site.js", body)
			if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "async function") {
				t.Fatal("unverified runtime delivered", w.Code, w.Body.String())
			}
		})
	}
}

// Run the actual frozen scripts after delivery projection, including their
// asynchronous initialization and fetch().json(). No failing local URL can
// pass this harness; genuine HTTP/CORS are additionally checked in Chrome.
const downloadRuntimeHarness = `
const assert=require('node:assert/strict'),vm=require('node:vm');
const p=JSON.parse(require('node:fs').readFileSync(0,'utf8'));
const requests=[], errors=[];
process.on('unhandledRejection',e=>errors.push(String(e)));
const stub=()=>({textContent:'',style:{},classList:{add(){},remove(){},contains(){return false}},
  addEventListener(){},removeEventListener(){},setAttribute(){},removeAttribute(){},closest(){return null},
  querySelector(){return stub()},querySelectorAll(){return []}});
const anchors=Object.fromEntries(Object.entries(p.nodes||{}).map(([s,a])=>[s,Object.assign(stub(),a)]));
const spans=(p.spans||[]).map(a=>Object.assign(stub(),a));
const hooks=['.js-latestGoVersion','.js-downloadBtn','.js-downloadWin','.js-downloadMac','.js-downloadLinux'];
const document={cookie:'cookie-consent=true',readyState:'loading',documentElement:stub(),body:stub(),addEventListener(){},
 querySelectorAll(s){return s==='.GoVersionSpan'?spans:[]},querySelector(s){
  if(hooks.includes(s))return anchors[s]??null;
  if(s==='.js-downloadDescription'||s==='.js-tabSection')return stub();
  return stub();
 },getElementById(){return null}};
const window={matchMedia(){return {matches:false,addEventListener(){}}},addEventListener(){},
 localStorage:{getItem(){return null},setItem(){}},location:{pathname:'/'}};
const context={document,window,location:window.location,navigator:{userAgent:'Linux'},
 localStorage:window.localStorage,console:{error(e){throw e}},
 fetch:async url=>{requests.push(url);assert.equal(url,p.local?'/dl/?mode=json':'https://go.dev/dl/?mode=json');
  return {json:async()=>[{version:'go1.27.2',files:[{filename:'go1.27.2.linux-amd64.tar.gz'}]}]}}};
vm.createContext(context);
let code=p.body;
if(p.source==='js/site.js'){
 // Invoke the real ready callback after giving initialization its DOM hooks.
 const callbacks=[];document.addEventListener=(name,fn)=>{if(name==='DOMContentLoaded')callbacks.push(fn)};
 vm.runInContext(code,context);for(const fn of callbacks)fn();
}else{vm.runInContext(code,context)}
setImmediate(()=>{assert.ok(requests.length>0,'runtime did not request download data');
 if(p.partial){assert.equal(errors.length,1);assert.match(errors[0],/incomplete platform download anchors/)}
 else assert.deepEqual(errors,[],'unexpected runtime rejection');
 const files={'.js-downloadWin':'windows-amd64.msi','.js-downloadMac':'darwin-amd64.pkg','.js-downloadLinux':'linux-amd64.tar.gz'};
 for(const [s,suffix] of Object.entries(files))if(anchors[s]){
  if(p.partial)assert.equal(anchors[s].href,p.nodes[s].href,'partial setup changed an anchor');
  else assert.equal(anchors[s].href,'https://go.dev/dl/go1.27.2.'+suffix);
 }
 if(anchors['.js-downloadBtn']){
  assert.equal(anchors['.js-downloadBtn'].href,p.nodes['.js-downloadBtn'].href,'generic button changed');
  assert.match(anchors['.js-downloadBtn'].href,/^https:\/\/go\.dev\/dl\/?$/);
 }
 for(const span of spans)assert.equal(span.textContent,'Download (1.27.2)');
});
`

func TestDownloadRuntimeDeliveryAnd404(t *testing.T) {
	r := downloadResolver(false)
	raw := "const data='/dl/?mode=json'; const file=`/dl/${version}.pkg`;\n" + frozenSetDownloadLinks
	c := &Content{Resolver: r, Pages: map[string]ContentPage{"/": {Route: "/", Path: "fixture.html", Format: "html", Content: []byte("---\ntitle: Fixture\n---\n\n<main><a href=\"/dl/\">Download</a><script src=\"/js/site.js\"></script></main>")}}, Resources: map[string][]byte{"/js/site.js": []byte(raw), "/doc/download.js": []byte(raw)}}
	preview, err := c.Compose("zh-CN", nil)
	if err != nil {
		t.Fatal(err)
	}
	// A synthetic sealed artifact tests the same adapter through the release
	// handler; it writes no repository files and runs no Production lifecycle.
	f := fstest.MapFS{"site/pages/home.html": {Data: []byte("<main>Fixture</main>")}, "js/site.js": {Data: []byte(raw)}, "doc/download.js": {Data: []byte(raw)}}
	plan, _ := json.Marshal(map[string]any{"schema": "go-learning/preview-browser-plan/v1", "locale": "zh-CN", "families": []string{"/"}, "canonical_routes": []string{"/"}, "pages": []map[string]string{{"route": "/", "family": "/", "format": "html", "source": "fixture.html", "html": "<main>Fixture</main>"}}})
	f[releasecontract.BrowserPlanPath] = &fstest.MapFile{Data: plan}
	a := &releasecontract.Artifact{Schema: releasecontract.Schema, Locale: "zh-CN", Origin: r.origin, CoverageIdentity: strings.Repeat("1", 64), Routes: []string{"/"}, Families: []string{"/"}, Pages: map[string]string{"/": "site/pages/home.html"}, Inputs: map[string]string{"input": strings.Repeat("2", 64)}, Files: map[string]string{}, Aliases: map[string]string{}, AssetPolicy: releasecontract.AssetPolicy, AssetOrigin: assets.BaseURL, SharedAssets: map[string]string{}}
	for p, b := range f {
		a.Files[p] = releasecontract.Digest(b.Data)
	}
	for _, p := range []string{"js/site.js", "doc/download.js"} {
		a.SharedAssets[p] = a.Files[p]
	}
	a.SharedAssetsIdentity = releasecontract.Identity(a.SharedAssets)
	a.Identity = a.ID()
	manifest, _ := json.Marshal(a)
	f[releasecontract.Path] = &fstest.MapFile{Data: manifest}
	release, err := NewReleaseHandler(f, "zh-CN", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for name, h := range map[string]http.Handler{"preview": preview, "sealed-handler": release} {
		t.Run(name, func(t *testing.T) {
			for _, route := range []string{"/dl/", "/dl/?mode=json", runtimeScriptPrefix + "unknown.js", runtimeScriptPrefix + "../js/site.js"} {
				w := httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest("GET", route, nil))
				if w.Code != 404 || w.Header().Get("Location") != "" {
					t.Fatalf("uncovered route %s: %d", route, w.Code)
				}
			}
			for _, source := range []string{"js/site.js", "doc/download.js"} {
				w := httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest("GET", runtimeScriptPrefix+source, nil))
				if w.Code != 200 || !strings.Contains(w.Body.String(), "https://go.dev/dl/?mode=json") || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/javascript") {
					t.Fatalf("runtime script %s: %d %s", source, w.Code, w.Body.String())
				}
				w = httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest("POST", runtimeScriptPrefix+source, nil))
				if w.Code != 405 {
					t.Fatal("runtime write method accepted")
				}
			}
		})
	}
	prior := f["js/site.js"]
	f["js/site.js"] = &fstest.MapFile{Data: []byte("tampered dependency")}
	w := httptest.NewRecorder()
	release.ServeHTTP(w, httptest.NewRequest("GET", runtimeScriptPrefix+"js/site.js", nil))
	if w.Code != 404 {
		t.Fatal("mutated sealed runtime input accepted")
	}
	f["js/site.js"] = prior
	w = httptest.NewRecorder()
	preview.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if !strings.Contains(w.Body.String(), `href="https://go.dev/dl/"`) || !strings.Contains(w.Body.String(), `src="/_/site-runtime/js/site.js"`) {
		t.Fatal("navigation or runtime delivery contract lost", w.Body.String())
	}
}

func TestDownloadRuntimeResourcePolicy(t *testing.T) {
	for _, local := range []bool{false, true} {
		raw := []byte("fetch(\"/dl/?mode=json\"); fetch(`/dl/?mode=json`); const external='https://go.dev/dl/?mode=json'; const covered='/learn/';")
		body, err := projectRuntimeScript(downloadResolver(local), "doc/download.js", raw)
		want := "https://go.dev/dl/?mode=json"
		if local {
			want = "/dl/?mode=json"
		}
		if err != nil || !bytes.Contains(body, []byte("fetch(\""+want+"\")")) || !bytes.Contains(body, []byte("fetch(`"+want+"`)")) || !bytes.Contains(body, []byte("const external='https://go.dev/dl/?mode=json'; const covered='/learn/';")) {
			t.Fatal("literal forms, covered dependency or external authority changed", string(body), err)
		}
	}
	for _, dev := range []bool{false, true} {
		body, err := DeliveryResources("/", []byte(`<script src="https://go.dev/js/site.js?v=1#runtime"></script><script async src="/doc/download.js"></script><a href="https://go.dev/dl/">Download</a><script src="https://external.invalid/js/site.js"></script>`), dev)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{`src="/_/site-runtime/js/site.js?v=1#runtime"`, `src="/_/site-runtime/doc/download.js"`, `href="https://go.dev/dl/"`, `src="https://external.invalid/js/site.js"`} {
			if !bytes.Contains(body, []byte(want)) {
				t.Fatal("dependency delivery changed", want, string(body))
			}
		}
	}
	if _, err := projectRuntimeScript(downloadResolver(false), "unknown.js", nil); err == nil {
		t.Fatal("unregistered script accepted")
	}
	r := downloadResolver(false)
	r.aliases["/dl/"] = "https://unregistered.invalid/"
	if _, err := projectRuntimeScript(r, "js/site.js", nil); err == nil {
		t.Fatal("unknown runtime authority accepted")
	}
}
