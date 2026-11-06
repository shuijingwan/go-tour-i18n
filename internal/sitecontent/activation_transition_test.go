package sitecontent

import (
	"bytes"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Copy only the references needed by the non-recursive transition proof.
// Full package/QC verification belongs to the sequential activation regression.
func reviewedActivationFixture(t *testing.T) (string, Reference, IntegratedSurfaceReceipt, Locale) {
	t.Helper()
	repository, root := repositoryRoot(t), t.TempDir()
	gate, err := ReferenceFile(repository, "data/locale-language/zh-CN/surface/20261011-zh-CN-integrated-surface-002.json")
	if err != nil {
		t.Fatal(err)
	}
	var r IntegratedSurfaceReceipt
	if err := readReference(repository, gate, &r); err != nil {
		t.Fatal(err)
	}
	refs := append(surfaceContextReferences(r.Scope), i18n.GlossaryArchiveReference{Path: gate.Path, SHA256: gate.SHA256}, i18n.GlossaryArchiveReference{Path: r.Evidence.Path, SHA256: r.Evidence.SHA256})
	for _, ref := range refs {
		b, err := readRegular(repository, ref.Path)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, ref.Path, b)
	}
	// The repository may have advanced to another reviewed source epoch.
	// This fixture belongs to this receipt, not the final mutable locale file.
	var global Global
	b, err := readRegular(root, GlobalPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := StrictJSON(b, &global); err != nil {
		t.Fatal(err)
	}
	state := r.Scope.ExistingCoverage
	state.Packages = append([]Completion{}, state.Packages...)
	for _, ref := range r.Scope.Closures {
		var closure PackageClosure
		if err := readReference(root, ref, &closure); err != nil {
			t.Fatal(err)
		}
		p, err := packageByID(&global, closure.Package)
		if err != nil || p.Identity != closure.PackageSHA {
			t.Fatal("fixture closure/source epoch mismatch", err)
		}
		refs := []Reference{ref, gate}
		state.Packages = append(state.Packages, Completion{Package: p.ID, State: "complete", PackageIdentity: p.Identity, SourceIdentity: p.SourceIdentity, Surfaces: p.Surfaces, Routes: p.Routes, RouteFamilies: routeFamilies(p.ID), EvidenceKind: PackageCompletionKind, Evidence: refs, EvidenceIdentity: identity(refs), ContextIdentity: closure.Identity})
	}
	b, err = Encode(state)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, LocalePath(r.Scope.Locale), b)
	return root, gate, r, state
}

