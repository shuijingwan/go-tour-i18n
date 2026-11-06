package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Bind the CAS bytes to the same locale state whose gates were checked. A
// concurrent activation in this reviewed scope must not silently drop a
// completion by combining an earlier Locale with a later prior SHA.
func checkActivationPreflightState(checked *Locale, prior []byte, missing bool) error {
	if missing {
		if len(checked.Packages) == 0 {
			return nil
		}
		return fmt.Errorf("activation preflight state changed")
	}
	var captured Locale
	if err := StrictJSON(prior, &captured); err != nil {
		return err
	}
	if !reflect.DeepEqual(*checked, captured) {
		return fmt.Errorf("activation preflight state changed")
	}
	return nil
}

// Prove append or exact baseline replacement: retained coverage stays exact,
// and each prospective slot binds this receipt or its sealed predecessor.
// Closure contents are fully revalidated by the surrounding activation/scope
// gates. This non-recursive proof checks their immutable bindings, never grants
// language authority and never substitutes for CheckPackageClosure.
func proveReviewedActivationTransition(root string, gate Reference, r IntegratedSurfaceReceipt) error {
	global, err := reviewedActivationGlobal(root)
	if err != nil {
		return err
	}
	state, err := readLocaleAuthority(root, global, r.Scope.Locale)
	if err != nil {
		return err
	}
	actual, actualErr := readRegular(root, LocalePath(r.Scope.Locale))
	if actualErr != nil && !os.IsNotExist(actualErr) {
		return actualErr
	}
	if err := proveReviewedActivationState(root, gate, r, global, *state, actual); err != nil {
		return err
	}
	after, afterErr := readRegular(root, LocalePath(r.Scope.Locale))
	if (afterErr != nil && !os.IsNotExist(afterErr)) || os.IsNotExist(afterErr) != os.IsNotExist(actualErr) || !bytes.Equal(after, actual) {
		return fmt.Errorf("activation content scope changed during proof")
	}
	return nil
}

// The current reader supplies live bytes. Historical traversal may supply only
// predecessor bytes reconstructed by an already proven parent transition.
// All receipt, closure, baseline and public-context checks remain identical.
func proveReviewedActivationState(root string, gate Reference, r IntegratedSurfaceReceipt, global *Global, state Locale, actual []byte) error {
	var boundReceipt IntegratedSurfaceReceipt
	if err := readReference(root, gate, &boundReceipt); err != nil {
		return err
	}
	if !reflect.DeepEqual(r, boundReceipt) {
		return fmt.Errorf("activation Surface receipt binding changed")
	}
	if r.Scope.Schema != IntegratedSurfaceSchema || r.Decision != "passed" || len(r.Findings) != 0 || len(r.ExactFindings) != 0 || len(r.Scope.Closures) == 0 {
		return fmt.Errorf("activation transition requires an exact complete Surface PASS")
	}
	if err := validateHistoricalIntegratedSurface(root, r.Scope.Locale, gate, r.Scope.Closures[0]); err != nil {
		return err
	}
	baseline, err := reviewedActivationBaseline(root, global, r, state)
	if err != nil {
		return err
	}
	globalBound := false
	for _, ref := range r.Scope.Runtime {
		globalBound = globalBound || ref.Path == GlobalPath
	}
	if !globalBound {
		return fmt.Errorf("activation Surface global context missing")
	}
	for _, ref := range surfaceContextReferences(r.Scope) {
		if ref.Path == LocalePath(r.Scope.Locale) {
			if ref.SHA256 != digest(baseline) {
				return fmt.Errorf("Surface baseline bytes cannot be reconstructed")
			}
			continue
		}
		b, err := readRegular(root, ref.Path)
		if err != nil || digest(b) != ref.SHA256 {
			return fmt.Errorf("Surface context changed: %s", ref.Path)
		}
	}
	// The global authority is also an exact runtime/public-context reference.
	if err := ValidateLocale(global, r.Scope.ExistingCoverage); err != nil {
		return err
	}
	if state.Locale != r.Scope.Locale || r.Scope.ExistingCoverage.Locale != r.Scope.Locale || len(state.Packages) < len(r.Scope.ExistingCoverage.Packages) {
		return fmt.Errorf("activation baseline coverage changed")
	}
	want := r.Scope.ExistingCoverage
	want.Packages = append([]Completion{}, want.Packages...)
	if !reflect.DeepEqual(state.Packages[:len(want.Packages)], want.Packages) {
		return fmt.Errorf("activation baseline completion changed")
	}
	closures := map[string]struct {
		ref Reference
		c   PackageClosure
	}{}
	for _, ref := range r.Scope.Closures {
		var c PackageClosure
		if err := readReference(root, ref, &c); err != nil {
			return err
		}
		copy := c
		copy.Identity = ""
		if c.Schema != ClosureSchema || c.Identity != identity(copy) || c.Locale != r.Scope.Locale || ref.Path != workflowPath(c.Locale, c.Package, "closure", c.Identity) {
			return fmt.Errorf("activation closure binding changed")
		}
		if _, exists := closures[c.Package]; exists {
			return fmt.Errorf("duplicate activation closure package")
		}
		closures[c.Package] = struct {
			ref Reference
			c   PackageClosure
		}{ref, c}
	}
	for _, c := range state.Packages[len(want.Packages):] {
		bound, ok := closures[c.Package]
		if !ok {
			return fmt.Errorf("activation outside reviewed closures")
		}
		p, err := packageByID(global, c.Package)
		if err != nil || p.Identity != bound.c.PackageSHA {
			return fmt.Errorf("activation package authority changed")
		}
		refs := []Reference{bound.ref, gate}
		exact := Completion{Package: p.ID, State: "complete", PackageIdentity: p.Identity, SourceIdentity: p.SourceIdentity, Surfaces: p.Surfaces, Routes: p.Routes, RouteFamilies: routeFamilies(p.ID), EvidenceKind: PackageCompletionKind, Evidence: refs, EvidenceIdentity: identity(refs), ContextIdentity: bound.c.Identity}
		if !reflect.DeepEqual(c, exact) {
			if err := checkReviewedPredecessorCompletion(root, global, r.Scope.Locale, bound.c, c); err != nil {
				return fmt.Errorf("activation completion is not bound to the reviewed transaction: %w", err)
			}
		}
		want.Packages = append(want.Packages, c)
	}
	expected, err := Encode(want)
	if err != nil {
		return err
	}
	if actual == nil && len(want.Packages) == 0 {
		return nil // Existing locale identity, first sparse activation.
	}
	if !bytes.Equal(actual, expected) {
		return fmt.Errorf("activation content scope bytes changed")
	}
	return nil
}

