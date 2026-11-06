package i18n

import (
	"fmt"
	"reflect"
	"strings"
)

const SurfaceTranslationUnitLanguage = "translation-unit-language"

// SurfaceCandidateArtifact is representation-neutral. Snapshot evidence binds
// the actual restored canonical bytes, never the protected model response.
type SurfaceCandidateArtifact struct {
	SnapshotID string                   `json:"snapshot_id"`
	Unit       QualityCheckSnapshotUnit `json:"unit"`
}
type SurfaceLanguageFinding struct {
	ID        string                     `json:"id"`
	Finding   string                     `json:"finding"`
	Class     string                     `json:"class,omitempty"`
	Artifacts []SurfaceCandidateArtifact `json:"artifacts,omitempty"`
}

// A read-only projection of an independent Surface receipt. The reader checks
// the original receipt/session/scope; shared code alone authorizes exact TUs.
type SurfaceReviewAuthority struct {
	Locale          string                     `json:"locale"`
	ReviewID        string                     `json:"review_id"`
	Stage           string                     `json:"stage"`
	Receipt         GlossaryArchiveReference   `json:"receipt"`
	ReceiptIdentity string                     `json:"receipt_identity"`
	ScopeIdentity   string                     `json:"scope_identity"`
	Evidence        GlossaryArchiveReference   `json:"evidence"`
	Context         []GlossaryArchiveReference `json:"context"`
	Findings        []SurfaceLanguageFinding   `json:"findings"`
}
type SurfaceAuthorityReader func(root, locale, reviewID string, current bool, inventory *Catalog) (*SurfaceReviewAuthority, error)

// ExactSourceTranslationUnit maps an explicit frozen source identity, not a
// route, fuzzy name or latest candidate. Tour section hashes disambiguate a
// multi-section physical article; identical matches are still rejected.
func (c *Catalog) ExactSourceTranslationUnit(path string, kind UnitKind, sha string) (*TranslationUnit, error) {
	if c == nil || path == "" || sha == "" {
		return nil, fmt.Errorf("exact source identity required")
	}
	// Inspect inventory directly: default Tour scheduling deliberately excludes
	// adapted packages. Scheduling must not hide mapping ambiguity.
	units := []*TranslationUnit{}
	for _, p := range c.Pages {
		units = append(units, &TranslationUnit{ID: p.ID, Kind: UnitKindPage, SourcePath: "_content/tour/" + p.Article, SourceSHA256: p.SourceSHA256, Source: p.Source})
	}
	for _, a := range c.Adapted {
		u := a.Unit
		u.Adapter = a.Adapter
		units = append(units, &u)
	}
	for _, e := range c.Examples {
		units = append(units, &TranslationUnit{ID: e.ID, Kind: UnitKindExample, SourcePath: e.SourcePath, SourceSHA256: e.SourceSHA256, Source: e.Source})
	}
	var found *TranslationUnit
	matches := 0
	for _, u := range units {
		if u.SourcePath == path && u.Kind == kind && u.SourceSHA256 == sha {
			found = u
			matches++
		}
	}
	if matches != 1 {
		return nil, fmt.Errorf("exact source mapping requires one TranslationUnit, got %d: %s", matches, path)
	}
	return found, nil
}

// ValidateSurfaceCandidateArtifact never trusts an artifact's claimed UnitID:
// exact source mapping and immutable Snapshot/validation prove it first.
func ValidateSurfaceCandidateArtifact(root string, c *Catalog, locale string, a SurfaceCandidateArtifact) error {
	return validateSurfaceCandidateArtifact(root, c, locale, a, nil, false)
}

// ValidateSurfaceCandidatePackageArtifacts binds a complete current inventory
// to its original Snapshot artifacts. Frozen provenance is not current language
// authorization: the Surface caller must also verify its current package closure.
func ValidateSurfaceCandidatePackageArtifacts(root string, c *Catalog, locale string, artifacts []SurfaceCandidateArtifact) error {
	_, _, _, units, err := translationAuthorizationScope(c)
	if err != nil || len(units) != len(artifacts) || len(units) == 0 {
		return fmt.Errorf("complete explicit registered package scope required for Surface candidates")
	}
	byID := map[string]QualityCheckSnapshotUnit{}
	for _, a := range artifacts {
		if _, duplicate := byID[a.Unit.UnitID]; duplicate {
			return fmt.Errorf("duplicate Surface candidate Unit")
		}
		byID[a.Unit.UnitID] = a.Unit
	}
	for _, u := range units {
		a, ok := byID[u.ID]
		if !ok || a.UnitKind != u.Kind || a.SourcePath != u.SourcePath || a.SourceSHA256 != u.SourceSHA256 {
			return fmt.Errorf("Surface candidate current package exact-set mismatch: %s", u.ID)
		}
	}
	if err := validateSurfaceCandidateArtifacts(root, c, locale, artifacts, true); err != nil {
		return err
	}
	return RequireCurrentGlossaryReview(root, locale)
}

