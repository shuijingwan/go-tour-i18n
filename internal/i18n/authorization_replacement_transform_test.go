package i18n_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// Exercise the representation resolver used by authorization, including a
// changed expected SHA at the same path and mutations after a warm lookup.
func TestAuthorizationReplacementTransformGates(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root := t.TempDir()
	source := []byte("# Heading\n\nUse `fmt.Println` to print the message.\n")
	d, err := sitecontent.ParseWorkflowDocument(sitecontent.Source{Path: "_content/doc/cache.md", Kind: "page", Route: "/doc/cache", SHA256: contentidentity.SHA256Hex(source)}, source, 1)
	if err != nil {
		t.Fatal(err)
	}
	original := sitecontent.HistoricalPageRepresentations{Package: "cache-fixture", PackageSHA256: contentidentity.SHA256Hex([]byte("package authority")), Documents: []sitecontent.Document{*d}}
	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	ref := i18n.GlossaryArchiveReference{Path: "frozen.json", SHA256: contentidentity.SHA256Hex(encoded)}
	write := func(b []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, ref.Path), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	catalog := func(document sitecontent.Document, raw []byte) *i18n.Catalog {
		t.Helper()
		c := &i18n.Catalog{}
		binding := i18n.AdaptedTranslationUnit{Unit: i18n.TranslationUnit{ID: document.Units[0].ID, Kind: i18n.UnitKindPage, SourcePath: document.Path, SourceSHA256: document.SourceSHA, Source: raw}, Adapter: sitecontent.PageAdapter{Document: document, Source: raw}, StableIndex: 1, Route: document.Route}
		if err := c.RegisterTranslationPackage(i18n.TranslationPackageDescriptor{ID: original.Package, AuthoritySHA256: original.PackageSHA256, Units: []i18n.AdaptedTranslationUnit{binding}}); err != nil {
			t.Fatal(err)
		}
		if err := c.UseTranslationPackage(original.Package); err != nil {
			t.Fatal(err)
		}
		sitecontent.AttachPageReplacementResolver(c)
		return c
	}
	write(encoded)
	c := catalog(*d, bytes.Clone(source))
	first, err := c.ReplacementResolver(root, c, ref)
	if err != nil {
		t.Fatal(err)
	}
	// Returned mutable objects must never contaminate subsequent resolutions.
	first.Units[d.Units[0].ID] = i18n.TranslationUnitReplacement{CurrentUnitID: "forged"}
	old, err := first.HistoricalCatalog.Unit(d.Units[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	old.Source[0] = '!'
	for _, kind := range []string{"unchanged", "expected SHA", "reference bytes", "source bytes", "source SHA", "route", "index", "old ID", "current ID", "frozen slot", "package", "omitted slice"} {
		t.Run(kind, func(t *testing.T) {
			write(encoded)
			var f sitecontent.HistoricalPageRepresentations
			if err := json.Unmarshal(encoded, &f); err != nil {
				t.Fatal(err)
			}
			var doc sitecontent.Document
			b, _ := json.Marshal(d)
			if err := json.Unmarshal(b, &doc); err != nil {
				t.Fatal(err)
			}
			raw := bytes.Clone(source)
			checkRef := ref
			switch kind {
			case "expected SHA":
				checkRef.SHA256 = strings.Repeat("0", 64)
			case "reference bytes":
				write(append(bytes.Clone(encoded), '\n'))
			case "source bytes":
				raw[5] = '!'
			case "source SHA":
				doc.SourceSHA = strings.Repeat("0", 64)
			case "route":
				f.Documents[0].Route = "/wrong-route"
			case "index":
				f.Documents[0].StableIndex = 2
			case "old ID":
				f.Documents[0].Units[0].ID = strings.Repeat("0", 64)
			case "frozen slot":
				f.Documents[0].Slots[0].End++
			case "package":
				f.PackageSHA256 = strings.Repeat("0", 64)
			case "omitted slice":
				doc.ReferenceDefinitions = []sitecontent.MarkdownReferenceDefinition{}
			}
			if kind == "route" || kind == "index" || kind == "old ID" || kind == "frozen slot" || kind == "package" {
				b, err := json.Marshal(f)
				if err != nil {
					t.Fatal(err)
				}
				write(b)
				checkRef.SHA256 = contentidentity.SHA256Hex(b)
			}
			c := catalog(*d, bytes.Clone(source))
			c.TranslationPackages[0].Units[0].Adapter = sitecontent.PageAdapter{Document: doc, Source: raw}
			if kind == "current ID" {
				c.TranslationPackages[0].Units[0].Unit.ID = strings.Repeat("0", 64)
			}
			got, err := c.ReplacementResolver(root, c, checkRef)
			if kind == "unchanged" {
				if err != nil || got.Units[d.Units[0].ID].CurrentUnitID != d.Units[0].ID {
					t.Fatal("cached representation changed", got, err)
				}
			} else if err == nil {
				t.Fatal("warm representation concealed changed authority", kind)
			}
		})
	}
}
