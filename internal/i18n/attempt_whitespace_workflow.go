package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

var whitespaceFrozenMembers = []string{"origin-raw.article", "origin-candidate.article", "origin-validation.json", "previous-raw.article", "previous-validation.json"}

// Exact invocation lookup, never latest state or marker-based authorization.
func whitespaceInvocation(root string, m *RetranslationBatchManifest, r RetranslationBatchUnit, attempt int, rawPath string, raw []byte) (map[string][]byte, GlossaryArchiveReference, GlossaryArchiveReference, error) {
	dir := filepath.Join(root, "data/retranslation-runs", m.Locale, m.BatchID)
	request := GenerationBundleManifest{Attempt: attempt}
	if attempt > 1 {
		request.TaskKind = "translation-unit-retry"
		request.ExpectedOutputs = []GenerationBundleExpectedOutput{{BundlePath: "outputs/" + filepath.Base(r.InputPath)}}
	}
	path, err := repositoryRelativePath(root, generationImportReceiptPath(dir, request))
	if err != nil {
		return nil, GlossaryArchiveReference{}, GlossaryArchiveReference{}, err
	}
	b, err := readCompatibilityFile(root, path)
	if err != nil {
		return nil, GlossaryArchiveReference{}, GlossaryArchiveReference{}, err
	}
	var receipt GenerationResultBundleManifest
	if err := decodeStrictBundleJSON(b, &receipt); err != nil {
		return nil, GlossaryArchiveReference{}, GlossaryArchiveReference{}, err
	}
	archivePath := importedGenerationArchive(m.Locale, receipt.GenerationBundleSHA256)
	archive, err := readCompatibilityFile(root, archivePath)
	if err != nil {
		return nil, GlossaryArchiveReference{}, GlossaryArchiveReference{}, err
	}
	gm, files, err := readGenerationBundle(archive)
	if err != nil {
		return nil, GlossaryArchiveReference{}, GlossaryArchiveReference{}, err
	}
	if err := checkImportedGenerationIdentity(receipt, gm, archive, files); err != nil {
		return nil, GlossaryArchiveReference{}, GlossaryArchiveReference{}, err
	}
	manifest, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil || !bytes.Equal(manifest, files["formal/manifest.json"]) || sum(files["formal/inputs/"+filepath.Base(r.InputPath)]) != r.InputSHA256 || gm.Locale != m.Locale || gm.BatchID != m.BatchID || gm.Attempt != attempt || gm.UnitKind != r.UnitKind || (attempt > 1 && (gm.TaskKind != "translation-unit-retry" || len(gm.ExpectedOutputs) != 1)) {
		return nil, GlossaryArchiveReference{}, GlossaryArchiveReference{}, fmt.Errorf("whitespace attempt import/batch identity mismatch")
	}
	matched := 0
	for _, out := range gm.ExpectedOutputs {
		if out.UnitID == r.UnitID && out.InstallPath == rawPath {
			for _, f := range receipt.Outputs {
				if f == NewTransportBundleFile(out.BundlePath, "", raw) {
					matched++
				}
			}
		}
	}
	disk, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rawPath)))
	if err != nil || !bytes.Equal(disk, raw) || matched != 1 {
		return nil, GlossaryArchiveReference{}, GlossaryArchiveReference{}, fmt.Errorf("whitespace attempt original raw/provenance mismatch")
	}
	return files, GlossaryArchiveReference{Path: path, SHA256: sum(b)}, GlossaryArchiveReference{Path: archivePath, SHA256: sum(archive)}, nil
}

