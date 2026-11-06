package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

const SameUnitBridgeAuthoritySource = "quality-check-transport"

// This adapter operation composes historical source-owned mappings. It is not
// Protect(candidate), a formal replacement, a language result or an A rating.
// Unsupported kinds/adapters fail closed.
type SameUnitBridgeAdapter interface {
	BridgeContextReferences(root string) ([]GlossaryArchiveReference, error)
	BridgeRestrictedRevision(ordinaryRaw, previousCandidate, reviewedCandidate, restrictedRaw []byte, transport *CommentRevisionTransport) (TranslationUnitAdapter, []byte, error)
}

// A separate source-owned construction operation, never Surface semantic
// authorization. The caller must verify the sealed Surface decoder and raw.
type SurfaceSameUnitBridgeAdapter interface {
	BridgeSurfaceRevision(TranslationUnitAdapter, []byte, []byte) (TranslationUnitAdapter, []byte, []byte, error)
}
type SameUnitBridgeOptions struct {
	Locale             string                   `json:"locale"`
	ContentPackage     string                   `json:"content_package"`
	BridgeID           string                   `json:"bridge_id"`
	PreviousSnapshotID string                   `json:"previous_snapshot_id"`
	SnapshotSHA256     string                   `json:"previous_snapshot_sha256"`
	ReviewerEvidence   GlossaryArchiveReference `json:"reviewer_evidence"`
	UnitIDs            []string                 `json:"unit_ids"`
	Record             bool                     `json:"-"`
}
type SameUnitBridgeBinding struct {
	ProtectedInput            GlossaryArchiveReference `json:"protected_input"`
	OriginalManifest          GlossaryArchiveReference `json:"original_manifest"`
	OriginalResult            GlossaryArchiveReference `json:"original_result"`
	UnitID                    string                   `json:"unit_id"`
	FormalInputSHA256         string                   `json:"formal_input_sha256"`
	Raw                       GlossaryArchiveReference `json:"original_raw"`
	Import                    GlossaryArchiveReference `json:"original_import"`
	Archive                   GlossaryArchiveReference `json:"original_archive"`
	TransportMember           string                   `json:"transport_member"`
	TransportSHA256           string                   `json:"transport_sha256"`
	OrdinaryAncestorSHA256    string                   `json:"ordinary_ancestor_raw_sha256"`
	OldRepresentationIdentity string                   `json:"old_representation_identity"`
	NewRepresentationIdentity string                   `json:"new_representation_identity"`
	Template                  string                   `json:"protected_template"`
	TemplateSHA256            string                   `json:"protected_template_sha256"`
	CurrentAuthority          string                   `json:"current_adapter_validator_identity"`
}
type SameUnitBridgeAuthority struct {
	ProjectionImplementationSHA256 string                     `json:"projection_implementation_sha256"`
	Options                        SameUnitBridgeOptions      `json:"options"`
	PackageIdentity                string                     `json:"package_identity"`
	OrderedUnits                   []string                   `json:"complete_ordered_unit_ids"`
	Dependencies                   []GlossaryArchiveReference `json:"dependencies"`
	FrozenDependencies             []GlossaryArchiveReference `json:"frozen_dependencies"`
	Bindings                       []SameUnitBridgeBinding    `json:"bindings"`
	Identity                       string                     `json:"identity_sha256"`
}
type SameUnitBridgePlan struct {
	Receipt       *QualityCheckSurfaceReopen `json:"receipt"`
	ReceiptPath   string                     `json:"receipt_path"`
	ReceiptSHA256 string                     `json:"receipt_sha256"`
	Recorded      bool                       `json:"recorded"`
	Mutation      string                     `json:"mutation"`
}

func sameUnitBridgeIdentity(r *QualityCheckSurfaceReopen) string {
	copy := *r
	a := *r.SameUnitBridge
	a.Identity = ""
	copy.SameUnitBridge = &a
	return sum(mustJSON(copy))
}
func bridgeReceiptBytes(r *QualityCheckSurfaceReopen) []byte { return append(mustJSON(r), '\n') }

