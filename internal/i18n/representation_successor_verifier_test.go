package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestRepresentationSuccessorVerifier(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, c, _, plan, ref, receiptBytes := authorizationMachineFixture(t)
	p := plan.Receipt.Units[0].Replacement
	v := NewRepresentationSuccessorVerifier(root, c, "zh-CN")
	if got, refs, err := v.Verify(p.CurrentUnitID, p.CurrentUnitID); err != nil || got != nil || len(refs) != 0 {
		t.Fatal("unchanged", got, err)
	}
	got, refs, err := v.Verify(p.OldUnitID, p.CurrentUnitID)
	if err != nil || !reflect.DeepEqual(got, &p) || len(refs) == 0 {
		t.Fatal("exact applied successor", got, err)
	}
	for _, ref := range refs {
		b, err := readSnapshotReferencedFile(root, ref.Path)
		if err != nil || sum(b) != ref.SHA256 {
			t.Fatal("incomplete immutable proof", ref, err)
		}
	}
	for _, tc := range []struct{ name, old, current string }{
		{"wrong-old", strings.Repeat("1", 64), p.CurrentUnitID},
		{"wrong-current", p.OldUnitID, strings.Repeat("2", 64)},
		{"unrelated-current", p.OldUnitID, plan.Receipt.Units[1].Replacement.CurrentUnitID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := v.Verify(tc.old, tc.current); err == nil {
				t.Fatal("guessed successor accepted")
			}
		})
	}
	for name, path := range map[string]string{"receipt-tamper": ref.Path, "validation-tamper": plan.Receipt.Units[0].Validation.Path, "candidate-drift": plan.Receipt.Units[0].Candidate.Path, "incomplete-apply": plan.Receipt.Units[0].CurrentRaw.Path} {
		t.Run(name, func(t *testing.T) {
			b, err := readSnapshotReferencedFile(root, path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeSnapshotSource(t, root, path, b)
			writeSnapshotSource(t, root, path, append(bytes.Clone(b), []byte("tamper")...))
			if _, _, err := v.Verify(p.OldUnitID, p.CurrentUnitID); err == nil {
				t.Fatal("warm verifier accepted tamper")
			}
		})
	}
	for _, kind := range []string{"old-id", "current-id", "source-path", "source-sha", "stable-index", "route", "old-input", "current-input", "reconciliation-sha", "validation-identity", "noncanonical-apply-option", "noncanonical-bytes"} {
		t.Run(kind, func(t *testing.T) {
			var r RepresentationReconciliation
			if err := contentidentity.StrictJSON(receiptBytes, &r); err != nil {
				t.Fatal(err)
			}
			u := &r.Units[0]
			switch kind {
			case "old-id":
				u.Replacement.OldUnitID = strings.Repeat("3", 64)
			case "current-id":
				u.Replacement.CurrentUnitID = strings.Repeat("4", 64)
			case "source-path":
				u.Replacement.SourcePath = "wrong/source.xml"
			case "source-sha":
				u.Replacement.SourceSHA256 = strings.Repeat("5", 64)
			case "stable-index":
				u.Replacement.StableIndex++
			case "route":
				u.Replacement.Route = "/wrong/"
			case "old-input":
				u.Replacement.OldInputSHA256 = strings.Repeat("6", 64)
			case "current-input":
				u.Replacement.CurrentInputSHA256 = strings.Repeat("7", 64)
			case "reconciliation-sha":
				u.Replacement.ReconciliationSHA256 = strings.Repeat("8", 64)
			case "validation-identity":
				u.Validation.SHA256 = strings.Repeat("9", 64)
			case "noncanonical-apply-option":
				r.Options.Apply = true
			}
			r.Identity = machineReceiptIdentity(r)
			b := append(mustJSON(r), '\n')
			if kind == "noncanonical-bytes" {
				b = append(b, '\n')
			}
			writeSnapshotSource(t, root, ref.Path, b)
			defer writeSnapshotSource(t, root, ref.Path, receiptBytes)
			if _, _, err := NewRepresentationSuccessorVerifier(root, c, "zh-CN").Verify(p.OldUnitID, p.CurrentUnitID); err == nil {
				t.Fatal("rehash forged receipt accepted", kind)
			}
		})
	}
	t.Run("missing-materialized-artifact", func(t *testing.T) {
		path := filepath.Join(root, filepath.FromSlash(plan.Receipt.Units[0].CurrentRaw.Path))
		if err := os.Rename(path, path+".fixture-backup"); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := os.Rename(path+".fixture-backup", path); err != nil {
				t.Fatal(err)
			}
		}()
		if _, _, err := NewRepresentationSuccessorVerifier(root, c, "zh-CN").Verify(p.OldUnitID, p.CurrentUnitID); err == nil {
			t.Fatal("unmaterialized preflight receipt accepted as applied")
		}
	})
	t.Run("current-source-drift", func(t *testing.T) {
		u, err := c.Unit(p.CurrentUnitID)
		if err != nil {
			t.Fatal(err)
		}
		// The catalog holds physical source bytes; mutate and restore the shared
		// backing bytes to exercise the verifier after a successful warm read.
		before := bytes.Clone(u.Source)
		u.Source[0] ^= 1
		defer copy(u.Source, before)
		if _, _, err := v.Verify(p.OldUnitID, p.CurrentUnitID); err == nil {
			t.Fatal("current source drift accepted")
		}
	})
	t.Run("missing-reconciliation-no-candidate-fallback", func(t *testing.T) {
		dir := filepath.Join(root, "data/translation-unit-reconciliations/zh-CN")
		moved := dir + "-fixture-backup"
		if err := os.Rename(dir, moved); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := os.Rename(moved, dir); err != nil {
				t.Fatal(err)
			}
		}()
		if _, _, err := NewRepresentationSuccessorVerifier(root, c, "zh-CN").Verify(p.OldUnitID, p.CurrentUnitID); err == nil {
			t.Fatal("old candidate guessed as successor")
		}
	})
	t.Run("historical-proof-does-not-grant-current-eligibility", func(t *testing.T) {
		corpus, err := readSnapshotReferencedFile(root, GlossaryCorpusPath)
		if err != nil {
			t.Fatal(err)
		}
		defer writeSnapshotSource(t, root, GlossaryCorpusPath, corpus)
		var successor GlossarySourceCorpus
		if err := json.Unmarshal(corpus, &successor); err != nil {
			t.Fatal(err)
		}
		successor.Contributors[0].Contexts[0].Text += " Newly adopted canonical prose."
		successor.Contributors[0].SourceSHA = ContributorIdentity(successor.Contributors[0])
		successor.Identity = CorpusIdentity(successor)
		if err := ValidateGlossaryCorpus(&successor); err != nil {
			t.Fatal(err)
		}
		writeSnapshotSource(t, root, GlossaryCorpusPath, mustJSON(successor))
		if err := RequireCurrentGlossaryReview(root, "zh-CN"); err == nil || !strings.Contains(err.Error(), "Review") {
			t.Fatal("new corpus must stale the old full Review", err)
		}
		verify := func() error {
			got, refs, err := v.VerifyHistorical(p.OldUnitID, p.CurrentUnitID)
			if err == nil && (!reflect.DeepEqual(got, &p) || len(refs) == 0) {
				t.Fatal("historical identity/proof lost", got, refs)
			}
			return err
		}
		if err := verify(); err != nil {
			t.Fatal("intact frozen proof must survive a new corpus", err)
		}
		// Even a warmed historical verifier must not grant current selection,
		// reconciliation, or actual Generation export eligibility.
		if _, _, err := v.Verify(p.OldUnitID, p.CurrentUnitID); err == nil || !strings.Contains(err.Error(), "Review") {
			t.Fatal("historical context bypassed current verifier gate", err)
		}
		if _, err := ReconcileTranslationUnitRepresentations(root, c, plan.Receipt.Options); err == nil || !strings.Contains(err.Error(), "Review") {
			t.Fatal("historical proof bypassed reconciliation eligibility", err)
		}
		if _, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitIDs: []string{p.CurrentUnitID}}); err == nil || !strings.Contains(err.Error(), "Review") {
			t.Fatal("historical proof bypassed Generation eligibility", err)
		}
		archive, _ := glossaryArchivePath("zh-CN", plan.Receipt.GlossarySHA256)
		for name, path := range map[string]string{
			"receipt": ref.Path, "raw": plan.Receipt.Units[0].CurrentRaw.Path,
			"candidate": plan.Receipt.Units[0].Candidate.Path, "validation": plan.Receipt.Units[0].Validation.Path,
			"original-raw":        plan.Receipt.Units[0].OriginalRaw.Path,
			"original-candidate":  plan.Receipt.Units[0].Predecessor.CandidatePath,
			"original-validation": plan.Receipt.Units[0].Predecessor.ValidationPath,
			"representation":      plan.Receipt.Options.HistoricalDocuments.Path, "archived-glossary": archive,
		} {
			t.Run(name, func(t *testing.T) {
				b, err := readSnapshotReferencedFile(root, path)
				if err != nil {
					t.Fatal(err)
				}
				defer writeSnapshotSource(t, root, path, b)
				writeSnapshotSource(t, root, path, append(bytes.Clone(b), []byte("tamper")...))
				// Fresh invocation as well as warm proof must reject corruption.
				if err := verify(); err == nil {
					t.Fatal("warm historical proof accepted corruption", name)
				}
				if _, _, err := NewRepresentationSuccessorVerifier(root, c, "zh-CN").VerifyHistorical(p.OldUnitID, p.CurrentUnitID); err == nil {
					t.Fatal("historical proof accepted corruption", name)
				}
			})
		}
		glossaryPath := glossaryReviewGlossaryPath("zh-CN")
		glossary, err := readSnapshotReferencedFile(root, glossaryPath)
		if err != nil {
			t.Fatal(err)
		}
		defer writeSnapshotSource(t, root, glossaryPath, glossary)
		writeSnapshotSource(t, root, glossaryPath, append(bytes.Clone(glossary), []byte("\nforbidden:\n  - apple\n")...))
		if err := verify(); err != nil {
			t.Fatal("historical reconstruction read mutable current glossary", err)
		}
	})
}