func whitespaceValidation(root string, unit *TranslationUnit, m *RetranslationBatchManifest, r RetranslationBatchUnit, data []byte) (*RetranslationValidation, error) {
	v, err := decodeRetranslationValidation(data, unit)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(r.InputPath)
	flat, ext := strings.TrimSuffix(name, filepath.Ext(name)), filepath.Ext(name)
	attempt, err := retryValidationAttemptForExtension(v.RawResponsePath, flat, ext)
	if err != nil || attempt != v.Attempt || v.SchemaVersion != retranslationProcessSchemaVersion || v.Locale != m.Locale || v.BatchID != m.BatchID || v.UnitID != r.UnitID || v.UnitKind != r.UnitKind || v.SourceSHA256 != r.SourceSHA256 || v.InputPath != r.InputPath || (v.Status != "validation_failed" && v.Status != "restore_failed") {
		return nil, fmt.Errorf("whitespace attempt validation binding mismatch")
	}
	dir := filepath.Join(root, "data/retranslation-runs", m.Locale, m.BatchID)
	for _, path := range []string{filepath.Join(dir, "validation", flat+".json"), filepath.Join(dir, "retries", flat, fmt.Sprintf("attempt-%03d-validation.json", attempt))} {
		b, err := os.ReadFile(path)
		if err == nil && bytes.Equal(b, data) {
			return v, nil
		}
	}
	return nil, fmt.Errorf("whitespace attempt validation is not exact current/immutable history evidence")
}

