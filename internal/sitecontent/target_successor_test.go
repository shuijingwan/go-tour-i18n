package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Real 122-TU and 49-Page finalized lineages share the current glossary.
// All mutations below are confined to the copied fixture, never formal state.
func TestCampaignPlanCrossPackageSuccessorSelection(t *testing.T) {
	root, cat := crossPackageReplacementFixture(t)
	f, _, err := i18n.RequireCurrentCanonicalQualityCheck(root, "zh-CN", cat)
	if err != nil || len(f.Units) != 122 {
		t.Fatalf("current Tour authorization: %v %v", f, err)
	}
	t.Log("historical and current foreign Snapshots excluded from Tour authorization")
	d, err := i18n.DiagnoseQualityCheckAuthorization(root, cat, "zh-CN", "qc-007")
	if err != nil || !d.CanonicalAuthorized || d.ReviewerACount != 122 {
		t.Fatalf("historical Tour authorization: %+v %v", d, err)
	}
	assertCrossPackageTourMutationBoundary(t, root, cat)
	// Verify the unchanged shared Learn/Docs authority with its own scope.
	packageCat := *cat
	if err := packageCat.UseTranslationPackage("learn-docs-v1"); err != nil {
		t.Fatal(err)
	}
	learn, err := i18n.VerifyQualityCheckFinalization(root, &packageCat, "zh-CN", "learn-docs-v1-qc-010")
	if err != nil || len(learn.Units) != 49 {
		t.Fatalf("shared Learn/Docs finalization: %v %v", learn, err)
	}
	if _, err := i18n.VerifyQualityCheckFinalization(root, &packageCat, "zh-CN", "learn-docs-v1-qc-009"); err == nil {
		t.Fatal("historical replacement predecessor became current")
	}
	for _, path := range []string{
		"data/site-workflow/zh-CN/learn-docs-v1/closure/8ca5ed968d28c26ac2c186514dff1cf2c50f4df29e7071c593a954390ea34ca7.json",
		"data/site-workflow/zh-CN/site-v2-shell/closure/5e34b571b665bf7f0819e98a5dd4a0b784bb4028a925f0a2b37b0d9f46eba8c7.json",
	} {
		ref, err := ReferenceFile(root, path)
		if err != nil {
			t.Fatal(err)
		}
		before, _ := readRegular(root, path)
		if _, _, err := CheckPackageClosure(root, ref); err != nil {
			t.Fatal(err)
		}
		after, _ := readRegular(root, path)
		if !bytes.Equal(before, after) {
			t.Fatal("closure verification mutated evidence")
		}
	}
}

