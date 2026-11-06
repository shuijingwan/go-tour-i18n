package site

import (
	"bytes"
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestResolveLinkFragmentPreservation(t *testing.T) {
	r := testResolver(t, "learn-docs-v1", "tour-v1")
	for input, want := range map[string]string{
		"#section":               "#section",
		"#section-1":             "#section-1",
		"#café":                  "#caf%C3%A9",
		"#caf%c3%a9":             "#caf%c3%a9",
		"#section%2Fpart":        "#section%2Fpart",
		"#section?literal=query": "#section?literal=query",
		"#":                      "https://locale.example/learn/",
		"/learn/#section":        "https://locale.example/learn/#section",
		"/doc/tutorial/#run":     "https://locale.example/doc/tutorial/#run",
		"/doc/install#run":       "https://go.dev/doc/install#run",
		"https://example.invalid/page?q=%2F#caf%c3%a9": "https://example.invalid/page?q=%2F#caf%c3%a9",
		"/doc/tutorial/?q=%2f&x=1+2#caf%c3%a9":         "https://locale.example/doc/tutorial/?q=%2f&x=1+2#caf%c3%a9",
		"/doc/install?mode=json#run":                   "https://go.dev/doc/install?mode=json#run",
		"?mode=json#run":                               "https://locale.example/learn/?mode=json#run",
		"/security/#section":                           "https://locale.example/doc/security/#section",
		"/tour/basics/1#run":                           "https://locale.example/tour/basics/1#run",
	} {
		t.Run(input, func(t *testing.T) {
			got, err := r.ResolveLink("/learn/", input)
			if err != nil || got != want {
				t.Fatalf("ResolveLink(%q) = %q, %v; want %q", input, got, err, want)
			}
		})
	}
	for _, from := range []string{"/doc/tutorial/index.html", "/doc/tutorial/"} {
		got, err := r.ResolveLink(from, "getting-started.html#run")
		if err != nil || got != "https://locale.example/doc/tutorial/getting-started#run" {
			t.Fatal("relative cross-document link changed", got, err)
		}
	}
	// Same-document references do not require coverage, even on a route that
	// would be externalized as a cross-document destination.
	if got, err := r.ResolveLink("/not-covered/", "#section"); err != nil || got != "#section" {
		t.Fatal("fragment incorrectly used coverage", got, err)
	}
	for _, input := range []string{"#bad%zz", "/bad\x00#section", "javascript:alert(1)#section", "/unsafe\\path#section", "https://user:password@example.invalid/#section"} {
		if _, err := r.ResolveLink("/learn/", input); err == nil {
			t.Fatalf("unsafe input accepted: %q", input)
		}
	}
	if _, err := r.ResolveLink("invalid-context", "#section"); err == nil {
		t.Fatal("fragment bypassed route context validation")
	}
	if _, err := r.Destination("#section"); err == nil {
		t.Fatal("absolute destination contract changed")
	}
}

func learnFragmentHrefs(t *testing.T, raw []byte) []string {
	t.Helper()
	doc, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	var links []string
	var walk func(*html.Node, bool)
	walk = func(n *html.Node, inNav bool) {
		attrs := map[string]string{}
		for _, a := range n.Attr {
			attrs[a.Key] = a.Val
		}
		for _, class := range strings.Fields(attrs["class"]) {
			inNav = inNav || class == "LearnNav"
		}
		if inNav && n.Type == html.ElementNode && n.Data == "a" {
			links = append(links, attrs["href"])
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch, inNav)
		}
	}
	walk(doc, false)
	return links
}

