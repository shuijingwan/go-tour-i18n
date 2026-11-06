package sitecontent

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"reflect"
	"strings"
)

type integratedScopeBuilder func(string, string, []Reference) (*IntegratedSurfaceScope, error)

// AttachIntegratedSurfaceAuthority supplies independent evidence/current scope
// only. It has no access to generation, attempt selection or QC mutations.
func AttachIntegratedSurfaceAuthority(c *i18n.Catalog) {
	attachIntegratedSurfaceAuthority(c, BuildIntegratedSurfaceScope)
}
func attachIntegratedSurfaceAuthority(c *i18n.Catalog, build integratedScopeBuilder) {
	c.HistoricalContextReader = readHistoricalLanguageContext
	c.SurfaceAuthorityReader = func(root, locale, reviewID string, current bool, inventory *i18n.Catalog) (*i18n.SurfaceReviewAuthority, error) {
		ref, err := ReferenceFile(root, "data/locale-language/"+locale+"/surface/"+reviewID+".json")
		if err != nil {
			return nil, err
		}
		var r IntegratedSurfaceReceipt
		if err := readReference(root, ref, &r); err != nil {
			return nil, err
		}
		if err := validateIntegratedSurfaceAuthority(root, inventory, r, build, current); err != nil {
			return nil, err
		}
		if r.ReviewID != reviewID || r.Scope.Locale != locale {
			return nil, fmt.Errorf("Surface review locale/id mismatch")
		}
		return &i18n.SurfaceReviewAuthority{Locale: locale, ReviewID: reviewID, Stage: r.Decision, Receipt: i18n.GlossaryArchiveReference{Path: ref.Path, SHA256: ref.SHA256}, ReceiptIdentity: r.Identity, ScopeIdentity: r.Scope.Identity, Evidence: i18n.GlossaryArchiveReference{Path: r.Evidence.Path, SHA256: r.Evidence.SHA256}, Context: surfaceContextReferences(r.Scope), Findings: r.ExactFindings}, nil
	}
}
func surfaceContextReferences(s IntegratedSurfaceScope) []i18n.GlossaryArchiveReference {
	refs := []Reference{s.UnifiedReview}
	refs = append(refs, s.Closures...)
	refs = append(refs, s.Runtime...)
	for _, a := range append(append([]SurfaceAsset{}, s.Carried...), s.Reviewed...) {
		refs = append(refs, a.Evidence...)
	}
	out := []i18n.GlossaryArchiveReference{}
	seen := map[string]bool{}
	for _, r := range refs {
		if !seen[r.Path] {
			out = append(out, i18n.GlossaryArchiveReference{Path: r.Path, SHA256: r.SHA256})
			seen[r.Path] = true
		}
	}
	return out
}
func validateIntegratedSurfaceAuthority(root string, c *i18n.Catalog, r IntegratedSurfaceReceipt, build integratedScopeBuilder, current bool) error {
	copy := r
	copy.Identity = ""
	s := r.Scope
	s.Identity = ""
	if r.Schema != IntegratedSurfaceSchema || r.Identity != identity(copy) || r.Scope.Identity != identity(s) || !component(r.ReviewID) || !component(r.Reviewer) {
		return fmt.Errorf("Integrated Surface receipt/scope identity mismatch")
	}
	if (r.Decision != "passed" && r.Decision != "failed") || (r.Decision == "failed" && len(r.Findings) == 0) || (r.Decision == "passed" && (len(r.Findings) > 0 || len(r.ExactFindings) > 0)) {
		return fmt.Errorf("Integrated Surface decision/finding mismatch")
	}
	if err := requireIndependentSession(root, r.Scope.Locale, r.Reviewer, "reviewer"); err != nil {
		return err
	}
	evidence, err := readRegular(root, r.Evidence.Path)
	if err != nil || digest(evidence) != r.Evidence.SHA256 || textBytes(evidence) != nil {
		return fmt.Errorf("Surface reviewer evidence changed")
	}
	for _, ref := range surfaceContextReferences(r.Scope) {
		// Historical integrity is the original independent decision, immutable
		// language artifacts and exact candidate bindings. Runtime, coverage,
		// canonical/status/UI/SEO projections are checked by current consumers.
		if !current && (ref.Path == GlobalPath || !strings.HasPrefix(ref.Path, "data/")) {
			continue
		}
		b, err := readRegular(root, ref.Path)
		if err != nil || digest(b) != ref.SHA256 {
			if current && r.Decision == "passed" && ref.Path == LocalePath(r.Scope.Locale) {
				baseline, e := Encode(r.Scope.ExistingCoverage)
				gate, e2 := ReferenceFile(root, "data/locale-language/"+r.Scope.Locale+"/surface/"+r.ReviewID+".json")
				if e == nil && e2 == nil && digest(baseline) == ref.SHA256 && proveReviewedActivationTransition(root, gate, r) == nil {
					continue
				}
			}
			if !current && verifyHistoricalActivationContext(root, r.Scope.Locale, ref) == nil {
				continue
			}
			return fmt.Errorf("Surface context changed: %s", ref.Path)
		}
	}
	if current {
		var want *IntegratedSurfaceScope
		if r.Decision == "passed" {
			gate, e := ReferenceFile(root, "data/locale-language/"+r.Scope.Locale+"/surface/"+r.ReviewID+".json")
			if e != nil {
				return e
			}
			want, err = currentReviewedScope(root, r, gate, build)
		} else {
			want, err = build(root, r.Scope.Locale, r.Scope.Closures)
		}
		if err != nil || !reflect.DeepEqual(r.Scope, *want) {
			return fmt.Errorf("Integrated Surface scope/closure stale")
		}
	}
	return validateIntegratedTUFindings(root, c, r)
}
func validateIntegratedTUFindings(root string, c *i18n.Catalog, r IntegratedSurfaceReceipt) error {
	if _, err := reviewedFindingAssets(root, &r); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, f := range r.ExactFindings {
		assets := append([]SurfaceAsset{}, r.Scope.Reviewed...)
		if f.Class == i18n.SurfaceTranslationUnitLanguage {
			assets = append(assets, r.Scope.Carried...)
		}
		if f.Class != i18n.SurfaceTranslationUnitLanguage {
			if len(f.Artifacts) > 0 {
				return fmt.Errorf("non-TU finding cannot claim TU artifacts")
			}
			continue
		}
		if len(f.Artifacts) == 0 || f.Finding == "" {
			return fmt.Errorf("TU finding requires explicit reviewed artifact scope")
		}
		for _, artifact := range f.Artifacts {
			if artifact.Unit.SourcePath != f.ID || seen[artifact.Unit.UnitID] {
				return fmt.Errorf("TU finding source/affected scope mismatch")
			}
			reviewed := false
			for _, a := range assets {
				for _, candidate := range a.CandidateArtifacts {
					if reflect.DeepEqual(candidate, artifact) {
						reviewed = true
					}
				}
			}
			if !reviewed {
				return fmt.Errorf("TU finding artifact was not in exact reviewed Surface scope")
			}
			if err := i18n.ValidateSurfaceCandidateArtifact(root, c, r.Scope.Locale, artifact); err != nil {
				return err
			}
			seen[artifact.Unit.UnitID] = true
		}
	}
	return nil
}

