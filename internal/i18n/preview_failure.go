package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

const PreviewTUFailureSchema = "go-learning/preview-tu-failure/v1"
const PreviewMissingSourceLink = "missing-source-bound-link"
const PreviewRevisionFeedback = "preview_acceptance"

// PreviewTUFailure is machine evidence, never a language rating or a Surface
// finding. The representation reader must independently reproduce its proof.
type PreviewTUFailure struct {
	Schema         string                     `json:"schema"`
	FailureID      string                     `json:"failure_id"`
	Locale         string                     `json:"locale"`
	Package        string                     `json:"package"`
	PackageSHA256  string                     `json:"package_sha256"`
	Route          string                     `json:"route"`
	PreviewURL     string                     `json:"preview_url"`
	Stage          string                     `json:"stage"`
	Check          string                     `json:"check"`
	Artifact       SurfaceCandidateArtifact   `json:"artifact"`
	Snapshot       GlossaryArchiveReference   `json:"snapshot"`
	Finalization   GlossaryArchiveReference   `json:"finalization"`
	Contracts      []GlossaryArchiveReference `json:"contracts"`
	ContextSHA256  string                     `json:"context_sha256"`
	ResponseHTML   string                     `json:"response_html"`
	ResponseSHA256 string                     `json:"response_sha256"`
	MissingLinks   []string                   `json:"missing_links"`
	Identity       string                     `json:"identity_sha256"`
}

// current=true additionally checks the live preview, completed package context,
// verifier/renderer contracts and latest finalized selection. Consumed revision
// invocations retain the frozen observation; they never depend on a live server.
type PreviewFailureReader func(string, *Catalog, *PreviewTUFailure, bool) error

// Frozen proof belongs to the representation adapter, so closure/authorization
// readers never need a live-preview dependency or a site renderer import.
type FrozenPreviewFailureAdapter interface {
	VerifyFrozenPreviewTUFailure([]byte, *PreviewTUFailure) error
}

func previewFailureIdentity(p PreviewTUFailure) string {
	p.Identity = ""
	return sum(mustJSON(p))
}
func PreviewTUFailurePath(locale, id string) string {
	return "data/preview-tu-failures/" + locale + "/" + id + ".json"
}
func (p PreviewTUFailure) Finding() string {
	return fmt.Sprintf("%s: route=%s source=%s missing=%q", p.Check, p.Route, p.Artifact.Unit.SourcePath, p.MissingLinks)
}

// PreparePreviewTUFailure selects only the explicitly finalized Unit, not a
// latest candidate or a caller-provided path/SHA.
func PreparePreviewTUFailure(root string, c *Catalog, locale, snapshot, unitID string) (*PreviewTUFailure, error) {
	f, err := VerifyQualityCheckFinalization(root, c, locale, snapshot)
	if err != nil {
		return nil, err
	}
	p := &PreviewTUFailure{Schema: PreviewTUFailureSchema, Locale: locale, Stage: "Site v2 navigation/assets", Check: PreviewMissingSourceLink}
	for _, u := range f.Units {
		if u.UnitID == unitID {
			p.Artifact = SurfaceCandidateArtifact{SnapshotID: snapshot, Unit: u.Snapshot}
		}
	}
	if p.Artifact.Unit.UnitID == "" {
		return nil, fmt.Errorf("preview failure Unit is absent from finalized predecessor")
	}
	for _, ref := range []struct {
		dst  *GlossaryArchiveReference
		name string
	}{
		{&p.Snapshot, "manifest.json"}, {&p.Finalization, "finalization.json"},
	} {
		ref.dst.Path = "data/quality-check-snapshots/" + locale + "/" + snapshot + "/" + ref.name
		b, err := readSnapshotReferencedFile(root, ref.dst.Path)
		if err != nil {
			return nil, err
		}
		ref.dst.SHA256 = sum(b)
	}
	return p, nil
}

