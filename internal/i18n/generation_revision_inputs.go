package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type generationBundleInput struct {
	path, repositoryPath string
	data                 []byte
}

// Revision feedback applies to the reviewed Snapshot bytes, never a latest
// candidate selected from another batch. Reuse the shared QC evidence boundary.
func revisionGenerationInputs(root string, catalog *Catalog, locale string, selected []RetranslationBatchUnit, transports ...map[string]protectedTranslation) ([]generationBundleInput, error) {
	return revisionGenerationInputsCore(root, catalog, locale, selected, transports, true)
}

func revisionGenerationInputsCore(root string, catalog *Catalog, locale string, selected []RetranslationBatchUnit, transports []map[string]protectedTranslation, surfaceCapability bool, selections ...*ApprovedIntervalSelection) ([]generationBundleInput, error) {
	return revisionGenerationInputsAtPredecessor(root, catalog, locale, selected, transports, surfaceCapability, prepareProcessOutputContext, selections...)
}

// The ordinary entry above always supplies the strict process reader. Snapshot
// owns a separate explicit frozen-decoder reader; this is no caller flag or
// relaxation of process/retry/export currentness. Both paths build current
// projections below and verify the predecessor raw/candidate equality.
func revisionGenerationInputsAtPredecessor(root string, catalog *Catalog, locale string, selected []RetranslationBatchUnit, transports []map[string]protectedTranslation, surfaceCapability bool, predecessorOutput func(string, *Catalog, *Glossary, *RetranslationBatchManifest) (map[string]protectedTranslation, func() error, error), selections ...*ApprovedIntervalSelection) ([]generationBundleInput, error) {
	var inputs []generationBundleInput
	var approvedSelection *ApprovedIntervalSelection
	if len(selections) > 0 {
		approvedSelection = selections[0]
	}
	if approvedSelection != nil {
		inputs = append(inputs, generationBundleInput{approvedIntervalSelectionMember, "", approvedSelection.bytes})
	}
	evaluation := newQualityCheckEvaluation(root, locale, catalog)
	type predecessorScope struct {
		catalog  *Catalog
		snapshot string
	}
	scopedEvaluations := map[predecessorScope]*qualityCheckEvaluation{}
	// Only current-validation receipts use an immutable, complete frozen A graph.
	// Reuse that verified graph within this call, then recheck its exact disk
	// references before returning. Ordinary QC/Surface/Preview paths are unchanged.
	type currentScope struct {
		catalog    *Catalog
		receipt    *QualityCheckSurfaceReopen
		evaluation *qualityCheckEvaluation
	}
	currentScopes := map[string]currentScope{}
	bridgeScopes := map[GlossaryArchiveReference]*QualityCheckSurfaceReopen{}
	seenSnapshots := map[string]bool{}
	seenSurfaceReferences := map[string]bool{}
	glossary, err := LoadGlossary(root, locale)
	if err != nil {
		return nil, err
	}
	// A QC successor can review several candidates from the same earlier
	// current-validation batch. Reuse the existing process projection boundary,
	// including its final fresh authority reconstruction, once per batch/scope.
	type outputScope struct {
		catalog *Catalog
		batch   string
	}
	predecessorOutputs := map[outputScope]map[string]protectedTranslation{}
	var outputChecks []func() error
	var historicalChecks []*bridgeResponseLineage
	feedbackRefs := map[string][]byte{}
	for _, record := range selected {
		if record.RevisionFeedbackSource != "quality_check" {
			continue
		}
		path := "data/quality-check-snapshots/" + locale + "/" + record.PreviousSnapshotID + "/manifest.json"
		if _, ok := feedbackRefs[path]; ok {
			continue
		}
		refs, err := qualityCheckFinalizationInputs(root, locale, record.PreviousSnapshotID, map[string]bool{})
		if err != nil {
			return nil, err
		}
		for _, ref := range append(refs, QualityCheckFinalizationInput{Path: path}) {
			b, err := readSnapshotReferencedFile(root, ref.Path)
			if err != nil || (ref.SHA256 != "" && sum(b) != ref.SHA256) {
				return nil, fmt.Errorf("QC revision feedback changed: %s", ref.Path)
			}
			feedbackRefs[ref.Path] = b
		}
	}
	for _, record := range selected {
		predecessorID := record.UnitID
		predecessorCatalog := catalog
		var successor *QualityCheckSurfaceReopen
		cached, reused := currentScopes[record.RevisionAuthorizationID]
		var unitEvaluation *qualityCheckEvaluation
		if record.RevisionFeedbackSource == CurrentValidationRevisionFeedback && reused {
			predecessorCatalog, successor, unitEvaluation = cached.catalog, cached.receipt, cached.evaluation
			predecessorID = successor.predecessorUnitID(record.UnitID)
		} else {
			if isReopenFeedback(record.RevisionFeedbackSource) {
				var err error
				predecessorCatalog, successor, err = surfaceRevisionCatalog(root, catalog, locale, record.RevisionAuthorizationID)
				if err != nil {
					return nil, err
				}
				predecessorID = successor.predecessorUnitID(record.UnitID)
			}
			unitEvaluation = evaluation
			if successor != nil && successor.originalQCReplacement() {
				unitEvaluation = newQualityCheckEvaluation(root, locale, predecessorCatalog)
			}
			if successor != nil && !successor.originalQCReplacement() {
				// Frozen reopen proof was checked above. Resolve its complete package
				// from that exact finalization before reading effective A: an unscoped
				// CLI inventory otherwise looks in Tour's authorization namespace.
				// Reuse the existing evaluation boundary for units of the same scope;
				// never mutate the caller or infer away an explicitly different scope.
				key := predecessorScope{predecessorCatalog, record.PreviousSnapshotID}
				if cached := scopedEvaluations[key]; cached != nil {
					unitEvaluation = cached
				} else {
					scoped, registered, err := SelectFinalizedTranslationPackage(root, predecessorCatalog, locale, record.PreviousSnapshotID)
					if err != nil {
						return nil, err
					}
					if registered {
						predecessorCatalog = scoped
					}
					if predecessorCatalog != catalog {
						unitEvaluation = newQualityCheckEvaluation(root, locale, predecessorCatalog)
					}
					scopedEvaluations[key] = unitEvaluation
				}
			}
			if successor != nil && successor.AuthoritySource == CurrentValidationAuthority {
				unitEvaluation = currentValidationPredecessorEvaluation(root, predecessorCatalog, locale)
				// surfaceRevisionCatalog has just verified this receipt's complete
				// frozen A graph. Consume that exact graph rather than rediscovering
				// the same historical finalization for every transport reconstruction.
				// currentScopes below still rechecks all refs before returning.
				assessment, err := decodeCurrentValidationAssessment(successor.CurrentValidation)
				if err != nil {
					return nil, err
				}
				if assessment.Historical.Finalization.SnapshotID != record.PreviousSnapshotID {
					return nil, fmt.Errorf("current-validation original predecessor mismatch")
				}
				if err := unitEvaluation.materializeOriginalFinalization(&assessment.Historical); err != nil {
					return nil, err
				}
			}
			if record.RevisionFeedbackSource == CurrentValidationRevisionFeedback {
				if successor == nil || successor.AuthoritySource != CurrentValidationAuthority {
					return nil, fmt.Errorf("current-validation revision authority mismatch")
				}
				currentScopes[record.RevisionAuthorizationID] = currentScope{predecessorCatalog, successor, unitEvaluation}
			}
		}
		var validatedBridge *QualityCheckSurfaceReopen
		var predecessor *QualityCheckSnapshotUnit
		var result effectiveQualityCheckResult
		var ok bool
		if record.RepresentationBridge != nil {
			// The canonical bridge builder already validated the complete QC
			// lineage and these exact predecessor bytes in this invocation.
			// Consume that same graph, not a second independent historical scan.
			validatedBridge = bridgeScopes[*record.RepresentationBridge]
			if validatedBridge == nil {
				validatedBridge, _, err = readSameUnitBridgeForInvocation(root, catalog, locale, *record.RepresentationBridge)
				if err != nil {
					return nil, err
				}
				bridgeScopes[*record.RepresentationBridge] = validatedBridge
			}
			if validatedBridge.UnitCount != len(selected) || validatedBridge.PreviousSnapshotID != record.PreviousSnapshotID || record.RevisionFeedbackSource != "quality_check" {
				return nil, fmt.Errorf("bridge QC scope mismatch")
			}
			for _, q := range validatedBridge.Units {
				if q.UnitID == predecessorID {
					copy := q.Snapshot
					predecessor = &copy
					result = effectiveQualityCheckResult{rating: q.PreviousRating, finding: q.Finding, rubric: TranslationQualityRubric, snapshotID: record.PreviousSnapshotID, unit: q.Snapshot}
					ok = true
				}
			}
		} else {
			snapshot, results, err := unitEvaluation.loadEffectiveQualityCheckResults(record.PreviousSnapshotID, map[string]bool{})
			if err != nil {
				return nil, err
			}
			for i := range snapshot.Units {
				if snapshot.Units[i].UnitID == predecessorID {
					predecessor = &snapshot.Units[i]
				}
			}
			result, ok = results[predecessorID]
		}
		if predecessor == nil || predecessor.UnitKind != record.UnitKind || predecessor.SourcePath != record.SourcePath || predecessor.SourceSHA256 != record.SourceSHA256 {
			return nil, fmt.Errorf("%s: revision predecessor source identity mismatch", record.UnitID)
		}
		if !ok || result.rubric != TranslationQualityRubric || !qualityCheckSnapshotIdentityMatches(*predecessor, result.unit) || result.rating != record.PreviousRating {
			return nil, fmt.Errorf("%s: revision finding candidate identity mismatch", record.UnitID)
		}
		switch record.RevisionFeedbackSource {
		case "quality_check":
			if result.rating == "A" || result.finding == "" || result.finding != record.PreviousFinding {
				return nil, fmt.Errorf("%s: revision finding authorization mismatch", record.UnitID)
			}
		case "surface_review", PreviewRevisionFeedback, QCRepresentationFeedback, RepresentationRevisionFeedback, CurrentValidationRevisionFeedback:
			reopen := successor
			ref := reopen.revisionEvidence()
			matches := false
			for _, unit := range reopen.Units {
				if unit.UnitID == record.UnitID && qualityCheckSnapshotIdentityMatches(unit.Snapshot, *predecessor) {
					matches = true
				}
			}
			ratingMatches := result.rating == "A"
			if reopen.AuthoritySource == "quality-check" {
				ratingMatches = (result.rating == "B" || result.rating == "C" || result.rating == "D") && result.finding == record.PreviousFinding
			}
			if !matches || record.RevisionFeedbackSource != reopen.revisionFeedbackSource() || !ratingMatches || reopen.PreviousSnapshotID != record.PreviousSnapshotID || reopen.findingForUnit(record.UnitID) != record.PreviousFinding || ref.Path != record.PreviousReviewPath || ref.SHA256 != record.PreviousReviewSHA256 || reopen.UnitCount != len(selected) {
				return nil, fmt.Errorf("%s: Surface revision authorization mismatch", record.UnitID)
			}
			refs := []GlossaryArchiveReference{{Path: filepath.ToSlash(filepath.Join("data", "quality-check-reopens", locale, reopen.ReopenID+".json"))}}
			if reopen.PreviewFailure != nil {
				refs = append(refs, *reopen.PreviewFailure)
				path := "docs/PREVIEW_TU_FAILURE.md"
				if !seenSurfaceReferences[path] {
					data, err := readSnapshotReferencedFile(root, path)
					if err != nil {
						return nil, err
					}
					inputs = append(inputs, generationBundleInput{"authority/" + path, path, data})
					seenSurfaceReferences[path] = true
				}
			}
			if reopen.ReplacementContext != nil {
				refs = append(refs, reopen.ReplacementContext.HistoricalDocuments)
				if a := reopen.ReplacementContext.QualityCheck; a != nil {
					refs = append(refs, a.Snapshot, a.Results, a.EngineeringProof)
				} else if a := reopen.ReplacementContext.Machine; a != nil {
					refs = append(refs, a.Options.Impact, a.OriginalRaw, a.OriginalManifest, a.OriginalResult)
				} else {
					refs = append(refs, reopen.ReplacementContext.PredecessorReopen)
				}
				path := "docs/TRANSLATION_UNIT_REPLACEMENT.md"
				if !seenSurfaceReferences[path] {
					data, err := readSnapshotReferencedFile(root, path)
					if err != nil {
						return nil, err
					}
					inputs = append(inputs, generationBundleInput{"authority/" + path, path, data})
					seenSurfaceReferences[path] = true
				}
			}
			for _, ref := range refs {
				data, err := readSnapshotReferencedFile(root, ref.Path)
				if err != nil || (ref.SHA256 != "" && sum(data) != ref.SHA256) {
					return nil, fmt.Errorf("replacement/reopen input stale")
				}
				hash := sum(data)
				if !seenSurfaceReferences[hash] {
					inputs = append(inputs, generationBundleInput{"formal/surface-authority/" + hash + ".txt", ref.Path, data})
					seenSurfaceReferences[hash] = true
				}
			}
			if reopen.IntegratedAuthority != nil {
				for _, ref := range surfaceAuthorityReferences(reopen.IntegratedAuthority) {
					if seenSurfaceReferences[ref.SHA256] {
						continue
					}
					data, err := readSnapshotReferencedFile(root, ref.Path)
					if err != nil || sum(data) != ref.SHA256 {
						return nil, fmt.Errorf("Surface authority input stale: %s", ref.Path)
					}
					inputs = append(inputs, generationBundleInput{"formal/surface-authority/" + ref.SHA256 + ".txt", ref.Path, data})
					seenSurfaceReferences[ref.SHA256] = true
				}
			}
		default:
			return nil, fmt.Errorf("%s: missing revision finding authorization", record.UnitID)
		}
		evidence, err := unitEvaluation.readSnapshotUnitRepositoryEvidence(record.PreviousSnapshotID, *predecessor)
		if err != nil {
			return nil, fmt.Errorf("%s: revision predecessor candidate: %w", record.UnitID, err)
		}
		inputs = append(inputs, generationBundleInput{"formal/candidates/" + filepath.Base(record.InputPath), predecessor.CandidatePath, evidence.candidate})
		// The passed validation identifies the exact raw-response attempt that
		// produced the reviewed candidate. Never reverse-protect restored prose.
		name := filepath.Base(evidence.record.InputPath)
		attempt, err := retryValidationAttemptForExtension(evidence.validation.RawResponsePath, strings.TrimSuffix(name, filepath.Ext(name)), filepath.Ext(name))
		if err != nil || attempt != predecessor.Attempt || evidence.validation.Status != "passed" {
			return nil, fmt.Errorf("%s: predecessor raw response attempt mismatch", record.UnitID)
		}
		rawPath := filepath.ToSlash(filepath.Join("data", "retranslation-runs", locale, predecessor.SelectedBatchID, evidence.validation.RawResponsePath))
		if evidence.machine != nil {
			rawPath = evidence.machine.unit.CurrentRaw.Path
			for _, ref := range machineAuthorityReferences(evidence.machine) {
				if seenSurfaceReferences[ref.SHA256] {
					continue
				}
				data, err := readSnapshotReferencedFile(root, ref.Path)
				if err != nil || sum(data) != ref.SHA256 {
					return nil, fmt.Errorf("machine reconciliation revision authority drift: %s", ref.Path)
				}
				inputs = append(inputs, generationBundleInput{"formal/representation-authority/" + ref.SHA256 + filepath.Ext(ref.Path), ref.Path, data})
				seenSurfaceReferences[ref.SHA256] = true
			}
		}
		raw, err := readSnapshotReferencedFile(root, rawPath)
		if err != nil {
			return nil, fmt.Errorf("%s: predecessor raw response: %w", record.UnitID, err)
		}
		protected, err := prepareTranslationUnitInput(evidence.unit, glossary)
		if err != nil {
			return nil, err
		}
		restored, failures := protected.restore(string(raw))
		candidate := []byte(restored)
		batch, err := readRetranslationProcessManifest(evidence.batchDir, locale, predecessor.SelectedBatchID)
		if err != nil {
			return nil, err
		}
		if batch.ArtifactEOF == retranslationArtifactEOFSingleLF {
			if err := validateRetranslationArtifactEOF(raw); err != nil {
				return nil, fmt.Errorf("%s: predecessor raw response EOF: %w", record.UnitID, err)
			}
			candidate = canonicalizeRetranslationArtifactEOF(candidate)
		}
		// Ordinary QC needs only this selected predecessor's editing baseline.
		// An exact ordinary inverse already proves it; discovering a historical
		// projection would unnecessarily recheck unselected batch siblings. Other
		// feedback and genuinely restricted outputs keep the strict reader below.
		ordinaryQCExactRestore := record.RevisionFeedbackSource == "quality_check" && len(failures) == 0 && bytes.Equal(candidate, evidence.candidate)
		if validatedBridge != nil {
			// Reuse the exact old-raw restore established by the full canonical
			// bridge reconstruction above, only after fresh raw/candidate/evidence
			// reads match that immutable binding. This is not prior-PASS carry.
			bound := false
			for _, b := range validatedBridge.SameUnitBridge.Bindings {
				if b.UnitID == record.UnitID && b.Raw.Path == rawPath && b.Raw.SHA256 == sum(raw) && b.FormalInputSHA256 == evidence.record.InputSHA256 && predecessor.Attempt == evidence.validation.Attempt {
					bound = true
				}
			}
			if !bound || sum(evidence.candidate) != predecessor.CandidateSHA256 {
				return nil, fmt.Errorf("bridge predecessor restore binding drift")
			}
			candidate, failures = evidence.candidate, nil
		} else if evidence.record.RevisionFeedbackSource == CurrentValidationRevisionFeedback && !ordinaryQCExactRestore {
			key := outputScope{predecessorCatalog, evidence.batchDir}
			projections, ok := predecessorOutputs[key]
			if !ok {
				var check func() error
				projections, check, err = predecessorOutput(evidence.batchDir, predecessorCatalog, glossary, batch)
				if err != nil {
					return nil, err
				}
				predecessorOutputs[key] = projections
				if check != nil {
					outputChecks = append(outputChecks, check)
				}
			}
			if p := projections[record.UnitID]; p.Adapter != nil {
				protected = p
			}
			restored, failures = protected.restore(string(raw))
			candidate = []byte(restored)
			if batch.ArtifactEOF == retranslationArtifactEOFSingleLF {
				candidate = canonicalizeRetranslationArtifactEOF(candidate)
			}
		}
		if record.RepresentationBridge == nil && hasInheritedBridgePredecessor(record) {
			decoder, lineage, err := inheritedBridgeOutput(root, catalog, glossary, locale, record)
			if err != nil {
				return nil, err
			}
			if lineage != nil {
				protected = decoder
				restored, failures = protected.restore(string(raw))
				candidate = []byte(restored)
				if batch.ArtifactEOF == retranslationArtifactEOFSingleLF {
					candidate = canonicalizeRetranslationArtifactEOF(candidate)
				}
				historicalChecks = append(historicalChecks, lineage)
				inputs = append(inputs, generationBundleInput{"formal/representation-lineage/" + record.UnitID + ".json", "", mustJSON(lineage)})
			}
		}
		var approvedProof *ApprovedIntervalTransport
		if approvedSelection.entry(record.UnitID) != nil {
			approvedProof, err = buildApprovedIntervalTransport(root, catalog, glossary, locale, record, successor, *predecessor, evidence, approvedSelection)
			if err != nil {
				return nil, err
			}
			candidate, failures = evidence.candidate, nil
		}
		var recovery *CurrentValidationPredecessor
		if (len(failures) != 0 || !bytes.Equal(candidate, evidence.candidate)) && successor != nil && successor.AuthoritySource == CurrentValidationAuthority {
			proof, err := inspectCurrentValidationPredecessor(root, catalog, predecessorCatalog, glossary, locale, selected, record, successor, *predecessor, evidence)
			if err != nil {
				return nil, err
			}
			if proof.RecoveryClass != "DETERMINISTIC_REBASE_PASS" {
				return nil, fmt.Errorf("%s: current-validation predecessor recovery blocked (%s): %s", record.UnitID, proof.RecoveryClass, proof.Blocker)
			}
			recovery = &proof
			candidate, failures = evidence.candidate, nil
		}
		if len(failures) != 0 || !bytes.Equal(candidate, evidence.candidate) {
			return nil, fmt.Errorf("%s: predecessor protected raw response does not restore to Snapshot candidate: %s", record.UnitID, strings.Join(failures, "; "))
		}
		baseline := raw
		baselineRepositoryPath := rawPath
		if recovery != nil {
			for _, implementation := range currentValidationPredecessorImplementationPaths {
				key := "formal/current-validation-predecessor-implementation/" + implementation
				if !seenSurfaceReferences[key] {
					b, err := readSnapshotReferencedFile(root, implementation)
					if err != nil {
						return nil, err
					}
					expected := ""
					for _, ref := range recovery.ImplementationInputs {
						if ref.Path == implementation {
							expected = ref.SHA256
						}
					}
					if expected == "" || sum(b) != expected {
						return nil, fmt.Errorf("recovery implementation changed during construction")
					}
					inputs = append(inputs, generationBundleInput{key, implementation, b})
					seenSurfaceReferences[key] = true
				}
			}
			baseline, baselineRepositoryPath = []byte(recovery.Template), ""
			inputs = append(inputs, generationBundleInput{"formal/current-validation-predecessor-transport/" + filepath.Base(record.InputPath) + ".json", "", mustJSON(recovery)})
			inputs = append(inputs, generationBundleInput{"formal/historical-protected-candidates/" + filepath.Base(record.InputPath), rawPath, raw})
			for _, ref := range recovery.Dependencies {
				resolved, data, err := readHistoricalBridgeReference(root, locale, ref)
				if err != nil {
					return nil, err
				}
				key := "formal/current-validation-predecessor-dependencies/" + sum([]byte(ref.Path+"\n"+ref.SHA256)) + "-" + filepath.Base(ref.Path)
				if !seenSurfaceReferences[key] {
					inputs = append(inputs, generationBundleInput{key, resolved.Path, data})
					seenSurfaceReferences[key] = true
				}
			}
			historicalChecks = append(historicalChecks, recovery.lineage)
			if len(transports) > 0 {
				transports[0][record.UnitID] = recovery.output
			}
		}

		if successor != nil && successor.AuthoritySource == CurrentValidationAuthority {
			current, err := catalog.Unit(record.UnitID)
			if err != nil {
				return nil, err
			}
			projection, err := buildCommentRevisionTransport(root, catalog, glossary, locale, current, successor, evidence.candidate)
			if err != nil {
				return nil, err
			}
			if projection != nil {
				baseline = canonicalizeRetranslationArtifactEOF([]byte(projection.Text))
				baselineRepositoryPath = ""
				inputs = append(inputs, generationBundleInput{commentTransportPath(record), "", mustJSON(projection)})
				if recovery == nil {
					inputs = append(inputs, generationBundleInput{"formal/historical-protected-candidates/" + filepath.Base(record.InputPath), rawPath, raw})
				}
				if len(transports) > 0 {
					text, tokens, err := projection.ProtectedInput(glossary)
					if err != nil {
						return nil, err
					}
					transports[0][record.UnitID] = protectedTranslation{Text: text, Tokens: tokens, Adapter: projection}
				}
			}
		}
		if predecessorID != record.UnitID {
			current, err := catalog.Unit(record.UnitID)
			if err != nil {
				return nil, err
			}
			a, ok := current.Adapter.(ReplacementRevisionAdapter)
			if !ok {
				return nil, fmt.Errorf("replacement adapter cannot bind current protected predecessor")
			}
			if successor != nil && successor.AuthoritySource == RepresentationMachineAuthority {
				ua, ok := current.Adapter.(RepresentationRevisionAdapter)
				if !ok {
					return nil, fmt.Errorf("missing unbindable revision adapter")
				}
				failure, scaffold, proofErr := ua.UnbindableRevision(evidence.unit.Adapter, raw)
				if proofErr != nil {
					return nil, proofErr
				}
				bound := successor.ReplacementContext.Machine
				if !bytes.Equal(mustJSON(failure), mustJSON(bound.Failure)) || sum(scaffold) != bound.ScaffoldSHA256 {
					return nil, fmt.Errorf("unbindable revision scaffold/finding drift")
				}
				baseline = scaffold
				inputs = append(inputs, generationBundleInput{"formal/representation-revision-scope.json", "", mustJSON(bound)})
			} else {
				baseline, err = a.RebaseRevisionPredecessor(evidence.unit.Adapter, raw)
			}
			if err != nil {
				return nil, err
			}
			inputs = append(inputs, generationBundleInput{"formal/historical-protected-candidates/" + filepath.Base(record.InputPath), rawPath, raw})
			baselineRepositoryPath = "" // deterministic derived input, not archived raw bytes
		}
		if record.RepresentationBridge != nil {
			receipt := validatedBridge
			if receipt.UnitCount != len(selected) {
				return nil, fmt.Errorf("bridge Bundle exact-set mismatch")
			}
			bound := false
			for _, unit := range receipt.Units {
				if unit.UnitID == record.UnitID && record.RevisionFeedbackSource == "quality_check" && receipt.PreviousSnapshotID == record.PreviousSnapshotID && unit.PreviousRating == record.PreviousRating && unit.Finding == record.PreviousFinding && qualityCheckSnapshotIdentityMatches(unit.Snapshot, *predecessor) {
					bound = true
				}
			}
			if !bound {
				return nil, fmt.Errorf("bridge Bundle per-Unit QC authority mismatch")
			}
			for _, binding := range receipt.SameUnitBridge.Bindings {
				if binding.UnitID == record.UnitID {
					baseline = []byte(binding.Template)
				}
			}
			for _, binding := range receipt.SameUnitBridge.Bindings {
				if binding.UnitID == record.UnitID {
					archive, err := readSnapshotReferencedFile(root, binding.Archive.Path)
					if err != nil || sum(archive) != binding.Archive.SHA256 {
						return nil, fmt.Errorf("bridge archive drift")
					}
					_, files, err := readGenerationBundle(archive)
					if err != nil || sum(files[binding.TransportMember]) != binding.TransportSHA256 {
						return nil, fmt.Errorf("bridge transport member drift")
					}
					inputs = append(inputs, generationBundleInput{"formal/representation-bridge/transport/" + record.UnitID + ".json", "", files[binding.TransportMember]})
				}
			}
			baselineRepositoryPath = ""
			inputs = append(inputs, generationBundleInput{"formal/historical-protected-candidates/" + filepath.Base(record.InputPath), rawPath, raw}, generationBundleInput{"formal/representation-bridge/" + record.UnitID + ".json", record.RepresentationBridge.Path, bridgeReceiptBytes(receipt)})
			// Historical dependencies are exact path/SHA-bound by the immutable
			// receipt, not extra target-language inputs. Recheck them fresh here;
			// current-check reconstructs the same complete graph independently.
			if err := verifySameUnitBridgeDependencies(root, receipt); err != nil {
				return nil, err
			}

		}
		if approvedProof != nil {
			baseline, baselineRepositoryPath = []byte(approvedProof.Projection.Text), ""
			inputs = append(inputs, generationBundleInput{approvedIntervalProofMember(record.UnitID), "", mustJSON(approvedProof)}, generationBundleInput{"formal/historical-protected-candidates/" + filepath.Base(record.InputPath), rawPath, raw})
			if len(transports) > 0 {
				transports[0][record.UnitID] = approvedProof.output
			}
		}
		validation, err := readSnapshotReferencedFile(root, predecessor.ValidationPath)
		if err != nil || sum(validation) != predecessor.ValidationSHA256 {
			return nil, fmt.Errorf("%s: predecessor validation identity mismatch", record.UnitID)
		}
		inputs = append(inputs,
			generationBundleInput{"formal/protected-candidates/" + filepath.Base(record.InputPath), baselineRepositoryPath, baseline},
			generationBundleInput{"formal/predecessors/" + record.PreviousSnapshotID + "/validation/" + filepath.Base(record.InputPath) + ".json", predecessor.ValidationPath, validation},
		)
		if !seenSnapshots[record.PreviousSnapshotID] {
			path := filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, record.PreviousSnapshotID, "manifest.json"))
			data, err := readSnapshotReferencedFile(root, path)
			if err != nil {
				return nil, err
			}
			inputs = append(inputs, generationBundleInput{"formal/predecessors/" + record.PreviousSnapshotID + "/manifest.json", path, data})
			seenSnapshots[record.PreviousSnapshotID] = true
		}
	}
	if surfaceCapability {
		inputs, err = applySurfaceTransport(root, catalog, locale, selected, inputs)
		if err != nil {
			return nil, err
		}
	}
	for _, p := range historicalChecks {
		if err := p.check(root, locale); err != nil {
			return nil, err
		}
	}
	for _, check := range outputChecks {
		if err := check(); err != nil {
			return nil, err
		}
	}
	for path, want := range feedbackRefs {
		got, err := readSnapshotReferencedFile(root, path)
		if err != nil || !bytes.Equal(got, want) {
			return nil, fmt.Errorf("QC revision feedback changed during projection: %s", path)
		}
	}
	if len(predecessorOutputs) > 0 || len(selected) > 0 {
		for _, input := range inputs {
			if input.repositoryPath == "" {
				continue
			}
			got, err := readSnapshotReferencedFile(root, input.repositoryPath)
			if err != nil || !bytes.Equal(got, input.data) {
				return nil, fmt.Errorf("revision predecessor changed during projection: %s", input.repositoryPath)
			}
		}
	}
	checkedBridgeScopes := map[GlossaryArchiveReference]bool{}
	for _, record := range selected {
		if record.RepresentationBridge != nil {
			if checkedBridgeScopes[*record.RepresentationBridge] {
				continue
			}
			checkedBridgeScopes[*record.RepresentationBridge] = true
			if _, _, err := readSameUnitBridgeForInvocation(root, catalog, locale, *record.RepresentationBridge); err != nil {
				return nil, err
			}
			b, err := readSnapshotReferencedFile(root, record.RepresentationBridge.Path)
			if err != nil || sum(b) != record.RepresentationBridge.SHA256 {
				return nil, fmt.Errorf("bridge receipt changed before output")
			}
			var receipt QualityCheckSurfaceReopen
			if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &receipt); err != nil || receipt.SameUnitBridge == nil {
				return nil, fmt.Errorf("bridge receipt invalid before output")
			}
			if err := verifySameUnitBridgeDependencies(root, &receipt); err != nil {
				return nil, err
			}
		}
	}
	for id, scope := range currentScopes {
		path := qualityCheckSurfaceReopenPath(root, locale, id)
		raw, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(raw, currentValidationReopenBytes(scope.receipt)) {
			return nil, fmt.Errorf("current-validation receipt changed during projection")
		}
		// Includes all frozen candidate/source/input/validation/QC/finalization
		// hashes and assessment/scope bindings, not just the receipt JSON.
		if err := verifyCurrentValidationReopen(root, scope.catalog, scope.receipt, false); err != nil {
			return nil, err
		}
	}
	if err := checkApprovedIntervalInputs(root, inputs); err != nil {
		return nil, err
	}
	return inputs, nil
}
