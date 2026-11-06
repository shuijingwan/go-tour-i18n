package main

import (
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func TestCommentResidualCanonicalExactSet(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	c, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = sitecontent.AttachPageTranslationUnits(root, c); err != nil {
		t.Fatal(err)
	}
	base, err := commentSourceInventory(c)
	if err != nil {
		t.Fatal(err)
	}
	closed, records, err := closeCommentInventory(c, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 432 {
		t.Fatal("source block exact set changed", len(records))
	}
	counts := map[string]int{}
	for _, record := range records {
		counts[record.Disposition]++
		if record.Disposition == "ambiguous-unsupported" {
			t.Fatalf("remaining %s %s: %q context=%q", record.SourcePath, record.Coordinate, record.FirstLine, func() string {
				for _, u := range closed {
					if u.ID == record.ParentID {
						for _, b := range u.Blocks {
							if b.Coordinate == record.Coordinate {
								return b.SourceContext
							}
						}
					}
				}
				return ""
			}())
		}
	}
	t.Logf("dispositions=%v counts=%v", counts, commentInventoryCounts(closed))
}
