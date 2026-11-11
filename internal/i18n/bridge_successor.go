package i18n

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// Representation lineage is derived from immutable selected Snapshot attempts.
// It is not revision authorization: the caller must independently validate the
// current revision's QC/Surface authority. No manifest or receipt is manufactured.
type bridgeResponseLineage struct {
	Bridge       GlossaryArchiveReference   `json:"bridge"`
	Dependencies []GlossaryArchiveReference `json:"dependencies"`
}

// Resolve only the sanctioned glossary archive in the historical read domain.
// The original lineage refs stay byte-identical; flattened historical proofs
// bind the immutable path instead of turning a mutable path into a SHA oracle.
func readHistoricalBridgeReference(root, locale string, ref GlossaryArchiveReference) (GlossaryArchiveReference, []byte, error) {
	var b []byte
	var err error
	switch {
	case ref.Path == glossaryReviewGlossaryPath(locale), strings.HasPrefix(ref.Path, "data/glossary-history/"):
		path, pathErr := glossaryArchivePath(locale, ref.SHA256)
		if pathErr != nil {
			return ref, nil, pathErr
		}
		if ref.Path != glossaryReviewGlossaryPath(locale) && ref.Path != path {
			return ref, nil, fmt.Errorf("historical glossary archive scope/path mismatch: %s", ref.Path)
		}
		b, err = readArchivedGlossary(root, locale, ref.SHA256)
		ref.Path = path
	case strings.HasPrefix(ref.Path, "locales/") && strings.HasSuffix(ref.Path, "/glossary.yaml"):
		return ref, nil, fmt.Errorf("historical glossary locale mismatch: %s", ref.Path)
	default:
		b, err = readSnapshotReferencedFile(root, ref.Path)
	}
	if err != nil {
		return ref, nil, err
	}
	if sum(b) != ref.SHA256 {
		return ref, nil, fmt.Errorf("historical reference hash mismatch: %s", ref.Path)
	}
	return ref, b, nil
}

func (p *bridgeResponseLineage) check(root, locale string) error {
	for _, ref := range p.Dependencies {
		if _, _, err := readHistoricalBridgeReference(root, locale, ref); err != nil {
			return fmt.Errorf("historical representation dependency drift: %s: %w", ref.Path, err)
		}
	}
	return nil
}

// The same invocation-owned reconstruction abstraction has separate current
// authorization and historical-decoder domains. A frozen result can never be
// reused as a current bridge authorization PASS. Every lookup rechecks disk.
func historicalBridgeDecoder(root string, c *Catalog, g *Glossary, locale string, record RetranslationBatchUnit) (protectedTranslation, *bridgeResponseLineage, error) {
	ref := *record.RepresentationBridge
	r, err := readFrozenSameUnitBridge(root, locale, ref)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	build := func() (*SameUnitBridgeResolution, error) {
		result := &SameUnitBridgeResolution{Receipt: r, Adapters: map[string]TranslationUnitAdapter{}}
		for _, q := range r.Units {
			// Each sibling has its own formal identity and restricted predecessor
			// semantics. Recover its exact SHA-bound record, never the caller's.
			var historical *RetranslationBatchUnit
			for _, binding := range r.SameUnitBridge.Bindings {
				if binding.UnitID != q.UnitID {
					continue
				}
				_, data, err := readHistoricalBridgeReference(root, locale, binding.OriginalManifest)
				if err != nil {
					return nil, err
				}
				manifest, err := decodeRetranslationManifest(data)
				if err != nil {
					return nil, err
				}
				if manifest.Locale != locale || manifest.BatchID != q.Snapshot.SelectedBatchID {
					return nil, fmt.Errorf("historical bridge predecessor manifest identity mismatch: %s", q.UnitID)
				}
				for _, predecessor := range manifest.Units {
					if predecessor.UnitID == q.UnitID {
						if historical != nil {
							return nil, fmt.Errorf("ambiguous historical bridge predecessor: %s", q.UnitID)
						}
						copy := predecessor
						historical = &copy
					}
				}
			}
			if historical == nil {
				return nil, fmt.Errorf("historical bridge predecessor missing: %s", q.UnitID)
			}
			historical.RepresentationBridge = &ref
			historical.PreviousSnapshotID = r.PreviousSnapshotID
			historical.PreviousRating, historical.PreviousFinding = q.PreviousRating, q.Finding
			historical.RevisionFeedbackSource = "quality_check"
			p, err := frozenBridgeOutput(root, c, g, locale, *historical)
			if err != nil {
				return nil, err
			}
			result.Adapters[q.UnitID] = p.Adapter
		}
		return result, nil
	}
	var result *SameUnitBridgeResolution
	if c != nil && c.BridgeResolver != nil {
		result, err = c.BridgeResolver(root, c, "historical", ref, build)
	} else {
		result, err = build()
	}
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	if result == nil || !reflect.DeepEqual(r, result.Receipt) || len(result.Adapters) != len(r.Units) {
		return protectedTranslation{}, nil, fmt.Errorf("historical bridge reconstruction differs")
	}
	bound := false
	for _, q := range r.Units {
		if q.UnitID == record.UnitID && record.RevisionFeedbackSource == "quality_check" && record.PreviousSnapshotID == r.PreviousSnapshotID && record.PreviousRating == q.PreviousRating && record.PreviousFinding == q.Finding {
			bound = true
		}
	}
	for _, b := range r.SameUnitBridge.Bindings {
		if b.UnitID == record.UnitID && b.FormalInputSHA256 != record.InputSHA256 {
			return protectedTranslation{}, nil, fmt.Errorf("historical bridge formal input drift")
		}
	}
	if !bound || result.Adapters[record.UnitID] == nil {
		return protectedTranslation{}, nil, fmt.Errorf("historical bridge record mismatch")
	}
	a := result.Adapters[record.UnitID]
	text, tokens, err := a.ProtectedInput(g)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	p := &bridgeResponseLineage{Bridge: ref, Dependencies: append([]GlossaryArchiveReference{ref}, r.SameUnitBridge.FrozenDependencies...)}
	if err := p.check(root, locale); err != nil {
		return protectedTranslation{}, nil, err
	}
	return protectedTranslation{Text: text, Tokens: tokens, Adapter: a}, p, nil
}

