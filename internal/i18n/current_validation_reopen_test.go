package i18n

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Synthetic historical adapter epoch: identical physical Page/input/source,
// with a later validator recognizing an explicitly declared XML code field.
// No real repository language evidence is created by this fixture.
type validationReopenAdapter struct {
	futureCardAdapter
	code, enabled bool
}

func (a validationReopenAdapter) LanguageAuditDocuments(candidate []byte) (languageaudit.Document, languageaudit.Document, error) {
	return testCardLanguageAuditDocuments(a.source, candidate, a.code)
}

func (a validationReopenAdapter) ReattestationAuthority(string) (string, error) {
	return sum([]byte(fmt.Sprintf("synthetic-validator-epoch/%t", a.enabled))), nil
}
func (a validationReopenAdapter) CommentSourceBlocks() ([]CommentCodeBlock, string, error) {
	if !a.code || !a.enabled {
		return nil, "", nil
	}
	return []CommentCodeBlock{AnalyzeCommentCode(a.source.Message, "go", "xml-code/1", false)}, "", nil
}
func (a validationReopenAdapter) CommentTargetBlocks(b []byte) ([]CommentCodeBlock, error) {
	if !a.code || !a.enabled {
		return nil, nil
	}
	var card futureCard
	if err := xml.Unmarshal(b, &card); err != nil {
		return nil, err
	}
	return []CommentCodeBlock{AnalyzeCommentCode(card.Message, "go", "xml-code/1", false)}, nil
}

type validationFixtureBody string

func (b validationFixtureBody) ProtectedInput(*Glossary) (string, []string, error) {
	return string(b), nil, nil
}
func (b validationFixtureBody) Restore(s string) ([]byte, error) {
	if strings.TrimSpace(s) == "" || strings.ContainsAny(s, "\n<>&⟪⟫") || strings.Contains(s, "//") {
		return nil, fmt.Errorf("invalid fixture comment")
	}
	return []byte(s), nil
}
func (b validationFixtureBody) Validate([]byte, *Glossary) error { return nil }
func (a validationReopenAdapter) CommentRevisionBodies(candidate []byte) ([]CommentRevisionBody, error) {
	blocks, err := a.CommentTargetBlocks(candidate)
	if err != nil {
		return nil, err
	}
	out := []CommentRevisionBody{}
	pos := 0
	for _, block := range blocks {
		for _, c := range block.Comments {
			at := strings.Index(string(candidate[pos:]), c.Text)
			if at < 0 {
				return nil, fmt.Errorf("fixture raw comment missing")
			}
			at += pos
			left, right := CommentPayloadBounds(c.Text)
			out = append(out, CommentRevisionBody{Coordinate: block.Coordinate, Position: c.Position, Start: at + left, End: at + right, Adapter: validationFixtureBody(c.Text[left:right])})
			pos = at + len(c.Text)
		}
	}
	return out, nil
}