// Inspect/record use the same deterministic builder. Record is write-once;
// duplicate IDs (including identical records) are rejected, never overwritten.
func PrepareSameUnitRepresentationBridge(root string, c *Catalog, o SameUnitBridgeOptions) (*SameUnitBridgePlan, error) {
	r, _, err := buildSameUnitBridge(root, c, o)
	if err != nil {
		return nil, err
	}
	path := filepath.ToSlash(filepath.Join("data/quality-check-reopens", o.Locale, o.BridgeID+".json"))
	plan := &SameUnitBridgePlan{Receipt: r, ReceiptPath: path, ReceiptSHA256: sum(bridgeReceiptBytes(r)), Mutation: "none"}
	if !o.Record {
		return plan, nil
	}
	if _, err := os.Lstat(filepath.Join(root, path)); !os.IsNotExist(err) {
		return nil, fmt.Errorf("bridge receipt already exists or cannot be inspected")
	}
	// Fresh complete construction before publishing, not just a prior PASS.
	fresh, _, err := buildSameUnitBridge(root, c, o)
	if err != nil || !bytes.Equal(bridgeReceiptBytes(r), bridgeReceiptBytes(fresh)) {
		return nil, fmt.Errorf("bridge authority changed before record: %v", err)
	}
	if err := verifySameUnitBridgeDependencies(root, r); err != nil {
		return nil, err
	}
	destination, err := compatibilityPath(root, path, true)
	if err != nil {
		return nil, err
	}
	if err := writeNewGenerationFile(destination, bridgeReceiptBytes(r)); err != nil {
		return nil, err
	}
	plan.Recorded, plan.Mutation = true, "known"
	return plan, nil
}
func buildSameUnitBridge(root string, c *Catalog, o SameUnitBridgeOptions) (*QualityCheckSurfaceReopen, map[string]protectedTranslation, error) {
	if ValidateLocaleName(o.Locale) != nil || !reviewIDPattern.MatchString(o.BridgeID) || validateSnapshotID(o.PreviousSnapshotID) != nil || !validSHA256(o.SnapshotSHA256) || len(o.UnitIDs) < 1 || len(o.UnitIDs) > MaxRetranslationExportLimit {
		return nil, nil, fmt.Errorf("explicit bridge/package/Snapshot/exact Unit scope required")
	}
	scoped, err := SelectCurrentValidationReopenPackage(c, o.ContentPackage)
	if err != nil {
		return nil, nil, err
	}
	c = scoped
	_, pkgSHA, _, _, err := translationAuthorizationScope(c)
	if err != nil {
		return nil, nil, err
	}
	if err := RequireCurrentGlossaryReview(root, o.Locale); err != nil {
		return nil, nil, err
	}
	g, err := LoadGlossary(root, o.Locale)
	if err != nil {
		return nil, nil, err
	}
	refs := map[string]string{}
	read := func(path string) ([]byte, error) {
		b, e := readSnapshotReferencedFile(root, path)
		if e != nil {
			return nil, e
		}
		h := sum(b)
		if old, ok := refs[path]; ok && old != h {
			return nil, fmt.Errorf("bridge dependency changed while reading: %s", path)
		}
		refs[path] = h
		return b, nil
	}
	snapshotPath := "data/quality-check-snapshots/" + o.Locale + "/" + o.PreviousSnapshotID + "/manifest.json"
	b, err := read(snapshotPath)
	if err != nil || sum(b) != o.SnapshotSHA256 {
		return nil, nil, fmt.Errorf("bridge Snapshot identity mismatch")
	}
	if !validSHA256(o.ReviewerEvidence.SHA256) {
		return nil, nil, fmt.Errorf("explicit Reviewer evidence path/SHA required")
	}
	b, err = read(o.ReviewerEvidence.Path)
	if err != nil || sum(b) != o.ReviewerEvidence.SHA256 {
		return nil, nil, fmt.Errorf("bridge Reviewer evidence identity mismatch")
	}
	ev := newQualityCheckEvaluation(root, o.Locale, c)
	snapshot, results, err := ev.loadEffectiveQualityCheckResults(o.PreviousSnapshotID, map[string]bool{})
	if err != nil {
		return nil, nil, err
	}
	ordered, _, _, err := localeWorkflowUnitList(c)
	if err != nil {
		return nil, nil, err
	}
	if len(snapshot.Units) != len(ordered) || len(results) != len(ordered) {
		return nil, nil, fmt.Errorf("bridge complete package/QC scope mismatch")
	}
	a := &SameUnitBridgeAuthority{Options: o, PackageIdentity: pkgSHA, Bindings: []SameUnitBridgeBinding{}}
	a.Options.Record = false
	a.Options.UnitIDs = nil
	selected := map[string]bool{}
	for _, id := range o.UnitIDs {
		if selected[id] {
			return nil, nil, fmt.Errorf("duplicate bridge Unit")
		}
		selected[id] = true
	}
	r := &QualityCheckSurfaceReopen{SchemaVersion: qualityCheckSurfaceReopenSchemaVersion, EvidenceType: qualityCheckSurfaceReopenEvidenceType, AuthoritySource: SameUnitBridgeAuthoritySource, Locale: o.Locale, ReopenID: o.BridgeID, PreviousSnapshotID: o.PreviousSnapshotID, SameUnitBridge: a}
	var records []RetranslationBatchUnit
	for i, s := range snapshot.Units {
		if s.UnitID != ordered[i].ID {
			return nil, nil, fmt.Errorf("bridge ordered package identity mismatch")
		}
		a.OrderedUnits = append(a.OrderedUnits, s.UnitID)
		if !selected[s.UnitID] {
			continue
		}
		delete(selected, s.UnitID)
		q, ok := results[s.UnitID]
		if !ok || q.rating == "A" || q.finding == "" || q.rubric != TranslationQualityRubric || !qualityCheckSnapshotIdentityMatches(s, q.unit) || s.UnitKind != UnitKindPage {
			return nil, nil, fmt.Errorf("bridge requires exact reviewed Page B/C/D finding")
		}
		if len(r.Units) > 0 && r.Units[0].Snapshot.UnitKind != s.UnitKind {
			return nil, nil, fmt.Errorf("mixed bridge kinds")
		}
		r.Units = append(r.Units, QualityCheckSurfaceReopenUnit{Index: s.Index, UnitID: s.UnitID, Snapshot: s, PreviousRating: q.rating, Finding: q.finding})
		e, err := ev.readSnapshotUnitRepositoryEvidence(o.PreviousSnapshotID, s)
		if err != nil {
			return nil, nil, err
		}
		protected, err := prepareTranslationUnitInput(ordered[i], g)
		if err != nil || e.record.InputPath != "inputs/"+retranslationUnitInputName(ordered[i]) || sum(canonicalizeRetranslationArtifactEOF([]byte(protected.Text))) != e.record.InputSHA256 {
			return nil, nil, fmt.Errorf("bridge formal input delta is not zero: %v", err)
		}
		saved, err := read(filepath.ToSlash(filepath.Join("data/retranslation-runs", o.Locale, s.SelectedBatchID, e.record.InputPath)))
		if err != nil || sum(saved) != e.record.InputSHA256 {
			return nil, nil, fmt.Errorf("bridge saved protected input drift")
		}
		records = append(records, RetranslationBatchUnit{UnitID: s.UnitID, UnitKind: s.UnitKind, SourcePath: s.SourcePath, SourceSHA256: s.SourceSHA256, InputPath: e.record.InputPath, InputSHA256: e.record.InputSHA256, ProtectedTokenCount: e.record.ProtectedTokenCount, PreviousSnapshotID: o.PreviousSnapshotID, RevisionFeedbackSource: "quality_check", PreviousRating: q.rating, PreviousFinding: q.finding})
		a.Options.UnitIDs = append(a.Options.UnitIDs, s.UnitID)
	}
	if len(selected) > 0 || len(r.Units) != len(o.UnitIDs) {
		return nil, nil, fmt.Errorf("bridge exact Unit set mismatch")
	}
	r.UnitCount = len(r.Units)
	// Reuse canonical predecessor/provenance/QC/current-validation readers. The
	// capability guard is intentionally not run until a recorded bridge exists.
	inputs, err := revisionGenerationInputs(root, c, o.Locale, records)
	if err != nil {
		return nil, nil, err
	}
	for _, in := range inputs {
		if in.repositoryPath != "" {
			b, err := read(in.repositoryPath)
			if err != nil || !bytes.Equal(b, in.data) {
				return nil, nil, fmt.Errorf("bridge predecessor changed: %s", in.repositoryPath)
			}
		}
	}
	qcRefs, err := qualityCheckFinalizationInputs(root, o.Locale, o.PreviousSnapshotID, map[string]bool{})
	if err != nil {
		return nil, nil, err
	}
	for _, ref := range qcRefs {
		b, e := read(ref.Path)
		if e != nil || (ref.SHA256 != "" && sum(b) != ref.SHA256) {
			return nil, nil, fmt.Errorf("bridge QC dependency stale: %s", ref.Path)
		}
	}
	// Historical bindings exclude mutable command policy/package metadata. New
	// exports still require the complete current dependency set below.
	frozen := map[string]string{}
	for path, sha := range refs {
		frozen[path] = sha
	}
	readFrozen := func(path string) ([]byte, error) {
		b, err := read(path)
		if err == nil {
			frozen[path] = sum(b)
		}
		return b, err
	}
	review, err := RequireUnifiedGlossaryReview(root, o.Locale)
	if err != nil {
		return nil, nil, err
	}
	paths := append([]string{}, translationUnitGenerationAuthorityPaths...)
	paths = append(paths, GlossaryCorpusPath, glossaryReviewGlossaryPath(o.Locale), review.Path, "internal/i18n/same_unit_bridge.go", "internal/sitecontent/same_unit_bridge.go", "internal/i18n/surface_same_unit_bridge.go", "internal/sitecontent/surface_same_unit_bridge.go")
	for _, path := range paths {
		if _, err := read(path); err != nil {
			return nil, nil, err
		}
	}
	a.ProjectionImplementationSHA256, err = TranslationMachineAuthority(root, []string{"internal/i18n/same_unit_bridge.go", "internal/sitecontent/same_unit_bridge.go", "internal/i18n/surface_same_unit_bridge.go", "internal/sitecontent/surface_same_unit_bridge.go", "internal/sitecontent/surface_transport.go", "internal/sitecontent/translation_adapter.go", "internal/sitecontent/formal_page.go"})
	if err != nil {
		return nil, nil, err
	}
	decoders := map[string]protectedTranslation{}
	for _, reopened := range r.Units {
		s := reopened.Snapshot
		u, err := c.Unit(s.UnitID)
		if err != nil {
			return nil, nil, err
		}
		factory, ok := u.Adapter.(SameUnitBridgeAdapter)
		if !ok {
			return nil, nil, fmt.Errorf("Page adapter has no deterministic restricted bridge operation")
		}
		contextRefs, err := factory.BridgeContextReferences(root)
		if err != nil || len(contextRefs) == 0 {
			return nil, nil, fmt.Errorf("bridge package context authority unavailable: %v", err)
		}
		for _, ref := range contextRefs {
			b, err := read(ref.Path)
			if err != nil || sum(b) != ref.SHA256 {
				return nil, nil, fmt.Errorf("bridge package context drift")
			}
		}
		beforeAuthority, err := currentTranslationAuthority(root, u, o.Locale)
		if err != nil {
			return nil, nil, err
		}
		if b, err := readFrozen(u.SourcePath); err != nil || validateSnapshotSourceIdentity(c, u, b) != nil {
			return nil, nil, fmt.Errorf("bridge current source changed")
		}
		e, err := ev.readSnapshotUnitRepositoryEvidence(o.PreviousSnapshotID, s)
		if err != nil {
			return nil, nil, err
		}
		batchBase, err := repositoryRelativePath(root, e.batchDir)
		if err != nil {
			return nil, nil, err
		}
		inputPath := batchBase + "/" + e.record.InputPath
		inputBytes, err := readFrozen(inputPath)
		if err != nil || sum(inputBytes) != e.record.InputSHA256 {
			return nil, nil, fmt.Errorf("bridge saved protected input drift")
		}
		manifestBytes, err := readFrozen(batchBase + "/manifest.json")
		if err != nil {
			return nil, nil, err
		}
		resultBytes, err := readFrozen(batchBase + "/result.json")
		if err != nil {
			return nil, nil, err
		}
		surface := e.record.RevisionFeedbackSource == "surface_review" && hasSurfaceTransport(root, o.Locale, e.record.RevisionAuthorizationID)
		if (!surface && e.record.RevisionFeedbackSource != CurrentValidationRevisionFeedback) || e.record.GenerationOrigin != nil {
			return nil, nil, fmt.Errorf("bridge requires passed restricted current-validation or sealed Surface predecessor")
		}
		rawPath := filepath.ToSlash(filepath.Join("data/retranslation-runs", o.Locale, s.SelectedBatchID, e.validation.RawResponsePath))
		raw, err := readFrozen(rawPath)
		if err != nil {
			return nil, nil, err
		}
		probe := GenerationBundleManifest{Attempt: s.Attempt}
		if s.Attempt > 1 {
			probe.TaskKind = "translation-unit-retry"
			probe.ExpectedOutputs = []GenerationBundleExpectedOutput{{BundlePath: "outputs/" + filepath.Base(e.record.InputPath)}}
		}
		importPath, err := repositoryRelativePath(root, generationImportReceiptPath(e.batchDir, probe))
		if err != nil {
			return nil, nil, err
		}
		imported, err := readFrozen(importPath)
		if err != nil {
			return nil, nil, err
		}
		var receipt GenerationResultBundleManifest
		if err := decodeStrictCourseSourceDescriptionReviewJSON(imported, &receipt); err != nil {
			return nil, nil, err
		}
		archivePath := importedGenerationArchive(o.Locale, receipt.GenerationBundleSHA256)
		archive, err := readFrozen(archivePath)
		if err != nil {
			return nil, nil, err
		}
		original, files, err := readGenerationBundle(archive)
		if err != nil {
			return nil, nil, err
		}
		if err := checkImportedGenerationIdentity(receipt, original, archive, files); err != nil {
			return nil, nil, err
		}
		matched := false
		for _, out := range original.ExpectedOutputs {
			if out.UnitID == u.ID && out.InstallPath == e.validation.RawResponsePath {
				for _, v := range receipt.Outputs {
					if v.BundlePath == out.BundlePath && v.SHA256 == sum(raw) && v.Size == int64(len(raw)) {
						matched = true
					}
				}
			}
		}
		if !matched || original.Locale != o.Locale || original.BatchID != s.SelectedBatchID || original.Attempt != s.Attempt {
			return nil, nil, fmt.Errorf("bridge restricted import/attempt mismatch")
		}
		name := filepath.Base(e.record.InputPath)
		member := commentTransportPath(e.record)
		historical := files["formal/candidates/"+name]
		ancestor := files["formal/historical-protected-candidates/"+name]
		if len(historical) == 0 || len(ancestor) == 0 {
			return nil, nil, fmt.Errorf("bridge lacks exact ordinary historical ancestor")
		}
		normal, err := prepareTranslationUnitInput(u, g)
		if err != nil {
			return nil, nil, err
		}
		ordinary, failures := normal.restore(string(raw))
		if len(failures) == 0 && bytes.Equal(canonicalizeRetranslationArtifactEOF([]byte(ordinary)), e.candidate) {
			return nil, nil, fmt.Errorf("ordinary predecessor requires no bridge")
		}
		input := canonicalizeRetranslationArtifactEOF([]byte(normal.Text))
		if sum(input) != e.record.InputSHA256 {
			return nil, nil, fmt.Errorf("bridge formal input delta is not zero")
		}
		var adapter TranslationUnitAdapter
		var template []byte
		if surface {
			member = surfaceBridgeMember(u.ID)
			adapter, template, ancestor, err = surfaceSameUnitBridgeOutput(root, c, o.Locale, e.record, raw, e.candidate, files, readFrozen)
			if err != nil {
				return nil, nil, err
			}
		} else {
			cv, err := readCurrentQualityCheckSurfaceReopen(root, c, o.Locale, e.record.RevisionAuthorizationID)
			if err != nil {
				return nil, nil, err
			}
			cvRef := cv.revisionEvidence()
			cvBytes, err := readFrozen(cvRef.Path)
			if err != nil || sum(cvBytes) != cvRef.SHA256 {
				return nil, nil, fmt.Errorf("bridge restricted reopen drift")
			}
			assessment, err := decodeCurrentValidationAssessment(cv.CurrentValidation)
			if err != nil {
				return nil, nil, err
			}
			for _, ref := range assessment.Historical.References {
				b, err := readFrozen(ref.Path)
				if err != nil || sum(b) != ref.SHA256 {
					return nil, nil, fmt.Errorf("bridge frozen historical reference drift: %s", ref.Path)
				}
			}
			p, err := buildCommentRevisionTransport(root, c, g, o.Locale, u, cv, historical)
			if err != nil || p == nil {
				return nil, nil, fmt.Errorf("bridge cannot reconstruct restricted authority: %v", err)
			}
			// The archived mapping is the model's original authority, not a newly
			// guessed projection. Exact canonical equality is mandatory.
			if !bytes.Equal(mustJSON(p), files[member]) {
				return nil, nil, fmt.Errorf("bridge old transport mapping drift")
			}
			restored, err := p.Restore(string(raw))
			if err != nil || !bytes.Equal(canonicalizeRetranslationArtifactEOF(restored), e.candidate) {
				return nil, nil, fmt.Errorf("bridge old representation restore mismatch: %v", err)
			}
			adapter, template, err = factory.BridgeRestrictedRevision(ancestor, historical, e.candidate, raw, p)
			if err != nil {
				return nil, nil, err
			}
		}
		got, err := adapter.Restore(string(template))
		if err != nil || !bytes.Equal(canonicalizeRetranslationArtifactEOF(got), e.candidate) {
			return nil, nil, fmt.Errorf("bridge reviewed bytes changed: %v", err)
		}
		if err := validateTranslationUnitWithGlossary(root, c, u, o.Locale, e.candidate, g); err != nil {
			return nil, nil, fmt.Errorf("bridge current validation failed: %v", err)
		}
		text, tokens, err := adapter.ProtectedInput(g)
		if err != nil {
			return nil, nil, err
		}
		decoders[u.ID] = protectedTranslation{Text: text, Tokens: tokens, Adapter: adapter}
		authority, err := currentTranslationAuthority(root, u, o.Locale)
		if err != nil {
			return nil, nil, err
		}
		if authority != beforeAuthority {
			return nil, nil, fmt.Errorf("bridge adapter/validator changed during construction")
		}
		binding := SameUnitBridgeBinding{
			ProtectedInput: GlossaryArchiveReference{inputPath, sum(inputBytes)}, OriginalManifest: GlossaryArchiveReference{batchBase + "/manifest.json", sum(manifestBytes)}, OriginalResult: GlossaryArchiveReference{batchBase + "/result.json", sum(resultBytes)}, UnitID: u.ID, FormalInputSHA256: sum(input), Raw: GlossaryArchiveReference{rawPath, sum(raw)}, Import: GlossaryArchiveReference{importPath, sum(imported)}, Archive: GlossaryArchiveReference{archivePath, sum(archive)}, TransportMember: member, TransportSHA256: sum(files[member]), OrdinaryAncestorSHA256: sum(ancestor), Template: string(template), TemplateSHA256: sum(template), CurrentAuthority: authority}
		binding.OldRepresentationIdentity = sum(mustJSON([]string{u.ID, binding.Raw.SHA256, binding.TransportSHA256, s.CandidateSHA256}))
		binding.NewRepresentationIdentity = sum(mustJSON([]string{a.ProjectionImplementationSHA256, binding.OldRepresentationIdentity, binding.OrdinaryAncestorSHA256, binding.TemplateSHA256, authority, sum([]byte(text)), sum(mustJSON(tokens))}))
		a.Bindings = append(a.Bindings, binding)
	}
	for path, sha := range frozen {
		a.FrozenDependencies = append(a.FrozenDependencies, GlossaryArchiveReference{path, sha})
	}
	sort.Slice(a.FrozenDependencies, func(i, j int) bool { return a.FrozenDependencies[i].Path < a.FrozenDependencies[j].Path })
	for path, sha := range refs {
		a.Dependencies = append(a.Dependencies, GlossaryArchiveReference{path, sha})
	}
	sort.Slice(a.Dependencies, func(i, j int) bool { return a.Dependencies[i].Path < a.Dependencies[j].Path })
	for _, binding := range a.Bindings {
		u, err := c.Unit(binding.UnitID)
		if err != nil {
			return nil, nil, err
		}
		identity, err := currentTranslationAuthority(root, u, o.Locale)
		if err != nil || identity != binding.CurrentAuthority {
			return nil, nil, fmt.Errorf("bridge current authority changed before output")
		}
	}
	a.Identity = sameUnitBridgeIdentity(r)
	if err := verifySameUnitBridgeDependencies(root, r); err != nil {
		return nil, nil, err
	}
	return r, decoders, nil
}
func verifySameUnitBridgeDependencies(root string, r *QualityCheckSurfaceReopen) error {
	for _, ref := range r.SameUnitBridge.Dependencies {
		b, err := readSnapshotReferencedFile(root, ref.Path)
		if err != nil || sum(b) != ref.SHA256 {
			return fmt.Errorf("bridge dependency drift: %s", ref.Path)
		}
	}
	return nil
}
func readSameUnitBridge(root string, c *Catalog, locale string, ref GlossaryArchiveReference) (*QualityCheckSurfaceReopen, map[string]protectedTranslation, error) {
	r, err := readFrozenSameUnitBridge(root, locale, ref)
	if err != nil {
		return nil, nil, err
	}
	if err := verifySameUnitBridgeDependencies(root, r); err != nil {
		return nil, nil, err
	}
	fresh, outputs, err := buildSameUnitBridge(root, c, r.SameUnitBridge.Options)
	if err != nil {
		return nil, nil, err
	}
	if !reflect.DeepEqual(fresh, r) {
		return nil, nil, fmt.Errorf("bridge authority reconstruction differs")
	}
	return r, outputs, nil
}
func bridgeOutput(root string, c *Catalog, g *Glossary, locale string, record RetranslationBatchUnit) (protectedTranslation, error) {
	if record.RepresentationBridge == nil {
		return protectedTranslation{}, fmt.Errorf("explicit recorded bridge required")
	}
	r, outputs, err := readSameUnitBridge(root, c, locale, *record.RepresentationBridge)
	if err != nil {
		return protectedTranslation{}, err
	}
	for _, u := range r.Units {
		if u.UnitID == record.UnitID && record.RevisionFeedbackSource == "quality_check" && record.PreviousSnapshotID == r.PreviousSnapshotID && record.PreviousRating == u.PreviousRating && record.PreviousFinding == u.Finding {
			out, ok := outputs[u.UnitID]
			if ok {
				return out, nil
			}
		}
	}
	return protectedTranslation{}, fmt.Errorf("bridge per-Unit QC feedback mismatch")
}

