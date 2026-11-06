package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// Additive successor reopen evidence, not a new lifecycle or a replacement
// for the immutable old Snapshot/finding. Both IDs remain explicit.
type TranslationUnitReplacement struct {
	OldUnitID            string `json:"old_unit_id"`
	CurrentUnitID        string `json:"current_unit_id"`
	Package              string `json:"package"`
	PackageSHA256        string `json:"package_sha256"`
	SourcePath           string `json:"source_path"`
	SourceSHA256         string `json:"source_sha256"`
	StableIndex          int    `json:"stable_index"`
	Route                string `json:"route"`
	OldInputSHA256       string `json:"old_input_sha256"`
	CurrentInputSHA256   string `json:"current_input_sha256"`
	ReconciliationSHA256 string `json:"reconciliation_sha256"`
}
type TranslationUnitReplacementContext struct {
	Machine             *RepresentationRevisionAuthority  `json:"machine,omitempty"`
	QualityCheck        *QualityCheckReplacementAuthority `json:"quality_check,omitempty"`
	PredecessorReopen   GlossaryArchiveReference          `json:"predecessor_reopen"`
	HistoricalDocuments GlossaryArchiveReference          `json:"historical_documents"`
}
type TranslationUnitReplacementResolution struct {
	HistoricalCatalog *Catalog
	Units             map[string]TranslationUnitReplacement // keyed by historical identity
	// EventCatalog is the reconciliation's frozen current descriptor, not
	// today's current descriptor. A source provider may expose it only after
	// verifying the complete canonical successor lineage to the supplied
	// current inventory. Current reconciliation never consumes this override.
	EventCatalog        *Catalog
	SuccessorReferences []GlossaryArchiveReference
}

// A representation adapter proves reconciliation; shared code validates all
// historical Snapshot/QC/Surface evidence using that explicit frozen catalog.
type TranslationUnitReplacementResolver func(string, *Catalog, GlossaryArchiveReference) (*TranslationUnitReplacementResolution, error)
type ReplacementRevisionAdapter interface {
	RebaseRevisionPredecessor(TranslationUnitAdapter, []byte) ([]byte, error)
}

func replacementResolution(root string, c *Catalog, context *TranslationUnitReplacementContext) (*TranslationUnitReplacementResolution, error) {
	return replacementResolutionInWorld(root, c, context, false)
}