// Rebuild solely from exact frozen origin + source-owned coordinates. No
// historical candidate/PASS is selected through a mutable current pointer.
func buildWhitespaceAttempt(root string, c *Catalog, m *RetranslationBatchManifest, r RetranslationBatchUnit, attempt int, files map[string][]byte) (*attemptWhitespaceResponse, error) {
	u, err := c.Unit(r.UnitID)
	if err != nil {
		return nil, err
	}
	if attempt < 2 || m.TransportRecovery != nil || u.Kind != UnitKindPage || u.SourcePath != r.SourcePath || u.SourceSHA256 != r.SourceSHA256 || sum(u.Source) != r.SourceSHA256 {
		return nil, fmt.Errorf("whitespace attempt Unit/source mismatch")
	}
	source, err := readCompatibilityFile(root, u.SourcePath)
	if err != nil || validateSnapshotSourceIdentity(c, u, source) != nil || m.ArtifactEOF != retranslationArtifactEOFSingleLF {
		return nil, fmt.Errorf("whitespace attempt fresh source/EOF identity mismatch")
	}
	g, err := LoadGlossary(root, m.Locale)
	if err != nil {
		return nil, err
	}
	base, err := prepareTranslationUnitInput(u, g)
	if err != nil {
		return nil, err
	}
	input, err := readCompatibilityFile(root, filepath.ToSlash(filepath.Join("data/retranslation-runs", m.Locale, m.BatchID, r.InputPath)))
	if err != nil || sum(input) != r.InputSHA256 || !bytes.Equal(input, canonicalizeRetranslationArtifactEOF([]byte(base.Text))) || len(base.Tokens) != r.ProtectedTokenCount {
		return nil, fmt.Errorf("whitespace attempt formal input/decoder mismatch")
	}
	inspector, ok := base.Adapter.(StructuralWhitespaceInspector)
	if !ok || r.RepresentationBridge != nil || r.GenerationOrigin != nil || r.RevisionFeedbackSource == CurrentValidationRevisionFeedback || r.RevisionFeedbackSource == "surface_review" {
		return nil, fmt.Errorf("protected-response representation cannot express this validation repair: unsupported original decoder")
	}
	for _, name := range whitespaceFrozenMembers {
		if _, ok := files[attemptWhitespacePrefix+name]; !ok {
			return nil, fmt.Errorf("whitespace attempt missing %s", name)
		}
	}
	originRaw, originCandidate := files[attemptWhitespacePrefix+whitespaceFrozenMembers[0]], files[attemptWhitespacePrefix+whitespaceFrozenMembers[1]]
	originData, previousRaw, previousData := files[attemptWhitespacePrefix+whitespaceFrozenMembers[2]], files[attemptWhitespacePrefix+whitespaceFrozenMembers[3]], files[attemptWhitespacePrefix+whitespaceFrozenMembers[4]]
	origin, err := whitespaceValidation(root, u, m, r, originData)
	if err != nil {
		return nil, err
	}
	previous, err := whitespaceValidation(root, u, m, r, previousData)
	if err != nil {
		return nil, err
	}
	if previous.Attempt+1 != attempt || origin.Attempt > previous.Attempt || origin.Status != "validation_failed" || origin.CandidatePath != "candidates/"+filepath.Base(r.InputPath) {
		return nil, fmt.Errorf("whitespace codec wrong attempt/origin")
	}
	_, oi, oa, err := whitespaceInvocation(root, m, r, origin.Attempt, origin.RawResponsePath, originRaw)
	if err != nil {
		return nil, err
	}
	previousFiles, pi, pa, err := whitespaceInvocation(root, m, r, previous.Attempt, previous.RawResponsePath, previousRaw)
	if err != nil {
		return nil, err
	}
	if origin.Attempt != previous.Attempt {
		var parent attemptWhitespaceCodec
		if err := decodeStrictBundleJSON(previousFiles[attemptWhitespaceMember], &parent); err != nil || parent.Schema != attemptWhitespaceSchema || parent.Identity != parent.identity() || parent.Locale != m.Locale || parent.BatchID != m.BatchID || parent.Attempt != previous.Attempt || parent.OriginalRawSHA256 != sum(originRaw) || parent.OriginalCandidateSHA256 != sum(originCandidate) || parent.OriginalValidationSHA256 != sum(originData) || !reflect.DeepEqual(parent.Record, r) {
			return nil, fmt.Errorf("whitespace attempt cannot change its immutable origin")
		}
		if previous.Status == "validation_failed" && !strings.HasPrefix(previous.Error, "candidate has trailing whitespace at line ") {
			return nil, fmt.Errorf("whitespace-only codec cannot repair a non-whitespace validation failure")
		}
	}
	restored, failures := base.restore(string(originRaw))
	if len(failures) != 0 || !bytes.Equal(canonicalizeRetranslationArtifactEOF([]byte(restored)), originCandidate) {
		return nil, fmt.Errorf("whitespace codec original decoder/candidate mismatch")
	}
	if err := validateRetranslationCandidateArtifact(originCandidate); err == nil || err.Error() != origin.Error {
		return nil, fmt.Errorf("whitespace codec original failure mismatch")
	}
	repairs, representation, err := inspector.InspectStructuralWhitespace(originRaw)
	if err != nil {
		return nil, err
	}
	manifestData, err := os.ReadFile(filepath.Join(root, "data/retranslation-runs", m.Locale, m.BatchID, "manifest.json"))
	if err != nil {
		return nil, err
	}
	glossary, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(m.Locale))
	if err != nil {
		return nil, err
	}
	// Imported historical readers use the frozen invocation glossary; current
	// exporter/import/process eligibility separately checks today's glossary.
	if frozen, ok := files["formal/glossary.yaml"]; ok {
		glossary = frozen
	}
	codec := attemptWhitespaceCodec{Schema: attemptWhitespaceSchema, Locale: m.Locale, BatchID: m.BatchID, Attempt: attempt, Record: r, ManifestSHA256: sum(manifestData), GlossarySHA256: sum(glossary), OriginalRepresentation: representation, OriginImport: oi, OriginArchive: oa, PreviousImport: pi, PreviousArchive: pa, OriginalRawSHA256: sum(originRaw), OriginalCandidateSHA256: sum(originCandidate), OriginalValidationSHA256: sum(originData), PreviousRawSHA256: sum(previousRaw), PreviousValidationSHA256: sum(previousData), Repairs: repairs}
	var expected bytes.Buffer
	pos := 0
	for _, repair := range repairs {
		expected.Write(originCandidate[pos:repair.Start])
		pos = repair.End
	}
	expected.Write(originCandidate[pos:])
	codec.RepairedCandidateSHA256 = sum(expected.Bytes())
	codec.Identity = codec.identity()
	return whitespaceResponse(codec, originCandidate, base.Adapter)
}

