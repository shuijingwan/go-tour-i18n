package i18n

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPreviewSyntheticChanges(t *testing.T) {
	base := &Catalog{Pages: []Page{
		testPage("lesson/1", "lesson.article", 1, "/lesson/1", "One", "first body\n.play demo.go\n"),
		testPage("lesson/2", "lesson.article", 2, "/lesson/2", "Two", "second body `x`\n"),
		testPage("lesson/3", "lesson.article", 3, "/lesson/3", "Three", "third body\n.image pic.png\n"),
	}}
	tests := []struct {
		name string
		next *Catalog
		want map[ChangeKind]int
	}{
		{"unchanged", cloneCatalog(base), map[ChangeKind]int{Unchanged: 3}},
		{"body changed", &Catalog{Pages: []Page{
			testPage("temporary/1", "lesson.article", 1, "/lesson/1", "One", "translated ordinary body\n.play demo.go\n"), base.Pages[1], base.Pages[2],
		}}, map[ChangeKind]int{Unchanged: 2, ContentChanged: 1}},
		{"title changed", &Catalog{Pages: []Page{
			testPage("temporary/1", "lesson.article", 1, "/lesson/1", "Renamed", "first body\n.play demo.go\n"), base.Pages[1], base.Pages[2],
		}}, map[ChangeKind]int{Unchanged: 2, ContentChanged: 1}},
		{"same article move", &Catalog{Pages: []Page{
			at(base.Pages[1], "lesson.article", 1, "/lesson/1"), at(base.Pages[0], "lesson.article", 2, "/lesson/2"), base.Pages[2],
		}}, map[ChangeKind]int{Unchanged: 1, Moved: 2}},
		{"cross article move", &Catalog{Pages: []Page{
			at(base.Pages[0], "other.article", 1, "/other/1"), base.Pages[1], base.Pages[2],
		}}, map[ChangeKind]int{Unchanged: 2, Moved: 1}},
		{"middle insertion", &Catalog{Pages: []Page{
			base.Pages[0], testPage("temporary/2", "lesson.article", 2, "/lesson/2", "New", "brand new\n"),
			at(base.Pages[1], "lesson.article", 3, "/lesson/3"), at(base.Pages[2], "lesson.article", 4, "/lesson/4"),
		}}, map[ChangeKind]int{Unchanged: 1, Moved: 2, Added: 1}},
		{"deletion", &Catalog{Pages: []Page{base.Pages[0], at(base.Pages[2], "lesson.article", 2, "/lesson/2")}}, map[ChangeKind]int{Unchanged: 1, Moved: 1, Removed: 1}},
		{"move and edit", &Catalog{Pages: []Page{
			testPage("temporary/1", "other.article", 1, "/other/1", "One changed", "different body\n.play demo.go\n"), base.Pages[1], base.Pages[2],
		}}, map[ChangeKind]int{Unchanged: 2, Ambiguous: 1}},
		{"protected change", &Catalog{Pages: []Page{
			testPage("temporary/1", "lesson.article", 1, "/lesson/1", "One", "first body\n.play other.go\n"), base.Pages[1], base.Pages[2],
		}}, map[ChangeKind]int{Unchanged: 2, Ambiguous: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := PreviewCatalog(base, tt.next)
			if err != nil {
				t.Fatal(err)
			}
			for _, kind := range ChangeKinds {
				if got := report.Count(kind); got != tt.want[kind] {
					t.Fatalf("%s=%d, want %d; changes=%+v", kind, got, tt.want[kind], report.Changes)
				}
			}
		})
	}
}

func TestPreviewDuplicateStructureIsAmbiguous(t *testing.T) {
	old := &Catalog{Pages: []Page{
		testPage("x/1", "x.article", 1, "/x/1", "A", "old a\n.play same.go\n"),
		testPage("x/2", "x.article", 2, "/x/2", "B", "old b\n.play same.go\n"),
	}}
	next := &Catalog{Pages: []Page{
		testPage("temporary/1", "x.article", 1, "/x/1", "B", "new b\n.play same.go\n"),
		testPage("temporary/2", "x.article", 2, "/x/2", "A", "new a\n.play same.go\n"),
	}}
	report, err := PreviewCatalog(old, next)
	if err != nil {
		t.Fatal(err)
	}
	if report.Count(Ambiguous) != 2 || report.SafeForCatalogWrite() {
		t.Fatalf("duplicate features were not blocked: %+v", report.Changes)
	}
}

