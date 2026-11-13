package i18n

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func currentValidationBridgeMember(r RetranslationBatchUnit) string {
	return "formal/current-validation-predecessor-transport/" + filepath.Base(r.InputPath) + ".json"
}

// Inventory dispatch is identity-based: mutually incompatible members cannot
// acquire meaning from branch precedence.
func sameUnitBridgeRepresentationUnambiguous(files map[string][]byte, r RetranslationBatchUnit) error {
	present := 0
	for _, member := range []string{surfaceBridgeMember(r.UnitID), approvedIntervalProofMember(r.UnitID), currentValidationBridgeMember(r), commentTransportPath(r)} {
		if files[member] != nil {
			present++
		}
	}
	if present > 1 {
		return fmt.Errorf("bridge predecessor representation is ambiguous")
	}
	return nil
}

func sameUnitPredecessorOrdinaryDecoder(normal, response TranslationUnitAdapter) error {
	if normal == nil || response == nil {
		return fmt.Errorf("bridge predecessor ordinary decoder missing")
	}
	if marker, ok := response.(interface{ UnrestrictedRevisionTransport() bool }); ok {
		if marker.UnrestrictedRevisionTransport() {
			return nil
		}
		return fmt.Errorf("bridge predecessor decoder is not generic ordinary QC authority")
	}
	if reflect.TypeOf(response) != reflect.TypeOf(normal) {
		return fmt.Errorf("bridge predecessor decoder has no ordinary QC capability")
	}
	return nil
}

