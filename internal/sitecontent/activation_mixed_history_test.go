package sitecontent

import (
	"bytes"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// The installed corpus still binds a pre-adoption completion file. Replacing
// one package under a later PASS leaves completions from two receipt epochs.
func TestIntegratedSurfaceActivationMixedReceiptHistory(t *testing.T) {
	root := copiedCanonicalLocaleFixture(t)
	gate, err := ReferenceFile(root, "data/locale-language/zh-CN/surface/20261018-zh-CN-integrated-surface-004.json")
	if err != nil {
		t.Fatal(err)
	}
	var r IntegratedSurfaceReceipt
	if err := readReference(root, gate, &r); err != nil {
		t.Fatal(err)
	}
	g, err := reviewedActivationGlobal(root)
	if err != nil {
		t.Fatal(err)
	}
	current, err := readLocaleAuthority(root, g, r.Scope.Locale)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := reviewedActivationBaseline(root, g, r, *current)
	if err != nil {
		t.Fatal(err)
	}
	var affected ShellAffectedScope
	b, err := readRegular(root, "data/site-shell/successor/affected/zh-CN/b106897bf70ef8c8bd3d439bebca598ea911635364c8a1000a40b3ce83b599e2.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := StrictJSON(b, &affected); err != nil {
		t.Fatal(err)
	}
	historical := i18n.GlossaryArchiveReference{Path: affected.CurrentCompletion.Path, SHA256: affected.CurrentCompletion.SHA256}
	writeFixture(t, root, LocalePath(r.Scope.Locale), baseline)
	original, err := readHistoricalActivationContext(root, r.Scope.Locale, historical)
	if err != nil || digest(original) != historical.SHA256 {
		t.Fatalf("pre-replacement historical context: %v", err)
	}
	var priorState Locale
	if err := StrictJSON(baseline, &priorState); err != nil {
		t.Fatal(err)
	}
	oldGate := priorState.Packages[len(r.Scope.ExistingCoverage.Packages)].Evidence[1]
	var oldReceipt IntegratedSurfaceReceipt
	if err := readReference(root, oldGate, &oldReceipt); err != nil {
		t.Fatal(err)
	}
	completions := map[string]Completion{}
	for _, ref := range r.Scope.Closures {
		var closure PackageClosure
		if err := readReference(root, ref, &closure); err != nil {
			t.Fatal(err)
		}
		p, err := packageByID(g, closure.Package)
		if err != nil {
			t.Fatal(err)
		}
		refs := []Reference{ref, gate}
		completions[p.ID] = Completion{Package: p.ID, State: "complete", PackageIdentity: p.Identity, SourceIdentity: p.SourceIdentity, Surfaces: p.Surfaces, Routes: p.Routes, RouteFamilies: routeFamilies(p.ID), EvidenceKind: PackageCompletionKind, Evidence: refs, EvidenceIdentity: identity(refs), ContextIdentity: closure.Identity}
	}
	for _, order := range [][]string{{"learn-docs-v1", "site-v2-shell"}, {"site-v2-shell", "learn-docs-v1"}} {
		t.Run(order[0]+"-first", func(t *testing.T) {
			var state Locale
			if err := StrictJSON(baseline, &state); err != nil {
				t.Fatal(err)
			}
			for _, pkg := range order {
				for i, old := range state.Packages {
					if old.Package == pkg {
						state.Packages[i] = completions[pkg]
					}
				}
				atStep, err := Encode(state)
				if err != nil {
					t.Fatal(err)
				}
				writeFixture(t, root, historical.Path, atStep)
				if err := proveReviewedActivationTransition(root, gate, r); err != nil {
					t.Fatal("current Surface-004 transition", pkg, err)
				}
				if err := proveReviewedActivationTransition(root, oldGate, oldReceipt); err == nil {
					t.Fatal("historical receipt incorrectly became current", pkg)
				}
				got, err := readHistoricalActivationContext(root, r.Scope.Locale, historical)
				if err != nil || !bytes.Equal(got, original) {
					t.Fatalf("%s replacement lost exact pre-adoption context %s: %v", pkg, historical.SHA256, err)
				}
				after, err := readRegular(root, historical.Path)
				if err != nil || !bytes.Equal(after, atStep) {
					t.Fatal("historical proof mutated current authority", err)
				}
			}
		})
	}
	// Return to the real intermediate shape: new Learn/Docs + old shell. A
	// chain is admissible only while both transaction epochs remain exact.
	for i, old := range priorState.Packages {
		if old.Package == "learn-docs-v1" {
			priorState.Packages[i] = completions[old.Package]
		}
	}
	mixed, err := Encode(priorState)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, historical.Path, mixed)
	for _, mutation := range []string{"current-context", "predecessor-context", "predecessor-reference", "formatting", "historical-closure", "historical-Surface", "current-Surface", "historical-Markdown", "ancestor-SHA"} {
		t.Run(mutation, func(t *testing.T) {
			var changed Locale
			if err := StrictJSON(mixed, &changed); err != nil {
				t.Fatal(err)
			}
			path, payload, target := historical.Path, mixed, historical
			switch mutation {
			case "current-context", "predecessor-context", "predecessor-reference":
				for i := range changed.Packages {
					c := &changed.Packages[i]
					if mutation == "current-context" && c.Package == "learn-docs-v1" || mutation != "current-context" && c.Package == "site-v2-shell" {
						if mutation == "predecessor-reference" {
							c.Evidence[0].SHA256 = digest([]byte("different predecessor"))
							c.EvidenceIdentity = identity(c.Evidence)
						} else {
							c.ContextIdentity = digest([]byte("different context"))
						}
					}
				}
				payload, err = Encode(changed)
				if err != nil {
					t.Fatal(err)
				}
			case "formatting":
				payload = append(bytes.Clone(mixed), '\n')
			case "historical-closure":
				path = priorState.Packages[len(priorState.Packages)-1].Evidence[0].Path
			case "historical-Surface":
				path = oldGate.Path
			case "current-Surface":
				path = gate.Path
			case "historical-Markdown":
				path = oldReceipt.Evidence.Path
			case "ancestor-SHA":
				target.SHA256 = digest([]byte("unproven ancestor"))
			}
			before, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			if path != historical.Path {
				payload = append(bytes.Clone(before), '\n')
			}
			writeFixture(t, root, path, payload)
			defer writeFixture(t, root, path, before)
			if _, err := readHistoricalActivationContext(root, r.Scope.Locale, target); err == nil {
				t.Fatal("inexact multi-epoch activation proof accepted")
			}
		})
	}
}
