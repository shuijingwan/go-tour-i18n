package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func expandedMachinePackage(t *testing.T, root string, c *Catalog, additions int) (*Catalog, TranslationPackageDescriptor) {
	t.Helper()
	original := c.TranslationPackages[0]
	merged := original
	merged.AuthoritySHA256 = sum([]byte("append-only-membership-successor"))
	merged.Units = append([]AdaptedTranslationUnit(nil), original.Units...)
	added := TranslationPackageDescriptor{ID: "independent-membership-addition", AuthoritySHA256: sum([]byte("independent-addition-authority"))}
	for i := 0; i < additions; i++ {
		u := original.Units[i%len(original.Units)]
		u.Unit.ID, u.Unit.SourcePath = fmt.Sprintf("addition-%d", i+1), fmt.Sprintf("fixtures/addition-%d.xml", i+1)
		u.Route, u.StableIndex = fmt.Sprintf("/fixture/addition/%d", i+1), len(merged.Units)+1
		u.WorkingSet = fmt.Sprintf("addition-%d", i+1)
		writeSnapshotSource(t, root, u.Unit.SourcePath, u.Unit.Source)
		added.Units, merged.Units = append(added.Units, u), append(merged.Units, u)
	}
	out := *c
	out.TranslationPackages = []TranslationPackageDescriptor{merged}
	out.HistoricalTranslationPackages = append(append([]TranslationPackageDescriptor(nil), c.HistoricalTranslationPackages...), original, added)
	out.Adapted = append(append([]AdaptedTranslationUnit(nil), c.Adapted...), added.Units...)
	if err := out.UseTranslationPackage(merged.ID); err != nil {
		t.Fatal(err)
	}
	return &out, added
}

