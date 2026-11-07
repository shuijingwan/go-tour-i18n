package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

type TransportRecoveryCheck struct {
	Schema                string                     `json:"schema"`
	Status                string                     `json:"status"`
	Authority             TransportRecoveryReference `json:"authority"`
	Child                 *RecoveryFileReference     `json:"child_manifest"`
	ChildBatch            string                     `json:"child_batch,omitempty"`
	UnitID                string                     `json:"unit_id,omitempty"`
	Attempt               int                        `json:"attempt,omitempty"`
	AllowedNextOperation  *string                    `json:"allowed_next_operation"`
	HistoricalIntegrity   bool                       `json:"historical_integrity"`
	CurrentActionEligible bool                       `json:"current_action_eligible"`
	Blockers              []TransportRecoveryBlocker `json:"blockers"`
	Recorded              bool                       `json:"recorded"`
	Mutation              string                     `json:"mutation"`
}

func findTransportRecoveryChild(root string, a *InstalledDerivedTransportRecovery) (*RetranslationBatchManifest, error) {
	base := "data/retranslation-runs/" + a.Locale
	entries, err := os.ReadDir(filepath.Join(root, base))
	if err != nil {
		return nil, err
	}
	var child *RetranslationBatchManifest
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if !entry.IsDir() {
			return nil, fmt.Errorf("unexpected retranslation entry")
		}
		b, err := readCompatibilityFile(root, base+"/"+entry.Name()+"/manifest.json")
		if err != nil {
			return nil, err
		}
		var header struct {
			TransportRecovery *TransportRecoveryReference `json:"transport_recovery"`
		}
		// Header discovery is only an index over registered manifests; the full
		// matching child is subsequently strict-validated. It grants no authority.
		if err = decodeRecoveryManifestHeader(b, &header); err != nil {
			return nil, err
		}
		if header.TransportRecovery == nil || header.TransportRecovery.Identity != a.Identity {
			continue
		}
		m, err := readRetranslationProcessManifest(filepath.Join(root, base, entry.Name()), a.Locale, entry.Name())
		if err != nil {
			return nil, err
		}
		if child != nil {
			return nil, fmt.Errorf("duplicate recovery child")
		}
		if err = validateRecoveryChildManifest(a, m); err != nil {
			return nil, err
		}
		child = m
	}
	return child, nil
}

func decodeRecoveryManifestHeader(b []byte, v any) error { return jsonUnmarshalRecoveryHeader(b, v) }

func validateRecoveryChildManifest(a *InstalledDerivedTransportRecovery, m *RetranslationBatchManifest) error {
	if m.TransportRecovery == nil || *m.TransportRecovery != recoveryAuthorityRef(a, mustJSON(a)) || m.Locale != a.Locale || len(m.Units) != len(a.RecoveryUnits.UnitIDs) || m.UnitCount != len(m.Units) {
		return fmt.Errorf("recovery child scope/ref mismatch")
	}
	for i, r := range m.Units {
		old := a.LanguageAuthority.Scope[i]
		if r.ReexportReason != transportRecoveryReason || r.PreviousBatchID != a.Predecessor.BatchID || r.PreviousInputPath != old.InputPath || r.PreviousInputSHA256 != old.InputSHA256 || r.PreviousProtectedTokens != old.ProtectedTokenCount {
			return fmt.Errorf("recovery child previous invocation mismatch")
		}
		r.ReexportReason, r.PreviousBatchID, r.PreviousInputPath, r.PreviousInputSHA256, r.PreviousProtectedTokens = old.ReexportReason, old.PreviousBatchID, old.PreviousInputPath, old.PreviousInputSHA256, old.PreviousProtectedTokens
		if !reflect.DeepEqual(r, old) {
			return fmt.Errorf("recovery child altered language finding/source/protection")
		}
	}
	return nil
}