func TestFrozenLearnFragmentDeliveryAndJumplinks(t *testing.T) {
	// Frozen English is a format/runtime fixture, never locale completion or
	// translation fallback. No repository artifacts are created by this test.
	raw, err := os.ReadFile("../../_content/learn/index.md")
	if err != nil {
		t.Fatal(err)
	}
	want := learnFragmentHrefs(t, raw)
	if len(want) == 0 {
		t.Fatal("frozen Learn fixture has no navigation")
	}
	c := &Content{Resolver: testResolver(t, "learn-docs-v1"), Data: map[string][]byte{}}
	for _, name := range []string{"site.tmpl", "menus.yaml", "resources.yaml", "testimonials.yaml", "learn/quickstart.yaml", "learn/guided.yaml", "learn/cloud.yaml", "learn/tutorials.yaml", "learn/training.yaml", "learn/books.yaml"} {
		c.Data["_content/"+name], err = os.ReadFile(filepath.Join("../../_content", name))
		if err != nil {
			t.Fatal(err)
		}
	}
	p := ContentPage{Route: "/learn/", Path: "_content/learn/index.md", Format: "md", Content: raw}
	body, err := c.renderPage("fr-FR", p)
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile("../../_content/js/jumplinks.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, preview := range []bool{true, false} {
		delivered, err := DeliveryResources(p.Route, body, preview)
		if err != nil {
			t.Fatal(err)
		}
		links := learnFragmentHrefs(t, delivered)
		if !reflect.DeepEqual(links, want) {
			t.Fatalf("Learn fragment shape changed: got %v; want %v", links, want)
		}
		doc, err := html.Parse(bytes.NewReader(delivered))
		if err != nil {
			t.Fatal(err)
		}
		for _, href := range links {
			u, err := url.Parse(href)
			if err != nil || !strings.HasPrefix(href, "#") || u.Fragment == "" {
				t.Fatal("fixture not a pure fragment", href)
			}
			var found bool
			var find func(*html.Node)
			find = func(n *html.Node) {
				for _, a := range n.Attr {
					if a.Key == "id" && a.Val == u.Fragment && strings.TrimSpace(nodeText(n)) != "" {
						found = true
					}
				}
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					find(ch)
				}
			}
			find(doc)
			if !found {
				t.Fatal("fragment lost substantive target", href)
			}
		}
		input, _ := json.Marshal(map[string]any{"script": string(script), "links": links})
		cmd := exec.Command("node", "-e", frozenJumplinksHarness)
		cmd.Stdin = bytes.NewReader(input)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("unmodified jumplinks integration: %v\n%s", err, out)
		}
	}
}

const frozenJumplinksHarness = `
const assert=require('node:assert/strict'),vm=require('node:vm');
const input=JSON.parse(require('node:fs').readFileSync(0,'utf8'));
const anchors=input.links.map(raw=>{const classes=new Set();return {raw,
 href:new URL(raw,'https://locale.example/learn/').href,
 classList:{add(c){classes.add(c)},remove(c){classes.delete(c)},contains(c){return classes.has(c)}}};});
const sections=input.links.map(raw=>({id:decodeURIComponent(raw.slice(1)),scrollIntoView(){}}));
let callback;const observed=[];
const context={location:{href:'https://locale.example/learn/',hash:''},console:{error(e){throw e}},
 IntersectionObserver:class{constructor(cb){callback=cb}observe(section){observed.push(section)}},
 document:{querySelector(selector){
  if(selector==='.LearnNav')return {children:anchors};
  if(selector==='.Learn-body')return {children:sections};
  const m=selector.match(/^\[href='#(.*)'\]$/);
  assert.ok(m,'unexpected selector: '+selector);
  return anchors.find(a=>a.raw==='#'+m[1])||null;
 },getElementById(id){return sections.find(s=>s.id===id)||null}}};
vm.runInNewContext(input.script,context);
setImmediate(async()=>{
 assert.equal(typeof callback,'function');assert.deepEqual(observed,sections);
 for(const section of sections){
  await callback([{isIntersecting:true,target:section}]);
  assert.equal(anchors.filter(a=>a.classList.contains('active')).length,1);
  assert.ok(anchors.find(a=>a.raw==='#'+section.id).classList.contains('active'));
 }
 // Removing a required anchor remains a genuine rejected runtime operation.
 anchors.splice(0,1);
 await assert.rejects(callback([{isIntersecting:true,target:sections[0]}]),/classList/);
});
`