func verifyPreviewTUFailure(root string, c *Catalog, p *PreviewTUFailure, current bool) error {
	if c == nil || (current && c.PreviewFailureReader == nil) {
		return fmt.Errorf("preview TU proof reader required")
	}
	if p.Schema != PreviewTUFailureSchema || ValidateLocaleName(p.Locale) != nil || !reviewIDPattern.MatchString(p.FailureID) ||
		p.Identity != previewFailureIdentity(*p) || p.Stage != "Site v2 navigation/assets" || p.Check != PreviewMissingSourceLink ||
		len(p.MissingLinks) == 0 || !validSHA256(p.ContextSHA256) || p.ResponseSHA256 != sum([]byte(p.ResponseHTML)) {
		return fmt.Errorf("invalid machine preview failure identity/check")
	}
	base := "data/quality-check-snapshots/" + p.Locale + "/" + p.Artifact.SnapshotID + "/"
	if p.Snapshot.Path != base+"manifest.json" || p.Finalization.Path != base+"finalization.json" {
		return fmt.Errorf("preview failure predecessor paths mismatch")
	}
	for _, ref := range []GlossaryArchiveReference{p.Snapshot, p.Finalization} {
		b, err := readSnapshotReferencedFile(root, ref.Path)
		if err != nil || sum(b) != ref.SHA256 {
			return fmt.Errorf("preview predecessor evidence stale: %s", ref.Path)
		}
	}
	if err := ValidateSurfaceCandidateArtifact(root, c, p.Locale, p.Artifact); err != nil {
		return err
	}
	var f QualityCheckFinalization
	b, _ := readSnapshotReferencedFile(root, p.Finalization.Path)
	if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &f); err != nil {
		return err
	}
	matched := false
	for _, u := range f.Units {
		if u.Rating == "A" && reflect.DeepEqual(u.Snapshot, p.Artifact.Unit) {
			matched = true
		}
	}
	if f.Locale != p.Locale || f.SnapshotID != p.Artifact.SnapshotID || !matched {
		return fmt.Errorf("preview artifact lacks finalized A membership")
	}
	if current {
		if _, err := VerifyQualityCheckFinalization(root, c, p.Locale, p.Artifact.SnapshotID); err != nil {
			return err
		}
		if err := finalizationMatchesLatestPromotionCandidates(root, c, p.Locale, p.Artifact.SnapshotID); err != nil {
			return err
		}
	}
	if !current {
		u, err := c.Unit(p.Artifact.Unit.UnitID)
		if err != nil {
			return err
		}
		if proof, ok := u.Adapter.(FrozenPreviewFailureAdapter); ok {
			candidate, err := readSnapshotReferencedFile(root, p.Artifact.Unit.CandidatePath)
			if err != nil {
				return err
			}
			return proof.VerifyFrozenPreviewTUFailure(candidate, p)
		}
	}
	if c.PreviewFailureReader == nil {
		return fmt.Errorf("frozen preview TU proof adapter required")
	}
	return c.PreviewFailureReader(root, c, p, current)
}

func RecordPreviewTUFailure(root string, c *Catalog, p PreviewTUFailure) (string, error) {
	p.Identity = previewFailureIdentity(p)
	if err := verifyPreviewTUFailure(root, c, &p, true); err != nil {
		return "", err
	}
	path := PreviewTUFailurePath(p.Locale, p.FailureID)
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0755); err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	if err := writeNewGenerationFile(filepath.Join(root, path), append(b, '\n')); err != nil {
		return "", err
	}
	return path, nil
}

func readPreviewTUFailure(root string, c *Catalog, ref GlossaryArchiveReference, current bool) (*PreviewTUFailure, error) {
	b, err := readSnapshotReferencedFile(root, ref.Path)
	if err != nil || sum(b) != ref.SHA256 {
		return nil, fmt.Errorf("preview failure receipt stale")
	}
	var p PreviewTUFailure
	if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &p); err != nil {
		return nil, err
	}
	if ref.Path != PreviewTUFailurePath(p.Locale, p.FailureID) {
		return nil, fmt.Errorf("unsafe preview failure path")
	}
	if err := verifyPreviewTUFailure(root, c, &p, current); err != nil {
		return nil, err
	}
	return &p, nil
}

