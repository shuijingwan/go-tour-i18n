package site

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/tour"
	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
	"golang.org/x/net/html"
)

func TestSitePlaygroundBootstrapRuntime(t *testing.T) {
	root, _ := filepath.Abs("../..")
	runtime, err := os.ReadFile(filepath.Join(root, "_content/js/playground.js"))
	if err != nil {
		t.Fatal(err)
	}
	for _, locale := range []string{"zh-CN", "fr-FR"} {
		c := &Content{Resolver: &Resolver{origin: "https://locale.example", routes: map[string]bool{"/": true}, canonical: map[string]bool{}, aliases: map[string]string{}}, Pages: map[string]ContentPage{"/": {Route: "/", Path: "fixture.html", Format: "html", Content: []byte("---\ntitle: Fixture\n---\n\n<main><textarea class=\"js-playgroundCodeEl\"></textarea><button class=\"js-playgroundRunEl\">Run</button></main><script src=\"/js/playground.js\"></script>")}}}
		h, err := c.Compose(locale, nil)
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 200 {
			t.Fatal("Site executable page failed", w.Code)
		}
		catalog, err := ui.Load(locale)
		if err != nil {
			t.Fatal(err)
		}
		for _, preview := range []bool{true, false} {
			raw, err := c.renderPage(locale, c.Pages["/"])
			if err != nil {
				t.Fatal(err)
			}
			body, err := DeliveryResources("/", raw, preview)
			if err != nil {
				t.Fatal(err)
			}
			if preview && !bytes.Equal(body, w.Body.Bytes()) {
				t.Fatal("Preview composition uses a different bootstrap path")
			}
			if strings.Count(string(body), `id="site-playground-ui"`) != 1 || bytes.Index(body, []byte(`id="site-playground-ui"`)) > bytes.Index(body, []byte(`/js/playground.js"`)) {
				t.Fatal("duplicate or misordered bootstrap")
			}
			doc, _ := html.Parse(bytes.NewReader(body))
			code := ""
			var walk func(*html.Node)
			walk = func(n *html.Node) {
				for _, a := range n.Attr {
					if a.Key == "id" && a.Val == playgroundUIBootstrapID {
						code = nodeText(n)
					}
				}
				for child := n.FirstChild; child != nil; child = child.NextSibling {
					walk(child)
				}
			}
			walk(doc)
			canonical, err := tour.UIMessagesBootstrap(locale)
			if err != nil || code != strings.TrimSpace(string(canonical)) {
				t.Fatal("Site bootstrap is not canonical Tour authority", err)
			}
			cmd := exec.Command("node", "-e", sitePlaygroundBootstrapHarness)
			input, _ := json.Marshal(map[string]string{"bootstrap": code, "runtime": string(runtime), "exited": catalog.Messages["execution.exited"].Text, "communication": catalog.Messages["execution.communication_error"].Text})
			cmd.Stdin = bytes.NewReader(input)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("Site runtime: locale=%s preview=%v %v\n%s", locale, preview, err, out)
			}
		}
	}
}

const sitePlaygroundBootstrapHarness = `
const vm=require('node:vm'),assert=require('node:assert/strict');
const p=JSON.parse(require('node:fs').readFileSync(0,'utf8')),spans=[];
const c={window:{},document:{createElement(){return {}}},setTimeout,clearTimeout,
 $:{ajax(url,options){assert.equal(url,'/_/compile?backend=');options.error()}}};
vm.createContext(c);vm.runInContext(p.bootstrap,c);vm.runInContext(p.runtime,c);
const output=c.PlaygroundOutput({innerHTML:'',scrollTop:0,offsetHeight:0,scrollHeight:0,appendChild(s){spans.push(s)}});
output({Kind:'start'});output({Kind:'end'});
assert.equal(c.window.__tourUIMessages['execution.exited'],p.exited);
assert.equal(spans.at(-1).innerHTML,'\n'+p.exited+'.');
const writes=[];c.HTTPTransport(false).Run('package main',w=>writes.push(w),{});
assert.ok(writes.some(w=>w.Kind==='stderr'&&w.Body===p.communication));
assert.equal(writes.at(-1).Kind,'end');
`

func TestSitePlaygroundBootstrapBoundaries(t *testing.T) {
	const runtime = `<script src="/js/playground.js"></script>`
	base := `<html lang="fr-FR"><head>` + runtime + `</head><body>Static content</body></html>`
	got, err := DeliveryResources("/", []byte(base), true)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DeliveryResources("/", got, true)
	if err != nil || !bytes.Equal(got, again) {
		t.Fatal("bootstrap delivery is not idempotent", err)
	}
	bootstrap, _ := tour.UIMessagesBootstrap("fr-FR")
	inline := `<script id="site-playground-ui">` + string(bootstrap) + `</script>`
	for _, body := range []string{
		strings.Replace(base, `lang="fr-FR"`, "", 1),
		strings.Replace(base, "fr-FR", "zz-UNKNOWN", 1),
		strings.Replace(base, runtime, runtime+inline, 1),
		strings.Replace(base, runtime, inline+inline+runtime, 1),
		strings.Replace(base, runtime, `<script id="site-playground-ui">window.__tourUIMessages={}</script>`+runtime, 1),
		strings.Replace(base, runtime, `<script>window['__tourUIMessages'] = {}</script>`+runtime, 1),
		strings.Replace(string(got), "fr-FR", "zh-CN", 1),
	} {
		if _, err := DeliveryResources("/", []byte(body), true); err == nil {
			t.Fatal("missing locale or conflicting/misordered bootstrap accepted")
		}
	}
	static := `<html lang="zz-UNKNOWN"><body><pre>static code</pre></body></html>`
	body, err := DeliveryResources("/", []byte(static), true)
	if err != nil || bytes.Contains(body, []byte("__tourUIMessages")) {
		t.Fatal("standalone static page gained runtime state", err)
	}
}