func inheritedBridgeOutput(root string, c *Catalog, g *Glossary, locale string, record RetranslationBatchUnit) (protectedTranslation, *bridgeResponseLineage, error) {
	return walkBridgeResponseLineage(root, c, g, locale, record, map[string]bool{})
}

// QC and Surface feedback authorize different revisions, but the same-ID
// predecessor's response representation comes from its immutable lineage.
// This predicate grants no bridge or revision authority on its own.
func hasInheritedBridgePredecessor(record RetranslationBatchUnit) bool {
	return record.PreviousSnapshotID != "" && (record.RevisionFeedbackSource == "quality_check" || record.RevisionFeedbackSource == "surface_review")
}

func walkBridgeResponseLineage(root string, c *Catalog, g *Glossary, locale string, record RetranslationBatchUnit, seen map[string]bool) (protectedTranslation, *bridgeResponseLineage, error) {
	if record.RepresentationBridge != nil {
		return historicalBridgeDecoder(root, c, g, locale, record)
	}
	// Restricted current-validation transports and replacement Units do not
	// become ordinary editors through this path. Only same-ID QC/Surface chains.
	if !hasInheritedBridgePredecessor(record) {
		return protectedTranslation{}, nil, nil
	}
	key := record.PreviousSnapshotID + ":" + record.UnitID
	if seen[key] {
		return protectedTranslation{}, nil, fmt.Errorf("cyclic historical representation lineage")
	}
	seen[key] = true
	s, err := readQualityCheckSnapshot(root, locale, record.PreviousSnapshotID, false)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	var selected *QualityCheckSnapshotUnit
	for i := range s.Units {
		if s.Units[i].UnitID == record.UnitID {
			selected = &s.Units[i]
		}
	}
	if selected == nil || selected.RepresentationReconciliation != nil {
		return protectedTranslation{}, nil, nil
	}
	// Discover a bridge along immutable same-ID manifests before consulting
	// current source adapters. Unrelated pre-replacement history must continue
	// through its existing representation-machine reader, never this decoder.
	ancestorBatch, err := readRetranslationProcessManifest(filepath.Join(root, "data/retranslation-runs", locale, selected.SelectedBatchID), locale, selected.SelectedBatchID)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	var ancestor *RetranslationBatchUnit
	for i := range ancestorBatch.Units {
		if ancestorBatch.Units[i].UnitID == record.UnitID {
			ancestor = &ancestorBatch.Units[i]
		}
	}
	if ancestor == nil {
		return protectedTranslation{}, nil, fmt.Errorf("historical selected manifest Unit missing")
	}
	p, lineage, err := surfaceTransportOutput(root, c, locale, *ancestor)
	if err == nil && lineage == nil {
		p, lineage, err = walkBridgeResponseLineage(root, c, g, locale, *ancestor, seen)
	}
	if err != nil || lineage == nil {
		return p, lineage, err
	}
	// Only this proven same-ID inherited decoder needs a complete predecessor
	// Catalog. Non-same-ID replacements returned above retain their original
	// frozen invocation codec and independently verified replacement authority.
	historical, scopeCheck, err := acceptedResponseCatalog(root, c, locale, record)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	if _, err := readQualityCheckSnapshot(root, locale, record.PreviousSnapshotID, false, historical); err != nil {
		return protectedTranslation{}, nil, err
	}
	e, err := readSnapshotUnitRepositoryEvidence(root, historical, locale, *selected)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	if !reflect.DeepEqual(*ancestor, e.record) || record.UnitKind != e.record.UnitKind || record.SourcePath != e.record.SourcePath || record.SourceSHA256 != e.record.SourceSHA256 || record.InputSHA256 != e.record.InputSHA256 {
		return protectedTranslation{}, nil, fmt.Errorf("historical bridge successor source/input identity mismatch")
	}
	refs := map[string]string{}
	for _, r := range lineage.Dependencies {
		refs[r.Path] = r.SHA256
	}
	bind := func(path, sha string) ([]byte, error) {
		var b []byte
		var err error
		if sha != "" {
			_, b, err = readHistoricalBridgeReference(root, locale, GlossaryArchiveReference{Path: path, SHA256: sha})
		} else {
			b, err = readSnapshotReferencedFile(root, path)
		}
		if err != nil || (sha != "" && sum(b) != sha) {
			return nil, fmt.Errorf("historical predecessor reference drift: %s", path)
		}
		if old, ok := refs[path]; ok && old != sum(b) {
			return nil, fmt.Errorf("historical predecessor changed: %s", path)
		}
		refs[path] = sum(b)
		return b, nil
	}
	base := "data/retranslation-runs/" + locale + "/" + selected.SelectedBatchID
	for path, sha := range map[string]string{
		"data/quality-check-snapshots/" + locale + "/" + record.PreviousSnapshotID + "/manifest.json": "",
		base + "/manifest.json": "", base + "/result.json": "", base + "/" + e.record.InputPath: e.record.InputSHA256,
		selected.CandidatePath: selected.CandidateSHA256, selected.ValidationPath: selected.ValidationSHA256,
	} {
		if _, err := bind(path, sha); err != nil {
			return protectedTranslation{}, nil, err
		}
	}
	if err := ensureNoGenerationImportPending(e.batchDir); err != nil {
		return protectedTranslation{}, nil, err
	}
	raw, err := bind(base+"/"+e.validation.RawResponsePath, "")
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	// Verify the actual imported invocation/archive, not today's reconstruction
	// of its old command policy. Its inventory binds the bridge ref and template.
	gm := GenerationBundleManifest{TaskKind: "translation-unit-revision", Attempt: 1}
	if selected.Attempt > 1 {
		gm.TaskKind = "translation-unit-retry"
		gm.Attempt = selected.Attempt
		gm.ExpectedOutputs = []GenerationBundleExpectedOutput{{BundlePath: filepath.Base(e.record.InputPath)}}
	}
	receiptPath, err := repositoryRelativePath(root, generationImportReceiptPath(e.batchDir, gm))
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	b, err := bind(receiptPath, "")
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	var receipt GenerationResultBundleManifest
	if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &receipt); err != nil {
		return protectedTranslation{}, nil, err
	}
	if !validSHA256(receipt.GenerationBundleSHA256) {
		return protectedTranslation{}, nil, fmt.Errorf("invalid historical archive identity")
	}
	archive, err := bind(importedGenerationArchive(locale, receipt.GenerationBundleSHA256), receipt.GenerationBundleSHA256)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	original, files, err := readGenerationBundle(archive)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	if err := checkImportedGenerationIdentity(receipt, original, archive, files); err != nil {
		return protectedTranslation{}, nil, err
	}
	if original.Locale != locale || original.BatchID != selected.SelectedBatchID || original.Attempt != selected.Attempt || original.TaskKind != gm.TaskKind {
		return protectedTranslation{}, nil, fmt.Errorf("historical bridge invocation/attempt mismatch")
	}
	// Inventory equality covers manifest, source, protected input, candidate and
	// mapping bytes. Mutable command policy and prior retry status stay historical.
	for _, f := range original.Files {
		if f.RepositoryPath == "" || strings.HasPrefix(f.BundlePath, "authority/") || strings.HasPrefix(f.BundlePath, "formal/retry/") {
			continue
		}
		got, err := bind(f.RepositoryPath, f.SHA256)
		if err != nil || !bytes.Equal(got, files[f.BundlePath]) {
			return protectedTranslation{}, nil, fmt.Errorf("historical bridge imported input drift: %s", f.BundlePath)
		}
	}
	if e.record.RepresentationBridge == nil && !(e.record.RevisionFeedbackSource == "surface_review" && hasSurfaceTransport(root, locale, e.record.RevisionAuthorizationID)) && !bytes.Equal(files["formal/representation-lineage/"+record.UnitID+".json"], mustJSON(lineage)) {
		return protectedTranslation{}, nil, fmt.Errorf("historical representation lineage mapping drift")
	}
	manifestBytes, err := bind(base+"/manifest.json", "")
	if err != nil || !bytes.Equal(files["formal/manifest.json"], manifestBytes) {
		return protectedTranslation{}, nil, fmt.Errorf("historical imported manifest mismatch")
	}
	matched := false
	for _, out := range original.ExpectedOutputs {
		if out.UnitID != record.UnitID {
			continue
		}
		if out.InstallPath != e.validation.RawResponsePath {
			return protectedTranslation{}, nil, fmt.Errorf("historical selected attempt/raw mismatch")
		}
		for _, f := range receipt.Outputs {
			if f.BundlePath == out.BundlePath && f.SHA256 == sum(raw) && f.Size == int64(len(raw)) {
				if matched {
					return protectedTranslation{}, nil, fmt.Errorf("duplicate historical output")
				}
				matched = true
			}
		}
	}
	if !matched {
		return protectedTranslation{}, nil, fmt.Errorf("historical raw provenance mismatch")
	}
	batch, err := readRetranslationProcessManifest(e.batchDir, locale, selected.SelectedBatchID)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	restored, failures := p.restore(string(raw))
	candidate := []byte(restored)
	if batch.ArtifactEOF == retranslationArtifactEOFSingleLF {
		if err := validateRetranslationArtifactEOF(raw); err != nil {
			return protectedTranslation{}, nil, err
		}
		candidate = canonicalizeRetranslationArtifactEOF(candidate)
	}
	if len(failures) != 0 || !bytes.Equal(candidate, e.candidate) {
		return protectedTranslation{}, nil, fmt.Errorf("historical bridge raw does not restore selected candidate")
	}
	lineage.Dependencies = nil
	for path, sha := range refs {
		lineage.Dependencies = append(lineage.Dependencies, GlossaryArchiveReference{Path: path, SHA256: sha})
	}
	sort.Slice(lineage.Dependencies, func(i, j int) bool { return lineage.Dependencies[i].Path < lineage.Dependencies[j].Path })
	if err := lineage.check(root, locale); err != nil {
		return protectedTranslation{}, nil, err
	}
	if scopeCheck != nil {
		if err := scopeCheck(); err != nil {
			return protectedTranslation{}, nil, err
		}
	}
	return p, lineage, nil
}