func crossPackageReplacementFixture(t *testing.T) (string, *i18n.Catalog) {
	t.Helper()
	root, _, cat, _ := currentTourCheckpointFixture(t)
	for _, directory := range []string{"data/generation-archives/zh-CN", "data/site-workflow/zh-CN", "data/locale-language/zh-CN", "data/quality-check-reopens/zh-CN", "data/translation-unit-replacements/zh-CN"} {
		if err := filepath.WalkDir(filepath.Join(repositoryRoot(t), directory), func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			rel, err := filepath.Rel(repositoryRoot(t), path)
			if err != nil {
				return err
			}
			b, err := readRegular(repositoryRoot(t), filepath.ToSlash(rel))
			if err != nil {
				return err
			}
			writeFixture(t, root, filepath.ToSlash(rel), b)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := AttachRegisteredTranslationPackages(root, cat); err != nil {
		t.Fatal(err)
	}
	AttachIntegratedSurfaceAuthority(cat)
	return root, cat
}

func assertCrossPackageTourMutationBoundary(t *testing.T, root string, cat *i18n.Catalog) {
	t.Helper()
	path := "data/quality-check-snapshots/zh-CN/qc-008/manifest.json"
	original, err := readRegular(root, path)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []string{"membership", "identity", "foreign-id", "foreign-descriptor-disguise", "current-foreign-descriptor-disguise"} {
		t.Run(mutation, func(t *testing.T) {
			var s i18n.QualityCheckSnapshotManifest
			if err := json.Unmarshal(original, &s); err != nil {
				t.Fatal(err)
			}
			switch mutation {
			case "membership":
				s.Units = s.Units[1:]
				s.UnitCount--
				s.PageCount--
				for i := range s.Units {
					s.Units[i].Index = i + 1
				}
			case "identity":
				s.Units[0].CandidateSHA256 = digest([]byte("corrupted candidate identity"))
			case "foreign-id":
				s.Units[0].UnitID = "unknown-unit"
			case "foreign-descriptor-disguise", "current-foreign-descriptor-disguise":
				id := "learn-docs-v1-qc-009"
				if mutation == "current-foreign-descriptor-disguise" {
					id = "learn-docs-v1-qc-010"
				}
				foreign, err := readRegular(root, "data/quality-check-snapshots/zh-CN/"+id+"/manifest.json")
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(foreign, &s); err != nil {
					t.Fatal(err)
				}
				s.SnapshotID = "qc-008"
			}
			b, _ := json.Marshal(s)
			writeFixture(t, root, path, b)
			defer writeFixture(t, root, path, original)
			if _, _, err := i18n.RequireCurrentCanonicalQualityCheck(root, "zh-CN", cat); err == nil {
				t.Fatal("corrupt current Tour finalized set authorized")
			}
		})
	}
}

func TestCampaignPlanCrossPackageReplacementEvidenceBoundary(t *testing.T) {
	root, cat := crossPackageReplacementFixture(t)
	const directory = "data/quality-check-snapshots/zh-CN/"
	const prior = directory + "learn-docs-v1-qc-009/"
	const successor = directory + "learn-docs-v1-qc-010/"
	const reopen = "data/quality-check-reopens/zh-CN/20261011-zh-CN-integrated-surface-001-tu-reopen-002.json"
	var authorization i18n.QualityCheckSurfaceReopen
	b, err := readRegular(root, reopen)
	if err != nil || json.Unmarshal(b, &authorization) != nil || authorization.ReplacementContext == nil {
		t.Fatal("missing real replacement fixture")
	}
	for _, mutation := range []string{
		"missing-replacement-proof", "changed-replacement-proof", "missing-successor-reopen", "changed-predecessor-reopen",
		"historical-manifest", "historical-finalization-binding", "current-finalization", "no-formal-successor-lineage",
		"partial-foreign", "mixed-package", "unknown-full-inventory", "source-drift", "source-bytes-drift", "rebound-fake-old-id", "missing-surface-reader",
	} {
		t.Run(mutation, func(t *testing.T) {
			originals := map[string][]byte{}
			set := func(path string, bytes []byte) {
				if _, ok := originals[path]; !ok {
					b, err := readRegular(root, path)
					if err != nil {
						t.Fatal(err)
					}
					originals[path] = b
				}
				writeFixture(t, root, path, bytes)
			}
			defer func() {
				for path, b := range originals {
					writeFixture(t, root, path, b)
				}
			}()
			appendBytes := func(path string, missing bool) {
				b, err := readRegular(root, path)
				if err != nil {
					t.Fatal(err)
				}
				set(path, append(bytes.Clone(b), '\n'))
				if missing {
					if err := os.Remove(filepath.Join(root, path)); err != nil {
						t.Fatal(err)
					}
				}
			}
			copyCat := *cat
			switch mutation {
			case "missing-replacement-proof", "changed-replacement-proof":
				appendBytes(authorization.ReplacementContext.HistoricalDocuments.Path, mutation == "missing-replacement-proof")
			case "missing-successor-reopen":
				appendBytes(reopen, true)
			case "changed-predecessor-reopen":
				appendBytes(authorization.ReplacementContext.PredecessorReopen.Path, false)
			case "source-bytes-drift":
				// Page source bytes live in the frozen upstream archive, not as
				// physical _content/doc files. Model an unsanctioned registered
				// source update without granting a replacement authorization.
				copyCat.TranslationPackages = append([]i18n.TranslationPackageDescriptor(nil), cat.TranslationPackages...)
				for i, descriptor := range copyCat.TranslationPackages {
					if descriptor.ID != "learn-docs-v1" {
						continue
					}
					descriptor.Units = append([]i18n.AdaptedTranslationUnit(nil), descriptor.Units...)
					descriptor.Units[0].Unit.Source = append(bytes.Clone(descriptor.Units[0].Unit.Source), '\n')
					descriptor.Units[0].Unit.SourceSHA256 = digest(descriptor.Units[0].Unit.Source)
					copyCat.TranslationPackages[i] = descriptor
				}
			case "historical-finalization-binding", "current-finalization", "no-formal-successor-lineage":
				path := prior + "finalization.json"
				if mutation != "historical-finalization-binding" {
					path = successor + "finalization.json"
				}
				b, _ := readRegular(root, path)
				var f i18n.QualityCheckFinalization
				if err := json.Unmarshal(b, &f); err != nil {
					t.Fatal(err)
				}
				if mutation == "no-formal-successor-lineage" {
					f.PredecessorFinalization, f.PreviousSnapshotID = nil, ""
				} else {
					f.SnapshotManifestSHA256 = digest([]byte("rewritten finalization binding"))
				}
				b, _ = json.Marshal(f)
				set(path, b)
			case "missing-surface-reader":
				copyCat.SurfaceAuthorityReader = nil
			default:
				b, _ := readRegular(root, prior+"manifest.json")
				var s i18n.QualityCheckSnapshotManifest
				if err := json.Unmarshal(b, &s); err != nil {
					t.Fatal(err)
				}
				switch mutation {
				case "partial-foreign":
					s.Units = s.Units[:48]
					s.UnitCount, s.PageCount = 48, 48
				case "mixed-package":
					b, _ := readRegular(root, directory+"qc-008/manifest.json")
					var tour i18n.QualityCheckSnapshotManifest
					if err := json.Unmarshal(b, &tour); err != nil {
						t.Fatal(err)
					}
					s.Units[0] = tour.Units[0]
				case "unknown-full-inventory":
					for i := range s.Units {
						s.Units[i].UnitID = digest([]byte(s.Units[i].UnitID + "invented"))
						s.Units[i].SourcePath = "unknown/" + s.Units[i].SourcePath
					}
				case "source-drift":
					s.Units[0].SourceSHA256 = digest([]byte("unsanctioned source drift"))
				case "historical-manifest":
					s.Units[0].CandidateSHA256 = digest([]byte("rewritten historical candidate"))
				case "rebound-fake-old-id":
					s.Units[31].UnitID = digest([]byte("invented historical representation"))
				}
				b, _ = json.Marshal(s)
				set(prior+"manifest.json", b)
				if mutation == "rebound-fake-old-id" {
					// Rebind the two immediate hashes and membership as well. Only
					// formal replacement evidence can authorize the invented ID.
					manifestSHA := digest(b)
					b, _ := readRegular(root, prior+"finalization.json")
					var f i18n.QualityCheckFinalization
					if err := json.Unmarshal(b, &f); err != nil {
						t.Fatal(err)
					}
					f.SnapshotManifestSHA256 = manifestSHA
					f.Units[31].UnitID, f.Units[31].Snapshot = s.Units[31].UnitID, s.Units[31]
					b, _ = json.Marshal(f)
					set(prior+"finalization.json", b)
					priorSHA := digest(b)
					b, _ = readRegular(root, successor+"finalization.json")
					if err := json.Unmarshal(b, &f); err != nil {
						t.Fatal(err)
					}
					f.PredecessorFinalization.SHA256 = priorSHA
					b, _ = json.Marshal(f)
					set(successor+"finalization.json", b)
				}
			}
			if _, _, err := i18n.RequireCurrentCanonicalQualityCheck(root, "zh-CN", &copyCat); err == nil {
				t.Fatal("unproven foreign history bypassed strict finalized Snapshot selection")
			} else {
				t.Logf("fail closed: %v", err)
			}
		})
	}
}

// Real frozen evidence is copied read-only to an isolated tree. The only apply
// below is in t.TempDir; repository historical bytes are never changed.
func TestCampaignPlanHistoricalPromotionRecoveryBoundary(t *testing.T) {
	root := copiedHistoricalTourFixture(t)
	// After a real SEO refresh, this older TU-only regression must supply its
	// original metadata explicitly to recreate the pre-refresh fixture world.
	if source := os.Getenv("COURSE_HISTORY_RECOVERY_SOURCE"); source != "" {
		b, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, "locales/zh-CN/course-metadata.json", b)
	} else {
		b, err := readRegular(root, LocalePath("zh-CN"))
		if err != nil {
			t.Fatal(err)
		}
		var closure Locale
		if err := StrictJSON(b, &closure); err != nil {
			t.Fatal(err)
		}
		current, err := readRegular(root, "locales/zh-CN/course-metadata.json")
		if err != nil {
			t.Fatal(err)
		}
		for _, completion := range closure.Packages {
			for _, ref := range completion.Evidence {
				if ref.Path == "locales/zh-CN/course-metadata.json" && digest(current) != ref.SHA256 {
					t.Skip("pre-refresh regression requires explicit exact historical metadata source")
				}
			}
		}
	}
	cat, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	runHistoricalPromotionBoundary(t, root, cat)
}

func copiedHistoricalTourFixture(t *testing.T) string {
	t.Helper()
	root, repository := siteFixture(t, "data/terminology-discovery-sources.json", "data/terminology-discovery-sources.zip"), repositoryRoot(t)
	for _, path := range []string{"data/promotion-recoveries/zh-CN"} {
		if err := filepath.WalkDir(filepath.Join(repository, path), func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(repository, p)
			if err != nil {
				return err
			}
			b, err := readRegular(repository, filepath.ToSlash(rel))
			if err != nil {
				return err
			}
			writeFixture(t, root, filepath.ToSlash(rel), b)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{"locales/zh-CN", "data/retranslation-runs/zh-CN", "data/quality-check-snapshots/zh-CN", "data/glossary-history/zh-CN", "data/glossary-compatibility/zh-CN", "data/glossary-reviews/zh-CN", "data/unified-glossary-reviews/zh-CN", "data/unified-glossary-reviews/zh-CN-bundles", "data/locale-surface-reviews/zh-CN", "data/course-seo", "data/surface-config-history", "internal/tour/ui/zh-CN.json", "internal/tour/tour.go", "internal/tour/production.go", "internal/tour/seo.go", "_content/js/playground.js", "data/glossary-review-legacy-coverage.json"} {
		if _, err := os.Lstat(filepath.Join(repository, path)); os.IsNotExist(err) {
			continue
		}
		if err := filepath.WalkDir(filepath.Join(repository, path), func(p string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			rel, e := filepath.Rel(repository, p)
			if e != nil {
				return e
			}
			// Exercise the manifest-only boundary even after the maintainer
			// legally finalizes this Snapshot in the real working tree.
			if filepath.ToSlash(rel) == "data/quality-check-snapshots/zh-CN/qc-008/finalization.json" || filepath.ToSlash(rel) == "data/quality-check-snapshots/zh-CN/qc-008/quality-check-results.json" {
				return nil
			}
			b, e := readRegular(repository, filepath.ToSlash(rel))
			if e != nil {
				return e
			}
			writeFixture(t, root, filepath.ToSlash(rel), b)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func runHistoricalPromotionBoundary(t *testing.T, root string, cat *i18n.Catalog) {
	d, err := i18n.DiagnoseQualityCheckAuthorization(root, cat, "zh-CN", "qc-007")
	if err != nil || d.ReviewerACount != 122 || d.CanonicalAuthorized {
		t.Fatalf("before current finalization: %+v %v", d, err)
	}
	scope, err := i18n.BuildQualityCheckScope(root, cat, i18n.QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-008", PreviousSnapshotID: "qc-007"})
	if err != nil || scope.CarryForwardCount != 122 || scope.CurrentResultCount != 0 || !scope.ReadyForFinalization {
		t.Fatalf("real evidence scope: %+v %v", scope, err)
	}
	closure, _ := readRegular(root, LocalePath("zh-CN"))
	f, _, err := i18n.FinalizeQualityCheck(root, cat, i18n.QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "qc-008", PreviousSnapshotID: "qc-007"})
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range f.Units {
		if u.SourceSnapshotID == "qc-008" || len(u.CompatibilityChain) == 0 {
			t.Fatalf("false current Reviewer judgment: %+v", u)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "data/quality-check-snapshots/zh-CN/qc-008/quality-check-results.json")); !os.IsNotExist(err) {
		t.Fatal("fake current results")
	}
	after, _ := readRegular(root, LocalePath("zh-CN"))
	if string(closure) != string(after) {
		t.Fatal("recovery rewrote closure")
	}
	if _, err = i18n.VerifyQualityCheckFinalization(root, cat, "zh-CN", "qc-008"); err != nil {
		t.Fatal(err)
	}
	d, err = i18n.DiagnoseQualityCheckAuthorization(root, cat, "zh-CN", "qc-007")
	if err != nil || !d.CanonicalAuthorized || d.ReviewerACount != 122 {
		t.Fatalf("current authorization: %+v %v", d, err)
	}
	_, err = PlanLocaleLanguage(root, "zh-CN")
	if err == nil {
		t.Fatal("TU successor incorrectly carried derived SEO")
	}
	if strings.Contains(err.Error(), "historical closure evidence changed") {
		t.Fatalf("did not cross authorized Tour successor: %v", err)
	}
	t.Logf("next real consumer blocker: %v", err)
}

func TestCampaignPlanCourseSEOHistoryRecoveryBoundary(t *testing.T) {
	source := os.Getenv("COURSE_HISTORY_RECOVERY_SOURCE")
	if source == "" {
		t.Skip("explicit historical metadata source required for real-evidence integration fixture")
	}
	old, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	root := copiedHistoricalTourFixture(t)
	// The maintainer has finalized qc-008. Copy that exact evidence; do not
	// regenerate a Reviewer result or a finalization in this regression.
	b, err := readRegular(repositoryRoot(t), "data/quality-check-snapshots/zh-CN/qc-008/finalization.json")
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "data/quality-check-snapshots/zh-CN/qc-008/finalization.json", b)
	cat, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	closure, _ := readRegular(root, LocalePath("zh-CN"))
	metadata, _ := readRegular(root, "locales/zh-CN/course-metadata.json")
	_, beforeErr := i18n.AuthorizeHistoricalCourseMetadataReference(root, "zh-CN", digest(old), cat)
	if beforeErr == nil {
		t.Fatal("unrecorded transition accepted")
	}
	e, _, err := i18n.RecoverCourseMetadataHistory(root, cat, "zh-CN", old, true)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"flowcontrol/9", "concurrency/1", "concurrency/2", "concurrency/5", "concurrency/6", "concurrency/9"}
	if strings.Join(e.StalePages, ",") != strings.Join(want, ",") {
		t.Fatal(e.StalePages)
	}
	plan, err := PlanLocaleLanguage(root, "zh-CN")
	if err != nil {
		t.Fatalf("post-recovery next blocker: %v", err)
	}
	blocked := []string{}
	for _, work := range plan.Work {
		if work.Package == "tour-v1" && work.State != "carry" {
			blocked = append(blocked, work.ID+":"+work.State)
		}
	}
	if len(blocked) == 0 {
		t.Fatal("derived recovery incorrectly carried affected consumers")
	}
	t.Logf("next actual Tour work: %v", blocked)
	after, _ := readRegular(root, LocalePath("zh-CN"))
	if string(after) != string(closure) {
		t.Fatal("closure changed")
	}
	after, _ = readRegular(root, "locales/zh-CN/course-metadata.json")
	if string(after) != string(metadata) {
		t.Fatal("metadata changed")
	}
}
