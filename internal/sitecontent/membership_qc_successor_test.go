package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestMembershipQualityCheckSuccessor49Plus2(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	f, options, historical, batch := membershipRevisionSnapshotFixture(t, 0, 10, 29, 30, 48)
	expectedPending := []string{f.ids[0], f.ids[10], f.ids[29], f.ids[30], f.ids[48]}
	root, c := f.root, f.current
	s, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, options)
	if err != nil {
		t.Fatal(err)
	}
	scopeOptions := i18n.QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: s.SnapshotID, PreviousSnapshotID: f.finals[0].SnapshotID}
	scope, err := i18n.BuildQualityCheckScope(root, c, scopeOptions)
	if err != nil {
		t.Fatal("formal membership successor re-QC", err)
	}
	if scope.UnitCount != 51 || scope.CarryForwardCount != 46 || scope.PendingCount != 5 || scope.ACount != 46 || scope.ReadyForFinalization || scope.PreviousSnapshotID != f.finals[0].SnapshotID {
		t.Fatalf("scope: %+v", scope)
	}
	ids := []string{}
	for _, u := range scope.Pending {
		ids = append(ids, u.UnitID)
		if u.RequiredAction != i18n.QualityCheckActionRequired || u.Reason != i18n.QualityCheckScopeReasonIdentityChanged || u.BatchID != batch || u.Rating != "" {
			t.Fatal("revision inherited A", u)
		}
	}
	if !reflect.DeepEqual(ids, expectedPending) {
		t.Fatal("pending exact set", ids)
	}
	sibling := 0
	for _, u := range scope.CarryForward {
		if u.UnitID == f.ids[49] || u.UnitID == f.ids[50] {
			sibling++
			if u.Rating != "A" || u.FromSnapshotID != f.finals[1].SnapshotID {
				t.Fatal("sibling A did not come from independent QC", u)
			}
		}
	}
	if sibling != 2 {
		t.Fatal("missing sibling QC carry")
	}
	// Bundle current-check uses the same scope and independently reconstructs it;
	// carried Units are context only, and no initial partition is applied.
	for _, path := range []string{"AGENTS.md", "docs/SHARED_TRANSLATION_UNIT_LIFECYCLE.md", "docs/WORKFLOW_HANDOFF.md", "docs/CHATGPT_LANGUAGE_GENERATION.md", "docs/GLOSSARY_REVIEW.md", "docs/TRANSLATION_QUALITY_REVIEW.md", "docs/TRANSLATION_TASK_SPEC.md"} {
		b, err := os.ReadFile(filepath.Join("../..", path))
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, path, b)
	}
	bundle, manifest, err := i18n.ExportQualityCheckReviewerBundle(root, c, i18n.QualityCheckReviewerBundleOptions{Locale: "zh-CN", SnapshotID: s.SnapshotID, PreviousSnapshotID: scopeOptions.PreviousSnapshotID, Limit: 60})
	if err != nil || manifest.UnitCount != 5 {
		t.Fatal("pending-only Bundle", manifest, err)
	}
	files, err := i18n.ReadTransportBundle(bundle, 512, 128<<20)
	if err != nil {
		t.Fatal(err)
	}
	var context i18n.QualityCheckReviewerContext
	if err := json.Unmarshal(files["quality-check.json"], &context); err != nil {
		t.Fatal(err)
	}
	if context.Scope.CarryForwardCount != 46 || context.Scope.PendingCount != 5 || len(context.Units) != 5 {
		t.Fatal("fresh judgment working set", context)
	}
	for i, u := range context.Units {
		if u.UnitID != expectedPending[i] || u.UnitKind != i18n.UnitKindPage {
			t.Fatal("carried Unit entered judgment", u)
		}
	}
	if _, err := i18n.VerifyCurrentQualityCheckReviewerBundle(root, c, bundle); err != nil {
		t.Fatal("JIT current-check", err)
	}
	// Future ordinary same-package re-QC can name an unfinalized complete
	// predecessor. Membership must not impose a new finalized-only gate there.
	next := *s
	next.SnapshotID = "complete-unfinalized-successor"
	nextBytes, _ := json.Marshal(next)
	writeFixture(t, root, "data/quality-check-snapshots/zh-CN/"+next.SnapshotID+"/manifest.json", nextBytes)
	ordinary, err := i18n.BuildQualityCheckScope(root, c, i18n.QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: next.SnapshotID, PreviousSnapshotID: s.SnapshotID})
	if err != nil || ordinary.CarryForwardCount != 0 || ordinary.PendingCount != 51 {
		t.Fatal("complete unfinalized same-package behavior changed", ordinary, err)
	}
	t.Logf("current=%d historical=49+2 carry=%d pending=%d reviewer=%d", scope.UnitCount, scope.CarryForwardCount, scope.PendingCount, manifest.UnitCount)
	for _, name := range []string{"missing-membership", "tampered-membership", "wrong-predecessor-package", "wrong-ordered-membership", "duplicate-membership", "current-membership-drift", "missing-primary-finalization", "missing-secondary-finalization", "primary-incomplete-QC", "secondary-incomplete-QC", "tampered-primary-finalization", "secondary-non-A", "missing-secondary-unit", "source-drift", "candidate-drift", "validation-attempt", "raw-drift", "current-review-missing", "current-glossary-drift", "wrong-primary", "unauthorized-extra"} {
		t.Run(name, func(t *testing.T) {
			local := *c
			local.TranslationScope = append([]string(nil), c.TranslationScope...)
			local.HistoricalTranslationPackages = append([]i18n.TranslationPackageDescriptor(nil), c.HistoricalTranslationPackages...)
			for i := range local.HistoricalTranslationPackages {
				local.HistoricalTranslationPackages[i].Units = append([]i18n.AdaptedTranslationUnit(nil), local.HistoricalTranslationPackages[i].Units...)
			}
			o := scopeOptions
			path := ""
			switch name {
			case "missing-membership", "tampered-membership":
				path = f.membershipRef.Path
			case "wrong-predecessor-package":
				local.HistoricalTranslationPackages[1].ID = "wrong-package"
			case "wrong-ordered-membership":
				p := &local.HistoricalTranslationPackages[0]
				p.Units[0], p.Units[1] = p.Units[1], p.Units[0]
			case "duplicate-membership":
				local.HistoricalTranslationPackages[1].Units[1] = local.HistoricalTranslationPackages[0].Units[0]
			case "current-membership-drift":
				local.TranslationScope[0], local.TranslationScope[1] = local.TranslationScope[1], local.TranslationScope[0]
			case "missing-primary-finalization", "tampered-primary-finalization":
				path = "data/quality-check-snapshots/zh-CN/" + f.finals[0].SnapshotID + "/finalization.json"
			case "missing-secondary-finalization", "secondary-non-A":
				path = "data/quality-check-snapshots/zh-CN/" + f.finals[1].SnapshotID + "/finalization.json"
			case "missing-secondary-unit":
				local.HistoricalTranslationPackages[1].Units = local.HistoricalTranslationPackages[1].Units[:1]
			case "primary-incomplete-QC":
				path = "data/quality-check-snapshots/zh-CN/" + f.finals[0].SnapshotID + "/quality-check-results.json"
			case "secondary-incomplete-QC":
				path = "data/quality-check-snapshots/zh-CN/" + f.finals[1].SnapshotID + "/quality-check-results.json"
			case "source-drift":
				path = s.Units[9].SourcePath
			case "candidate-drift":
				path = s.Units[49].CandidatePath
			case "validation-attempt":
				path = s.Units[0].ValidationPath
			case "raw-drift":
				var v i18n.RetranslationValidation
				b, err := readRegular(root, s.Units[0].ValidationPath)
				if err != nil || json.Unmarshal(b, &v) != nil {
					t.Fatal(err)
				}
				path = "data/retranslation-runs/zh-CN/" + batch + "/" + v.RawResponsePath
			case "current-review-missing":
				a, err := i18n.RequireCurrentTranslationAuthorization(root, &local, "zh-CN", f.finals[0].SnapshotID)
				// The full current 51 Catalog must never accept the primary 49 alone.
				if err == nil && a.Ready() {
					t.Fatal("primary authorized current 51")
				}
				inventory := *c
				inventory.TranslationScope = nil
				h, _, err := i18n.SelectFinalizedTranslationPackage(root, &inventory, "zh-CN", f.finals[0].SnapshotID)
				if err != nil {
					t.Fatal(err)
				}
				proof, err := i18n.RequireCurrentTranslationAuthorization(root, h, "zh-CN", f.finals[0].SnapshotID)
				if err != nil {
					t.Fatal(err)
				}
				path = proof.UnifiedReview.Path
			case "current-glossary-drift":
				path = "locales/zh-CN/glossary.yaml"
			case "wrong-primary":
				o.PreviousSnapshotID = "missing-primary"
			case "unauthorized-extra":
				local.TranslationScope = append(local.TranslationScope, "unauthorized-extra")
			}
			if path != "" {
				before, err := readRegular(root, path)
				if err != nil {
					t.Fatal(err)
				}
				defer writeFixture(t, root, path, before)
				switch name {
				case "missing-membership", "missing-primary-finalization", "missing-secondary-finalization", "current-review-missing":
					if err := os.Remove(filepath.Join(root, path)); err != nil {
						t.Fatal(err)
					}
				case "primary-incomplete-QC", "secondary-incomplete-QC":
					var q i18n.QualityCheckResults
					if json.Unmarshal(before, &q) != nil {
						t.Fatal("QC fixture")
					}
					q.Results = q.Results[:len(q.Results)-1]
					q.ResultCount = len(q.Results)
					b, _ := json.Marshal(q)
					writeFixture(t, root, path, b)
				case "secondary-non-A", "tampered-primary-finalization":
					var f i18n.QualityCheckFinalization
					if json.Unmarshal(before, &f) != nil {
						t.Fatal("finalization fixture")
					}
					if name == "secondary-non-A" {
						f.Units[0].Rating = "B"
					} else {
						f.Units[0].Snapshot.CandidateSHA256 = digest([]byte("tamper"))
					}
					b, _ := json.Marshal(f)
					writeFixture(t, root, path, b)
				case "validation-attempt":
					var v i18n.RetranslationValidation
					if json.Unmarshal(before, &v) != nil {
						t.Fatal("validation fixture")
					}
					v.Attempt++
					b, _ := json.Marshal(v)
					writeFixture(t, root, path, b)
				default:
					writeFixture(t, root, path, append(bytes.Clone(before), ' '))
				}
			}
			if _, err := i18n.BuildQualityCheckScope(root, &local, o); err == nil {
				t.Fatal("invalid membership/QC authority accepted", name)
			} else {
				t.Log(err)
			}
		})
	}
	for path, before := range historical {
		after, err := readRegular(root, path)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("historical manifest modified", path, err)
		}
	}
}
