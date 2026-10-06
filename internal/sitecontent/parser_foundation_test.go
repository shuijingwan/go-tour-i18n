package sitecontent

import (
	"bytes"
	"strings"
	"testing"
)

func TestSiteFoundationParserProtection(t *testing.T) {
	samples := []struct{ path, kind, body string }{
		{"_content/doc/security/example.md", "page", "---\ntitle: A natural title\nlayout: article\nbreadcrumb: true\n---\n\n# Heading\n\nTranslate prose with `fmt.Println` and [a label](https://go.dev/x#anchor).\n\n```go\nfmt.Println(\"keep this command\")\n```\n\n<pre>go run ./cmd\nunchanged output</pre>\n\n<p id=\"anchor\" class=\"machine\"><a href=\"/doc/security/\" title=\"Visible title\">Visible label</a></p>\n"},
		{"_content/doc/tutorial/example.html", "page", "<!--{\"Title\":\"Natural title\",\"Path\":\"/doc/tutorial/example\",\"Breadcrumb\":true}-->\n<p>Natural <code>GoIdentifier</code> text with <a href=\"https://example.com/x\">link label</a>.</p>\n<pre>package main\nvar machine = 1</pre>\n"},
		{"_content/learn/example.yaml", "data", "- title: Natural title\n  content: <i>Natural words</i> with prose.\n  url: /doc/tutorial/example\n  credits: 5\n  rating: 4.25\n  thumbnail: /images/example.png\n  cta: Read more\n"},
	}
	for _, sample := range samples {
		t.Run(sample.kind+sample.path, func(t *testing.T) {
			b := []byte(sample.body)
			s := Source{Path: sample.path, Kind: sample.kind, SHA256: digest(b), Route: "/doc/security/example"}
			d, err := ParseDocument(s, b)
			if err != nil {
				t.Fatal(err)
			}
			targets := map[string]string{}
			for _, u := range d.Units {
				targets[u.ID] = "译文 " + u.Source
				for _, protected := range []string{"fmt.Println", "GoIdentifier", "go run", "package main", "https://", "/images/", "/doc/"} {
					if strings.Contains(u.Source, protected) {
						t.Fatalf("protected text became language: %s", u.Source)
					}
				}
			}
			first, err := Reconstruct(s, b, targets)
			if err != nil {
				t.Fatal(err)
			}
			second, err := Reconstruct(s, b, targets)
			if err != nil || !bytes.Equal(first, second) {
				t.Fatal("nondeterministic reconstruction", err)
			}
			for _, protected := range []string{"fmt.Println", "go run ./cmd", "GoIdentifier", "package main", "href=\"https://example.com/x\"", "credits: 5", "rating: 4.25", "id=\"anchor\"", "layout: article", "/images/example.png"} {
				if strings.Contains(sample.body, protected) && !bytes.Contains(first, []byte(protected)) {
					t.Fatal("protected bytes changed", protected)
				}
			}
			one := d.Units[0].ID
			targets[one] = "<script>bad</script>"
			if _, err := Reconstruct(s, b, targets); err == nil {
				t.Fatal("markup injection")
			}
			delete(targets, one)
			if _, err := Reconstruct(s, b, targets); err == nil {
				t.Fatal("missing unit accepted")
			}
		})
	}
}
func TestSiteFoundationParserUnknown(t *testing.T) {
	for _, body := range []string{"---\ntitle: Text\nunknown: true\n---\nText\n", "<!--{\"Title\":\"Text\",\"Unknown\":true}-->\nText\n", "<!--{\"Title\":\"Text\",\"Title\":\"Duplicate\"}-->\nText\n", "---\ntitle: Text\nlayout: unknown\n---\nText\n"} {
		s := Source{Path: "x.md", Kind: "page", SHA256: digest([]byte(body))}
		if _, err := ParseDocument(s, []byte(body)); err == nil {
			t.Fatal("unknown metadata accepted", body)
		}
	}
	for _, body := range []string{"- title: Text\n  url: /x\n  unknown: y\n", "- title: Text\n  url: /x\n  content: |\n    multiline\n", "- title: Text\n  title: Duplicate\n  url: /x\n", "- title: Text\n  url: /x\n  content:\n    nested: invalid\n"} {
		s := Source{Path: "x.yaml", Kind: "data", SHA256: digest([]byte(body))}
		if _, err := ParseDocument(s, []byte(body)); err == nil {
			t.Fatal("unknown data accepted", body)
		}
	}
	for _, kind := range []string{"redirect", "asset", "unknown"} {
		s := Source{Path: "x.md", Kind: kind, SHA256: digest([]byte("Text\n"))}
		if _, err := ParseDocument(s, []byte("Text\n")); err == nil {
			t.Fatal("non-language became unit")
		}
	}
	for _, body := range []string{"<unknown-element>Text</unknown-element>\n", "Text {{unknownDirective}}\n"} {
		s := Source{Path: "unknown.md", Kind: "page", SHA256: digest([]byte(body))}
		if _, err := ParseDocument(s, []byte(body)); err == nil {
			t.Fatal("unknown structure/directive accepted")
		}
	}
}
func TestSiteFoundationReconciliation(t *testing.T) {
	parse := func(p, text string) Document {
		b := []byte(text)
		d, err := ParseDocument(Source{Path: p, Kind: "page", Route: "/doc/security/example", SHA256: digest(b)}, b)
		if err != nil {
			t.Fatal(err)
		}
		return *d
	}
	old := []Document{parse("x.md", "# Heading\n\nUnchanged text.\n\nOld text.\n")}
	next := []Document{parse("x.md", "# Heading\n\nUnchanged text.\n\nNew text.\n")}
	r, err := ReconcileDocuments(old, next)
	if err != nil {
		t.Fatal(err)
	}
	states := map[string]int{}
	for _, u := range r.Units {
		states[u.State]++
	}
	if states["carry"] != 2 || states["generation"] != 1 || states["stale"] != 1 {
		t.Fatal("whole document invalidation", states)
	}
	changedRoute := append([]Document{}, next...)
	changedRoute[0].Route = "/doc/security/changed"
	rr, err := ReconcileDocuments(next, changedRoute)
	if err != nil || len(rr.NeedsMapping) != 1 {
		t.Fatal("route mapping", err)
	}
	bad := append([]Document{}, old...)
	bad[0].Contract = "unknown"
	if _, err := ReconcileDocuments(bad, next); err == nil {
		t.Fatal("unknown parser reconciliation")
	}
}