// BindIntegratedSurfaceCandidates binds the bytes presented to the reviewer.
// Physical-file assets require exact projected candidate SHA; representation
// adapters such as Tour explicitly identify their section before this call.
func BindIntegratedSurfaceCandidates(root string, c *i18n.Catalog, scope *IntegratedSurfaceScope, finalizations ...*i18n.QualityCheckFinalization) error {
	artifacts := []i18n.SurfaceCandidateArtifact{}
	for _, f := range finalizations {
		if f == nil || f.Locale != scope.Locale {
			return fmt.Errorf("Surface candidate locale mismatch")
		}
		for _, u := range f.Units {
			artifacts = append(artifacts, i18n.SurfaceCandidateArtifact{SnapshotID: f.SnapshotID, Unit: u.Snapshot})
		}
	}
	if err := i18n.ValidateSurfaceCandidatePackageArtifacts(root, c, scope.Locale, artifacts); err != nil {
		return err
	}
	for _, artifact := range artifacts {
		bound := 0
		for _, assets := range [][]SurfaceAsset{scope.Reviewed, scope.Carried} {
			for i := range assets {
				a := &assets[i]
				if a.SourcePath != artifact.Unit.SourcePath && a.ID != artifact.Unit.SourcePath {
					continue
				}
				// A context section adapter can explicitly bind its exact UnitID. File
				// projection instead requires the candidate's exact bytes.
				if (a.UnitID != "" && a.UnitID == artifact.Unit.UnitID) || (a.UnitID == "" && a.TargetSHA == artifact.Unit.CandidateSHA256) {
					a.SourcePath = artifact.Unit.SourcePath
					a.CandidateArtifacts = append(a.CandidateArtifacts, artifact)
					bound++
				}
			}
		}
		if bound != 1 {
			return fmt.Errorf("Surface candidate requires one exact presented target: %s", artifact.Unit.SourcePath)
		}
	}
	return nil
}