func whitespaceAttemptFromFiles(root string, c *Catalog, m *RetranslationBatchManifest, r RetranslationBatchUnit, attempt int, files map[string][]byte) (*attemptWhitespaceResponse, error) {
	b, present := files[attemptWhitespaceMember]
	if !present {
		return nil, nil
	}
	var frozen attemptWhitespaceCodec
	if err := decodeStrictBundleJSON(b, &frozen); err != nil {
		return nil, err
	}
	if frozen.Schema != attemptWhitespaceSchema || frozen.Locale != m.Locale || frozen.BatchID != m.BatchID || frozen.Attempt != attempt || !reflect.DeepEqual(frozen.Record, r) || frozen.Identity != frozen.identity() {
		return nil, fmt.Errorf("whitespace attempt codec identity/scope mismatch")
	}
	if !bytes.Equal(b, mustJSON(frozen)) {
		return nil, fmt.Errorf("non-canonical whitespace attempt codec")
	}
	p, err := buildWhitespaceAttempt(root, c, m, r, attempt, files)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(mustJSON(p.codec), mustJSON(frozen)) {
		return nil, fmt.Errorf("whitespace attempt codec source/coordinate/evidence drift")
	}
	text, _, _ := p.ProtectedInput(nil)
	if !bytes.Equal(files[attemptWhitespacePrefix+"response.article"], canonicalizeRetranslationArtifactEOF([]byte(text))) {
		return nil, fmt.Errorf("whitespace attempt supplied response mismatch")
	}
	allowed := map[string]bool{attemptWhitespaceMember: true, attemptWhitespacePrefix + "response.article": true}
	for _, name := range whitespaceFrozenMembers {
		allowed[attemptWhitespacePrefix+name] = true
	}
	for path := range files {
		if strings.HasPrefix(path, attemptWhitespacePrefix) && !allowed[path] {
			return nil, fmt.Errorf("unexpected whitespace attempt member")
		}
	}
	return p, nil
}

func importedWhitespaceAttempt(root string, c *Catalog, m *RetranslationBatchManifest, r RetranslationBatchUnit, attempt int, rawPath string, raw []byte) (*attemptWhitespaceResponse, string, error) {
	if attempt <= 1 {
		return nil, "", nil
	}
	files, receipt, archive, err := whitespaceInvocation(root, m, r, attempt, rawPath, raw)
	if os.IsNotExist(err) && !bytes.Contains(raw, []byte("⟪GTI18N_WS1_")) {
		probe := GenerationBundleManifest{TaskKind: "translation-unit-retry", Attempt: attempt, ExpectedOutputs: []GenerationBundleExpectedOutput{{BundlePath: "outputs/" + filepath.Base(r.InputPath)}}}
		_, missing := os.Lstat(generationImportReceiptPath(filepath.Join(root, "data/retranslation-runs", m.Locale, m.BatchID), probe))
		if os.IsNotExist(missing) {
			return nil, "", nil
		} // Only genuinely manual/legacy retries.
	}
	if err != nil {
		return nil, "", err
	}
	p, err := whitespaceAttemptFromFiles(root, c, m, r, attempt, files)
	if err != nil {
		return nil, "", err
	}
	if p == nil {
		if bytes.Contains(raw, []byte("⟪GTI18N_WS1_")) {
			return nil, "", fmt.Errorf("whitespace codec requires exact imported attempt")
		}
		return nil, "", nil
	}
	return p, receipt.SHA256 + "/" + archive.SHA256, nil
}

