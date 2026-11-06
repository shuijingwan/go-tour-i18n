package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

const representationReconciliationSchema = "go-learning/representation-reconciliation/v1"

type RepresentationMachineAdapter interface {
	ReconcileRepresentation(TranslationUnitAdapter, []byte) ([]byte, error)
}
type RepresentationReconciliationOptions struct {
	Locale              string                   `json:"locale"`
	Package             string                   `json:"package"`
	ReconciliationID    string                   `json:"reconciliation_id"`
	PreviousSnapshotID  string                   `json:"previous_snapshot_id"`
	SnapshotSHA256      string                   `json:"snapshot_sha256"`
	Impact              GlossaryArchiveReference `json:"formal_impact"`
	HistoricalDocuments GlossaryArchiveReference `json:"historical_documents"`
	LanguageReviewIDs   []string                 `json:"language_review_ids,omitempty"`
	Apply               bool                     `json:"apply"`
}
type representationImpact struct {
	Units []representationImpactUnit `json:"units"`
}
type representationImpactUnit struct {
	CurrentUnitID string          `json:"current_unit_id"`
	OldUnitID     string          `json:"snapshot_representation_unit_id"`
	SourcePath    string          `json:"source_path"`
	SourceSHA256  string          `json:"source_sha256"`
	Labels        map[string]bool `json:"labels"`
}
type RepresentationMachineUnit struct {
	Replacement            TranslationUnitReplacement `json:"replacement"`
	Predecessor            QualityCheckSnapshotUnit   `json:"predecessor"`
	OriginalRaw            GlossaryArchiveReference   `json:"original_raw"`
	OriginalManifest       GlossaryArchiveReference   `json:"original_manifest"`
	OriginalResult         GlossaryArchiveReference   `json:"original_result"`
	CurrentRaw             GlossaryArchiveReference   `json:"current_raw"`
	Candidate              GlossaryArchiveReference   `json:"candidate"`
	Validation             GlossaryArchiveReference   `json:"validation"`
	SemanticAuthority      *GlossaryArchiveReference  `json:"semantic_authority,omitempty"`
	SemanticInputs         []GlossaryArchiveReference `json:"semantic_inputs,omitempty"`
	UnknownResolvedMachine bool                       `json:"unknown_resolved_machine"`
}
type RepresentationReconciliation struct {
	Schema         string                              `json:"schema"`
	Options        RepresentationReconciliationOptions `json:"options"`
	Snapshot       GlossaryArchiveReference            `json:"snapshot"`
	GlossarySHA256 string                              `json:"glossary_sha256"`
	QCInputs       []QualityCheckFinalizationInput     `json:"qc_inputs"`
	Units          []RepresentationMachineUnit         `json:"units"`
	Identity       string                              `json:"identity"`
}
type RepresentationReconciliationRow struct {
	UnitID     string `json:"unit_id"`
	Route      string `json:"route"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
	WasUnknown bool   `json:"was_unknown"`
}
type RepresentationReconciliationPlan struct {
	Receipt  *RepresentationReconciliation     `json:"receipt"`
	Rows     []RepresentationReconciliationRow `json:"rows"`
	Ready    int                               `json:"ready"`
	Excluded int                               `json:"excluded"`
	Blocked  int                               `json:"blocked"`
	Applied  bool                              `json:"applied"`
}
type machineReconciliationContext struct {
	currentGlossarySHA string
	historical         bool
	receipt            *RepresentationReconciliation
	resolution         *TranslationUnitReplacementResolution
	snapshot           *QualityCheckSnapshotManifest
	effective          map[string]effectiveQualityCheckResult
	impact             map[string]representationImpactUnit
	glossary           *Glossary
	compatibility      *PreparedGlossaryCompatibility
	oldLatest          *latestRetranslationUnits
}

func machineReceiptPath(o RepresentationReconciliationOptions) string {
	return filepath.ToSlash(filepath.Join("data/translation-unit-reconciliations", o.Locale, o.ReconciliationID, "receipt.json"))
}
func machineReceiptIdentity(r RepresentationReconciliation) string {
	r.Identity = ""
	return sum(mustJSON(r))
}

func prepareMachineReconciliation(root string, c *Catalog, o RepresentationReconciliationOptions) (*machineReconciliationContext, error) {
	return prepareMachineReconciliationInWorld(root, c, o, "")
}

// A nonempty historicalGlossary pins an already recorded receipt's world. It
// verifies representation integrity only; current selection/apply always use
// prepareMachineReconciliation and its current full Review gate.
func prepareMachineReconciliationInWorld(root string, c *Catalog, o RepresentationReconciliationOptions, historicalGlossary string, worlds ...*PreparedGlossaryCompatibility) (*machineReconciliationContext, error) {
	return prepareMachineReconciliationScope(root, c, o, historicalGlossary, historicalGlossary != "", worlds...)
}

// Reading an already recorded transition always binds its complete historical
// package, even when the caller assesses today's glossary. Scope and glossary
// are independent axes: appended Units need their own current proof, but cannot
// retroactively become members of this receipt's predecessor Snapshot.
func prepareRecordedMachineReconciliation(root string, c *Catalog, receipt *RepresentationReconciliation, historicalGlossary string) (*machineReconciliationContext, error) {
	return prepareMachineReconciliationScope(root, c, receipt.Options, historicalGlossary, true)
}

func prepareMachineReconciliationScope(root string, c *Catalog, o RepresentationReconciliationOptions, historicalGlossary string, historicalScope bool, worlds ...*PreparedGlossaryCompatibility) (*machineReconciliationContext, error) {
	if len(worlds) > 1 {
		return nil, fmt.Errorf("one explicit historical compatibility world required")
	}
	if err := ValidateLocaleName(o.Locale); err != nil {
		return nil, err
	}
	if !reviewIDPattern.MatchString(o.ReconciliationID) || validateSnapshotID(o.PreviousSnapshotID) != nil || !validSHA256(o.SnapshotSHA256) {
		return nil, fmt.Errorf("explicit reconciliation/Snapshot identity required")
	}
	pkg, _, _, _, err := translationAuthorizationScope(c)
	if o.Package == "" {
		o.Package = pkg
	}
	if err != nil || pkg != o.Package || len(c.TranslationScope) == 0 {
		return nil, fmt.Errorf("complete explicit registered package scope required")
	}
	r, err := replacementResolutionInWorld(root, c, &TranslationUnitReplacementContext{HistoricalDocuments: o.HistoricalDocuments}, historicalGlossary != "")
	if err != nil {
		return nil, err
	}
	if historicalGlossary != "" && r.EventCatalog != nil {
		c = r.EventCatalog
	}
	snapshotRef, err := successorReference(root, filepath.ToSlash(filepath.Join("data/quality-check-snapshots", o.Locale, o.PreviousSnapshotID, "manifest.json")))
	if err != nil || snapshotRef.SHA256 != o.SnapshotSHA256 {
		return nil, fmt.Errorf("predecessor Snapshot hash mismatch")
	}
	b, err := readSnapshotReferencedFile(root, o.Impact.Path)
	if err != nil || !validSHA256(o.Impact.SHA256) || sum(b) != o.Impact.SHA256 {
		return nil, fmt.Errorf("formal impact evidence hash mismatch")
	}
	var impact representationImpact
	if err = json.Unmarshal(b, &impact); err != nil {
		return nil, err
	}
	byID := map[string]representationImpactUnit{}
	for _, u := range impact.Units {
		if u.CurrentUnitID == "" || byID[u.CurrentUnitID].CurrentUnitID != "" {
			return nil, fmt.Errorf("duplicate/empty impact identity")
		}
		byID[u.CurrentUnitID] = u
	}
	evaluation := newQualityCheckEvaluation(root, o.Locale, r.HistoricalCatalog)
	evaluation.originalOnly = true
	s, effective, err := evaluation.loadEffectiveQualityCheckResults(o.PreviousSnapshotID, map[string]bool{})
	if err != nil {
		return nil, err
	}
	qcCatalog := c
	if historicalScope {
		// The resolver has verified the frozen package authority and complete
		// replacement mapping. Today's membership may have appended Units;
		// those require independent current proof, not retrospective QC here.
		qcCatalog = r.HistoricalCatalog
		pkg, _, _, _, err := translationAuthorizationScope(qcCatalog)
		if err != nil || pkg != o.Package || len(qcCatalog.TranslationScope) == 0 {
			return nil, fmt.Errorf("complete historical registered package scope required")
		}
	}
	if len(s.Units) != len(qcCatalog.TranslationScope) || len(effective) != s.UnitCount {
		return nil, fmt.Errorf("complete initial independent QC required for machine reconciliation")
	}
	qcInputs, err := qualityCheckFinalizationInputs(root, o.Locale, o.PreviousSnapshotID, map[string]bool{})
	if err != nil {
		return nil, err
	}
	var glossaryBytes []byte
	if historicalGlossary != "" {
		glossaryBytes, err = readArchivedGlossary(root, o.Locale, historicalGlossary)
	} else {
		if err = RequireCurrentGlossaryReview(root, o.Locale); err != nil {
			return nil, err
		}
		glossaryBytes, err = readSnapshotReferencedFile(root, glossaryReviewGlossaryPath(o.Locale))
	}
	if err != nil {
		return nil, err
	}
	g, err := parseGlossary(o.Locale, bytes.NewReader(glossaryBytes))
	if err != nil {
		return nil, err
	}
	o.Apply = false
	receipt := &RepresentationReconciliation{Schema: representationReconciliationSchema, Options: o, Snapshot: snapshotRef, GlossarySHA256: sum(glossaryBytes), QCInputs: qcInputs, Units: []RepresentationMachineUnit{}}
	oldLatest, err := selectLatestRetranslationUnits(root, r.HistoricalCatalog, o.Locale, true)
	if err != nil {
		return nil, err
	}
	var compatibility *PreparedGlossaryCompatibility
	if len(worlds) == 1 {
		compatibility = worlds[0]
		if compatibility == nil || historicalGlossary == "" || compatibility.root != root || compatibility.locale != o.Locale || compatibility.historicalSHA != historicalGlossary {
			return nil, fmt.Errorf("explicit historical compatibility world binding mismatch")
		}
		// Preparation above still revalidates all receipt/source/QC inputs.
		// The existing resolver is usable only for this exact historical scope.
		pkg, authority, scope, units, err := translationAuthorizationScope(compatibility.catalog)
		nextPkg, nextAuthority, nextScope, nextUnits, nextErr := translationAuthorizationScope(r.HistoricalCatalog)
		if err != nil || nextErr != nil || pkg != nextPkg || authority != nextAuthority || scope != nextScope || len(units) != len(nextUnits) {
			return nil, fmt.Errorf("explicit historical compatibility package scope mismatch")
		}
		for i, u := range units {
			next := nextUnits[i]
			if u.ID != next.ID || u.SourceSHA256 != next.SourceSHA256 || !bytes.Equal(u.Source, next.Source) {
				return nil, fmt.Errorf("explicit historical compatibility source mismatch")
			}
		}
	} else {
		compatibility = PrepareGlossaryCompatibility(root, o.Locale, r.HistoricalCatalog)
		compatibility.historicalSHA = historicalGlossary
	}
	return &machineReconciliationContext{receipt: receipt, currentGlossarySHA: sum(glossaryBytes), historical: historicalGlossary != "", resolution: r, snapshot: s, effective: effective, impact: byID, glossary: g, compatibility: compatibility, oldLatest: oldLatest}, nil
}

// buildUnit repeats the complete machine proof. No reviewer rating or model
// invocation is emitted. The final candidate must be byte-identical to its A
// predecessor, including its recorded EOF policy.
type trustedReconciliationUnit struct {
	replacement TranslationUnitReplacement
	old         QualityCheckSnapshotUnit
	evidence    *snapshotUnitRepositoryEvidence
	batch       *RetranslationBatchManifest
	rawRef      GlossaryArchiveReference
	raw         []byte
	wasUnknown  bool
}

func (m *machineReconciliationContext) trustedUnit(root string, id string) (*trustedReconciliationUnit, error) {
	impact, ok := m.impact[id]
	if !ok || !impact.Labels["representation_reconciliation_required"] {
		return nil, fmt.Errorf("Unit lacks formal representation impact")
	}
	for _, key := range []string{"true_language_defect", "unknown"} {
		if _, ok := impact.Labels[key]; !ok {
			return nil, fmt.Errorf("incomplete impact labels")
		}
	}
	if impact.Labels["true_language_defect"] {
		return nil, fmt.Errorf("language defect requires formal revision")
	}
	p, ok := m.resolution.Units[impact.OldUnitID]
	if !ok || p.Package != m.receipt.Options.Package || p.CurrentUnitID != id || p.OldUnitID == id || p.SourcePath != impact.SourcePath || p.SourceSHA256 != impact.SourceSHA256 {
		return nil, fmt.Errorf("impact/replacement exact binding mismatch")
	}
	prior, ok := m.effective[p.OldUnitID]
	if !ok || prior.rating != "A" || prior.rubric != TranslationQualityRubric {
		return nil, fmt.Errorf("trusted predecessor QC A required")
	}
	var old QualityCheckSnapshotUnit
	for _, u := range m.snapshot.Units {
		if u.UnitID == p.OldUnitID {
			old = u
		}
	}
	if !qualityCheckSnapshotIdentityMatches(old, prior.unit) {
		return nil, fmt.Errorf("predecessor A identity mismatch")
	}
	latest, ok := m.oldLatest.selectedByID[old.UnitID]
	if !ok || latest.result.Status != "passed" || !choiceMatchesSnapshot(m.receipt.Options.Locale, latest, old) {
		return nil, fmt.Errorf("newer historical attempt supersedes the reviewed predecessor; refusing fallback")
	}
	resolution := m.compatibility.Resolve(m.snapshot.GlossarySHA256, m.currentGlossarySHA, "tu:"+p.OldUnitID)
	if resolution.Status != "exact" && resolution.Status != "compatible" {
		return nil, fmt.Errorf("glossary-stale predecessor: %s", resolution.Status)
	}
	evidence, err := readSnapshotUnitRepositoryEvidence(root, m.resolution.HistoricalCatalog, m.receipt.Options.Locale, old)
	if err != nil {
		return nil, err
	}
	if evidence.record.InputSHA256 != p.OldInputSHA256 {
		return nil, fmt.Errorf("frozen old protected input mismatch")
	}
	batch, err := readRetranslationProcessManifest(evidence.batchDir, m.receipt.Options.Locale, old.SelectedBatchID)
	if err != nil {
		return nil, err
	}
	// An accepted historical invocation keeps its actual archived glossary and
	// provenance. Reuse the existing frozen-origin reader; current compatibility
	// and validation are independent gates above/below, never bypassed here.
	if err = verifyAcceptedFrozenInput(root, m.receipt.Options.Locale, evidence.unit, old, evidence.candidate, m.snapshot.GlossarySHA256, m.resolution.HistoricalCatalog); err != nil {
		return nil, err
	}
	name := filepath.Base(evidence.record.InputPath)
	ext := filepath.Ext(name)
	flat := strings.TrimSuffix(name, ext)
	attempt, err := retryValidationAttemptForExtension(evidence.validation.RawResponsePath, flat, ext)
	if err != nil || attempt != old.Attempt {
		return nil, fmt.Errorf("original final attempt mismatch")
	}
	retryDir := filepath.Join(evidence.batchDir, "retries", flat)
	if attempt > 1 {
		if err = validateRetryAttemptSequenceForExtension(retryDir, attempt, attempt, ext); err != nil {
			return nil, err
		}
	} else {
		entries, err := os.ReadDir(retryDir)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		if len(entries) > 0 {
			return nil, fmt.Errorf("original attempt is no longer final")
		}
	}
	rawRef, err := successorReference(root, filepath.ToSlash(filepath.Join("data/retranslation-runs", m.receipt.Options.Locale, old.SelectedBatchID, evidence.validation.RawResponsePath)))
	if err != nil {
		return nil, err
	}
	raw, err := readSnapshotReferencedFile(root, rawRef.Path)
	if err != nil {
		return nil, err
	}
	return &trustedReconciliationUnit{p, old, evidence, batch, rawRef, raw, impact.Labels["unknown"]}, nil
}

func (m *machineReconciliationContext) buildUnit(root string, c *Catalog, id string, historical ...bool) (RepresentationMachineUnit, []byte, []byte, error) {
	var empty RepresentationMachineUnit
	trust, err := m.trustedUnit(root, id)
	if err != nil {
		return empty, nil, nil, err
	}
	p, old, evidence, batch, rawRef, raw := trust.replacement, trust.old, trust.evidence, trust.batch, trust.rawRef, trust.raw
	current, err := c.Unit(id)
	if err != nil {
		return empty, nil, nil, err
	}
	adapter, ok := current.Adapter.(RepresentationMachineAdapter)
	if !ok {
		return empty, nil, nil, fmt.Errorf("adapter lacks deterministic machine reconciliation proof")
	}
	next, err := adapter.ReconcileRepresentation(evidence.unit.Adapter, raw)
	var semanticRef *GlossaryArchiveReference
	var semanticCandidate []byte
	if err != nil {
		if len(m.receipt.Options.LanguageReviewIDs) == 0 {
			return empty, nil, nil, err
		}
		// A repeated-site machine ambiguity must retain its actual reason. A
		// receipt for another Unit does not make this a semantic-review question.
		if ua, ok := current.Adapter.(RepresentationRevisionAdapter); ok {
			if _, _, proofErr := ua.UnbindableRevision(evidence.unit.Adapter, raw); proofErr == nil {
				return empty, nil, nil, fmt.Errorf("representation-unbindable: %w; exact machine-authorized revision required, not semantic occurrence selection", err)
			}
		}
		review, ref, authorityErr := m.semanticAuthority(root, c, id)
		if authorityErr != nil {
			return empty, nil, nil, authorityErr
		}
		sa, ok := current.Adapter.(RepresentationSemanticAdapter)
		if !ok {
			return empty, nil, nil, fmt.Errorf("no semantic reconciliation adapter")
		}
		next, semanticCandidate, err = sa.ReviewedReconciliation(evidence.unit.Adapter, raw, review.Question)
		if err != nil {
			return empty, nil, nil, err
		}
		semanticRef = ref
	}
	input, err := prepareTranslationUnitInput(current, m.glossary)
	if err != nil {
		return empty, nil, nil, err
	}
	if sum(canonicalizeRetranslationArtifactEOF([]byte(input.Text))) != p.CurrentInputSHA256 {
		return empty, nil, nil, fmt.Errorf("current protected input mismatch")
	}
	restored, failures := input.restore(string(next))
	candidate := []byte(restored)
	if semanticRef != nil {
		candidate = semanticCandidate
		failures = nil
	}
	if batch.ArtifactEOF == retranslationArtifactEOFSingleLF {
		candidate = canonicalizeRetranslationArtifactEOF(candidate)
	}
	if len(failures) > 0 || !bytes.Equal(candidate, evidence.candidate) {
		return empty, nil, nil, fmt.Errorf("successor changes reviewed translation bytes: %v", failures)
	}
	validate := validateTranslationUnitWithGlossary
	if len(historical) > 0 && historical[0] {
		// A recorded receipt proves its original byte-preserving transition.
		// Current re-attestation separately enforces today's comment gate.
		validate = validateTranslationUnitRepresentation
	}
	if err = validate(root, c, current, m.receipt.Options.Locale, candidate, m.glossary); err != nil {
		return empty, nil, nil, err
	}
	base := filepath.ToSlash(filepath.Dir(machineReceiptPath(m.receipt.Options))) + "/" + id
	u := RepresentationMachineUnit{Replacement: p, Predecessor: old, OriginalRaw: rawRef, CurrentRaw: GlossaryArchiveReference{Path: base + ".raw", SHA256: sum(next)}, Candidate: GlossaryArchiveReference{Path: base + ".candidate", SHA256: sum(candidate)}, UnknownResolvedMachine: trust.wasUnknown}
	u.SemanticAuthority = semanticRef
	if semanticRef != nil {
		u.UnknownResolvedMachine = false
		data, err := readSnapshotReferencedFile(root, semanticRef.Path)
		if err != nil {
			return empty, nil, nil, err
		}
		var review RepresentationReviewReceipt
		if err = json.Unmarshal(data, &review); err != nil {
			return empty, nil, nil, err
		}
		u.SemanticInputs = []GlossaryArchiveReference{*semanticRef, review.Bundle, review.Evidence}
	}
	oldBase := filepath.ToSlash(filepath.Join("data/retranslation-runs", m.receipt.Options.Locale, old.SelectedBatchID))
	u.OriginalManifest, err = successorReference(root, oldBase+"/manifest.json")
	if err != nil {
		return empty, nil, nil, err
	}
	u.OriginalResult, err = successorReference(root, oldBase+"/result.json")
	if err != nil {
		return empty, nil, nil, err
	}
	// This is a machine proof record, deliberately not RetranslationValidation
	// and never stored in retranslation-runs as a new translation attempt.
	validation := mustJSON(struct {
		Kind                                       string
		Replacement                                TranslationUnitReplacement
		Predecessor                                QualityCheckSnapshotUnit
		RawSHA256, CandidateSHA256, GlossarySHA256 string
	}{representationReconciliationSchema, p, old, sum(next), sum(candidate), m.receipt.GlossarySHA256})
	u.Validation = GlossaryArchiveReference{Path: base + ".validation.json", SHA256: sum(validation)}
	return u, next, validation, nil
}

// ReconcileTranslationUnitRepresentations is read-only unless Apply is true.
// Every pending impacted Unit is inspected; any unresolved proof blocks the
// entire write. Current candidates (including revisions) are never overwritten.
func ReconcileTranslationUnitRepresentations(root string, c *Catalog, o RepresentationReconciliationOptions) (result *RepresentationReconciliationPlan, resultErr error) {
	ctx, err := prepareMachineReconciliation(root, c, o)
	if err != nil {
		return nil, err
	}
	latest, err := selectLatestRetranslationUnits(root, c, o.Locale)
	if err != nil {
		return nil, err
	}
	plan := &RepresentationReconciliationPlan{Receipt: ctx.receipt, Rows: []RepresentationReconciliationRow{}}
	exported, _, err := scanRetranslationBatches(filepath.Join(root, "data/retranslation-runs", o.Locale), o.Locale, c)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	for _, unit := range latest.ordered {
		impact, ok := ctx.impact[unit.ID]
		if !ok || !impact.Labels["representation_reconciliation_required"] {
			continue
		}
		p := ctx.resolution.Units[impact.OldUnitID]
		row := RepresentationReconciliationRow{UnitID: unit.ID, Route: p.Route, WasUnknown: impact.Labels["unknown"]}
		if choice, ok := latest.selectedByID[unit.ID]; ok {
			if _, err := validateSelectedRetranslationCandidate(root, c, o.Locale, ctx.glossary, unit, choice); err != nil {
				row.Status = "blocked"
				row.Reason = "existing current successor is failed/stale: " + err.Error()
				plan.Blocked++
			} else {
				row.Status = "excluded-current-passed"
				plan.Excluded++
			}
		} else if exported[unit.ID].BatchID != "" {
			row.Status = "blocked"
			row.Reason = "current successor already exported; inspect its immutable batch before machine reconciliation"
			plan.Blocked++
		} else {
			u, raw, validation, err := ctx.buildUnit(root, c, unit.ID)
			if err != nil {
				row.Status = "blocked"
				row.Reason = err.Error()
				plan.Blocked++
			} else {
				row.Status = "ready"
				plan.Ready++
				plan.Receipt.Units = append(plan.Receipt.Units, u)
				files[u.CurrentRaw.Path] = raw
				files[u.Validation.Path] = validation
				files[u.Candidate.Path], err = readSnapshotReferencedFile(root, u.Predecessor.CandidatePath)
				if err != nil {
					return nil, err
				}
			}
		}
		plan.Rows = append(plan.Rows, row)
	}
	plan.Receipt.Identity = machineReceiptIdentity(*plan.Receipt)
	if !o.Apply {
		return plan, nil
	}
	if plan.Blocked > 0 || plan.Ready == 0 {
		return plan, fmt.Errorf("machine reconciliation requires a non-empty exact set without unresolved proofs: blocked=%d ready=%d", plan.Blocked, plan.Ready)
	}
	parentFile, err := compatibilityPath(root, filepath.ToSlash(filepath.Join("data/translation-unit-reconciliations", o.Locale, ".inventory")), true)
	if err != nil {
		return plan, err
	}
	finalDir := filepath.Join(filepath.Dir(parentFile), o.ReconciliationID)
	if err = requireMissingSnapshotDirectory(finalDir); err != nil {
		return plan, err
	}
	parent := filepath.Dir(finalDir)
	if err = os.MkdirAll(parent, 0755); err != nil {
		return plan, err
	}
	staging, err := os.MkdirTemp(parent, ".representation-staging-")
	if err != nil {
		return plan, err
	}
	defer func() {
		if err := os.RemoveAll(staging); err != nil {
			resultErr = fmt.Errorf("mutation=unknown; inspect retained representation staging %s: %v; previous error: %v", staging, err, resultErr)
		}
	}()
	files[machineReceiptPath(o)] = append(mustJSON(plan.Receipt), '\n')
	names := []string{}
	for path := range files {
		names = append(names, path)
	}
	sort.Strings(names)
	for _, path := range names {
		if err = os.WriteFile(filepath.Join(staging, filepath.Base(path)), files[path], 0644); err != nil {
			return plan, err
		}
	}
	if err = os.Rename(staging, finalDir); err != nil {
		return plan, err
	}
	plan.Applied = true
	return plan, nil
}

type machineSelection struct {
	receipt *RepresentationReconciliation
	unit    RepresentationMachineUnit
	ref     GlossaryArchiveReference
	ctx     *machineReconciliationContext
}

func readMachineReceipt(root string, ref GlossaryArchiveReference) (*RepresentationReconciliation, error) {
	b, err := readSnapshotReferencedFile(root, ref.Path)
	if err != nil || sum(b) != ref.SHA256 {
		return nil, fmt.Errorf("reconciliation receipt hash mismatch")
	}
	var r RepresentationReconciliation
	if err = decodeStrictCourseSourceDescriptionReviewJSON(b, &r); err != nil {
		return nil, err
	}
	if r.Schema != representationReconciliationSchema || r.Identity != machineReceiptIdentity(r) || machineReceiptPath(r.Options) != ref.Path || len(r.Units) == 0 {
		return nil, fmt.Errorf("invalid reconciliation receipt identity")
	}
	seen := map[string]bool{}
	for _, u := range r.Units {
		if seen[u.Replacement.CurrentUnitID] {
			return nil, fmt.Errorf("duplicate reconciled Unit")
		}
		seen[u.Replacement.CurrentUnitID] = true
	}
	return &r, nil
}
func attachMachineSelections(root string, c *Catalog, locale string, latest *latestRetranslationUnits) error {
	base := filepath.Join(root, "data/translation-unit-reconciliations", locale)
	entries, err := os.ReadDir(base)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	ordered := map[string]bool{}
	for _, u := range latest.ordered {
		ordered[u.ID] = true
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			return fmt.Errorf("unfinished representation reconciliation transaction: %s", e.Name())
		}
		ref, err := successorReference(root, filepath.ToSlash(filepath.Join("data/translation-unit-reconciliations", locale, e.Name(), "receipt.json")))
		if err != nil {
			return err
		}
		r, err := readMachineReceipt(root, ref)
		if err != nil {
			return err
		}
		if r.Options.Locale != locale {
			return fmt.Errorf("reconciliation locale mismatch")
		}
		var ctx *machineReconciliationContext
		for _, u := range r.Units {
			id := u.Replacement.CurrentUnitID
			if !ordered[id] {
				continue
			}
			if existing, ok := latest.selectedByID[id]; ok {
				if existing.machine != nil {
					return fmt.Errorf("ambiguous representation successor evidence")
				}
				continue
			}
			if _, err := c.Unit(id); err != nil {
				continue
			}
			if ctx == nil {
				ctx = &machineReconciliationContext{receipt: r}
			}
			latest.selectedByID[id] = selectedRetranslationUnit{batchID: u.Predecessor.SelectedBatchID, machine: &machineSelection{r, u, ref, ctx}, result: RetranslationUnitResult{UnitID: id, UnitKind: UnitKindPage, Status: "passed", CandidatePath: u.Candidate.Path, ValidationPath: u.Validation.Path}}
		}
	}
	return nil
}
func (s *machineSelection) snapshot(index int) QualityCheckSnapshotUnit {
	u := s.unit.Predecessor
	u.Index = index
	u.UnitID = s.unit.Replacement.CurrentUnitID
	u.CandidatePath = s.unit.Candidate.Path
	u.CandidateSHA256 = s.unit.Candidate.SHA256
	u.ValidationPath = s.unit.Validation.Path
	u.ValidationSHA256 = s.unit.Validation.SHA256
	u.RepresentationReconciliation = &s.ref
	return u
}
func (s *machineSelection) validate(root string, c *Catalog) (*snapshotUnitRepositoryEvidence, error) {
	current, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(s.receipt.Options.Locale))
	if err != nil {
		return nil, err
	}
	if sum(current) == s.receipt.GlossarySHA256 {
		return s.validateInWorld(root, c, false)
	}
	return s.validateCurrentAssessment(root, c)
}

func (s *machineSelection) validateInWorld(root string, c *Catalog, historical bool) (*snapshotUnitRepositoryEvidence, error) {
	if s.ctx.resolution == nil || s.ctx.historical != historical {
		glossarySHA := ""
		if historical {
			glossarySHA = s.receipt.GlossarySHA256
		}
		ctx, err := prepareRecordedMachineReconciliation(root, c, s.receipt, glossarySHA)
		if err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(ctx.receipt.QCInputs, s.receipt.QCInputs) || ctx.receipt.Snapshot != s.receipt.Snapshot {
			return nil, fmt.Errorf("reconciliation authority drift")
		}
		ctx.receipt.GlossarySHA256 = s.receipt.GlossarySHA256
		*s.ctx = *ctx
	}
	if historical && s.ctx.resolution.EventCatalog != nil {
		c = s.ctx.resolution.EventCatalog
	}
	compatibility := s.ctx.compatibility.Resolve(s.receipt.GlossarySHA256, s.ctx.currentGlossarySHA, "tu:"+s.unit.Predecessor.UnitID)
	if compatibility.Status != "exact" && compatibility.Status != "compatible" {
		return nil, fmt.Errorf("reconciled successor glossary-stale: status=%s reason=%s", compatibility.Status, compatibility.Reason)
	}
	expected, raw, validation, err := s.ctx.buildUnit(root, c, s.unit.Replacement.CurrentUnitID, true)
	if err != nil || !reflect.DeepEqual(expected, s.unit) {
		return nil, fmt.Errorf("reconciliation machine proof mismatch: %v", err)
	}
	for _, item := range []struct {
		ref  GlossaryArchiveReference
		want []byte
	}{{s.unit.CurrentRaw, raw}, {s.unit.Validation, validation}, {s.unit.Candidate, nil}} {
		b, err := readSnapshotReferencedFile(root, item.ref.Path)
		if err != nil || sum(b) != item.ref.SHA256 || (item.want != nil && !bytes.Equal(b, item.want)) {
			return nil, fmt.Errorf("reconciliation artifact drift: %s", item.ref.Path)
		}
	}
	evidence, err := readSnapshotUnitRepositoryEvidence(root, s.ctx.resolution.HistoricalCatalog, s.receipt.Options.Locale, s.unit.Predecessor)
	if err != nil {
		return nil, err
	}
	// Keep the real historical batch/record/validation; expose current source and
	// byte-exact candidate. The explicit Snapshot ref distinguishes this evidence.
	evidence.unit, err = c.Unit(s.unit.Replacement.CurrentUnitID)
	evidence.machine = s
	return evidence, err
}

func machineSnapshotEvidence(root string, c *Catalog, locale string, u QualityCheckSnapshotUnit, contexts map[string]*machineSelection) (*snapshotUnitRepositoryEvidence, *machineSelection, error) {
	return machineSnapshotEvidenceInWorld(root, c, locale, u, contexts, false)
}

// Historical readers reconstruct the receipt-bound glossary world. Current
// readers retain the default current validation; neither world authorizes the
// other, and each selection revalidates its exact machine artifacts.
func machineSnapshotEvidenceInWorld(root string, c *Catalog, locale string, u QualityCheckSnapshotUnit, contexts map[string]*machineSelection, historical bool) (*snapshotUnitRepositoryEvidence, *machineSelection, error) {
	ref := u.RepresentationReconciliation
	if ref == nil {
		return nil, nil, fmt.Errorf("explicit reconciliation reference required")
	}
	var selected *machineSelection
	if contexts != nil {
		selected = contexts[ref.SHA256+u.UnitID]
	}
	if selected == nil {
		r, err := readMachineReceipt(root, *ref)
		if err != nil {
			return nil, nil, err
		}
		if r.Options.Locale != locale {
			return nil, nil, fmt.Errorf("machine Snapshot locale mismatch")
		}
		ctx := &machineReconciliationContext{receipt: r}
		if contexts != nil {
			for _, other := range contexts {
				if other.ref == *ref {
					ctx = other.ctx
					break
				}
			}
		}
		for _, item := range r.Units {
			if item.Replacement.CurrentUnitID == u.UnitID {
				selected = &machineSelection{r, item, *ref, ctx}
			}
		}
		if selected == nil {
			return nil, nil, fmt.Errorf("Snapshot Unit not in reconciliation exact set")
		}
		if contexts != nil {
			contexts[ref.SHA256+u.UnitID] = selected
		}
	}
	if !qualityCheckSnapshotIdentityMatches(u, selected.snapshot(u.Index)) {
		return nil, nil, fmt.Errorf("Snapshot differs from reconciled successor")
	}
	var e *snapshotUnitRepositoryEvidence
	var err error
	if historical {
		e, err = selected.validateInWorld(root, c, true)
	} else {
		e, err = selected.validate(root, c)
	}
	return e, selected, err
}

func machineAuthorityReferences(s *machineSelection) []GlossaryArchiveReference {
	refs := append([]GlossaryArchiveReference{s.ref, s.receipt.Snapshot, s.receipt.Options.Impact, s.receipt.Options.HistoricalDocuments, s.unit.OriginalManifest, s.unit.OriginalResult, s.unit.OriginalRaw, s.unit.CurrentRaw, s.unit.Candidate, s.unit.Validation}, s.unit.SemanticInputs...)
	if s.ctx.resolution != nil {
		refs = append(refs, s.ctx.resolution.SuccessorReferences...)
	}
	return refs
}

func machineBoundPredecessor(root string, s *QualityCheckSnapshotManifest) (string, error) {
	predecessor := ""
	for _, u := range s.Units {
		if u.RepresentationReconciliation == nil {
			continue
		}
		r, err := readMachineReceipt(root, *u.RepresentationReconciliation)
		if err != nil {
			return "", err
		}
		if predecessor != "" && predecessor != r.Options.PreviousSnapshotID {
			return "", fmt.Errorf("mixed machine reconciliation predecessors")
		}
		predecessor = r.Options.PreviousSnapshotID
	}
	return predecessor, nil
}
func (e *qualityCheckEvaluation) machinePredecessorEvaluation(s *QualityCheckSnapshotManifest, previous string) (*qualityCheckEvaluation, error) {
	var historical *Catalog
	for _, u := range s.Units {
		if u.RepresentationReconciliation == nil {
			continue
		}
		_, selected, err := machineSnapshotEvidenceInWorld(e.root, e.catalog, e.locale, u, e.machineEvidence, e.originalOnly || e.compatibility.historicalSHA != "")
		if err != nil {
			return nil, err
		}
		if selected.receipt.Options.PreviousSnapshotID != previous {
			p, err := readQualityCheckSnapshot(e.root, e.locale, previous, false)
			if err != nil {
				return nil, err
			}
			found := false
			for _, v := range p.Units {
				if qualityCheckSnapshotIdentityMatches(u, v) {
					found = true
				}
			}
			if !found {
				return nil, fmt.Errorf("machine successor requires its exact QC predecessor")
			}
			continue
		}
		if historical == nil {
			historical = selected.ctx.resolution.HistoricalCatalog
		}
	}
	if historical != nil {
		prior := newQualityCheckEvaluation(e.root, e.locale, historical)
		prior.originalOnly = true
		return prior, nil
	}
	return e, nil
}
func (e *qualityCheckEvaluation) machineCarry(u QualityCheckSnapshotUnit, previous string, inherited map[string]effectiveQualityCheckResult) (effectiveQualityCheckResult, bool, error) {
	_, s, err := machineSnapshotEvidence(e.root, e.catalog, e.locale, u, e.machineEvidence)
	if err != nil {
		return effectiveQualityCheckResult{}, false, err
	}
	if s.receipt.Options.PreviousSnapshotID != previous {
		p, ok := inherited[u.UnitID]
		return p, ok, nil
	}
	p, ok := inherited[s.unit.Predecessor.UnitID]
	if !ok || p.rating != "A" || p.rubric != TranslationQualityRubric || !qualityCheckSnapshotIdentityMatches(p.unit, s.unit.Predecessor) {
		return p, false, fmt.Errorf("machine successor lacks its original exact A judgment")
	}
	p.unit = u
	p.currentReattested = true
	return p, true, nil
}