func CheckTransportRecovery(root string, c *Catalog, locale, pkg string, ref TransportRecoveryReference) (*TransportRecoveryCheck, error) {
	r := &TransportRecoveryCheck{Schema: "go-learning/transport-recovery-plan/v1", Status: "BLOCKED", Authority: ref, Mutation: "none", Blockers: []TransportRecoveryBlocker{}}
	fail := func(code string, err error) (*TransportRecoveryCheck, error) {
		r.CurrentActionEligible = false
		r.AllowedNextOperation = nil
		if code == "BLOCKED" {
			r.HistoricalIntegrity = false
		}
		r.Status = code
		r.Blockers = append(r.Blockers, TransportRecoveryBlocker{Code: code, Detail: err.Error()})
		return r, err
	}
	a, _, err := readRecoveryExactAuthority(root, ref)
	if err != nil {
		return fail("BLOCKED", err)
	}
	r.HistoricalIntegrity = true
	if err = verifyRecoveryHistoricalBytes(root, a); err != nil {
		return fail("BLOCKED", err)
	}
	if a.Locale != locale || a.ContentPackage.ID != pkg {
		return fail("BLOCKED", fmt.Errorf("recovery authority locale/package mismatch"))
	}
	c, err = recoveryPackageCatalog(c, pkg)
	if err != nil {
		return fail("BLOCKED", err)
	}
	if err = recoveryRequireAbsent(root, recoveryRootDirectory(locale)+"/supersessions/"+a.Identity+".json"); err != nil {
		return fail("STALE", fmt.Errorf("recovery authority superseded"))
	}
	if err = recoveryRequireAbsent(root, recoveryRecordPending(a)); err != nil {
		return fail("BLOCKED", err)
	}
	child, err := findTransportRecoveryChild(root, a)
	if err != nil {
		return fail("BLOCKED", err)
	}
	if err = recoveryRequireAbsent(root, transportRecoveryDirectory(a.Locale, a.Identity)+"/.export.pending"); err != nil {
		return fail("BLOCKED", err)
	}
	childID := ""
	if child != nil {
		childID = child.BatchID
	}
	fresh, _, err := prepareTransportRecovery(root, c, TransportRecoveryOptions{Locale: locale, ContentPackage: pkg, PredecessorBatch: a.Predecessor.BatchID, Supersedes: a.Supersedes}, childID)
	if err != nil || fresh == nil || fresh.Identity != a.Identity {
		return fail("STALE", fmt.Errorf("recorded construction/assessment epoch drift: %v", err))
	}
	r.CurrentActionEligible = true
	next := "retranslation-export-with-transport-recovery"
	r.Status = "READY_FOR_EXPORT"
	if child == nil {
		r.AllowedNextOperation = &next
		return r, nil
	}
	r.ChildBatch = child.BatchID
	base := "data/retranslation-runs/" + locale + "/" + child.BatchID + "/"
	manifestRef, _, err := recoveryDiskReference(root, base+"manifest.json")
	if err != nil {
		return fail("BLOCKED", err)
	}
	r.Child = &manifestRef
	if err = ensureNoGenerationImportPending(filepath.Join(root, base)); err != nil {
		return fail("BLOCKED", err)
	}
	_, err = readCompatibilityFile(root, base+"generation-imports/initial-manifest.json")
	if os.IsNotExist(err) {
		r.Status = "EXPORTED"
		next = "verify-or-export-child-generation-bundle"
		r.AllowedNextOperation = &next
		return r, nil
	}
	if err != nil {
		return fail("BLOCKED", err)
	}
	outputs := map[string]protectedTranslation{}
	inputs, err := approvedResponseInputs(root, c, child, outputs)
	if err != nil {
		return fail("BLOCKED", err)
	}
	if err = checkApprovedIntervalInputs(root, inputs); err != nil {
		return fail("STALE", err)
	}
	_, err = readCompatibilityFile(root, base+"result.json")
	if os.IsNotExist(err) {
		r.Status = "IMPORTED"
		r.Attempt = 1
		next = "retranslation-process"
		r.AllowedNextOperation = &next
		return r, nil
	}
	if err != nil {
		return fail("BLOCKED", err)
	}
	result, err := readPromotionResult(filepath.Join(root, base), locale, child.BatchID, child.UnitCount)
	if err != nil {
		return fail("BLOCKED", err)
	}
	r.Status = "VALIDATED"
	next = "quality-check-snapshot-preflight"
	for _, u := range result.Units {
		if u.Status == "passed" {
			continue
		}
		if u.Status != "restore_failed" && u.Status != "validation_failed" {
			return fail("BLOCKED", fmt.Errorf("unknown recovery result status"))
		}
		r.UnitID = u.UnitID
		r.Status = strings.ToUpper(u.Status)
		r.Attempt, err = nextRecoveryAttempt(root, child, u.UnitID)
		if err != nil {
			return fail("BLOCKED", err)
		}
		next = "generation-bundle-export-exact-retry"
		path := base + "retries/" + u.UnitID + "/attempt-" + fmt.Sprintf("%03d", r.Attempt) + filepath.Ext(recoveryRecordForUnit(child, u.UnitID).InputPath)
		if _, err = readCompatibilityFile(root, path); err == nil {
			receipt := generationImportReceiptPath(filepath.Join(root, base), GenerationBundleManifest{TaskKind: "translation-unit-retry", Attempt: r.Attempt, ExpectedOutputs: []GenerationBundleExpectedOutput{{UnitID: u.UnitID}}})
			rel, err := repositoryRelativePath(root, receipt)
			if err != nil {
				return fail("BLOCKED", err)
			}
			if _, err = readCompatibilityFile(root, rel); err != nil {
				return fail("BLOCKED", fmt.Errorf("retry raw has no exact import provenance: %w", err))
			}
			r.Status = "IMPORTED"
			next = "retranslation-retry"
		} else if !os.IsNotExist(err) {
			return fail("BLOCKED", err)
		}
		break
	}
	if r.Status == "VALIDATED" {
		latest, err := selectLatestRetranslationUnits(root, c, locale)
		if err != nil {
			return fail("BLOCKED", err)
		}
		g, err := LoadGlossary(root, locale)
		if err != nil {
			return fail("BLOCKED", err)
		}
		for _, record := range child.Units {
			u, err := c.Unit(record.UnitID)
			if err != nil {
				return fail("BLOCKED", err)
			}
			choice, ok := latest.selectedByID[u.ID]
			if !ok || choice.batchID != child.BatchID {
				return fail("STALE", fmt.Errorf("validated recovery is no longer the current full-R selection"))
			}
			p, ok := outputs[u.ID]
			if !ok {
				p, err = prepareTranslationUnitInput(u, g)
				if err != nil {
					return fail("BLOCKED", err)
				}
			}
			if _, err = validateSelectedRetranslationCandidateWithOutput(root, c, locale, g, u, choice, &p); err != nil {
				return fail("BLOCKED", err)
			}
		}
	}
	r.AllowedNextOperation = &next
	return r, nil
}