// A prospective closure may replace only its exact sealed structured baseline.
// This non-recursive proof binds historical identities. Full historical content
// integrity and current eligibility are checked by the prospective closure in
// currentReviewedScope's builder, including its sealed structured baseline.
func checkReviewedPredecessorCompletion(root string, g *Global, locale string, successor PackageClosure, old Completion) error {
	if successor.StructuredBaseline == nil || old.Package != successor.Package || len(old.Evidence) != 2 || old.Evidence[0] != *successor.StructuredBaseline {
		return fmt.Errorf("completion is not the exact reviewed closure baseline")
	}
	var baseline PackageClosure
	if err := readReference(root, *successor.StructuredBaseline, &baseline); err != nil {
		return err
	}
	copy := baseline
	copy.Identity = ""
	if baseline.Schema != ClosureSchema || baseline.Identity != identity(copy) || successor.StructuredBaseline.Path != workflowPath(locale, successor.Package, "closure", baseline.Identity) || baseline.Locale != locale || baseline.Package != successor.Package || baseline.PackageSHA != old.PackageIdentity || baseline.Identity != old.ContextIdentity {
		return fmt.Errorf("predecessor completion/closure identity mismatch")
	}
	if err := ValidateLocale(g, Locale{Schema: LocaleSchema, Locale: locale, Packages: []Completion{old}}); err != nil {
		return err
	}
	return validateHistoricalIntegratedSurface(root, locale, old.Evidence[1], old.Evidence[0])
}