// Historical A binds the imported proof without replaying today's glossary,
// validator or QC policy. Current eligibility remains the separate caller gate.
func frozenSuccessorBridgeReferences(root, locale string, record RetranslationBatchUnit, batch string, attempt int) ([]GlossaryArchiveReference, error) {
	if record.RevisionFeedbackSource == "surface_review" && hasSurfaceTransport(root, locale, record.RevisionAuthorizationID) {
		return frozenSurfaceTransportReferences(root, locale, record, batch)
	}
	originalRecord := record
	sameInput := true
	seen := map[string]bool{}
	for record.RepresentationBridge == nil {
		if !hasInheritedBridgePredecessor(record) {
			return nil, nil
		}
		if seen[record.PreviousSnapshotID] {
			return nil, fmt.Errorf("cyclic frozen representation lineage")
		}
		seen[record.PreviousSnapshotID] = true
		s, err := readQualityCheckSnapshot(root, locale, record.PreviousSnapshotID, false)
		if err != nil {
			return nil, err
		}
		var selected *QualityCheckSnapshotUnit
		for i := range s.Units {
			if s.Units[i].UnitID == record.UnitID {
				selected = &s.Units[i]
			}
		}
		if selected == nil || selected.RepresentationReconciliation != nil {
			return nil, nil
		}
		m, err := readRetranslationProcessManifest(filepath.Join(root, "data/retranslation-runs", locale, selected.SelectedBatchID), locale, selected.SelectedBatchID)
		if err != nil {
			return nil, err
		}
		var next *RetranslationBatchUnit
		for i := range m.Units {
			if m.Units[i].UnitID == record.UnitID {
				next = &m.Units[i]
			}
		}
		if next == nil {
			return nil, fmt.Errorf("frozen representation Unit missing")
		}
		// A real replacement is handled by its existing representation authority.
		if next.UnitKind != record.UnitKind || next.SourcePath != record.SourcePath || next.SourceSHA256 != record.SourceSHA256 || next.InputSHA256 != record.InputSHA256 {
			sameInput = false
		}
		record = *next
	}
	if !sameInput {
		return nil, fmt.Errorf("frozen successor source/input drift")
	}
	rootReceipt, err := readFrozenSameUnitBridge(root, locale, *record.RepresentationBridge)
	if err != nil {
		return nil, err
	}
	matched := false
	for _, q := range rootReceipt.Units {
		if q.UnitID == record.UnitID && record.PreviousSnapshotID == rootReceipt.PreviousSnapshotID && record.PreviousRating == q.PreviousRating && record.PreviousFinding == q.Finding && record.RevisionFeedbackSource == "quality_check" {
			matched = true
		}
	}
	if !matched {
		return nil, fmt.Errorf("frozen successor root bridge QC binding drift")
	}
	dir := filepath.Join(root, "data/retranslation-runs", locale, batch)
	gm := GenerationBundleManifest{TaskKind: "translation-unit-revision", Attempt: 1}
	if attempt > 1 {
		gm.TaskKind = "translation-unit-retry"
		gm.Attempt = attempt
		gm.ExpectedOutputs = []GenerationBundleExpectedOutput{{BundlePath: filepath.Base(originalRecord.InputPath)}}
	}
	path, err := repositoryRelativePath(root, generationImportReceiptPath(dir, gm))
	if err != nil {
		return nil, err
	}
	b, err := readSnapshotReferencedFile(root, path)
	if err != nil {
		return nil, err
	}
	var receipt GenerationResultBundleManifest
	if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &receipt); err != nil {
		return nil, err
	}
	if !validSHA256(receipt.GenerationBundleSHA256) {
		return nil, fmt.Errorf("invalid historical lineage archive SHA")
	}
	archivePath := importedGenerationArchive(locale, receipt.GenerationBundleSHA256)
	archive, err := readSnapshotReferencedFile(root, archivePath)
	if err != nil {
		return nil, err
	}
	original, files, err := readGenerationBundle(archive)
	if err != nil {
		return nil, err
	}
	if err := checkImportedGenerationIdentity(receipt, original, archive, files); err != nil {
		return nil, err
	}
	var lineage bridgeResponseLineage
	if err := decodeStrictCourseSourceDescriptionReviewJSON(files["formal/representation-lineage/"+record.UnitID+".json"], &lineage); err != nil {
		return nil, err
	}
	if original.BatchID != batch || original.Locale != locale || original.Attempt != attempt || original.TaskKind != gm.TaskKind || lineage.Bridge != *record.RepresentationBridge {
		return nil, fmt.Errorf("passed successor imported representation lineage mismatch")
	}
	if err := lineage.check(root, locale); err != nil {
		return nil, err
	}
	refs := map[string]string{}
	for _, r := range lineage.Dependencies {
		if refs[r.Path] != "" {
			return nil, fmt.Errorf("duplicate frozen successor dependency")
		}
		refs[r.Path] = r.SHA256
	}
	for _, r := range append([]GlossaryArchiveReference{lineage.Bridge}, rootReceipt.SameUnitBridge.FrozenDependencies...) {
		if refs[r.Path] != r.SHA256 {
			return nil, fmt.Errorf("incomplete frozen successor bridge proof")
		}
	}
	// Do not rewrite the imported lineage. Its consumers bind resolved immutable
	// refs so their later disk checks remain in the same historical domain.
	resolved := make([]GlossaryArchiveReference, 0, len(lineage.Dependencies)+2)
	for _, ref := range lineage.Dependencies {
		r, _, err := readHistoricalBridgeReference(root, locale, ref)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, r)
	}
	return append(resolved, GlossaryArchiveReference{Path: path, SHA256: sum(b)}, GlossaryArchiveReference{Path: archivePath, SHA256: sum(archive)}), nil
}