// Historical transport is not a new generation authorization. It binds frozen
// source/mapping/candidate lineage without replaying old mutable command docs.
// Every new export/import/process/retry continues to use readSameUnitBridge.
func readFrozenSameUnitBridge(root, locale string, ref GlossaryArchiveReference) (*QualityCheckSurfaceReopen, error) {
	b, err := readSnapshotReferencedFile(root, ref.Path)
	if err != nil || !validSHA256(ref.SHA256) || sum(b) != ref.SHA256 {
		return nil, fmt.Errorf("bridge receipt identity drift")
	}
	var r QualityCheckSurfaceReopen
	if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &r); err != nil {
		return nil, err
	}
	if r.SchemaVersion != qualityCheckSurfaceReopenSchemaVersion || r.EvidenceType != qualityCheckSurfaceReopenEvidenceType || r.UnitCount < 1 || r.UnitCount != len(r.Units) || r.SameUnitBridge == nil || r.CurrentValidation != nil || r.PreviewFailure != nil || r.IntegratedAuthority != nil || r.ReplacementContext != nil || r.AuthoritySource != SameUnitBridgeAuthoritySource || r.Locale != locale || r.SameUnitBridge.Identity != sameUnitBridgeIdentity(&r) || !bytes.Equal(b, bridgeReceiptBytes(&r)) || filepath.ToSlash(filepath.Join("data/quality-check-reopens", locale, r.ReopenID+".json")) != ref.Path {
		return nil, fmt.Errorf("invalid exclusive bridge authority")
	}

	if len(r.SameUnitBridge.FrozenDependencies) == 0 {
		return nil, fmt.Errorf("bridge lacks frozen history")
	}
	all := map[string]string{}
	for _, v := range r.SameUnitBridge.Dependencies {
		all[v.Path] = v.SHA256
	}
	seen := map[string]bool{}
	for _, v := range r.SameUnitBridge.FrozenDependencies {
		if seen[v.Path] || !validSHA256(v.SHA256) || all[v.Path] != v.SHA256 {
			return nil, fmt.Errorf("invalid frozen bridge dependency")
		}
		seen[v.Path] = true
		b, err := readSnapshotReferencedFile(root, v.Path)
		if err != nil || sum(b) != v.SHA256 {
			return nil, fmt.Errorf("frozen bridge dependency drift: %s", v.Path)
		}
	}
	return &r, nil
}

