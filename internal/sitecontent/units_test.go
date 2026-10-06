package sitecontent

import (
	"testing"
)

func TestSiteUnitsFrozenInventory(t *testing.T) {
	g := repositoryGlobal(t)
	docs, raw, err := PackageDocuments(repositoryRoot(t), g, "learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	units := 0
	for _, d := range docs {
		counts[d.Kind]++
		units += len(d.Units)
		if len(d.Units) == 0 {
			t.Fatalf("no units: %s", d.Path)
		}
		targets := map[string]string{}
		for _, u := range d.Units {
			targets[u.ID] = u.Source
		}
		s := documentSource(d)
		b, err := Reconstruct(s, raw[d.Path], targets)
		if err != nil {
			t.Fatalf("%s: %v", d.Path, err)
		}
		if string(b) != string(raw[d.Path]) {
			t.Fatal("identity reconstruction changed")
		}
	}
	if counts["page"] != 49 || counts["data"] != 7 {
		t.Fatal(counts)
	}
	t.Logf("documents=56 units=%d", units)
}
