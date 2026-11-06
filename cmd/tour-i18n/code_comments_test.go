package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func TestCommentAuditCanonicalInventoryOrdering(t *testing.T) {
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
	a, err := commentSourceInventory(c)
	if err != nil {
		t.Fatal(err)
	}
	b, err := commentSourceInventory(c)
	if err != nil {
		t.Fatal(err)
	}
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	if string(x) != string(y) {
		t.Fatal("source inventory nondeterministic")
	}
	seen := map[string]bool{}
	slots := 0
	for _, u := range a {
		for _, block := range u.Blocks {
			for _, s := range block.Comments {
				if s.SlotID == "" || seen[s.SlotID] {
					t.Fatal("slot identity missing/duplicate", s.SlotID)
				}
				seen[s.SlotID] = true
				slots++
			}
		}
	}
	if slots == 0 {
		t.Fatal("empty inventory")
	}
	locales, err := commentRoadmapLocales(root)
	if err != nil || len(locales) == 0 || locales[0] != "zh-CN" {
		t.Fatal(locales, err)
	}
	// Missing package scope is absence, not a translation or currentness pass.
	p := auditCommentPackage(t.TempDir(), c, "zh-CN", "zh", "learn-docs-v1", a)
	if p.State != "NOT_ACCEPTED" || len(p.Rows) != 0 {
		t.Fatal(p)
	}
}