// ValidateSurfaceCandidateArtifacts shares only reconciliation preparation for
// this call. Every artifact still passes exact mapping, Snapshot membership and
// the native per-Unit machine proof. No validation result survives the call.
func ValidateSurfaceCandidateArtifacts(root string, c *Catalog, locale string, artifacts []SurfaceCandidateArtifact) error {
	return validateSurfaceCandidateArtifacts(root, c, locale, artifacts, false)
}

func validateSurfaceCandidateArtifacts(root string, c *Catalog, locale string, artifacts []SurfaceCandidateArtifact, historical bool) error {
	machines := map[string]*machineSelection{}
	for _, a := range artifacts {
		if err := validateSurfaceCandidateArtifact(root, c, locale, a, machines, historical); err != nil {
			return err
		}
	}
	// Preparation reads frozen and mutable inputs. Rebind them before returning
	// so shared preparation cannot conceal a change during the batch.
	dependencies := map[GlossaryArchiveReference]bool{}
	contexts := map[*machineReconciliationContext]bool{}
	for _, m := range machines {
		refs := append(machineAuthorityReferences(m),
			GlossaryArchiveReference{Path: m.unit.Predecessor.CandidatePath, SHA256: m.unit.Predecessor.CandidateSHA256},
			GlossaryArchiveReference{Path: m.unit.Predecessor.ValidationPath, SHA256: m.unit.Predecessor.ValidationSHA256})
		for _, q := range m.receipt.QCInputs {
			refs = append(refs, GlossaryArchiveReference{Path: q.Path, SHA256: q.SHA256})
		}
		for _, ref := range refs {
			dependencies[ref] = true
		}
		contexts[m.ctx] = true
	}
	for ref := range dependencies {
		b, err := readSnapshotReferencedFile(root, ref.Path)
		if err != nil || sum(b) != ref.SHA256 {
			return fmt.Errorf("Surface machine dependency changed: %s", ref.Path)
		}
	}
	for ctx := range contexts {
		var b []byte
		var err error
		if historical {
			b, err = readArchivedGlossary(root, locale, ctx.currentGlossarySHA)
		} else {
			b, err = readSnapshotReferencedFile(root, glossaryReviewGlossaryPath(locale))
		}
		if err != nil || sum(b) != ctx.currentGlossarySHA {
			return fmt.Errorf("Surface machine glossary changed during invocation")
		}
		latest, err := selectLatestRetranslationUnits(root, ctx.resolution.HistoricalCatalog, locale, true)
		if err != nil || !reflect.DeepEqual(latest.selectedByID, ctx.oldLatest.selectedByID) {
			return fmt.Errorf("Surface machine historical selection changed during invocation: %v", err)
		}
	}
	if len(machines) > 0 && !historical {
		return RequireCurrentGlossaryReview(root, locale)
	}
	return nil
}

func validateSurfaceCandidateArtifact(root string, c *Catalog, locale string, a SurfaceCandidateArtifact, machines map[string]*machineSelection, historical bool) error {
	u, err := c.ExactSourceTranslationUnit(a.Unit.SourcePath, a.Unit.UnitKind, a.Unit.SourceSHA256)
	if err != nil {
		return err
	}
	if u.ID != a.Unit.UnitID {
		return fmt.Errorf("Surface artifact UnitID differs from exact source mapping")
	}
	return validateHistoricalSurfaceCandidateArtifact(root, c, locale, a, machines, historical)
}

// This validates original membership/bytes against an explicitly reconstructed
// historical inventory. It never grants current authorization or aliases IDs.
func ValidateHistoricalSurfaceCandidateArtifact(root string, c *Catalog, locale string, a SurfaceCandidateArtifact) error {
	return validateHistoricalSurfaceCandidateArtifact(root, c, locale, a, nil, true)
}

