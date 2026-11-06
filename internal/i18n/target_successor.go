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
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

const HistoricalPromotionRecoverySchema = "go-learning/historical-promotion-recovery/v1"
const NativePromotionReceiptSchema = "go-learning/native-promotion-receipt/v1"

// TargetSuccessorEvidence records an accepted-target relation. Recovery is
// explicitly NOT an original apply event. Native receipts are published only
// after both canonical and status installation have completed.
type TargetSuccessorEvidence struct {
	Schema            string                          `json:"schema"`
	Kind              string                          `json:"evidence_kind"`
	Reason            string                          `json:"reason"`
	Locale            string                          `json:"locale"`
	SnapshotID        string                          `json:"snapshot_id"`
	HistoricalClosure *GlossaryArchiveReference       `json:"historical_closure,omitempty"`
	Snapshot          GlossaryArchiveReference        `json:"snapshot"`
	Finalization      GlossaryArchiveReference        `json:"finalization"`
	QCResults         []QualityCheckFinalizationInput `json:"quality_check_results"`
	ProofInputs       []GlossaryArchiveReference      `json:"proof_inputs"`
	GlossarySHA       string                          `json:"glossary_sha256"`
	Status            GlossaryArchiveReference        `json:"resulting_status"`
	StatusBytes       string                          `json:"resulting_status_bytes"`
	PreviousStatusSHA string                          `json:"previous_status_sha256,omitempty"`
	Membership        []string                        `json:"unit_membership"`
	Units             []TargetSuccessorUnit           `json:"units"`
	PlanIdentity      string                          `json:"promotion_plan_identity_sha256"`
	Identity          string                          `json:"identity_sha256"`
}

type TargetSuccessorUnit struct {
	UnitID        string                   `json:"unit_id"`
	Kind          UnitKind                 `json:"unit_kind"`
	SourcePath    string                   `json:"source_path"`
	SourceSHA     string                   `json:"source_sha256"`
	BatchID       string                   `json:"selected_batch_id"`
	Candidate     GlossaryArchiveReference `json:"selected_candidate"`
	Validation    GlossaryArchiveReference `json:"validation"`
	CanonicalPath string                   `json:"canonical_path"`
	OldSHA        string                   `json:"old_target_sha256"`
	OldPresent    bool                     `json:"old_target_present"`
	NewSHA        string                   `json:"new_target_sha256"`
	Changed       bool                     `json:"changed"`
	// Native receipts retain exact old bytes for historical context rebuilding.
	// Recovery obtains these bytes from closure-bound historical candidates.
	OldBytes string `json:"old_target_bytes,omitempty"`
}

type targetSuccessorProof struct {
	evidence      *TargetSuccessorEvidence
	compatibility *PreparedGlossaryCompatibility
	oldTargets    map[string][]byte
}

func successorReference(root, path string) (GlossaryArchiveReference, error) {
	b, err := readCompatibilityFile(root, path)
	return GlossaryArchiveReference{Path: path, SHA256: sum(b)}, err
}

func successorIdentity(e TargetSuccessorEvidence) string {
	e.Identity = ""
	return sum(mustJSON(e))
}

func successorPath(e *TargetSuccessorEvidence) (string, error) {
	if err := ValidateLocaleName(e.Locale); err != nil {
		return "", err
	}
	if !validSHA256(e.Identity) {
		return "", fmt.Errorf("invalid successor identity")
	}
	switch {
	case e.Schema == HistoricalPromotionRecoverySchema && e.Kind == "historical-promotion-recovery" && e.Reason == "missing-original-promotion-receipt":
		return "data/promotion-recoveries/" + e.Locale + "/" + e.Identity + ".json", nil
	case e.Schema == NativePromotionReceiptSchema && e.Kind == "native-promotion-receipt" && e.Reason == "formal-promotion-apply" && e.HistoricalClosure == nil:
		return "locales/" + e.Locale + "/candidates/.promotion-" + e.Identity + ".json", nil
	default:
		return "", fmt.Errorf("invalid successor schema/kind/reason")
	}
}

// historicalCanonicalTargets verifies the immutable closure refs and recovers
// old canonical bytes from its bound full Snapshot. status.tsv is a mutable
// workflow projection, not historical Reviewer evidence; new evidence binds
// the exact resulting status separately. Glossary bytes come from the archive.
func historicalCanonicalTargets(root, locale string, catalog *Catalog, course ...[]byte) (GlossaryArchiveReference, map[string][]byte, error) {
	return historicalCanonicalTargetsScoped(root, locale, catalog, false, nil, course...)
}

// Recorded non-TU context is not a permanent current-filesystem oracle.
// Recovery creation still uses the original full preflight above.
func historicalCanonicalTargetsForTU(root, locale string, catalog *Catalog, frozen ...GlossaryArchiveReference) (GlossaryArchiveReference, map[string][]byte, error) {
	var ref *GlossaryArchiveReference
	if len(frozen) > 1 {
		return GlossaryArchiveReference{}, nil, fmt.Errorf("one exact historical closure required")
	}
	if len(frozen) == 1 {
		ref = &frozen[0]
	}
	return historicalCanonicalTargetsScoped(root, locale, catalog, true, ref)
}

func historicalCanonicalTargetsScoped(root, locale string, catalog *Catalog, tuOnly bool, frozen *GlossaryArchiveReference, course ...[]byte) (GlossaryArchiveReference, map[string][]byte, error) {
	path := "locales/" + locale + "/content-scope.json"
	data, err := readCompatibilityFile(root, path)
	if err != nil {
		return GlossaryArchiveReference{}, nil, err
	}
	if frozen != nil {
		if frozen.Path != path {
			return GlossaryArchiveReference{}, nil, fmt.Errorf("historical closure path mismatch")
		}
		data, err = readHistoricalContextReference(root, catalog, locale, *frozen)
		if err != nil {
			return GlossaryArchiveReference{}, nil, err
		}
	}
	var l contentidentity.Locale
	if err := contentidentity.StrictJSON(data, &l); err != nil {
		return GlossaryArchiveReference{}, nil, err
	}
	if l.Locale != locale || l.Schema != contentidentity.LocaleSchema {
		return GlossaryArchiveReference{}, nil, fmt.Errorf("historical closure locale/schema mismatch")
	}
	var closure *contentidentity.Completion
	for i := range l.Packages {
		if l.Packages[i].Package == "tour-v1" {
			if closure != nil {
				return GlossaryArchiveReference{}, nil, fmt.Errorf("duplicate Tour closure")
			}
			closure = &l.Packages[i]
		}
	}
	if closure == nil || closure.State != "complete" || closure.EvidenceKind != "legacy-tour-closure/v1" || closure.EvidenceIdentity != sum(mustJSON(closure.Evidence)) {
		return GlossaryArchiveReference{}, nil, fmt.Errorf("invalid historical closure identity")
	}
	refs := map[string]string{}
	for _, r := range closure.Evidence {
		if refs[r.Path] != "" || !validSHA256(r.SHA256) || validateGenerationInstallPath(r.Path) != nil {
			return GlossaryArchiveReference{}, nil, fmt.Errorf("invalid historical evidence membership")
		}
		refs[r.Path] = r.SHA256
	}
	old := map[string][]byte{}
	for _, r := range closure.Evidence {
		if !strings.HasSuffix(r.Path, "/manifest.json") || !strings.HasPrefix(r.Path, "data/quality-check-snapshots/"+locale+"/") {
			continue
		}
		b, err := readCompatibilityFile(root, r.Path)
		if err != nil || sum(b) != r.SHA256 {
			return GlossaryArchiveReference{}, nil, fmt.Errorf("historical Snapshot changed %s", r.Path)
		}
		var s QualityCheckSnapshotManifest
		if err := contentidentity.StrictJSON(b, &s); err != nil {
			return GlossaryArchiveReference{}, nil, err
		}
		for _, u := range s.Units {
			unit, err := catalog.Unit(u.UnitID)
			if err != nil {
				return GlossaryArchiveReference{}, nil, err
			}
			p, err := canonicalTranslationUnitCandidatePath(locale, unit)
			if err != nil {
				return GlossaryArchiveReference{}, nil, err
			}
			b, err := readCompatibilityFile(root, u.CandidatePath)
			if err != nil || sum(b) != u.CandidateSHA256 || refs[u.CandidatePath] != u.CandidateSHA256 {
				continue
			}
			if unit.Kind == UnitKindPage {
				b = canonicalizeCandidateEOF(b)
			}
			if sum(b) == refs[p] {
				old[p] = b
			}
		}
	}
	ordered, _, _, err := localeWorkflowUnitList(catalog)
	if err != nil {
		return GlossaryArchiveReference{}, nil, err
	}
	for _, u := range ordered {
		p, _ := canonicalTranslationUnitCandidatePath(locale, u)
		if old[p] == nil {
			return GlossaryArchiveReference{}, nil, fmt.Errorf("missing closure-bound old target %s", u.ID)
		}
	}
	for _, r := range closure.Evidence {
		if old[r.Path] != nil {
			continue
		}
		if r.Path == "locales/"+locale+"/status.tsv" {
			continue
		}
		if tuOnly && !strings.HasPrefix(r.Path, "data/") && r.Path != glossaryReviewGlossaryPath(locale) {
			critical := false
			for _, u := range ordered {
				if r.Path == u.SourcePath {
					critical = true
				}
			}
			if !critical {
				continue
			}
		}
		var b []byte
		if r.Path == glossaryReviewGlossaryPath(locale) {
			b, err = readArchivedGlossary(root, locale, r.SHA256)
		} else if r.Path == courseMetadataRelativePath(locale) {
			b, err = historicalCourseBytes(root, locale, r.SHA256, course...)
		} else {
			b, err = readCompatibilityFile(root, r.Path)
		}
		if err != nil || sum(b) != r.SHA256 {
			return GlossaryArchiveReference{}, nil, fmt.Errorf("historical evidence changed %s", r.Path)
		}
	}
	return GlossaryArchiveReference{path, sum(data)}, old, nil
}