// Consume the existing source-owned cvpred, never compose another mapping.
// The archive proves the original response representation; the canonical
// inspector supplies today's private decoder. Snapshot's typed comparator
// permits only its existing producer drift, without granting Snapshot authority.
func currentValidationSameUnitBridgeOutput(root string, c *Catalog, g *Glossary, locale string, r RetranslationBatchUnit, raw, reviewed []byte, files map[string][]byte, read, readFrozen func(string) ([]byte, error)) (TranslationUnitAdapter, []byte, []byte, error) {
	if err := sameUnitBridgeRepresentationUnambiguous(files, r); err != nil {
		return nil, nil, nil, err
	}
	var old CurrentValidationPredecessor
	if err := contentidentity.StrictJSON(files[currentValidationBridgeMember(r)], &old); err != nil {
		return nil, nil, nil, err
	}
	if old.Schema != "go-learning/current-validation-predecessor-transport/v1" || old.Locale != locale || old.UnitID != r.UnitID || old.SourcePath != r.SourcePath || old.CurrentInputSHA256 != r.InputSHA256 || old.RecoveryClass != "DETERMINISTIC_REBASE_PASS" || old.Template == "" || !validSHA256(old.TemplateSHA256) || sum([]byte(old.Template)) != old.TemplateSHA256 {
		return nil, nil, nil, fmt.Errorf("bridge current-validation predecessor identity/template mismatch")
	}
	var m RetranslationBatchManifest
	if err := contentidentity.StrictJSON(files["formal/manifest.json"], &m); err != nil {
		return nil, nil, nil, err
	}
	if m.Locale != locale || m.TransportRecovery != nil {
		return nil, nil, nil, fmt.Errorf("bridge current-validation predecessor invocation mismatch")
	}
	matched := 0
	for _, record := range m.Units {
		if reflect.DeepEqual(record, r) {
			matched++
		}
	}
	if matched != 1 {
		return nil, nil, nil, fmt.Errorf("bridge current-validation predecessor record mismatch")
	}
	u, err := c.Unit(r.UnitID)
	if err != nil {
		return nil, nil, nil, err
	}
	canonical := c
	if factory, ok := u.Adapter.(ApprovedIntervalSameUnitBridgeAdapter); ok {
		canonical, err = sameUnitBridgeSourceCatalog(c, factory)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	rows, err := inspectCurrentValidationPredecessors(root, canonical, locale, m.Units)
	if err != nil {
		return nil, nil, nil, err
	}
	var fresh *CurrentValidationPredecessor
	for i := range rows {
		if rows[i].UnitID == r.UnitID {
			if fresh != nil {
				return nil, nil, nil, fmt.Errorf("duplicate bridge predecessor")
			}
			fresh = &rows[i]
		}
	}
	if fresh == nil || fresh.RecoveryClass != "DETERMINISTIC_REBASE_PASS" || fresh.output.Adapter == nil {
		return nil, nil, nil, fmt.Errorf("bridge current-validation predecessor is not a deterministic rebase")
	}
	freshFiles := map[string][]byte{}
	expected := map[string]bool{}
	for _, path := range currentValidationPredecessorImplementationPaths {
		key := "formal/current-validation-predecessor-implementation/" + path
		expected[key] = true
		data, err := read(path)
		if err != nil {
			return nil, nil, nil, err
		}
		freshFiles[key] = data
	}
	for path := range files {
		if strings.HasPrefix(path, "formal/current-validation-predecessor-implementation/") && !expected[path] {
			return nil, nil, nil, fmt.Errorf("bridge orphan predecessor implementation member: %s", path)
		}
	}
	if _, err := snapshotCurrentValidationPredecessorEquivalent(root, &old, fresh, files, freshFiles); err != nil {
		return nil, nil, nil, err
	}
	// Freeze historical references via the same archive-aware reader as cvpred;
	// current implementation refs remain current construction dependencies.
	refs := append(append([]GlossaryArchiveReference{}, fresh.Dependencies...), fresh.SnapshotManifest, fresh.Reopen)
	for _, ref := range refs {
		resolved, _, err := readHistoricalBridgeReference(root, locale, ref)
		if err != nil {
			return nil, nil, nil, err
		}
		data, err := readFrozen(resolved.Path)
		if err != nil || sum(data) != resolved.SHA256 {
			return nil, nil, nil, fmt.Errorf("bridge predecessor dependency drift: %s", ref.Path)
		}
	}
	// Canonical formal adapters supply the normal generic QC decoder. A distinct
	// transport must explicitly declare itself unrestricted; Surface-bounded
	// decoders cannot acquire ordinary editing authority through this consumer.
	formal, err := canonical.Unit(r.UnitID)
	if err != nil {
		return nil, nil, nil, err
	}
	normal, err := prepareTranslationUnitInput(formal, g)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := sameUnitPredecessorOrdinaryDecoder(normal.Adapter, fresh.output.Adapter); err != nil {
		return nil, nil, nil, err
	}
	baseline, failed := fresh.output.restore(fresh.Template)
	if len(failed) != 0 || fresh.evidence == nil || !bytes.Equal(canonicalizeRetranslationArtifactEOF([]byte(baseline)), fresh.evidence.candidate) {
		return nil, nil, nil, fmt.Errorf("bridge predecessor template inverse mismatch")
	}
	// This is the selected invocation's exact imported FINAL response, not
	// cvpred.Raw (the older sealed predecessor), and never Protect(candidate).
	got, failed := fresh.output.restore(string(raw))
	if len(failed) != 0 || !bytes.Equal(canonicalizeRetranslationArtifactEOF([]byte(got)), reviewed) {
		return nil, nil, nil, fmt.Errorf("bridge predecessor reviewed inverse mismatch")
	}
	if err := validateTranslationUnitWithGlossary(root, c, u, locale, reviewed, g); err != nil {
		return nil, nil, nil, fmt.Errorf("bridge current validation failed: %v", err)
	}
	// An already ordinary decoder may still contain source formatting atoms.
	// Only its source owner may project those bytes into a private response;
	// the original cvpred/template and all authority proofs above stay exact.
	if projection, ok := fresh.output.Adapter.(interface {
		BridgeResponseFormattingWhitespace([]byte) (TranslationUnitAdapter, []byte, error)
	}); ok {
		adapter, template, err := projection.BridgeResponseFormattingWhitespace(raw)
		if err != nil {
			return nil, nil, nil, err
		}
		restored, err := adapter.Restore(string(template))
		if err != nil || !bytes.Equal(canonicalizeRetranslationArtifactEOF(restored), reviewed) {
			return nil, nil, nil, fmt.Errorf("bridge formatting reviewed inverse mismatch: %v", err)
		}
		return adapter, canonicalizeRetranslationArtifactEOF(template), []byte(fresh.Template), nil
	}
	return fresh.output.Adapter, canonicalizeRetranslationArtifactEOF(raw), []byte(fresh.Template), nil
}
