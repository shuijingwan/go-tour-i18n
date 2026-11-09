package i18n

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

type TransportRecoveryOptions struct {
	contextEpoch     []RecoveryDependency
	decoderContext   bool // Readonly installed decoder use; never record/export admission.
	Locale           string
	ContentPackage   string
	PredecessorBatch string
	Supersedes       *TransportRecoveryReference
}

type TransportRecoveryBlocker struct {
	Code   string `json:"code"`
	UnitID string `json:"unit_id,omitempty"`
	Detail string `json:"detail"`
}

type TransportRecoveryPlan struct {
	Schema           string                             `json:"schema"`
	Status           string                             `json:"status"`
	Locale           string                             `json:"locale"`
	ContentPackage   string                             `json:"content_package"`
	PredecessorBatch string                             `json:"predecessor_batch"`
	DirectDriftUnits []string                           `json:"direct_drift_units"`
	RecoveryUnits    []string                           `json:"recovery_units"`
	DirectDriftCount int                                `json:"direct_drift_count"`
	RecoveryCount    int                                `json:"recovery_unit_count"`
	Blockers         []TransportRecoveryBlocker         `json:"blockers"`
	PlannedAuthority *InstalledDerivedTransportRecovery `json:"planned_authority"`
	Identity         *string                            `json:"identity"`
	Path             *string                            `json:"path"`
	Recorded         bool                               `json:"recorded"`
	Mutation         string                             `json:"mutation"`
	members          map[string][]byte
}

func PreflightTransportRecovery(root string, catalog *Catalog, o TransportRecoveryOptions) (*TransportRecoveryPlan, error) {
	o.decoderContext = false
	o.contextEpoch = nil
	p := &TransportRecoveryPlan{Schema: "go-learning/transport-recovery-plan/v1", Status: "BLOCKED", Locale: o.Locale, ContentPackage: o.ContentPackage, PredecessorBatch: o.PredecessorBatch, Mutation: "none", DirectDriftUnits: []string{}, RecoveryUnits: []string{}, Blockers: []TransportRecoveryBlocker{}}
	a, members, err := prepareTransportRecovery(root, catalog, o)
	if err != nil {
		p.Blockers = append(p.Blockers, TransportRecoveryBlocker{Code: recoveryBlockerCode(err), Detail: err.Error()})
		return p, err
	}
	if a == nil {
		p.Status = "CURRENT"
		return p, nil
	}
	if _, err = verifyRecoveryRecordPrior(root, a); err == nil {
		err = recoveryRequireAbsent(root, recoveryRecordPending(a))
	}
	if err != nil {
		p.Blockers = append(p.Blockers, TransportRecoveryBlocker{Code: "OVERLAPPING_OR_RECORDED_RECOVERY", Detail: err.Error()})
		return p, err
	}
	p.Status, p.PlannedAuthority, p.members = "READY", a, members
	for _, d := range a.DirectDrift {
		p.DirectDriftUnits = append(p.DirectDriftUnits, d.UnitID)
	}
	p.RecoveryUnits = append(p.RecoveryUnits, a.RecoveryUnits.UnitIDs...)
	p.DirectDriftCount, p.RecoveryCount = len(p.DirectDriftUnits), len(p.RecoveryUnits)
	id, path := a.Identity, transportRecoveryDirectory(a.Locale, a.Identity)+"/authority.json"
	p.Identity, p.Path = &id, &path
	return p, nil
}

type recoveryTypedBlocker struct{ code, detail string }

func (e recoveryTypedBlocker) Error() string { return e.code + ": " + e.detail }
func recoveryBlockerCode(err error) string {
	if e, ok := err.(recoveryTypedBlocker); ok {
		return e.code
	}
	return "INTEGRITY_OR_CURRENTNESS_FAILED"
}

func recoveryDiskReference(root, path string) (RecoveryFileReference, []byte, error) {
	b, err := readCompatibilityFile(root, path)
	if err != nil {
		return RecoveryFileReference{}, nil, err
	}
	return recoveryMemberReference(path, b), b, nil
}

