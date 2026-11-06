package i18n

import (
	"bytes"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestFinalizedNativeForeignScopeBoundary(t *testing.T) {
	c := &Catalog{Pages: []Page{
		{ID: "lesson/1", Article: "lesson.article", SectionNumber: 1, Route: "/lesson/1", SourceTitle: "First", SourceSHA256: sum([]byte("first"))},
		{ID: "lesson/2", Article: "lesson.article", SectionNumber: 2, Route: "/lesson/2", SourceTitle: "Second", SourceSHA256: sum([]byte("second"))},
	}}
	s := QualityCheckSnapshotManifest{}
	for i, p := range c.Pages {
		u, _ := c.Unit(p.ID)
		s.Units = append(s.Units, QualityCheckSnapshotUnit{Index: i + 1, UnitID: u.ID, UnitKind: u.Kind, SourcePath: u.SourcePath, SourceSHA256: u.SourceSHA256, PageSection: &QualityCheckPageSectionIdentity{Article: p.Article, SectionNumber: p.SectionNumber, SourceTitle: p.SourceTitle, Route: p.Route}})
	}
	expected := []*TranslationUnit{{ID: "registered/1", Kind: UnitKindPage, SourcePath: "fixtures/registered.xml"}}
	if !unrelatedSuccessorSnapshot(&s, expected, c) {
		t.Fatal("complete disjoint native workflow not recognized")
	}
	for _, name := range []string{"partial", "wrong-id", "wrong-order", "source", "kind", "section", "overlap-id", "overlap-path"} {
		t.Run(name, func(t *testing.T) {
			x := s
			x.Units = append([]QualityCheckSnapshotUnit(nil), s.Units...)
			switch name {
			case "partial":
				x.Units = x.Units[:1]
			case "wrong-id":
				x.Units[0].UnitID = "unknown/1"
			case "wrong-order":
				x.Units[0], x.Units[1] = x.Units[1], x.Units[0]
			case "source":
				x.Units[0].SourceSHA256 = sum([]byte("changed"))
			case "kind":
				x.Units[0].UnitKind = UnitKindExample
			case "section":
				p := *x.Units[0].PageSection
				p.Route = "/wrong"
				x.Units[0].PageSection = &p
			case "overlap-id":
				x.Units[0].UnitID = expected[0].ID
			case "overlap-path":
				x.Units[0].SourcePath = expected[0].SourcePath
			}
			if unrelatedSuccessorSnapshot(&x, expected, c) {
				t.Fatal("nonexact/overlapping native scope silently skipped")
			}
		})
	}
	u, _ := c.Unit(c.Pages[0].ID)
	if unrelatedSuccessorSnapshot(&s, []*TranslationUnit{u}, c) {
		t.Fatal("same/partial-overlap scope treated as foreign")
	}
}

func TestFinalizedForeignMachineScopeWithoutFinalizationEdge(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, current, oldSnapshot, plan, receiptRef, receiptBytes := authorizationMachineFixture(t)
	resolution, err := replacementResolution(root, current, &TranslationUnitReplacementContext{HistoricalDocuments: plan.Receipt.Options.HistoricalDocuments})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := FinalizeQualityCheck(root, resolution.HistoricalCatalog, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: oldSnapshot.SnapshotID}); err != nil {
		t.Fatal(err)
	}
	endpoint := futureSnapshot(t, root, current, "independent-machine-finalization")
	ids := []string{}
	for _, u := range endpoint.Units {
		ids = append(ids, u.UnitID)
	}
	for _, id := range ids {
		futureRecord(t, root, current, endpoint.SnapshotID, "", "A", "", []string{id})
	}
	f, _, err := FinalizeQualityCheck(root, current, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: endpoint.SnapshotID})
	if err != nil || f.PredecessorFinalization != nil {
		t.Fatal("fresh endpoint must have no finalization predecessor", err)
	}
	expected := []*TranslationUnit{{ID: "native/1", Kind: UnitKindPage, SourcePath: "fixtures/native.article"}}
	if exclude, err := excludeUnrelatedSuccessorSnapshot(root, oldSnapshot, expected, current, map[string]bool{}); err != nil || !exclude {
		t.Fatal("exact foreign machine scope not proven", exclude, err)
	}
	if checkpoint, err := findScopeCurrentAuthorization(root, current, "zh-CN"); err != nil || checkpoint != nil {
		t.Fatal("read-only scope proof must not persist an authorization", err)
	}
	for name, path := range map[string]string{"receipt": receiptRef.Path, "validation": plan.Receipt.Units[0].Validation.Path, "candidate": plan.Receipt.Units[0].Candidate.Path} {
		t.Run(name, func(t *testing.T) {
			b, err := readSnapshotReferencedFile(root, path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeSnapshotSource(t, root, path, b)
			writeSnapshotSource(t, root, path, append(bytes.Clone(b), []byte("tamper")...))
			if exclude, err := excludeUnrelatedSuccessorSnapshot(root, oldSnapshot, expected, current, map[string]bool{}); err == nil && exclude {
				t.Fatal("tampered machine proof excluded foreign Snapshot")
			}
		})
	}
	if b, _ := readSnapshotReferencedFile(root, receiptRef.Path); !bytes.Equal(b, receiptBytes) {
		t.Fatal("fixture receipt not restored")
	}
	for _, name := range []string{"partial", "wrong-old", "wrong-order", "source-drift", "overlap", "missing-machine-proof"} {
		t.Run(name, func(t *testing.T) {
			s := *oldSnapshot
			s.Units = append([]QualityCheckSnapshotUnit(nil), s.Units...)
			switch name {
			case "partial":
				s.Units = s.Units[:1]
			case "wrong-old":
				s.Units[0].UnitID = sum([]byte("unproven old ID"))
			case "wrong-order":
				s.Units[0], s.Units[1] = s.Units[1], s.Units[0]
			case "source-drift":
				s.Units[0].SourceSHA256 = sum([]byte("different source"))
			case "overlap":
				s.Units[0].SourcePath = expected[0].SourcePath
			case "missing-machine-proof":
				// Preserve a valid current descriptor but remove its only exact
				// applied authority from this synthetic filesystem.
				writeSnapshotSource(t, root, receiptRef.Path, []byte("missing"))
				defer writeSnapshotSource(t, root, receiptRef.Path, receiptBytes)
			}
			if exclude, err := excludeUnrelatedSuccessorSnapshot(root, &s, expected, current, map[string]bool{}); err == nil && exclude {
				t.Fatal("unproven/ambiguous old membership excluded")
			}
		})
	}
}
