package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

const RepresentationMachineAuthority = "representation-machine"
const RepresentationRevisionFeedback = "representation_machine"

type RepresentationAmbiguousAtom struct {
	Literal     string `json:"literal"`
	SourceStart int    `json:"source_start"`
	SourceEnd   int    `json:"source_end"`
	TargetSites []int  `json:"target_raw_sites"`
}
type RepresentationAmbiguousInterval struct {
	SourceStart    int                           `json:"source_start"`
	SourceEnd      int                           `json:"source_end"`
	SourceText     string                        `json:"source_text"`
	ReviewedTarget string                        `json:"reviewed_target"`
	LeftAnchor     string                        `json:"left_anchor"`
	RightAnchor    string                        `json:"right_anchor"`
	Atoms          []RepresentationAmbiguousAtom `json:"atoms"`
}
type RepresentationBindingFailure struct {
	Kind      string                            `json:"kind"`
	Intervals []RepresentationAmbiguousInterval `json:"intervals"`
}

// A source-only scaffold is an explicitly incomplete revision input. It is
// never a reconciled candidate or an assertion of language correctness.
type RepresentationRevisionAdapter interface {
	UnbindableRevision(TranslationUnitAdapter, []byte) (*RepresentationBindingFailure, []byte, error)
}
type RepresentationRevisionAuthority struct {
	Options          RepresentationReconciliationOptions `json:"options"`
	UnitID           string                              `json:"unit_id"`
	Predecessor      QualityCheckSnapshotUnit            `json:"predecessor"`
	QCInputs         []QualityCheckFinalizationInput     `json:"qc_inputs"`
	OriginalRaw      GlossaryArchiveReference            `json:"original_raw"`
	OriginalManifest GlossaryArchiveReference            `json:"original_manifest"`
	OriginalResult   GlossaryArchiveReference            `json:"original_result"`
	Failure          RepresentationBindingFailure        `json:"failure"`
	ScaffoldSHA256   string                              `json:"scaffold_sha256"`
}
type RepresentationRevisionOptions struct {
	Reconciliation RepresentationReconciliationOptions `json:"reconciliation"`
	UnitID         string                              `json:"unit_id"`
	ReopenID       string                              `json:"reopen_id"`
	Record         bool                                `json:"record"`
}

func machineRevisionFinding(f RepresentationBindingFailure) string {
	return "representation-unbindable: " + string(mustJSON(f)) + ". Revise only the listed frozen prose intervals and necessary adjacent grammar to express the complete original meaning with the current protected token set. The source-only scaffold is incomplete English input, not an accepted translation. Do not select a historical occurrence mechanically, copy a guessed binding, change unrelated prose, URLs, code or machine identities. This machine finding does not assert a language defect or change the predecessor A. Validation and independent re-QC are required."
}

func expectedRepresentationRevision(root string, c *Catalog, o RepresentationRevisionOptions) (*QualityCheckSurfaceReopen, error) {
	return expectedRepresentationRevisionInWorld(root, c, o, nil)
}

// This world is derived from the revision's hash-bound predecessor Snapshot,
// not from an incidental resolver used by predecessor QC or today's glossary.
// It is internal reconstruction input, never a new receipt or eligibility gate.
type representationRevisionHistoricalWorld struct {
	snapshot GlossaryArchiveReference
	glossary GlossaryArchiveReference
}

func historicalRepresentationRevisionWorld(root string, c *Catalog, r *QualityCheckSurfaceReopen) (*representationRevisionHistoricalWorld, error) {
	if r.SchemaVersion != qualityCheckSurfaceReopenSchemaVersion || r.EvidenceType != qualityCheckSurfaceReopenEvidenceType || r.AuthoritySource != RepresentationMachineAuthority || !reviewIDPattern.MatchString(r.ReopenID) || r.UnitCount != 1 || r.ReplacementContext == nil || r.ReplacementContext.Machine == nil || len(r.Units) != 1 || r.Units[0].Replacement == nil {
		return nil, fmt.Errorf("historical machine revision authority required")
	}
	a := r.ReplacementContext.Machine
	if r.Locale != a.Options.Locale || r.PreviousSnapshotID != a.Options.PreviousSnapshotID || r.Units[0].UnitID != a.UnitID || !qualityCheckSnapshotIdentityMatches(r.Units[0].Snapshot, a.Predecessor) {
		return nil, fmt.Errorf("historical revision predecessor binding mismatch")
	}
	resolution, err := replacementResolution(root, c, r.ReplacementContext)
	if err != nil {
		return nil, err
	}
	p, ok := resolution.Units[a.Predecessor.UnitID]
	if !ok || !reflect.DeepEqual(p, *r.Units[0].Replacement) || p.CurrentUnitID != a.UnitID || p.Package != a.Options.Package {
		return nil, fmt.Errorf("historical revision exact remap mismatch")
	}
	path := filepath.ToSlash(filepath.Join("data/quality-check-snapshots", r.Locale, a.Options.PreviousSnapshotID, "manifest.json"))
	ref, err := successorReference(root, path)
	if err != nil || ref.SHA256 != a.Options.SnapshotSHA256 {
		return nil, fmt.Errorf("historical revision Snapshot hash mismatch")
	}
	s, err := readQualityCheckSnapshot(root, r.Locale, a.Options.PreviousSnapshotID, false)
	if err != nil {
		return nil, err
	}
	found := false
	for _, u := range s.Units {
		found = found || qualityCheckSnapshotIdentityMatches(u, a.Predecessor)
	}
	if !found {
		return nil, fmt.Errorf("historical revision Snapshot predecessor missing")
	}
	if _, err := readArchivedGlossary(root, r.Locale, s.GlossarySHA256); err != nil {
		return nil, err
	}
	archive, err := glossaryArchivePath(r.Locale, s.GlossarySHA256)
	if err != nil {
		return nil, err
	}
	return &representationRevisionHistoricalWorld{snapshot: ref, glossary: GlossaryArchiveReference{Path: archive, SHA256: s.GlossarySHA256}}, nil
}

