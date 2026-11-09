package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Only the historical response decoder changes; source semantics and current
// candidate validation remain identical. No production bypass is involved.
type candidateSHADecoderEpoch struct {
	validationReopenAdapter
	calls *int
}

func (a candidateSHADecoderEpoch) Restore(string) ([]byte, error) {
	*a.calls++
	return nil, fmt.Errorf("historical response decoder is unavailable in the current epoch")
}

type candidateSHAValidatorFailure struct{ candidateSHADecoderEpoch }

func (a candidateSHAValidatorFailure) Validate([]byte, *Glossary) error {
	return fmt.Errorf("synthetic active validator failure")
}

type candidateSHAUnknownAuthority struct{ candidateSHADecoderEpoch }

func (a candidateSHAUnknownAuthority) ReattestationAuthority(string) (string, error) {
	return "", fmt.Errorf("unknown current adapter authority")
}

func TestCurrentValidationCandidateSHABinding(t *testing.T) {
	root, c, o, e := importedCommentProcessFixture(t)
	r, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: o.Locale, BatchID: e.BatchID})
	if err != nil || r.ValidationPassed != 9 {
		t.Fatal(r, err)
	}
	batchDir := filepath.Join(root, "data/retranslation-runs", o.Locale, e.BatchID)
	m, err := readRetranslationProcessManifest(batchDir, o.Locale, e.BatchID)
	if err != nil {
		t.Fatal(err)
	}
	old := map[string][]byte{}
	for _, record := range m.Units {
		path := "validation/" + strings.TrimSuffix(filepath.Base(record.InputPath), filepath.Ext(record.InputPath)) + ".json"
		b, err := os.ReadFile(filepath.Join(batchDir, path))
		if err != nil {
			t.Fatal(err)
		}
		var v RetranslationValidation
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatal(err)
		}
		candidate, err := os.ReadFile(filepath.Join(batchDir, v.CandidatePath))
		if err != nil || v.CandidateSHA256 != sum(candidate) {
			t.Fatal("process PASS did not bind exact candidate bytes", err)
		}
		// A legacy record is not upgraded merely by reading it. Revalidation
		// must create new evidence and archive these exact old bytes.
		v.CandidateSHA256 = ""
		if err := writeTranslationJSON(filepath.Join(batchDir, path), v); err != nil {
			t.Fatal(err)
		}
		old[record.UnitID], _ = os.ReadFile(filepath.Join(batchDir, path))
	}
	latest, err := selectLatestRetranslationUnits(root, c, o.Locale)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := c.Unit(m.Units[0].UnitID)
	if v, _, err := snapshotBoundCurrentValidationCandidate(root, c, o.Locale, u, latest.selectedByID[u.ID]); err != nil || v != nil {
		t.Fatal("legacy evidence acquired bound current eligibility", err)
	}
	calls := 0
	for i, binding := range c.Adapted {
		for _, record := range m.Units {
			if binding.Unit.ID == record.UnitID {
				binding.Adapter = candidateSHADecoderEpoch{binding.Adapter.(validationReopenAdapter), &calls}
				c.Adapted[i] = binding
				c.TranslationPackages[0].Units[i] = binding
			}
		}
	}
	for _, record := range m.Units {
		got, err := RevalidateRetranslationCandidate(root, c, RetranslationRevalidateOptions{Locale: o.Locale, BatchID: e.BatchID, UnitID: record.UnitID})
		if err != nil || got.Status != "passed" || got.Attempt != 1 {
			t.Fatal("current revalidation must not replay historical decoder", got, err)
		}
		history, _ := os.ReadFile(filepath.Join(batchDir, got.HistoryPath))
		if !bytes.Equal(history, old[record.UnitID]) {
			t.Fatal("old unbound evidence was rewritten")
		}
	}
	options := QualityCheckSnapshotOptions{Locale: o.Locale, SnapshotID: "bound-current-candidate"}
	var checks []func() error
	s, err := prepareQualityCheckCandidateSnapshot(root, c, options, &checks)
	if err != nil || s.UnitCount != 49 || calls != 0 {
		t.Fatal("Snapshot replayed historical decoder or lost exact scope", err, calls)
	}
	selected := 0
	for _, unit := range s.Units {
		if unit.SelectedBatchID == e.BatchID {
			selected++
		}
	}
	if selected != 9 {
		t.Fatal("Snapshot did not select current successors", selected)
	}
	latest, err = selectLatestRetranslationUnits(root, c, o.Locale)
	if err != nil {
		t.Fatal(err)
	}
	choice := latest.selectedByID[u.ID]
	u, _ = c.Unit(u.ID)
	g, _ := LoadGlossary(root, o.Locale)
	if _, err := validateSelectedRetranslationCandidate(root, c, o.Locale, g, u, choice); err == nil {
		t.Fatal("ordinary non-Snapshot restoration was relaxed")
	}
	candidatePath := filepath.Join(batchDir, choice.result.CandidatePath)
	validationPath := filepath.Join(batchDir, choice.result.ValidationPath)
	manifestPath := filepath.Join(batchDir, "manifest.json")
	resultPath := filepath.Join(batchDir, "result.json")
	candidate, _ := os.ReadFile(candidatePath)
	validation, _ := os.ReadFile(validationPath)
	manifest, _ := os.ReadFile(manifestPath)
	result, _ := os.ReadFile(resultPath)
	for _, mode := range []string{"valid-replacement", "corruption", "wrong-unit", "wrong-attempt", "invalid-sha", "validation-failed", "restore-failed", "wrong-result", "source-changed", "wrong-finding", "wrong-previous-snapshot", "unauthorized-successor", "scope-mismatch", "current-validator-failure", "unknown-authority"} {
		t.Run(mode, func(t *testing.T) {
			var v RetranslationValidation
			_ = json.Unmarshal(validation, &v)
			var m RetranslationBatchManifest
			_ = json.Unmarshal(manifest, &m)
			var r RetranslationProcessResult
			_ = json.Unmarshal(result, &r)
			defer os.WriteFile(candidatePath, candidate, 0644)
			defer os.WriteFile(validationPath, validation, 0644)
			defer os.WriteFile(manifestPath, manifest, 0644)
			defer os.WriteFile(resultPath, result, 0644)
			switch mode {
			case "valid-replacement":
				replacement := bytes.ReplaceAll(candidate, []byte("返回教学值。"), []byte("返回示例值。"))
				if bytes.Equal(replacement, candidate) {
					t.Fatal("fixture replacement was not different")
				}
				if err := ValidateTranslationUnitCandidate(root, c, u.ID, o.Locale, replacement); err != nil {
					t.Fatal("replacement must independently pass current validator", err)
				}
				_ = os.WriteFile(candidatePath, replacement, 0644)
			case "corruption":
				_ = os.WriteFile(candidatePath, []byte("broken candidate"), 0644)
			case "wrong-unit":
				v.UnitID = "wrong-unit"
			case "wrong-attempt":
				v.Attempt++
			case "invalid-sha":
				v.CandidateSHA256 = "invalid-sha"
			case "validation-failed", "restore-failed":
				v.Status, v.Error = strings.ReplaceAll(mode, "-", "_"), "synthetic failure"
				r.Units[0].Status, r.Units[0].Error = v.Status, v.Error
				if mode == "restore-failed" {
					v.CandidateSHA256, v.CandidatePath, r.Units[0].CandidatePath = "", "", ""
				}
				recountRetranslationResult(&r)
			case "wrong-result":
				r.BatchID = "wrong-batch"
			case "source-changed":
				path := filepath.Join(root, u.SourcePath)
				b, _ := os.ReadFile(path)
				defer os.WriteFile(path, b, 0644)
				_ = os.WriteFile(path, append(bytes.Clone(b), ' '), 0644)
			case "wrong-finding":
				m.Units[0].PreviousFinding = "unauthorized edit"
			case "wrong-previous-snapshot":
				m.Units[0].PreviousSnapshotID = "wrong-snapshot"
			case "unauthorized-successor":
				m.Units[0].RevisionAuthorizationID = "missing-reopen"
			case "scope-mismatch":
				m.Units = m.Units[:len(m.Units)-1]
				m.UnitCount--
			case "current-validator-failure", "unknown-authority":
				binding := c.Adapted[0]
				defer func() { c.Adapted[0], c.TranslationPackages[0].Units[0] = binding, binding }()
				changed := binding
				a := changed.Adapter.(candidateSHADecoderEpoch)
				if mode == "current-validator-failure" {
					changed.Adapter = candidateSHAValidatorFailure{a}
				} else {
					changed.Adapter = candidateSHAUnknownAuthority{a}
				}
				c.Adapted[0], c.TranslationPackages[0].Units[0] = changed, changed
			}
			if err := writeTranslationJSON(validationPath, v); err != nil {
				t.Fatal(err)
			}
			if err := writeTranslationJSON(manifestPath, m); err != nil {
				t.Fatal(err)
			}
			if err := writeTranslationJSON(resultPath, r); err != nil {
				t.Fatal(err)
			}
			if _, err := prepareQualityCheckCandidateSnapshot(root, c, options); err == nil {
				t.Fatal("invalid current evidence accepted", mode)
			}
			if mode == "valid-replacement" {
				if _, err := RevalidateRetranslationCandidate(root, c, RetranslationRevalidateOptions{Locale: o.Locale, BatchID: e.BatchID, UnitID: u.ID}); err == nil {
					t.Fatal("revalidation laundered replacement of already-bound bytes")
				}
			}
		})
	}
	for _, check := range checks {
		if err := check(); err != nil {
			t.Fatal("unchanged current evidence failed commit check", err)
		}
	}
}

