package i18n

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

var transportRecoveryImplementationPaths = []string{
	"internal/i18n/transport_recovery_origins.go",
	"internal/i18n/transport_recovery_response.go", "internal/i18n/transport_recovery_integrity.go",
	"internal/i18n/transport_recovery_language.go",
	"internal/i18n/retranslation_export_lock.go",
	"internal/i18n/transport_recovery_drift.go",
	"internal/i18n/transport_recovery_schema.go", "internal/i18n/transport_recovery_construction.go", "internal/i18n/transport_recovery_preflight.go", "internal/i18n/transport_recovery_record.go", "internal/i18n/transport_recovery_child.go", "internal/i18n/transport_recovery_export.go", "internal/i18n/transport_recovery_execution.go", "internal/i18n/transport_recovery_context.go", "internal/i18n/transport_recovery_inventory.go", "internal/sitecontent/transport_recovery_codec.go",
}

func currentRecoveryInvocationAuthority(root string, c *Catalog, m *RetranslationBatchManifest) (*InstalledDerivedTransportRecovery, map[string][]byte, error) {
	if err := ensureNoGenerationImportPending(filepath.Join(root, "data/retranslation-runs", m.Locale, m.BatchID)); err != nil {
		return nil, nil, err
	}
	if m.TransportRecovery == nil {
		return nil, nil, fmt.Errorf("recovery manifest authority missing")
	}
	a, members, err := readRecoveryExactAuthority(root, *m.TransportRecovery)
	if err != nil {
		return nil, nil, err
	}
	if err = validateRecoveryChildManifest(a, m); err != nil {
		return nil, nil, err
	}
	if err = recoveryRequireAbsent(root, recoveryRecordPending(a)); err != nil {
		return nil, nil, err
	}
	if err = recoveryRequireAbsent(root, transportRecoveryDirectory(a.Locale, a.Identity)+"/.export.pending"); err != nil {
		return nil, nil, err
	}
	if err = recoveryRequireAbsent(root, recoveryRootDirectory(a.Locale)+"/supersessions/"+a.Identity+".json"); err != nil {
		return nil, nil, fmt.Errorf("recovery authority superseded")
	}
	// After this complete child passed, later independent QC/Surface revisions
	// may replace some latest selections. Reading its still-current full decoder
	// does not renew export rights or grant candidate eligibility; each consumer
	// retains its own exact Snapshot/latest/active-validation gates.
	var recordedSelections []RecoveryDependency
	for _, r := range a.CurrentEpoch {
		if r.Role == "latest-selection" {
			recordedSelections = append(recordedSelections, r)
		}
	}
	fresh, _, err := prepareTransportRecovery(root, c, TransportRecoveryOptions{Locale: a.Locale, ContentPackage: a.ContentPackage.ID, PredecessorBatch: a.Predecessor.BatchID, Supersedes: a.Supersedes, decoderContext: recoveryChildAllPassed(root, m), contextEpoch: recordedSelections}, m.BatchID)
	if err != nil || fresh == nil || fresh.Identity != a.Identity {
		return nil, nil, fmt.Errorf("recovery invocation construction/authority epoch stale: %v", err)
	}
	return a, members, nil
}

// Reconstruct the immutable predecessor assessment directly under the
// importer's own pending marker. This is not a pending-export override.
func checkRecoveryInstallCAS(root string, c *Catalog, m *RetranslationBatchManifest) error {
	if m.TransportRecovery == nil {
		return fmt.Errorf("recovery import authority missing")
	}
	a, _, err := readRecoveryExactAuthority(root, *m.TransportRecovery)
	if err != nil {
		return err
	}
	if err = recoveryRequireAbsent(root, recoveryRecordPending(a)); err != nil {
		return err
	}
	if err = recoveryRequireAbsent(root, recoveryRootDirectory(a.Locale)+"/supersessions/"+a.Identity+".json"); err != nil {
		return err
	}
	if err = recoveryRequireAbsent(root, transportRecoveryDirectory(a.Locale, a.Identity)+"/.export.pending"); err != nil {
		return err
	}
	fresh, _, err := prepareTransportRecovery(root, c, TransportRecoveryOptions{Locale: a.Locale, ContentPackage: a.ContentPackage.ID, PredecessorBatch: a.Predecessor.BatchID, Supersedes: a.Supersedes}, m.BatchID)
	if err != nil || fresh == nil || fresh.Identity != a.Identity {
		return fmt.Errorf("recovery import publication CAS changed: %v", err)
	}
	return nil
}

func derivedRecoverySelection(root string, c *Catalog, m *RetranslationBatchManifest) (*ApprovedIntervalSelection, error) {
	a, _, err := currentRecoveryInvocationAuthority(root, c, m)
	if err != nil {
		return nil, err
	}
	ab, err := readCompatibilityFile(root, a.Predecessor.Archive.Path)
	if err != nil {
		return nil, err
	}
	_, files, err := readGenerationBundle(ab)
	if err != nil {
		return nil, err
	}
	var prior ApprovedIntervalSelection
	if err = contentidentity.StrictJSON(files[approvedIntervalSelectionMember], &prior); err != nil {
		return nil, err
	}
	prior.BatchID = m.BatchID
	b := mustJSON(prior)
	return parseApprovedIntervalSelectionBytes(root, b, c, m)
}