func TestHistoricalReconciliationPackageExpansion(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, event, original, plan, ref, before := authorizationMachineFixtureSize(t, 49)
	resolution, err := event.ReplacementResolver(root, event, plan.Receipt.Options.HistoricalDocuments)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := FinalizeQualityCheck(root, resolution.HistoricalCatalog, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: original.SnapshotID}); err != nil {
		t.Fatal(err)
	}
	s := futureSnapshot(t, root, event, "reconciled-composition-predecessor")
	_, finalPath, err := FinalizeQualityCheck(root, event, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: s.SnapshotID, PreviousSnapshotID: original.SnapshotID})
	if err != nil {
		t.Fatal(err)
	}
	current, added := expandedMachinePackage(t, root, event, 2)
	ctx, err := prepareMachineReconciliationInWorld(root, current, plan.Receipt.Options, plan.Receipt.GlossarySHA256)
	if err != nil || len(ctx.snapshot.Units) != 49 || len(ctx.effective) != 49 || len(ctx.resolution.HistoricalCatalog.TranslationScope) != 49 || len(current.TranslationScope) != 51 {
		t.Fatal("historical/current scope separation", err)
	}
	u := plan.Receipt.Units[0].Replacement
	if _, _, err := NewRepresentationSuccessorVerifier(root, current, "zh-CN").VerifyHistorical(u.OldUnitID, u.CurrentUnitID); err != nil {
		t.Fatal("native historical receipt reader after expansion", err)
	}
	if _, err := prepareMachineReconciliation(root, current, plan.Receipt.Options); err == nil {
		t.Fatal("historical override leaked into current reconciliation creation")
	}
	if a, err := RequireCurrentTranslationAuthorization(root, current, "zh-CN", s.SnapshotID); err == nil && a.Ready() {
		t.Fatal("historical 49 authorized current 51")
	}
	finalBytes, err := readSnapshotReferencedFile(root, finalPath)
	if err != nil {
		t.Fatal(err)
	}
	inputs := []TranslationFinalizationInput{
		{Package: event.TranslationPackages[0].ID, PackageAuthority: event.TranslationPackages[0].AuthoritySHA256, Finalization: GlossaryArchiveReference{finalPath, sum(finalBytes)}},
		{Package: added.ID, PackageAuthority: added.AuthoritySHA256, Finalization: GlossaryArchiveReference{"missing-independent-finalization.json", sum(nil)}},
	}
	if _, _, err := ComposeFinalizedTranslationPackage(root, current, "zh-CN", inputs); err == nil {
		t.Fatal("new Units inherited historical QC")
	}
	additionCatalog, err := HistoricalTranslationPackageCatalog(current, added.ID, added.AuthoritySHA256)
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, u := range added.Units {
		ids = append(ids, u.Unit.ID)
	}
	checkpointGenerate(t, root, additionCatalog, ids)
	addedSnapshot := futureSnapshot(t, root, additionCatalog, "independent-addition-finalized")
	for _, id := range ids {
		futureRecord(t, root, additionCatalog, addedSnapshot.SnapshotID, "", "A", "", []string{id})
	}
	_, path, err := FinalizeQualityCheck(root, additionCatalog, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: addedSnapshot.SnapshotID})
	if err != nil {
		t.Fatal(err)
	}
	b, err := readSnapshotReferencedFile(root, path)
	if err != nil {
		t.Fatal(err)
	}
	inputs[1].Finalization = GlossaryArchiveReference{path, sum(b)}
	proof, files, err := ComposeFinalizedTranslationPackage(root, current, "zh-CN", inputs)
	if err != nil || len(proof.Units) != 51 || len(files) != 51 || proof.Assessments[0].Compatible != 49 || proof.Assessments[1].Compatible != 2 {
		t.Fatal("independent 49 + 2 current authority", err)
	}
	artifacts := []SurfaceCandidateArtifact{}
	for _, a := range proof.Assessments {
		for _, u := range a.Units {
			artifacts = append(artifacts, SurfaceCandidateArtifact{SnapshotID: a.Historical.Finalization.SnapshotID, Unit: u.Snapshot})
		}
	}
	if err := ValidateSurfaceCandidatePackageArtifacts(root, current, "zh-CN", artifacts); err != nil {
		t.Fatal("current 51 with historical 49 provenance", err)
	}
	if err := ValidateSurfaceCandidatePackageArtifacts(root, current, "zh-CN", artifacts[:50]); err == nil {
		t.Fatal("historical 49 authorized a missing new Page")
	}
	// The source-provider projection is tested end-to-end in sitecontent. This
	// adapter fixture injects the same explicit descriptor metadata to exercise
	// the machine-backed QC consumer, including its original-only QC gate.
	membershipPath := "fixtures/machine-membership.json"
	declaration := TranslationPackageMembership{Predecessors: []TranslationPackagePredecessor{
		{ID: event.TranslationPackages[0].ID, AuthoritySHA256: event.TranslationPackages[0].AuthoritySHA256},
		{ID: added.ID, AuthoritySHA256: added.AuthoritySHA256},
	}}
	membershipBytes := mustJSON(declaration)
	writeSnapshotSource(t, root, membershipPath, membershipBytes)
	declaration.References = []GlossaryArchiveReference{{Path: membershipPath, SHA256: sum(membershipBytes)}}
	current.TranslationPackages[0].Membership = &declaration
	currentSnapshot, _, err := CreateQualityCheckCandidateSnapshot(root, current, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "machine-membership-successor", ContentPackage: current.TranslationPackages[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	scope, err := BuildQualityCheckScope(root, current, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: currentSnapshot.SnapshotID, PreviousSnapshotID: s.SnapshotID})
	if err != nil || scope.UnitCount != 51 || scope.CarryForwardCount != 51 || scope.PendingCount != 0 {
		t.Fatal("machine-backed membership QC baseline", scope, err)
	}
	if len(current.TranslationScope) != 51 || len(event.TranslationScope) != 49 {
		t.Fatal("membership QC changed current/historical scopes")
	}
	for _, u := range currentSnapshot.Units[:49] {
		if u.RepresentationReconciliation == nil {
			t.Fatal("machine-backed fixture lost historical lineage")
		}
	}
	t.Run("shared-preparation-final-guards", func(t *testing.T) {
		// A prepared world never replaces the consumer's fresh acceptance
		// boundary. Exercise that boundary after the entire graph is warm.
		evaluation := newQualityCheckEvaluation(root, "zh-CN", current)
		if _, err := buildQualityCheckScopeWithEvaluation(evaluation, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: currentSnapshot.SnapshotID, PreviousSnapshotID: s.SnapshotID}); err != nil {
			t.Fatal(err)
		}
		if evaluation.membership[currentSnapshot.SnapshotID] == nil {
			t.Fatal("missing complete membership QC graph")
		}
		paths := map[string]string{
			"primary-qc": "data/quality-check-snapshots/zh-CN/" + original.SnapshotID + "/quality-check-results.json",
			"membership": membershipPath,
			"receipt":    ref.Path,
			"source":     currentSnapshot.Units[0].SourcePath,
			"candidate":  plan.Receipt.Units[0].Candidate.Path,
			"validation": plan.Receipt.Units[0].Validation.Path,
			"raw":        plan.Receipt.Units[0].OriginalRaw.Path,
			"glossary":   "locales/zh-CN/glossary.yaml",
			"snapshot":   "data/quality-check-snapshots/zh-CN/" + currentSnapshot.SnapshotID + "/manifest.json",
		}
		for name, path := range paths {
			t.Run(name, func(t *testing.T) {
				before, err := readSnapshotReferencedFile(root, path)
				if err != nil {
					t.Fatal(err)
				}
				defer writeSnapshotSource(t, root, path, before)
				writeSnapshotSource(t, root, path, append(bytes.Clone(before), ' '))
				if err := evaluation.membershipChecks(); err == nil {
					t.Fatal("warm prepared graph accepted changed dependency", name)
				}
			})
		}
		if err := evaluation.membershipChecks(); err != nil {
			t.Fatal("failed guard was carried after restoring exact bytes", err)
		}
	})
	after, err := readSnapshotReferencedFile(root, ref.Path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("receipt changed", err)
	}
	t.Log("historical scope=49; current scope=51; independent current authority=49+2")
}