func TestSnapshotCandidateBindingDoesNotAuthorizeOrdinaryFeedback(t *testing.T) {
	root, c, batchID, _, _, candidate := makeRevalidationFixture(t)
	got, err := RevalidateRetranslationCandidate(root, c, RetranslationRevalidateOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1"})
	if err != nil || got.Status != "passed" {
		t.Fatal(got, err)
	}
	latest, err := selectLatestRetranslationUnits(root, c, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	u, _ := c.Unit("lesson/1")
	choice := latest.selectedByID[u.ID]
	if v, _, err := snapshotBoundCurrentValidationCandidate(root, c, "zh-CN", u, choice); err != nil || v != nil {
		t.Fatal("ordinary feedback acquired current-validation eligibility", err)
	}
	validation, err := readPromotionValidation(choice.batchDir, batchID, "zh-CN", choice.manifest, choice.result)
	if err != nil || validation.CandidateSHA256 != sum(candidate) {
		t.Fatal("ordinary revalidation did not bind candidate SHA", err)
	}
	copy := *validation
	copy.CandidateSHA256 = sum([]byte("other candidate"))
	if sum(mustJSON(copy)) == sum(mustJSON(validation)) {
		t.Fatal("validation bytes identity did not cover candidate binding")
	}
	if err := writeTranslationJSON(filepath.Join(choice.batchDir, choice.result.ValidationPath), copy); err != nil {
		t.Fatal(err)
	}
	g, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validateSelectedRetranslationCandidate(root, c, "zh-CN", g, u, choice); err == nil {
		t.Fatal("ordinary candidate reader ignored an existing SHA binding")
	}
	validation.CandidateSHA256 = "wrong-sha"
	if _, err := decodeRetranslationValidation(mustJSON(validation), u); err == nil {
		t.Fatal("malformed SHA binding accepted")
	}
}