func validateHistoricalSurfaceCandidateArtifact(root string, c *Catalog, locale string, a SurfaceCandidateArtifact, machines map[string]*machineSelection, historical bool) error {
	// An integrated receipt can include several packages' immutable Snapshots.
	// Artifact identity lookup is inventory-wide; reopen/finalization separately
	// enforce the selected complete lifecycle scope.
	s, err := readQualityCheckSnapshot(root, locale, a.SnapshotID, false)
	if err != nil {
		return err
	}
	if s.ContentPackage != nil {
		// Resolve exactly the Snapshot's registered package era. An unscoped
		// inventory cannot validate package metadata, and today's membership
		// must not replace a predecessor's complete ordered scope.
		inventory, err := HistoricalTranslationPackageCatalog(c, s.ContentPackage.ID, s.ContentPackage.AuthoritySHA256)
		if err != nil {
			return err
		}
		s, err = readQualityCheckSnapshot(root, locale, a.SnapshotID, false, inventory)
		if err != nil {
			return err
		}
	}
	matched := false
	for _, v := range s.Units {
		if reflect.DeepEqual(v, a.Unit) {
			matched = true
		}
	}
	if !matched {
		return fmt.Errorf("Surface artifact differs from reviewed Snapshot")
	}
	if a.Unit.RepresentationReconciliation != nil {
		// A warm context never replaces verification of the claimed receipt.
		if _, err := readMachineReceipt(root, *a.Unit.RepresentationReconciliation); err != nil {
			return err
		}
		_, _, err = machineSnapshotEvidenceInWorld(root, c, locale, a.Unit, machines, historical)
	} else {
		_, err = readSnapshotUnitRepositoryEvidence(root, c, locale, a.Unit)
	}
	return err
}

func surfaceAuthorityReferences(a *SurfaceReviewAuthority) []GlossaryArchiveReference {
	return append([]GlossaryArchiveReference{a.Receipt, a.Evidence}, a.Context...)
}
func verifySurfaceAuthorityFiles(root string, c *Catalog, a *SurfaceReviewAuthority, current bool) error {
	for i, ref := range surfaceAuthorityReferences(a) {
		b, err := readSnapshotReferencedFile(root, ref.Path)
		if err != nil || sum(b) != ref.SHA256 {
			// Only context (after receipt and Markdown), only consumed historical
			// proof, and only an explicit adapter may prove the exact transition.
			if i >= 2 && !current {
				if _, err := readHistoricalContextReference(root, c, a.Locale, ref); err == nil {
					continue
				}
			}
			return fmt.Errorf("Surface authority reference stale: %s", ref.Path)
		}
	}
	return nil
}

type surfaceReopenResolution struct {
	authority    *SurfaceReviewAuthority
	unitFindings map[string]string
	catalog      *Catalog
	history      *HistoricalFinalizationIntegrity
}

// Select only the predecessor's complete package era. The caller's explicit
// package remains a constraint; its newer membership is not historical QC.
func surfaceReopenPredecessorCatalog(root string, c *Catalog, locale, snapshot string, authority *SurfaceReviewAuthority) (*Catalog, error) {
	inventory := *c
	inventory.TranslationScope = nil
	scoped, _, err := SelectFinalizedTranslationPackage(root, &inventory, locale, snapshot)
	if err != nil {
		return nil, err
	}
	if len(c.TranslationScope) > 0 {
		pkg, _, _, current, err := translationAuthorizationScope(c)
		if err != nil {
			return nil, err
		}
		selected, _, _, historical, err := translationAuthorizationScope(scoped)
		if err != nil {
			return nil, err
		}
		if selected != pkg {
			// A membership composition can retain a sibling's historical package
			// name. The already validated Surface reader proves the complete
			// current closure/composition and its exact Snapshot artifacts; this
			// check constrains that proof to the caller's explicit current owner.
			bound := false
			if authority != nil {
				for _, ref := range authority.Context {
					if strings.HasPrefix(ref.Path, "data/site-workflow/"+locale+"/"+pkg+"/closure/") {
						bound = true
					}
				}
			}
			if !bound || len(historical) >= len(current) {
				return nil, fmt.Errorf("Surface predecessor package differs from explicit content package")
			}
		}
		members := map[string]*TranslationUnit{}
		for _, u := range current {
			members[u.ID] = u
		}
		for _, old := range historical {
			u := members[old.ID]
			if u == nil || u.Kind != old.Kind || u.SourcePath != old.SourcePath || u.SourceSHA256 != old.SourceSHA256 {
				return nil, fmt.Errorf("Surface predecessor Unit outside explicit current package membership: %s", old.ID)
			}
		}
	}
	return scoped, nil
}