// existing_coverage omits prospective packages. Reconstruct their exact old
// completions from sealed baseline closures and independent historical PASS,
// retaining the current slot order across replacement. No current old-TU
// eligibility or mutable projection participates in this byte proof.
func reviewedActivationBaseline(root string, g *Global, r IntegratedSurfaceReceipt, state Locale) ([]byte, error) {
	baseline := r.Scope.ExistingCoverage
	baseline.Packages = append([]Completion{}, baseline.Packages...)
	closures := map[string]PackageClosure{}
	for _, ref := range r.Scope.Closures {
		var c PackageClosure
		if err := readReference(root, ref, &c); err != nil {
			return nil, err
		}
		closures[c.Package] = c
	}
	if len(state.Packages) < len(baseline.Packages) {
		return nil, fmt.Errorf("activation baseline coverage changed")
	}
	for _, current := range state.Packages[len(baseline.Packages):] {
		c, ok := closures[current.Package]
		if !ok {
			return nil, fmt.Errorf("activation outside reviewed closures")
		}
		if c.StructuredBaseline == nil {
			continue
		}
		var old PackageClosure
		if err := readReference(root, *c.StructuredBaseline, &old); err != nil {
			return nil, err
		}
		var p Package
		for _, candidate := range append(append([]Package{}, g.Packages...), g.HistoricalPackages...) {
			if candidate.ID == old.Package && candidate.Identity == old.PackageSHA {
				p = candidate
			}
		}
		if p.ID == "" {
			return nil, fmt.Errorf("unknown reviewed predecessor package")
		}
		entries, err := os.ReadDir(pathJoin(root, "data/locale-language/"+r.Scope.Locale+"/surface"))
		if err != nil {
			return nil, err
		}
		var match *Completion
		for _, entry := range entries {
			if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".json") {
				return nil, fmt.Errorf("unsafe predecessor Surface receipt")
			}
			gate, err := ReferenceFile(root, "data/locale-language/"+r.Scope.Locale+"/surface/"+entry.Name())
			if err != nil {
				return nil, err
			}
			var receipt IntegratedSurfaceReceipt
			if err := readReference(root, gate, &receipt); err != nil {
				return nil, err
			}
			if receipt.Decision != "passed" {
				continue
			}
			included := false
			for _, ref := range receipt.Scope.Closures {
				included = included || ref == *c.StructuredBaseline
			}
			if !included {
				continue
			}
			if err := validateHistoricalIntegratedSurface(root, r.Scope.Locale, gate, *c.StructuredBaseline); err != nil {
				return nil, err
			}
			refs := []Reference{*c.StructuredBaseline, gate}
			v := Completion{Package: p.ID, State: "complete", PackageIdentity: p.Identity, SourceIdentity: p.SourceIdentity, Surfaces: p.Surfaces, Routes: p.Routes, RouteFamilies: routeFamilies(p.ID), EvidenceKind: PackageCompletionKind, Evidence: refs, EvidenceIdentity: identity(refs), ContextIdentity: old.Identity}
			if match != nil && !reflect.DeepEqual(*match, v) {
				return nil, fmt.Errorf("ambiguous reviewed predecessor completion")
			}
			match = &v
		}
		if match == nil {
			return nil, fmt.Errorf("reviewed predecessor Surface PASS missing")
		}
		baseline.Packages = append(baseline.Packages, *match)
	}
	return Encode(baseline)
}

// A consumed failed Surface remains a historical replacement proof only when
// a completed package binds an immutable PASS whose exact baseline has this
// original SHA. No new/current reopen can use this predecessor exception.
func verifyHistoricalActivationContext(root, locale string, ref i18n.GlossaryArchiveReference) error {
	_, err := readHistoricalActivationContext(root, locale, ref)
	return err
}

func readHistoricalActivationContext(root, locale string, ref i18n.GlossaryArchiveReference) ([]byte, error) {
	if ref.Path != LocalePath(locale) {
		return nil, fmt.Errorf("not a reviewed activation context")
	}
	b, err := readRegular(root, ref.Path)
	if err != nil {
		return nil, err
	}
	g, err := reviewedActivationGlobal(root)
	if err != nil {
		return nil, err
	}
	baseline, err := readReviewedActivationAncestor(root, locale, ref, g, b, map[string]bool{})
	if err != nil {
		return nil, err
	}
	after, err := readRegular(root, ref.Path)
	if err != nil || !bytes.Equal(after, b) {
		return nil, fmt.Errorf("activation content scope changed during historical proof")
	}
	return baseline, nil
}

