package i18n

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestSuccessorSnapshotDescriptorBoundary(t *testing.T) {
	catalog := &Catalog{}
	source := []byte("<card>source</card>\n")
	descriptor := TranslationPackageDescriptor{ID: "independent-content", AuthoritySHA256: sum([]byte("frozen authority"))}
	for i, id := range []string{"card-one", "card-two"} {
		descriptor.Units = append(descriptor.Units, AdaptedTranslationUnit{
			Unit:    TranslationUnit{ID: id, Kind: UnitKindPage, SourcePath: "fixtures/" + id + ".xml", Source: source, SourceSHA256: sum(source)},
			Adapter: futureCardAdapter{}, StableIndex: i + 1, Route: "/" + id,
		})
	}
	if err := catalog.RegisterTranslationPackage(descriptor); err != nil {
		t.Fatal(err)
	}
	expected := []*TranslationUnit{{ID: "lesson/1", Kind: UnitKindPage, SourcePath: "_content/tour/lesson.article"}}
	snapshot := QualityCheckSnapshotManifest{}
	for _, registered := range descriptor.Units {
		u := registered.Unit
		snapshot.Units = append(snapshot.Units, QualityCheckSnapshotUnit{UnitID: u.ID, UnitKind: u.Kind, SourcePath: u.SourcePath, SourceSHA256: u.SourceSHA256})
	}
	if !unrelatedSuccessorSnapshot(&snapshot, expected, catalog) {
		t.Fatal("complete independent descriptor not excluded")
	}
	for _, mutation := range []string{"partial", "unknown", "source", "kind", "representation", "overlap-id", "overlap-source", "unbound-authority"} {
		t.Run(mutation, func(t *testing.T) {
			s := snapshot
			s.Units = append([]QualityCheckSnapshotUnit(nil), snapshot.Units...)
			c := *catalog
			c.TranslationPackages = append([]TranslationPackageDescriptor(nil), catalog.TranslationPackages...)
			switch mutation {
			case "partial":
				s.Units = s.Units[:1]
			case "unknown":
				s.Units[0].UnitID = "unknown"
			case "source":
				s.Units[0].SourceSHA256 = sum([]byte("different source"))
			case "kind":
				s.Units[0].UnitKind = UnitKindExample
			case "representation":
				s.Units[0].PageSection = &QualityCheckPageSectionIdentity{Article: "lesson.article", SectionNumber: 1}
			case "overlap-id":
				s.Units[0].UnitID = expected[0].ID
			case "overlap-source":
				s.Units[0].SourcePath = expected[0].SourcePath
			case "unbound-authority":
				c.TranslationPackages[0].AuthoritySHA256 = ""
			}
			if unrelatedSuccessorSnapshot(&s, expected, &c) {
				t.Fatal("unproven or overlapping inventory bypassed strict successor checks")
			}
		})
	}
}

func TestHistoricalTUCompatibilityRecordedInventory(t *testing.T) {
	root, cat, _ := historicalRecoveryFixture(t)
	e, path, err := RecoverHistoricalPromotion(root, cat, "zh-CN", "qc-002", true)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := readCompatibilityFile(root, path)
	closure, _ := readCompatibilityFile(root, "locales/zh-CN/content-scope.json")
	for _, input := range []string{"internal/tour/ui/zh-CN.json", "internal/tour/ui/en.json", "locales/zh-CN/course-metadata.json", "locales/zh-CN/article-metadata.json", "internal/tour/languages.go", "internal/tour/project.go", "internal/tour/seo.go", "internal/tour/production.go", "_content/js/playground.js", "_content/tour/static/js/app.js", "_content/tour/template/index.tmpl", "_content/tour/static/partials/index.html"} {
		t.Run(input, func(t *testing.T) {
			b, err := readCompatibilityFile(root, input)
			if err != nil {
				t.Fatal(err)
			}
			writeSnapshotSource(t, root, input, append(append([]byte(nil), b...), '\n'))
			defer writeSnapshotSource(t, root, input, b)
			proof, err := verifyTargetSuccessor(root, path, cat)
			if err != nil {
				t.Fatal(err)
			}
			if err := requireSuccessorCurrentLanguage(root, cat, proof); err != nil {
				t.Fatal(err)
			}
			if err := AuthorizeHistoricalCanonicalReference(root, "zh-CN", input, sum(b), cat); err == nil {
				t.Fatal("TU event authorized derived consumer")
			}
			if r := proof.compatibility.Resolve(e.GlossarySHA, e.GlossarySHA, "ui:test.one"); r.Status == "exact" || r.Status == "compatible" {
				t.Fatal("TU-only resolver leaked derived authority")
			}
		})
	}
	after, _ := readCompatibilityFile(root, path)
	if !bytes.Equal(before, after) {
		t.Fatal("receipt changed")
	}
	after, _ = readCompatibilityFile(root, "locales/zh-CN/content-scope.json")
	if !bytes.Equal(closure, after) {
		t.Fatal("closure changed")
	}
}