func replacementResolutionInWorld(root string, c *Catalog, context *TranslationUnitReplacementContext, historical bool) (*TranslationUnitReplacementResolution, error) {
	if c == nil || c.ReplacementResolver == nil || context == nil {
		return nil, fmt.Errorf("explicit replacement resolver/context required")
	}
	b, err := readSnapshotReferencedFile(root, context.HistoricalDocuments.Path)
	if err != nil || sum(b) != context.HistoricalDocuments.SHA256 {
		return nil, fmt.Errorf("historical representation evidence stale")
	}
	r, err := c.ReplacementResolver(root, c, context.HistoricalDocuments)
	if err != nil {
		return nil, err
	}
	if r == nil || r.HistoricalCatalog == nil || len(r.Units) == 0 {
		return nil, fmt.Errorf("empty replacement reconciliation")
	}
	if historical && r.EventCatalog != nil {
		if len(r.SuccessorReferences) == 0 {
			return nil, fmt.Errorf("frozen reconciliation current world requires exact successor authority")
		}
		for _, ref := range r.SuccessorReferences {
			b, err := readSnapshotReferencedFile(root, ref.Path)
			if err != nil || !validSHA256(ref.SHA256) || sum(b) != ref.SHA256 {
				return nil, fmt.Errorf("reconciliation successor authority stale: %s", ref.Path)
			}
		}
		c = r.EventCatalog
	}
	for oldID, p := range r.Units {
		old, e := r.HistoricalCatalog.Unit(oldID)
		current, e2 := c.ExactSourceTranslationUnit(p.SourcePath, UnitKindPage, p.SourceSHA256)
		if e != nil || e2 != nil || old.Kind != UnitKindPage || old.SourcePath != p.SourcePath || old.SourceSHA256 != p.SourceSHA256 || p.OldUnitID != oldID || current.ID != p.CurrentUnitID || p.Package == "" || p.PackageSHA256 == "" || p.StableIndex < 1 || p.Route == "" || p.ReconciliationSHA256 == "" || !bytes.Equal(old.Source, current.Source) {
			return nil, fmt.Errorf("replacement exact source/descriptor identity mismatch: %s", oldID)
		}
		for _, inventory := range []struct {
			c  *Catalog
			id string
		}{{c, p.CurrentUnitID}, {r.HistoricalCatalog, p.OldUnitID}} {
			matches := 0
			matchedAuthorities := map[string]bool{}
			for _, descriptor := range append(append([]TranslationPackageDescriptor{}, inventory.c.TranslationPackages...), inventory.c.HistoricalTranslationPackages...) {
				if descriptor.ID != p.Package || descriptor.AuthoritySHA256 != p.PackageSHA256 {
					continue
				}
				key := descriptor.ID + ":" + descriptor.AuthoritySHA256
				if matchedAuthorities[key] {
					continue
				}
				found := 0
				for _, unit := range descriptor.Units {
					if unit.Unit.ID == inventory.id && unit.Unit.Kind == UnitKindPage && unit.StableIndex == p.StableIndex && unit.Route == p.Route && unit.Unit.SourcePath == p.SourcePath && unit.Unit.SourceSHA256 == p.SourceSHA256 {
						found++
					}
				}
				// A scoped historical projection may also retain its original
				// discovery descriptor. Count the exact authority once; duplicate
				// Units within that descriptor still fail the uniqueness check.
				if found > 0 {
					matchedAuthorities[key] = true
					matches += found
				}
			}
			if matches != 1 {
				return nil, fmt.Errorf("replacement package/registry mapping is not unique")
			}
		}
		glossary := &Glossary{}
		a, e := prepareTranslationUnitInput(old, glossary)
		b, e2 := prepareTranslationUnitInput(current, glossary)
		if e != nil || e2 != nil || sum(canonicalizeRetranslationArtifactEOF([]byte(a.Text))) != p.OldInputSHA256 || sum(canonicalizeRetranslationArtifactEOF([]byte(b.Text))) != p.CurrentInputSHA256 || (oldID == current.ID) != (p.OldInputSHA256 == p.CurrentInputSHA256) {
			return nil, fmt.Errorf("replacement protected representation identity mismatch")
		}
	}
	if (context.QualityCheck != nil || context.Machine != nil) && len(c.TranslationScope) == 0 {
		// QC successors may have an unfinalized B/C/D predecessor, so the
		// Surface finalization-based scope selector cannot supply its package.
		// Use only the fully verified replacement package, never selected IDs or
		// path guessing. This preserves sealed A carry-forward in that package's
		// authorization namespace for export, Bundle and retry alike.
		pkg := ""
		for _, p := range r.Units {
			if pkg != "" && pkg != p.Package {
				return nil, fmt.Errorf("QC reconciliation requires one exact package")
			}
			pkg = p.Package
		}
		historical := *r.HistoricalCatalog
		if err := historical.UseTranslationPackage(pkg); err != nil {
			return nil, err
		}
		copy := *r
		copy.HistoricalCatalog = &historical
		r = &copy
	}
	return r, nil
}

func replacementPredecessor(root string, c *Catalog, locale string, r *QualityCheckSurfaceReopen) (*QualityCheckSurfaceReopen, *TranslationUnitReplacementResolution, error) {
	resolution, err := replacementResolution(root, c, r.ReplacementContext)
	if err != nil {
		return nil, nil, err
	}
	ref := r.ReplacementContext.PredecessorReopen
	b, err := readSnapshotReferencedFile(root, ref.Path)
	if err != nil || sum(b) != ref.SHA256 {
		return nil, nil, fmt.Errorf("predecessor reopen evidence stale")
	}
	var prior QualityCheckSurfaceReopen
	if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &prior); err != nil {
		return nil, nil, err
	}
	if prior.ReplacementContext != nil || prior.ReopenID == r.ReopenID || ref.Path != filepath.ToSlash(filepath.Join("data", "quality-check-reopens", locale, prior.ReopenID+".json")) {
		return nil, nil, fmt.Errorf("invalid replacement predecessor reopen")
	}
	verified, err := readCurrentQualityCheckSurfaceReopen(root, resolution.HistoricalCatalog, locale, prior.ReopenID)
	return verified, resolution, err
}

