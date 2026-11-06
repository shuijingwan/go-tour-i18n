package i18n

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
)

// validateCurrentAssessment separates an immutable representation transition
// from today's language-input eligibility. The legacy edge verifier remains
// available to legacy compatibility consumers; its inventory is not expanded.
func (s *machineSelection) validateCurrentAssessment(root string, c *Catalog) (*snapshotUnitRepositoryEvidence, error) {
	locale := s.receipt.Options.Locale
	if err := CheckPromotionTransactionState(root, locale); err != nil {
		return nil, err
	}
	// This verifies original independent A, complete historical QC, mapping,
	// frozen glossary/input and every machine artifact in the receipt's world.
	evidence, err := s.validateInWorld(root, c, true)
	if err != nil {
		return nil, err
	}
	prior := s.ctx.effective[s.unit.Predecessor.UnitID]
	reviewer, err := readQualityCheckSnapshot(root, locale, prior.snapshotID, false)
	if err != nil {
		return nil, err
	}
	reviewerResults, err := readQualityCheckResults(root, locale, reviewer)
	if err != nil || reviewerResults == nil {
		return nil, fmt.Errorf("reconciliation original Reviewer Snapshot binding: %v", err)
	}
	review, err := RequireUnifiedGlossaryReview(root, locale)
	if err != nil {
		return nil, err
	}
	current, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(locale))
	if err != nil {
		return nil, err
	}
	g, err := parseGlossary(locale, bytes.NewReader(current))
	if err != nil {
		return nil, err
	}
	u, err := c.Unit(s.unit.Replacement.CurrentUnitID)
	if err != nil {
		return nil, err
	}
	// Use exactly the primitive used by AssessCurrentTranslation, not merely a
	// validator PASS. Semantic impact and unknown visibility remain fail-closed.
	x, err := assessCurrentTranslationUnit(root, c, locale, u, s.snapshot(s.unit.Predecessor.Index), reviewer.GlossarySHA256, current, g, map[string]*machineSelection{s.ref.SHA256 + u.ID: s})
	if err != nil {
		return nil, err
	}
	if x.Classification != "compatible" || x.Validation != "passed" {
		return nil, fmt.Errorf("reconciled successor current assessment: %s: %s", x.Classification, strings.Join(x.Reasons, "; "))
	}
	// No earlier PASS is authority. Re-observe both the frozen graph and the
	// current inputs before returning this individual machine selection.
	refs := machineAuthorityReferences(s)
	refs = append(refs, GlossaryArchiveReference{Path: filepath.ToSlash(qualityCheckSnapshotManifestPath("", locale, prior.snapshotID)), SHA256: reviewerResults.SnapshotManifestSHA256})
	for _, q := range s.receipt.QCInputs {
		refs = append(refs, GlossaryArchiveReference{Path: q.Path, SHA256: q.SHA256})
	}
	for _, ref := range refs {
		b, err := readSnapshotReferencedFile(root, ref.Path)
		if err != nil || sum(b) != ref.SHA256 {
			return nil, fmt.Errorf("reconciliation assessment dependency drift: %s", ref.Path)
		}
	}
	after, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(locale))
	if err != nil || !bytes.Equal(current, after) {
		return nil, fmt.Errorf("reconciliation assessment glossary drift")
	}
	afterReview, err := RequireUnifiedGlossaryReview(root, locale)
	if err != nil || !reflect.DeepEqual(review, afterReview) {
		return nil, fmt.Errorf("reconciliation assessment Unified Review drift")
	}
	source, err := readCompatibilityFile(root, u.SourcePath)
	if err != nil || validateSnapshotSourceIdentity(c, u, source) != nil {
		return nil, fmt.Errorf("reconciliation assessment source drift")
	}
	authority, err := currentTranslationAuthority(root, u, locale)
	if err != nil || authority != x.AuthoritySHA {
		return nil, fmt.Errorf("reconciliation assessment adapter/validator drift")
	}
	evidence.unit = u
	return evidence, nil
}