func currentValidationFixture(t *testing.T) (string, *Catalog, CurrentValidationReopenOptions, []string) {
	t.Helper()
	root, _, _ := futureContentFixture(t)
	copyBundleAuthority(t, root, currentValidationReopenDependencies)
	c := &Catalog{}
	d := TranslationPackageDescriptor{ID: "test-current-validation", AuthoritySHA256: sum([]byte("frozen source package"))}
	counts := []int{3, 3, 1, 11, 1, 2, 1, 17, 2}
	ids := []string{}
	for i := 0; i < 49; i++ {
		message := "Read this ordinary prose message."
		if i < len(counts) {
			var code strings.Builder
			code.WriteString("package p\n")
			for j := 0; j < counts[i]; j++ {
				fmt.Fprintf(&code, "// Return the teaching value number %d.\n", j+1)
			}
			code.WriteString("var value string\n")
			message = code.String()
		}
		card := futureCard{ID: fmt.Sprintf("page-%d", i+1), Href: "https://example.invalid/source", Heading: "A fixture Page", Message: message}
		b, err := xml.Marshal(card)
		if err != nil {
			t.Fatal(err)
		}
		b = append(b, '\n')
		id, path := fmt.Sprintf("current/page-%d", i+1), fmt.Sprintf("fixtures/current/page-%d.xml", i+1)
		writeSnapshotSource(t, root, path, b)
		d.Units = append(d.Units, AdaptedTranslationUnit{Unit: TranslationUnit{ID: id, Kind: UnitKindPage, SourcePath: path, Source: b, SourceSHA256: sum(b)}, Adapter: validationReopenAdapter{futureCardAdapter{card}, i < len(counts), false}, StableIndex: i + 1, Route: fmt.Sprintf("/fixture/%d", i+1)})
		ids = append(ids, id)
	}
	if err := c.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	e, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitIDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	b, m, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	_, files, err := readGenerationBundle(b)
	if err != nil {
		t.Fatal(err)
	}
	outputs := map[string][]byte{}
	for _, u := range m.ExpectedOutputs {
		outputs[u.UnitID] = files["formal/inputs/"+filepath.Base(u.BundlePath)]
	}
	futureImport(t, root, c, b, outputs)
	if result, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID}); err != nil || result.ValidationPassed != 49 {
		t.Fatal(result, err)
	}
	const snapshot = "current-validation-predecessor"
	futureSnapshot(t, root, c, snapshot)
	futureRecord(t, root, c, snapshot, "", "A", "", ids)
	if _, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: snapshot}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot); err != nil {
		t.Fatal(err)
	}
	for i, binding := range c.Adapted {
		a := binding.Adapter.(validationReopenAdapter)
		a.enabled = true
		binding.Adapter = a
		c.Adapted[i] = binding
		c.TranslationPackages[0].Units[i] = binding
	}
	a, err := AssessCurrentTranslation(root, c, "zh-CN", snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if a.Compatible != 40 || a.Affected != 9 || a.Stale != 0 || a.Unknown != 0 {
		t.Fatal("fixture impact", a.Compatible, a.Affected, a.Stale, a.Unknown)
	}
	raw, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	ref := GlossaryArchiveReference{Path: "engineering/current-assessment.json", SHA256: sum(raw)}
	writeSnapshotSource(t, root, ref.Path, raw)
	snapshotRef, err := successorReference(root, "data/quality-check-snapshots/zh-CN/"+snapshot+"/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	o := CurrentValidationReopenOptions{ContentPackage: d.ID, Locale: "zh-CN", ReopenID: "current-validation-comments", PreviousSnapshotID: snapshot, SnapshotSHA256: snapshotRef.SHA256, Evidence: ref, AssessmentIdentity: a.Identity}
	return root, c, o, ids
}

func TestCurrentValidationReopenAuthority(t *testing.T) {
	root, c, o, ids := currentValidationFixture(t)
	qcPath := qualityCheckResultsPath(root, o.Locale, o.PreviousSnapshotID)
	finalPath := qualityCheckFinalizationPath(root, o.Locale, o.PreviousSnapshotID)
	beforeQC, _ := os.ReadFile(qcPath)
	beforeFinal, _ := os.ReadFile(finalPath)
	for _, name := range []string{"absent-package", "wrong-package", "wrong-predecessor", "wrong-sha", "wrong-identity", "wrong-evidence-sha", "wrong-unit", "subset", "superset", "stale-policy", "stale-candidate", "stale-evidence", "wrong-evidence-package"} {
		t.Run(name, func(t *testing.T) {
			bad := o
			switch name {
			case "absent-package":
				bad.ContentPackage = ""
			case "wrong-package":
				bad.ContentPackage = "unregistered-package"
			case "wrong-predecessor":
				bad.PreviousSnapshotID = "another-snapshot"
			case "wrong-sha":
				bad.SnapshotSHA256 = sum([]byte("wrong"))
			case "wrong-identity":
				bad.AssessmentIdentity = sum([]byte("wrong"))
			case "wrong-evidence-sha":
				bad.Evidence.SHA256 = sum([]byte("wrong"))
			case "wrong-unit":
				bad.UnitIDs = []string{"not-a-current-unit"}
			case "subset":
				bad.UnitIDs = ids[:8]
			case "superset":
				bad.UnitIDs = ids[:10]
			case "stale-policy":
				binding := c.Adapted[0]
				a := binding.Adapter.(validationReopenAdapter)
				a.enabled = false
				changed := binding
				changed.Adapter = a
				c.Adapted[0] = changed
				c.TranslationPackages[0].Units[0] = changed
				defer func() { c.Adapted[0] = binding; c.TranslationPackages[0].Units[0] = binding }()
			case "stale-candidate":
				var a CurrentTranslationAuthorization
				data, _ := os.ReadFile(filepath.Join(root, o.Evidence.Path))
				_ = json.Unmarshal(data, &a)
				path := filepath.Join(root, a.Units[0].Snapshot.CandidatePath)
				old, _ := os.ReadFile(path)
				_ = os.WriteFile(path, []byte("stale"), 0600)
				defer os.WriteFile(path, old, 0600)
			case "stale-evidence":
				path := filepath.Join(root, o.Evidence.Path)
				old, _ := os.ReadFile(path)
				_ = os.WriteFile(path, append(old, ' '), 0600)
				defer os.WriteFile(path, old, 0600)
			case "wrong-evidence-package":
				data, err := os.ReadFile(filepath.Join(root, o.Evidence.Path))
				if err != nil {
					t.Fatal(err)
				}
				var a CurrentTranslationAuthorization
				if err := json.Unmarshal(data, &a); err != nil {
					t.Fatal(err)
				}
				a.Package = "another-registered-package"
				a.Identity = authorizationIdentity(a)
				data = mustJSON(a)
				bad.Evidence = GlossaryArchiveReference{Path: "engineering/wrong-package.json", SHA256: sum(data)}
				bad.AssessmentIdentity = a.Identity
				writeSnapshotSource(t, root, bad.Evidence.Path, data)
			}
			if _, _, err := InspectCurrentValidationReopen(root, c, bad); err == nil {
				t.Fatal("invalid authority accepted")
			}
		})
	}
	t.Run("complete-package-against-global-inventory", func(t *testing.T) {
		global := *c
		global.TranslationScope = nil
		global.Pages = retranslationTestCatalog(50).Pages
		other := c.TranslationPackages[0]
		other.ID = "other-registered-package"
		other.Units = append([]AdaptedTranslationUnit(nil), other.Units[:1]...)
		other.Units[0].Unit.ID = "other/page"
		if err := global.RegisterTranslationPackage(other); err != nil {
			t.Fatal(err)
		}
		s, err := readQualityCheckSnapshot(root, o.Locale, o.PreviousSnapshotID, false)
		if err != nil {
			t.Fatal(err)
		}
		if err := requireCurrentValidationBaseline(root, &global, o.Locale, s); err == nil || !strings.Contains(err.Error(), "exact-set count mismatch") {
			t.Fatal("fixture must reproduce unscoped baseline mismatch", err)
		}
		scoped, err := SelectCurrentValidationReopenPackage(&global, o.ContentPackage)
		if err != nil || len(scoped.TranslationScope) != 49 || len(global.TranslationScope) != 0 {
			t.Fatal("registered package selection mutated caller or lost exact set", err)
		}
		pkg, _, _, ordered, err := translationAuthorizationScope(scoped)
		if err != nil || pkg != o.ContentPackage || len(ordered) != 49 {
			t.Fatal("package authority scope", pkg, len(ordered), err)
		}
		for i, u := range ordered {
			if u.ID != ids[i] {
				t.Fatal("registered stable order lost", i, u.ID)
			}
		}
		explicit := o
		explicit.UnitIDs = ids[:9]
		r, _, err := InspectCurrentValidationReopen(root, &global, explicit)
		if err != nil || r.CurrentValidation.Package != o.ContentPackage || r.UnitCount != 9 {
			t.Fatal("global inventory leaked into package preflight", err)
		}
		x, err := decodeCurrentValidationAssessment(r.CurrentValidation)
		if err != nil || len(x.Units) != 49 || x.Compatible != 40 || x.Affected != 9 || x.Stale != 0 || x.Unknown != 0 {
			t.Fatal("scoped assessment impact", err)
		}
		for i, u := range r.Units {
			if u.UnitID != ids[i] {
				t.Fatal("affected order changed", i)
			}
		}
		for _, scope := range [][]string{ids[:48], append(append([]string{}, ids...), "other/page"), {"other/page"}} {
			partial := global
			if err := partial.UseTranslationUnitScope(scope); err != nil {
				t.Fatal(err)
			}
			if _, _, err := InspectCurrentValidationReopen(root, &partial, explicit); err == nil {
				t.Fatal("explicit partial/wrong scope widened", scope)
			}
		}
		wrong := explicit
		wrong.ContentPackage = other.ID
		if _, _, err := InspectCurrentValidationReopen(root, &global, wrong); err == nil {
			t.Fatal("wrong registered package accepted")
		}
	})
	t.Run("conflicting-successor-export", func(t *testing.T) {
		s, err := readQualityCheckSnapshot(root, o.Locale, o.PreviousSnapshotID, false)
		if err != nil {
			t.Fatal(err)
		}
		m := readRetranslationManifest(t, root, s.Units[0].SelectedBatchID)
		m.BatchID = "chatgpt-zh-CN-002"
		path := filepath.Join("data/retranslation-runs", o.Locale, m.BatchID)
		writeSnapshotSource(t, root, filepath.Join(path, "manifest.json"), mustJSON(m))
		defer os.RemoveAll(filepath.Join(root, path)) // synthetic TempDir only
		if _, _, err := InspectCurrentValidationReopen(root, c, o); err == nil || !strings.Contains(err.Error(), "already exported to conflicting successor") {
			t.Fatal("conflicting export accepted or rejected for unrelated reason", err)
		}
	})
	r, path, err := InspectCurrentValidationReopen(root, c, o)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, path)); !os.IsNotExist(err) {
		t.Fatal("preflight wrote formal receipt", err)
	}
	slots := 0
	for _, f := range r.CurrentValidation.Findings {
		slots += len(f.CommentSlots)
	}
	if r.UnitCount != 9 || slots != 41 || r.AuthoritySource != CurrentValidationAuthority || r.PreviousFinalizationSHA256 != sum(beforeFinal) {
		t.Fatal("wrong machine scope")
	}
	options := QualityCheckSurfaceReopenOptions{AuthoritySource: CurrentValidationAuthority, Locale: o.Locale, ReopenID: o.ReopenID, PreviousSnapshotID: o.PreviousSnapshotID, CurrentValidation: &o}
	sealed, recordPath, err := RecordQualityCheckSurfaceReopen(root, c, options)
	if err != nil {
		t.Fatal(err)
	}
	if recordPath != path || !reflect.DeepEqual(sealed, r) {
		t.Fatal("record differs from preflight")
	}
	if _, _, err := RecordQualityCheckSurfaceReopen(root, c, options); err == nil {
		t.Fatal("duplicate receipt accepted")
	}
	for _, name := range []string{"subset", "superset", "reason", "slot", "mixed-kind"} {
		t.Run("receipt-"+name, func(t *testing.T) {
			data := currentValidationReopenBytes(r)
			var bad QualityCheckSurfaceReopen
			_ = json.Unmarshal(data, &bad)
			switch name {
			case "subset":
				bad.Units = bad.Units[:8]
				bad.UnitCount = 8
			case "superset":
				bad.Units = append(bad.Units, bad.Units[0])
				bad.UnitCount++
			case "reason":
				bad.CurrentValidation.Findings[0].Reason = "unbound reason"
			case "slot":
				bad.CurrentValidation.Findings[0].CommentSlots[0].SlotID = "unbound-slot"
			case "mixed-kind":
				bad.Units[0].Snapshot.UnitKind = UnitKindExample
			}
			bad.CurrentValidation.Identity = currentValidationReopenIdentity(&bad)
			writeSnapshotSource(t, root, path, currentValidationReopenBytes(&bad))
			defer writeSnapshotSource(t, root, path, data)
			if _, err := CheckQualityCheckSurfaceReopen(root, c, o.Locale, o.ReopenID); err == nil {
				t.Fatal("tampered receipt accepted")
			}
		})
	}
	for _, scope := range [][]string{ids[:8], ids[:10], append([]string{"wrong-unit"}, ids[1:9]...)} {
		if _, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: o.Locale, Generator: RetranslationGeneratorChatGPT, UnitIDs: scope, AllowReexport: true, PreviousSnapshotID: o.PreviousSnapshotID, SurfaceReopenID: o.ReopenID}); err == nil {
			t.Fatal("wrong export scope accepted")
		}
	}
	if _, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: o.Locale, UnitIDs: ids[:9], AllowReexport: true, PreviousSnapshotID: "wrong-snapshot", SurfaceReopenID: o.ReopenID}); err == nil {
		t.Fatal("wrong export predecessor accepted")
	}
	export, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: o.Locale, Generator: RetranslationGeneratorChatGPT, UnitIDs: ids[:9], AllowReexport: true, PreviousSnapshotID: o.PreviousSnapshotID, SurfaceReopenID: o.ReopenID})
	if err != nil {
		t.Fatal(err)
	}
	manifest := readRetranslationManifest(t, root, export.BatchID)
	if len(manifest.Units) != 9 {
		t.Fatal("unaffected Pages exported")
	}
	for i, u := range manifest.Units {
		if u.UnitID != ids[i] || u.UnitKind != UnitKindPage || u.PreviousSnapshotID != o.PreviousSnapshotID || u.PreviousRating != "A" || u.RevisionFeedbackSource != CurrentValidationRevisionFeedback || u.PreviousFinding != r.Units[i].Finding {
			t.Fatal("machine finding/lineage lost", u)
		}
	}
	// Exercise the ordinary revision input reader after export has consumed the
	// authorization. Removing the transport file cannot erase frozen authority.
	if err := os.Remove(filepath.Join(root, o.Evidence.Path)); err != nil {
		t.Fatal(err)
	}
	if _, err := revisionGenerationInputs(root, c, o.Locale, manifest.Units); err != nil {
		t.Fatal("existing generation reader rejected authority", err)
	}
	if _, err := CheckQualityCheckSurfaceReopen(root, c, o.Locale, o.ReopenID); err == nil {
		t.Fatal("consumed authority accepted as unconsumed")
	}
	afterQC, _ := os.ReadFile(qcPath)
	afterFinal, _ := os.ReadFile(finalPath)
	if !bytes.Equal(beforeQC, afterQC) || !bytes.Equal(beforeFinal, afterFinal) {
		t.Fatal("historical A rewritten")
	}
	if output := os.Getenv("GO_TOUR_CURRENT_VALIDATION_REOPEN_FIXTURE_EVIDENCE"); output != "" {
		proof := map[string]any{
			"result": "PASS", "test": "TestCurrentValidationReopenAuthority", "scope": "TempDir synthetic repository only",
			"authority_dependency_identity_sha256": r.CurrentValidation.ImplementationIdentity,
			"total_units":                          49, "exported_units": 9, "unaffected_not_exported": 40, "exact_comment_slots": slots,
			"exported_unit_ids": ids[:9], "unaffected_unit_ids": ids[9:], "historical_a_immutable": true,
			"exact_scope_wrong_predecessor_stale_evidence_duplicate_and_tamper_controls": "PASS",
			"page_kind_constraint": "PASS", "consumed_generation_input_reader": "PASS", "frozen_evidence_survives_transport_removal": true,
		}
		data, err := json.MarshalIndent(proof, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_, writeErr := f.Write(append(data, '\n'))
		closeErr := f.Close()
		if writeErr != nil || closeErr != nil {
			t.Fatal(writeErr, closeErr)
		}
	}
}

func TestCurrentValidationReopenScopeLimit(t *testing.T) {
	root, c, o, _ := currentValidationFixture(t)
	o.UnitIDs = make([]string, 61)
	if _, _, err := InspectCurrentValidationReopen(root, c, o); err == nil {
		t.Fatal("oversized scope accepted")
	}
}