func expectedRepresentationRevisionInWorld(root string, c *Catalog, o RepresentationRevisionOptions, world *representationRevisionHistoricalWorld) (*QualityCheckSurfaceReopen, error) {
	if !reviewIDPattern.MatchString(o.ReopenID) || o.Reconciliation.Apply || len(o.Reconciliation.LanguageReviewIDs) != 0 {
		return nil, fmt.Errorf("exclusive read-only machine revision authority required")
	}
	var ctx *machineReconciliationContext
	var err error
	if world == nil {
		ctx, err = prepareMachineReconciliation(root, c, o.Reconciliation)
	} else {
		path := filepath.ToSlash(filepath.Join("data/quality-check-snapshots", o.Reconciliation.Locale, o.Reconciliation.PreviousSnapshotID, "manifest.json"))
		b, readErr := readSnapshotReferencedFile(root, path)
		if readErr != nil || world.snapshot != (GlossaryArchiveReference{Path: path, SHA256: o.Reconciliation.SnapshotSHA256}) || sum(b) != world.snapshot.SHA256 {
			return nil, fmt.Errorf("historical revision world Snapshot changed")
		}
		var snapshot QualityCheckSnapshotManifest
		if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &snapshot); err != nil {
			return nil, err
		}
		archive, archiveErr := glossaryArchivePath(o.Reconciliation.Locale, snapshot.GlossarySHA256)
		if archiveErr != nil || snapshot.Locale != o.Reconciliation.Locale || snapshot.SnapshotID != o.Reconciliation.PreviousSnapshotID || world.glossary != (GlossaryArchiveReference{Path: archive, SHA256: snapshot.GlossarySHA256}) {
			return nil, fmt.Errorf("historical revision world glossary is not Snapshot-bound")
		}
		ctx, err = prepareMachineReconciliationInWorld(root, c, o.Reconciliation, world.glossary.SHA256)
	}
	if err != nil {
		return nil, err
	}
	t, err := ctx.trustedUnit(root, o.UnitID)
	if err != nil {
		return nil, err
	}
	if !t.wasUnknown || t.replacement.OldUnitID == t.replacement.CurrentUnitID {
		return nil, fmt.Errorf("explicit unresolved representation replacement required")
	}
	current, err := c.Unit(o.UnitID)
	if err != nil {
		return nil, err
	}
	a, ok := current.Adapter.(RepresentationRevisionAdapter)
	if !ok {
		return nil, fmt.Errorf("adapter lacks exact unbindable revision proof")
	}
	if machine, ok := current.Adapter.(RepresentationMachineAdapter); !ok {
		return nil, fmt.Errorf("adapter lacks machine proof")
	} else if _, err = machine.ReconcileRepresentation(t.evidence.unit.Adapter, t.raw); err == nil {
		return nil, fmt.Errorf("bindable predecessor must use machine reconciliation")
	}
	failure, scaffold, err := a.UnbindableRevision(t.evidence.unit.Adapter, t.raw)
	if err != nil {
		return nil, err
	}
	if failure.Kind != "duplicate-prose-literal" || len(failure.Intervals) == 0 {
		return nil, fmt.Errorf("unsupported machine binding finding")
	}
	input, err := prepareTranslationUnitInput(current, ctx.glossary)
	if err != nil {
		return nil, err
	}
	if sum(canonicalizeRetranslationArtifactEOF([]byte(input.Text))) != t.replacement.CurrentInputSHA256 {
		return nil, fmt.Errorf("current revision protected input mismatch")
	}
	restored, failures := input.restore(string(scaffold))
	if len(failures) != 0 || bytes.Equal(canonicalizeRetranslationArtifactEOF([]byte(restored)), canonicalizeRetranslationArtifactEOF(t.evidence.candidate)) {
		return nil, fmt.Errorf("revision scaffold must be valid but explicitly incomplete; %v", failures)
	}
	authority := &RepresentationRevisionAuthority{Options: ctx.receipt.Options, UnitID: o.UnitID, Predecessor: t.old, QCInputs: ctx.receipt.QCInputs, OriginalRaw: t.rawRef, Failure: *failure, ScaffoldSHA256: sum(scaffold)}
	base := filepath.ToSlash(filepath.Join("data/retranslation-runs", o.Reconciliation.Locale, t.old.SelectedBatchID))
	authority.OriginalManifest, err = successorReference(root, base+"/manifest.json")
	if err != nil {
		return nil, err
	}
	authority.OriginalResult, err = successorReference(root, base+"/result.json")
	if err != nil {
		return nil, err
	}
	context := &TranslationUnitReplacementContext{Machine: authority, HistoricalDocuments: ctx.receipt.Options.HistoricalDocuments}
	p := t.replacement
	return &QualityCheckSurfaceReopen{SchemaVersion: qualityCheckSurfaceReopenSchemaVersion, EvidenceType: qualityCheckSurfaceReopenEvidenceType, AuthoritySource: RepresentationMachineAuthority, Locale: o.Reconciliation.Locale, ReopenID: o.ReopenID, PreviousSnapshotID: o.Reconciliation.PreviousSnapshotID, ReplacementContext: context, UnitCount: 1, Units: []QualityCheckSurfaceReopenUnit{{PreviousRating: "A", Replacement: &p, Index: t.old.Index, UnitID: o.UnitID, Finding: machineRevisionFinding(*failure), Snapshot: t.old}}}, nil
}