func historicalSuccessorCompatibility(root, locale string, catalog *Catalog, old map[string][]byte) (*PreparedGlossaryCompatibility, error) {
	p := PrepareGlossaryCompatibility(root, locale, catalog)
	p.disableSuccessor = true
	p.historicalTargets = old
	return p, nil
}

func historicalCanonicalContexts(contexts []GlossaryCompatibilityContext, old map[string][]byte) []GlossaryCompatibilityContext {
	for i := range contexts {
		c := &contexts[i]
		if !strings.HasPrefix(c.Scope, "seo-page:") {
			continue
		}
		for j := range c.References {
			r := &c.References[j]
			if b, ok := old[r.Path]; ok {
				c.Target = string(b)
				r.SHA256 = sum(b)
			}
		}
		c.Identity = ""
		c.Identity = sum(mustJSON(*c))
	}
	return contexts
}

func verifySuccessorFinalization(root, locale, snapshotID string, catalog *Catalog, p *PreparedGlossaryCompatibility) (*QualityCheckFinalization, error) {
	// A selected package constrains membership, not the event's glossary world.
	// Re-entering the current verifier here discards p and can recursively
	// rediscover this same successor while resolving the Snapshot glossary.
	// The complete evaluation below retains the selected scope and rebuilds
	// the frozen A-only event before its caller checks current eligibility.
	path := filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, snapshotID, "finalization.json"))
	b, err := readCompatibilityFile(root, path)
	if err != nil {
		return nil, err
	}
	var f QualityCheckFinalization
	if err := contentidentity.StrictJSON(b, &f); err != nil {
		return nil, err
	}
	for _, input := range f.QCResults {
		b, err := readCompatibilityFile(root, input.Path)
		if err != nil || sum(b) != input.SHA256 {
			return nil, fmt.Errorf("finalized QC input changed %s", input.Path)
		}
		var r QualityCheckResults
		if err := contentidentity.StrictJSON(b, &r); err != nil {
			return nil, err
		}
		manifestPath := filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, input.SnapshotID, "manifest.json"))
		b, err = readCompatibilityFile(root, manifestPath)
		if err != nil {
			return nil, err
		}
		var s QualityCheckSnapshotManifest
		if err := contentidentity.StrictJSON(b, &s); err != nil {
			return nil, err
		}
	}
	t, err := time.Parse(time.RFC3339Nano, f.FinalizedAt)
	if err != nil {
		return nil, err
	}
	evaluation := newQualityCheckEvaluation(root, locale, catalog)
	evaluation.originalOnly = true
	// Rebuild the event in its immutable glossary world, never in the latest
	// glossary. The returned compatibility world is confined to TU history.
	historical := PrepareGlossaryCompatibility(root, locale, catalog)
	historical.disableSuccessor = true
	historical.tuOnly = true
	historical.historicalEvidenceRefs = map[string]string{}
	historical.historicalTargets = map[string][]byte{}
	historical.historicalBaselineTargets = p.historicalTargets
	if p.historicalBaselineTargets != nil {
		historical.historicalBaselineTargets = p.historicalBaselineTargets
	}
	seenContexts := map[string]string{}
	// Only a closure/native-bound old target may replace a canonical reference.
	// Later finalized snapshots can instead bind the exact current canonical.
	for _, u := range f.Units {
		for _, ref := range u.CompatibilityChain {
			if previous, exists := seenContexts[ref.Path]; exists {
				if previous != ref.SHA256 {
					return nil, fmt.Errorf("conflicting finalized compatibility reference")
				}
				continue
			}
			seenContexts[ref.Path] = ref.SHA256
			historical.historicalEvidenceRefs[ref.Path] = ref.SHA256
			b, err := readCompatibilityFile(root, ref.Path)
			if err != nil || sum(b) != ref.SHA256 {
				return nil, fmt.Errorf("finalized compatibility reference changed %s", ref.Path)
			}
			var edge GlossaryCompatibilityEvidence
			if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &edge); err != nil {
				return nil, err
			}
			// The strict, immutable inventory remains recorded. Its TU projection
			// is verified against source/selected-candidate bytes during prepare.
		}
	}
	historical.historicalSHA = f.GlossarySHA256
	evaluation.compatibility = historical
	want, err := buildQualityCheckFinalizationWithEvaluation(evaluation, snapshotID, func() time.Time { return t }, f.PreviousSnapshotID)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(&f, want) {
		return nil, fmt.Errorf("historical finalized A-only judgment mismatch")
	}
	*p = *historical
	return &f, nil
}

