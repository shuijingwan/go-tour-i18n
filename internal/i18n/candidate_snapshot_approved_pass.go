package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// This is invocation-local Snapshot evidence, not a receipt, recovery action,
// or exemption for process/retry/revalidate. The archive remains immutable.
type snapshotApprovedPass struct {
	inputs   []generationBundleInput
	outputs  map[string]protectedTranslation
	observed map[string]string
}

// CurrentAuthority is deliberately not part of this historical semantic body:
// every final candidate is independently restored and run through the active
// validator below. All language/representation/provenance fields remain exact.
type snapshotApprovedSemanticBody struct {
	Schema                 string
	Mode                   string
	Selection              string
	Record                 RetranslationBatchUnit
	Reopen                 GlossaryArchiveReference
	Snapshot               QualityCheckSnapshotUnit
	SemanticAuthority      GlossaryArchiveReference
	ProvenanceSource       string
	LegacyReference        *GlossaryArchiveReference
	HistoricalVerification *VerifiedLegacyGenerationProvenance
	RequestedOptions       LegacyGenerationProvenanceOptions
	Projection             *DerivedApprovedConstruction
}

func snapshotApprovedSemantic(c *Catalog, u *TranslationUnit, p *ApprovedIntervalTransport) (snapshotApprovedSemanticBody, error) {
	x, _, err := typedApprovedConstruction(c, u, p)
	if err != nil {
		return snapshotApprovedSemanticBody{}, err
	}
	return snapshotApprovedSemanticBody{p.Schema, p.Mode, p.SelectionSHA256, p.Record, p.Reopen, p.Snapshot, p.SemanticAuthority, p.ProvenanceSource, p.LegacyReference, p.HistoricalVerification, p.RequestedOptions, x}, nil
}

func snapshotApprovedProofEquivalent(c *Catalog, u *TranslationUnit, old, current *ApprovedIntervalTransport) error {
	a, err := snapshotApprovedSemantic(c, u, old)
	if err != nil {
		return err
	}
	b, err := snapshotApprovedSemantic(c, u, current)
	if err != nil {
		return err
	}
	if !bytes.Equal(mustJSON(a), mustJSON(b)) {
		return fmt.Errorf("Snapshot historical approved semantic evidence changed: %s", u.ID)
	}
	if !validSHA256(old.CurrentAuthority) || !validSHA256(current.CurrentAuthority) || len(old.Dependencies) != len(current.Dependencies) {
		return fmt.Errorf("Snapshot historical approved dependency inventory changed")
	}
	implementation := map[string]bool{}
	for _, path := range approvedIntervalImplementationPaths {
		implementation[path] = true
	}
	seen := map[string]string{}
	for i, x := range old.Dependencies {
		y := current.Dependencies[i]
		if x.Path != y.Path || validateGenerationInstallPath(x.Path) != nil || !validSHA256(x.SHA256) || !validSHA256(y.SHA256) {
			return fmt.Errorf("Snapshot historical approved dependency identity mismatch")
		}
		if prior, ok := seen[x.Path]; ok && prior != x.SHA256 {
			return fmt.Errorf("Snapshot historical approved dependency conflict")
		}
		seen[x.Path] = x.SHA256
		if x.SHA256 != y.SHA256 && !implementation[x.Path] {
			return fmt.Errorf("Snapshot historical approved semantic dependency changed: %s", x.Path)
		}
	}
	return nil
}

func snapshotCanonicalHistoricalJSON(value any, data []byte) error {
	expected, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if !bytes.Equal(append(expected, '\n'), data) {
		return fmt.Errorf("Snapshot historical result/validation bytes do not match the canonical passed evidence")
	}
	return nil
}

