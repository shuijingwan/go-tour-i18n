package i18n

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
)

// Only reconstruct an existing passed response and its source-owned mapping.
// Surface authorization is checked by the caller, independently of this read
// domain. The QC bridge receipt/builder remains QC-only.
func surfaceTransportOrigin(root string, c *Catalog, g *Glossary, locale string, r RetranslationBatchUnit, s QualityCheckSnapshotUnit, e *snapshotUnitRepositoryEvidence, raw []byte, read func(string) ([]byte, error)) (protectedTranslation, []byte, error) {
	u, err := c.Unit(r.UnitID)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	if e.record.RevisionFeedbackSource != CurrentValidationRevisionFeedback {
		p, lineage, err := inheritedBridgeOutput(root, c, g, locale, r)
		if err != nil {
			return p, nil, err
		}
		if lineage == nil {
			p, err = prepareTranslationUnitInput(u, g)
			if err != nil {
				return p, nil, err
			}
		} else {
			for _, ref := range lineage.Dependencies {
				resolved, b, err := readHistoricalBridgeReference(root, locale, ref)
				if err != nil {
					return p, nil, err
				}
				got, err := read(resolved.Path)
				if err != nil || !bytes.Equal(got, b) {
					return p, nil, fmt.Errorf("Surface historical decoder drift")
				}
			}
		}
		text, failures := p.restore(string(raw))
		if len(failures) != 0 || !bytes.Equal(canonicalizeRetranslationArtifactEOF([]byte(text)), e.candidate) {
			return p, nil, fmt.Errorf("Surface ordinary origin does not restore exact candidate")
		}
		return p, raw, nil
	}
	factory, ok := u.Adapter.(SameUnitBridgeAdapter)
	if !ok || e.record.GenerationOrigin != nil {
		return protectedTranslation{}, nil, fmt.Errorf("Surface restricted source composition unavailable")
	}
	probe := GenerationBundleManifest{Attempt: s.Attempt}
	if s.Attempt > 1 {
		probe.TaskKind = "translation-unit-retry"
		probe.ExpectedOutputs = []GenerationBundleExpectedOutput{{BundlePath: "outputs/" + filepath.Base(e.record.InputPath)}}
	}
	importPath, err := repositoryRelativePath(root, generationImportReceiptPath(e.batchDir, probe))
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	b, err := read(importPath)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	var imported GenerationResultBundleManifest
	if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &imported); err != nil {
		return protectedTranslation{}, nil, err
	}
	archivePath := importedGenerationArchive(locale, imported.GenerationBundleSHA256)
	archive, err := read(archivePath)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	m, files, err := readGenerationBundle(archive)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	if err := checkImportedGenerationIdentity(imported, m, archive, files); err != nil {
		return protectedTranslation{}, nil, err
	}
	if m.Locale != locale || m.BatchID != s.SelectedBatchID || m.Attempt != s.Attempt {
		return protectedTranslation{}, nil, fmt.Errorf("Surface restricted import/attempt mismatch")
	}
	matched := 0
	for _, out := range m.ExpectedOutputs {
		if out.UnitID == u.ID && out.InstallPath == e.validation.RawResponsePath {
			for _, v := range imported.Outputs {
				if out.BundlePath == v.BundlePath && v.SHA256 == sum(raw) && v.Size == int64(len(raw)) {
					matched++
				}
			}
		}
	}
	if matched != 1 {
		return protectedTranslation{}, nil, fmt.Errorf("Surface restricted raw provenance mismatch")
	}
	for _, f := range m.Files {
		if f.RepositoryPath == "" || strings.HasPrefix(f.BundlePath, "authority/") || strings.HasPrefix(f.BundlePath, "formal/retry/") {
			continue
		}
		ref, want, err := readHistoricalBridgeReference(root, locale, GlossaryArchiveReference{f.RepositoryPath, f.SHA256})
		if err != nil {
			return protectedTranslation{}, nil, err
		}
		got, err := read(ref.Path)
		if err != nil || !bytes.Equal(got, want) || !bytes.Equal(got, files[f.BundlePath]) {
			return protectedTranslation{}, nil, fmt.Errorf("Surface restricted archived input drift: %s", f.BundlePath)
		}
	}
	cv, err := readCurrentQualityCheckSurfaceReopen(root, c, locale, e.record.RevisionAuthorizationID)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	if cv.AuthoritySource != CurrentValidationAuthority {
		return protectedTranslation{}, nil, fmt.Errorf("Surface origin is not current-validation transport")
	}
	path := "data/quality-check-reopens/" + locale + "/" + cv.ReopenID + ".json"
	cb, err := read(path)
	if err != nil || !bytes.Equal(cb, currentValidationReopenBytes(cv)) {
		return protectedTranslation{}, nil, fmt.Errorf("Surface restricted reopen drift")
	}
	assessment, err := decodeCurrentValidationAssessment(cv.CurrentValidation)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	for _, ref := range assessment.Historical.References {
		b, err := read(ref.Path)
		if err != nil || sum(b) != ref.SHA256 {
			return protectedTranslation{}, nil, fmt.Errorf("Surface restricted historical dependency drift: %s", ref.Path)
		}
	}
	name := filepath.Base(e.record.InputPath)
	before, ancestor := files["formal/candidates/"+name], files["formal/historical-protected-candidates/"+name]
	if len(before) == 0 || len(ancestor) == 0 {
		return protectedTranslation{}, nil, fmt.Errorf("Surface restricted ordinary ancestor missing")
	}
	p, err := buildCommentRevisionTransport(root, c, g, locale, u, cv, before)
	if err != nil || p == nil || !bytes.Equal(mustJSON(p), files[commentTransportPath(e.record)]) {
		return protectedTranslation{}, nil, fmt.Errorf("Surface restricted archived mapping mismatch: %v", err)
	}
	restored, err := p.Restore(string(raw))
	if err != nil || !bytes.Equal(canonicalizeRetranslationArtifactEOF(restored), e.candidate) {
		return protectedTranslation{}, nil, fmt.Errorf("Surface restricted exact restore failed")
	}
	adapter, template, err := factory.BridgeRestrictedRevision(ancestor, before, e.candidate, raw, p)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	restored, err = adapter.Restore(string(template))
	if err != nil || !bytes.Equal(canonicalizeRetranslationArtifactEOF(restored), e.candidate) {
		return protectedTranslation{}, nil, fmt.Errorf("Surface ordinary source composition failed exact restore")
	}
	text, tokens, err := adapter.ProtectedInput(g)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	return protectedTranslation{Text: text, Tokens: tokens, Adapter: adapter}, template, nil
}