func recoveryRecordForUnit(m *RetranslationBatchManifest, id string) RetranslationBatchUnit {
	for _, r := range m.Units {
		if r.UnitID == id {
			return r
		}
	}
	return RetranslationBatchUnit{}
}

func nextRecoveryAttempt(root string, m *RetranslationBatchManifest, id string) (int, error) {
	r := recoveryRecordForUnit(m, id)
	b, err := readCompatibilityFile(root, "data/retranslation-runs/"+m.Locale+"/"+m.BatchID+"/validation/"+id+".json")
	if err != nil {
		return 0, err
	}
	var v RetranslationValidation
	if err = contentidentity.StrictJSON(b, &v); err != nil {
		return 0, err
	}
	n, err := retryValidationAttemptForExtension(v.RawResponsePath, stringsRecoveryInputBase(r.InputPath), filepath.Ext(r.InputPath))
	return n + 1, err
}

// Recovery successor export has a separate authority entry. The ordinary
// exporter's consumed-reopen gate is intentionally not relaxed.
func ExportTransportRecoveryBatch(root string, c *Catalog, locale, pkg string, generator RetranslationGenerator, ref TransportRecoveryReference, resume bool) (*RetranslationExportResult, error) {
	unlock, err := lockRetranslationBatchNamespace(root)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if resume {
		a, _, err := readRecoveryExactAuthority(root, ref)
		if err != nil {
			return nil, err
		}
		if a.Locale != locale || a.ContentPackage.ID != pkg || string(generator) != a.AllowedAction.Provider {
			return nil, fmt.Errorf("recovery resume scope/provider mismatch")
		}
		return resumeTransportRecoveryBatch(root, c, a, ref)
	}
	check, err := CheckTransportRecovery(root, c, locale, pkg, ref)
	if err != nil {
		return nil, err
	}
	if check.Status != "READY_FOR_EXPORT" {
		return nil, fmt.Errorf("recovery already has child or is not export eligible")
	}
	a, _, err := readRecoveryExactAuthority(root, ref)
	if err != nil {
		return nil, err
	}
	if string(generator) != a.AllowedAction.Provider {
		return nil, fmt.Errorf("recovery provider must remain selected provider")
	}
	c, err = recoveryPackageCatalog(c, pkg)
	if err != nil {
		return nil, err
	}
	base := filepath.Join(root, "data/retranslation-runs", locale)
	_, max, err := scanRetranslationBatches(base, locale, c)
	if err != nil {
		return nil, err
	}
	id := string(generator) + "-" + locale + "-" + fmt.Sprintf("%03d", max)
	m := &RetranslationBatchManifest{TransportRecovery: &ref, SchemaVersion: 2, BatchID: id, Locale: locale, ProtectionMode: "default", ArtifactEOF: retranslationArtifactEOFSingleLF, UnitKind: a.LanguageAuthority.Scope[0].UnitKind, UnitCount: len(a.RecoveryUnits.UnitIDs)}
	for _, old := range a.LanguageAuthority.Scope {
		r := old
		r.ReexportReason = transportRecoveryReason
		r.PreviousBatchID = a.Predecessor.BatchID
		r.PreviousInputPath = old.InputPath
		r.PreviousInputSHA256 = old.InputSHA256
		r.PreviousProtectedTokens = old.ProtectedTokenCount
		m.Units = append(m.Units, r)
	}
	if err = validateRetranslationManifestReexportProvenance(m); err != nil {
		return nil, err
	}
	// Publication is delegated to the sealed export transaction. Its ordinal
	// lock rechecks max+1 and every source/input immediately before commit.
	return publishTransportRecoveryBatch(root, c, a, m, ref)
}

// Keep header decoding separate from strict authority/member decoding: legacy
// schema-v1 manifests remain legal inputs to the shared batch scanner.
func jsonUnmarshalRecoveryHeader(b []byte, v any) error { return json.Unmarshal(b, v) }
