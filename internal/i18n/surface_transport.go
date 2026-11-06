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

const SurfaceTransportSchema = "go-learning/surface-revision-transport/v1"

// Regions are emitted by the source adapter, never extracted from finding prose.
// A request explicitly binds the persisted finding to an exact set of these
// machine coordinates. It is separate from language-review authorization.
type SurfaceTransportRegion struct {
	ID               string `json:"id"`
	Kind             string `json:"kind"`
	SourceCoordinate string `json:"source_coordinate"`
	Preview          string `json:"preview"`
	RestoredPreview  string `json:"restored_preview,omitempty"`
}
type SurfaceTransportUnitRequest struct {
	UnitID        string   `json:"unit_id"`
	FindingSHA256 string   `json:"finding_sha256"`
	Regions       []string `json:"regions"`
}
type SurfaceTransportRequest struct {
	Schema         string                        `json:"schema"`
	Locale         string                        `json:"locale"`
	BatchID        string                        `json:"batch_id"`
	ManifestSHA256 string                        `json:"manifest_sha256"`
	Units          []SurfaceTransportUnitRequest `json:"units"`
}

// The adapter composes verified source-owned mappings. Implementations must
// not Protect(candidate), infer language findings, or change canonical inputs.
type SurfaceTransportAdapter interface {
	WithCurrentSurfaceTransportContext(string, *Catalog, func(string) error) error
	SurfaceTransportProjection(origin TranslationUnitAdapter, raw []byte, regions []string) (SurfaceTransportProjection, TranslationUnitAdapter, error)
	ReadSurfaceTransportProjection(json.RawMessage) (TranslationUnitAdapter, error)
}
type SurfaceTransportProjection struct {
	Kind      string                   `json:"kind"`
	Available []SurfaceTransportRegion `json:"available"`
	Editable  []string                 `json:"editable"`
	Template  string                   `json:"template"`
	Codec     json.RawMessage          `json:"codec"`
}
type SurfaceTransportBinding struct {
	UnitID           string                     `json:"unit_id"`
	Predecessor      QualityCheckSnapshotUnit   `json:"predecessor"`
	FindingSHA256    string                     `json:"finding_sha256"`
	Raw              GlossaryArchiveReference   `json:"raw"`
	CurrentAuthority string                     `json:"current_authority"`
	Projection       SurfaceTransportProjection `json:"projection"`
}
type SurfaceTransportAuthority struct {
	Schema                 string                     `json:"schema"`
	Request                SurfaceTransportRequest    `json:"request"`
	Reopen                 GlossaryArchiveReference   `json:"reopen"`
	CurrentReview          GlossaryArchiveReference   `json:"current_review"`
	Bindings               []SurfaceTransportBinding  `json:"bindings"`
	Dependencies           []GlossaryArchiveReference `json:"dependencies"`
	FrozenDependencies     []GlossaryArchiveReference `json:"frozen_dependencies"`
	CurrentContextIdentity string                     `json:"current_context_identity_sha256"`
	Identity               string                     `json:"identity_sha256"`
}
type SurfaceTransportPlan struct {
	Authority                *SurfaceTransportAuthority `json:"authority"`
	Path                     string                     `json:"path"`
	CurrentInputs            bool                       `json:"current_inputs"`
	RevisionTransportCapable bool                       `json:"revision_transport_capable"`
	Recorded                 bool                       `json:"recorded"`
	Mutation                 string                     `json:"mutation"`
}

func surfaceTransportPath(locale, reopen string) string {
	return "data/surface-revision-transports/" + locale + "/" + reopen + ".json"
}
func surfaceTransportBytes(a *SurfaceTransportAuthority) []byte { return append(mustJSON(a), '\n') }
func surfaceTransportIdentity(a SurfaceTransportAuthority) string {
	a.Identity = ""
	return sum(mustJSON(a))
}

