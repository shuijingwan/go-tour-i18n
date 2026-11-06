package sitecontent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestNavigationHeadingVisibleProse(t *testing.T) {
	for _, prefix := range []string{"Package", "Command", "Type", "Function"} {
		source := `<section id="references"><h3 id="entry"><a href="/fixed/" aria-describedby="description">` + prefix + ` Documentation <code>ItemID</code></a></h3></section>` + "\n"
		d, raw := formalFixture(t, "navigation.html", source)
		if !strings.Contains(d.Units[0].Source, prefix+" Documentation") {
			t.Fatalf("visible heading prose became opaque: %q", d.Units[0].Source)
		}
		// Synthetic transport sentinel; no repository language replacement.
		target := strings.Replace(d.Units[0].Source, "Documentation", "EditableLabel", 1)
		candidate := checkFormal(t, d, raw, target, true)
		if err := (PageAdapter{d, raw}).Validate(candidate, &i18n.Glossary{Mandatory: map[string]string{}}); err != nil {
			t.Fatal(err)
		}
		for _, item := range []string{`id="entry"`, `href="/fixed/"`, `aria-describedby="description"`, `<code>ItemID</code>`} {
			if !bytes.Contains(candidate, []byte(item)) {
				t.Fatal("protected identity changed", item)
			}
		}
		for _, token := range d.Units[0].Protected {
			if token.Raw == "Documentation" {
				t.Fatal("visible label locked")
			}
			for _, mutation := range []string{"", token.Token + token.Token, token.Raw, "⟦P99999⟧"} {
				checkFormal(t, d, raw, strings.Replace(target, token.Token, mutation, 1), false)
			}
		}
		// The formal output is protected raw, never a restored candidate.
		// Attempting to supply changed machine bytes instead of their token
		// must fail at restore before canonical candidate validation.
		for _, mutation := range [][2]string{
			{"/fixed/", "/other/"}, {`id="entry"`, `id="other"`},
			{"description", "other"}, {"<h3", "<h2"}, {"ItemID", "OtherID"},
		} {
			found := false
			for _, p := range d.Units[0].Protected {
				if strings.Contains(p.Raw, mutation[0]) {
					found = true
					changed := strings.Replace(p.Raw, mutation[0], mutation[1], 1)
					checkFormal(t, d, raw, strings.Replace(target, p.Token, changed, 1), false)
				}
			}
			if !found {
				t.Fatal("negative control did not touch a protected identity", mutation)
			}
		}
	}
}

func TestNavigationHeadingLinkBinding(t *testing.T) {
	d, source := formalFixture(t, "navigation.html", "<h3><a href=\"/one\">Package Documentation</a></h3>\n<h3><a href=\"/two\">Command Documentation</a></h3>\n")
	links := []string{}
	for _, p := range d.Units[0].Protected {
		if strings.Contains(p.Raw, "<a href=") {
			links = append(links, p.Token)
		}
	}
	if len(links) != 2 {
		t.Fatal("exact link controls missing")
	}
	swapped := strings.NewReplacer(links[0], links[1], links[1], links[0]).Replace(d.Units[0].Source)
	checkFormal(t, d, source, swapped, false)
	// Unrelated structure/prose cannot be introduced through an opaque boundary.
	boundary := d.Units[0].Protected[0].Token
	checkFormal(t, d, source, strings.Replace(d.Units[0].Source, boundary, "Unrelated paragraph"+boundary, 1), false)
}

func TestNavigationHeadingMachineIdentities(t *testing.T) {
	for _, item := range []string{"API", "sql.DB", "GOPATH", "--race", "main.go", "https://example.test/fixed"} {
		input, tokens, err := protectBlock([]byte(`<a href="/fixed">Package `+item+`</a>`), "html/h3")
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, p := range tokens {
			found = found || p.Raw == item
		}
		if !found || strings.Contains(input, item) {
			t.Fatal("heading role released explicit machine identity", item, input)
		}
	}
}
