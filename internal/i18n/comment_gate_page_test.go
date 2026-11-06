package i18n_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func TestCommentGateFutureRegisteredPage(t *testing.T) {
	root := t.TempDir()
	for _, locale := range []string{"zh-CN", "en-GB"} {
		b, err := os.ReadFile("../../locales/" + locale + "/locale.json")
		if err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(root, "locales", locale)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "locale.json"), b, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "glossary.yaml"), []byte("mandatory:\n  Go: Go\nkeep:\n  - Go\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, fixture := range []struct {
		name, path, code string
		invalid          bool
	}{
		{"go", "future.md", "```go\npackage p\n// Return a greeting for name.\nvar name string\n```\n", false},
		{"schema", "future.md", "```json\n{\n // The vulnerability ID.\n \"id\": string\n}\n```\n", false},
		{"unclassified", "future.md", "```json\n{ //go:generate tool\n \"id\": string\n}\n```\n", true},
		{"unsupported", "future.md", "```mystery\nunknown ???\n```\n", true},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			raw := []byte("# Example\n\n" + fixture.code)
			d, err := sitecontent.ParseWorkflowDocument(sitecontent.Source{Path: fixture.path, Kind: "page", Route: "/future/example", SHA256: contentidentity.SHA256Hex(raw)}, raw, 1)
			if err != nil {
				t.Fatal(err)
			}
			adapter := sitecontent.PageAdapter{Document: *d, Source: raw}
			c := &i18n.Catalog{}
			id := d.Units[0].ID
			if err := c.RegisterTranslationPackage(i18n.TranslationPackageDescriptor{ID: "future-registered-pages", AuthoritySHA256: contentidentity.SHA256Hex([]byte("fixture authority")), Units: []i18n.AdaptedTranslationUnit{{Unit: i18n.TranslationUnit{ID: id, Kind: i18n.UnitKindPage, SourcePath: fixture.path, SourceSHA256: d.SourceSHA, Source: raw}, Adapter: adapter, StableIndex: 1, Route: d.Route}}}); err != nil {
				t.Fatal(err)
			}
			if err := c.UseTranslationPackage("future-registered-pages"); err != nil {
				t.Fatal(err)
			}
			for _, locale := range []string{"zh-CN", "en-GB"} {
				err := i18n.ValidateTranslationUnitCandidate(root, c, id, locale, raw)
				if fixture.invalid || locale == "zh-CN" {
					if err == nil {
						t.Fatal("gate bypassed", locale)
					}
				} else if err != nil {
					t.Fatal("English-base exact carry rejected", err)
				}
			}
			if !fixture.invalid {
				target := strings.Replace(string(raw), "Return a greeting for name.", "为 name 返回问候语。", 1)
				target = strings.Replace(target, "The vulnerability ID.", "漏洞 ID。", 1)
				if err := i18n.ValidateTranslationUnitCandidate(root, c, id, "zh-CN", []byte(target)); err != nil {
					t.Fatal("translated teaching rejected", err)
				}
			}
		})
	}
}

