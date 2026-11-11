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

func TestOrdinaryQCSelectedCurrentValidationPredecessor(t *testing.T) {
	checkApprovedIntervalBundleLifecycle(t, true, func(t *testing.T, root string, c *i18n.Catalog, reopen i18n.CurrentValidationReopenOptions, batch, restricted string, _ i18n.GenerationBundleOptions) {
		// The existing fixture includes one ordinary current-validation result,
		// one sealed Surface sibling, and one approved-interval sibling, all passed.
		base := "data/retranslation-runs/" + reopen.Locale + "/" + batch + "/"
		var m i18n.RetranslationBatchManifest
		var result i18n.RetranslationProcessResult
		if err := json.Unmarshal(mustReadFixture(t, root, base+"manifest.json"), &m); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(mustReadFixture(t, root, base+"result.json"), &result); err != nil {
			t.Fatal(err)
		}
		ordinary := c.TranslationScope[1]
		var record i18n.RetranslationBatchUnit
		var final i18n.RetranslationUnitResult
		ids := []string{}
		for _, r := range m.Units {
			ids = append(ids, r.UnitID)
			if r.UnitID == ordinary {
				record = r
			}
		}
		for _, r := range result.Units {
			if r.UnitID == ordinary {
				final = r
			}
		}
		if record.RevisionFeedbackSource != i18n.CurrentValidationRevisionFeedback || final.Status != "passed" {
			t.Fatal("fixture ordinary predecessor identity/status")
		}
		var v i18n.RetranslationValidation
		if err := json.Unmarshal(mustReadFixture(t, root, base+final.ValidationPath), &v); err != nil {
			t.Fatal(err)
		}
		raw := mustReadFixture(t, root, base+v.RawResponsePath)
		reviewed := mustReadFixture(t, root, base+final.CandidatePath)
		u, err := c.Unit(ordinary)
		if err != nil {
			t.Fatal(err)
		}
		got, err := u.Adapter.Restore(string(raw))
		if err != nil || !bytes.Equal(bridgeEOF(got), reviewed) {
			t.Fatal("fixture does not prove ordinary exact restore", err)
		}
		if err := i18n.ValidateTranslationUnitCandidate(root, c, ordinary, reopen.Locale, reviewed); err != nil {
			t.Fatal(err)
		}
		if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: reopen.Locale, SnapshotID: "approved-after-retry", PreviousSnapshotID: reopen.PreviousSnapshotID, UnitIDs: ids, Rating: "B", Finding: "Improve the ordinary prose.", ReviewerSession: "independent-selected-predecessor-reviewer"}); err != nil {
			t.Fatal(err)
		}
		implementation := "internal/i18n/generation_revision_inputs.go"
		writeFixture(t, root, implementation, append(bytes.Clone(mustReadFixture(t, root, implementation)), []byte("\n// synthetic code-owned implementation drift\n")...))
		// Full-batch execution must still reject the stale installed sibling proof.
		for name, call := range map[string]func() error{
			"process": func() error {
				// Model the pre-process fixture state without deleting any evidence.
				// Restore all synthetic outputs before the other assertions run.
				saved := t.TempDir()
				for _, name := range []string{"result.json", "candidates", "validation"} {
					from, to := filepath.Join(root, base, name), filepath.Join(saved, name)
					if err := os.Rename(from, to); err != nil {
						t.Fatal(err)
					}
					defer func() {
						if err := os.Rename(to, from); err != nil {
							t.Fatal("process mutated fixture output", err)
						}
					}()
				}
				_, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: reopen.Locale, BatchID: batch})
				return err
			},
			"retry": func() error {
				_, err := i18n.ProcessRetranslationRetry(root, c, i18n.RetranslationRetryOptions{Locale: reopen.Locale, BatchID: batch, UnitID: restricted})
				return err
			},
			"revalidate": func() error {
				_, err := i18n.RevalidateRetranslationCandidate(root, c, i18n.RetranslationRevalidateOptions{Locale: reopen.Locale, BatchID: batch, UnitID: restricted})
				return err
			},
		} {
			t.Run("strict-"+name, func(t *testing.T) {
				if err := call(); err == nil || !strings.Contains(err.Error(), "approved interval installed decoder/proof drift: formal/approved-interval-transport/"+restricted+".json") {
					t.Fatal("full batch lost strict installed-proof gate", err)
				}
			})
		}
		next, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: reopen.Locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: []string{ordinary}, AllowReexport: true, PreviousSnapshotID: "approved-after-retry"})
		if err != nil {
			t.Fatal("ordinary successor scheduling", err)
		}
		bundle, bm, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: reopen.Locale, BatchID: next.BatchID})
		if err != nil {
			t.Fatal("selected ordinary Bundle must not require unselected sibling proof", err)
		}
		if len(bm.ExpectedOutputs) != 1 || bm.ExpectedOutputs[0].UnitID != ordinary {
			t.Fatal("sibling output scope leaked", bm.ExpectedOutputs)
		}
		files, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
		if err != nil {
			t.Fatal(err)
		}
		for path := range files {
			if strings.HasPrefix(path, "formal/approved-interval-transport/") {
				t.Fatal("unselected sibling approved proof leaked", path)
			}
		}
		name := filepath.Base(record.InputPath)
		if !bytes.Equal(files["formal/protected-candidates/"+name], raw) || !bytes.Equal(files["formal/candidates/"+name], reviewed) {
			t.Fatal("ordinary editing baseline/candidate changed")
		}
		if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle); err != nil {
			t.Fatal("selected ordinary current-check", err)
		}
		t.Run("selected-restricted-still-requires-capability", func(t *testing.T) {
			next, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: reopen.Locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: []string{restricted}, AllowReexport: true, PreviousSnapshotID: "approved-after-retry"})
			if err != nil {
				t.Log("restricted scheduling rejected", err)
				return
			}
			data, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: reopen.Locale, BatchID: next.BatchID})
			if err == nil || len(data) != 0 {
				t.Fatal("restricted selected Unit bypassed capability gate", err)
			}
			t.Log(err)
		})
	})
}