func recoveryGenerationImportCatalog(root string, c *Catalog, bundle []byte) (*Catalog, error) {
	gm, files, err := readGenerationBundle(bundle)
	if err != nil {
		return nil, err
	}
	var m RetranslationBatchManifest
	if err = contentidentity.StrictJSON(files["formal/manifest.json"], &m); err != nil {
		return nil, err
	}
	if m.TransportRecovery == nil {
		return nil, fmt.Errorf("execution import requires recovery manifest")
	}
	a, _, err := readRecoveryExactAuthority(root, *m.TransportRecovery)
	if err != nil {
		return nil, err
	}
	if gm.BatchID != m.BatchID || gm.Locale != m.Locale {
		return nil, fmt.Errorf("recovery imported manifest identity mismatch")
	}
	if err = validateRecoveryChildManifest(a, &m); err != nil {
		return nil, err
	}
	return recoveryPackageCatalog(c, a.ContentPackage.ID)
}

func recoveryGenerationInputs(root string, c *Catalog, m *RetranslationBatchManifest, inputs []generationBundleInput) ([]generationBundleInput, error) {
	a, members, err := currentRecoveryInvocationAuthority(root, c, m)
	if err != nil {
		return nil, err
	}
	var recipe DerivedTransportConstruction
	if err = contentidentity.StrictJSON(members[a.Construction.Path], &recipe); err != nil {
		return nil, err
	}
	// The authority owns the complete original semantic context. Child batch
	// locators are independently frozen in the manifest and approved proof.
	files := map[string][]byte{}
	for _, in := range inputs {
		files[in.path] = in.data
	}
	base := "data/retranslation-runs/" + m.Locale + "/" + m.BatchID + "/"
	for _, r := range m.Units {
		b, err := readCompatibilityFile(root, base+r.InputPath)
		if err != nil {
			return nil, err
		}
		files["formal/"+r.InputPath] = b
		p, err := recoveryProofFromFiles(files, r.UnitID)
		if err != nil {
			return nil, err
		}
		if p != nil {
			for _, ref := range p.Dependencies {
				b, err := readCompatibilityReference(root, ref)
				if err != nil {
					return nil, err
				}
				files["dependencies/"+ref.SHA256] = b
			}
		}
	}
	constructed, _, err := deriveRecoveryConstruction(root, c, m, files, recipe.SemanticContext, false)
	if err != nil {
		return nil, err
	}
	// Complete typed construction must be stable across invocation locators.
	if !bytes.Equal(mustJSON(constructed), members[a.Construction.Path]) {
		return nil, fmt.Errorf("successor construction differs from recovery authority recipe")
	}
	ab, err := readCompatibilityFile(root, m.TransportRecovery.Path)
	if err != nil {
		return nil, err
	}
	inputs = append(inputs, generationBundleInput{"formal/transport-recovery/authority.json", m.TransportRecovery.Path, ab})
	for _, ref := range a.Members {
		name := "formal/transport-recovery/authority-members/" + ref.Path
		if ref.Path == a.Construction.Path {
			name = transportRecoveryConstructionMember
		}
		inputs = append(inputs, generationBundleInput{name, filepath.ToSlash(filepath.Dir(m.TransportRecovery.Path)) + "/" + ref.Path, members[ref.Path]})
	}
	supplementalInputs, err := historicalBytesGenerationInputs(root, c, a)
	if err != nil {
		return nil, err
	}
	inputs = append(inputs, supplementalInputs...)
	for _, ref := range a.CurrentEpoch {
		if ref.Role != "recovery-implementation" {
			continue
		}
		b, err := readCompatibilityFile(root, ref.Ref.Path)
		if err != nil || sum(b) != ref.Ref.SHA256 || int64(len(b)) != ref.Ref.Size {
			return nil, fmt.Errorf("recovery construction reader implementation drift")
		}
		inputs = append(inputs, generationBundleInput{"authority/transport-recovery/" + ref.Ref.Path, ref.Ref.Path, b})
	}
	// The completed historical candidate is context, not the new model output
	// or a replacement finding baseline.
	for _, u := range a.Predecessor.FinalUnits {
		b, err := readCompatibilityFile(root, u.Candidate.Path)
		if err != nil || sum(b) != u.Candidate.SHA256 {
			return nil, fmt.Errorf("historical completed revision context drift")
		}
		inputs = append(inputs, generationBundleInput{"formal/transport-recovery/historical-completed-candidates/" + u.UnitID + filepath.Ext(u.Candidate.Path), u.Candidate.Path, b})
	}
	return inputs, nil
}