func prepareSnapshotApprovedPass(root string, c *Catalog, g *Glossary, m *RetranslationBatchManifest, selected map[string]selectedRetranslationUnit) (*snapshotApprovedPass, error) {
	if m.TransportRecovery != nil {
		return nil, nil
	} // Existing recovery authority/currentness remains strict.
	base := "data/retranslation-runs/" + m.Locale + "/" + m.BatchID + "/"
	if err := ensureNoGenerationImportPending(filepath.Join(root, base)); err != nil {
		return nil, err
	}
	selection, files, err := importedApprovedIntervalSelection(root, c, m)
	if err != nil || selection == nil {
		return nil, err
	}
	if err := checkImportedSurfaceOutputHashes(root, m); err != nil {
		return nil, err
	}
	if err := checkSurfaceTransportBatch(root, m.Locale, m.BatchID, m.Units, false); err != nil {
		return nil, err
	}
	if err := RequireNewLanguageGlossaryReview(root, m.Locale); err != nil {
		return nil, err
	}
	// Preserve the complete ordinary fast path, including non-approved sibling
	// decoder variants. Only Snapshot owns the independent reconstruction below
	// when strict reproduction cannot establish an exact-current invocation.
	outputs := map[string]protectedTranslation{}
	inputs, strictErr := revisionGenerationInputsCore(root, c, m.Locale, m.Units, []map[string]protectedTranslation{outputs}, false, selection)
	if strictErr == nil {
		inputs, strictErr = applySurfaceTransportInWorld(root, c, m.Locale, m.Units, inputs, true)
		if strictErr == nil && compareInstalledApprovedProofs(inputs, files) == nil {
			return nil, nil
		}
	}
	observed := map[string]string{}
	if strictErr != nil {
		outputs = map[string]protectedTranslation{}
		predecessorOutput := func(dir string, catalog *Catalog, glossary *Glossary, manifest *RetranslationBatchManifest) (map[string]protectedTranslation, func() error, error) {
			return snapshotFrozenPredecessorOutputs(root, dir, catalog, glossary, manifest, observed)
		}
		inputs, err = revisionGenerationInputsAtPredecessor(root, c, m.Locale, m.Units, []map[string]protectedTranslation{outputs}, false, predecessorOutput, selection)
		if err != nil {
			return nil, err
		}
		inputs, err = applySurfaceTransportInWorld(root, c, m.Locale, m.Units, inputs, true)
		if err != nil {
			return nil, err
		}
	}
	state := &snapshotApprovedPass{inputs: append([]generationBundleInput{}, inputs...), outputs: outputs, observed: observed}
	observe := func(path string) ([]byte, error) {
		b, err := readSnapshotReferencedFile(root, path)
		if err == nil {
			state.observed[path] = sum(b)
		}
		return b, err
	}
	review, err := RequireUnifiedGlossaryReview(root, m.Locale)
	if err != nil {
		return nil, err
	}
	paths := append(append([]string{}, translationUnitGenerationAuthorityPaths...), currentValidationReopenDependencies...)
	paths = append(paths, review.Path, GlossaryCorpusPath, glossaryReviewGlossaryPath(m.Locale))
	for _, path := range paths {
		if _, err := observe(path); err != nil {
			return nil, err
		}
	}
	rb, err := observe(base + "result.json")
	if err != nil {
		return nil, err
	}
	result, err := readPromotionResult(filepath.Join(root, base), m.Locale, m.BatchID, len(m.Units))
	if err != nil {
		return nil, err
	}
	n := len(m.Units)
	if n == 0 || n > MaxRetranslationExportLimit || result.NoPendingBatches || result.RestorePassed != n || result.RestoreFailed != 0 || result.ValidationPassed != n || result.ValidationFailed != 0 {
		return nil, fmt.Errorf("Snapshot historical approved batch is not complete full PASS")
	}
	if err := snapshotCanonicalHistoricalJSON(result, rb); err != nil {
		return nil, err
	}
	// Admission verifies the planner's exact choices; it must never select a
	// second candidate set or impose global no-fallback rules on other callers.
	// The normal selector deliberately ignores export-only batches. Historical
	// admission additionally requires no overlapping successor, including one
	// which has not yet produced a result.
	match := promotionBatchRE(m.Locale).FindStringSubmatch(m.BatchID)
	if match == nil {
		return nil, fmt.Errorf("Snapshot historical batch has no formal ordinal")
	}
	ordinal, _ := strconv.Atoi(match[2])
	batchBase := filepath.Join(root, "data/retranslation-runs", m.Locale)
	entries, err := os.ReadDir(batchBase)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		next := promotionBatchRE(m.Locale).FindStringSubmatch(entry.Name())
		if next == nil {
			continue // Only formal successor invocations can conflict here.
		}
		number, _ := strconv.Atoi(next[2])
		if number <= ordinal {
			continue
		}
		successor, err := readRetranslationProcessManifest(filepath.Join(batchBase, entry.Name()), m.Locale, entry.Name())
		if err != nil {
			return nil, err
		}
		for _, newer := range successor.Units {
			for _, old := range m.Units {
				if newer.UnitID == old.UnitID {
					return nil, fmt.Errorf("Snapshot historical PASS has a conflicting successor: %s", successor.BatchID)
				}
			}
		}
	}
	ib, err := observe(base + "generation-imports/initial-manifest.json")
	if err != nil {
		return nil, err
	}
	var receipt GenerationResultBundleManifest
	if err := contentidentity.StrictJSON(ib, &receipt); err != nil {
		return nil, err
	}
	rootRef := recoveryRootDirectory(m.Locale) + "/roots/" + receipt.GenerationBundleSHA256
	for _, suffix := range []string{".json", ".record.pending"} {
		if err := recoveryRequireAbsent(root, rootRef+suffix); err != nil {
			return nil, fmt.Errorf("Snapshot historical PASS conflicts with recovery lineage: %w", err)
		}
	}
	if _, err := observe(importedGenerationArchive(m.Locale, receipt.GenerationBundleSHA256)); err != nil {
		return nil, err
	}
	var initial GenerationBundleManifest
	if err := contentidentity.StrictJSON(files["manifest.json"], &initial); err != nil {
		return nil, err
	}
	if initial.TaskKind != "translation-unit-revision" || initial.Attempt != 1 {
		return nil, fmt.Errorf("Snapshot historical approved initial invocation kind/attempt mismatch")
	}
	expected := map[string]bool{}
	for i, record := range m.Units {
		choice, ok := selected[record.UnitID]
		if !ok || choice.batchID != m.BatchID || !reflect.DeepEqual(choice.manifest, record) || result.Units[i].UnitID != record.UnitID || result.Units[i].UnitKind != record.UnitKind || result.Units[i].Status != "passed" || result.Units[i].Error != "" || !reflect.DeepEqual(result.Units[i], choice.result) {
			return nil, fmt.Errorf("Snapshot historical approved PASS scope/selection conflicts with latest result")
		}
		u, err := c.Unit(record.UnitID)
		if err != nil {
			return nil, err
		}
		old, err := recoveryProofFromFiles(files, record.UnitID)
		if err != nil {
			return nil, err
		}
		var adapter TranslationUnitAdapter
		if selection.entry(record.UnitID) != nil {
			if old == nil {
				return nil, fmt.Errorf("Snapshot historical approved proof missing: %s", record.UnitID)
			}
			path := approvedIntervalProofMember(record.UnitID)
			expected[path] = true
			found := false
			for j, input := range inputs {
				if input.path != path {
					continue
				}
				found = true
				current, err := recoveryProofFromFiles(map[string][]byte{path: input.data}, record.UnitID)
				if err != nil {
					return nil, err
				}
				if err := snapshotApprovedProofEquivalent(c, u, old, current); err != nil {
					return nil, err
				}
				for _, ref := range current.Dependencies {
					b, err := observe(ref.Path)
					if err != nil || sum(b) != ref.SHA256 {
						return nil, fmt.Errorf("Snapshot approved current dependency drift: %s", ref.Path)
					}
				}
				// Only the verified archive-comparison input refers to the old
				// proof. The decoder/output remains freshly constructed above.
				state.inputs[j].data = files[path]
			}
			if !found {
				return nil, fmt.Errorf("Snapshot current approved proof missing")
			}
			_, adapter, err = typedApprovedConstruction(c, u, old)
			if err != nil {
				return nil, err
			}
		} else if old != nil {
			return nil, fmt.Errorf("Snapshot extra historical approved proof")
		}
		final, err := recoveryHistoricalFinalUnit(root, c, g, m, choice, initial, files, adapter, outputs[record.UnitID], receipt.Provider, receipt.Model)
		if err != nil {
			return nil, err
		}
		for _, ref := range append(append([]RecoveryDependency{}, final.History...), RecoveryDependency{Ref: final.Raw}, RecoveryDependency{Ref: final.Candidate}, RecoveryDependency{Ref: final.Validation}) {
			b, err := observe(ref.Ref.Path)
			if err != nil || sum(b) != ref.Ref.SHA256 || int64(len(b)) != ref.Ref.Size {
				return nil, fmt.Errorf("Snapshot historical PASS evidence drift")
			}
		}
		vb, err := observe(final.Validation.Path)
		if err != nil {
			return nil, err
		}
		v, err := readPromotionValidation(choice.batchDir, m.BatchID, m.Locale, record, choice.result)
		if err != nil {
			return nil, err
		}
		if v.Error != "" {
			return nil, fmt.Errorf("Snapshot historical passed validation contains an error")
		}
		if err := snapshotCanonicalHistoricalJSON(v, vb); err != nil {
			return nil, err
		}
	}
	for path := range files {
		if strings.HasPrefix(path, "formal/approved-interval-transport/") && !expected[path] {
			return nil, fmt.Errorf("Snapshot extra historical approved proof: %s", path)
		}
	}
	for _, in := range state.inputs {
		if !bytes.Equal(in.data, files[in.path]) {
			return nil, fmt.Errorf("Snapshot historical semantic/finding input changed: %s", in.path)
		}
		if in.repositoryPath != "" {
			if _, err := observe(in.repositoryPath); err != nil {
				return nil, err
			}
		}
	}
	if _, err := observe(base + "manifest.json"); err != nil {
		return nil, err
	}
	return state, nil
}