func buildTargetSuccessor(root, locale, snapshotID string, catalog *Catalog, old map[string][]byte, closure *GlossaryArchiveReference, native bool, recordedStatus []byte, course ...[]byte) (*targetSuccessorProof, error) {
	if err := ValidateLocaleName(locale); err != nil {
		return nil, err
	}
	if err := validateSnapshotID(snapshotID); err != nil {
		return nil, err
	}
	if recordedStatus == nil {
		if err := RequireCurrentGlossaryReview(root, locale); err != nil {
			return nil, err
		}
	}
	p, err := historicalSuccessorCompatibility(root, locale, catalog, old)
	if err != nil {
		return nil, err
	}
	if len(course) > 0 {
		p.historicalCourse = course[0]
	}
	f, err := verifySuccessorFinalization(root, locale, snapshotID, catalog, p)
	if err != nil {
		return nil, fmt.Errorf("successor finalization: %w", err)
	}
	s, err := readQualityCheckSnapshot(root, locale, snapshotID, false)
	if err != nil {
		return nil, err
	}
	glossary, err := readArchivedGlossary(root, locale, s.GlossarySHA256)
	if err != nil {
		return nil, err
	}
	latest, err := selectLatestRetranslationUnits(root, catalog, locale)
	if err != nil {
		return nil, err
	}
	if len(latest.ordered) != len(s.Units) {
		return nil, fmt.Errorf("successor Snapshot membership mismatch")
	}
	normalized, err := normalizeCompatibilityGlossary(locale, glossary)
	if err != nil {
		return nil, err
	}
	g := &Glossary{Locale: locale, Mandatory: map[string]string{}, Preferred: map[string]string{}}
	for _, entry := range normalized {
		switch entry.Category {
		case "mandatory":
			g.Mandatory[entry.Term] = entry.Value
		case "preferred":
			g.Preferred[entry.Term] = entry.Value
		case "keep":
			g.Keep = append(g.Keep, entry.Term)
		case "forbidden":
			g.Forbidden = append(g.Forbidden, entry.Term)
		}
	}
	e := &TargetSuccessorEvidence{Schema: HistoricalPromotionRecoverySchema, Kind: "historical-promotion-recovery", Reason: "missing-original-promotion-receipt", Locale: locale, SnapshotID: snapshotID, HistoricalClosure: closure, GlossarySHA: s.GlossarySHA256, QCResults: f.QCResults, Membership: []string{}, Units: []TargetSuccessorUnit{}}
	if native {
		e.Schema = NativePromotionReceiptSchema
		e.Kind = "native-promotion-receipt"
		e.Reason = "formal-promotion-apply"
	}
	e.Snapshot, err = successorReference(root, f.SnapshotManifestPath)
	if err != nil {
		return nil, err
	}
	e.Finalization, err = successorReference(root, filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, snapshotID, "finalization.json")))
	if err != nil {
		return nil, err
	}
	statusPath := "locales/" + locale + "/status.tsv"
	if recordedStatus == nil {
		recordedStatus, err = readCompatibilityFile(root, statusPath)
		if err != nil {
			return nil, err
		}
	}
	e.Status = GlossaryArchiveReference{statusPath, sum(recordedStatus)}
	e.StatusBytes = string(recordedStatus)
	statuses, err := readStatuses(bytes.NewReader(recordedStatus))
	if err != nil {
		return nil, err
	}
	if len(statuses) != len(s.Units) {
		return nil, fmt.Errorf("successor resulting status membership mismatch")
	}
	statusByID := map[string]Status{}
	for _, st := range statuses {
		if _, ok := statusByID[st.UnitID]; ok {
			return nil, fmt.Errorf("duplicate resulting status Unit")
		}
		statusByID[st.UnitID] = st
	}
	e.ProofInputs, err = successorProofInputs(root, catalog, locale, f, course...)
	if err != nil {
		return nil, err
	}
	for _, u := range s.Units {
		unit, err := catalog.Unit(u.UnitID)
		if err != nil {
			return nil, err
		}
		source, err := readCompatibilityFile(root, u.SourcePath)
		if err != nil {
			return nil, err
		}
		if err := validateSnapshotSourceIdentity(catalog, unit, source); err != nil {
			return nil, err
		}
		for _, path := range []string{u.CandidatePath, u.ValidationPath, "data/retranslation-runs/" + locale + "/" + u.SelectedBatchID + "/manifest.json", "data/retranslation-runs/" + locale + "/" + u.SelectedBatchID + "/result.json"} {
			if _, err := readCompatibilityFile(root, path); err != nil {
				return nil, err
			}
		}
		choice, ok := latest.selectedByID[u.UnitID]
		if !ok || choice.batchID != u.SelectedBatchID {
			return nil, fmt.Errorf("selected candidate no longer current: %s", u.UnitID)
		}
		v, err := validateSelectedRetranslationCandidate(root, catalog, locale, g, unit, choice, p)
		if err != nil {
			return nil, err
		}
		if sum(v.candidate) != u.CandidateSHA256 || sum(v.validationData) != u.ValidationSHA256 {
			return nil, fmt.Errorf("selected candidate/validation mismatch %s", u.UnitID)
		}
		canonical := v.candidate
		if unit.Kind == UnitKindPage {
			canonical = canonicalizeCandidateEOF(canonical)
		}
		path, err := canonicalTranslationUnitCandidatePath(locale, unit)
		if err != nil {
			return nil, err
		}
		current, err := readCompatibilityFile(root, path)
		if err != nil || !bytes.Equal(current, canonical) {
			return nil, fmt.Errorf("canonical differs from finalized selected candidate: %s", u.UnitID)
		}
		st, ok := statusByID[u.UnitID]
		if !ok || st.State != "ready" || st.Attempts != v.attempt || st.SourceSHA256 != u.SourceSHA256 || st.CandidatePath != path {
			return nil, fmt.Errorf("successor resulting status does not match finalized candidate: %s", u.UnitID)
		}
		if native {
			if st.Note != fmt.Sprintf("retranslation promoted from %s; passed canonical validator", u.SelectedBatchID) {
				return nil, fmt.Errorf("native resulting status provenance mismatch: %s", u.UnitID)
			}
			if _, err := time.Parse(time.RFC3339, st.UpdatedAt); err != nil {
				return nil, fmt.Errorf("native resulting status time invalid: %s", u.UnitID)
			}
		}
		prior, ok := old[path]
		if !ok {
			return nil, fmt.Errorf("missing old canonical identity %s", u.UnitID)
		}
		e.Membership = append(e.Membership, u.UnitID)
		changed := !bytes.Equal(prior, canonical)
		if !changed && !native {
			continue
		}
		x := TargetSuccessorUnit{UnitID: u.UnitID, Kind: u.UnitKind, SourcePath: u.SourcePath, SourceSHA: u.SourceSHA256, BatchID: u.SelectedBatchID, Candidate: GlossaryArchiveReference{u.CandidatePath, u.CandidateSHA256}, Validation: GlossaryArchiveReference{u.ValidationPath, u.ValidationSHA256}, CanonicalPath: path, OldSHA: sum(prior), OldPresent: true, NewSHA: sum(canonical), Changed: changed}
		if native {
			x.OldBytes = string(prior)
			if len(prior) == 0 {
				x.OldSHA = ""
				x.OldPresent = false
			}
		}
		e.Units = append(e.Units, x)
	}
	e.PlanIdentity = sum(mustJSON(e.Units))
	e.Identity = successorIdentity(*e)
	return &targetSuccessorProof{e, p, old}, nil
}

// RecoverHistoricalPromotion is explicitly invoked by the maintainer. Dry-run
// builds the same deterministic proof as apply, but creates no artifact.
func RecoverHistoricalPromotion(root string, catalog *Catalog, locale, snapshotID string, apply bool) (*TargetSuccessorEvidence, string, error) {
	if err := ValidateLocaleName(locale); err != nil {
		return nil, "", err
	}
	if err := validateSnapshotID(snapshotID); err != nil {
		return nil, "", err
	}
	if err := CheckPromotionTransactionState(root, locale); err != nil {
		return nil, "", err
	}
	if err := requireMissingNativePromotionReceipt(root, locale, snapshotID); err != nil {
		return nil, "", err
	}
	ref, old, err := historicalCanonicalTargets(root, locale, catalog)
	if err != nil {
		return nil, "", err
	}
	proof, err := buildTargetSuccessor(root, locale, snapshotID, catalog, old, &ref, false, nil)
	if err != nil {
		return nil, "", err
	}
	e := proof.evidence
	path, err := successorPath(e)
	if err != nil {
		return nil, "", err
	}
	if apply {
		p, err := compatibilityPath(root, path, true)
		if err != nil {
			return nil, "", err
		}
		b, _ := json.MarshalIndent(e, "", "  ")
		if err := writeNewGenerationFile(p, append(b, '\n')); err != nil {
			return nil, "", err
		}
	}
	return e, path, nil
}