func TestIntegratedSurfaceActivationTransition(t *testing.T) {
	root, gate, r, state := reviewedActivationFixture(t)
	before, _ := readRegular(root, gate.Path)
	baseline, _ := Encode(r.Scope.ExistingCoverage)
	old := i18n.GlossaryArchiveReference{Path: LocalePath(r.Scope.Locale), SHA256: digest(baseline)}
	if err := proveReviewedActivationTransition(root, gate, r); err != nil {
		t.Fatal("first package completion from the same reviewed scope", err)
	}
	if err := verifyHistoricalActivationContext(root, r.Scope.Locale, old); err != nil {
		t.Fatal("frozen historical replacement context", err)
	}
	// Both activation orders and the fully completed reviewed transaction have
	// the same exact closure/receipt requirements, without another judgment.
	var global Global
	globalBytes, _ := readRegular(root, GlobalPath)
	if err := StrictJSON(globalBytes, &global); err != nil {
		t.Fatal(err)
	}
	completions := map[string]Completion{}
	for _, ref := range r.Scope.Closures {
		var closure PackageClosure
		if err := readReference(root, ref, &closure); err != nil {
			t.Fatal(err)
		}
		p, err := packageByID(&global, closure.Package)
		if err != nil {
			t.Fatal(err)
		}
		refs := []Reference{ref, gate}
		completions[p.ID] = Completion{Package: p.ID, State: "complete", PackageIdentity: p.Identity, SourceIdentity: p.SourceIdentity, Surfaces: p.Surfaces, Routes: p.Routes, RouteFamilies: routeFamilies(p.ID), EvidenceKind: PackageCompletionKind, Evidence: refs, EvidenceIdentity: identity(refs), ContextIdentity: closure.Identity}
	}
	for _, order := range [][]string{{"learn-docs-v1", "site-v2-shell"}, {"site-v2-shell", "learn-docs-v1"}} {
		expected := r.Scope.ExistingCoverage
		expected.Packages = append([]Completion{}, expected.Packages...)
		for _, pkg := range order {
			expected.Packages = append(expected.Packages, completions[pkg])
			b, _ := Encode(expected)
			writeFixture(t, root, old.Path, b)
			if err := proveReviewedActivationTransition(root, gate, r); err != nil {
				t.Fatal("reviewed completion order", order, pkg, err)
			}
		}
	}
	accepted, _ := Encode(state)
	writeFixture(t, root, old.Path, accepted)
	build := func(string, string, []Reference) (*IntegratedSurfaceScope, error) {
		b, _ := Encode(r.Scope)
		var s IntegratedSurfaceScope
		if err := StrictJSON(b, &s); err != nil {
			return nil, err
		}
		current, _ := readRegular(root, old.Path)
		for i := range s.Carried {
			for j := range s.Carried[i].Evidence {
				if s.Carried[i].Evidence[j].Path == old.Path {
					s.Carried[i].Evidence[j].SHA256 = digest(current)
				}
			}
		}
		s.Identity = ""
		s.Identity = identity(s)
		return &s, nil
	}
	if ref, err := requireIntegratedSurface(root, r.Scope.Locale, r.Scope.Closures[1], build); err != nil || ref != gate {
		t.Fatal("remaining prospective closure lost the exact PASS", err)
	}
	frozenCheckpoint := func(string, string, []Reference) (*IntegratedSurfaceScope, error) {
		b, _ := Encode(r.Scope)
		var s IntegratedSurfaceScope
		if err := StrictJSON(b, &s); err != nil {
			return nil, err
		}
		return &s, nil // Tour checkpoint retains its SHA-exact baseline ref.
	}
	if ref, err := requireIntegratedSurface(root, r.Scope.Locale, r.Scope.Closures[1], frozenCheckpoint); err != nil || ref != gate {
		t.Fatal("frozen current checkpoint lost the exact PASS", err)
	}
	// Historical failed scope shares the shell closure but binds qc-009. Its
	// immutable identity is checked; it must never enter current PASS rebuild.
	failedPath := "data/locale-language/zh-CN/surface/20261011-zh-CN-integrated-surface-001.json"
	failedBytes, err := readRegular(repositoryRoot(t), failedPath)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, failedPath, failedBytes)
	calls := 0
	currentOnly := func(root, locale string, refs []Reference) (*IntegratedSurfaceScope, error) {
		calls++
		if len(refs) != len(r.Scope.Closures) || refs[0] != r.Scope.Closures[0] {
			t.Fatal("failed historical scope entered current PASS selection")
		}
		return build(root, locale, refs)
	}
	if ref, err := requireIntegratedSurface(root, r.Scope.Locale, r.Scope.Closures[1], currentOnly); err != nil || ref != gate || calls != 1 {
		t.Fatal("historical failed scope blocked current PASS", err)
	}
	// Baseline bytes remain admissible before any activation.
	writeFixture(t, root, old.Path, baseline)
	if err := proveReviewedActivationTransition(root, gate, r); err != nil {
		t.Fatal("reviewed baseline", err)
	}
	if err := verifyHistoricalActivationContext(root, r.Scope.Locale, old); err == nil {
		t.Fatal("a baseline without activated completion claimed a successor transition")
	}
	b, _ := Encode(state)
	writeFixture(t, root, old.Path, b)
	after, _ := readRegular(root, gate.Path)
	if !bytes.Equal(before, after) {
		t.Fatal("transition proof rewrote immutable receipt")
	}
	// Only coverage evidence may normalize; language/public fields remain exact.
	bad := func(root, locale string, refs []Reference) (*IntegratedSurfaceScope, error) {
		s, err := build(root, locale, refs)
		if err == nil {
			s.Reviewed[0].TargetSHA = digest([]byte("changed language candidate"))
		}
		return s, err
	}
	if _, err := requireIntegratedSurface(root, r.Scope.Locale, r.Scope.Closures[1], bad); err == nil {
		t.Fatal("changed reviewed language carried")
	}
	badReference := func(root, locale string, refs []Reference) (*IntegratedSurfaceScope, error) {
		s, err := build(root, locale, refs)
		if err == nil {
			for i := range s.Carried {
				for j := range s.Carried[i].Evidence {
					if s.Carried[i].Evidence[j].Path == old.Path {
						s.Carried[i].Evidence[j].SHA256 = digest([]byte("unrelated reference"))
					}
				}
			}
		}
		return s, err
	}
	if _, err := requireIntegratedSurface(root, r.Scope.Locale, r.Scope.Closures[1], badReference); err == nil {
		t.Fatal("path whitelist accepted an unproven coverage reference")
	}
	if _, err := requireIntegratedSurface(root, r.Scope.Locale, Reference{Path: "data/unrelated-closure.json", SHA256: old.SHA256}, build); err == nil {
		t.Fatal("out-of-scope package received activation authority")
	}
}