// A nested ancestor supplies a frozen response decoder, not a current candidate
// selection or an admission decision. Its exact imported archive is the sole
// locator. The surrounding revision builder checks raw -> Snapshot candidate;
// the selected batch still reconstructs its own current proof and passes the
// full historical-PASS/equivalence/current-validator checks above. No recursion
// through process or Snapshot selection is used for an approved ancestor.
func snapshotFrozenPredecessorOutputs(root, dir string, c *Catalog, g *Glossary, m *RetranslationBatchManifest, observed map[string]string) (map[string]protectedTranslation, func() error, error) {
	if m.TransportRecovery != nil {
		return prepareProcessOutputContext(dir, c, g, m)
	}
	selection, files, err := importedApprovedIntervalSelection(root, c, m)
	if err != nil || selection == nil {
		if err != nil {
			return nil, nil, err
		}
		return prepareProcessOutputContext(dir, c, g, m)
	}
	base, err := repositoryRelativePath(root, dir)
	if err != nil {
		return nil, nil, err
	}
	refs := map[string]string{}
	bind := func(path string) ([]byte, error) {
		b, err := readSnapshotReferencedFile(root, path)
		if err != nil {
			return nil, err
		}
		sha := sum(b)
		if old, ok := observed[path]; ok && old != sha {
			return nil, fmt.Errorf("Snapshot frozen predecessor changed: %s", path)
		}
		observed[path], refs[path] = sha, sha
		return b, nil
	}
	ib, err := bind(base + "/generation-imports/initial-manifest.json")
	if err != nil {
		return nil, nil, err
	}
	var receipt GenerationResultBundleManifest
	if err := contentidentity.StrictJSON(ib, &receipt); err != nil {
		return nil, nil, err
	}
	if _, err := bind(importedGenerationArchive(m.Locale, receipt.GenerationBundleSHA256)); err != nil {
		return nil, nil, err
	}
	if _, err := bind(base + "/manifest.json"); err != nil {
		return nil, nil, err
	}
	implementation := map[string]bool{}
	for _, path := range approvedIntervalImplementationPaths {
		implementation[path] = true
	}
	outputs := map[string]protectedTranslation{}
	frozenInputs := []generationBundleInput{{path: approvedIntervalSelectionMember, data: selection.bytes}, {path: "formal/manifest.json", data: files["formal/manifest.json"]}}
	expectedProofs := map[string]bool{}
	for _, r := range m.Units {
		u, err := c.Unit(r.UnitID)
		if err != nil {
			return nil, nil, err
		}
		p, err := prepareTranslationUnitInput(u, g)
		if err != nil {
			return nil, nil, err
		}
		if r.SourcePath != u.SourcePath || r.SourceSHA256 != u.SourceSHA256 || sum(canonicalizeRetranslationArtifactEOF([]byte(p.Text))) != r.InputSHA256 || !bytes.Equal(files["formal/"+r.InputPath], canonicalizeRetranslationArtifactEOF([]byte(p.Text))) {
			return nil, nil, fmt.Errorf("Snapshot frozen predecessor source/input changed: %s", r.UnitID)
		}
		proof, err := recoveryProofFromFiles(files, r.UnitID)
		if err != nil {
			return nil, nil, err
		}
		if selection.entry(r.UnitID) != nil {
			if proof == nil || proof.SelectionSHA256 != sum(selection.bytes) || !reflect.DeepEqual(proof.Record, r) {
				return nil, nil, fmt.Errorf("Snapshot frozen predecessor proof/selection mismatch")
			}
			_, decoder, err := typedApprovedConstruction(c, u, proof)
			if err != nil || decoder == nil {
				return nil, nil, fmt.Errorf("Snapshot frozen predecessor inverse unavailable: %v", err)
			}
			p.Adapter = decoder
			expectedProofs[approvedIntervalProofMember(r.UnitID)] = true
			frozenInputs = append(frozenInputs, generationBundleInput{path: approvedIntervalProofMember(r.UnitID), data: files[approvedIntervalProofMember(r.UnitID)]})
			for _, ref := range proof.Dependencies {
				if implementation[ref.Path] {
					continue
				}
				b, err := bind(ref.Path)
				if err != nil || sum(b) != ref.SHA256 {
					return nil, nil, fmt.Errorf("Snapshot frozen predecessor dependency changed: %s", ref.Path)
				}
			}
		} else if proof != nil || r.RepresentationBridge != nil || len(files[commentTransportPath(r)]) != 0 || len(files["formal/current-validation-predecessor-transport/"+filepath.Base(r.InputPath)+".json"]) != 0 {
			return nil, nil, fmt.Errorf("Snapshot frozen predecessor requires its strict non-approved decoder: %s", r.UnitID)
		}
		outputs[r.UnitID] = p
	}
	for path := range files {
		if strings.HasPrefix(path, "formal/approved-interval-transport/") && !expectedProofs[path] {
			return nil, nil, fmt.Errorf("Snapshot frozen predecessor has an extra approved proof")
		}
	}
	// Bind the exact original import and all final raw provenance, without
	// reconstructing its historical implementation or granting it eligibility.
	result, err := readPromotionResult(dir, m.Locale, m.BatchID, len(m.Units))
	if err != nil {
		return nil, nil, err
	}
	rb, err := bind(base + "/result.json")
	if err != nil {
		return nil, nil, err
	}
	if err := snapshotCanonicalHistoricalJSON(result, rb); err != nil {
		return nil, nil, err
	}
	if result.RestorePassed != len(m.Units) || result.RestoreFailed != 0 || result.ValidationPassed != len(m.Units) || result.ValidationFailed != 0 {
		return nil, nil, fmt.Errorf("Snapshot frozen predecessor is not complete historical PASS")
	}
	for i, r := range m.Units {
		if i >= len(result.Units) || result.Units[i].UnitID != r.UnitID || result.Units[i].Status != "passed" {
			return nil, nil, fmt.Errorf("Snapshot frozen predecessor final result mismatch")
		}
		v, err := readPromotionValidation(dir, m.BatchID, m.Locale, r, result.Units[i])
		if err != nil {
			return nil, nil, err
		}
		choice := selectedRetranslationUnit{batchID: m.BatchID, batchDir: dir, manifest: r, result: result.Units[i]}
		candidate := &validatedRetranslationCandidate{validation: v, attempt: v.Attempt}
		history, err := verifyImportedRevisionInvocationInputs(root, c, m.Locale, choice, candidate, m, frozenInputs)
		if err != nil {
			return nil, nil, err
		}
		for _, ref := range history {
			if b, err := bind(ref.Path); err != nil || sum(b) != ref.SHA256 {
				return nil, nil, fmt.Errorf("Snapshot frozen predecessor import changed")
			}
		}
		for _, path := range []string{result.Units[i].CandidatePath, result.Units[i].ValidationPath, v.RawResponsePath} {
			if _, err := bind(base + "/" + path); err != nil {
				return nil, nil, err
			}
		}
	}
	return outputs, func() error {
		for path, sha := range refs {
			b, err := readSnapshotReferencedFile(root, path)
			if err != nil || sum(b) != sha {
				return fmt.Errorf("Snapshot frozen predecessor changed: %s", path)
			}
		}
		return nil
	}, nil
}

