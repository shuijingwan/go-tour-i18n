package site

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/assets"
	"golang.org/x/net/html"
)

// DeliveryResources is a downstream execution adapter. It leaves the frozen
// language renderer/source-adoption contract intact; both Preview composition
// and publication consume the same resource policy after language rendering.
func DeliveryResources(route string, body []byte, development ...bool) ([]byte, error) {
	local := len(development) == 1 && development[0]
	doc, e := html.Parse(bytes.NewReader(body))
	if e != nil {
		return nil, e
	}
	if local {
		if e := omitPreviewSourceTelemetry(doc); e != nil {
			return nil, e
		}
	}
	base, e := url.Parse("https://go.dev" + route)
	if e != nil {
		return nil, e
	}
	shared, e := url.Parse(assets.BaseURL)
	if e != nil {
		return nil, e
	}
	var playgroundRuntime *html.Node
	var walk func(*html.Node) error
	walk = func(n *html.Node) error {
		if n.Type == html.ElementNode {
			for i, a := range n.Attr {
				if a.Key != "src" && !(n.Data == "link" && a.Key == "href") {
					continue
				}
				u, e := url.Parse(a.Val)
				if e != nil {
					return e
				}
				if u.Host != "" && u.Host != "go.dev" && u.Host != shared.Host {
					continue
				}
				u = base.ResolveReference(u)
				p := strings.TrimPrefix(u.Path, "/")
				if n.Data == "script" && a.Key == "src" && p == "js/playground.js" && playgroundRuntime == nil {
					playgroundRuntime = n
				}
				if !assets.StaticPath(p) {
					continue
				}
				target, adapted := runtimeScriptURL(p)
				if !adapted || n.Data != "script" {
					target, e = assets.URL("", local, p)
				}
				if e != nil {
					return e
				}
				if u.RawQuery != "" {
					target += "?" + u.RawQuery
				}
				if u.Fragment != "" {
					target += "#" + u.EscapedFragment()
				}
				n.Attr[i].Val = target
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if e := walk(c); e != nil {
				return e
			}
		}
		return nil
	}
	if e = walk(doc); e != nil {
		return nil, e
	}
	if e = bindPlaygroundUI(doc, playgroundRuntime); e != nil {
		return nil, e
	}
	var out bytes.Buffer
	e = html.Render(&out, doc)
	return out.Bytes(), e
}

// Complete-locale Preview omits production analytics, including GTM embedded
// in the frozen upstream shell rather than supplied by TOUR_ANALYTICS. The
// template engine removes HTML comments, so rendered nodes are identified by
// their complete loader/fallback role as well as source markers when retained.
// Production delivery and ad policy are untouched; unknown resources still fail.
func omitPreviewSourceTelemetry(parent *html.Node) error {
	for n := parent.FirstChild; n != nil; {
		next := n.NextSibling
		if n.Type == html.ElementNode && previewGTMNode(n) {
			parent.RemoveChild(n)
		} else if n.Type == html.CommentNode && (strings.TrimSpace(n.Data) == "Google Tag Manager" || strings.TrimSpace(n.Data) == "Google Tag Manager (noscript)") {
			end := n.NextSibling
			for end != nil && !(end.Type == html.CommentNode && strings.TrimSpace(end.Data) == "End "+strings.TrimSpace(n.Data)) {
				if !previewGTMNode(end) {
					return fmt.Errorf("unexpected content inside source GTM block")
				}
				end = end.NextSibling
			}
			if end == nil {
				return fmt.Errorf("unclosed source GTM block")
			}
			next = end.NextSibling
			for node := n; node != next; {
				following := node.NextSibling
				parent.RemoveChild(node)
				node = following
			}
		} else if err := omitPreviewSourceTelemetry(n); err != nil {
			return err
		}
		n = next
	}
	return nil
}

func previewGTMNode(n *html.Node) bool {
	if n.Type == html.TextNode {
		return strings.TrimSpace(n.Data) == ""
	}
	if n.Type != html.ElementNode {
		return false
	}
	attrs := map[string]string{}
	for _, a := range n.Attr {
		attrs[a.Key] = a.Val
	}
	switch n.Data {
	case "link":
		return len(attrs) == 2 && attrs["rel"] == "preconnect" && attrs["href"] == "https://www.googletagmanager.com"
	case "script":
		// Match the whole standard bootstrap, allowing whitespace/container
		// identity to vary. A script on a similar host, or a script containing
		// GTM text plus functional code, must never be silently discarded.
		const prefix = `(function(w,d,s,l,i){w[l]=w[l]||[];w[l].push({'gtm.start':newDate().getTime(),event:'gtm.js'});varf=d.getElementsByTagName(s)[0],j=d.createElement(s),dl=l!='dataLayer'?'&l='+l:'';j.async=true;j.src='https://www.googletagmanager.com/gtm.js?id='+i+dl;f.parentNode.insertBefore(j,f);})(window,document,'script','dataLayer','`
		const suffix = `');`
		code := strings.Join(strings.Fields(nodeText(n)), "")
		container := strings.TrimSuffix(strings.TrimPrefix(code, prefix), suffix)
		if len(attrs) != 0 || !strings.HasPrefix(code, prefix) || !strings.HasSuffix(code, suffix) || !strings.HasPrefix(container, "GTM-") || len(container) <= 4 {
			return false
		}
		for _, c := range container[4:] {
			if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
				return false
			}
		}
		return true
	case "noscript":
		if len(attrs) != 0 {
			return false
		}
		// With scripting enabled the HTML parser keeps noscript as raw text.
		// Parse that fragment to verify an actual GTM fallback iframe.
		doc, err := html.Parse(strings.NewReader(nodeText(n)))
		if err != nil {
			return false
		}
		frames := 0
		var valid func(*html.Node) bool
		valid = func(node *html.Node) bool {
			if node.Type == html.ElementNode {
				switch node.Data {
				case "html", "head", "body":
				case "iframe":
					frames++
					frame := map[string]string{}
					for _, a := range node.Attr {
						frame[a.Key] = a.Val
					}
					if len(frame) != 4 || frame["height"] != "0" || frame["width"] != "0" || strings.Join(strings.Fields(frame["style"]), "") != "display:none;visibility:hidden" {
						return false
					}
					u, err := url.Parse(frame["src"])
					if err != nil || u.Scheme != "https" || u.Host != "www.googletagmanager.com" || u.Path != "/ns.html" || u.User != nil || u.Query().Get("id") == "" {
						return false
					}
				default:
					return false
				}
			} else if node.Type == html.TextNode && strings.TrimSpace(node.Data) != "" {
				return false
			}
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if !valid(child) {
					return false
				}
			}
			return true
		}
		return valid(doc) && frames == 1
	}
	return false
}
