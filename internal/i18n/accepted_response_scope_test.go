package i18n

import (
	"fmt"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestAcceptedResponseMembershipScope(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, old, original, plan, _, _ := authorizationMachineFixtureSize(t, 49)
	resolution, err := old.ReplacementResolver(root, old, plan.Receipt.Options.HistoricalDocuments)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := FinalizeQualityCheck(root, resolution.HistoricalCatalog, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: original.SnapshotID}); err != nil {
		t.Fatal(err)
	}
	s := futureSnapshot(t, root, old, "accepted-response-old")
	if _, _, err := FinalizeQualityCheck(root, old, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: s.SnapshotID, PreviousSnapshotID: original.SnapshotID}); err != nil {
		t.Fatal(err)
	}
	current, added := expandedMachinePackage(t, root, old, 2)
	decl := TranslationPackageMembership{Predecessors: []TranslationPackagePredecessor{
		{ID: old.TranslationPackages[0].ID, AuthoritySHA256: old.TranslationPackages[0].AuthoritySHA256},
		{ID: added.ID, AuthoritySHA256: added.AuthoritySHA256},
	}}
	data := mustJSON(decl)
	writeSnapshotSource(t, root, "fixtures/accepted-response-membership.json", data)
	decl.References = []GlossaryArchiveReference{{Path: "fixtures/accepted-response-membership.json", SHA256: sum(data)}}
	current.TranslationPackages[0].Membership = &decl
	u := s.Units[0]
	record := RetranslationBatchUnit{UnitID: u.UnitID, UnitKind: u.UnitKind, SourcePath: u.SourcePath, SourceSHA256: u.SourceSHA256, PreviousSnapshotID: s.SnapshotID}
	t.Run("explicit-historical-world-with-retained-eras", func(t *testing.T) {
		copy := *old
		era := old.TranslationPackages[0]
		era.AuthoritySHA256 = sum([]byte("another retained representation era"))
		copy.HistoricalTranslationPackages = append(append([]TranslationPackageDescriptor(nil), old.HistoricalTranslationPackages...), era)
		h, check, err := acceptedResponseCatalog(root, &copy, "zh-CN", record)
		if err != nil || h != &copy {
			t.Fatal("explicit historical world was rediscovered", err)
		}
		if err := check(); err != nil {
			t.Fatal(err)
		}
	})
	// These are context-owner tests, not manufactured Generation invocations.
	// Exact real055/054/051 restoration is separately covered by the bounded
	// production-path diagnostic, using their unchanged manifests/raw bytes.
	for _, name := range []string{"batch055-like", "batch054-like", "batch051-like"} {
		t.Run(name, func(t *testing.T) {
			r := record
			r.RevisionFeedbackSource = "quality_check"
			if name == "batch051-like" {
				r.RevisionFeedbackSource = CurrentValidationRevisionFeedback
			}
			h, check, err := acceptedResponseCatalog(root, current, "zh-CN", r)
			if err != nil || len(h.TranslationScope) != 49 || len(current.TranslationScope) != 51 {
				t.Fatal("historical/current world routing", err)
			}
			if _, err := readQualityCheckSnapshot(root, "zh-CN", s.SnapshotID, false, h); err != nil {
				t.Fatal(err)
			}
			if name == "batch051-like" {
				if _, _, err := SelectFinalizedTranslationPackage(root, h, "zh-CN", s.SnapshotID); err != nil {
					t.Fatal(err)
				}
			}
			if err := check(); err != nil {
				t.Fatal(err)
			}
		})
	}
	// The strict original readers are deliberately unchanged. A historical
	//49-Snapshot never becomes a complete current51-Snapshot or QC plan.
	if _, err := readQualityCheckSnapshot(root, "zh-CN", s.SnapshotID, false, current); err == nil {
		t.Fatal("initial complete-scope gate weakened")
	}
	if _, _, err := SelectFinalizedTranslationPackage(root, current, "zh-CN", s.SnapshotID); err == nil {
		t.Fatal("explicit package scope gate weakened")
	}
	if _, err := PrepareQualityCheckCandidateSnapshot(root, current, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "incomplete-current", ContentPackage: current.TranslationPackages[0].ID}); err == nil {
		t.Fatal("missing generation for the two new members accepted")
	}
	for _, name := range []string{"missing-membership", "partial-explicit", "wrong-explicit", "tampered-membership", "historical-batch-source", "historical-batch-scope", "concurrent-predecessor"} {
		t.Run(name, func(t *testing.T) {
			r := cloneCompositionFixture(t, root)
			c := *current
			c.TranslationPackages = append([]TranslationPackageDescriptor(nil), current.TranslationPackages...)
			rec := record
			switch name {
			case "missing-membership":
				c.TranslationPackages[0].Membership = nil
			case "partial-explicit":
				c.TranslationScope = c.TranslationScope[:50]
			case "wrong-explicit":
				h, err := HistoricalTranslationPackageCatalog(current, added.ID, added.AuthoritySHA256)
				if err != nil {
					t.Fatal(err)
				}
				c = *h
			case "tampered-membership":
				writeSnapshotSource(t, r, "fixtures/accepted-response-membership.json", []byte("{}"))
			case "historical-batch-source":
				rec.SourceSHA256 = sum([]byte("wrong source"))
			case "historical-batch-scope":
				rec.PreviousSnapshotID = original.SnapshotID // Old representation IDs are not this same-ID response chain.
			}
			_, check, err := acceptedResponseCatalog(r, &c, "zh-CN", rec)
			if name == "concurrent-predecessor" {
				if err != nil {
					t.Fatal(err)
				}
				path := "data/quality-check-snapshots/zh-CN/" + s.SnapshotID + "/manifest.json"
				b, e := readSnapshotReferencedFile(r, path)
				if e != nil {
					t.Fatal(e)
				}
				writeSnapshotSource(t, r, path, append(b, ' '))
				if check() == nil {
					t.Fatal("predecessor mutation not detected")
				}
				return
			}
			if err == nil {
				t.Fatal("wrong scope accepted", name)
			}
		})
	}
	g, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	unit, err := current.Unit(u.UnitID)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := readSnapshotReferencedFile(root, u.CandidatePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateTranslationUnitWithGlossary(root, current, unit, "zh-CN", candidate, g); err != nil {
		t.Fatal("current51 validator failed", err)
	}
	if err := validateTranslationUnitWithGlossary(root, current, unit, "zh-CN", []byte("broken candidate"), g); err == nil {
		t.Fatal("bad candidate accepted")
	}
	// Current validation cannot be replaced by a historical decoder's PASS.
	changed := *unit
	changed.Adapter = acceptedResponseRejectAdapter{unit.Adapter}
	x, err := assessCurrentTranslationUnit(root, current, "zh-CN", &changed, u, s.GlossarySHA256, mustReadResponseGlossary(t, root), g, map[string]*machineSelection{})
	if err != nil || x.Classification != "affected" || x.Validation != "failed" {
		t.Fatal("changed rejecting validator bypassed", x, err)
	}
}

type acceptedResponseRejectAdapter struct{ TranslationUnitAdapter }

func (a acceptedResponseRejectAdapter) ReattestationAuthority(string) (string, error) {
	return sum([]byte("rejecting-current-validator")), nil
}
func (a acceptedResponseRejectAdapter) GlossaryVisiblePair([]byte) (string, string, error) {
	return "source", "target", nil
}
func (a acceptedResponseRejectAdapter) Validate([]byte, *Glossary) error {
	return fmt.Errorf("current validator epoch rejects this candidate")
}
func mustReadResponseGlossary(t *testing.T, root string) []byte {
	t.Helper()
	b, e := readCompatibilityFile(root, "locales/zh-CN/glossary.yaml")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