// An installed historical consumer can span several independently reviewed
// replacements. A receipt from an older completion cannot validate today's
// mixed state directly: first prove the newer transaction, then descend into
// its exact sealed baseline. No state is written and no old PASS is made current.
func readReviewedActivationAncestor(root, locale string, ref i18n.GlossaryArchiveReference, g *Global, current []byte, visited map[string]bool) ([]byte, error) {
	key := digest(current)
	if visited[key] {
		return nil, fmt.Errorf("cyclic reviewed activation history")
	}
	visited[key] = true
	var state Locale
	if err := StrictJSON(current, &state); err != nil {
		return nil, err
	}
	if state.Locale != locale {
		return nil, fmt.Errorf("historical activation locale mismatch")
	}
	if err := ValidateLocale(g, state); err != nil {
		return nil, err
	}
	seen := map[Reference]bool{}
	for _, c := range state.Packages {
		if c.EvidenceKind != PackageCompletionKind || len(c.Evidence) != 2 || seen[c.Evidence[1]] {
			continue
		}
		gate := c.Evidence[1]
		seen[gate] = true
		var r IntegratedSurfaceReceipt
		if err := readReference(root, gate, &r); err != nil {
			return nil, err
		}
		baseline, err := reviewedActivationBaseline(root, g, r, state)
		if err != nil {
			continue
		}
		if err := proveReviewedActivationState(root, gate, r, g, state, current); err != nil {
			continue // Another active receipt may prove the newer transaction.
		}
		retained, err := Encode(r.Scope.ExistingCoverage)
		if err == nil && digest(retained) == ref.SHA256 {
			return retained, nil
		}
		if digest(baseline) == ref.SHA256 {
			return baseline, nil
		}
		if ancestor, err := readReviewedActivationAncestor(root, locale, ref, g, baseline, visited); err == nil {
			// Descending is another validation stage. Recheck the parent, rather
			// than reusing a PASS after reading the older transaction's inputs.
			if err := proveReviewedActivationState(root, gate, r, g, state, current); err != nil {
				return nil, err
			}
			return ancestor, nil
		}
	}
	return nil, fmt.Errorf("no exact reviewed activation predecessor")
}

func currentReviewedScope(root string, r IntegratedSurfaceReceipt, gate Reference, build integratedScopeBuilder) (*IntegratedSurfaceScope, error) {
	if err := proveReviewedActivationTransition(root, gate, r); err != nil {
		return nil, err
	}
	want, err := build(root, r.Scope.Locale, r.Scope.Closures)
	if err != nil {
		return nil, err
	}
	actual, actualErr := readRegular(root, LocalePath(r.Scope.Locale))
	if actualErr != nil && !os.IsNotExist(actualErr) {
		return nil, actualErr
	}
	// Only the mechanically proven coverage reference may retain its frozen
	// baseline SHA. All other scope fields and language/public context stay exact.
	g, err := reviewedActivationGlobal(root)
	if err != nil {
		return nil, err
	}
	state, err := readLocaleAuthority(root, g, r.Scope.Locale)
	if err != nil {
		return nil, err
	}
	baseline, err := reviewedActivationBaseline(root, g, r, *state)
	if err != nil {
		return nil, err
	}
	for _, assets := range [][]SurfaceAsset{want.Carried, want.Reviewed} {
		for i := range assets {
			for j := range assets[i].Evidence {
				if assets[i].Evidence[j].Path == LocalePath(r.Scope.Locale) {
					// Current Tour checkpoints already retain the exact frozen
					// baseline reference. The transition proof above authenticates
					// those original bytes as well as the current coverage bytes.
					if assets[i].Evidence[j].SHA256 == digest(baseline) {
						continue
					}
					if actualErr != nil || assets[i].Evidence[j].SHA256 != digest(actual) {
						return nil, fmt.Errorf("unproven coverage reference: %s asset=%s sha256=%s", assets[i].Evidence[j].Path, assets[i].ID, assets[i].Evidence[j].SHA256)
					}
					assets[i].Evidence[j].SHA256 = digest(baseline)
				}
			}
		}
	}
	want.Identity = ""
	want.Identity = identity(want)
	return want, nil
}

// Transition proof binds the registry bytes; surrounding closure validation
// performs the full source inventory gate. Apply the independently checked
// shell successor so old/current package coordinates use the same registry.
func reviewedActivationGlobal(root string) (*Global, error) {
	b, err := readRegular(root, GlobalPath)
	if err != nil {
		return nil, err
	}
	var g Global
	if err := StrictJSON(b, &g); err != nil {
		return nil, err
	}
	if err := ValidateGlobal(g); err != nil {
		return nil, err
	}
	return applyCurrentShellAuthority(root, &g)
}