func prepareWhitespaceRetry(root string, c *Catalog, m *RetranslationBatchManifest, r RetranslationBatchUnit, v *RetranslationValidation, validation []byte, enabled bool) ([]generationBundleInput, error) {
	u, err := c.Unit(r.UnitID)
	if err != nil {
		return nil, err
	}
	if _, err := whitespaceValidation(root, u, m, r, validation); err != nil {
		return nil, err
	}
	dir := filepath.Join(root, "data/retranslation-runs", m.Locale, m.BatchID)
	raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(v.RawResponsePath)))
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	if v.Attempt > 1 {
		prior, _, _, err := whitespaceInvocation(root, m, r, v.Attempt, v.RawResponsePath, raw)
		if err != nil {
			probe := GenerationBundleManifest{TaskKind: "translation-unit-retry", Attempt: v.Attempt, ExpectedOutputs: []GenerationBundleExpectedOutput{{BundlePath: "outputs/" + filepath.Base(r.InputPath)}}}
			_, missing := os.Lstat(generationImportReceiptPath(dir, probe))
			if !os.IsNotExist(err) || !os.IsNotExist(missing) {
				return nil, err
			}
		}
		if _, present := prior[attemptWhitespaceMember]; present {
			if !enabled {
				return nil, fmt.Errorf("protected-response representation cannot express this validation repair; exact attempt codec requires --protected-whitespace-repair")
			}
			if v.Status == "validation_failed" && !strings.HasPrefix(v.Error, "candidate has trailing whitespace at line ") {
				return nil, fmt.Errorf("whitespace-only codec cannot repair a non-whitespace validation failure")
			}
			if _, err := whitespaceAttemptFromFiles(root, c, m, r, v.Attempt, prior); err != nil {
				return nil, err
			}
			for _, name := range whitespaceFrozenMembers[:3] {
				files[attemptWhitespacePrefix+name] = prior[attemptWhitespacePrefix+name]
			}
		}
	}
	if len(files) == 0 {
		if v.Status != "validation_failed" || !strings.HasPrefix(v.Error, "candidate has trailing whitespace at line ") {
			if enabled {
				return nil, fmt.Errorf("whitespace repair requires exact protected structural-whitespace failure")
			}
			return nil, nil
		}
		inspector, ok := u.Adapter.(StructuralWhitespaceInspector)
		if !ok {
			if enabled || u.Kind == UnitKindPage && u.Adapter != nil {
				return nil, fmt.Errorf("source-owned structural whitespace proof unavailable")
			}
			return nil, nil
		}
		repairs, _, err := inspector.InspectStructuralWhitespace(raw)
		if err != nil {
			return nil, fmt.Errorf("protected-response representation cannot express this validation repair: %w", err)
		}
		blocked := false
		for _, repair := range repairs {
			blocked = blocked || repair.OpaqueLine
		}
		if !blocked {
			if enabled {
				return nil, fmt.Errorf("no protected-only structural-whitespace blocker")
			}
			return nil, nil
		}
		if !enabled {
			return nil, fmt.Errorf("protected-response representation cannot express this validation repair; use explicit --protected-whitespace-repair after contract gate")
		}
		if v.CandidatePath != "candidates/"+filepath.Base(r.InputPath) {
			return nil, fmt.Errorf("whitespace repair candidate path mismatch")
		}
		candidate, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(v.CandidatePath)))
		if err != nil {
			return nil, err
		}
		files[attemptWhitespacePrefix+whitespaceFrozenMembers[0]], files[attemptWhitespacePrefix+whitespaceFrozenMembers[1]], files[attemptWhitespacePrefix+whitespaceFrozenMembers[2]] = raw, candidate, validation
	}
	files[attemptWhitespacePrefix+whitespaceFrozenMembers[3]], files[attemptWhitespacePrefix+whitespaceFrozenMembers[4]] = raw, validation
	p, err := buildWhitespaceAttempt(root, c, m, r, v.Attempt+1, files)
	if err != nil {
		return nil, err
	}
	text, _, _ := p.ProtectedInput(nil)
	inputs := []generationBundleInput{{attemptWhitespaceMember, "", mustJSON(p.codec)}, {attemptWhitespacePrefix + "response.article", "", canonicalizeRetranslationArtifactEOF([]byte(text))}}
	for _, name := range whitespaceFrozenMembers {
		inputs = append(inputs, generationBundleInput{attemptWhitespacePrefix + name, "", files[attemptWhitespacePrefix+name]})
	}
	return inputs, nil
}