// Empty Units is discovery only: it lists machine-owned coordinates and cannot
// be recorded or authorize a Generation handoff.
func PrepareSurfaceRevisionTransport(root string, c *Catalog, locale, batch string, request *SurfaceTransportRequest, record bool) (*SurfaceTransportPlan, error) {
	a, _, err := buildSurfaceTransport(root, c, locale, batch, request)
	if err != nil {
		return nil, err
	}
	capable := len(a.Request.Units) == len(a.Bindings) && len(a.Bindings) > 0
	path := surfaceTransportPath(locale, strings.TrimSuffix(filepath.Base(a.Reopen.Path), ".json"))
	p := &SurfaceTransportPlan{Authority: a, Path: path, CurrentInputs: true, RevisionTransportCapable: capable, Mutation: "none"}
	if !record {
		return p, nil
	}
	if !capable {
		return nil, fmt.Errorf("Surface transport discovery is not capability authority")
	}
	if err := ensureNoGenerationImportPending(filepath.Join(root, "data/retranslation-runs", locale, batch)); err != nil {
		return nil, err
	}
	for _, name := range []string{"raw-responses", "generation-imports", "result.json", "validation", "retries"} {
		if _, err := os.Lstat(filepath.Join(root, "data/retranslation-runs", locale, batch, name)); !os.IsNotExist(err) {
			return nil, fmt.Errorf("Surface transport record requires an unconsumed batch without outputs/pending attempts")
		}
	}
	fresh, _, err := buildSurfaceTransport(root, c, locale, batch, &a.Request)
	if err != nil || !bytes.Equal(surfaceTransportBytes(a), surfaceTransportBytes(fresh)) {
		return nil, fmt.Errorf("Surface transport changed before record: %v", err)
	}
	dst, err := compatibilityPath(root, path, true)
	if err != nil {
		return nil, err
	}
	if err := writeNewGenerationFile(dst, surfaceTransportBytes(a)); err != nil {
		return nil, err
	}
	p.Recorded, p.Mutation = true, "known"
	return p, nil
}