// A finalized Snapshot is applied once. A missing historical event cannot be
// recovered if an original receipt already exists, nor re-applied to manufacture
// a second contradictory old→new event for the same Snapshot.
func requireMissingNativePromotionReceipt(root, locale, snapshotID string) error {
	dir, err := compatibilityPath(root, "locales/"+locale+"/candidates/.inventory", false)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Dir(dir))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".promotion-") {
			continue
		}
		path := "locales/" + locale + "/candidates/" + entry.Name()
		b, err := readCompatibilityFile(root, path)
		if err != nil {
			return err
		}
		var e TargetSuccessorEvidence
		if err := contentidentity.StrictJSON(b, &e); err != nil {
			return err
		}
		want, err := successorPath(&e)
		if err != nil || want != path || e.Identity != successorIdentity(e) {
			return fmt.Errorf("invalid native receipt identity")
		}
		if e.SnapshotID == snapshotID {
			return fmt.Errorf("original native promotion receipt already exists for %s", snapshotID)
		}
	}
	return nil
}

func verifyTargetSuccessor(root, path string, catalog *Catalog, course ...[]byte) (*targetSuccessorProof, error) {
	b, err := readCompatibilityFile(root, path)
	if err != nil {
		return nil, err
	}
	var e TargetSuccessorEvidence
	if err := contentidentity.StrictJSON(b, &e); err != nil {
		return nil, err
	}
	encoded := mustJSON(&e)
	if e.Schema == HistoricalPromotionRecoverySchema {
		encoded, _ = json.MarshalIndent(&e, "", "  ")
	}
	if !bytes.Equal(b, append(encoded, '\n')) {
		return nil, fmt.Errorf("successor immutable encoding changed")
	}
	wantPath, err := successorPath(&e)
	if err != nil || wantPath != path || e.Identity != successorIdentity(e) {
		return nil, fmt.Errorf("successor identity/path mismatch")
	}
	var closure *GlossaryArchiveReference
	old := map[string][]byte{}
	if e.Schema == HistoricalPromotionRecoverySchema {
		if e.HistoricalClosure == nil {
			return nil, fmt.Errorf("historical closure reference mismatch")
		}
		ref, targets, err := historicalCanonicalTargetsForTU(root, e.Locale, catalog, *e.HistoricalClosure)
		if err != nil {
			return nil, err
		}
		if e.HistoricalClosure == nil || *e.HistoricalClosure != ref {
			return nil, fmt.Errorf("historical closure reference mismatch")
		}
		closure = &ref
		old = targets
	} else {
		for _, u := range e.Units {
			if (u.OldPresent && sum([]byte(u.OldBytes)) != u.OldSHA) || (!u.OldPresent && (u.OldSHA != "" || u.OldBytes != "")) || old[u.CanonicalPath] != nil {
				return nil, fmt.Errorf("native old target identity mismatch")
			}
			old[u.CanonicalPath] = []byte(u.OldBytes)
		}
	}
	if err := CheckPromotionTransactionState(root, e.Locale); err != nil {
		return nil, err
	}
	proof, err := buildTargetSuccessor(root, e.Locale, e.SnapshotID, catalog, old, closure, e.Schema == NativePromotionReceiptSchema, []byte(e.StatusBytes), course...)
	if err != nil {
		return nil, err
	}
	if e.Status.Path != "locales/"+e.Locale+"/status.tsv" || sum([]byte(e.StatusBytes)) != e.Status.SHA256 {
		return nil, fmt.Errorf("successor resulting status identity mismatch")
	}
	// Status is a mutable projection. Its immutable recorded bytes prove the
	// transaction result; later status changes do not authorize target changes.
	proof.evidence.Status, proof.evidence.StatusBytes = e.Status, e.StatusBytes
	if e.Schema == NativePromotionReceiptSchema {
		if !validSHA256(e.PreviousStatusSHA) {
			return nil, fmt.Errorf("native previous status identity missing")
		}
		proof.evidence.PreviousStatusSHA = e.PreviousStatusSHA
	}
	proof.evidence.Identity = successorIdentity(*proof.evidence)
	if !reflect.DeepEqual(&e, proof.evidence) {
		return nil, fmt.Errorf("successor exact-set/proof mismatch")
	}
	return proof, nil
}

func findCurrentTargetSuccessor(root, locale string, catalog *Catalog, course ...[]byte) (*targetSuccessorProof, error) {
	if err := CheckPromotionTransactionState(root, locale); err != nil {
		return nil, err
	}
	var found *targetSuccessorProof
	for _, directory := range []string{"data/promotion-recoveries/" + locale, "locales/" + locale + "/candidates"} {
		p, err := compatibilityPath(root, directory+"/.inventory", false)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(filepath.Dir(p))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if directory == "locales/"+locale+"/candidates" && !strings.HasPrefix(entry.Name(), ".promotion-") {
				continue
			}
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				return nil, fmt.Errorf("unknown successor artifact")
			}
			path := directory + "/" + entry.Name()
			b, err := readCompatibilityFile(root, path)
			if err != nil {
				return nil, err
			}
			var e TargetSuccessorEvidence
			if err := contentidentity.StrictJSON(b, &e); err != nil {
				return nil, err
			}
			want, err := successorPath(&e)
			if err != nil || want != path || e.Locale != locale || e.Identity != successorIdentity(e) {
				return nil, fmt.Errorf("invalid successor artifact identity")
			}
			current := true
			// Recovery receipts record changed transitions only. Currentness
			// belongs to their full finalized membership, including unchanged
			// targets that a later legitimate promotion may have replaced.
			refs, err := successorSnapshotCanonicalReferences(root, catalog, &e)
			if err != nil {
				return nil, err
			}
			for _, ref := range refs {
				b, err := readCompatibilityFile(root, ref.Path)
				if err != nil {
					return nil, err
				}
				if sum(b) != ref.SHA256 {
					current = false
				}
			}
			// Prior artifacts remain immutable history. They are never selected
			// merely because current bytes differ from their resulting targets.
			if !current {
				continue
			}
			proof, err := verifyTargetSuccessor(root, directory+"/"+entry.Name(), catalog, course...)
			if err != nil {
				return nil, err
			}
			if found != nil {
				return nil, fmt.Errorf("ambiguous current successor authority")
			}
			found = proof
		}
	}
	return found, nil
}