func TestIntegratedSurfaceActivationTransitionFailClosed(t *testing.T) {
	root, gate, r, _ := reviewedActivationFixture(t)
	path := LocalePath(r.Scope.Locale)
	original, _ := readRegular(root, path)
	baseline, _ := Encode(r.Scope.ExistingCoverage)
	old := i18n.GlossaryArchiveReference{Path: path, SHA256: digest(baseline)}
	for _, mutation := range []string{"formatting", "baseline", "package-order", "fake-completion", "different-closure", "different-Surface", "unrelated-package", "locale"} {
		t.Run(mutation, func(t *testing.T) {
			var state Locale
			if err := StrictJSON(original, &state); err != nil {
				t.Fatal(err)
			}
			switch mutation {
			case "baseline":
				state.Packages[0].EvidenceIdentity = digest([]byte("rewritten baseline"))
			case "package-order":
				state.Packages[0], state.Packages[1] = state.Packages[1], state.Packages[0]
			case "fake-completion":
				state.Packages[1].ContextIdentity = digest([]byte("fake completion"))
			case "different-closure":
				state.Packages[1].Evidence[0].SHA256 = digest([]byte("unreviewed closure"))
				state.Packages[1].EvidenceIdentity = identity(state.Packages[1].Evidence)
			case "different-Surface":
				state.Packages[1].Evidence[1].Path = "data/locale-language/zh-CN/surface/another-pass.json"
				state.Packages[1].EvidenceIdentity = identity(state.Packages[1].Evidence)
			case "unrelated-package":
				state.Packages[1].Package = "unrelated-package"
			case "locale":
				state.Locale = "fr-FR"
			}
			b, _ := Encode(state)
			if mutation == "formatting" {
				b = append(b, '\n')
			}
			writeFixture(t, root, path, b)
			defer writeFixture(t, root, path, original)
			if err := proveReviewedActivationTransition(root, gate, r); err == nil {
				t.Fatal("unexpected coverage mutation authorized")
			}
			if err := verifyHistoricalActivationContext(root, r.Scope.Locale, old); err == nil {
				t.Fatal("unexpected coverage mutation accepted as historical proof")
			}
		})
	}
	for _, ref := range []Reference{gate, r.Evidence, r.Scope.Closures[0], r.Scope.UnifiedReview, r.Scope.Runtime[0]} {
		t.Run(ref.Path, func(t *testing.T) {
			before, _ := readRegular(root, ref.Path)
			writeFixture(t, root, ref.Path, append(bytes.Clone(before), '\n'))
			defer writeFixture(t, root, ref.Path, before)
			if err := proveReviewedActivationTransition(root, gate, r); err == nil {
				t.Fatal("changed immutable/public context accepted")
			}
		})
	}
}

func TestIntegratedSurfaceActivationPreflightCASState(t *testing.T) {
	root, _, r, checked := reviewedActivationFixture(t)
	current, _ := readRegular(root, LocalePath(r.Scope.Locale))
	if err := checkActivationPreflightState(&checked, current, false); err != nil {
		t.Fatal("exact checked state lost its CAS binding", err)
	}
	if err := checkActivationPreflightState(&r.Scope.ExistingCoverage, current, false); err == nil {
		t.Fatal("earlier baseline combined with later prior SHA would drop the first completion")
	}
	if err := checkActivationPreflightState(&checked, nil, true); err == nil {
		t.Fatal("complete coverage survived concurrent deletion")
	}
	empty := Locale{Schema: LocaleSchema, Locale: r.Scope.Locale, Packages: []Completion{}}
	if err := checkActivationPreflightState(&empty, nil, true); err != nil {
		t.Fatal("first sparse creation rejected", err)
	}
}