func verifyReplacementSurfaceReopen(root string, c *Catalog, r *QualityCheckSurfaceReopen) error {
	if r.AuthoritySource == RepresentationMachineAuthority {
		return verifyRepresentationRevision(root, c, r)
	}
	if r.AuthoritySource == "quality-check" {
		return verifyQualityCheckReplacement(root, c, r)
	}
	if r.AuthoritySource != "" || r.ReplacementContext.QualityCheck != nil || r.ReplacementContext.Machine != nil {
		return fmt.Errorf("mixed/unknown replacement authority")
	}
	prior, resolution, err := replacementPredecessor(root, c, r.Locale, r)
	if err != nil {
		return err
	}
	if prior.IntegratedAuthority == nil {
		return fmt.Errorf("replacement requires exact Integrated Surface artifacts")
	}
	want := *prior
	want.ReopenID, want.ReplacementContext = r.ReopenID, r.ReplacementContext
	want.Units = append([]QualityCheckSurfaceReopenUnit(nil), prior.Units...)
	changed := 0
	seen := map[string]bool{}
	for i, u := range prior.Units {
		p, ok := resolution.Units[u.UnitID]
		if !ok || seen[p.CurrentUnitID] || p.StableIndex != u.Index || u.Snapshot.SourcePath != p.SourcePath || u.Snapshot.SourceSHA256 != p.SourceSHA256 || u.Snapshot.UnitKind != UnitKindPage {
			return fmt.Errorf("replacement exact predecessor membership mismatch")
		}
		seen[p.CurrentUnitID] = true
		want.Units[i].UnitID = p.CurrentUnitID
		if p.CurrentUnitID != p.OldUnitID {
			want.Units[i].Replacement = &p
			changed++
		}
		// Anchor the old representation to the actually reviewed input, never
		// merely to a document that happens to share a path.
		e, err := readSnapshotUnitRepositoryEvidence(root, resolution.HistoricalCatalog, r.Locale, u.Snapshot)
		if err != nil || e.record.InputSHA256 != p.OldInputSHA256 {
			return fmt.Errorf("replacement historical reviewed input mismatch: %s: %v", u.UnitID, err)
		}
	}
	if changed == 0 || !reflect.DeepEqual(&want, r) {
		return fmt.Errorf("successor replacement authorization differs from exact reconciliation")
	}
	return nil
}

func recordReplacementSurfaceReopen(root string, c *Catalog, o QualityCheckSurfaceReopenOptions) (*QualityCheckSurfaceReopen, string, error) {
	if err := ValidateLocaleName(o.Locale); err != nil {
		return nil, "", err
	}
	if !reviewIDPattern.MatchString(o.ReopenID) || !reviewIDPattern.MatchString(o.PreviousReopenID) || !o.IntegratedSurface {
		return nil, "", fmt.Errorf("successor requires distinct stable reopen IDs and Integrated authority")
	}
	ref := GlossaryArchiveReference{Path: filepath.ToSlash(filepath.Join("data", "quality-check-reopens", o.Locale, o.PreviousReopenID+".json"))}
	b, err := readSnapshotReferencedFile(root, ref.Path)
	if err != nil {
		return nil, "", err
	}
	ref.SHA256 = sum(b)
	context := &TranslationUnitReplacementContext{PredecessorReopen: ref, HistoricalDocuments: o.HistoricalDocuments}
	probe := &QualityCheckSurfaceReopen{Locale: o.Locale, ReopenID: o.ReopenID, ReplacementContext: context}
	prior, resolution, err := replacementPredecessor(root, c, o.Locale, probe)
	if err != nil {
		return nil, "", err
	}
	findings, err := surfaceReopenSelectedFindings(o)
	if err != nil {
		return nil, "", err
	}
	oldFindings, err := surfaceReopenSelectedFindings(QualityCheckSurfaceReopenOptions{Finding: prior.Finding, Findings: prior.Findings, IntegratedSurface: true})
	if err != nil {
		return nil, "", err
	}
	if prior.PreviousSnapshotID != o.PreviousSnapshotID || prior.SurfaceReviewID != o.SurfaceReviewID || !reflect.DeepEqual(findings, oldFindings) {
		return nil, "", fmt.Errorf("successor finding/Snapshot identity mismatch")
	}
	r := *prior
	r.ReopenID, r.ReplacementContext = o.ReopenID, context
	r.Units = append([]QualityCheckSurfaceReopenUnit(nil), prior.Units...)
	selected := map[string]bool{}
	for _, id := range o.UnitIDs {
		if selected[id] {
			return nil, "", fmt.Errorf("duplicate successor Unit")
		}
		selected[id] = true
	}
	for i, u := range prior.Units {
		p, ok := resolution.Units[u.UnitID]
		if !ok || !selected[p.CurrentUnitID] {
			return nil, "", fmt.Errorf("successor exact mapped scope required")
		}
		r.Units[i].UnitID = p.CurrentUnitID
		if p.CurrentUnitID != p.OldUnitID {
			r.Units[i].Replacement = &p
		}
	}
	if len(selected) != len(r.Units) {
		return nil, "", fmt.Errorf("successor exact mapped scope required")
	}
	if err := verifyReplacementSurfaceReopen(root, c, &r); err != nil {
		return nil, "", err
	}
	if err := replacementAuthorizationUnconsumed(root, c, &r); err != nil {
		return nil, "", err
	}
	path := qualityCheckSurfaceReopenPath(root, o.Locale, o.ReopenID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, "", err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, "", err
	}
	if err := writeNewGenerationFile(path, append(data, '\n')); err != nil {
		return nil, "", err
	}
	rel, err := repositoryRelativePath(root, path)
	return &r, rel, err
}

