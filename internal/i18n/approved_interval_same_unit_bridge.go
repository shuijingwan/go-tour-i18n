package i18n

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Frozen proof bytes remain archive evidence. The source-owned adapter builds
// today's response projection; no historical implementation is executed.
func approvedSameUnitBridgeOutput(c *Catalog, u *TranslationUnit, record RetranslationBatchUnit, raw, reviewed []byte, files map[string][]byte) (TranslationUnitAdapter, []byte, []byte, error) {
	var old ApprovedIntervalTransport
	if err := contentidentity.StrictJSON(files[approvedIntervalProofMember(u.ID)], &old); err != nil {
		return nil, nil, nil, err
	}
	if old.Schema != ApprovedIntervalTransportSchema || old.Mode != ApprovedIntervalMode || old.Identity != approvedIntervalIdentity(old) || !reflect.DeepEqual(old.Record, record) || old.Projection.UnitID != u.ID || old.Projection.SourceSHA256 != u.SourceSHA256 {
		return nil, nil, nil, fmt.Errorf("approved bridge proof identity mismatch")
	}
	factory, ok := u.Adapter.(ApprovedIntervalSameUnitBridgeAdapter)
	if !ok {
		return nil, nil, nil, fmt.Errorf("Page adapter lacks approved-interval bridge composition")
	}
	return factory.BridgeApprovedIntervalRevision(old.Projection, raw, reviewed)
}

// The complete original current-validation scope supplies the exact machine
// reconciliation/semantic authority. Reuse its existing frozen-A evaluation,
// without entering ordinary installed-response decoding of the selected batch.
func verifyApprovedSameUnitBridgeCurrent(root string, c *Catalog, g *Glossary, locale string, record RetranslationBatchUnit, files map[string][]byte, read, readFrozen func(string) ([]byte, error)) error {
	unit, err := c.Unit(record.UnitID)
	if err != nil {
		return err
	}
	factory, ok := unit.Adapter.(ApprovedIntervalSameUnitBridgeAdapter)
	if !ok {
		return fmt.Errorf("Page adapter lacks approved-interval bridge composition")
	}
	canonical, err := sameUnitBridgeSourceCatalog(c, factory)
	if err != nil {
		return err
	}
	c = canonical
	var m RetranslationBatchManifest
	if err := contentidentity.StrictJSON(files["formal/manifest.json"], &m); err != nil {
		return err
	}
	if m.TransportRecovery != nil {
		return fmt.Errorf("approved bridge does not authorize recovery invocation")
	}
	selection, err := parseApprovedIntervalSelectionBytes(root, files[approvedIntervalSelectionMember], c, &m)
	if err != nil || selection.entry(record.UnitID) == nil {
		return fmt.Errorf("approved bridge original selection mismatch: %v", err)
	}
	expected := map[string]bool{}
	for _, unit := range selection.Units {
		path := approvedIntervalProofMember(unit.CurrentUnitID)
		expected[path] = true
		if files[path] == nil {
			return fmt.Errorf("approved bridge proof inventory incomplete")
		}
	}
	for path := range files {
		if strings.HasPrefix(path, "formal/approved-interval-transport/") && !expected[path] {
			return fmt.Errorf("approved bridge unselected proof member")
		}
	}
	historical, reopen, err := surfaceRevisionCatalog(root, c, locale, record.RevisionAuthorizationID)
	if err != nil {
		return err
	}
	if reopen.AuthoritySource != CurrentValidationAuthority || reopen.CurrentValidation == nil {
		return fmt.Errorf("approved bridge requires original current-validation authority")
	}
	assessment, err := decodeCurrentValidationAssessment(reopen.CurrentValidation)
	if err != nil {
		return err
	}
	evaluation := currentValidationPredecessorEvaluation(root, historical, locale)
	if err := evaluation.materializeOriginalFinalization(&assessment.Historical); err != nil {
		return err
	}
	var baseline *QualityCheckSnapshotUnit
	for _, unit := range reopen.Units {
		if unit.UnitID == record.UnitID {
			copy := unit.Snapshot
			baseline = &copy
		}
	}
	if baseline == nil {
		return fmt.Errorf("approved bridge original Snapshot Unit missing")
	}
	evidence, err := evaluation.readSnapshotUnitRepositoryEvidence(record.PreviousSnapshotID, *baseline)
	if err != nil {
		return err
	}
	current, err := buildApprovedIntervalTransport(root, c, g, locale, record, reopen, *baseline, evidence, selection)
	if err != nil {
		return err
	}
	var old ApprovedIntervalTransport
	if err := contentidentity.StrictJSON(files[approvedIntervalProofMember(record.UnitID)], &old); err != nil {
		return err
	}
	if old.Identity != approvedIntervalIdentity(old) {
		return fmt.Errorf("approved bridge historical proof identity mismatch")
	}
	u, err := c.Unit(record.UnitID)
	if err != nil {
		return err
	}
	// Compare all typed semantic facts through the source-owned operation.
	// No Snapshot/recovery admission is involved.
	if err := sameUnitApprovedProofEquivalent(u, &old, current); err != nil {
		return err
	}
	refs := append([]GlossaryArchiveReference{}, assessment.Historical.References...)
	refs = append(refs, reopen.revisionEvidence())
	for _, ref := range refs {
		b, err := readFrozen(ref.Path)
		if err != nil || sum(b) != ref.SHA256 {
			return fmt.Errorf("approved bridge dependency drift: %s", ref.Path)
		}
	}
	implementation := map[string]bool{}
	for _, path := range approvedIntervalImplementationPaths {
		implementation[path] = true
	}
	for _, ref := range current.Dependencies {
		reader := readFrozen
		if implementation[ref.Path] {
			reader = read
		}
		b, err := reader(ref.Path)
		if err != nil || sum(b) != ref.SHA256 {
			return fmt.Errorf("approved bridge dependency drift: %s", ref.Path)
		}
	}
	input := files["formal/"+record.InputPath]
	if sum(input) != record.InputSHA256 || !bytes.Equal(files["formal/candidates/"+filepath.Base(record.InputPath)], evidence.candidate) {
		return fmt.Errorf("approved bridge original input/candidate binding mismatch")
	}
	return nil
}