func TestCanonicalAuthorizationKeepsNonTUSurfaceBoundary(t *testing.T) {
	root, cat, _ := historicalRecoveryFixture(t)
	e, _, err := RecoverHistoricalPromotion(root, cat, "zh-CN", "qc-002", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, cat, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-003"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := FinalizeQualityCheck(root, cat, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "qc-003", PreviousSnapshotID: "qc-002"}); err != nil {
		t.Fatal(err)
	}
	ui := "internal/tour/ui/zh-CN.json"
	oldUI, err := readCompatibilityFile(root, ui)
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotSource(t, root, ui, append(bytes.Clone(oldUI), '\n'))
	if _, _, err := RequireCurrentCanonicalQualityCheck(root, "zh-CN", cat); err != nil {
		t.Fatalf("non-TU UI revoked canonical authorization: %v", err)
	}
	d, err := DiagnoseQualityCheckAuthorization(root, cat, "zh-CN", "qc-003")
	if err != nil || !d.CanonicalAuthorized || d.ReviewerACount != 3 {
		t.Fatalf("current finalized successor diagnostic: %+v %v", d, err)
	}
	if err := AuthorizeHistoricalCanonicalReference(root, "zh-CN", ui, sum(oldUI), cat); err == nil {
		t.Fatal("TU successor authorized UI")
	}
	// A different historical A candidate set cannot borrow the current receipt.
	if d, err := DiagnoseQualityCheckAuthorization(root, cat, "zh-CN", "qc-001"); err == nil && d.CanonicalAuthorized {
		t.Fatal("different historical canonical identity authorized")
	}
	writeSnapshotSource(t, root, e.Units[0].CanonicalPath, []byte("* unauthorized canonical mutation\n"))
	if _, _, err := RequireCurrentCanonicalQualityCheck(root, "zh-CN", cat); err == nil {
		t.Fatal("canonical mutation without successor authorized")
	}
	if d, err := DiagnoseQualityCheckAuthorization(root, cat, "zh-CN", "qc-003"); err == nil && d.CanonicalAuthorized {
		t.Fatal("diagnostic authorized canonical mutation without successor")
	}
}

func TestHistoricalTUEventRetainsNonTUProofInputsExactSet(t *testing.T) {
	root, cat, _ := historicalRecoveryFixture(t)
	e, _, err := RecoverHistoricalPromotion(root, cat, "zh-CN", "qc-002", true)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []string{"remove", "change", "add"} {
		t.Run(mutation, func(t *testing.T) {
			copy := *e
			copy.ProofInputs = append([]GlossaryArchiveReference(nil), e.ProofInputs...)
			for i, ref := range copy.ProofInputs {
				if ref.Path != "internal/tour/ui/en.json" {
					continue
				}
				switch mutation {
				case "remove":
					copy.ProofInputs = append(copy.ProofInputs[:i], copy.ProofInputs[i+1:]...)
				case "change":
					copy.ProofInputs[i].SHA256 = sum([]byte("invented historical context"))
				case "add":
					copy.ProofInputs = append(copy.ProofInputs, GlossaryArchiveReference{Path: "internal/tour/invented-context.go", SHA256: ref.SHA256})
				}
				break
			}
			copy.Identity = successorIdentity(copy)
			path, err := successorPath(&copy)
			if err != nil {
				t.Fatal(err)
			}
			b, _ := json.MarshalIndent(&copy, "", "  ")
			writeSnapshotSource(t, root, path, append(b, '\n'))
			if _, err := verifyTargetSuccessor(root, path, cat); err == nil {
				t.Fatal("reidentified receipt changed non-TU recorded exact-set")
			}
		})
	}
}

func TestHistoricalTUEventCriticalInputsRemainExact(t *testing.T) {
	root, cat, compat := historicalRecoveryFixture(t)
	e, path, err := RecoverHistoricalPromotion(root, cat, "zh-CN", "qc-002", true)
	if err != nil {
		t.Fatal(err)
	}
	inputs := []string{path, "locales/zh-CN/content-scope.json", compat, e.Snapshot.Path, e.Finalization.Path, "data/quality-check-snapshots/zh-CN/qc-001/manifest.json", "data/quality-check-snapshots/zh-CN/qc-001/finalization.json", e.Units[0].CanonicalPath}
	for _, ref := range e.ProofInputs {
		if len(ref.Path) >= 5 && ref.Path[:5] == "data/" {
			inputs = append(inputs, ref.Path)
		}
	}
	inputs = append(inputs, e.Units[0].SourcePath)
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			b, err := readCompatibilityFile(root, input)
			if err != nil {
				t.Fatal(err)
			}
			writeSnapshotSource(t, root, input, append(append([]byte(nil), b...), '\n'))
			defer writeSnapshotSource(t, root, input, b)
			if _, err := verifyTargetSuccessor(root, path, cat); err == nil {
				t.Fatal("changed critical input authorized")
			}
		})
	}
}