func successorSnapshotCanonicalReferences(root string, catalog *Catalog, e *TargetSuccessorEvidence) ([]GlossaryArchiveReference, error) {
	path := filepath.ToSlash(qualityCheckSnapshotManifestPath("", e.Locale, e.SnapshotID))
	b, err := readCompatibilityFile(root, path)
	if err != nil || e.Snapshot.Path != path || sum(b) != e.Snapshot.SHA256 {
		return nil, fmt.Errorf("successor Snapshot reference changed")
	}
	s, err := readQualityCheckSnapshot(root, e.Locale, e.SnapshotID, false)
	if err != nil {
		return nil, err
	}
	if len(e.Membership) != len(s.Units) {
		return nil, fmt.Errorf("successor Snapshot membership mismatch")
	}
	refs := make([]GlossaryArchiveReference, 0, len(s.Units))
	for i, u := range s.Units {
		unit, err := catalog.Unit(u.UnitID)
		if err != nil || e.Membership[i] != u.UnitID || unit.Kind != u.UnitKind {
			return nil, fmt.Errorf("successor Snapshot membership mismatch")
		}
		candidate, err := readCompatibilityFile(root, u.CandidatePath)
		if err != nil || sum(candidate) != u.CandidateSHA256 {
			return nil, fmt.Errorf("successor selected candidate changed: %s", u.UnitID)
		}
		if unit.Kind == UnitKindPage {
			candidate = canonicalizeCandidateEOF(candidate)
		}
		canonical, err := canonicalTranslationUnitCandidatePath(e.Locale, unit)
		if err != nil {
			return nil, err
		}
		refs = append(refs, GlossaryArchiveReference{Path: canonical, SHA256: sum(candidate)})
	}
	return refs, nil
}

// CheckPromotionTransactionState rejects any pending/unrecovered staging,
// including older randomly named staging directories. No command blindly
// clears or replays these; exact backups must be inspected first.
func CheckPromotionTransactionState(root, locale string) error {
	if err := ValidateLocaleName(locale); err != nil {
		return err
	}
	path, err := compatibilityPath(root, "locales/"+locale+"/.inventory", false)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".promotion-staging-") {
			return fmt.Errorf("promotion transaction pending/unrecovered; retained locales/%s/%s; inspect backups before further writes", locale, entry.Name())
		}
	}
	return nil
}

func successorCanonicalReferences(e *TargetSuccessorEvidence) []GlossaryArchiveReference {
	refs := []GlossaryArchiveReference{}
	for _, u := range e.Units {
		refs = append(refs, GlossaryArchiveReference{u.CanonicalPath, u.NewSHA})
	}
	return refs
}

func successorProofInputs(root string, catalog *Catalog, locale string, f *QualityCheckFinalization, course ...[]byte) ([]GlossaryArchiveReference, error) {
	paths := map[string]bool{}
	critical := map[string]bool{}
	for _, u := range f.Units {
		critical[u.Snapshot.SourcePath] = true
	}
	for prior := f.PredecessorFinalization; prior != nil; {
		if paths[prior.Path] {
			return nil, fmt.Errorf("finalization predecessor cycle")
		}
		b, err := readCompatibilityFile(root, prior.Path)
		if err != nil || sum(b) != prior.SHA256 {
			return nil, fmt.Errorf("predecessor finalization reference changed")
		}
		var predecessor QualityCheckFinalization
		if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &predecessor); err != nil {
			return nil, err
		}
		paths[prior.Path] = true
		prior = predecessor.PredecessorFinalization
	}
	canonicalPaths := map[string]bool{}
	seenEdges := map[string]string{}
	seenDependencies := map[string]string{}
	for _, fu := range f.Units {
		u, err := catalog.Unit(fu.UnitID)
		if err != nil {
			return nil, err
		}
		p, err := canonicalTranslationUnitCandidatePath(locale, u)
		if err != nil {
			return nil, err
		}
		canonicalPaths[p] = true
	}
	for _, q := range f.QCResults {
		paths[q.Path] = true
		paths[filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, q.SnapshotID, "manifest.json"))] = true
	}
	for _, fu := range f.Units {
		for _, ref := range fu.CompatibilityChain {
			if prior, ok := seenEdges[ref.Path]; ok {
				if prior != ref.SHA256 {
					return nil, fmt.Errorf("conflicting finalized compatibility reference %s", ref.Path)
				}
				continue
			}
			seenEdges[ref.Path] = ref.SHA256
			b, err := readCompatibilityFile(root, ref.Path)
			if err != nil || sum(b) != ref.SHA256 {
				return nil, fmt.Errorf("finalized compatibility reference changed %s", ref.Path)
			}
			paths[ref.Path] = true
			var edge GlossaryCompatibilityEvidence
			// Legacy semantic deltas legitimately encode old/new=null. Keep
			// their existing strict compatibility contract, not receipt rules.
			if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &edge); err != nil {
				return nil, err
			}
			deps := []GlossaryArchiveReference{edge.OldGlossary, edge.NewGlossary, edge.NewFullReview}
			for _, context := range edge.Contexts {
				deps = append(deps, context.References...)
			}
			for _, dep := range deps {
				// Canonical refs are verified against exact old bytes by the
				// historical context proof, and against new bytes at installation.
				// Status is separately retained as a mutable historical projection.
				if canonicalPaths[dep.Path] || dep.Path == "locales/"+locale+"/status.tsv" {
					continue
				}
				if prior, ok := seenDependencies[dep.Path]; ok {
					if prior != dep.SHA256 {
						return nil, fmt.Errorf("conflicting finalized compatibility dependency %s", dep.Path)
					}
					continue
				}
				seenDependencies[dep.Path] = dep.SHA256
				if !critical[dep.Path] && !strings.HasPrefix(dep.Path, "data/") {
					paths[dep.Path] = true
					continue
				}
				b, err := readCompatibilityFile(root, dep.Path)
				if dep.Path == courseMetadataRelativePath(locale) {
					b, err = historicalCourseBytes(root, locale, dep.SHA256, course...)
				}
				if err != nil || sum(b) != dep.SHA256 {
					return nil, fmt.Errorf("finalized compatibility dependency changed %s", dep.Path)
				}
				paths[dep.Path] = true
			}
		}
		u := fu.Snapshot
		for _, p := range []string{u.SourcePath, u.CandidatePath, u.ValidationPath} {
			if _, err := readCompatibilityFile(root, p); err != nil {
				return nil, err
			}
			paths[p] = true
		}
		base := "data/retranslation-runs/" + locale + "/" + u.SelectedBatchID + "/"
		for _, p := range []string{base + "manifest.json", base + "result.json"} {
			if _, err := readCompatibilityFile(root, p); err != nil {
				return nil, err
			}
			paths[p] = true
		}
		v, err := readSnapshotUnitRepositoryEvidence(root, catalog, locale, u)
		if err != nil {
			return nil, err
		}
		paths[base+v.record.InputPath] = true
		paths[base+v.validation.RawResponsePath] = true
	}
	keys := []string{}
	for p := range paths {
		keys = append(keys, p)
	}
	sort.Strings(keys)
	refs := []GlossaryArchiveReference{}
	for _, p := range keys {
		if sha, recorded := seenDependencies[p]; recorded && !critical[p] && !strings.HasPrefix(p, "data/") {
			refs = append(refs, GlossaryArchiveReference{p, sha})
			continue
		}
		r, err := successorReference(root, p)
		if p == courseMetadataRelativePath(locale) {
			sha := seenDependencies[p]
			_, err = historicalCourseBytes(root, locale, sha, course...)
			r = GlossaryArchiveReference{p, sha}
		}
		if err != nil {
			return nil, err
		}
		refs = append(refs, r)
	}
	return refs, nil
}

type QualityCheckAuthorizationDiagnostic struct {
	Locale              string `json:"locale"`
	SnapshotID          string `json:"snapshot_id"`
	ReviewerACount      int    `json:"historical_reviewer_a_count"`
	UnitCount           int    `json:"unit_count"`
	CanonicalAuthorized bool   `json:"current_canonical_authorized"`
	SuccessorKind       string `json:"successor_kind,omitempty"`
	Reason              string `json:"reason"`
}