func sameUnitApprovedProofEquivalent(u *TranslationUnit, old, current *ApprovedIntervalTransport) error {
	factory, ok := u.Adapter.(ApprovedIntervalSameUnitBridgeAdapter)
	if !ok {
		return fmt.Errorf("Page adapter lacks approved-interval bridge composition")
	}
	if err := factory.CompareApprovedIntervalBridge(old.Projection, current.Projection); err != nil {
		return err
	}
	// These are precisely the producer fields. Projection semantics were checked
	// above, and the complete dependency inventory is checked separately below.
	a, b := *old, *current
	a.Projection, b.Projection = ApprovedIntervalProjection{}, ApprovedIntervalProjection{}
	a.Dependencies, b.Dependencies = nil, nil
	a.CurrentAuthority, b.CurrentAuthority = "", ""
	a.Identity, b.Identity = "", ""
	if !bytes.Equal(mustJSON(a), mustJSON(b)) {
		return fmt.Errorf("approved bridge semantic proof changed")
	}
	if !validSHA256(old.CurrentAuthority) || !validSHA256(current.CurrentAuthority) || len(old.Dependencies) != len(current.Dependencies) {
		return fmt.Errorf("approved bridge dependency inventory changed")
	}
	implementation := map[string]bool{}
	for _, p := range approvedIntervalImplementationPaths {
		implementation[p] = true
	}
	seen := map[string]GlossaryArchiveReference{}
	for i, x := range old.Dependencies {
		y := current.Dependencies[i]
		if x.Path != y.Path || validateGenerationInstallPath(x.Path) != nil || !validSHA256(x.SHA256) || !validSHA256(y.SHA256) {
			return fmt.Errorf("approved bridge dependency binding mismatch")
		}
		if prior, ok := seen[x.Path]; ok && prior != x {
			return fmt.Errorf("approved bridge conflicting dependency binding")
		}
		seen[x.Path] = x
		if x.SHA256 != y.SHA256 && !implementation[x.Path] {
			return fmt.Errorf("approved bridge semantic dependency changed: %s", x.Path)
		}
	}
	return nil
}

// Only remove source-owner observation decorators. This is not a substitute
// package/Unit authority: every current source fact must remain identical.
func sameUnitBridgeSourceCatalog(c *Catalog, factory ApprovedIntervalSameUnitBridgeAdapter) (*Catalog, error) {
	canonical := factory.ApprovedIntervalBridgeCatalog(c)
	p, a, s, ordered, err := translationAuthorizationScope(c)
	if err != nil {
		return nil, err
	}
	cp, ca, cs, currentUnits, err := translationAuthorizationScope(canonical)
	if err != nil || cp != p || ca != a || cs != s || len(currentUnits) != len(ordered) {
		return nil, fmt.Errorf("approved bridge construction package/scope changed")
	}
	for i, u := range ordered {
		if v := currentUnits[i]; u.ID != v.ID || u.Kind != v.Kind || u.SourcePath != v.SourcePath || u.SourceSHA256 != v.SourceSHA256 || !bytes.Equal(u.Source, v.Source) {
			return nil, fmt.Errorf("approved bridge construction source changed")
		}
	}
	return canonical, nil
}