func previewReopenProjection(p *PreviewTUFailure, ref GlossaryArchiveReference, id string) QualityCheckSurfaceReopen {
	return QualityCheckSurfaceReopen{SchemaVersion: qualityCheckSurfaceReopenSchemaVersion, EvidenceType: qualityCheckSurfaceReopenEvidenceType,
		Locale: p.Locale, ReopenID: id, PreviousSnapshotID: p.Artifact.SnapshotID, PreviousFinalizationPath: p.Finalization.Path, PreviousFinalizationSHA256: p.Finalization.SHA256,
		PreviewFailure: &ref, Finding: p.Finding(), UnitCount: 1, Units: []QualityCheckSurfaceReopenUnit{{Index: p.Artifact.Unit.Index, UnitID: p.Artifact.Unit.UnitID, Snapshot: p.Artifact.Unit}}}
}
func verifyPreviewReopen(root string, c *Catalog, r *QualityCheckSurfaceReopen, current bool) error {
	p, err := readPreviewTUFailure(root, c, *r.PreviewFailure, current)
	if err != nil {
		return err
	}
	want := previewReopenProjection(p, *r.PreviewFailure, r.ReopenID)
	if !reflect.DeepEqual(want, *r) {
		return fmt.Errorf("preview reopen differs from exact machine authorization")
	}
	return nil
}
func recordPreviewReopen(root string, c *Catalog, o QualityCheckSurfaceReopenOptions) (*QualityCheckSurfaceReopen, string, error) {
	if !reviewIDPattern.MatchString(o.ReopenID) || o.SurfaceReviewID != "" || o.IntegratedSurface || o.Finding != "" || len(o.Findings) != 0 || o.PreviousReopenID != "" || o.HistoricalDocuments.Path != "" {
		return nil, "", fmt.Errorf("preview authority cannot be mixed with Surface findings/replacement")
	}
	p, err := readPreviewTUFailure(root, c, o.PreviewFailure, true)
	if err != nil {
		return nil, "", err
	}
	if o.Locale != p.Locale || o.PreviousSnapshotID != p.Artifact.SnapshotID || len(o.UnitIDs) != 1 || o.UnitIDs[0] != p.Artifact.Unit.UnitID {
		return nil, "", fmt.Errorf("preview reopen requires exact one-TU/Snapshot/locale scope")
	}
	r := previewReopenProjection(p, o.PreviewFailure, o.ReopenID)
	path := qualityCheckSurfaceReopenPath(root, o.Locale, o.ReopenID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, "", err
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, "", err
	}
	if err := writeNewGenerationFile(path, append(b, '\n')); err != nil {
		return nil, "", err
	}
	rel, err := repositoryRelativePath(root, path)
	return &r, rel, err
}

func (r *QualityCheckSurfaceReopen) revisionFeedbackSource() string {
	if r.AuthoritySource == CurrentValidationAuthority {
		return CurrentValidationRevisionFeedback
	}
	if r.AuthoritySource == RepresentationMachineAuthority {
		return RepresentationRevisionFeedback
	}
	if r.AuthoritySource == "quality-check" {
		return QCRepresentationFeedback
	}
	if r.PreviewFailure != nil {
		return PreviewRevisionFeedback
	}
	return "surface_review"
}
func (r *QualityCheckSurfaceReopen) revisionEvidence() GlossaryArchiveReference {
	if r.AuthoritySource == CurrentValidationAuthority {
		return GlossaryArchiveReference{Path: "data/quality-check-reopens/" + r.Locale + "/" + r.ReopenID + ".json", SHA256: sum(currentValidationReopenBytes(r))}
	}
	if r.AuthoritySource == RepresentationMachineAuthority && r.ReplacementContext != nil && r.ReplacementContext.Machine != nil {
		return r.ReplacementContext.Machine.Options.Impact
	}
	if r.AuthoritySource == "quality-check" && r.ReplacementContext != nil && r.ReplacementContext.QualityCheck != nil {
		return r.ReplacementContext.QualityCheck.Results
	}
	if r.PreviewFailure != nil {
		return *r.PreviewFailure
	}
	return GlossaryArchiveReference{Path: r.SurfaceReviewPath, SHA256: r.SurfaceReviewSHA256}
}
func isReopenFeedback(s string) bool {
	return s == "surface_review" || s == PreviewRevisionFeedback || s == QCRepresentationFeedback || s == RepresentationRevisionFeedback || s == CurrentValidationRevisionFeedback
}