func resolveSurfaceReopenAuthority(root string, c *Catalog, o QualityCheckSurfaceReopenOptions, current bool) (*surfaceReopenResolution, error) {
	findings, err := surfaceReopenSelectedFindings(o)
	if err != nil {
		return nil, err
	}
	if c == nil || c.SurfaceAuthorityReader == nil {
		return nil, fmt.Errorf("Integrated Surface authority reader required")
	}
	a, err := c.SurfaceAuthorityReader(root, o.Locale, o.SurfaceReviewID, current, c)
	if err != nil {
		return nil, err
	}
	if a == nil || a.Locale != o.Locale || a.ReviewID != o.SurfaceReviewID || a.Stage != "failed" || a.ReceiptIdentity == "" || a.ScopeIdentity == "" {
		return nil, fmt.Errorf("Integrated Surface receipt identity/stage mismatch")
	}
	if a.Receipt.Path != "data/locale-language/"+o.Locale+"/surface/"+o.SurfaceReviewID+".json" {
		return nil, fmt.Errorf("Integrated Surface receipt path mismatch")
	}
	if err := verifySurfaceAuthorityFiles(root, c, a, current); err != nil {
		return nil, err
	}
	scoped, err := surfaceReopenPredecessorCatalog(root, c, o.Locale, o.PreviousSnapshotID, a)
	if err != nil {
		return nil, err
	}
	var history *HistoricalFinalizationIntegrity
	if current {
		authorization, err := RequireCurrentTranslationAuthorization(root, scoped, o.Locale, o.PreviousSnapshotID)
		if err != nil {
			return nil, fmt.Errorf("Integrated Surface predecessor current authorization: %w", err)
		}
		history = &authorization.Historical
	} else {
		history, err = ReadHistoricalTranslationIntegrity(root, scoped, o.Locale, o.PreviousSnapshotID)
		if err != nil {
			return nil, err
		}
	}
	selected := map[string]bool{}
	for _, id := range o.UnitIDs {
		if selected[id] {
			return nil, fmt.Errorf("duplicate affected UnitID")
		}
		selected[id] = true
	}
	affected := map[string]bool{}
	unitFindings := map[string]string{}
	requested := map[string]bool{}
	matched := map[string]bool{}
	for _, finding := range findings {
		requested[finding] = true
	}
	for _, f := range a.Findings {
		if !requested[f.Finding] {
			continue
		}
		if f.Class != SurfaceTranslationUnitLanguage || len(f.Artifacts) == 0 {
			return nil, fmt.Errorf("non-TU Surface finding remains a package/Surface blocker")
		}
		matched[f.Finding] = true
		for _, artifact := range f.Artifacts {
			if artifact.Unit.SourcePath != f.ID || artifact.SnapshotID != o.PreviousSnapshotID || affected[artifact.Unit.UnitID] {
				return nil, fmt.Errorf("Surface finding exact source/Snapshot/affected scope mismatch")
			}
			u, err := c.ExactSourceTranslationUnit(artifact.Unit.SourcePath, artifact.Unit.UnitKind, artifact.Unit.SourceSHA256)
			if err != nil || u.ID != artifact.Unit.UnitID {
				return nil, fmt.Errorf("Surface artifact differs from exact current source mapping: %v", err)
			}
			if err := ValidateHistoricalSurfaceCandidateArtifact(root, scoped, o.Locale, artifact); err != nil {
				return nil, err
			}
			affected[artifact.Unit.UnitID] = true
			unitFindings[artifact.Unit.UnitID] = f.Finding
		}
	}
	if len(matched) != len(requested) || len(affected) == 0 || !reflect.DeepEqual(selected, affected) {
		return nil, fmt.Errorf("Surface finding requires exact explicit affected TU scope and exact findings")
	}
	if current {
		if err := finalizationMatchesLatestPromotionCandidates(root, scoped, o.Locale, o.PreviousSnapshotID); err != nil {
			return nil, err
		}
	}
	return &surfaceReopenResolution{authority: a, unitFindings: unitFindings, catalog: scoped, history: history}, nil
}
