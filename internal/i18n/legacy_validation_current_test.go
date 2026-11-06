package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLegacyPageValidationCurrentFrozenProof(t *testing.T) {
	root, catalog, _ := makeRetranslationReviewBatchFixture(t, 1, "legacy-validation-fixture")
	manifest, err := readQualityCheckSnapshot(root, "zh-CN", "legacy-validation-fixture", false)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := manifest.Units[0]
	// Process fixtures omit locale metadata, but frozen compatibility proof
	// requires a complete glossary authority. Add metadata only in TempDir;
	// the exact terminology/protection rules remain unchanged.
	oldGlossary, err := readArchivedGlossary(root, "zh-CN", manifest.GlossarySHA256)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(oldGlossary, []byte("locale:")) {
		oldGlossary = append([]byte("locale: zh-CN\n"), oldGlossary...)
	}
	if _, err := archiveGlossaryBytes(root, "zh-CN", oldGlossary); err != nil {
		t.Fatal(err)
	}
	manifest.GlossarySHA256 = sum(oldGlossary)
	unit, err := catalog.Unit(snapshot.UnitID)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := readCompatibilityFile(root, snapshot.CandidatePath)
	if err != nil {
		t.Fatal(err)
	}
	check := func(u *TranslationUnit, s QualityCheckSnapshotUnit, target []byte) error {
		return verifyAcceptedFrozenInput(root, "zh-CN", u, s, target, manifest.GlossarySHA256, catalog)
	}
	if err := check(unit, snapshot, candidate); err != nil {
		t.Fatal("current-format baseline", err)
	}
	validationBytes, err := readCompatibilityFile(root, snapshot.ValidationPath)
	if err != nil {
		t.Fatal(err)
	}
	var current RetranslationValidation
	if err := json.Unmarshal(validationBytes, &current); err != nil {
		t.Fatal(err)
	}
	legacy := legacyRetranslationValidation{SchemaVersion: 1, BatchID: current.BatchID, Locale: current.Locale, PageID: unit.ID, Status: current.Status, InputPath: current.InputPath, RawResponsePath: current.RawResponsePath, CandidatePath: current.CandidatePath}
	base := filepath.Join(root, "data", "retranslation-runs", "zh-CN", snapshot.SelectedBatchID)
	batch, err := readRetranslationProcessManifest(base, "zh-CN", snapshot.SelectedBatchID)
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.Page(unit.ID)
	if err != nil {
		t.Fatal(err)
	}
	r := batch.Units[0]
	oldManifest := map[string]any{"schema_version": 1, "batch_id": batch.BatchID, "locale": batch.Locale, "protection_mode": "default", "translation_unit": "present.Section", "page_count": 1, "pages": []any{map[string]any{"page_id": unit.ID, "article": page.Article, "source_sha256": unit.SourceSHA256, "input_path": r.InputPath, "input_sha256": r.InputSHA256, "protected_token_count": r.ProtectedTokenCount}}}
	b, _ := json.Marshal(oldManifest)
	writeSnapshotSource(t, root, filepath.ToSlash(filepath.Join("data", "retranslation-runs", "zh-CN", snapshot.SelectedBatchID, "manifest.json")), b)
	legacyBytes, _ := json.Marshal(legacy)
	writeSnapshotSource(t, root, snapshot.ValidationPath, legacyBytes)
	snapshot.ValidationSHA256 = sum(legacyBytes)
	if err := check(unit, snapshot, candidate); err != nil {
		t.Fatal("exact schema-v1 Page proof", err)
	}
	glossary, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateTranslationUnitWithGlossary(root, catalog, unit, "zh-CN", candidate, glossary); err != nil {
		t.Fatal("legacy candidate current validation", err)
	}
	for _, label := range []string{"wrong-page-id", "unknown-member", "current-page-id", "wrong-batch", "wrong-locale", "wrong-candidate-path", "wrong-input-path", "wrong-source", "wrong-candidate", "wrong-hash", "wrong-attempt"} {
		t.Run(label, func(t *testing.T) {
			var value map[string]any
			if err := json.Unmarshal(legacyBytes, &value); err != nil {
				t.Fatal(err)
			}
			u, s, target := *unit, snapshot, candidate
			switch label {
			case "wrong-page-id":
				value["page_id"] = "different/identity"
			case "unknown-member":
				value["unexpected"] = true
			case "current-page-id":
				if err := json.Unmarshal(validationBytes, &value); err != nil {
					t.Fatal(err)
				}
				value["page_id"] = unit.ID
			case "wrong-batch":
				value["batch_id"] = "codex-zh-CN-999"
			case "wrong-locale":
				value["locale"] = "different-locale"
			case "wrong-candidate-path":
				value["candidate_path"] = "candidates/different.article"
			case "wrong-input-path":
				value["input_path"] = "inputs/different.article"
			case "wrong-source":
				u.SourceSHA256 = sum([]byte("wrong source"))
			case "wrong-candidate":
				target = append(append([]byte{}, candidate...), 'x')
			case "wrong-attempt":
				s.Attempt++
			}
			b, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			writeSnapshotSource(t, root, snapshot.ValidationPath, b)
			defer func() {
				if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(snapshot.ValidationPath)), legacyBytes, 0644); err != nil {
					t.Fatal(err)
				}
			}()
			if label != "wrong-hash" {
				s.ValidationSHA256 = sum(b)
			} else {
				s.ValidationSHA256 = sum([]byte("wrong validation hash"))
			}
			if err := check(&u, s, target); err == nil {
				t.Fatal("invalid legacy/current validation proof accepted")
			}
		})
	}
}