func buildSurfaceTransport(root string, c *Catalog, locale, batch string, request *SurfaceTransportRequest) (*SurfaceTransportAuthority, map[string]protectedTranslation, error) {
	if c == nil || ValidateLocaleName(locale) != nil || validateBatchID(batch) != nil {
		return nil, nil, fmt.Errorf("exact Surface transport catalog/locale/batch required")
	}
	if err := RequireNewLanguageGlossaryReview(root, locale); err != nil {
		return nil, nil, err
	}
	m, err := readRetranslationProcessManifest(filepath.Join(root, "data/retranslation-runs", locale, batch), locale, batch)
	if err != nil || len(m.Units) == 0 {
		return nil, nil, fmt.Errorf("Surface transport batch missing: %v", err)
	}
	u, err := c.Unit(m.Units[0].UnitID)
	if err != nil {
		return nil, nil, err
	}
	factory, ok := u.Adapter.(SurfaceTransportAdapter)
	if !ok {
		return nil, nil, fmt.Errorf("Surface current source context unavailable")
	}
	var a *SurfaceTransportAuthority
	var outputs map[string]protectedTranslation
	err = factory.WithCurrentSurfaceTransportContext(root, c, func(context string) error {
		var e error
		a, outputs, e = buildSurfaceTransportInContext(root, c, locale, batch, request)
		if e != nil {
			return e
		}
		a.CurrentContextIdentity = context
		a.Identity = surfaceTransportIdentity(*a)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if review, e := RequireUnifiedGlossaryReview(root, locale); e != nil || review != a.CurrentReview {
		return nil, nil, fmt.Errorf("Surface transport current Review selection changed")
	}
	for _, ref := range a.Dependencies {
		b, e := readSnapshotReferencedFile(root, ref.Path)
		if e != nil || sum(b) != ref.SHA256 {
			return nil, nil, fmt.Errorf("Surface transport changed after context guard: %s", ref.Path)
		}
	}
	return a, outputs, nil
}

func buildSurfaceTransportInContext(root string, c *Catalog, locale, batch string, request *SurfaceTransportRequest) (*SurfaceTransportAuthority, map[string]protectedTranslation, error) {
	if c == nil || ValidateLocaleName(locale) != nil || validateBatchID(batch) != nil {
		return nil, nil, fmt.Errorf("exact Surface transport catalog/locale/batch required")
	}
	base := "data/retranslation-runs/" + locale + "/" + batch
	refs := map[string]string{}
	read := func(path string) ([]byte, error) {
		b, e := readSnapshotReferencedFile(root, path)
		if e != nil {
			return nil, e
		}
		if old := refs[path]; old != "" && old != sum(b) {
			return nil, fmt.Errorf("Surface transport dependency changed: %s", path)
		}
		refs[path] = sum(b)
		return b, nil
	}
	mb, err := read(base + "/manifest.json")
	if err != nil {
		return nil, nil, err
	}
	m, err := readRetranslationProcessManifest(filepath.Join(root, base), locale, batch)
	if err != nil {
		return nil, nil, err
	}
	if len(m.Units) == 0 || len(m.Units) > MaxRetranslationExportLimit {
		return nil, nil, fmt.Errorf("Surface transport exact Page scope required")
	}
	for _, r := range m.Units {
		if r.UnitKind != UnitKindPage || r.RevisionFeedbackSource != "surface_review" || r.RevisionAuthorizationID != m.Units[0].RevisionAuthorizationID || r.PreviousSnapshotID != m.Units[0].PreviousSnapshotID || r.RepresentationBridge != nil {
			return nil, nil, fmt.Errorf("Surface transport requires one exact Surface reopen/predecessor")
		}
	}
	rq := SurfaceTransportRequest{Schema: SurfaceTransportSchema, Locale: locale, BatchID: batch, ManifestSHA256: sum(mb)}
	if request != nil {
		rq = *request
		if rq.Schema != SurfaceTransportSchema || rq.Locale != locale || rq.BatchID != batch || rq.ManifestSHA256 != sum(mb) || len(rq.Units) != len(m.Units) {
			return nil, nil, fmt.Errorf("Surface transport request exact batch scope mismatch")
		}
	}
	reopen, err := readCurrentQualityCheckSurfaceReopen(root, c, locale, m.Units[0].RevisionAuthorizationID)
	if err != nil {
		return nil, nil, fmt.Errorf("Surface transport current eligibility: %w", err)
	}
	if reopen.revisionFeedbackSource() != "surface_review" || reopen.UnitCount != len(m.Units) {
		return nil, nil, fmt.Errorf("Surface transport wrong reopen authority")
	}
	predecessorCatalog, err := surfaceReopenPredecessorCatalog(root, c, locale, m.Units[0].PreviousSnapshotID, reopen.IntegratedAuthority)
	if err != nil {
		return nil, nil, err
	}
	if _, err := RequireCurrentTranslationAuthorization(root, predecessorCatalog, locale, m.Units[0].PreviousSnapshotID); err != nil {
		return nil, nil, fmt.Errorf("Surface transport current predecessor eligibility: %w", err)
	}
	// The validated Surface reader constrains the explicit CURRENT owner above.
	// Reconstruct historical response evidence from the existing inventory-based
	// predecessor selector; never compare 51 current members to one 49-Unit era.
	inventory := *c
	inventory.TranslationScope = nil
	inputs, err := revisionGenerationInputsCore(root, &inventory, locale, m.Units, nil, false)
	if err != nil {
		return nil, nil, err
	}
	for _, in := range inputs {
		if in.repositoryPath != "" {
			b, e := read(in.repositoryPath)
			if e != nil || !bytes.Equal(b, in.data) {
				return nil, nil, fmt.Errorf("Surface transport frozen input changed: %s", in.repositoryPath)
			}
		}
	}
	g, err := LoadGlossary(root, locale)
	if err != nil {
		return nil, nil, err
	}
	review, err := RequireUnifiedGlossaryReview(root, locale)
	if err != nil {
		return nil, nil, err
	}
	paths := append([]string{}, translationUnitGenerationAuthorityPaths...)
	paths = append(paths, glossaryReviewGlossaryPath(locale), review.Path, GlossaryCorpusPath, "internal/i18n/surface_transport.go", "internal/sitecontent/surface_transport.go")
	for _, path := range paths {
		if _, err := read(path); err != nil {
			return nil, nil, err
		}
	}
	pc, _, err := surfaceRevisionCatalog(root, &inventory, locale, m.Units[0].RevisionAuthorizationID)
	if err != nil {
		return nil, nil, err
	}
	scoped, registered, err := SelectFinalizedTranslationPackage(root, pc, locale, m.Units[0].PreviousSnapshotID)
	if err != nil {
		return nil, nil, err
	}
	if registered {
		pc = scoped
	}
	a := &SurfaceTransportAuthority{Schema: SurfaceTransportSchema, Request: rq, CurrentReview: review}
	reopenPath := "data/quality-check-reopens/" + locale + "/" + reopen.ReopenID + ".json"
	rb, err := read(reopenPath)
	if err != nil {
		return nil, nil, err
	}
	a.Reopen = GlossaryArchiveReference{reopenPath, sum(rb)}
	outputs := map[string]protectedTranslation{}
	for i, r := range m.Units {
		u, err := c.Unit(r.UnitID)
		if err != nil {
			return nil, nil, err
		}
		disk, err := read(u.SourcePath)
		if err != nil || validateSnapshotSourceIdentity(c, u, disk) != nil {
			return nil, nil, fmt.Errorf("Surface transport current source drift: %s", u.ID)
		}
		protected, err := prepareTranslationUnitInput(u, g)
		if err != nil {
			return nil, nil, err
		}
		input := []byte(protected.Text)
		if m.ArtifactEOF == retranslationArtifactEOFSingleLF {
			input = canonicalizeRetranslationArtifactEOF(input)
		}
		saved, err := read(base + "/" + r.InputPath)
		if err != nil || sum(saved) != r.InputSHA256 || !bytes.Equal(input, saved) || len(protected.Tokens) != r.ProtectedTokenCount {
			return nil, nil, fmt.Errorf("Surface transport current protected input drift: %s", r.UnitID)
		}
		factory, ok := u.Adapter.(SurfaceTransportAdapter)
		if !ok {
			return nil, nil, fmt.Errorf("Surface transport source adapter unavailable: %s", r.UnitID)
		}
		var s *QualityCheckSnapshotUnit
		for j := range reopen.Units {
			if reopen.Units[j].UnitID == r.UnitID {
				s = &reopen.Units[j].Snapshot
			}
		}
		if s == nil {
			return nil, nil, fmt.Errorf("Surface transport Unit outside reopen")
		}
		e, err := readSnapshotUnitRepositoryEvidence(root, pc, locale, *s)
		if err != nil {
			return nil, nil, err
		}
		rawPath := "data/retranslation-runs/" + locale + "/" + s.SelectedBatchID + "/" + e.validation.RawResponsePath
		raw, err := read(rawPath)
		if err != nil {
			return nil, nil, err
		}
		origin, template, err := surfaceTransportOrigin(root, pc, g, locale, r, *s, e, raw, read)
		if err != nil {
			return nil, nil, err
		}
		var regions []string
		if request != nil {
			q := rq.Units[i]
			if q.UnitID != r.UnitID || q.FindingSHA256 != sum([]byte(r.PreviousFinding)) || len(q.Regions) == 0 {
				return nil, nil, fmt.Errorf("Surface transport exact finding/Unit mapping required")
			}
			regions = q.Regions
		}
		projection, adapter, err := factory.SurfaceTransportProjection(origin.Adapter, template, regions)
		if err != nil {
			return nil, nil, err
		}
		if request != nil {
			got, restoreErr := adapter.Restore(projection.Template)
			if restoreErr != nil || !bytes.Equal(canonicalizeRetranslationArtifactEOF(got), e.candidate) {
				return nil, nil, fmt.Errorf("Surface projection does not restore exact predecessor: %v", restoreErr)
			}
			outputs[r.UnitID] = protectedTranslation{Text: projection.Template, Adapter: adapter}
		}
		authority, err := currentTranslationAuthority(root, u, locale)
		if err != nil {
			return nil, nil, err
		}
		a.Bindings = append(a.Bindings, SurfaceTransportBinding{UnitID: r.UnitID, Predecessor: *s, FindingSHA256: sum([]byte(r.PreviousFinding)), Raw: GlossaryArchiveReference{rawPath, sum(raw)}, CurrentAuthority: authority, Projection: projection})
	}
	for path, sha := range refs {
		b, err := readSnapshotReferencedFile(root, path)
		if err != nil || sum(b) != sha {
			return nil, nil, fmt.Errorf("Surface transport dependency changed before return: %s", path)
		}
		a.Dependencies = append(a.Dependencies, GlossaryArchiveReference{path, sha})
		if strings.HasPrefix(path, "data/retranslation-runs/") || strings.HasPrefix(path, "data/generation-archives/") || strings.HasPrefix(path, "data/quality-check-") || strings.HasPrefix(path, "data/translation-unit-reconciliations/") || strings.HasPrefix(path, "data/translation-authorizations/") || strings.HasPrefix(path, "data/locale-language/") || strings.HasPrefix(path, "data/locale-surface-reviews/") || strings.HasPrefix(path, "data/glossary-history/") || strings.HasPrefix(path, "data/unified-glossary-reviews/") || strings.HasPrefix(path, "_content/") {
			a.FrozenDependencies = append(a.FrozenDependencies, GlossaryArchiveReference{path, sha})
		}
	}
	if fresh, err := RequireUnifiedGlossaryReview(root, locale); err != nil || fresh != review {
		return nil, nil, fmt.Errorf("Surface current Review selection changed during projection")
	}
	sort.Slice(a.Dependencies, func(i, j int) bool { return a.Dependencies[i].Path < a.Dependencies[j].Path })
	sort.Slice(a.FrozenDependencies, func(i, j int) bool { return a.FrozenDependencies[i].Path < a.FrozenDependencies[j].Path })
	a.Identity = surfaceTransportIdentity(*a)
	return a, outputs, nil
}

func readSurfaceTransport(root string, c *Catalog, locale, reopen string) (*SurfaceTransportAuthority, map[string]protectedTranslation, error) {
	path := surfaceTransportPath(locale, reopen)
	b, err := readSnapshotReferencedFile(root, path)
	if err != nil {
		return nil, nil, fmt.Errorf("Surface revision transport capability authority required: %w", err)
	}
	var a SurfaceTransportAuthority
	if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &a); err != nil || a.Schema != SurfaceTransportSchema || a.Identity != surfaceTransportIdentity(a) || a.Request.Locale != locale {
		return nil, nil, fmt.Errorf("Surface transport authority identity mismatch")
	}
	want, outputs, err := buildSurfaceTransport(root, c, locale, a.Request.BatchID, &a.Request)
	if err != nil {
		return nil, nil, err
	}
	if !bytes.Equal(b, surfaceTransportBytes(want)) {
		return nil, nil, fmt.Errorf("Surface transport authority stale/tampered")
	}
	return &a, outputs, nil
}

func applySurfaceTransport(root string, c *Catalog, locale string, selected []RetranslationBatchUnit, inputs []generationBundleInput) ([]generationBundleInput, error) {
	return applySurfaceTransportInWorld(root, c, locale, selected, inputs, false)
}

func applySurfaceTransportInWorld(root string, c *Catalog, locale string, selected []RetranslationBatchUnit, inputs []generationBundleInput, historical bool) ([]generationBundleInput, error) {
	var surface []RetranslationBatchUnit
	for _, r := range selected {
		if r.RevisionFeedbackSource == "surface_review" && (!historical || hasSurfaceTransport(root, locale, r.RevisionAuthorizationID)) {
			surface = append(surface, r)
		}
	}
	if len(surface) == 0 {
		return inputs, nil
	}
	var a *SurfaceTransportAuthority
	var err error
	if historical {
		for _, r := range surface {
			if _, _, err := surfaceTransportOutput(root, c, locale, r); err != nil {
				return nil, err
			}
		}
		b, readErr := readSnapshotReferencedFile(root, surfaceTransportPath(locale, surface[0].RevisionAuthorizationID))
		if readErr != nil {
			return nil, readErr
		}
		a = &SurfaceTransportAuthority{}
		err = decodeStrictCourseSourceDescriptionReviewJSON(b, a)
	} else {
		a, _, err = readSurfaceTransport(root, c, locale, surface[0].RevisionAuthorizationID)
	}
	if err != nil {
		return nil, err
	}
	for _, r := range selected {
		if r.RevisionFeedbackSource != "surface_review" {
			continue
		}
		if historical && !hasSurfaceTransport(root, locale, r.RevisionAuthorizationID) {
			continue
		} // pre-capability imported history
		if surface[0].RevisionAuthorizationID != r.RevisionAuthorizationID {
			return nil, fmt.Errorf("mixed Surface capability authorities")
		}
		var binding *SurfaceTransportBinding
		for i := range a.Bindings {
			if a.Bindings[i].UnitID == r.UnitID {
				binding = &a.Bindings[i]
			}
		}
		if binding == nil || binding.FindingSHA256 != sum([]byte(r.PreviousFinding)) || binding.Predecessor.UnitID != r.UnitID {
			return nil, fmt.Errorf("Surface capability crosses Unit/finding boundary")
		}
		mb, err := readSnapshotReferencedFile(root, "data/retranslation-runs/"+locale+"/"+a.Request.BatchID+"/manifest.json")
		if err != nil || sum(mb) != a.Request.ManifestSHA256 {
			return nil, fmt.Errorf("Surface capability batch changed")
		}
		var m RetranslationBatchManifest
		if json.Unmarshal(mb, &m) != nil {
			return nil, fmt.Errorf("Surface capability batch invalid")
		}
		found := false
		for _, u := range m.Units {
			found = found || reflect.DeepEqual(u, r)
		}
		if !found {
			return nil, fmt.Errorf("Surface capability different revision record")
		}
		name := filepath.Base(r.InputPath)
		target := "formal/protected-candidates/" + name
		for i := range inputs {
			if inputs[i].path == target {
				inputs = append(inputs, generationBundleInput{"formal/historical-protected-candidates/" + name, inputs[i].repositoryPath, inputs[i].data})
				inputs[i].repositoryPath = ""
				inputs[i].data = []byte(binding.Projection.Template)
			}
		}
		inputs = append(inputs, generationBundleInput{"formal/surface-transport/" + r.UnitID + ".json", "", mustJSON(binding.Projection)}, generationBundleInput{"formal/surface-transport-authority/" + r.UnitID + ".json", surfaceTransportPath(locale, r.RevisionAuthorizationID), surfaceTransportBytes(a)})
	}
	return inputs, nil
}

func revisionGenerationResponseInputs(root string, c *Catalog, locale string, selected []RetranslationBatchUnit, transports ...map[string]protectedTranslation) ([]generationBundleInput, error) {
	inputs, err := revisionGenerationInputsCore(root, c, locale, selected, transports, false)
	if err != nil {
		return nil, err
	}
	return applySurfaceTransportInWorld(root, c, locale, selected, inputs, true)
}

// Existence is used only to select a reader, never to accept currentness.
func hasSurfaceTransport(root, locale, reopen string) bool {
	_, err := os.Lstat(filepath.Join(root, surfaceTransportPath(locale, reopen)))
	return !os.IsNotExist(err)
}

func checkSurfaceTransportBatch(root, locale, batch string, units []RetranslationBatchUnit, required bool) error {
	for _, r := range units {
		if r.RevisionFeedbackSource != "surface_review" || (!required && !hasSurfaceTransport(root, locale, r.RevisionAuthorizationID)) {
			continue
		}
		b, err := readSnapshotReferencedFile(root, surfaceTransportPath(locale, r.RevisionAuthorizationID))
		if err != nil {
			return fmt.Errorf("Surface revision transport capability authority required: %w", err)
		}
		var a SurfaceTransportAuthority
		if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &a); err != nil || a.Schema != SurfaceTransportSchema || a.Identity != surfaceTransportIdentity(a) || a.Request.Locale != locale || a.Request.BatchID != batch {
			return fmt.Errorf("Surface transport wrong batch authority")
		}
		manifest, err := readSnapshotReferencedFile(root, "data/retranslation-runs/"+locale+"/"+batch+"/manifest.json")
		if err != nil || sum(manifest) != a.Request.ManifestSHA256 {
			return fmt.Errorf("Surface transport batch manifest changed")
		}
	}
	return nil
}