func TestConditionalChangesReportedSeparately(t *testing.T) {
	old := &Catalog{Conditional: []ConditionalPage{{Article: "welcome.article", Condition: "appengine", ConditionalIndex: 1, SourceTitle: "Old", SourceSHA256: strings.Repeat("a", 64)}}}
	next := &Catalog{Conditional: []ConditionalPage{{Article: "welcome.article", Condition: "appengine", ConditionalIndex: 1, SourceTitle: "New", SourceSHA256: strings.Repeat("b", 64)}}}
	report, err := PreviewCatalog(old, next)
	if err != nil {
		t.Fatal(err)
	}
	if report.ConditionalCount(Removed) != 1 || report.ConditionalCount(Added) != 1 || report.SafeForCatalogWrite() {
		t.Fatalf("conditional changes=%+v", report.ConditionalChanges)
	}
}

func TestPersistentIDsAndReconcile(t *testing.T) {
	root := repoRoot(t)
	committed, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, page := range committed.Pages {
		ids = append(ids, page.ID)
	}
	h := sha256.Sum256([]byte(strings.Join(ids, "\n") + "\n"))
	if got, want := hex.EncodeToString(h[:]), "76313a2fd5fd79405589925070c9fb63657f56767d279d6d4c9764c520238175"; got != want {
		t.Fatalf("frozen page_id set changed: digest=%s", got)
	}
	statuses, err := ReadStatuses(filepath.Join(root, "locales", "zh-CN", "status.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	var statusIDs []string
	for _, status := range statuses {
		unit, err := committed.Unit(status.UnitID())
		if err != nil {
			t.Fatal(err)
		}
		if unit.Kind == UnitKindPage {
			statusIDs = append(statusIDs, status.UnitID())
		}
	}
	if !reflect.DeepEqual(ids, statusIDs) {
		t.Fatal("status page IDs differ from the frozen catalog")
	}

	old := &Catalog{Pages: []Page{testPage("basics/17", "basics.article", 17, "/basics/17", "Stable", "ordinary body\n")}}
	next := &Catalog{Pages: []Page{at(old.Pages[0], "basics.article", 5, "/basics/5")}}
	report, err := PreviewCatalog(old, next)
	if err != nil {
		t.Fatal(err)
	}
	reconciled, err := ReconcileCatalog(old, next, report)
	if err != nil {
		t.Fatal(err)
	}
	if got := reconciled.Pages[0]; got.ID != "basics/17" || got.Route != "/basics/5" || got.SectionNumber != 5 {
		t.Fatalf("identity was not preserved independently of route: %+v", got)
	}
	page, err := reconciled.Page("basics/17")
	if err != nil || page.Route != "/basics/5" {
		t.Fatalf("page export lookup did not follow persistent ID: page=%+v err=%v", page, err)
	}
	if err := ValidateCandidate(root, reconciled, "basics/17", page.Source); err != nil {
		t.Fatalf("candidate lookup did not follow persistent ID: %v", err)
	}
}

func TestPreviewAndCatalogWriteDoNotTouchStatus(t *testing.T) {
	root := repoRoot(t)
	before, err := os.ReadFile(filepath.Join(root, "locales", "zh-CN", "status.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	committed, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	current, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := HydrateCatalogSources(committed, current); err != nil {
		t.Fatal(err)
	}
	report, err := PreviewCatalog(committed, current)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReconcileCatalog(committed, current, report); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(root, "locales", "zh-CN", "status.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("read-only preview or reconciliation changed status.tsv")
	}
}

func testPage(id, article string, section int, route, title, body string) Page {
	source := []byte("* " + title + "\n\n" + body)
	if source[len(source)-1] != '\n' {
		source = append(source, '\n')
	}
	return Page{ID: id, Article: article, SectionNumber: section, Route: route, SourceTitle: title, SourceSHA256: sum(source), Source: source}
}

func at(page Page, article string, section int, route string) Page {
	page.Article, page.SectionNumber, page.Route = article, section, route
	return page
}

func cloneCatalog(c *Catalog) *Catalog {
	return &Catalog{Pages: append([]Page(nil), c.Pages...), Conditional: append([]ConditionalPage(nil), c.Conditional...)}
}
