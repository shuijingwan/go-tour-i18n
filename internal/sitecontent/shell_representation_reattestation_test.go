package sitecontent

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"
)

func shellReattestationFixture(t *testing.T) (string, *Global, *ShellRepresentationSuccessor) {
	t.Helper()
	root, g, _ := shellRepresentationFixture(t)
	for _, path := range shellReattestationValidationPaths {
		b, err := readRegular(repositoryRoot(t), path)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, path, b)
	}
	p, input := representationFixturePlan(t, root, g)
	if _, err := applyShellRepresentationSuccessor(root, input, p.PriorIdentity, p.PriorPointerSHA, func(*structuredSuccessorValidation) (*Global, error) { return g, nil }, nil); err != nil {
		t.Fatal(err)
	}
	return root, g, &p.Authority
}

func driftShellContract(t *testing.T, root string) {
	t.Helper()
	path := "docs/SHELL_REPRESENTATION_SUCCESSOR.md"
	b, err := readRegular(root, path)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, path, append(b, []byte("\nSynthetic command documentation evolution.\n")...))
}

func reattestationFixturePlan(t *testing.T, root string, g *Global) (*ShellRepresentationReattestationPlan, Reference) {
	t.Helper()
	var p *ShellRepresentationReattestationPlan
	err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		var err error
		p, err = planShellReattestationAt(root, g, v, false)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	path := "data/reattestation-" + p.Identity + ".json"
	if err := saveImmutable(root, path, p); err != nil {
		t.Fatal(err)
	}
	ref, err := ReferenceFile(root, path)
	if err != nil {
		t.Fatal(err)
	}
	return p, ref
}

