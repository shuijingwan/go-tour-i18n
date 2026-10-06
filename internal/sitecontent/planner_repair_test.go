package sitecontent

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"testing"
)

func TestSiteRepairFrozenPlanner(t *testing.T) {
	docs, raw, err := PackageDocuments(repositoryRoot(t), repositoryGlobal(t), "learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	stats, err := SummarizeUnits(docs, raw)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Punctuation != 0 {
		t.Fatal("punctuation-only language units")
	}
	b, _ := Encode(stats)
	t.Log(string(b))
	ids := selectAll(docs)
	plan, err := PlanWorkingSets(docs, raw, ids, nil)
	if err != nil {
		t.Fatal(err)
	}
	again, err := PlanWorkingSets(docs, raw, ids, nil)
	if err != nil || !reflect.DeepEqual(plan, again) {
		t.Fatal("nondeterministic plan", err)
	}
	seen := map[string]bool{}
	for _, set := range plan.Sets {
		if len(set.Selected) > 60 || set.TextBytes > MaxGenerationTextBytes || set.ContextBytes > MaxGenerationContextBytes {
			t.Fatal("unbounded plan")
		}
		for _, id := range set.Selected {
			if seen[id] {
				t.Fatal("duplicate planned unit")
			}
			seen[id] = true
		}
	}
	if len(seen) != len(ids) {
		t.Fatal("incomplete plan")
	}
	if err := requireWorkingSet(docs, raw, ids, nil); err == nil {
		t.Fatal("whole package accepted as invocation")
	}
	// Large target expansion triggers the text cap before the unit count.
	selected := plan.Sets[0].Selected[:2]
	targets := map[string]string{}
	for _, id := range selected {
		targets[id] = string(bytes.Repeat([]byte("x"), 13000))
	}
	review, err := PlanWorkingSets(docs, raw, selected, targets)
	if err != nil || len(review.Sets) != 2 {
		t.Fatal("Reviewer text boundary", err)
	}
	if p, e := DiagnosticSlotPlan(docs, raw, selected, targets); e == nil && len(p.Sets) == 1 {
		t.Fatal("oversized Reviewer invocation")
	}
	// A single linguistic block is never split to evade a cap.
	targets[selected[0]] = string(bytes.Repeat([]byte("x"), (24 << 10)))
	if _, err := PlanWorkingSets(docs, raw, selected, targets); err == nil {
		t.Fatal("oversized linguistic block accepted")
	}
}

func TestSiteRepairPlannerPageDataAndContext(t *testing.T) {
	parse := func(name, kind, body string) Document {
		d, err := ParseDocument(Source{Path: name, Kind: kind, SHA256: digest([]byte(body))}, []byte(body))
		if err != nil {
			t.Fatal(err)
		}
		return *d
	}
	body := "A page sentence.\n"
	data := "- title: A visible title\n  url: /x\n"
	docs := []Document{parse("x.md", "page", body), parse("x.yaml", "data", data)}
	raw := map[string][]byte{"x.md": []byte(body), "x.yaml": []byte(data)}
	p, err := PlanWorkingSets(docs, raw, selectAll(docs), nil)
	if err != nil || len(p.Sets) != 2 {
		t.Fatal("mixed Page/Data", err)
	}
	if err := requireWorkingSet(docs, raw, selectAll(docs), nil); err == nil {
		t.Fatal("mixed export accepted")
	}
	large := "<!--" + string(bytes.Repeat([]byte("x"), MaxGenerationContextBytes)) + "-->\n\nA sentence.\n"
	d := parse("large.md", "page", large)
	if _, err := PlanWorkingSets([]Document{d}, map[string][]byte{"large.md": []byte(large)}, selectAll([]Document{d}), nil); err == nil {
		t.Fatal("context cap bypass")
	}
	// Source ordering, not hash lexical ordering, decides working-set membership.
	if !sort.StringsAreSorted(p.Sets[0].Selected) {
		t.Fatal(fmt.Sprint(p))
	}
}