// Only already-passed candidate verification may use this frozen decoder.
// It grants neither a new export nor a process/retry authorization.
func frozenBridgeOutput(root string, c *Catalog, g *Glossary, locale string, record RetranslationBatchUnit) (protectedTranslation, error) {
	if record.RepresentationBridge == nil {
		return protectedTranslation{}, fmt.Errorf("missing frozen bridge reference")
	}
	r, err := readFrozenSameUnitBridge(root, locale, *record.RepresentationBridge)
	if err != nil {
		return protectedTranslation{}, err
	}
	var baseline *QualityCheckSurfaceReopenUnit
	for i := range r.Units {
		q := &r.Units[i]
		if q.UnitID == record.UnitID && record.RevisionFeedbackSource == "quality_check" && record.PreviousSnapshotID == r.PreviousSnapshotID && record.PreviousRating == q.PreviousRating && record.PreviousFinding == q.Finding {
			baseline = q
		}
	}
	if baseline == nil {
		return protectedTranslation{}, fmt.Errorf("frozen bridge QC feedback mismatch")
	}
	u, err := c.Unit(record.UnitID)
	if err != nil {
		return protectedTranslation{}, err
	}
	factory, ok := u.Adapter.(SameUnitBridgeAdapter)
	if !ok || u.SourcePath != baseline.Snapshot.SourcePath || u.SourceSHA256 != baseline.Snapshot.SourceSHA256 {
		return protectedTranslation{}, fmt.Errorf("frozen bridge source/adapter mismatch")
	}
	for _, binding := range r.SameUnitBridge.Bindings {
		if binding.UnitID != u.ID {
			continue
		}
		archive, err := readSnapshotReferencedFile(root, binding.Archive.Path)
		if err != nil || sum(archive) != binding.Archive.SHA256 {
			return protectedTranslation{}, fmt.Errorf("frozen bridge archive drift")
		}
		_, files, err := readGenerationBundle(archive)
		if err != nil {
			return protectedTranslation{}, err
		}
		if binding.TransportMember == surfaceBridgeMember(u.ID) {
			if sum(files[binding.TransportMember]) != binding.TransportSHA256 {
				return protectedTranslation{}, fmt.Errorf("frozen Surface bridge mapping drift")
			}
			mb, err := readSnapshotReferencedFile(root, binding.OriginalManifest.Path)
			if err != nil || sum(mb) != binding.OriginalManifest.SHA256 || !bytes.Equal(files["formal/manifest.json"], mb) {
				return protectedTranslation{}, fmt.Errorf("frozen Surface bridge manifest drift")
			}
			m, err := decodeRetranslationManifest(mb)
			if err != nil {
				return protectedTranslation{}, err
			}
			var predecessor RetranslationBatchUnit
			for _, item := range m.Units {
				if item.UnitID == u.ID {
					predecessor = item
				}
			}
			if predecessor.InputSHA256 != binding.FormalInputSHA256 || predecessor.InputSHA256 != record.InputSHA256 || m.BatchID != baseline.Snapshot.SelectedBatchID {
				return protectedTranslation{}, fmt.Errorf("frozen Surface bridge input/batch mismatch")
			}
			raw, err := readSnapshotReferencedFile(root, binding.Raw.Path)
			if err != nil || sum(raw) != binding.Raw.SHA256 {
				return protectedTranslation{}, fmt.Errorf("frozen Surface bridge raw drift")
			}
			reviewed, err := readSnapshotReferencedFile(root, baseline.Snapshot.CandidatePath)
			if err != nil || sum(reviewed) != baseline.Snapshot.CandidateSHA256 {
				return protectedTranslation{}, fmt.Errorf("frozen Surface bridge reviewed baseline drift")
			}
			decoder, template, ancestor, err := surfaceSameUnitBridgeOutput(root, c, locale, predecessor, raw, reviewed, files, func(path string) ([]byte, error) { return readSnapshotReferencedFile(root, path) })
			if err != nil || !bytes.Equal(template, []byte(binding.Template)) || sum(template) != binding.TemplateSHA256 || sum(ancestor) != binding.OrdinaryAncestorSHA256 {
				return protectedTranslation{}, fmt.Errorf("frozen Surface bridge reconstruction mismatch: %v", err)
			}
			text, tokens, err := decoder.ProtectedInput(g)
			return protectedTranslation{Text: text, Tokens: tokens, Adapter: decoder}, err
		}
		var p CommentRevisionTransport
		if err := decodeStrictCourseSourceDescriptionReviewJSON(files[binding.TransportMember], &p); err != nil || sum(files[binding.TransportMember]) != binding.TransportSHA256 {
			return protectedTranslation{}, fmt.Errorf("frozen restricted mapping drift")
		}
		oldName := filepath.Base(binding.ProtectedInput.Path)
		old := files["formal/candidates/"+oldName]
		ancestor := files["formal/historical-protected-candidates/"+oldName]
		cv, err := readCurrentQualityCheckSurfaceReopen(root, c, locale, strings.TrimSuffix(filepath.Base(p.Reopen.Path), ".json"))
		if err != nil || cv.revisionEvidence() != p.Reopen {
			return protectedTranslation{}, fmt.Errorf("frozen restricted authority drift: %v", err)
		}
		// Rebind using the original source-selected parser; the frozen mapping
		// must match byte-for-byte. No old current docs or command policy replay.
		bound, err := buildCommentRevisionTransport(root, c, g, locale, u, cv, old)
		if err != nil || !bytes.Equal(mustJSON(bound), files[binding.TransportMember]) {
			return protectedTranslation{}, fmt.Errorf("frozen restricted source mapping mismatch: %v", err)
		}
		raw, err := readSnapshotReferencedFile(root, binding.Raw.Path)
		if err != nil || sum(raw) != binding.Raw.SHA256 {
			return protectedTranslation{}, fmt.Errorf("frozen restricted raw drift")
		}
		reviewed, err := readSnapshotReferencedFile(root, baseline.Snapshot.CandidatePath)
		if err != nil || sum(reviewed) != baseline.Snapshot.CandidateSHA256 {
			return protectedTranslation{}, fmt.Errorf("frozen bridge reviewed baseline drift")
		}
		got, err := bound.Restore(string(raw))
		if err != nil || !bytes.Equal(canonicalizeRetranslationArtifactEOF(got), reviewed) {
			return protectedTranslation{}, fmt.Errorf("frozen restricted restore mismatch")
		}
		decoder, template, err := factory.BridgeRestrictedRevision(ancestor, old, reviewed, raw, bound)
		if err != nil || !bytes.Equal(template, []byte(binding.Template)) || sum(template) != binding.TemplateSHA256 {
			return protectedTranslation{}, fmt.Errorf("frozen bridge reconstruction mismatch: %v", err)
		}
		text, tokens, err := decoder.ProtectedInput(g)
		if err != nil {
			return protectedTranslation{}, err
		}
		return protectedTranslation{Text: text, Tokens: tokens, Adapter: decoder}, nil
	}
	return protectedTranslation{}, fmt.Errorf("frozen bridge Unit missing")
}