func TestHistoricalReconciliationScopeFailClosed(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, event, _, plan, _, _ := authorizationMachineFixtureSize(t, 49)
	current, _ := expandedMachinePackage(t, root, event, 2)
	for _, kind := range []string{"incomplete-qc", "snapshot-too-small", "snapshot-too-large", "missing-historical-authority", "wrong-historical-package", "historical-authority-tamper"} {
		t.Run(kind, func(t *testing.T) {
			r := cloneCompositionFixture(t, root)
			c := *current
			o := plan.Receipt.Options
			if strings.HasPrefix(kind, "snapshot-") || kind == "incomplete-qc" {
				manifest := qualityCheckSnapshotManifestPath(r, o.Locale, o.PreviousSnapshotID)
				results := qualityCheckResultsPath(r, o.Locale, o.PreviousSnapshotID)
				var s QualityCheckSnapshotManifest
				var q QualityCheckResults
				b, err := os.ReadFile(manifest)
				if err != nil || json.Unmarshal(b, &s) != nil {
					t.Fatal("fixture Snapshot", err)
				}
				b, err = os.ReadFile(results)
				if err != nil || json.Unmarshal(b, &q) != nil {
					t.Fatal("fixture QC", err)
				}
				if kind == "snapshot-too-small" {
					s.Units, s.UnitCount, s.PageCount = s.Units[:48], 48, 48
					q.Results = q.Results[:48]
				} else if kind == "snapshot-too-large" {
					u := s.Units[len(s.Units)-1]
					u.Index, u.UnitID = 50, "extra-snapshot-only-unit"
					s.Units, s.UnitCount, s.PageCount = append(s.Units, u), 50, 50
					q.Results = append(q.Results, QualityCheckResult{Index: u.Index, UnitID: u.UnitID, Rating: "A"})
				} else {
					q.Results = q.Results[:48]
				}
				b = mustJSON(s)
				writeSnapshotSource(t, r, filepath.ToSlash(strings.TrimPrefix(manifest, r+string(os.PathSeparator))), b)
				o.SnapshotSHA256, q.SnapshotManifestSHA256 = sum(b), sum(b)
				q.ResultCount = len(q.Results)
				writeSnapshotSource(t, r, filepath.ToSlash(strings.TrimPrefix(results, r+string(os.PathSeparator))), mustJSON(q))
			} else {
				resolver := c.ReplacementResolver
				c.ReplacementResolver = func(root string, catalog *Catalog, ref GlossaryArchiveReference) (*TranslationUnitReplacementResolution, error) {
					res, err := resolver(root, catalog, ref)
					if err != nil {
						return nil, err
					}
					h := *res.HistoricalCatalog
					h.TranslationPackages = append([]TranslationPackageDescriptor(nil), h.TranslationPackages...)
					switch kind {
					case "missing-historical-authority":
						h.TranslationPackages = nil
					case "wrong-historical-package":
						h.TranslationPackages[0].ID = "wrong-package"
					case "historical-authority-tamper":
						h.TranslationPackages[0].AuthoritySHA256 = sum([]byte("tamper"))
					}
					res.HistoricalCatalog = &h
					return res, nil
				}
			}
			_, err := prepareMachineReconciliationInWorld(r, &c, o, plan.Receipt.GlossarySHA256)
			if err == nil {
				t.Fatal("invalid historical proof accepted", kind)
			}
			if (kind == "incomplete-qc" || strings.HasPrefix(kind, "snapshot-")) && !strings.Contains(err.Error(), "complete initial independent QC required for machine reconciliation") {
				t.Fatal("lost incomplete QC gate", err)
			}
			t.Log(err)
		})
	}
}

func TestHistoricalReconciliationUnexpandedScope(t *testing.T) {
	root, c, _, plan, _, _ := authorizationMachineFixtureSize(t, 1)
	ctx, err := prepareMachineReconciliationInWorld(root, c, plan.Receipt.Options, plan.Receipt.GlossarySHA256)
	if err != nil || len(ctx.snapshot.Units) != 1 || len(ctx.effective) != 1 || len(c.TranslationScope) != 1 || len(ctx.resolution.HistoricalCatalog.TranslationScope) != 1 {
		t.Fatal("unchanged 1-to-1 control", err)
	}
}