// DiagnoseQualityCheckAuthorization reports historical language judgment and
// current canonical authorization separately. It never creates authority.
func DiagnoseQualityCheckAuthorization(root string, catalog *Catalog, locale, snapshotID string) (*QualityCheckAuthorizationDiagnostic, error) {
	// Use the same sealed authority as the canonical consumer, scoped to the
	// requested Snapshot. A verified checkpoint must not fall back to legacy
	// promotion discovery; a language-only epoch still cannot authorize install.
	if a, err := FindCurrentTranslationAuthorization(root, catalog, locale, snapshotID); err != nil {
		return nil, err
	} else if a != nil {
		d := &QualityCheckAuthorizationDiagnostic{Locale: locale, SnapshotID: snapshotID, UnitCount: len(a.Units), ReviewerACount: len(a.Historical.Finalization.Units), SuccessorKind: a.Schema, Reason: "current checkpoint lacks canonical installation binding"}
		if len(a.Canonical) > 0 {
			d.CanonicalAuthorized = true
			d.Reason = "verified-current-canonical-checkpoint"
		}
		return d, nil
	}
	proof, err := findCurrentTargetSuccessor(root, locale, catalog)
	if err != nil {
		return nil, err
	}
	if proof != nil && proof.evidence.SnapshotID == snapshotID {
		d := &QualityCheckAuthorizationDiagnostic{Locale: locale, SnapshotID: snapshotID, UnitCount: len(proof.evidence.Membership), ReviewerACount: len(proof.evidence.Membership), CanonicalAuthorized: true, SuccessorKind: proof.evidence.Kind, Reason: "verified-successor-authority"}
		if err := requireSuccessorCurrentLanguage(root, catalog, proof); err != nil {
			d.CanonicalAuthorized = false
			d.Reason = err.Error()
		}
		return d, nil
	}
	// A later glossary-bound Snapshot need not be the installation Snapshot.
	// Retain the verified event's TU baseline; full closure Surface refs are
	// recorded context, not current canonical authorization inputs.
	var old map[string][]byte
	if proof != nil {
		old = proof.oldTargets
	} else {
		_, old, err = historicalCanonicalTargetsForTU(root, locale, catalog)
		if err != nil {
			return nil, err
		}
	}
	p, err := historicalSuccessorCompatibility(root, locale, catalog, old)
	if err != nil {
		return nil, err
	}
	f, err := verifySuccessorFinalization(root, locale, snapshotID, catalog, p)
	if err != nil {
		return nil, err
	}
	d := &QualityCheckAuthorizationDiagnostic{Locale: locale, SnapshotID: snapshotID, UnitCount: len(f.Units), ReviewerACount: len(f.Units), Reason: "missing-successor-authority"}
	if proof != nil {
		original, err := readQualityCheckSnapshot(root, locale, proof.evidence.SnapshotID, false)
		if err != nil {
			return nil, err
		}
		if len(f.Units) != len(original.Units) {
			d.Reason = "successor Snapshot membership mismatch"
			return d, nil
		}
		for i, u := range f.Units {
			if !qualityCheckSnapshotIdentityMatches(u.Snapshot, original.Units[i]) {
				d.Reason = "successor Snapshot identity mismatch: " + u.UnitID
				return d, nil
			}
		}
		d.SuccessorKind = proof.evidence.Kind
		if err := requireSuccessorCurrentLanguage(root, catalog, proof); err != nil {
			d.Reason = err.Error()
			return d, nil
		}
		d.CanonicalAuthorized = true
		d.Reason = "verified-successor-authority"
	}
	return d, nil
}

// CanonicalTranslationUnitTargetPaths derives the exact installation scope
// from the selected formal catalog. It never includes Surface assets.
func CanonicalTranslationUnitTargetPaths(catalog *Catalog, locale string) (map[string]bool, error) {
	if err := ValidateLocaleName(locale); err != nil {
		return nil, err
	}
	units, _, _, err := localeWorkflowUnitList(catalog)
	if err != nil {
		return nil, err
	}
	paths := map[string]bool{}
	for _, unit := range units {
		path, err := canonicalTranslationUnitCandidatePath(locale, unit)
		if err != nil {
			return nil, err
		}
		if paths[path] {
			return nil, fmt.Errorf("canonical TU projection collision: %s", path)
		}
		paths[path] = true
	}
	return paths, nil
}

// A Snapshot has no package field. Only a complete registered descriptor or
// native workflow with disjoint Unit/source identities proves a foreign scope.
// Partial, unknown or mixed inventories must reach the strict lineage checks.
func unrelatedSuccessorSnapshot(s *QualityCheckSnapshotManifest, expected []*TranslationUnit, catalog *Catalog) bool {
	for _, candidate := range s.Units {
		for _, unit := range expected {
			if candidate.UnitID == unit.ID || candidate.SourcePath == unit.SourcePath {
				return false
			}
		}
	}
	if snapshotMatchesNativeWorkflow(s, catalog) {
		return true
	}
	for _, descriptor := range append(append([]TranslationPackageDescriptor{}, catalog.TranslationPackages...), catalog.HistoricalTranslationPackages...) {
		if snapshotMatchesTranslationDescriptor(s, descriptor, true) {
			return true
		}
	}
	return false
}

// Source coordinates are only a prefilter when exactIDs is false. Historical
// IDs gain authority exclusively from the verified successor/replacement chain.
func snapshotMatchesTranslationDescriptor(s *QualityCheckSnapshotManifest, descriptor TranslationPackageDescriptor, exactIDs bool) bool {
	if !validSHA256(descriptor.AuthoritySHA256) || len(descriptor.Units) != len(s.Units) {
		return false
	}
	for i, registered := range descriptor.Units {
		u, candidate := registered.Unit, s.Units[i]
		if (exactIDs && candidate.UnitID != u.ID) || candidate.UnitKind != u.Kind ||
			candidate.SourcePath != u.SourcePath || candidate.SourceSHA256 != u.SourceSHA256 || candidate.PageSection != nil {
			return false
		}
	}
	return true
}

func excludeUnrelatedSuccessorSnapshot(root string, s *QualityCheckSnapshotManifest, expected []*TranslationUnit, catalog *Catalog, proven map[string]bool) (bool, error) {
	if !unrelatedSuccessorSnapshot(s, expected, catalog) && !proven[s.SnapshotID] {
		foreign, err := proveHistoricalForeignSnapshot(root, s, expected, catalog, proven)
		if err != nil || !foreign {
			return false, err
		}
	}
	_, err := boundSuccessorSnapshotFinalization(root, s)
	return err == nil, err
}

func boundSuccessorSnapshotFinalization(root string, s *QualityCheckSnapshotManifest) (*QualityCheckFinalization, error) {
	// A rewritten same-lineage manifest must not disguise itself as a foreign
	// descriptor. Cross-check the immutable finalization's bound membership.
	path := filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", s.Locale, s.SnapshotID, "manifest.json"))
	manifest, err := readCompatibilityFile(root, path)
	if err != nil {
		return nil, err
	}
	b, err := readCompatibilityFile(root, filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", s.Locale, s.SnapshotID, "finalization.json")))
	if err != nil {
		return nil, err
	}
	var f QualityCheckFinalization
	if err := contentidentity.StrictJSON(b, &f); err != nil {
		return nil, err
	}
	if f.SchemaVersion != QualityCheckFinalizationSchemaVersion || f.EvidenceType != QualityCheckFinalizationEvidenceType ||
		f.Locale != s.Locale || f.SnapshotID != s.SnapshotID || f.SnapshotManifestPath != path ||
		f.SnapshotManifestSHA256 != sum(manifest) || len(f.Units) != len(s.Units) {
		return nil, fmt.Errorf("unrelated successor Snapshot finalization binding mismatch: %s", s.SnapshotID)
	}
	for i, u := range f.Units {
		if u.UnitID != s.Units[i].UnitID || !reflect.DeepEqual(u.Snapshot, s.Units[i]) {
			return nil, fmt.Errorf("unrelated successor Snapshot finalized membership mismatch: %s", s.SnapshotID)
		}
	}
	return &f, nil
}

// Check every current-glossary manifest's immutable membership before package
// classification. A rewritten same-lineage Snapshot cannot hide behind a
// foreign inventory, regardless of discovery order.
func boundCurrentGlossaryFinalizedSnapshots(root, locale, glossarySHA string) ([]*QualityCheckSnapshotManifest, error) {
	ids, err := qualityCheckFinalizedSnapshotIDs(root, locale)
	if err != nil {
		return nil, err
	}
	snapshots := []*QualityCheckSnapshotManifest{}
	for _, id := range ids {
		s, err := readQualityCheckSnapshot(root, locale, id, false)
		if err != nil {
			return nil, err
		}
		if s.GlossarySHA256 != glossarySHA {
			continue
		}
		if _, err := boundSuccessorSnapshotFinalization(root, s); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, s)
	}
	return snapshots, nil
}