func prepareSnapshotOutputContext(batchDir string, c *Catalog, g *Glossary, m *RetranslationBatchManifest, selected map[string]selectedRetranslationUnit) (map[string]protectedTranslation, func() error, error) {
	root := filepath.Clean(filepath.Join(batchDir, "../../../.."))
	state, err := prepareSnapshotApprovedPass(root, c, g, m, selected)
	if err != nil {
		return nil, nil, err
	}
	if state == nil {
		return prepareProcessOutputContext(batchDir, c, g, m)
	}
	return state.outputs, func() error {
		fresh, err := prepareSnapshotApprovedPass(root, c, g, m, selected)
		if err != nil || fresh == nil || !reflect.DeepEqual(state.inputs, fresh.inputs) || !reflect.DeepEqual(state.observed, fresh.observed) {
			return fmt.Errorf("Snapshot historical PASS admission changed before commit: %v", err)
		}
		return nil
	}, nil
}

func verifySnapshotRevisionInvocation(root string, c *Catalog, locale string, choice selectedRetranslationUnit, v *validatedRetranslationCandidate, selected map[string]selectedRetranslationUnit) ([]GlossaryArchiveReference, error) {
	m, err := readRetranslationProcessManifest(choice.batchDir, locale, choice.batchID)
	if err != nil {
		return nil, err
	}
	g, err := LoadGlossary(root, locale)
	if err != nil {
		return nil, err
	}
	state, err := prepareSnapshotApprovedPass(root, c, g, m, selected)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return verifyImportedRevisionInvocation(root, c, locale, choice, v)
	}
	return verifyImportedRevisionInvocationInputs(root, c, locale, choice, v, m, state.inputs)
}