func TestHistoricalSurfaceRevisionLoadsAppliedSuccessor(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, c, snapshot, plan, receiptRef, _ := authorizationMachineFixture(t)
	resolution, err := replacementResolution(root, c, &TranslationUnitReplacementContext{HistoricalDocuments: plan.Receipt.Options.HistoricalDocuments})
	if err != nil {
		t.Fatal(err)
	}
	e, err := readSnapshotUnitRepositoryEvidence(root, resolution.HistoricalCatalog, "zh-CN", snapshot.Units[0])
	if err != nil {
		t.Fatal(err)
	}
	bytes := []byte("Synthetic frozen historical evidence\n")
	writeSnapshotSource(t, root, "history-evidence.md", bytes)
	prior := QualityCheckSurfaceReopen{
		SchemaVersion: qualityCheckSurfaceReopenSchemaVersion, EvidenceType: qualityCheckSurfaceReopenEvidenceType, Locale: "zh-CN", ReopenID: "historical-prior", PreviousSnapshotID: snapshot.SnapshotID,
		PreviousFinalizationPath: "history-evidence.md", PreviousFinalizationSHA256: sum(bytes), SurfaceReviewPath: "history-evidence.md", SurfaceReviewSHA256: sum(bytes), Finding: "Synthetic exact finding", UnitCount: 2,
		Units: []QualityCheckSurfaceReopenUnit{{Index: 1, UnitID: snapshot.Units[0].UnitID, Snapshot: snapshot.Units[0]}, {Index: 2, UnitID: snapshot.Units[1].UnitID, Snapshot: snapshot.Units[1]}},
	}
	priorPath := "data/quality-check-reopens/zh-CN/" + prior.ReopenID + ".json"
	priorBytes := mustJSON(prior)
	writeSnapshotSource(t, root, priorPath, priorBytes)
	reopen := prior
	reopen.ReopenID = "historical-event"
	reopen.Units = append([]QualityCheckSurfaceReopenUnit(nil), prior.Units...)
	reopen.ReplacementContext = &TranslationUnitReplacementContext{HistoricalDocuments: plan.Receipt.Options.HistoricalDocuments, PredecessorReopen: GlossaryArchiveReference{priorPath, sum(priorBytes)}}
	// One explicit event-time replacement authorizes this mixed historical
	// reopen. The first Unit was unchanged then and acquired its machine edge
	// later; its frozen Snapshot intentionally has no reconciliation field.
	p := plan.Receipt.Units[1].Replacement
	reopen.Units[1].UnitID, reopen.Units[1].Replacement = p.CurrentUnitID, &p
	writeSnapshotSource(t, root, "data/quality-check-reopens/zh-CN/"+reopen.ReopenID+".json", mustJSON(reopen))
	record := e.record
	record.RevisionAuthorizationID, record.PreviousSnapshotID = reopen.ReopenID, reopen.PreviousSnapshotID
	record.RevisionFeedbackSource = reopen.revisionFeedbackSource()
	evidence := reopen.revisionEvidence()
	record.PreviousReviewPath, record.PreviousReviewSHA256 = evidence.Path, evidence.SHA256
	record.PreviousFinding = reopen.findingForUnit(record.UnitID)
	bound := map[string]string{}
	bind := func(path, sha string) ([]byte, error) {
		b, err := readSnapshotReferencedFile(root, path)
		if err != nil {
			return nil, err
		}
		if sha != "" && sum(b) != sha {
			return nil, os.ErrInvalid
		}
		bound[path] = sum(b)
		return b, nil
	}
	if err := bindHistoricalSurfaceRevision(root, "zh-CN", c, record, bind, map[TranslationUnitReplacementContext]*TranslationUnitReplacementResolution{}, nil, NewRepresentationSuccessorVerifier(root, c, "zh-CN")); err != nil {
		t.Fatal("old Snapshot did not consume the formal successor", err)
	}
	corpus, err := readSnapshotReferencedFile(root, GlossaryCorpusPath)
	if err != nil {
		t.Fatal(err)
	}
	var successor GlossarySourceCorpus
	if err := json.Unmarshal(corpus, &successor); err != nil {
		t.Fatal(err)
	}
	successor.Contributors[0].Contexts[0].Text += " Newly adopted source content."
	successor.Contributors[0].SourceSHA = ContributorIdentity(successor.Contributors[0])
	successor.Identity = CorpusIdentity(successor)
	writeSnapshotSource(t, root, GlossaryCorpusPath, mustJSON(successor))
	if err := RequireCurrentGlossaryReview(root, "zh-CN"); err == nil {
		t.Fatal("fixture old Review must be stale")
	}
	if err := bindHistoricalSurfaceRevision(root, "zh-CN", c, record, bind, map[TranslationUnitReplacementContext]*TranslationUnitReplacementResolution{}, nil, NewRepresentationSuccessorVerifier(root, c, "zh-CN")); err != nil {
		t.Fatal("historical finalization reader re-entered current eligibility", err)
	}
	refs := machineAuthorityReferences(&machineSelection{ref: receiptRef, receipt: plan.Receipt, unit: plan.Receipt.Units[0]})
	for _, q := range plan.Receipt.QCInputs {
		refs = append(refs, GlossaryArchiveReference{q.Path, q.SHA256})
	}
	for _, ref := range refs {
		if bound[ref.Path] != ref.SHA256 {
			t.Fatal("machine proof omitted from immutable history", ref)
		}
	}
}