// A response decoder is historical representation, not current authorization.
// It may only be consumed after the importer has sealed these exact authority
// bytes in the original Generation archive. Fresh source/glossary/validator
// checks remain the process/Snapshot/current-assessment caller's responsibility.
func surfaceTransportOutput(root string, c *Catalog, locale string, r RetranslationBatchUnit) (protectedTranslation, *bridgeResponseLineage, error) {
	if r.RevisionFeedbackSource != "surface_review" || !hasSurfaceTransport(root, locale, r.RevisionAuthorizationID) {
		return protectedTranslation{}, nil, nil
	}
	if c == nil {
		return protectedTranslation{}, nil, fmt.Errorf("Surface historical response catalog required")
	}
	path := surfaceTransportPath(locale, r.RevisionAuthorizationID)
	b, err := readSnapshotReferencedFile(root, path)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	var a SurfaceTransportAuthority
	if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &a); err != nil || a.Schema != SurfaceTransportSchema || a.Identity != surfaceTransportIdentity(a) || a.Request.Locale != locale || !bytes.Equal(b, surfaceTransportBytes(&a)) {
		return protectedTranslation{}, nil, fmt.Errorf("Surface historical transport identity mismatch")
	}
	base := "data/retranslation-runs/" + locale + "/" + a.Request.BatchID
	mb, err := readSnapshotReferencedFile(root, base+"/manifest.json")
	if err != nil || sum(mb) != a.Request.ManifestSHA256 {
		return protectedTranslation{}, nil, fmt.Errorf("Surface historical batch mismatch")
	}
	m, err := readRetranslationProcessManifest(filepath.Join(root, base), locale, a.Request.BatchID)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	matched := false
	for _, u := range m.Units {
		matched = matched || reflect.DeepEqual(u, r)
	}
	if !matched {
		return protectedTranslation{}, nil, fmt.Errorf("Surface historical record crosses batch/Unit")
	}
	importPath := base + "/generation-imports/initial-manifest.json"
	ib, err := readSnapshotReferencedFile(root, importPath)
	if err != nil {
		return protectedTranslation{}, nil, fmt.Errorf("Surface response decoder requires sealed import: %w", err)
	}
	var imported GenerationResultBundleManifest
	if err := decodeStrictCourseSourceDescriptionReviewJSON(ib, &imported); err != nil {
		return protectedTranslation{}, nil, err
	}
	archivePath := importedGenerationArchive(locale, imported.GenerationBundleSHA256)
	archive, err := readSnapshotReferencedFile(root, archivePath)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	gm, files, err := readGenerationBundle(archive)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	if err := checkImportedGenerationIdentity(imported, gm, archive, files); err != nil {
		return protectedTranslation{}, nil, err
	}
	if gm.Locale != locale || gm.BatchID != a.Request.BatchID || !bytes.Equal(files["formal/surface-transport-authority/"+r.UnitID+".json"], b) {
		return protectedTranslation{}, nil, fmt.Errorf("Surface decoder authority differs from imported archive")
	}
	var binding *SurfaceTransportBinding
	for i := range a.Bindings {
		if a.Bindings[i].UnitID == r.UnitID {
			binding = &a.Bindings[i]
		}
	}
	if binding == nil || binding.FindingSHA256 != sum([]byte(r.PreviousFinding)) || binding.Predecessor.UnitID != r.UnitID || r.PreviousSnapshotID == "" {
		return protectedTranslation{}, nil, fmt.Errorf("Surface historical finding/predecessor mismatch")
	}
	u, err := c.Unit(r.UnitID)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	factory, ok := u.Adapter.(SurfaceTransportAdapter)
	if !ok {
		return protectedTranslation{}, nil, fmt.Errorf("Surface historical source adapter unavailable")
	}
	adapter, err := factory.ReadSurfaceTransportProjection(binding.Projection.Codec)
	if err != nil {
		return protectedTranslation{}, nil, err
	}
	got, err := adapter.Restore(binding.Projection.Template)
	candidate, readErr := readSnapshotReferencedFile(root, binding.Predecessor.CandidatePath)
	if err != nil || readErr != nil || sum(candidate) != binding.Predecessor.CandidateSHA256 || !bytes.Equal(canonicalizeRetranslationArtifactEOF(got), candidate) {
		return protectedTranslation{}, nil, fmt.Errorf("Surface historical template/candidate binding mismatch")
	}
	lineage := &bridgeResponseLineage{Bridge: GlossaryArchiveReference{path, sum(b)}, Dependencies: append([]GlossaryArchiveReference(nil), a.FrozenDependencies...)}
	lineage.Dependencies = append(lineage.Dependencies, GlossaryArchiveReference{path, sum(b)}, GlossaryArchiveReference{importPath, sum(ib)}, GlossaryArchiveReference{archivePath, sum(archive)})
	sort.Slice(lineage.Dependencies, func(i, j int) bool { return lineage.Dependencies[i].Path < lineage.Dependencies[j].Path })
	if err := lineage.check(root, locale); err != nil {
		return protectedTranslation{}, nil, err
	}
	return protectedTranslation{Text: binding.Projection.Template, Adapter: adapter}, lineage, nil
}