// Only a complete registered descriptor can authorize recovery. An unscoped
// caller is projected explicitly; an explicitly different scope is rejected.
func recoveryPackageCatalog(c *Catalog, id string) (*Catalog, error) {
	if id == "" || id == tourNativeTranslationPackage {
		return nil, fmt.Errorf("explicit registered recovery package required")
	}
	return SelectCurrentValidationReopenPackage(c, id)
}

func prepareTransportRecovery(root string, catalog *Catalog, o TransportRecoveryOptions, installedChild ...string) (*InstalledDerivedTransportRecovery, map[string][]byte, error) {
	if ValidateLocaleName(o.Locale) != nil || validateBatchID(o.PredecessorBatch) != nil {
		return nil, nil, fmt.Errorf("exact locale/predecessor required")
	}
	c, err := recoveryPackageCatalog(catalog, o.ContentPackage)
	if err != nil {
		return nil, nil, err
	}
	pkg, auth, scope, ordered, err := translationAuthorizationScope(c)
	if err != nil || pkg != o.ContentPackage {
		return nil, nil, fmt.Errorf("recovery complete package mismatch")
	}
	base := "data/retranslation-runs/" + o.Locale + "/" + o.PredecessorBatch + "/"
	mref, mb, err := recoveryDiskReference(root, base+"manifest.json")
	if err != nil {
		return nil, nil, err
	}
	m, err := readRetranslationProcessManifest(filepath.Join(root, filepath.FromSlash(base)), o.Locale, o.PredecessorBatch)
	if err != nil {
		return nil, nil, err
	}
	if len(m.Units) == 0 || len(m.Units) > MaxRetranslationExportLimit {
		return nil, nil, fmt.Errorf("recovery original batch count outside 1..60")
	}
	if err := ensureNoGenerationImportPending(filepath.Join(root, filepath.FromSlash(base))); err != nil {
		return nil, nil, err
	}
	selection, oldFiles, err := importedApprovedIntervalSelection(root, c, m)
	if err != nil || selection == nil {
		return nil, nil, fmt.Errorf("recovery requires exact installed approved selection: %v", err)
	}
	for _, r := range m.Units {
		if r.RevisionFeedbackSource != CurrentValidationRevisionFeedback || r.UnitKind != m.UnitKind || r.PreviousSnapshotID != m.Units[0].PreviousSnapshotID || r.RevisionAuthorizationID != m.Units[0].RevisionAuthorizationID {
			return nil, nil, fmt.Errorf("recovery requires exclusive original current-validation scope")
		}
	}
	iref, ib, err := recoveryDiskReference(root, base+"generation-imports/initial-manifest.json")
	if err != nil {
		return nil, nil, err
	}
	var receipt GenerationResultBundleManifest
	if err := contentidentity.StrictJSON(ib, &receipt); err != nil {
		return nil, nil, err
	}
	aref, ab, err := recoveryDiskReference(root, importedGenerationArchive(o.Locale, receipt.GenerationBundleSHA256))
	if err != nil {
		return nil, nil, err
	}
	gm, _, err := readGenerationBundle(ab)
	if err != nil {
		return nil, nil, err
	}
	if err = checkImportedGenerationIdentity(receipt, gm, ab, oldFiles); err != nil {
		return nil, nil, err
	}
	currentModel, err := DefaultGenerationModelForProvider(receipt.Provider)
	if err != nil || receipt.Model != currentModel {
		return nil, nil, fmt.Errorf("recovery cannot change provider/model or reuse an obsolete model selection")
	}
	if gm.TaskKind != "translation-unit-revision" && gm.TaskKind != TransportRecoveryTaskKind {
		return nil, nil, fmt.Errorf("unsupported predecessor invocation task")
	}
	if gm.Attempt != 1 || len(gm.ExpectedOutputs) != len(m.Units) || !bytes.Equal(oldFiles["formal/manifest.json"], mb) {
		return nil, nil, fmt.Errorf("recovery initial invocation scope mismatch")
	}
	resultRef, _, err := recoveryDiskReference(root, base+"result.json")
	if err != nil {
		return nil, nil, err
	}
	latest, err := selectLatestRetranslationUnits(root, c, o.Locale)
	if err != nil {
		return nil, nil, err
	}
	if len(installedChild) > 0 && installedChild[0] != "" {
		if err = normalizeRecoveryContextSelection(root, m, latest, installedChild[0], o.decoderContext); err != nil {
			return nil, nil, err
		}
	}
	selectionIdentity := recoverySelectionIdentity(latest)
	g, err := LoadGlossary(root, o.Locale)
	if err != nil {
		return nil, nil, err
	}
	if err = RequireCurrentGlossaryReview(root, o.Locale); err != nil {
		return nil, nil, err
	}
	if !bytes.Equal(oldFiles["formal/glossary.yaml"], mustReadRecoveryGlossary(root, o.Locale)) {
		return nil, nil, fmt.Errorf("recovery glossary changed")
	}
	projections := map[string]protectedTranslation{}
	inputs, err := revisionGenerationInputsCore(root, c, o.Locale, m.Units, []map[string]protectedTranslation{projections}, false, selection)
	if err != nil {
		return nil, nil, err
	}
	inputs, err = applySurfaceTransportInWorld(root, c, o.Locale, m.Units, inputs, true)
	if err != nil {
		return nil, nil, err
	}
	currentFiles := map[string][]byte{}
	for k, v := range oldFiles {
		currentFiles[k] = v
	}
	for _, in := range inputs {
		currentFiles[in.path] = in.data
	}
	for _, unit := range m.Units {
		p, err := recoveryProofFromFiles(currentFiles, unit.UnitID)
		if err != nil {
			return nil, nil, err
		}
		if p == nil {
			continue
		}
		for _, ref := range p.Dependencies {
			b, err := readCompatibilityReference(root, ref)
			if err != nil {
				return nil, nil, err
			}
			currentFiles["diagnostic-dependencies/"+ref.SHA256] = b
		}
	}
	_, reopen, err := surfaceRevisionCatalog(root, c, o.Locale, m.Units[0].RevisionAuthorizationID)
	if err != nil {
		return nil, nil, err
	}
	if reopen.AuthoritySource != CurrentValidationAuthority || reopen.CurrentValidation == nil || len(reopen.Units) != len(m.Units) {
		return nil, nil, fmt.Errorf("original language authority scope mismatch")
	}
	languageAssessment, languageRefs, err := assessRecoveryLanguageAuthority(root, c, m, reopen, g)
	if err != nil {
		return nil, nil, err
	}
	rref, _, err := recoveryDiskReference(root, reopen.revisionEvidence().Path)
	if err != nil {
		return nil, nil, err
	}
	sref, _, err := recoveryDiskReference(root, reopen.CurrentValidation.Snapshot.Path)
	if err != nil {
		return nil, nil, err
	}
	packageBinding := RecoveryPackage{ID: pkg, AuthoritySHA256: auth, ScopeIdentitySHA256: scope, Units: []RecoverySourceUnit{}, References: []RecoveryDependency{}}
	for _, d := range c.TranslationPackages {
		if d.ID == pkg && d.AuthoritySHA256 == auth && d.Membership != nil {
			for _, r := range d.Membership.References {
				ref, b, err := recoveryDiskReference(root, r.Path)
				if err != nil || ref.SHA256 != r.SHA256 {
					return nil, nil, fmt.Errorf("recovery package membership reference mismatch")
				}
				_ = b
				packageBinding.References = append(packageBinding.References, RecoveryDependency{"package-membership", ref})
			}
		}
	}
	packageBinding.References, err = canonicalRecoveryDependencySet(packageBinding.References)
	if err != nil {
		return nil, nil, err
	}
	for _, u := range ordered {
		r, b, err := recoveryDiskReference(root, u.SourcePath)
		if err != nil || validateSnapshotSourceIdentity(c, u, b) != nil {
			return nil, nil, fmt.Errorf("recovery current package source mismatch")
		}
		packageBinding.Units = append(packageBinding.Units, RecoverySourceUnit{UnitID: u.ID, Kind: u.Kind, Source: r})
	}
	language := RecoveryLanguageAuthority{Reopen: rref, ReopenIdentity: reopen.CurrentValidation.Identity, Snapshot: sref, Scope: append([]RetranslationBatchUnit(nil), m.Units...)}
	language.References = languageRefs
	if m.TransportRecovery != nil {
		prior, _, err := readRecoveryExactAuthority(root, *m.TransportRecovery)
		if err != nil {
			return nil, nil, err
		}
		if err = validateRecoveryChildManifest(prior, m); err != nil {
			return nil, nil, err
		}
		if err = verifyRecoveryInstalledExecution(root, c, m); err != nil {
			return nil, nil, err
		}
		language.Scope = append([]RetranslationBatchUnit(nil), prior.LanguageAuthority.Scope...)
	}
	semantic := DerivedTransportSemanticContext{Locale: o.Locale, Package: packageBinding, Language: language}
	for _, r := range m.Units {
		u, err := c.Unit(r.UnitID)
		if err != nil {
			return nil, nil, err
		}
		p, err := prepareTranslationUnitInput(u, g)
		if err != nil {
			return nil, nil, err
		}
		if sum(canonicalizeRetranslationArtifactEOF([]byte(p.Text))) != r.InputSHA256 {
			return nil, nil, fmt.Errorf("recovery source/protection drift")
		}
		semantic.OrderedScope = append(semantic.OrderedScope, RecoverySourceUnit{UnitID: r.UnitID, Kind: r.UnitKind, Source: recoveryMemberReference(r.SourcePath, u.Source), InputSHA256: r.InputSHA256, RepresentationIdentity: sum(mustJSON(p))})
	}
	materials := newHistoricalConstructionResolver(root, c, m, oldFiles)
	oldConstruction, oldDecoders, err := deriveRecoveryConstruction(root, c, m, oldFiles, semantic, true, materials)
	if err != nil {
		return nil, nil, err
	}
	newConstruction, _, err := deriveRecoveryConstruction(root, c, m, currentFiles, semantic, false)
	if err != nil {
		return nil, nil, err
	}
	a := &InstalledDerivedTransportRecovery{
		// The exact stored construction member is another installed contract;
		// its mismatch cannot be hidden by an unchanged approved proof envelope.
		Schema: TransportRecoverySchema, Capability: TransportRecoveryCapability, Locale: o.Locale,
		ContentPackage: packageBinding, Predecessor: RecoveryInvocation{BatchID: m.BatchID, Manifest: mref, Result: resultRef, TaskKind: gm.TaskKind, ImportReceipt: iref, Archive: aref, InputIdentitySHA256: gm.InputIdentitySHA256, Provider: receipt.Provider, Model: receipt.Model},
		LanguageAuthority: language, CurrentAssessment: RecoveryAssessment{OriginalBaseline: languageAssessment, HistoricalIntegrity: true, LanguageAuthorityIdentity: language.ReopenIdentity},
		AllowedAction: RecoveryAllowedAction{Operation: "retranslation-export-with-transport-recovery", Provider: receipt.Provider, Model: receipt.Model, NewLanguageGenerationRequired: true}, Supersedes: o.Supersedes,
	}
	members := map[string][]byte{}
	contractMismatch := len(oldFiles[transportRecoveryConstructionMember]) > 0 && !bytes.Equal(oldFiles[transportRecoveryConstructionMember], mustJSON(newConstruction))
	constructionDeltas, err := classifyRecoveryConstructionDifference(oldConstruction, newConstruction, oldFiles, currentFiles, root, materials)
	if err != nil {
		return nil, nil, err
	}
	for _, r := range m.Units {
		u, _ := c.Unit(r.UnitID)
		choice, ok := latest.selectedByID[r.UnitID]
		if !ok || choice.batchID != m.BatchID || choice.result.Status != "passed" || !selectedRetranslationIdentityMatches(u, choice) {
			return nil, nil, fmt.Errorf("recovery latest selection is not complete original PASS")
		}
		final, err := recoveryHistoricalFinalUnit(root, c, g, m, choice, gm, oldFiles, oldDecoders[r.UnitID], projections[r.UnitID], receipt.Provider, receipt.Model)
		if err != nil {
			return nil, nil, err
		}
		a.Predecessor.FinalUnits = append(a.Predecessor.FinalUnits, *final)
		validator, err := currentTranslationAuthority(root, u, o.Locale)
		if err != nil {
			return nil, nil, err
		}
		a.CurrentAssessment.CurrentCandidateValidation = append(a.CurrentAssessment.CurrentCandidateValidation, RecoveryValidationAssessment{r.UnitID, final.Candidate.SHA256, validator, "passed"})
		oldProof, err := recoveryProofFromFiles(oldFiles, r.UnitID)
		if err != nil {
			return nil, nil, err
		}
		newProof, err := recoveryProofFromFiles(currentFiles, r.UnitID)
		if err != nil {
			return nil, nil, err
		}
		if oldProof == nil && newProof == nil {
			continue
		}
		if oldProof == nil || newProof == nil {
			return nil, nil, fmt.Errorf("approved proof inventory drift")
		}
		oldBytes, newBytes := oldFiles[approvedIntervalProofMember(r.UnitID)], currentFiles[approvedIntervalProofMember(r.UnitID)]
		if bytes.Equal(oldBytes, newBytes) && !contractMismatch {
			continue
		}
		deltas, err := classifyRecoveryProofDifference(c, u, oldProof, newProof, oldFiles, currentFiles, materials)
		if err != nil {
			return nil, nil, err
		}
		byPath := map[string]RecoveryDependencyDelta{}
		for _, d := range append(deltas, constructionDeltas...) {
			byPath[d.Role+"\x00"+d.Old.Path] = d
		}
		deltas = nil
		var keys []string
		for k := range byPath {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			deltas = append(deltas, byPath[k])
		}
		if len(deltas) == 0 || oldConstruction.Identity == newConstruction.Identity {
			return nil, nil, recoveryTypedBlocker{"ENCODING_ONLY_MISMATCH", "exact proof changed without typed construction dependency drift"}
		}
		op, np := "proofs/old/"+r.UnitID+".json", "proofs/current/"+r.UnitID+".json"
		members[op], members[np] = oldBytes, newBytes
		a.DirectDrift = append(a.DirectDrift, RecoveryDirectDrift{UnitID: r.UnitID, Classification: "construction-dependency-drift", OldProof: recoveryMemberReference(op, oldBytes), OldProofIdentity: oldProof.Identity, CurrentProof: recoveryMemberReference(np, newBytes), CurrentProofIdentity: newProof.Identity, OldConstructionIdentity: oldConstruction.Identity, CurrentConstructionIdentity: newConstruction.Identity, Dependencies: deltas})
	}
	if len(a.DirectDrift) == 0 {
		if err = materials.Verify(); err != nil {
			return nil, nil, err
		}
		if err = verifyRecoveryHistoricalBytes(root, a); err != nil {
			return nil, nil, err
		}
		var refs []RecoveryFileReference
		for _, r := range newConstruction.Dependencies {
			refs = append(refs, r.Ref)
		}
		if err = verifyRecoveryByteReferences(root, refs); err != nil {
			return nil, nil, err
		}
		fresh, err := selectLatestRetranslationUnits(root, c, o.Locale)
		if err != nil {
			return nil, nil, err
		}
		if len(installedChild) > 0 {
			if err = normalizeRecoveryContextSelection(root, m, fresh, installedChild[0], o.decoderContext); err != nil {
				return nil, nil, err
			}
		}
		if !o.decoderContext && selectionIdentity != recoverySelectionIdentity(fresh) {
			return nil, nil, fmt.Errorf("recovery latest selection changed during assessment")
		}
		return nil, nil, nil
	}
	rootName := aref.SHA256
	a.RecoveryUnits.ContextRoot = rootName
	direct := map[string]bool{}
	for _, d := range a.DirectDrift {
		direct[d.UnitID] = true
	}
	for _, r := range m.Units {
		a.RecoveryUnits.UnitIDs = append(a.RecoveryUnits.UnitIDs, r.UnitID)
		a.RecoveryUnits.Edges = append(a.RecoveryUnits.Edges, RecoveryContextEdge{r.UnitID, rootName, "complete-installed-decoder-context"})
		if !direct[r.UnitID] {
			a.RecoveryUnits.NonDirectUnitIDs = append(a.RecoveryUnits.NonDirectUnitIDs, r.UnitID)
		}
	}
	members["construction.json"] = mustJSON(newConstruction)
	members["construction-old-derived.json"] = mustJSON(oldConstruction)
	if b := oldFiles[transportRecoveryConstructionMember]; len(b) > 0 {
		members["construction-old-frozen.json"] = b
		oldRef := recoveryMemberReference("construction-old-frozen.json", b)
		currentRef := recoveryMemberReference("construction.json", members["construction.json"])
		for i := range a.DirectDrift {
			a.DirectDrift[i].FrozenConstruction = &oldRef
			a.DirectDrift[i].CurrentConstruction = &currentRef
		}
	}
	materialRefs, err := materials.References()
	if err != nil {
		return nil, nil, err
	}
	a.CurrentAssessment.References = append(a.CurrentAssessment.References, materialRefs...)
	members["assessment.json"] = mustJSON(a.CurrentAssessment)
	a.Construction = recoveryMemberReference("construction.json", members["construction.json"])
	// Freeze latest selections outside R as well. Changing another candidate
	// between preflight and record must not silently alter the package context.
	for _, u := range latest.ordered {
		if o.decoderContext {
			continue
		}
		choice, ok := latest.selectedByID[u.ID]
		if !ok {
			return nil, nil, fmt.Errorf("incomplete current package selections")
		}
		if choice.machine != nil {
			for _, r := range []GlossaryArchiveReference{choice.machine.unit.Candidate, choice.machine.unit.Validation, choice.machine.unit.CurrentRaw} {
				ref, _, err := recoveryDiskReference(root, r.Path)
				if err != nil || ref.SHA256 != r.SHA256 {
					return nil, nil, fmt.Errorf("latest machine selection changed")
				}
				a.CurrentEpoch = append(a.CurrentEpoch, RecoveryDependency{"latest-selection", ref})
			}
			continue
		}
		p, err := repositoryRelativePath(root, choice.batchDir)
		if err != nil {
			return nil, nil, err
		}
		for _, name := range []string{"manifest.json", "result.json"} {
			ref, _, err := recoveryDiskReference(root, p+"/"+name)
			if err != nil {
				return nil, nil, err
			}
			a.CurrentEpoch = append(a.CurrentEpoch, RecoveryDependency{Role: "latest-selection", Ref: ref})
		}
		for _, name := range []string{choice.result.CandidatePath, choice.result.ValidationPath} {
			if name == "" {
				continue
			}
			ref, _, err := recoveryDiskReference(root, p+"/"+name)
			if err != nil {
				return nil, nil, err
			}
			a.CurrentEpoch = append(a.CurrentEpoch, RecoveryDependency{"latest-selection", ref})
		}
	}
	if o.decoderContext {
		a.CurrentEpoch = append(a.CurrentEpoch, o.contextEpoch...)
	}
	epoch := map[string]RecoveryDependency{}
	for _, d := range append(append(append(append(a.CurrentEpoch, newConstruction.Dependencies...), languageRefs...), packageBinding.References...), materialRefs...) {
		epoch[d.Role+"\x00"+d.Ref.Path] = d
	}
	a.CurrentEpoch = nil
	for _, d := range epoch {
		a.CurrentEpoch = append(a.CurrentEpoch, d)
	}
	for _, path := range transportRecoveryImplementationPaths {
		ref, _, err := recoveryDiskReference(root, path)
		if err != nil {
			return nil, nil, err
		}
		a.CurrentEpoch = append(a.CurrentEpoch, RecoveryDependency{"recovery-implementation", ref})
	}
	a.CurrentEpoch, err = sortedRecoveryDependencies(a.CurrentEpoch)
	if err != nil {
		return nil, nil, err
	}
	var names []string
	for k := range members {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, name := range names {
		a.Members = append(a.Members, recoveryMemberReference(name, members[name]))
	}
	a.Identity = transportRecoveryIdentity(*a)
	if err = validateRecoveryAuthority(a); err != nil {
		return nil, nil, err
	}
	refs := []RecoveryFileReference{}
	for _, r := range a.CurrentEpoch {
		refs = append(refs, r.Ref)
	}
	if err = verifyRecoveryByteReferences(root, refs); err != nil {
		return nil, nil, err
	}
	if err = verifyRecoveryHistoricalBytes(root, a); err != nil {
		return nil, nil, err
	}
	if err = materials.Verify(); err != nil {
		return nil, nil, err
	}
	fresh, err := selectLatestRetranslationUnits(root, c, o.Locale)
	if err != nil {
		return nil, nil, err
	}
	if len(installedChild) > 0 {
		if err = normalizeRecoveryContextSelection(root, m, fresh, installedChild[0], o.decoderContext); err != nil {
			return nil, nil, err
		}
	}
	if !o.decoderContext && selectionIdentity != recoverySelectionIdentity(fresh) {
		return nil, nil, fmt.Errorf("recovery latest selection changed during assessment")
	}
	return a, members, nil
}

func mustReadRecoveryGlossary(root, locale string) []byte {
	b, _ := readCompatibilityFile(root, glossaryReviewGlossaryPath(locale))
	return b
}

func classifyRecoveryProofDifference(c *Catalog, u *TranslationUnit, old, current *ApprovedIntervalTransport, oldFiles, currentFiles map[string][]byte, materials ...*historicalConstructionResolver) ([]RecoveryDependencyDelta, error) {
	a, err := approvedConstructionInvariantV1(c, u, old)
	if err != nil {
		return nil, err
	}
	b, err := approvedConstructionInvariantV1(c, u, current)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(mustJSON(a), mustJSON(b)) {
		return nil, recoveryTypedBlocker{"UNCLASSIFIED_PROOF_MISMATCH", "non-construction authority or typed payload changed"}
	}
	if len(old.Dependencies) != len(current.Dependencies) {
		return nil, recoveryTypedBlocker{"UNCLASSIFIED_PROOF_MISMATCH", "dependency inventory changed"}
	}
	var out []RecoveryDependencyDelta
	for i, x := range old.Dependencies {
		y := current.Dependencies[i]
		if x.Path != y.Path {
			return nil, recoveryTypedBlocker{"UNCLASSIFIED_PROOF_MISMATCH", "dependency ordering/roles changed"}
		}
		if x.SHA256 == y.SHA256 {
			continue
		}
		ob, err := recoveryArchiveReference(oldFiles, x)
		if len(materials) > 0 {
			g, e := materials[0].graph(u.ID, x.Path)
			if e != nil {
				return nil, e
			}
			ob, err = materials[0].Read(u.ID, x, "approved-construction-implementation", g.Declaration.Producer.Version)
		}
		if err != nil {
			return nil, err
		}
		nb, err := recoveryArchiveReference(currentFiles, y)
		if err != nil {
			return nil, err
		}
		decls, err := classifyApprovedDependencyDelta(x.Path, ob, nb)
		if err != nil {
			return nil, recoveryTypedBlocker{"UNCLASSIFIED_PROOF_MISMATCH", err.Error()}
		}
		out = append(out, RecoveryDependencyDelta{Role: "approved-construction-implementation", Old: recoveryMemberReference(x.Path, ob), Current: recoveryMemberReference(y.Path, nb), Declarations: decls})
	}
	return out, nil
}

// Complete version-owned invariant fields are explicit. No arbitrary JSON
// normalization or deletion of proof fields is used to create an equivalence.
type approvedRecoveryInvariantV1 struct {
	Schema                 string
	Mode                   string
	SelectionSHA256        string
	RevisionRecord         RetranslationBatchUnit
	LanguageAuthority      GlossaryArchiveReference
	Snapshot               QualityCheckSnapshotUnit
	SemanticAuthority      GlossaryArchiveReference
	ProvenanceSource       string
	LegacyReference        *GlossaryArchiveReference
	HistoricalVerification *VerifiedLegacyGenerationProvenance
	RequestedOptions       LegacyGenerationProvenanceOptions
	CurrentAuthority       string
	Projection             *DerivedApprovedConstruction
}

func approvedConstructionInvariantV1(c *Catalog, u *TranslationUnit, p *ApprovedIntervalTransport) (approvedRecoveryInvariantV1, error) {
	projection, _, err := typedApprovedConstruction(c, u, p)
	if err != nil {
		return approvedRecoveryInvariantV1{}, err
	}
	return approvedRecoveryInvariantV1{p.Schema, p.Mode, p.SelectionSHA256, p.Record, p.Reopen, p.Snapshot, p.SemanticAuthority, p.ProvenanceSource, p.LegacyReference, p.HistoricalVerification, p.RequestedOptions, p.CurrentAuthority, projection}, nil
}

func recoveryHistoricalFinalUnit(root string, c *Catalog, g *Glossary, m *RetranslationBatchManifest, choice selectedRetranslationUnit, initial GenerationBundleManifest, files map[string][]byte, oldAdapter TranslationUnitAdapter, current protectedTranslation, provider, model string) (*RecoveryFinalUnit, error) {
	verified, err := verifyHistoricalFinalCandidate(root, c, g, m, choice, oldAdapter, current)
	if err != nil {
		return nil, err
	}
	u, attempt := verified.unit, verified.attempt
	base, err := repositoryRelativePath(root, choice.batchDir)
	if err != nil {
		return nil, err
	}
	cr, cb, err := recoveryDiskReference(root, verified.candidate.path)
	if err != nil {
		return nil, err
	}
	vr, vb, err := recoveryDiskReference(root, verified.validationFile.path)
	if err != nil {
		return nil, err
	}
	rr, rb, err := recoveryDiskReference(root, verified.raw.path)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(cb, verified.candidate.data) || !bytes.Equal(vb, verified.validationFile.data) || !bytes.Equal(rb, verified.raw.data) {
		return nil, fmt.Errorf("historical final candidate changed during verification")
	}
	f := &RecoveryFinalUnit{UnitID: u.ID, Attempt: attempt, Raw: rr, Candidate: cr, Validation: vr}
	if err = verifyRecoveryFinalHistory(root, m, f, choice.manifest); err != nil {
		return nil, err
	}
	for n := 1; n <= attempt; n++ {
		gm := GenerationBundleManifest{Attempt: n, TaskKind: initial.TaskKind}
		if n > 1 {
			gm.TaskKind = "translation-unit-retry"
			gm.ExpectedOutputs = []GenerationBundleExpectedOutput{{UnitID: u.ID, BundlePath: "outputs/" + filepath.Base(choice.manifest.InputPath)}}
		}
		path := generationImportReceiptPath(choice.batchDir, gm)
		rel, err := repositoryRelativePath(root, path)
		if err != nil {
			return nil, err
		}
		r, ib, err := recoveryDiskReference(root, rel)
		if err != nil {
			return nil, err
		}
		var receipt GenerationResultBundleManifest
		if err = contentidentity.StrictJSON(ib, &receipt); err != nil {
			return nil, err
		}
		ar, ab, err := recoveryDiskReference(root, importedGenerationArchive(m.Locale, receipt.GenerationBundleSHA256))
		if err != nil {
			return nil, err
		}
		bm, bf, err := readGenerationBundle(ab)
		if err != nil {
			return nil, err
		}
		if err = checkImportedGenerationIdentity(receipt, bm, ab, bf); err != nil {
			return nil, err
		}
		if bm.Locale != m.Locale || bm.BatchID != m.BatchID || bm.Attempt != n || bm.TaskKind != gm.TaskKind || !bytes.Equal(bf["formal/manifest.json"], files["formal/manifest.json"]) {
			return nil, fmt.Errorf("historical invocation attempt scope mismatch")
		}
		if err = verifyRecoveryHistoricalBundleScope(m, bm, choice.manifest.UnitID, n); err != nil {
			return nil, err
		}
		if receipt.Provider != provider || receipt.Model != model {
			return nil, fmt.Errorf("predecessor attempts have inconsistent provider/model")
		}
		outs := map[string]TransportBundleFile{}
		for _, o := range receipt.Outputs {
			if _, ok := outs[o.BundlePath]; ok {
				return nil, fmt.Errorf("duplicate historical output origin")
			}
			outs[o.BundlePath] = o
		}
		for _, o := range bm.ExpectedOutputs {
			f, ok := outs[o.BundlePath]
			b, err := readCompatibilityFile(root, base+"/"+o.InstallPath)
			if !ok || err != nil || f.SHA256 != sum(b) || f.Size != int64(len(b)) {
				return nil, fmt.Errorf("historical output provenance mismatch")
			}
		}
		f.History = append(f.History, RecoveryDependency{"invocation-receipt", r}, RecoveryDependency{"invocation-archive", ar})
	}
	return f, nil
}

func stringsRecoveryInputBase(p string) string {
	p = filepath.Base(p)
	return p[:len(p)-len(filepath.Ext(p))]
}
