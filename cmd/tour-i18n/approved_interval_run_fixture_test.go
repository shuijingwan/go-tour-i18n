package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// Setup alone uses domain publishers, all in TempDir with synthetic language,
// QC and semantic decisions. The tested approved export/check are exclusively
// run(rawArgv); no Catalog or alternate parser is injected into that entry.
func approvedIntervalRunFixture(t *testing.T) nativeGenerationFixture {
	return approvedIntervalRunScopeFixture(t, 1)
}

func approvedIntervalRunScopeFixture(t *testing.T, scopeCount int) nativeGenerationFixture {
	t.Helper()
	root, current := generationRunSourceFixture(t)
	f := currentValidationExportFixture{root: root, locale: "en-GB", pkg: "learn-docs-v1", snapshot: "cli-approved-current", reopen: "cli-approved-reopen", catalog: current}
	metadata := installCurrentValidationExportAuthority(t, f)
	originalMetadata := courseContextRead(t, root, "locales/"+f.locale+"/locale.json")
	if scopeCount > 1 {
		probe := metadata
		policy := *metadata.LanguageAudit
		probe.LanguageAudit = &policy
		probe.LanguageAudit.Policy.ProseScripts = []string{"Latin", "Han"}
		probe.LanguageAudit.Policy.Spacing = []languageaudit.SpacingRule{{ProseScript: "Han", TechnicalScript: "Latin", Kinds: []languageaudit.Kind{languageaudit.Code}}}
		courseContextWrite(t, root, "locales/"+f.locale+"/locale.json", courseContextJSON(t, probe))
	}
	fixtureGlossary, err := i18n.LoadGlossary(root, f.locale)
	if err != nil {
		t.Fatal(err)
	}
	if err := current.UseTranslationPackage(f.pkg); err != nil {
		t.Fatal(err)
	}
	var descriptor i18n.TranslationPackageDescriptor
	for _, d := range current.TranslationPackages {
		if d.ID == f.pkg {
			descriptor = d
		}
	}
	oldDescriptor := descriptor
	oldDescriptor.Units = nil
	frozen := sitecontent.HistoricalPageRepresentations{Package: f.pkg, PackageSHA256: descriptor.AuthoritySHA256}
	oldTarget, currentTarget, targetSourceSHA := "", "", ""
	adjacentTokens := map[string]string{}
	var recoveryIDs []string
	for _, binding := range descriptor.Units {
		a := binding.Adapter.(sitecontent.PageAdapter)
		courseContextWrite(t, root, binding.Unit.SourcePath, a.Source)
		before := a.Document
		if scopeCount > 1 && binding.Unit.SourcePath != "_content/doc/tutorial/json.md" && len(adjacentTokens) < scopeCount-1 {
			protected, _, err := a.ProtectedInput(fixtureGlossary)
			if err != nil {
				t.Fatal(err)
			}
			currentUnit, err := current.Unit(binding.Unit.ID)
			if err != nil {
				t.Fatal(err)
			}
			for _, slot := range a.Document.Units {
				for _, p := range slot.Protected {
					if strings.HasPrefix(p.Raw, "`") && strings.HasSuffix(p.Raw, "`") && !strings.Contains(p.Raw, "\n") {
						candidate, err := a.Restore(strings.Replace(protected, p.Token, "说明"+p.Token, 1))
						if err != nil {
							continue
						}
						audit, err := i18n.TranslationUnitLanguageAudit(root, current, currentUnit, f.locale, candidate, fixtureGlossary)
						if err != nil {
							t.Fatal(err)
						}
						if languageaudit.Blocking(audit) == nil {
							continue
						}
						adjacentTokens[binding.Unit.ID] = p.Token
						break
					}
				}
				if adjacentTokens[binding.Unit.ID] != "" {
					break
				}
			}
		}
		if binding.Unit.SourcePath == "_content/doc/tutorial/json.md" {
			currentTarget = binding.Unit.ID
			targetSourceSHA = binding.Unit.SourceSHA256
			before.Slots = append([]sitecontent.Unit(nil), before.Slots...)
			for i, slot := range before.Slots {
				keep := []sitecontent.ProtectedToken{}
				for _, token := range slot.Protected {
					if token.Raw == "Marshal" && strings.Contains(slot.Source, "Custom ") {
						slot.Source = strings.ReplaceAll(slot.Source, token.Token, token.Raw)
					} else {
						keep = append(keep, token)
					}
				}
				slot.Protected = keep
				// Frozen slot identity is the source/protection object. Production
				// FormalPage constructs the actual historical Page codec and ID.
				identity, err := json.Marshal(struct {
					Text      string
					Protected []sitecontent.ProtectedToken
				}{slot.Source, keep})
				if err != nil {
					t.Fatal(err)
				}
				slot.SourceSHA = courseContextSHA(identity)
				before.Slots[i] = slot
			}
			before.Units = before.Slots
			var err error
			before, err = sitecontent.FormalPage(before, a.Source, binding.StableIndex)
			if err != nil {
				t.Fatal(err)
			}
			oldTarget = before.Units[0].ID
			if oldTarget == currentTarget {
				t.Fatal("historical representation delta missing")
			}
		}
		frozen.Documents = append(frozen.Documents, before)
		binding.Unit.ID = before.Units[0].ID
		binding.Adapter = sitecontent.PageAdapter{Document: before, Source: a.Source}
		oldDescriptor.Units = append(oldDescriptor.Units, binding)
	}
	courseContextWrite(t, root, "locales/"+f.locale+"/locale.json", originalMetadata)
	if scopeCount < 1 || len(adjacentTokens) != scopeCount-1 {
		t.Fatalf("synthetic complete scope requires %d source-owned code contexts, got %d", scopeCount-1, len(adjacentTokens))
	}
	for _, binding := range descriptor.Units {
		if binding.Unit.ID == currentTarget || adjacentTokens[binding.Unit.ID] != "" {
			recoveryIDs = append(recoveryIDs, binding.Unit.ID)
		}
	}
	if oldTarget == "" {
		t.Fatal("registered source fixture lacks json Page")
	}
	old := &i18n.Catalog{}
	if err := old.RegisterTranslationPackage(oldDescriptor); err != nil {
		t.Fatal(err)
	}
	if err := old.UseTranslationPackage(f.pkg); err != nil {
		t.Fatal(err)
	}
	sitecontent.AttachPageReplacementResolver(old)
	recordA := func(c *i18n.Catalog, snapshot, reviewer string) {
		for start := 0; start < len(c.TranslationScope); start += 30 {
			end := start + 30
			if end > len(c.TranslationScope) {
				end = len(c.TranslationScope)
			}
			if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: f.locale, SnapshotID: snapshot, UnitIDs: c.TranslationScope[start:end], Rating: "A", ReviewerSession: reviewer}); err != nil {
				t.Fatal("synthetic QC group", err)
			}
		}
	}
	// The canonical Learn/Docs initial scheduler requires separate <=30 groups.
	for start := 0; start < len(old.TranslationScope); start += 30 {
		end := start + 30
		if end > len(old.TranslationScope) {
			end = len(old.TranslationScope)
		}
		ids := old.TranslationScope[start:end]
		initial, err := i18n.ExportRetranslationBatch(root, old, i18n.RetranslationExportOptions{Locale: f.locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: ids})
		if err != nil {
			t.Fatal(err)
		}
		bundle, bm, err := i18n.ExportTranslationUnitGenerationBundle(root, old, i18n.GenerationBundleOptions{Locale: f.locale, BatchID: initial.BatchID})
		if err != nil {
			t.Fatal("initial fixture Bundle", err)
		}
		files, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
		if err != nil {
			t.Fatal(err)
		}
		outputs := t.TempDir()
		for _, out := range bm.ExpectedOutputs {
			raw := files["formal/inputs/"+filepath.Base(out.BundlePath)]
			if out.UnitID == oldTarget {
				raw = bytes.ReplaceAll(raw, []byte("Custom Marshal and Unmarshal"), []byte("自定义编组与解组"))
				raw = bytes.ReplaceAll(raw, []byte("The "), []byte("说明"))
			}
			if token := adjacentTokens[out.UnitID]; token != "" {
				if bytes.Count(raw, []byte(token)) != 1 {
					t.Fatal("source-owned token fixture must be unique", out.UnitID)
				}
				raw = bytes.Replace(raw, []byte(token), []byte("说明"+token), 1)
			}
			courseContextWrite(t, outputs, filepath.Base(out.BundlePath), raw)
		}
		if _, err = i18n.ImportGenerationOutputDirectory(root, old, bundle, "chatgpt", i18n.FormalChatGPTGenerationModel, outputs); err != nil {
			t.Fatal(err)
		}
		processed, err := i18n.ProcessRetranslationBatch(root, old, i18n.RetranslationProcessOptions{Locale: f.locale, BatchID: initial.BatchID})
		if err != nil || processed.ValidationPassed != len(ids) {
			t.Fatal("initial fixture validation", processed, err)
		}
	}
	const previous = "cli-approved-historical"
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, old, i18n.QualityCheckSnapshotOptions{Locale: f.locale, SnapshotID: previous}); err != nil {
		t.Fatal(err)
	}
	recordA(old, previous, "synthetic-cli-predecessor-reviewer")
	frozenBytes := courseContextJSON(t, frozen)
	frozenPath := "engineering/cli-historical-representations.json"
	courseContextWrite(t, root, frozenPath, frozenBytes)
	impactPath := "engineering/cli-impact.json"
	impact := courseContextJSON(t, map[string]any{"units": []map[string]any{{"current_unit_id": currentTarget, "snapshot_representation_unit_id": oldTarget, "source_path": "_content/doc/tutorial/json.md", "source_sha256": targetSourceSHA, "labels": map[string]bool{"representation_reconciliation_required": true, "true_language_defect": false, "unknown": true}}}})
	courseContextWrite(t, root, impactPath, impact)
	snapshotSHA := courseContextSHA(courseContextRead(t, root, "data/quality-check-snapshots/"+f.locale+"/"+previous+"/manifest.json"))
	reconcile := i18n.RepresentationReconciliationOptions{Locale: f.locale, Package: f.pkg, ReconciliationID: "cli-approved-reconciliation", PreviousSnapshotID: previous, SnapshotSHA256: snapshotSHA, Impact: i18n.GlossaryArchiveReference{Path: impactPath, SHA256: courseContextSHA(impact)}, HistoricalDocuments: i18n.GlossaryArchiveReference{Path: frozenPath, SHA256: courseContextSHA(frozenBytes)}}
	reviewOptions := i18n.RepresentationReviewOptions{UnitID: currentTarget, ReviewID: "cli-approved-semantic", Reconciliation: reconcile}
	reviewBundle, review, err := i18n.ExportRepresentationReviewBundle(root, current, reviewOptions)
	if err != nil {
		t.Fatal("synthetic semantic scope", err)
	}
	decision := i18n.RepresentationReviewDecision{ReviewID: reviewOptions.ReviewID, BundleIdentity: review.Identity, BundleSHA256: courseContextSHA(reviewBundle), Locale: f.locale, Package: f.pkg, OldUnitID: oldTarget, CurrentUnitID: currentTarget, PreviousSnapshotID: previous, FormalImpactSHA256: reconcile.Impact.SHA256, GenerationIndependent: true, ReviewerSession: "synthetic-cli-semantic-reviewer", Provider: "chatgpt", Model: i18n.FormalChatGPTGenerationModel, Decision: "APPROVE_RECONCILIATION", Finding: "Synthetic whole frozen interval approval only."}
	if _, _, err = i18n.RecordRepresentationReview(root, current, reviewBundle, courseContextJSON(t, decision)); err != nil {
		t.Fatal(err)
	}
	reconcile.Apply, reconcile.LanguageReviewIDs = true, []string{reviewOptions.ReviewID}
	if _, err = i18n.ReconcileTranslationUnitRepresentations(root, current, reconcile); err != nil {
		t.Fatal(err)
	}
	if _, _, err = i18n.CreateQualityCheckCandidateSnapshot(root, current, i18n.QualityCheckSnapshotOptions{Locale: f.locale, SnapshotID: f.snapshot}); err != nil {
		t.Fatal(err)
	}
	recordA(current, f.snapshot, "synthetic-cli-current-reviewer")
	if _, _, err = i18n.FinalizeQualityCheck(root, current, i18n.QualityCheckFinalizeOptions{Locale: f.locale, SnapshotID: f.snapshot}); err != nil {
		t.Fatal(err)
	}
	metadata.LanguageAudit.Version++
	metadata.LanguageAudit.Policy.ProseScripts = []string{"Latin", "Han"}
	metadata.LanguageAudit.Policy.Spacing = []languageaudit.SpacingRule{{ProseScript: "Han", TechnicalScript: "Latin", Kinds: []languageaudit.Kind{languageaudit.Code}}}
	courseContextWrite(t, root, "locales/"+f.locale+"/locale.json", courseContextJSON(t, metadata))
	assessment, err := i18n.AssessCurrentTranslation(root, current, f.locale, f.snapshot)
	if err != nil || assessment.Affected != scopeCount || assessment.Unknown != 0 || assessment.Stale != 0 {
		if assessment == nil {
			t.Fatal("synthetic finding scope", err)
		}
		t.Fatalf("synthetic finding scope affected=%d want=%d unknown=%d stale=%d error=%v", assessment.Affected, scopeCount, assessment.Unknown, assessment.Stale, err)
	}
	assessmentBytes := courseContextJSON(t, assessment)
	courseContextWrite(t, root, "engineering/cli-assessment.json", assessmentBytes)
	currentSHA := courseContextSHA(courseContextRead(t, root, "data/quality-check-snapshots/"+f.locale+"/"+f.snapshot+"/manifest.json"))
	o := i18n.CurrentValidationReopenOptions{Locale: f.locale, ContentPackage: f.pkg, ReopenID: f.reopen, PreviousSnapshotID: f.snapshot, SnapshotSHA256: currentSHA, Evidence: i18n.GlossaryArchiveReference{Path: "engineering/cli-assessment.json", SHA256: courseContextSHA(assessmentBytes)}, AssessmentIdentity: assessment.Identity}
	if _, _, err = i18n.RecordQualityCheckSurfaceReopen(root, current, i18n.QualityCheckSurfaceReopenOptions{AuthoritySource: i18n.CurrentValidationAuthority, Locale: f.locale, ReopenID: f.reopen, PreviousSnapshotID: f.snapshot, CurrentValidation: &o}); err != nil {
		t.Fatal(err)
	}
	batch, err := i18n.ExportRetranslationBatch(root, current, i18n.RetranslationExportOptions{Locale: f.locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: recoveryIDs, UnitKind: i18n.UnitKindPage, PreviousSnapshotID: f.snapshot, SurfaceReopenID: f.reopen, AllowReexport: true})
	if err != nil {
		t.Fatal(err)
	}
	selection := i18n.ApprovedIntervalSelection{Schema: i18n.ApprovedIntervalSelectionSchema, Locale: f.locale, BatchID: batch.BatchID, ContentPackage: f.pkg, Units: []i18n.ApprovedIntervalSelectionUnit{{CurrentUnitID: currentTarget, Mode: i18n.ApprovedIntervalMode}}}
	courseContextWrite(t, root, "engineering/cli-selection.json", courseContextJSON(t, selection))
	f.ids = recoveryIDs
	return nativeGenerationFixture{f: f, batch: *batch, output: filepath.Join(t.TempDir(), "approved-generation.zip")}
}