func frozenSurfaceTransportReferences(root, locale string, r RetranslationBatchUnit, batch string) ([]GlossaryArchiveReference, error) {
	path := surfaceTransportPath(locale, r.RevisionAuthorizationID)
	b, err := readSnapshotReferencedFile(root, path)
	if err != nil {
		return nil, err
	}
	var a SurfaceTransportAuthority
	if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &a); err != nil || a.Schema != SurfaceTransportSchema || a.Identity != surfaceTransportIdentity(a) || a.Request.BatchID != batch || a.Request.Locale != locale || !bytes.Equal(b, surfaceTransportBytes(&a)) {
		return nil, fmt.Errorf("frozen Surface capability identity mismatch")
	}
	base := "data/retranslation-runs/" + locale + "/" + batch
	m, err := readRetranslationProcessManifest(filepath.Join(root, base), locale, batch)
	if err != nil {
		return nil, err
	}
	found := false
	for _, u := range m.Units {
		found = found || reflect.DeepEqual(u, r)
	}
	if !found {
		return nil, fmt.Errorf("frozen Surface capability record mismatch")
	}
	importPath := base + "/generation-imports/initial-manifest.json"
	ib, err := readSnapshotReferencedFile(root, importPath)
	if err != nil {
		return nil, err
	}
	var imported GenerationResultBundleManifest
	if err := decodeStrictCourseSourceDescriptionReviewJSON(ib, &imported); err != nil {
		return nil, err
	}
	archivePath := importedGenerationArchive(locale, imported.GenerationBundleSHA256)
	archive, err := readSnapshotReferencedFile(root, archivePath)
	if err != nil {
		return nil, err
	}
	gm, files, err := readGenerationBundle(archive)
	if err != nil {
		return nil, err
	}
	if err := checkImportedGenerationIdentity(imported, gm, archive, files); err != nil {
		return nil, err
	}
	if gm.Locale != locale || gm.BatchID != batch || !bytes.Equal(files["formal/surface-transport-authority/"+r.UnitID+".json"], b) {
		return nil, fmt.Errorf("frozen Surface capability archive mismatch")
	}
	refs := append([]GlossaryArchiveReference(nil), a.FrozenDependencies...)
	refs = append(refs, GlossaryArchiveReference{path, sum(b)}, GlossaryArchiveReference{importPath, sum(ib)}, GlossaryArchiveReference{archivePath, sum(archive)})
	for _, ref := range refs {
		if _, _, err := readHistoricalBridgeReference(root, locale, ref); err != nil {
			return nil, err
		}
	}
	return refs, nil
}