func TestShellRepresentationDependencyReattestationLifecycle(t *testing.T) {
	root, g, base := shellReattestationFixture(t)
	baseRef := shellRepresentationReceipt(base)
	pointer, _ := readRegular(root, ShellRepresentationCurrent)
	receipt, _ := readRegular(root, baseRef.Path)
	p, input := reattestationFixturePlan(t, root, g)
	load := func(*structuredSuccessorValidation) (*Global, error) { return g, nil }
	if !p.BaseCurrent || p.Eligible || len(p.Authority.Changes) != 0 {
		t.Fatal("current base must not require re-attestation", p)
	}
	if _, err := applyShellReattestation(root, input, p.PriorPointerSHA, load, nil); err == nil {
		t.Fatal("unnecessary same-dependency authority accepted")
	}
	if _, _, err := currentShellRepresentation(root, g, successorValidation(root, nil), false); err != nil {
		t.Fatal("exact current base requires no additive authority", err)
	}
	if refs, err := shellReattestationContextReferencesAt(root, g, base, successorValidation(root, nil)); err != nil || len(refs) != 0 {
		t.Fatal("current base acquired an unnecessary additive context", refs, err)
	}
	driftShellContract(t, root)
	if _, err := checkShellRepresentation(root, g, baseRef, successorValidation(root, nil), map[Reference]bool{}, 0); err == nil {
		t.Fatal("stale base falsely reported valid")
	}
	if _, _, err := currentShellRepresentation(root, g, successorValidation(root, nil), false); err == nil {
		t.Fatal("stale base accepted without additive authority")
	}
	p, input = reattestationFixturePlan(t, root, g)
	if p.BaseCurrent || !p.Eligible || len(p.Authority.Changes) != 1 || p.State != "generation-required" || !reflect.DeepEqual(p.Authority.Representation, base.New) || !reflect.DeepEqual(p.Authority.Released, base.Released) {
		t.Fatal("dependency-only equivalence/state changed")
	}
	if p.Authority.Changes[0].Old.Reference.Path != "docs/SHELL_REPRESENTATION_SUCCESSOR.md" {
		t.Fatal("unexpected dependency change")
	}
	if _, err := applyShellReattestation(root, input, digest([]byte("wrong CAS")), load, nil); err == nil {
		t.Fatal("wrong prior accepted")
	}
	r, err := applyShellReattestation(root, input, p.PriorPointerSHA, load, nil)
	if err != nil || !r.Applied || r.AlreadyCurrent || r.Mutation != "known" {
		t.Fatal(r, err)
	}
	var previous *structuredSuccessorValidation
	for i := 0; i < 2; i++ {
		err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
			if previous == v {
				t.Fatal("independent invocation reused mutable world")
			}
			previous = v
			selected, _, err := selectShellRepresentationAt(root, g, []byte{}, v)
			if err == nil || selected != nil {
				t.Fatal("wrong source bytes accepted")
			}
			raw, _ := readRegular(root, base.Basis.Source.Path)
			selected, actual, err := selectShellRepresentationAt(root, g, raw, v)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(*selected, base.New.Source) || !reflect.DeepEqual(*actual, *base) || selected.Page.Units[0].ID != p.Authority.Representation.Source.Page.Units[0].ID {
				t.Fatal("reattestation changed representation/Unit/protected slots")
			}
			selected.Slots[0].Unit.Source = "caller mutation"
			refs, err := shellReattestationContextReferencesAt(root, g, base, v)
			if err != nil || len(refs) != 2 || refs[0].Path != p.CurrentPath || refs[1] != p.Receipt {
				t.Fatal("Generation context dropped additive proof", refs, err)
			}
			inspection, err := inspectShellReattestationAt(root, g, input, v, false)
			if err != nil || !inspection.AlreadyCurrent || inspection.Applied || inspection.Mutation != "none" {
				t.Fatal("readonly inspection claimed apply", inspection, err)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	r, err = applyShellReattestation(root, input, p.PriorPointerSHA, load, nil)
	if err != nil || !r.AlreadyCurrent || r.Applied || r.Mutation != "none" {
		t.Fatal("idempotent invocation claimed apply", r, err)
	}
	if _, err := planShellReattestationAt(root, g, successorValidation(root, nil), false); err == nil || !strings.Contains(err.Error(), "already current") {
		t.Fatal("duplicate same-dependency authority allowed", err)
	}
	// Strict base verification remains stale even after current selection works.
	if _, err := checkShellRepresentation(root, g, baseRef, successorValidation(root, nil), map[Reference]bool{}, 0); err == nil {
		t.Fatal("additive proof laundered stale base as valid")
	}
	for path, want := range map[string][]byte{ShellRepresentationCurrent: pointer, baseRef.Path: receipt} {
		got, err := readRegular(root, path)
		if err != nil || !bytes.Equal(want, got) {
			t.Fatal("base authority rewritten", path, err)
		}
	}
	for _, path := range []string{ShellRepresentationReattestationPending, "data/.glossary-corpus.lock"} {
		if _, err := os.Stat(pathJoin(root, path)); !os.IsNotExist(err) {
			t.Fatal("publication residue", path, err)
		}
	}
	// Further dependency evolution creates a continuous independent chain.
	driftShellContract(t, root)
	if _, _, err := currentShellRepresentation(root, g, successorValidation(root, nil), false); err == nil {
		t.Fatal("old re-attestation masked new dependency bytes")
	}
	next, nextInput := reattestationFixturePlan(t, root, g)
	if next.Authority.Previous == nil || *next.Authority.Previous != p.Receipt {
		t.Fatal("re-attestation lineage lost")
	}
	if _, err := applyShellReattestation(root, nextInput, next.PriorPointerSHA, load, nil); err != nil {
		t.Fatal(err)
	}
	if _, _, err := currentShellRepresentation(root, g, successorValidation(root, nil), false); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, next.Receipt.Path, []byte("tampered additive receipt"))
	if _, _, err := currentShellRepresentation(root, g, successorValidation(root, nil), false); err == nil {
		t.Fatal("tampered additive receipt accepted")
	}
}

func TestShellRepresentationReattestationAuthorityDrift(t *testing.T) {
	for _, name := range []string{"raw", "foundation", "adoption", "heads", "pointer", "base-receipt", "unit", "protected-input", "slots", "released", "parser-output", "mode"} {
		t.Run(name, func(t *testing.T) {
			root, g, base := shellReattestationFixture(t)
			driftShellContract(t, root)
			switch name {
			case "raw":
				writeFixture(t, root, base.Basis.Source.Path, []byte("changed raw source"))
			case "foundation":
				writeFixture(t, root, base.Basis.Foundation.Path, []byte("changed foundation"))
			case "adoption":
				writeFixture(t, root, base.Basis.Adoption.Path, []byte("changed adoption"))
			case "heads":
				g.StructuredSuccessors = []Reference{{"data/missing-head.json", digest(nil)}}
			case "pointer":
				writeFixture(t, root, ShellRepresentationCurrent, []byte(`{"path":"wrong","sha256":"wrong"}`))
			case "base-receipt":
				writeFixture(t, root, shellRepresentationReceipt(base).Path, []byte("tampered base"))
			case "mode":
				if err := os.Chmod(pathJoin(root, "docs/SHELL_REPRESENTATION_SUCCESSOR.md"), 0700); err != nil {
					t.Fatal(err)
				}
			default:
				// Model a different previously installed parser result with valid
				// receipt identity. Current deterministic output must reject it.
				switch name {
				case "unit":
					base.New.Source.Page.Units[0].ID = digest([]byte("different Unit"))
				case "protected-input":
					base.New.ProtectedInputSHA = digest([]byte("different protected input"))
				case "slots":
					base.New.Source.Slots[0].Role = "machine"
				case "released":
					base.Released[0].Start++
				case "parser-output":
					base.New.Source.Page.Units[0].Source += "injected content"
				}
				base.Identity = ""
				base.Identity = identity(*base)
				ref := shellRepresentationReceipt(base)
				if err := saveImmutable(root, ref.Path, base); err != nil {
					t.Fatal(err)
				}
				b, _ := Encode(ref)
				writeFixture(t, root, ShellRepresentationCurrent, b)
			}
			if _, err := planShellReattestationAt(root, g, successorValidation(root, nil), false); err == nil {
				t.Fatal("authority/semantic drift accepted", name)
			}
		})
	}
}

func TestShellRepresentationReattestationConcurrentMutation(t *testing.T) {
	for _, name := range []string{"contract", "raw", "validation-code", "mode", "base-pointer"} {
		t.Run(name, func(t *testing.T) {
			root, g, base := shellReattestationFixture(t)
			driftShellContract(t, root)
			p, input := reattestationFixturePlan(t, root, g)
			load := func(*structuredSuccessorValidation) (*Global, error) { return g, nil }
			_, err := applyShellReattestation(root, input, p.PriorPointerSHA, load, func() {
				switch name {
				case "contract":
					driftShellContract(t, root)
				case "raw":
					writeFixture(t, root, base.Basis.Source.Path, []byte("concurrent source change"))
				case "validation-code":
					writeFixture(t, root, shellReattestationValidationPaths[0], []byte("concurrent verifier change"))
				case "mode":
					if err := os.Chmod(pathJoin(root, shellReattestationValidationPaths[0]), 0700); err != nil {
						t.Fatal(err)
					}
				case "base-pointer":
					writeFixture(t, root, ShellRepresentationCurrent, []byte("concurrent pointer change"))
				}
			})
			if err == nil {
				t.Fatal("concurrent mutation accepted")
			}
			if _, err := os.Stat(pathJoin(root, p.CurrentPath)); !os.IsNotExist(err) {
				t.Fatal("failed apply installed authority")
			}
		})
	}
}

func TestShellRepresentationReattestationPlanAndLineageFailClosed(t *testing.T) {
	for _, name := range []string{"base-receipt", "base-pointer", "new-unit", "released", "missing-predecessor", "plan-schema"} {
		t.Run(name, func(t *testing.T) {
			root, g, _ := shellReattestationFixture(t)
			driftShellContract(t, root)
			p, _ := reattestationFixturePlan(t, root, g)
			switch name {
			case "base-receipt":
				p.Authority.BaseReceipt.SHA256 = digest([]byte("wrong base"))
			case "base-pointer":
				p.Authority.BasePointer.SHA256 = digest([]byte("wrong base pointer"))
			case "new-unit":
				p.Authority.Representation.Source.Page.Units[0].ID = digest([]byte("different Unit"))
			case "released":
				p.Authority.Released[0].End++
			case "missing-predecessor":
				p.Authority.Previous = &Reference{ShellRepresentationReattestationRoot + "receipts/missing.json", digest(nil)}
			case "plan-schema":
				p.Schema = ShellRepresentationPlanSchema
			}
			p.Authority.Identity = ""
			p.Authority.Identity = identity(p.Authority)
			p.Receipt = shellReattestationReceipt(&p.Authority)
			p.Identity = ""
			p.Identity = identity(*p)
			if err := saveImmutable(root, "data/tampered-plan.json", p); err != nil {
				t.Fatal(err)
			}
			input, _ := ReferenceFile(root, "data/tampered-plan.json")
			if _, err := inspectShellReattestationAt(root, g, input, successorValidation(root, nil), false); err == nil {
				t.Fatal("tampered/reconstructed plan accepted")
			}
		})
	}
}

func TestShellRepresentationReattestationPending(t *testing.T) {
	for _, path := range []string{ShellRepresentationPending, ShellRepresentationReattestationPending, "data/.glossary-corpus.lock"} {
		t.Run(path, func(t *testing.T) {
			root, g, _ := shellReattestationFixture(t)
			driftShellContract(t, root)
			writeFixture(t, root, path, []byte("unknown publication"))
			if _, err := planShellReattestationAt(root, g, successorValidation(root, nil), false); err == nil {
				t.Fatal("pending/lock ignored")
			}
			if _, _, err := currentShellRepresentation(root, g, successorValidation(root, nil), false); err == nil {
				t.Fatal("current selection ignored pending/lock")
			}
			if _, err := readRegular(root, path); err != nil {
				t.Fatal("diagnosis removed marker/lock")
			}
		})
	}
}
