package site

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/tour"
	"golang.org/x/net/html"
)

const playgroundUIBootstrapID = "site-playground-ui"

var uiMessageAssignment = regexp.MustCompile(`window\s*(?:\.\s*__tourUIMessages|\[\s*['"]__tourUIMessages['"]\s*\])\s*=`)

// The frozen Site shell intentionally shares playground.js across its pages.
// Bind its message dependency whenever that runtime is loaded, without adding
// state to standalone static pages that do not load the shared runtime.
func bindPlaygroundUI(doc, runtime *html.Node) error {
	if runtime == nil {
		return nil
	}
	locale := ""
	var existing *html.Node
	seenRuntime, existingBeforeRuntime := false, false
	var walk func(*html.Node) error
	walk = func(n *html.Node) error {
		if n == runtime {
			seenRuntime = true
		}
		if n.Type == html.ElementNode {
			id := ""
			for _, a := range n.Attr {
				if n.Data == "html" && a.Key == "lang" {
					locale = a.Val
				}
				if a.Key == "id" {
					id = a.Val
				}
			}
			if id == playgroundUIBootstrapID {
				if existing != nil || n.Data != "script" || len(n.Attr) != 1 {
					return fmt.Errorf("duplicate or invalid Site Playground bootstrap")
				}
				existing, existingBeforeRuntime = n, !seenRuntime
			} else if n.Data == "script" && uiMessageAssignment.MatchString(nodeText(n)) {
				return fmt.Errorf("conflicting Site Playground UI messages")
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(doc); err != nil {
		return err
	}
	bootstrap, err := tour.UIMessagesBootstrap(locale)
	if err != nil {
		return err
	}
	if existing != nil {
		if !existingBeforeRuntime || nodeText(existing) != strings.TrimSpace(string(bootstrap)) {
			return fmt.Errorf("stale or misordered Site Playground bootstrap")
		}
		return nil
	}
	n := &html.Node{Type: html.ElementNode, Data: "script", Attr: []html.Attribute{{Key: "id", Val: playgroundUIBootstrapID}}}
	n.AppendChild(&html.Node{Type: html.TextNode, Data: string(bootstrap)})
	runtime.Parent.InsertBefore(n, runtime)
	return nil
}