func verifyRepresentationRevision(root string, c *Catalog, r *QualityCheckSurfaceReopen) error {
	return verifyRepresentationRevisionInWorld(root, c, r, nil)
}

func verifyRepresentationRevisionInWorld(root string, c *Catalog, r *QualityCheckSurfaceReopen, world *representationRevisionHistoricalWorld) error {
	if r.ReplacementContext == nil || r.ReplacementContext.Machine == nil || r.ReplacementContext.QualityCheck != nil || r.ReplacementContext.PredecessorReopen != (GlossaryArchiveReference{}) || r.PreviewFailure != nil || len(r.Units) != 1 {
		return fmt.Errorf("exclusive machine replacement authority required")
	}
	a := r.ReplacementContext.Machine
	want, err := expectedRepresentationRevisionInWorld(root, c, RepresentationRevisionOptions{Reconciliation: a.Options, UnitID: a.UnitID, ReopenID: r.ReopenID}, world)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(want, r) {
		return fmt.Errorf("machine revision differs from exact recomputed authority")
	}
	return nil
}

// Uses the existing immutable quality-check-reopens receipt and export reader.
// Default is inspection. Record is a separate explicit Local-terminal action.
func PrepareRepresentationRevision(root string, c *Catalog, o RepresentationRevisionOptions) (*QualityCheckSurfaceReopen, string, error) {
	r, err := expectedRepresentationRevision(root, c, o)
	if err != nil {
		return nil, "", err
	}
	latest, err := selectLatestRetranslationUnits(root, c, r.Locale)
	if err != nil {
		return nil, "", err
	}
	if _, ok := latest.selectedByID[o.UnitID]; ok {
		return nil, "", fmt.Errorf("current successor already has a result; no machine revision fallback")
	}
	if err = qualityCheckReplacementUnconsumed(root, c, r); err != nil {
		return nil, "", err
	}
	path := qualityCheckSurfaceReopenPath(root, r.Locale, r.ReopenID)
	if _, err = os.Lstat(path); !os.IsNotExist(err) {
		return nil, "", fmt.Errorf("machine revision authority already exists or cannot be inspected")
	}
	rel, err := repositoryRelativePath(root, path)
	if err != nil {
		return nil, "", err
	}
	if !o.Record {
		return r, rel, nil
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, "", err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, "", err
	}
	if err = writeNewGenerationFile(path, append(data, '\n')); err != nil {
		return nil, "", err
	}
	return r, rel, nil
}

func (r *QualityCheckSurfaceReopen) originalQCReplacement() bool {
	return r.AuthoritySource == "quality-check" || r.AuthoritySource == RepresentationMachineAuthority
}