func replacementAuthorizationUnconsumed(root string, c *Catalog, r *QualityCheckSurfaceReopen) error {
	if r.originalQCReplacement() {
		return qualityCheckReplacementUnconsumed(root, c, r)
	}
	prior, resolution, err := replacementPredecessor(root, c, r.Locale, r)
	if err != nil {
		return err
	}
	base := filepath.Join(root, "data", "retranslation-runs", r.Locale)
	entries, err := os.ReadDir(base)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		dir := filepath.Join(base, entry.Name())
		m, err := readRetranslationProcessManifest(dir, r.Locale, entry.Name())
		if err != nil {
			return err
		}
		bound := false
		for _, record := range m.Units {
			if record.RevisionAuthorizationID == prior.ReopenID || record.RevisionAuthorizationID == r.ReopenID {
				bound = true
			}
		}
		if !bound {
			continue
		}
		if err := ensureNoGenerationImportPending(dir); err != nil {
			return err
		}
		for _, name := range []string{"raw-responses", "generation-imports"} {
			if _, err := os.Lstat(filepath.Join(dir, name)); err == nil {
				return fmt.Errorf("replacement authorization already has formal generation state; continue existing lifecycle")
			} else if !os.IsNotExist(err) {
				return err
			}
		}
	}
	if err := finalizationMatchesLatestPromotionCandidates(root, resolution.HistoricalCatalog, r.Locale, r.PreviousSnapshotID); err != nil {
		return err
	}
	latest, err := selectLatestRetranslationUnits(root, c, r.Locale)
	if err != nil {
		return err
	}
	for _, u := range r.Units {
		if u.Replacement == nil {
			continue
		}
		if _, ok := latest.selectedByID[u.UnitID]; ok {
			return fmt.Errorf("replacement Unit already has formal results; continue existing lifecycle")
		}
	}
	return nil
}

// Revalidate a historical finalized predecessor only through the successor
// authorization bound by current selected batches. This does not carry its A
// to a replacement ID; ordinary QC exact-identity rules remain unchanged.
func replacementCatalogForPredecessor(root string, c *Catalog, locale, previous string, snapshot *QualityCheckSnapshotManifest) (*Catalog, error) {
	var context *TranslationUnitReplacementContext
	for _, u := range snapshot.Units {
		m, err := readRetranslationProcessManifest(filepath.Join(root, "data", "retranslation-runs", locale, u.SelectedBatchID), locale, u.SelectedBatchID)
		if err != nil {
			return nil, err
		}
		for _, record := range m.Units {
			if record.UnitID != u.UnitID || record.PreviousSnapshotID != previous || (record.RevisionFeedbackSource != "surface_review" && record.RevisionFeedbackSource != QCRepresentationFeedback && record.RevisionFeedbackSource != RepresentationRevisionFeedback) {
				continue
			}
			r, err := readCurrentQualityCheckSurfaceReopen(root, c, locale, record.RevisionAuthorizationID)
			if err != nil {
				return nil, err
			}
			if r.ReplacementContext == nil {
				continue
			}
			if r.PreviousSnapshotID != previous || r.predecessorUnitID(u.UnitID) == "" {
				return nil, fmt.Errorf("replacement QC predecessor binding mismatch")
			}
			if context != nil && !reflect.DeepEqual(context, r.ReplacementContext) {
				return nil, fmt.Errorf("ambiguous replacement QC predecessor")
			}
			context = r.ReplacementContext
		}
	}
	if context == nil {
		return c, nil
	}
	resolution, err := replacementResolution(root, c, context)
	if err != nil {
		return nil, err
	}
	return resolution.HistoricalCatalog, nil
}

// The historical evidence catalog is local to this explicit successor. Never
// add old IDs to the current scheduling inventory or alias Catalog.Unit.
func surfaceRevisionCatalog(root string, c *Catalog, locale, reopenID string) (*Catalog, *QualityCheckSurfaceReopen, error) {
	r, err := readCurrentQualityCheckSurfaceReopen(root, c, locale, reopenID)
	if err != nil {
		return nil, nil, err
	}
	if r.ReplacementContext == nil {
		return c, r, nil
	}
	resolution, err := replacementResolution(root, c, r.ReplacementContext)
	if err != nil {
		return nil, nil, err
	}
	return resolution.HistoricalCatalog, r, nil
}

func (r *QualityCheckSurfaceReopen) predecessorUnitID(id string) string {
	for _, u := range r.Units {
		if u.UnitID == id {
			return u.Snapshot.UnitID
		}
	}
	return ""
}