// A historical foreign Snapshot must be an immutable finalized ancestor of an
// exact registered foreign descriptor. Full shared verification proves QC and
// any replacement authorization; resolving each edge supplies its historical
// inventory. No old IDs are added to the caller's scheduling catalog.
func proveHistoricalForeignSnapshot(root string, s *QualityCheckSnapshotManifest, expected []*TranslationUnit, catalog *Catalog, proven map[string]bool) (bool, error) {
	for _, candidate := range s.Units {
		for _, unit := range expected {
			if candidate.UnitID == unit.ID || candidate.SourcePath == unit.SourcePath {
				return false, nil
			}
		}
	}
	var descriptors []TranslationPackageDescriptor
	for _, d := range catalog.TranslationPackages {
		if snapshotMatchesTranslationDescriptor(s, d, false) {
			descriptors = append(descriptors, d)
		}
	}
	if len(descriptors) == 0 {
		return false, nil
	}
	if len(descriptors) != 1 {
		return false, fmt.Errorf("ambiguous historical foreign Snapshot descriptor: %s", s.SnapshotID)
	}
	descriptor := descriptors[0]
	ids, err := qualityCheckFinalizedSnapshotIDs(root, s.Locale)
	if err != nil {
		return false, err
	}
	for _, id := range ids {
		endpoint, err := readQualityCheckSnapshot(root, s.Locale, id, false)
		if err != nil {
			return false, err
		}
		if !snapshotMatchesTranslationDescriptor(endpoint, descriptor, true) || !unrelatedSuccessorSnapshot(endpoint, expected, catalog) {
			continue
		}
		// Follow only explicit hash-bound finalization references before doing
		// expensive full verification. A path match alone never excludes s.
		chain := []*QualityCheckSnapshotManifest{endpoint}
		seen := map[string]bool{}
		for child := endpoint; child.SnapshotID != s.SnapshotID; {
			if seen[child.SnapshotID] {
				return false, fmt.Errorf("cyclic foreign finalization predecessor")
			}
			seen[child.SnapshotID] = true
			f, err := boundSuccessorSnapshotFinalization(root, child)
			if err != nil {
				return false, err
			}
			if f.PredecessorFinalization == nil || f.PreviousSnapshotID == "" {
				break
			}
			ref := f.PredecessorFinalization
			want := filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", s.Locale, f.PreviousSnapshotID, "finalization.json"))
			b, err := readCompatibilityFile(root, ref.Path)
			if err != nil || ref.Path != want || sum(b) != ref.SHA256 {
				return false, fmt.Errorf("foreign predecessor finalization binding changed: %s", child.SnapshotID)
			}
			child, err = readQualityCheckSnapshot(root, s.Locale, f.PreviousSnapshotID, false)
			if err != nil {
				return false, err
			}
			chain = append(chain, child)
		}
		foreign := *catalog
		if err := foreign.UseTranslationPackage(descriptor.ID); err != nil {
			return false, err
		}
		if chain[len(chain)-1].SnapshotID != s.SnapshotID {
			historical, err := foreignSnapshotMachineCatalog(root, &foreign, endpoint, s, descriptor)
			if err != nil {
				return false, fmt.Errorf("foreign machine Snapshot scope: %w", err)
			}
			if historical == nil {
				continue
			}
			// This proves foreign scope, not current language eligibility for
			// another package. The shared historical reader verifies original A
			// and every endpoint machine receipt/candidate/validation against the
			// exact current descriptor. The caller's own scope still goes through
			// all ordinary current language/canonical checks below.
			if _, err := VerifyHistoricalQualityCheckFinalization(root, s.Locale, endpoint.SnapshotID, &foreign); err != nil {
				return false, fmt.Errorf("foreign machine successor finalization: %w", err)
			}
			// The frozen catalog above proves event-time membership only. This
			// verifier's optional catalog supplies today's representation-owned
			// readers, which independently resolve frozen Surface events and
			// formally verified current successors.
			if _, err := VerifyHistoricalQualityCheckFinalization(root, s.Locale, s.SnapshotID, &foreign); err != nil {
				return false, fmt.Errorf("foreign machine predecessor finalization: %w", err)
			}
			proven[s.SnapshotID], proven[endpoint.SnapshotID] = true, true
			return true, nil
		}
		if _, err := VerifyQualityCheckFinalization(root, &foreign, s.Locale, endpoint.SnapshotID); err != nil {
			return false, fmt.Errorf("foreign successor finalization: %w", err)
		}
		historical := &foreign
		for i := 1; i < len(chain); i++ {
			historical, err = replacementCatalogForPredecessor(root, historical, s.Locale, chain[i].SnapshotID, chain[i-1])
			if err != nil {
				return false, err
			}
			if !unrelatedSuccessorSnapshot(chain[i], expected, historical) {
				return false, fmt.Errorf("historical foreign Snapshot lacks exact replacement inventory: %s", chain[i].SnapshotID)
			}
		}
		for _, ancestor := range chain {
			proven[ancestor.SnapshotID] = true
		}
		return true, nil
	}
	return false, nil
}

// Event authority is historical. After a glossary refresh, current language
// authorization additionally needs an exact current finalized Snapshot.
func requireSuccessorCurrentLanguage(root string, catalog *Catalog, proof *targetSuccessorProof) error {
	e := proof.evidence
	current, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(e.Locale))
	if err != nil {
		return err
	}
	if err := RequireCurrentGlossaryReview(root, e.Locale); err != nil {
		return err
	}
	if sum(current) == e.GlossarySHA {
		return nil
	}
	ordered, _, _, err := localeWorkflowUnitList(catalog)
	if err != nil {
		return err
	}
	snapshots, err := boundCurrentGlossaryFinalizedSnapshots(root, e.Locale, sum(current))
	if err != nil {
		return err
	}
	provenForeign := map[string]bool{}
	for _, s := range snapshots {
		id := s.SnapshotID
		exclude, err := excludeUnrelatedSuccessorSnapshot(root, s, ordered, catalog, provenForeign)
		if err != nil {
			return err
		}
		if exclude {
			continue
		}
		if len(s.Units) != len(e.Membership) {
			return fmt.Errorf("current successor Snapshot membership mismatch")
		}
		original, err := readQualityCheckSnapshot(root, e.Locale, e.SnapshotID, false)
		if err != nil {
			return err
		}
		for i, u := range s.Units {
			if !qualityCheckSnapshotIdentityMatches(u, original.Units[i]) {
				return fmt.Errorf("current successor Snapshot identity mismatch: %s", u.UnitID)
			}
		}
		// Rebuild the immutable TU judgment in its original glossary world;
		// derived whole-file currentness is not part of this event.
		p, _ := historicalSuccessorCompatibility(root, e.Locale, catalog, proof.oldTargets)
		if _, err := verifySuccessorFinalization(root, e.Locale, id, catalog, p); err != nil {
			return fmt.Errorf("current successor finalization: %w", err)
		}
		for _, unitID := range e.Membership {
			r := p.Resolve(e.GlossarySHA, sum(current), "tu:"+unitID)
			if r.Status != "exact" && r.Status != "compatible" {
				return fmt.Errorf("current TU glossary authorization %s: %s (%s)", unitID, r.Status, r.Reason)
			}
		}
		return nil
	}
	return fmt.Errorf("missing finalized Snapshot for current glossary authorization")
}

