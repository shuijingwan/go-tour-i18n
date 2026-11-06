package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/site"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
	"github.com/shuijingwan/go-tour-i18n/internal/tour"
)

// Explicit, read-only integration inspection. The gate is composed after the
// actual canonical validator in this test only; no production dispatch changes.
// Expected membership comes from the diagnostic inventory, never a route filter.
func TestCommentGateActivationPreflight(t *testing.T) {
	output := os.Getenv("GO_TOUR_COMMENT_GATE_PREFLIGHT")
	if output == "" {
		t.Skip("explicit diagnostic evidence output required")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	read := func(path string, value any) i18n.GlossaryArchiveReference {
		t.Helper()
		resolved := path
		if !filepath.IsAbs(path) {
			resolved = filepath.Join(root, path)
		}
		b, err := os.ReadFile(resolved)
		if err != nil {
			t.Fatal(err)
		}
		if value != nil {
			if err := json.Unmarshal(b, value); err != nil {
				t.Fatal(err)
			}
		}
		return i18n.GlossaryArchiveReference{Path: path, SHA256: contentidentity.SHA256Hex(b)}
	}
	const locale, pkg, snapshot = "zh-CN", "learn-docs-v1", "learn-docs-v1-qc-017"
	var manifest i18n.QualityCheckSnapshotManifest
	snapshotRef := read("data/quality-check-snapshots/"+locale+"/"+snapshot+"/manifest.json", &manifest)
	if snapshotRef.SHA256 != "bfc5c76cab7966bdc8bf7eeb99d8da2fd06e447fc39ac26c194de22422da8cd2" || manifest.SnapshotID != snapshot || manifest.Locale != locale || manifest.UnitCount != 49 {
		t.Fatal("predecessor identity/exact set changed")
	}
	c, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := sitecontent.AttachRegisteredTranslationPackages(root, c); err != nil {
		t.Fatal(err)
	}
	sitecontent.AttachIntegratedSurfaceAuthority(c)
	site.AttachPreviewFailureAuthority(c)
	source, err := commentSourceInventory(c)
	if err != nil {
		t.Fatal(err)
	}
	source, _, err = closeCommentInventory(c, source)
	if err != nil {
		t.Fatal(err)
	}
	if err := requireCompleteCommentInventory(source); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	inventorySHA := contentidentity.SHA256Hex(b)
	var proposal struct {
		Inventory string `json:"source_inventory_identity_sha256"`
		Locales   []struct {
			Locale   string `json:"locale"`
			Packages []struct {
				Package string `json:"package"`
				Units   []struct {
					ID    string            `json:"parent_unit_id"`
					Route string            `json:"route"`
					Slots []commentAuditRow `json:"slots"`
				} `json:"units"`
			} `json:"packages"`
		} `json:"locales"`
	}
	proposalRef := read("/tmp/code-comment-review-scope-20261020.json", &proposal)
	if inventorySHA != proposal.Inventory || inventorySHA != "c3dfa3656f096b29796a89d6b249fab1e40fe582cfd89bd9e2425137b13974e1" {
		t.Fatal("source inventory/proposal identity drift")
	}
	expectedIDs, expectedSlots := []string{}, []string{}
	for _, l := range proposal.Locales {
		for _, p := range l.Packages {
			if l.Locale == locale && p.Package == pkg {
				for _, u := range p.Units {
					expectedIDs = append(expectedIDs, u.ID)
					for _, s := range u.Slots {
						if s.SemanticKind == "teaching-comment" && s.ExactSourceCarry {
							expectedSlots = append(expectedSlots, s.SlotID)
						}
					}
				}
			}
		}
	}
	// Resolve base language through the current registry AND its locale metadata.
	baseOf := func(locale string) (string, i18n.GlossaryArchiveReference) {
		t.Helper()
		registered := false
		for _, l := range tour.LanguageRegistry() {
			registered = registered || (!l.Official && l.Locale == locale)
		}
		var metadata i18n.Locale
		ref := read("locales/"+locale+"/locale.json", &metadata)
		if !registered || metadata.Locale != locale || metadata.HTMLLang != locale {
			t.Fatal("locale metadata/registry mismatch", locale)
		}
		base, err := i18n.CommentBaseLanguage(metadata.HTMLLang)
		if err != nil {
			t.Fatal(err)
		}
		return base, ref
	}
	base, localeRef := baseOf(locale)
	englishControls := []i18n.GlossaryArchiveReference{}
	for _, l := range tour.LanguageRegistry() {
		if l.Official {
			continue
		}
		base, ref := baseOf(l.Locale)
		if base != "en" {
			continue
		}
		code := "package p\n// Return the vulnerability ID.\nvar id string\n//go:generate tool\n"
		block := i18n.AnalyzeCommentCode(code, "go", "example/1", true)
		rows, diagnostics := i18n.CompareCommentBlocks("control", base, []i18n.CommentCodeBlock{block}, []i18n.CommentCodeBlock{block})
		if len(rows) != 2 || !rows[0].ExactSourceCarry || i18n.ValidateCommentCoverage(rows, diagnostics, true) != nil {
			t.Fatal("English-base exact carry rejected", l.Locale)
		}
		englishControls = append(englishControls, ref)
	}
	if len(englishControls) == 0 {
		t.Fatal("English-base control authority missing")
	}
	learn := *c
	if err := learn.UseTranslationPackage(pkg); err != nil {
		t.Fatal(err)
	}
	// Both calls are existing read-only currentness boundaries. The accepted
	// target reader also rejects newer/stale candidate selection, independently
	// of the specified predecessor's historical A and re-attestation below.
	targets, targetAuthority, err := i18n.CurrentAcceptedCommentTargets(root, &learn, locale)
	if err != nil {
		t.Fatal(err)
	}
	authorization, err := i18n.RequireCurrentTranslationAuthorization(root, &learn, locale, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 49 || len(authorization.Units) != 49 || authorization.Historical.Finalization.SnapshotID != snapshot || !authorization.Ready() {
		t.Fatal("complete current accepted authority required")
	}
	byID := map[string]i18n.AcceptedCommentTarget{}
	for _, target := range targets {
		if _, duplicate := byID[target.UnitID]; duplicate {
			t.Fatal("duplicate accepted target")
		}
		byID[target.UnitID] = target
	}
	type pageResult struct {
		StableIndex int                                   `json:"stable_index"`
		Route       string                                `json:"route"`
		UnitID      string                                `json:"unit_id"`
		Source      i18n.GlossaryArchiveReference         `json:"source"`
		Candidate   i18n.GlossaryArchiveReference         `json:"candidate"`
		Current     i18n.CurrentTranslationAssessmentUnit `json:"current_reattestation"`
		Baseline    string                                `json:"canonical_validation"`
		Gate        string                                `json:"simulated_validation"`
		Reason      string                                `json:"validation_reason"`
		Error       string                                `json:"gate_error,omitempty"`
		Exact       int                                   `json:"exact_carry_count"`
		Slots       []string                              `json:"exact_carry_slot_ids"`
	}
	pages := []pageResult{}
	affected, carry, affectedRoutes, exactSlots := []string{}, []string{}, []string{}, []string{}
	unrelated, teaching, delta := 0, 0, 0
	knownSample := false
	for _, u := range source {
		if u.Package != pkg {
			continue
		}
		i := len(pages)
		target, ok := byID[u.ID]
		if !ok || i >= len(manifest.Units) || u.ID != manifest.Units[i].UnitID || u.Path != manifest.Units[i].SourcePath || u.SourceSHA != target.SourceSHA256 || target.Candidate.Path != manifest.Units[i].CandidatePath || target.Candidate.SHA256 != manifest.Units[i].CandidateSHA256 || authorization.Units[i].Snapshot.UnitID != u.ID {
			t.Fatal("source/current candidate/predecessor exact binding mismatch", u.Route)
		}
		current, err := learn.Unit(u.ID)
		if err != nil {
			t.Fatal(err)
		}
		if current.ID != manifest.Units[i].UnitID {
			delta++
		}
		r := pageResult{StableIndex: u.StableIndex, Route: u.Route, UnitID: u.ID, Source: i18n.GlossaryArchiveReference{Path: u.Path, SHA256: u.SourceSHA}, Candidate: target.Candidate, Current: authorization.Units[i], Baseline: "PASS", Gate: "PASS", Reason: "none", Slots: []string{}}
		if err := i18n.ValidateTranslationUnitCandidate(root, &learn, u.ID, locale, target.Bytes); err != nil {
			r.Baseline, r.Error = "FAILED", err.Error()
			unrelated++
		}
		blocks, err := commentBlocks(u, target.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		blocks = i18n.SourceSelectedTargetCommentBlocks(u.Blocks, blocks)
		prefix := pkg + "/" + u.Path + "/" + string(u.Kind) + "/" + u.ID
		rows, diagnostics := i18n.CompareCommentBlocks(prefix, base, u.Blocks, blocks)
		if err := i18n.ValidateCommentCoverage(rows, diagnostics, false); err != nil {
			r.Error = err.Error()
			unrelated++
		}
		for _, row := range rows {
			if row.SemanticKind == "teaching-comment" {
				teaching++
				if row.ExactSourceCarry {
					r.Exact++
					r.Slots = append(r.Slots, row.SlotID)
					if u.Route == "/doc/security/vuln/database" {
						knownSample = true // fixture assertion only, never classification
					}
				}
			}
		}
		if err := i18n.ValidateCommentCoverage(rows, diagnostics, true); err != nil {
			r.Gate, r.Error = "FAILED", err.Error()
			if r.Exact == 0 || base == "en" {
				unrelated++
			}
			r.Reason = "non-English teaching-comment exact source carry"
			affected = append(affected, u.ID)
			affectedRoutes = append(affectedRoutes, u.Route)
			exactSlots = append(exactSlots, r.Slots...)
		} else {
			carry = append(carry, u.ID)
		}
		pages = append(pages, r)
	}
	t.Logf("current Page validation: pages=%d affected=%d carry=%d exact=%d unrelated=%d", len(pages), len(affected), len(carry), len(exactSlots), unrelated)
	// Read only zh-CN Tour accepted targets; do not scan any other locale.
	tourAudit := auditCommentPackage(root, c, locale, base, "tour-v1", source)
	if tourAudit.State != "CURRENT" {
		t.Fatal("Tour currentness unavailable", tourAudit.Error)
	}
	tourAffected := []string{}
	for _, u := range source {
		if u.Package != "tour-v1" {
			continue
		}
		rows := []i18n.CommentSlotComparison{}
		for _, row := range tourAudit.Rows {
			if row.ID == u.ID {
				rows = append(rows, row.CommentSlotComparison)
			}
		}
		if i18n.ValidateCommentCoverage(rows, nil, true) != nil {
			tourAffected = append(tourAffected, u.ID)
		}
	}
	unrelated += len(tourAudit.Diagnostics)
	// Existing cross-locale output is a diagnostic impact summary only. Its
	// targets are not read/revalidated or used as current authority this run.
	var cross struct {
		Schema    string               `json:"schema"`
		Inventory string               `json:"source_inventory_identity_sha256"`
		Locales   []commentLocaleAudit `json:"locales"`
	}
	crossRef := read("/tmp/code-comment-current-locale-audit-20261020.json", &cross)
	// audit hashes the complete source envelope; closure/review-scope hash
	// only its ordered Units. Reconstruct the existing audit encoding rather
	// than comparing hashes of different projections or weakening identity.
	languages, unsupported := map[string]int{}, map[string]int{}
	for _, u := range source {
		for _, block := range u.Blocks {
			languages[block.Language]++
			if block.Unsupported {
				unsupported[block.Language]++
			}
		}
	}
	envelope := struct {
		Schema      string                   `json:"schema"`
		Counts      map[string]commentCounts `json:"counts"`
		Languages   map[string]int           `json:"languages"`
		Unsupported map[string]int           `json:"unsupported_languages"`
		Units       []commentInventoryUnit   `json:"units"`
	}{i18n.CommentInventorySchema, commentInventoryCounts(source), languages, unsupported, source}
	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if cross.Schema != i18n.CommentInventorySchema || cross.Inventory != contentidentity.SHA256Hex(envelopeBytes) {
		t.Fatal("cross-locale diagnostic inventory drift")
	}
	knownLocales := []string{}
	for _, l := range cross.Locales {
		hasCarry := false
		for _, p := range l.Packages {
			for _, row := range p.Rows {
				hasCarry = hasCarry || (p.State == "CURRENT" && l.Base != "en" && row.SemanticKind == "teaching-comment" && row.ExactSourceCarry)
			}
		}
		if hasCarry {
			knownLocales = append(knownLocales, l.Locale)
		}
	}
	pass := len(pages) == 49 && len(affected) == 9 && len(carry) == 40 && len(exactSlots) == 41 && teaching == 182 && delta == 0 && unrelated == 0 && len(tourAffected) == 0 && tourAudit.Page.Exact == 0 && tourAudit.Example.Exact == 0 && knownSample && len(knownLocales) == 6 && reflect.DeepEqual(affected, expectedIDs) && reflect.DeepEqual(exactSlots, expectedSlots)
	result := "FAILED"
	if pass {
		result = "PASS"
	}
	evidence := map[string]any{
		"result": result, "stage": "code-comment-validator-activation-preflight", "inspection_only": true, "comment_gate_active": false, "activation_executed": false,
		"snapshot": snapshotRef, "locale": locale, "base_language": base, "locale_metadata": localeRef, "content_package": pkg,
		"source_inventory_identity_sha256": inventorySHA, "source_comment_extraction_coverage": "COMPLETE", "machine_scope_proposal": proposalRef, "proposal_is_lifecycle_authority": false,
		"current_authorization": authorization, "current_target_authority": targetAuthority, "formal_representation_delta": delta,
		"canonical_validator_source": read("internal/i18n/go_example_validate.go", nil), "comment_policy_source": read("internal/i18n/comment_inventory.go", nil), "current_reattestation_source": read("internal/i18n/translation_authorization.go", nil),
		"total_page_count": len(pages), "affected_page_count": len(affected), "carry_page_count": len(carry), "teaching_comment_count": teaching, "exact_carry_comment_slots": len(exactSlots),
		"pages": pages, "affected_unit_ids": affected, "affected_routes": affectedRoutes, "carry_unit_ids": carry, "exact_carry_slot_ids": exactSlots, "machine_scope_exact_set_match": reflect.DeepEqual(affected, expectedIDs) && reflect.DeepEqual(exactSlots, expectedSlots),
		"unrelated_validation_failures": unrelated, "english_base_exception": "PASS", "english_base_control_metadata": englishControls, "language_registry_source": read("internal/tour/languages.go", nil),
		"tour_current_audit": tourAudit, "zh_CN_tour_affected_by_comment_gate": len(tourAffected), "known_vuln_database_sample_detected": knownSample,
		"reused_cross_locale_diagnostic": crossRef, "cross_locale_audit_rerun": false, "known_affected_locales": knownLocales, "known_affected_locale_count": len(knownLocales),
		"cross_locale_source_envelope_identity_sha256": cross.Inventory, "cross_locale_source_envelope_matches_current_inventory": true,
		"proposed_gate_integration_point": "internal/i18n/go_example_validate.go:validateTranslationUnitWithGlossary",
		"integration_design":              "After existing representation validation, use source-owned adapter/comment extraction plus current locale metadata base language to call ValidateCommentCoverage. Generation ValidateTranslationUnitCandidate and AssessCurrentTranslation both dispatch here. Bind shared policy/extractor/locale authority into current validation authority; preserve historical readers and affected-only lifecycle. No production wiring in this preflight.",
	}
	b, err = json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := f.Write(append(b, '\n'))
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatal(writeErr, closeErr)
	}
	if !pass {
		t.Fatal(fmt.Sprintf("activation preflight failed: pages=%d affected=%d carry=%d exact=%d unrelated=%d tour=%d; evidence=%s", len(pages), len(affected), len(carry), len(exactSlots), unrelated, len(tourAffected), output))
	}
}