// Read-only activation proof. Reuse the checked predecessor witness, read real
// repository candidates, and inspect the existing latest selector. No inventory
// rebuild, authorization write, Snapshot or lifecycle command is performed.
func TestCommentGateActivationAcceptedTargets(t *testing.T) {
	output := os.Getenv("GO_TOUR_COMMENT_GATE_ACTIVATION_EVIDENCE")
	if output == "" {
		t.Skip("explicit read-only evidence output required")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	read := func(ref i18n.GlossaryArchiveReference) []byte {
		t.Helper()
		path := ref.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if ref.SHA256 != "" && contentidentity.SHA256Hex(b) != ref.SHA256 {
			t.Fatal("reference drift", ref.Path)
		}
		return b
	}
	type pageWitness struct {
		StableIndex int                           `json:"stable_index"`
		Route       string                        `json:"route"`
		UnitID      string                        `json:"unit_id"`
		Source      i18n.GlossaryArchiveReference `json:"source"`
		Candidate   i18n.GlossaryArchiveReference `json:"candidate"`
	}
	var prior struct {
		Result          string                               `json:"result"`
		Snapshot        i18n.GlossaryArchiveReference        `json:"snapshot"`
		Inventory       string                               `json:"source_inventory_identity_sha256"`
		Authorization   i18n.CurrentTranslationAuthorization `json:"current_authorization"`
		Pages           []pageWitness                        `json:"pages"`
		Affected        []string                             `json:"affected_unit_ids"`
		Carry           []string                             `json:"carry_unit_ids"`
		Slots           []string                             `json:"exact_carry_slot_ids"`
		EnglishMetadata []i18n.GlossaryArchiveReference      `json:"english_base_control_metadata"`
		Tour            struct {
			Authority []i18n.GlossaryArchiveReference `json:"current_authority"`
			Targets   []i18n.AcceptedCommentTarget    `json:"accepted_targets"`
		} `json:"tour_current_audit"`
	}
	priorBytes := read(i18n.GlossaryArchiveReference{Path: "/tmp/zh-CN-code-comment-validator-activation-preflight-20261020.json"})
	if err := json.Unmarshal(priorBytes, &prior); err != nil {
		t.Fatal(err)
	}
	if prior.Result != "PASS" || len(prior.Pages) != 49 || len(prior.Affected) != 9 || len(prior.Carry) != 40 || len(prior.Slots) != 41 {
		t.Fatal("complete accepted preflight witness required")
	}
	var manifest i18n.QualityCheckSnapshotManifest
	if err := json.Unmarshal(read(prior.Snapshot), &manifest); err != nil {
		t.Fatal(err)
	}
	c, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := sitecontent.AttachRegisteredTranslationPackages(root, c); err != nil {
		t.Fatal(err)
	}
	sitecontent.AttachIntegratedSurfaceAuthority(c)
	learn := *c
	if err := learn.UseTranslationPackage("learn-docs-v1"); err != nil {
		t.Fatal(err)
	}
	if err := i18n.CommentGateLatestSelectionForTest(root, &learn, "zh-CN", manifest.SnapshotID); err != nil {
		t.Fatal(err)
	}
	assessment, err := i18n.AssessCurrentTranslation(root, &learn, "zh-CN", manifest.SnapshotID)
	if err != nil {
		t.Fatal(err)
	}
	if assessment.Glossary != prior.Authorization.Glossary || assessment.UnifiedReview != prior.Authorization.UnifiedReview {
		t.Fatal("current glossary/review authority changed since preflight")
	}
	base, metadata, err := i18n.CommentLocaleBaseLanguage(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		pageWitness
		Validation string                                `json:"validation"`
		Reason     string                                `json:"validation_reason"`
		Slots      []string                              `json:"exact_carry_slot_ids"`
		Assessment i18n.CurrentTranslationAssessmentUnit `json:"current_reattestation"`
	}
	results := []result{}
	affected, carry, slots, routes := []string{}, []string{}, []string{}, []string{}
	unrelated := 0
	knownSample := false
	for i, p := range prior.Pages {
		u, err := learn.Unit(p.UnitID)
		if err != nil {
			t.Fatal(err)
		}
		old := manifest.Units[i]
		if u.ID != old.UnitID || u.SourcePath != p.Source.Path || u.SourceSHA256 != p.Source.SHA256 || old.CandidatePath != p.Candidate.Path || old.CandidateSHA256 != p.Candidate.SHA256 || assessment.Units[i].Snapshot.UnitID != u.ID {
			t.Fatal("formal representation/source/candidate mapping drift", p.Route)
		}
		read(p.Source)
		candidate := read(p.Candidate)
		err = i18n.ValidateTranslationUnitCandidate(root, &learn, u.ID, "zh-CN", candidate)
		r := result{pageWitness: p, Validation: "PASS", Reason: "none", Slots: []string{}, Assessment: assessment.Units[i]}
		if err != nil {
			r.Validation = "FAILED"
			r.Reason = err.Error()
			var gate *i18n.CommentGateError
			if !errors.As(err, &gate) || gate.Reason != "non-English teaching-comment exact source carry" {
				unrelated++
			} else {
				r.Slots = gate.SlotIDs
				affected = append(affected, u.ID)
				routes = append(routes, p.Route)
				slots = append(slots, gate.SlotIDs...)
				knownSample = knownSample || p.Route == "/doc/security/vuln/database"
			}
			// Re-attestation retains its compatible glossary diagnostics after
			// the new validation failure. Exactly the gate error may be added;
			// no source/glossary/representation failure may replace that proof.
			expectedReasons := append([]string{"current_validation_failed: " + err.Error()}, prior.Authorization.Units[i].Reasons...)
			if r.Assessment.Classification != "affected" || r.Assessment.Validation != "failed" || !reflect.DeepEqual(r.Assessment.Reasons, expectedReasons) {
				unrelated++
			}
		} else {
			carry = append(carry, u.ID)
			if r.Assessment.Classification != "compatible" || r.Assessment.Validation != "passed" {
				unrelated++
			}
		}
		results = append(results, r)
	}
	t.Logf("active current validation: affected=%d carry=%d slots=%d unrelated=%d", len(affected), len(carry), len(slots), unrelated)
	tourSnapshot := ""
	for _, ref := range prior.Tour.Authority {
		b := read(ref)
		if strings.HasSuffix(ref.Path, "/finalization.json") {
			var f i18n.QualityCheckFinalization
			if err := json.Unmarshal(b, &f); err != nil {
				t.Fatal(err)
			}
			tourSnapshot = f.SnapshotID
		}
	}
	if tourSnapshot == "" {
		t.Fatal("Tour accepted witness missing")
	}
	if err := i18n.CommentGateLatestSelectionForTest(root, c, "zh-CN", tourSnapshot); err != nil {
		t.Fatal(err)
	}
	tourPages, tourExamples, tourPageAffected, tourExampleAffected := 0, 0, 0, 0
	for _, target := range prior.Tour.Targets {
		u, err := c.Unit(target.UnitID)
		if err != nil || u.SourceSHA256 != target.SourceSHA256 {
			t.Fatal("Tour source identity drift", err)
		}
		failed := i18n.ValidateTranslationUnitCandidate(root, c, u.ID, "zh-CN", read(target.Candidate)) != nil
		if target.Kind == i18n.UnitKindPage {
			tourPages++
			if failed {
				tourPageAffected++
			}
		} else {
			tourExamples++
			if failed {
				tourExampleAffected++
			}
		}
	}
	english := []i18n.GlossaryArchiveReference{}
	for _, ref := range prior.EnglishMetadata {
		var meta i18n.Locale
		if err := json.Unmarshal(read(ref), &meta); err != nil {
			t.Fatal(err)
		}
		base, _, err := i18n.CommentLocaleBaseLanguage(root, meta.Locale)
		if err != nil || base != "en" {
			t.Fatal("English control authority", err)
		}
		// Same real candidate bytes: only locale metadata changes this rule.
		for _, p := range prior.Pages {
			if !containsGateID(prior.Affected, p.UnitID) {
				continue
			}
			u, _ := learn.Unit(p.UnitID)
			if _, err := i18n.ValidateTranslationUnitCommentCoverage(root, &learn, u, meta.Locale, read(p.Candidate)); err != nil {
				t.Fatal("English-base control failed", err)
			}
		}
		english = append(english, ref)
	}
	dependency, err := i18n.CommentGateDependencyIdentityForTest(root)
	if err != nil {
		t.Fatal(err)
	}
	implementation := i18n.GlossaryArchiveReference{Path: "internal/i18n/comment_gate.go"}
	implementation.SHA256 = contentidentity.SHA256Hex(read(implementation))
	pass := i18n.CommentGateActive && len(results) == 49 && len(affected) == 9 && len(carry) == 40 && len(slots) == 41 && reflect.DeepEqual(affected, prior.Affected) && reflect.DeepEqual(carry, prior.Carry) && reflect.DeepEqual(slots, prior.Slots) && assessment.Affected == 9 && assessment.Compatible == 40 && assessment.Stale == 0 && assessment.Unknown == 0 && unrelated == 0 && tourPages == 103 && tourExamples == 19 && tourPageAffected == 0 && tourExampleAffected == 0 && len(english) > 0 && knownSample
	status := "FAILED"
	if pass {
		status = "PASS"
	}
	evidence := map[string]any{
		"result": status, "stage": "code-comment-validator-gate-activation", "comment_gate_active": i18n.CommentGateActive, "source_comment_extraction_coverage": "COMPLETE",
		"gate_implementation": implementation, "integration_point": "internal/i18n/go_example_validate.go:validateTranslationUnitWithGlossary", "validator_dependency_identity_sha256": dependency,
		"source_inventory_identity_sha256": prior.Inventory, "trusted_preflight": i18n.GlossaryArchiveReference{Path: "/tmp/zh-CN-code-comment-validator-activation-preflight-20261020.json", SHA256: contentidentity.SHA256Hex(priorBytes)}, "source_inventory_rebuilt": false,
		"base_language": base, "base_language_authority": metadata, "snapshot": prior.Snapshot, "formal_representation_delta": 0, "current_latest_selection": "PASS",
		"total_page_count": len(results), "affected_page_count": len(affected), "carry_page_count": len(carry), "exact_carry_comment_slots": len(slots), "pages": results,
		"affected_unit_ids": affected, "carry_unit_ids": carry, "exact_carry_slot_ids": slots, "affected_routes": routes, "unrelated_validation_failures": unrelated, "current_reattestation": assessment,
		"tour_page_count": tourPages, "tour_example_count": tourExamples, "tour_page_affected": tourPageAffected, "tour_example_affected": tourExampleAffected, "tour_candidate_refs": prior.Tour.Targets,
		"english_base_exception": "PASS", "english_base_metadata_controls": english, "known_vuln_database_sample_detected": knownSample, "formal_language_artifact_mutation": "none",
		"targeted_test_summary": "canonical current validator and re-attestation exact 9/40/41; latest selection; Tour 103 Page/19 Example; English metadata controls; companion unit tests cover structural failures, future registered Page, protection and identity",
	}
	b, err := json.MarshalIndent(evidence, "", "  ")
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
		t.Fatal(fmt.Sprintf("activation proof failed: affected=%d carry=%d slots=%d unrelated=%d Tour=%d/%d", len(affected), len(carry), len(slots), unrelated, tourPageAffected, tourExampleAffected))
	}
}

func containsGateID(ids []string, id string) bool {
	for _, s := range ids {
		if s == id {
			return true
		}
	}
	return false
}