// RequireCurrentCanonicalQualityCheck composes current finalized TU language
// authority with independently authorized canonical bytes. It records nothing
// and never grants authority to derived consumers.
func RequireCurrentCanonicalQualityCheck(root, locale string, catalog *Catalog) (*QualityCheckFinalization, []GlossaryArchiveReference, error) {
	if a, err := findScopeCurrentAuthorization(root, catalog, locale); err != nil {
		return nil, nil, err
	} else if a != nil {
		if len(a.Canonical) == 0 {
			return nil, nil, fmt.Errorf("current checkpoint lacks canonical installation binding")
		}
		ref, err := successorReference(root, CurrentTranslationAuthorizationPath(a))
		if err != nil {
			return nil, nil, err
		}
		return &a.Historical.Finalization, append([]GlossaryArchiveReference{ref}, a.Canonical...), nil
	}
	proof, err := findCurrentTargetSuccessor(root, locale, catalog)
	if err != nil {
		return nil, nil, err
	}
	refs := []GlossaryArchiveReference{}
	var old map[string][]byte
	if proof != nil {
		// Current finalization/per-TU compatibility below independently proves
		// language authorization; do not rebuild the same gate twice.
		path, _ := successorPath(proof.evidence)
		ref, err := successorReference(root, path)
		if err != nil {
			return nil, nil, err
		}
		refs = append(refs, ref)
		old = proof.oldTargets
	} else {
		_, old, err = historicalCanonicalTargetsForTU(root, locale, catalog)
		if err != nil {
			return nil, nil, err
		}
		for path, bytes := range old {
			current, err := readCompatibilityFile(root, path)
			if err != nil || sum(current) != sum(bytes) {
				return nil, nil, fmt.Errorf("canonical target lacks successor authority: %s", path)
			}
		}
	}
	glossary, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(locale))
	if err != nil {
		return nil, nil, err
	}
	if err := RequireCurrentGlossaryReview(root, locale); err != nil {
		return nil, nil, err
	}
	snapshots, err := boundCurrentGlossaryFinalizedSnapshots(root, locale, sum(glossary))
	if err != nil {
		return nil, nil, err
	}
	provenForeign := map[string]bool{}
	for _, s := range snapshots {
		id := s.SnapshotID
		ordered, _, _, err := localeWorkflowUnitList(catalog)
		if err != nil {
			return nil, nil, err
		}
		exclude, err := excludeUnrelatedSuccessorSnapshot(root, s, ordered, catalog, provenForeign)
		if err != nil {
			return nil, nil, err
		}
		if exclude {
			continue
		}
		if len(s.Units) != len(ordered) {
			return nil, nil, fmt.Errorf("current finalized TU exact-set mismatch")
		}
		for i, u := range s.Units {
			if u.UnitID != ordered[i].ID {
				return nil, nil, fmt.Errorf("current finalized TU membership/order mismatch")
			}
		}
		if proof != nil {
			original, err := readQualityCheckSnapshot(root, locale, proof.evidence.SnapshotID, false)
			if err != nil {
				return nil, nil, err
			}
			if len(s.Units) != len(original.Units) {
				return nil, nil, fmt.Errorf("current successor Snapshot membership mismatch")
			}
			for i, u := range s.Units {
				if !qualityCheckSnapshotIdentityMatches(u, original.Units[i]) {
					return nil, nil, fmt.Errorf("current successor Snapshot identity mismatch: %s", u.UnitID)
				}
			}
		}
		p, _ := historicalSuccessorCompatibility(root, locale, catalog, old)
		f, err := verifySuccessorFinalization(root, locale, id, catalog, p)
		if err != nil {
			return nil, nil, err
		}
		for _, u := range f.Units {
			unit, err := catalog.Unit(u.UnitID)
			if err != nil {
				return nil, nil, err
			}
			path, err := canonicalTranslationUnitCandidatePath(locale, unit)
			if err != nil {
				return nil, nil, err
			}
			canonical, err := readCompatibilityFile(root, path)
			candidate, candidateErr := readCompatibilityFile(root, u.Snapshot.CandidatePath)
			if unit.Kind == UnitKindPage {
				candidate = canonicalizeCandidateEOF(candidate)
			}
			if err != nil || candidateErr != nil || !bytes.Equal(canonical, candidate) {
				return nil, nil, fmt.Errorf("current finalized canonical mismatch: %s", u.UnitID)
			}
			original, err := readQualityCheckSnapshot(root, locale, u.SourceSnapshotID, false)
			if err != nil {
				return nil, nil, err
			}
			// This is the exact compatibility chain of the current finalization,
			// with current TU context re-bound above. Non-TU inventories remain
			// the responsibility of their ordinary consumer, not this TU gate.
			r := p.Resolve(original.GlossarySHA256, sum(glossary), "tu:"+u.UnitID)
			if r.Status != "exact" && r.Status != "compatible" {
				return nil, nil, fmt.Errorf("current TU compatibility %s: %s (%s)", u.UnitID, r.Status, r.Reason)
			}
		}
		ref, err := successorReference(root, filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, id, "finalization.json")))
		if err != nil {
			return nil, nil, err
		}
		return f, append(refs, ref), nil
	}
	return nil, nil, fmt.Errorf("missing current glossary-bound finalized TU Snapshot")
}

// AuthorizeHistoricalCanonicalReference is intentionally restricted to exact
// canonical target transitions and the resulting mutable status projection.
// It does not authorize any derived consumer.
func AuthorizeHistoricalCanonicalReference(root, locale, path, oldSHA string, catalog *Catalog) error {
	proof, err := findCurrentTargetSuccessor(root, locale, catalog)
	if err != nil {
		return err
	}
	if proof == nil {
		return fmt.Errorf("missing successor authority")
	}
	if err := requireSuccessorCurrentLanguage(root, catalog, proof); err != nil {
		return err
	}
	if path == proof.evidence.Status.Path {
		b, err := readCompatibilityFile(root, path)
		if err != nil || sum(b) != proof.evidence.Status.SHA256 {
			return fmt.Errorf("resulting status projection changed outside successor transaction")
		}
		if proof.evidence.HistoricalClosure != nil {
			b, err := readHistoricalContextReference(root, catalog, locale, *proof.evidence.HistoricalClosure)
			if err != nil {
				return err
			}
			var l contentidentity.Locale
			if err := contentidentity.StrictJSON(b, &l); err != nil {
				return err
			}
			for _, c := range l.Packages {
				if c.Package == "tour-v1" {
					for _, r := range c.Evidence {
						if r.Path == path && r.SHA256 == oldSHA {
							return nil
						}
					}
				}
			}
		} else if proof.evidence.PreviousStatusSHA == oldSHA {
			return nil
		}
		return fmt.Errorf("old status reference mismatch")
	}
	for _, u := range proof.evidence.Units {
		if u.CanonicalPath == path && u.OldSHA == oldSHA && u.Changed {
			return nil
		}
	}
	return fmt.Errorf("no exact authorized successor for %s", path)
}
